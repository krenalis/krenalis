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
	"sync"
	"sync/atomic"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	batchSize  = 64
	batchDelay = 2 * time.Millisecond
	queueSize  = batchSize * 4

	refillBackoffDuration = 250 * time.Millisecond
	maxWaitDuration       = time.Second
	defaultAcquireTimeout = 5 * time.Second
)

// ErrCapacityExceeded is returned when a request cannot be served from local
// capacity or admitted to the current refill.
var ErrCapacityExceeded = errors.New("API rate-limit capacity exceeded")

// ErrInvalidCost is returned when an API request has an unsupported cost.
var ErrInvalidCost = errors.New("invalid API cost")

// Limiter coordinates refills for local buckets owned by organizations and
// workspaces. It owns the refill queue, batching, PostgreSQL lease acquisition,
// metrics, and shutdown lifecycle.
//
// Public consumption never accesses PostgreSQL directly.
type Limiter struct {
	acquireLeases  leaseAcquirer
	maxWait        time.Duration
	acquireTimeout time.Duration

	queue          chan *refill // bounded so queue publication never blocks.
	queueSaturated atomic.Bool  // suppresses repeated warnings while the queue is full.

	// retryAfter is the Unix-nanosecond deadline for the shared refill
	// backoff. It is atomic because every consumption checks it, while only the
	// single batcher updates it. A shared deadline prevents a database outage
	// from causing rapid retry batches for different buckets.
	retryAfter atomic.Int64

	acquisitionErrors prometheus.Counter
	queueFullCounter  prometheus.Counter

	shutdown struct {
		ctx    context.Context
		cancel context.CancelFunc
		sync.WaitGroup
	}
}

// leaseAcquirer acquires leases for a batch of organization,
// workspace, and ingestion subjects.
type leaseAcquirer func(context.Context, []leaseRequest) ([]leaseResult, error)

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

// New starts a limiter and its single refill batcher.
func New(database *db.DB) *Limiter {
	limiter := &Limiter{
		acquireLeases:  newLeaseAcquirer(database),
		maxWait:        maxWaitDuration,
		acquireTimeout: defaultAcquireTimeout,
		queue:          make(chan *refill, queueSize),
	}
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	limiter.shutdown.Add(1)
	go limiter.runBatcher()
	limiter.registerMetrics()
	return limiter
}

