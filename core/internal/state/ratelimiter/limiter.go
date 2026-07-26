// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"container/list"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	batchSize  = 64
	batchDelay = 2 * time.Millisecond
	queueSize  = batchSize * 4

	refillBackoffDuration = 250 * time.Millisecond
)

var (
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

type acquireFunc func(context.Context, []leaseRequest) ([]leaseResult, error)

// Limiter coordinates refills for local buckets. It owns the refill queue,
// batching, backoff, and shutdown lifecycle.
type Limiter struct {
	db      *db.DB
	acquire acquireFunc

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
		done   chan struct{}
	}
}

// refill represents one refill generation, including its immutable lease
// request and the waiters admitted to it. Mutations are protected by bucket.mu.
// Closing done publishes final waiter results to readers.
// The published channel prevents the batcher from processing the refill before
// queue publication and activation are complete.
type refill struct {
	bucket      *Bucket
	request     leaseRequest
	active      bool
	published   chan struct{}
	done        chan struct{}
	waiters     list.List
	pendingCost int
}

// New starts a limiter backed by the rate-limit lease store in db.
func New(db *db.DB, metrics Metrics) *Limiter {
	limiter := &Limiter{
		db:      db,
		queue:   make(chan *refill, queueSize),
		metrics: metrics,
	}
	limiter.acquire = limiter.acquireLeases
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	limiter.shutdown.done = make(chan struct{})
	// Process one collected refill batch at a time.
	go func() {
		defer close(limiter.shutdown.done)
		for {
			select {
			case <-limiter.shutdown.ctx.Done():
				return
			case refill := <-limiter.queue:
				if !limiter.collectAndRefill(refill) {
					return
				}
			}
		}
	}()
	return limiter
}

// Close closes the limiter, waiting for shutdown until ctx is done.
func (limiter *Limiter) Close(ctx context.Context) {
	limiter.shutdown.cancel()
	select {
	case <-limiter.shutdown.done:
	case <-ctx.Done():
	}
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

// acquireLeases acquires rate-limit capacity from PostgreSQL.
func (limiter *Limiter) acquireLeases(ctx context.Context, requests []leaseRequest) ([]leaseResult, error) {
	encoded, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("cannot encode rate-limit lease requests: %w", err)
	}
	rows, err := limiter.db.Query(ctx, `
		SELECT subject_kind, subject_id, granted_units, capacity_units
		FROM acquire_rate_limit_leases($1::jsonb)`, string(encoded))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]leaseResult, 0, len(requests))
	for rows.Next() {
		var result leaseResult
		if err := rows.Scan(&result.SubjectKind, &result.SubjectID, &result.GrantedUnits, &result.CapacityUnits); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// backoffActive reports whether new refill attempts are temporarily suppressed
// after an acquisition error or an invalid batch response.
func (limiter *Limiter) backoffActive(now time.Time) bool {
	return now.UnixNano() < limiter.retryAfter.Load()
}

// collectAndRefill collects refill generations for a short window, then
// acquires their leases in one call. It returns false when shutdown interrupts
// collection and the batcher should stop.
func (limiter *Limiter) collectAndRefill(first *refill) bool {
	pendingRefills := map[*refill]leaseRequest{first: {}}
	timer := time.NewTimer(batchDelay)
	for len(pendingRefills) < batchSize {
		select {
		case <-limiter.shutdown.ctx.Done():
			failCollectedRefills(pendingRefills)
			return false
		case refill := <-limiter.queue:
			if _, exists := pendingRefills[refill]; !exists {
				pendingRefills[refill] = leaseRequest{}
			}
		case <-timer.C:
			limiter.refill(pendingRefills)
			return true
		}
	}
	limiter.refill(pendingRefills)
	return true
}

// failRefill fails a refill batch while preserving current local capacity and
// starts a fixed backoff that temporarily prevents new refill attempts.
// The shared deadline applies to every bucket, including those outside this
// batch, so an acquisition failure cannot trigger rapid retries across the
// queue.
//
// Shutdown only finishes pending refills; it does not start a backoff.
func (limiter *Limiter) failRefill(pendingRefills map[*refill]leaseRequest) {
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

// refill builds, acquires, validates, and applies leases for one collected
// batch. An acquisition error or invalid response fails the whole batch before
// any local capacity is changed.
func (limiter *Limiter) refill(pendingRefills map[*refill]leaseRequest) {
	if limiter.backoffActive(time.Now()) {
		// Generations queued before a failed batch may still be in the channel.
		// They are not acquired during global backoff. Rejecting them also
		// prevents waiters from relying on a refill that will not run.
		failCollectedRefills(pendingRefills)
		return
	}

	leaseRequests := make([]leaseRequest, 0, len(pendingRefills))
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

	acquireCtx, cancelAcquire := context.WithTimeout(limiter.shutdown.ctx, defaultAcquireTimeout)
	leaseResults, err := limiter.acquire(acquireCtx, leaseRequests)
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
		limiter.failRefill(pendingRefills)
		return
	}

	requestedUnitsBySubject := make(map[subjectKey]int, len(pendingRefills))
	for _, request := range pendingRefills {
		requestedUnitsBySubject[subjectKey{kind: request.SubjectKind, id: request.SubjectID}] = request.RequestedUnits
	}
	resultsBySubject := make(map[subjectKey]leaseResult, len(leaseResults))
	for _, result := range leaseResults {
		key := subjectKey{kind: result.SubjectKind, id: result.SubjectID}
		requestedUnits, ok := requestedUnitsBySubject[key]
		if !ok {
			limiter.failRefill(pendingRefills)
			slog.Error("rate limiter lease result does not match its request", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		if _, duplicate := resultsBySubject[key]; duplicate {
			limiter.failRefill(pendingRefills)
			slog.Error("rate limiter lease batch returned a duplicate result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		invalid := result.GrantedUnits < 0 || result.GrantedUnits > requestedUnits ||
			result.CapacityUnits <= 0 || result.GrantedUnits > result.CapacityUnits
		if invalid {
			limiter.failRefill(pendingRefills)
			slog.Error("rate limiter lease batch returned an invalid result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID, "granted_units", result.GrantedUnits, "requested_units", requestedUnits, "capacity_units", result.CapacityUnits)
			return
		}
		resultsBySubject[key] = result
	}
	if len(resultsBySubject) != len(pendingRefills) {
		limiter.failRefill(pendingRefills)
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

// waitForRefill waits for the waiter's refill to finish, caller cancellation,
// limiter shutdown, or the limiter's finite internal deadline. Cancellation
// competes with refill completion under the bucket mutex, making the first
// decision definitive.
func (limiter *Limiter) waitForRefill(ctx context.Context, waiter *waiter) error {
	timer := time.NewTimer(maxWaitDuration)
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
func failCollectedRefills(pendingRefills map[*refill]leaseRequest) {
	for refill := range pendingRefills {
		refill.bucket.rejectRefill(refill)
	}
}

// leaseRequest requests capacity for one subject kind and identifier.
type leaseRequest struct {
	SubjectKind    SubjectKind `json:"subject_kind"`
	SubjectID      string      `json:"subject_id"`
	RequestedUnits int         `json:"requested_units"`
}

// leaseResult reports capacity granted for one subject kind and identifier.
type leaseResult struct {
	SubjectKind   SubjectKind
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
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
