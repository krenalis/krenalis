// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"container/list"
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"
)

const (
	batchSize  = 64
	batchDelay = 2 * time.Millisecond
	queueSize  = batchSize * 4

	refillBackoffDuration = 250 * time.Millisecond
	maxWaitDuration       = time.Second
	defaultAcquireTimeout = 5 * time.Second
)

// ErrCapacityExceeded is returned when an operation cannot be served from
// local capacity or admitted to the current refill.
var ErrCapacityExceeded = errors.New("rate-limit capacity exceeded")

// ErrInvalidCost is returned when an operation has an unsupported cost.
var ErrInvalidCost = errors.New("invalid rate-limit cost")

// Counter records a rate-limiter event.
type Counter interface {
	Inc()
}

// Metrics contains optional counters for rate-limiter events.
type Metrics struct {
	AcquisitionErrors Counter
	QueueFull         Counter
}

// LeaseRequest requests capacity for one subject kind and identifier.
type LeaseRequest struct {
	SubjectKind    SubjectKind
	SubjectID      string
	RequestedUnits int
}

// LeaseResult reports capacity granted for one subject kind and identifier.
type LeaseResult struct {
	SubjectKind   SubjectKind
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
}

// AcquireFunc acquires capacity for a batch of lease requests.
//
// Each result must match exactly one request. The acquirer must reserve every
// granted unit in its authoritative store before returning it.
type AcquireFunc func(context.Context, []LeaseRequest) ([]LeaseResult, error)

// Limiter coordinates refills for local buckets. It owns the refill queue,
// batching, backoff, and shutdown lifecycle.
type Limiter struct {
	acquireLeases  AcquireFunc
	maxWait        time.Duration
	acquireTimeout time.Duration

	queue          chan *refill // bounded so queue publication never blocks.
	queueSaturated atomic.Bool  // suppresses repeated warnings while the queue is full.

	// retryAfter is the Unix-nanosecond deadline for the shared refill
	// backoff. It is atomic because every consumption checks it, while only the
	// single batcher updates it. A shared deadline prevents a database outage
	// from causing rapid retry batches for different buckets.
	retryAfter atomic.Int64

	metrics Metrics

	shutdown struct {
		ctx    context.Context
		cancel context.CancelFunc
		sync.WaitGroup
	}
}

// refill represents one refill generation, including its immutable lease
// request and the waiters admitted to it. Mutations are protected by bucket.mu.
// Closing done publishes final waiter results to readers.
// The published channel prevents the batcher from processing the refill before
// queue publication and activation are complete.
type refill struct {
	bucket      *Bucket
	request     LeaseRequest
	active      bool
	published   chan struct{}
	done        chan struct{}
	waiters     list.List
	pendingCost int
}

// New starts a limiter and its single refill batcher.
func New(acquire AcquireFunc, metrics Metrics) *Limiter {
	limiter := &Limiter{
		acquireLeases:  acquire,
		maxWait:        maxWaitDuration,
		acquireTimeout: defaultAcquireTimeout,
		queue:          make(chan *refill, queueSize),
		metrics:        metrics,
	}
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	limiter.shutdown.Add(1)
	go limiter.runBatcher()
	return limiter
}

// Close closes the limiter's refill lifecycle by preventing new refills,
// cancelling lease acquisition, and waiting for the batcher to stop.
// The batch currently being handled is finished without applying unconfirmed
// capacity, but entries still buffered in the queue are not drained.
// Their waiters observe the shutdown context and cannot remain blocked.
//
// Any remaining local leases are discarded. A best-effort return could credit
// capacity already consumed locally twice.
func (limiter *Limiter) Close() {
	limiter.shutdown.cancel()
	limiter.shutdown.Wait()
}