// Consume consumes cost from bucket. It returns ErrInvalidCost when cost is
// outside the bucket's supported range. If local capacity is insufficient, it
// may wait for the refill generation to which the request was admitted.
//
// It never accesses PostgreSQL directly.
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
		}
		if queueFull {
			if limiter.queueFullCounter != nil {
				limiter.queueFullCounter.Inc()
			}
			if limiter.queueSaturated.CompareAndSwap(false, true) {
				slog.Warn("core/state: API rate-limit refill queue is full")
			}
		}
		if queued {
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

// Close closes the limiter's refill lifecycle by preventing new refills,
// cancelling PostgreSQL work, and waiting for the batcher to stop.
// The batch currently being handled is finished without applying unconfirmed
// capacity, but entries still buffered in the queue are not drained.
// Their waiters observe the shutdown context and cannot remain blocked.
//
// Any remaining local leases are discarded. They are not returned because
// PostgreSQL has already subtracted them, and a best-effort return could credit
// the same capacity twice.
func (limiter *Limiter) Close() {
	limiter.shutdown.cancel()
	limiter.shutdown.Wait()
}

// backoffActive reports whether new refill attempts are temporarily suppressed
// after an acquisition error or an invalid batch response.
func (limiter *Limiter) backoffActive(now time.Time) bool {
	return now.UnixNano() < limiter.retryAfter.Load()
}

// collectAndRefillBatch collects refill generations for a short window, then
// requests their leases in one PostgreSQL call. It returns false when shutdown
// interrupts collection and the batcher should stop.
func (limiter *Limiter) collectAndRefillBatch(firstRefill *refill) bool {
	pendingRefills := map[*refill]leaseRequest{firstRefill: {}}
	timer := time.NewTimer(batchDelay)
	defer timer.Stop()
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
// batch, so a PostgreSQL outage cannot trigger rapid retries across the queue.
//
// Shutdown only finishes pending refills; it does not start a backoff.
func (limiter *Limiter) failRefillBatch(pendingRefills map[*refill]leaseRequest) {
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
func (limiter *Limiter) refillBatch(pendingRefills map[*refill]leaseRequest) {

	if limiter.backoffActive(time.Now()) {
		// Generations queued before a failed batch may still be in the channel.
		// They are not sent to PostgreSQL during global backoff. Rejecting them also
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
			slog.Error("core/state: cannot refill API rate-limit leases", "error", err)
			if limiter.acquisitionErrors != nil {
				limiter.acquisitionErrors.Inc()
			}
		}
		limiter.failRefillBatch(pendingRefills)
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
			limiter.failRefillBatch(pendingRefills)
			slog.Error("core/state: API rate-limit lease result does not match its request", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		if _, duplicate := resultsBySubject[key]; duplicate {
			limiter.failRefillBatch(pendingRefills)
			slog.Error("core/state: API rate-limit lease batch returned a duplicate result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
			return
		}
		invalid := result.GrantedUnits < 0 || result.GrantedUnits > requestedUnits ||
			result.CapacityUnits <= 0 || result.GrantedUnits > result.CapacityUnits
		if invalid {
			limiter.failRefillBatch(pendingRefills)
			slog.Error("core/state: API rate-limit lease batch returned an invalid result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID, "granted_units", result.GrantedUnits, "requested_units", requestedUnits, "capacity_units", result.CapacityUnits)
			return
		}
		resultsBySubject[key] = result
	}
	if len(resultsBySubject) != len(pendingRefills) {
		limiter.failRefillBatch(pendingRefills)
		slog.Error("core/state: API rate-limit lease batch returned incomplete results")
		return
	}

	for refill, request := range pendingRefills {
		key := subjectKey{kind: request.SubjectKind, id: request.SubjectID}
		result := resultsBySubject[key]
		refill.bucket.completeRefill(refill, result.GrantedUnits, result.CapacityUnits)
	}
	limiter.retryAfter.Store(0)

}

// registerMetrics registers the process-wide rate-limiter counters.
func (limiter *Limiter) registerMetrics() {
	limiter.acquisitionErrors = registerCounter(prometheus.CounterOpts{
		Name: "krenalis_api_rate_limit_refill_errors_total",
		Help: "Total number of API rate-limit lease refill errors",
	})
	limiter.queueFullCounter = registerCounter(prometheus.CounterOpts{
		Name: "krenalis_api_rate_limit_refill_queue_full_total",
		Help: "Total number of API rate-limit refill attempts rejected because the queue was full",
	})
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
func failCollectedRefills(pendingRefills map[*refill]leaseRequest) {
	for refill := range pendingRefills {
		refill.bucket.rejectRefill(refill)
	}
}

// newLeaseAcquirer returns an adapter that calls the PostgreSQL function
// acquire_api_rate_limit_leases.
func newLeaseAcquirer(database *db.DB) leaseAcquirer {
	return func(ctx context.Context, leaseRequests []leaseRequest) ([]leaseResult, error) {
		payload, err := json.Marshal(leaseRequests)
		if err != nil {
			return nil, fmt.Errorf("cannot encode API rate-limit lease requests: %w", err)
		}
		rows, err := database.Query(ctx, `
			SELECT subject_kind, subject_id, granted_units, capacity_units
			FROM acquire_api_rate_limit_leases($1::jsonb)`, string(payload))
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		leaseResults := make([]leaseResult, 0, len(leaseRequests))
		for rows.Next() {
			var result leaseResult
			if err := rows.Scan(&result.SubjectKind, &result.SubjectID, &result.GrantedUnits, &result.CapacityUnits); err != nil {
				return nil, err
			}
			leaseResults = append(leaseResults, result)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return leaseResults, nil
	}
}

// registerCounter registers a counter in the default Prometheus registry.
// If a counter with the same name is already registered, it returns the
// existing counter. It returns nil when registration fails or the existing
// collector is not a counter.
func registerCounter(counterOptions prometheus.CounterOpts) prometheus.Counter {
	counter := prometheus.NewCounter(counterOptions)
	if err := prometheus.Register(counter); err != nil {
		if registered, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if counter, ok := registered.ExistingCollector.(prometheus.Counter); ok {
				return counter
			}
		}
		slog.Error("core/state: cannot register API rate-limit metric", "metric", counterOptions.Name, "error", err)
		return nil
	}
	return counter
}

// leaseRequest is one input entry for the PostgreSQL lease acquisition
// function.
type leaseRequest struct {
	SubjectKind    subjectKind `json:"subject_kind"`
	SubjectID      string      `json:"subject_id"`
	RequestedUnits int         `json:"requested_units"`
}

// leaseResult is one result returned by the PostgreSQL lease acquisition
// function.
type leaseResult struct {
	SubjectKind   subjectKind
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
}

// subjectKey identifies one rate-limit subject in a batch response.
type subjectKey struct {
	kind subjectKind
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