// Consume consumes cost from bucket. It returns ErrInvalidCost when cost is
// outside the bucket's supported range. If local capacity is insufficient, it
// may wait for the refill generation to which the request was admitted.
func (limiter *Limiter) Consume(ctx context.Context, bucket *Bucket, cost int) error {
	if cost < 1 || cost > bucket.maxCost {
		return ErrInvalidCost
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now()
	refillAllowed := limiter.shutdown.ctx.Err() == nil && !limiter.backoffActive(now)
	satisfied, refill, waiter := bucket.consume(cost, refillAllowed)
	if refill != nil {
		queued, queueFull := limiter.queueRefill(refill)
		if !queued {
			bucket.rejectRefill(refill)
			if queueFull {
				if limiter.metrics.QueueFull != nil {
					limiter.metrics.QueueFull.Inc()
				}
				if limiter.queueSaturated.CompareAndSwap(false, true) {
					slog.Warn("rate limiter refill queue is full")
				}
			}
		} else {
			limiter.queueSaturated.Store(false)
			refillAllowed = limiter.shutdown.ctx.Err() == nil && !limiter.backoffActive(time.Now())
			waiter = bucket.activateRefill(refill, cost, !satisfied, refillAllowed)
		}
	}
	if satisfied {
		return nil
	}
	if waiter == nil {
		return ErrCapacityExceeded
	}
	return limiter.waitForRefill(ctx, waiter)
}

// backoffActive reports whether new refill attempts are temporarily suppressed
// after an acquisition error or an invalid batch response.
func (limiter *Limiter) backoffActive(now time.Time) bool {
	return now.UnixNano() < limiter.retryAfter.Load()
}

// collectAndRefillBatch collects refill generations for a short window, then
// acquires their leases in one call. It returns false when shutdown interrupts
// collection and the batcher should stop.
func (limiter *Limiter) collectAndRefillBatch(firstRefill *refill) bool {
	pendingRefills := map[*refill]LeaseRequest{firstRefill: {}}
	timer := time.NewTimer(batchDelay)
	defer timer.Stop()
	for len(pendingRefills) < batchSize {
		select {
		case <-limiter.shutdown.ctx.Done():
			failCollectedRefills(pendingRefills)
			return false
		case refill := <-limiter.queue:
			if _, exists := pendingRefills[refill]; !exists {
				pendingRefills[refill] = LeaseRequest{}
			}
		case <-timer.C:
			limiter.refillBatch(pendingRefills)
			return true
		}
	}
	limiter.refillBatch(pendingRefills)
	return true
}

// failRefillBatch fails a refill batch while preserving current local capacity
// and starts a fixed backoff that temporarily prevents new refill attempts.
// The shared deadline applies to every bucket, including those outside this
// batch, so an acquisition failure cannot trigger rapid retries across the
// queue.
//
// Shutdown only finishes pending refills; it does not start a backoff.
func (limiter *Limiter) failRefillBatch(pendingRefills map[*refill]LeaseRequest) {
	if limiter.shutdown.ctx.Err() != nil {
		failCollectedRefills(pendingRefills)
		return
	}
	now := time.Now()
	limiter.retryAfter.Store(now.Add(refillBackoffDuration).UnixNano())
	failCollectedRefills(pendingRefills)
}

// queueRefill queues a refill generation without blocking the request path.
func (limiter *Limiter) queueRefill(refill *refill) (queued, queueFull bool) {
	if limiter.shutdown.ctx.Err() != nil {
		return false, false
	}
	select {
	case limiter.queue <- refill:
		return true, false
	default:
		return false, true
	}
}

// refillBatch builds, acquires, validates, and applies leases for one collected
// batch. An acquisition error or invalid response fails the whole batch before
// any local capacity is changed.
func (limiter *Limiter) refillBatch(pendingRefills map[*refill]LeaseRequest) {
	if limiter.backoffActive(time.Now()) {
		// Generations queued before a failed batch may still be in the channel.
		// They are not acquired during global backoff. Rejecting them also
		// prevents waiters from relying on a refill that will not run.
		failCollectedRefills(pendingRefills)
		return
	}

	leaseRequests := make([]LeaseRequest, 0, len(pendingRefills))
	for refill := range pendingRefills {
		select {
		case <-refill.published:
		case <-limiter.shutdown.ctx.Done():
			failCollectedRefills(pendingRefills)
			return
		}
		request, needed := refill.bucket.refillRequest(refill)
		if !needed {
			delete(pendingRefills, refill)
			continue
		}
		pendingRefills[refill] = request
		leaseRequests = append(leaseRequests, request)
	}
	if len(leaseRequests) == 0 {
		return
	}

	acquireTimeout := limiter.acquireTimeout
	if acquireTimeout <= 0 {
		acquireTimeout = defaultAcquireTimeout
	}
	acquireCtx, cancelAcquire := context.WithTimeout(limiter.shutdown.ctx, acquireTimeout)
	leaseResults, err := limiter.acquireLeases(acquireCtx, leaseRequests)
	if err == nil {
		err = acquireCtx.Err()
	}
	cancelAcquire()
	if err != nil {
		if limiter.shutdown.ctx.Err() == nil {
			slog.Error("cannot acquire rate-limit leases", "error", err)
			if limiter.metrics.AcquisitionErrors != nil {
				limiter.metrics.AcquisitionErrors.Inc()
			}
		}
		limiter.failRefillBatch(pendingRefills)
		return
	}

	requestedUnitsBySubject := make(map[subjectKey]int, len(pendingRefills))
	for _, request := range pendingRefills {
		requestedUnitsBySubject[subjectKey{kind: request.SubjectKind, id: request.SubjectID}] = request.RequestedUnits
	}
	resultsBySubject := make(map[subjectKey]LeaseResult, len(leaseResults))
	for _, result := range leaseResults {
		key := subjectKey{kind: result.SubjectKind, id: result.SubjectID}
		requestedUnits, ok := requestedUnitsBySubject[key]
		if !ok {
			limiter.failRefillBatch(pendingRefills)
			slog.Error("rate limiter lease result does not match its request", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		if _, duplicate := resultsBySubject[key]; duplicate {
			limiter.failRefillBatch(pendingRefills)
			slog.Error("rate limiter lease batch returned a duplicate result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		invalid := result.GrantedUnits < 0 || result.GrantedUnits > requestedUnits ||
			result.CapacityUnits <= 0 || result.GrantedUnits > result.CapacityUnits
		if invalid {
			limiter.failRefillBatch(pendingRefills)
			slog.Error("rate limiter lease batch returned an invalid result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID, "granted_units", result.GrantedUnits, "requested_units", requestedUnits, "capacity_units", result.CapacityUnits)
			return
		}
		resultsBySubject[key] = result
	}
	if len(resultsBySubject) != len(pendingRefills) {
		limiter.failRefillBatch(pendingRefills)
		slog.Error("rate limiter lease batch returned incomplete results")
		return
	}

	for refill, request := range pendingRefills {
		key := subjectKey{kind: request.SubjectKind, id: request.SubjectID}
		result := resultsBySubject[key]
		refill.bucket.completeRefill(refill, result.GrantedUnits, result.CapacityUnits)
	}
	limiter.retryAfter.Store(0)
}

// runBatcher processes one collected refill batch at a time until Close
// cancels the limiter context.
func (limiter *Limiter) runBatcher() {
	defer limiter.shutdown.Done()
	for {
		select {
		case <-limiter.shutdown.ctx.Done():
			return
		case firstRefill := <-limiter.queue:
			if !limiter.collectAndRefillBatch(firstRefill) {
				return
			}
		}
	}
}

// waitForRefill waits for the waiter's refill to finish, caller cancellation,
// limiter shutdown, or the limiter's finite internal deadline. Cancellation
// competes with refill completion under the bucket mutex, making the first
// decision definitive.
func (limiter *Limiter) waitForRefill(ctx context.Context, waiter *waiter) error {
	timer := time.NewTimer(limiter.maxWait)
	defer timer.Stop()

	var cancellation error
	select {
	case <-waiter.refill.done:
		return waiter.err
	case <-ctx.Done():
		cancellation = ctx.Err()
	case <-limiter.shutdown.ctx.Done():
		cancellation = ErrCapacityExceeded
	case <-timer.C:
		cancellation = ErrCapacityExceeded
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		cancellation = ctxErr
	}
	return waiter.refill.bucket.cancelWaiter(waiter, cancellation)
}

func closeRejectedRefill(refill *refill) {
	if !refill.active {
		close(refill.published)
	}
	close(refill.done)
}

// failCollectedRefills rejects every waiter in a collected batch without
// applying capacity.
func failCollectedRefills(pendingRefills map[*refill]LeaseRequest) {
	for refill := range pendingRefills {
		refill.bucket.rejectRefill(refill)
	}
}

// subjectKey identifies one subject kind and identifier in a batch response.
type subjectKey struct {
	kind SubjectKind
	id   string
}

// waiter represents one request admitted to a refill. element is non-nil only
// while the waiter belongs to the refill's FIFO queue.
type waiter struct {
	refill  *refill
	cost    int
	element *list.Element
	err     error
}
