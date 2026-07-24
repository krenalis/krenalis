// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

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

// apiRateLimitLeaseAcquirer acquires leases for a batch of organization,
// workspace, and ingestion subjects.
type apiRateLimitLeaseAcquirer func(context.Context, []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error)

const (
	apiRateLimitMaxCost            = 100
	apiRateLimitLeaseSize          = apiRateLimitMaxCost
	ingestionRateLimitMaxCost      = 20_000
	ingestionRateLimitLeaseSize    = ingestionRateLimitMaxCost
	apiRateLimitMaxRefillThreshold = 25

	apiRateLimitBatchSize  = 64
	apiRateLimitBatchDelay = 2 * time.Millisecond
	apiRateLimitQueueSize  = apiRateLimitBatchSize * 4

	// apiRateLimitOverdraftLimit bounds the temporary negative balance allowed
	// for cost-1 operations while a refill is pending. For one subject, each
	// node can exceed global capacity by at most this many operations.
	apiRateLimitOverdraftLimit = 5

	apiRateLimitRefillBackoffDuration = 250 * time.Millisecond
	apiRateLimitMaxWaitDuration       = time.Second
	apiRateLimitAcquireTimeout        = 5 * time.Second
)

// ErrAPICapacityExceeded is returned when a request cannot be served from local
// capacity or admitted to the current refill.
var ErrAPICapacityExceeded = errors.New("API rate-limit capacity exceeded")

// ErrInvalidAPICost is returned when an API request has an unsupported cost.
var ErrInvalidAPICost = errors.New("invalid API cost")

// rateLimiter coordinates refills for every local bucket owned by a State.
// Organization and Workspace instances own their buckets, while rateLimiter
// owns the refill queue, batching, PostgreSQL lease acquisition, metrics, and
// close lifecycle. Public consumption never accesses PostgreSQL directly.
type rateLimiter struct {
	acquireLeases  apiRateLimitLeaseAcquirer
	now            func() time.Time
	maxWait        time.Duration
	acquireTimeout time.Duration

	refillQueue chan *rateLimitRefill // bounded so queue publication never blocks.
	closed      atomic.Bool           // prevents enqueue after close starts.

	// refillRetryAfter is the Unix-nanosecond deadline for the shared refill
	// backoff. It is atomic because every consumption checks it, while only the
	// single batcher updates it. A shared deadline prevents a database outage
	// from causing rapid retry batches for different buckets.
	refillRetryAfter atomic.Int64

	refillErrors         prometheus.Counter
	refillQueueFull      prometheus.Counter
	refillQueueSaturated atomic.Bool // suppresses repeated warnings while the queue is full.
	close                struct {
		ctx    context.Context
		cancel context.CancelFunc
		sync.WaitGroup
	}
}

// newRateLimiter starts the single refill batcher used by a State.
func newRateLimiter(database *db.DB) *rateLimiter {
	limiter := &rateLimiter{
		acquireLeases:  newAPIRateLimitLeaseAcquirer(database),
		now:            time.Now,
		maxWait:        apiRateLimitMaxWaitDuration,
		acquireTimeout: apiRateLimitAcquireTimeout,
		refillQueue:    make(chan *rateLimitRefill, apiRateLimitQueueSize),
	}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.Add(1)
	go limiter.runBatcher()
	limiter.registerMetrics()
	return limiter
}

// Close prevents new refills, cancels PostgreSQL work, and waits for the
// batcher to stop. The batch currently being handled is finished without
// applying unconfirmed capacity, but entries still buffered in the queue are
// not drained. Their waiters observe the close context and cannot remain
// blocked.
//
// Any remaining local leases are discarded. They are not returned because
// PostgreSQL has already subtracted them, and a best-effort return could credit
// the same capacity twice.
func (limiter *rateLimiter) Close() {
	if !limiter.closed.CompareAndSwap(false, true) {
		return
	}
	limiter.close.cancel()
	limiter.close.Wait()
}

// collectAndRefillBatch collects refill generations for a short window, then
// requests their leases in one PostgreSQL call. It returns false when close
// interrupts collection and the batcher should stop.
func (limiter *rateLimiter) collectAndRefillBatch(firstRefill *rateLimitRefill) bool {
	pendingRefills := map[*rateLimitRefill]apiRateLimitLeaseRequest{firstRefill: {}}
	timer := time.NewTimer(apiRateLimitBatchDelay)
	defer timer.Stop()
	for len(pendingRefills) < apiRateLimitBatchSize {
		select {
		case <-limiter.close.ctx.Done():
			failCollectedRefills(pendingRefills)
			return false
		case refill := <-limiter.refillQueue:
			if _, exists := pendingRefills[refill]; !exists {
				pendingRefills[refill] = apiRateLimitLeaseRequest{}
			}
		case <-timer.C:
			limiter.refillBatch(pendingRefills)
			return true
		}
	}
	limiter.refillBatch(pendingRefills)
	return true
}

// consume validates a request cost and consumes from the local bucket. If local
// capacity is insufficient, it may wait for the refill generation to which the
// request was admitted. It never accesses PostgreSQL directly.
func (limiter *rateLimiter) consume(ctx context.Context, bucket *rateLimitBucket, operationCost int) error {
	if operationCost < 1 || operationCost > bucket.maxCost {
		return ErrInvalidAPICost
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	now := limiter.now()
	refillAllowed := !limiter.closed.Load() && !limiter.refillBackoffActive(now)
	satisfied, refill, waiter := bucket.consume(operationCost, refillAllowed)
	if refill != nil {
		queued, queueFull := limiter.queueRefill(refill)
		if !queued {
			bucket.rejectRefill(refill)
		}
		if queueFull {
			if limiter.refillQueueFull != nil {
				limiter.refillQueueFull.Inc()
			}
			if limiter.refillQueueSaturated.CompareAndSwap(false, true) {
				slog.Warn("core/state: API rate-limit refill queue is full")
			}
		}
		if queued {
			limiter.refillQueueSaturated.Store(false)
			refillAllowed = !limiter.closed.Load() && !limiter.refillBackoffActive(limiter.now())
			waiter = bucket.activateRefill(refill, operationCost, !satisfied, refillAllowed)
		}
	}
	if satisfied {
		return nil
	}
	if waiter == nil {
		return ErrAPICapacityExceeded
	}
	return limiter.waitForRefill(ctx, waiter)
}

// waitForRefill waits for the waiter's refill to finish, caller cancellation,
// limiter shutdown, or the limiter's finite internal deadline. Cancellation
// competes with refill completion under the bucket mutex, making the first
// decision definitive.
func (limiter *rateLimiter) waitForRefill(ctx context.Context, waiter *rateLimitWaiter) error {
	timer := time.NewTimer(limiter.maxWait)
	defer timer.Stop()

	var cancellation error
	select {
	case <-waiter.refill.done:
		return waiter.err
	case <-ctx.Done():
		cancellation = ctx.Err()
	case <-limiter.close.ctx.Done():
		cancellation = ErrAPICapacityExceeded
	case <-timer.C:
		cancellation = ErrAPICapacityExceeded
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		cancellation = ctxErr
	}
	return waiter.refill.bucket.cancelWaiter(waiter, cancellation)
}

// failRefillBatch preserves the current local capacity, including any
// overdraft, and starts a fixed backoff that temporarily prevents new refill
// attempts.
// The shared deadline applies to every bucket, including those outside this
// batch, so a PostgreSQL outage cannot trigger rapid retries across the queue.
// Shutdown only finishes pending refills; it does not start a backoff.
func (limiter *rateLimiter) failRefillBatch(pendingRefills map[*rateLimitRefill]apiRateLimitLeaseRequest) {
	if limiter.close.ctx.Err() != nil {
		failCollectedRefills(pendingRefills)
		return
	}
	now := limiter.now()
	limiter.refillRetryAfter.Store(now.Add(apiRateLimitRefillBackoffDuration).UnixNano())
	failCollectedRefills(pendingRefills)
}

// queueRefill adds a generation to the refill queue without blocking the
// request path.
func (limiter *rateLimiter) queueRefill(refill *rateLimitRefill) (queued, queueFull bool) {
	if limiter.closed.Load() {
		return false, false
	}
	select {
	case limiter.refillQueue <- refill:
		return true, false
	default:
		return false, true
	}
}

// refillBackoffActive reports whether new refill attempts are temporarily
// suppressed after an acquisition error or an invalid batch response.
func (limiter *rateLimiter) refillBackoffActive(now time.Time) bool {
	return now.UnixNano() < limiter.refillRetryAfter.Load()
}

// refillBatch builds, acquires, validates, and applies leases for one collected
// batch. An acquisition error or invalid response fails the whole batch before
// any local capacity is changed.
func (limiter *rateLimiter) refillBatch(pendingRefills map[*rateLimitRefill]apiRateLimitLeaseRequest) {

	if limiter.refillBackoffActive(limiter.now()) {
		// Generations queued before a failed batch may still be in the channel.
		// They are not sent to PostgreSQL during global backoff. Rejecting them also
		// prevents further overdraft from relying on a refill that will not run.
		failCollectedRefills(pendingRefills)
		return
	}

	leaseRequests := make([]apiRateLimitLeaseRequest, 0, len(pendingRefills))
	for refill := range pendingRefills {
		select {
		case <-refill.published:
		case <-limiter.close.ctx.Done():
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
		acquireTimeout = apiRateLimitAcquireTimeout
	}
	acquireCtx, cancelAcquire := context.WithTimeout(limiter.close.ctx, acquireTimeout)
	leaseResults, err := limiter.acquireLeases(acquireCtx, leaseRequests)
	if err == nil {
		err = acquireCtx.Err()
	}
	cancelAcquire()
	if err != nil {
		if limiter.close.ctx.Err() == nil {
			slog.Error("core/state: cannot refill API rate-limit leases", "error", err)
			if limiter.refillErrors != nil {
				limiter.refillErrors.Inc()
			}
		}
		limiter.failRefillBatch(pendingRefills)
		return
	}

	requestedUnitsBySubject := make(map[apiRateLimitSubjectKey]int, len(pendingRefills))
	for _, request := range pendingRefills {
		requestedUnitsBySubject[apiRateLimitSubjectKey{kind: request.SubjectKind, id: request.SubjectID}] = request.RequestedUnits
	}
	resultsBySubject := make(map[apiRateLimitSubjectKey]apiRateLimitLeaseResult, len(leaseResults))
	for _, result := range leaseResults {
		key := apiRateLimitSubjectKey{kind: result.SubjectKind, id: result.SubjectID}
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
		key := apiRateLimitSubjectKey{kind: request.SubjectKind, id: request.SubjectID}
		result := resultsBySubject[key]
		refill.bucket.completeRefill(refill, result.GrantedUnits, result.CapacityUnits)
	}
	limiter.refillRetryAfter.Store(0)

}

// registerMetrics initializes the process-wide rate-limiter counters.
func (limiter *rateLimiter) registerMetrics() {
	limiter.refillErrors = registerAPIRateLimiterCounter(prometheus.CounterOpts{
		Name: "krenalis_api_rate_limit_refill_errors_total",
		Help: "Total number of API rate-limit lease refill errors",
	})
	limiter.refillQueueFull = registerAPIRateLimiterCounter(prometheus.CounterOpts{
		Name: "krenalis_api_rate_limit_refill_queue_full_total",
		Help: "Total number of API rate-limit refill attempts rejected because the queue was full",
	})
}

// runBatcher processes one collected refill batch at a time until Close
// cancels the limiter context.
func (limiter *rateLimiter) runBatcher() {
	defer limiter.close.Done()
	for {
		select {
		case <-limiter.close.ctx.Done():
			return
		case firstRefill := <-limiter.refillQueue:
			if !limiter.collectAndRefillBatch(firstRefill) {
				return
			}
		}
	}
}

// rateLimitBucket stores process-local capacity for one organization, workspace,
// or workspace ingestion subject. Its constructors set the subject kind, so
// callers cannot create an inconsistent subject combination.
//
// mu protects local capacity and refill state. It is deliberately separate from
// State and entity locks, so consuming API capacity does not contend on State
// maps or unrelated organization data.
type rateLimitBucket struct {
	mu                    sync.Mutex
	subjectKind           string
	subjectID             string
	available             int
	target                int
	threshold             int
	refill                *rateLimitRefill
	disabled              bool
	allowInitialOverdraft bool
	leaseSize             int
	maxCost               int
}

// rateLimitRefill represents one refill generation, including its immutable
// lease request and the waiters admitted to it. Mutations are protected by
// bucket.mu. Closing done publishes final waiter results to readers. The
// published channel prevents the batcher from processing the refill before
// queue publication and activation are complete.
type rateLimitRefill struct {
	bucket      *rateLimitBucket
	request     apiRateLimitLeaseRequest
	active      bool
	published   chan struct{}
	done        chan struct{}
	waiters     list.List
	pendingCost int
}

// rateLimitWaiter represents one request admitted to a refill. element is
// non-nil only while the waiter belongs to the refill's FIFO queue.
type rateLimitWaiter struct {
	refill  *rateLimitRefill
	cost    int
	element *list.Element
	err     error
}

// newWorkspaceBucket creates the empty local bucket owned by a Workspace
// instance.
func newWorkspaceBucket(workspaceID string) *rateLimitBucket {
	return newRateLimitBucket("workspace", workspaceID, apiRateLimitLeaseSize, apiRateLimitMaxCost)
}

// newIngestionBucket creates the empty local ingestion bucket owned by a
// Workspace instance.
func newIngestionBucket(workspaceID string) *rateLimitBucket {
	bucket := newRateLimitBucket("ingestion", workspaceID, ingestionRateLimitLeaseSize, ingestionRateLimitMaxCost)
	bucket.allowInitialOverdraft = true
	return bucket
}

// newNonspecificBucket creates an organization's empty local bucket for
// nonspecific requests.
func newNonspecificBucket(organizationID string) *rateLimitBucket {
	return newRateLimitBucket("nonspecific", organizationID, apiRateLimitLeaseSize, apiRateLimitMaxCost)
}

// newRateLimitBucket creates an empty local bucket for the specified subject.
func newRateLimitBucket(subjectKind, subjectID string, leaseSize, maxCost int) *rateLimitBucket {
	return &rateLimitBucket{
		subjectKind: subjectKind,
		subjectID:   subjectID,
		leaseSize:   leaseSize,
		maxCost:     maxCost,
	}
}

// applyLeaseLocked adds capacity already removed from PostgreSQL, capped at the
// local target. A partial lease may leave the local balance negative.
func (bucket *rateLimitBucket) applyLeaseLocked(grantedUnits, capacityUnits int) {
	bucket.target = min(bucket.leaseSize, capacityUnits)
	bucket.allowInitialOverdraft = false
	// A fixed threshold would trigger a refill after almost every request when
	// the local target is small. Scale it with the target, with one unit as the
	// minimum.
	bucket.threshold = max(1, min(apiRateLimitMaxRefillThreshold, bucket.target/4))
	if bucket.available > bucket.target {
		bucket.available = bucket.target
	}
	bucket.available = min(bucket.target, bucket.available+grantedUnits)
}

// consume tries to serve a request from local capacity, admit it to an active
// refill, or prepare a new refill. A returned refill is still being published
// and must be placed on the limiter queue before it can admit a waiter.
func (bucket *rateLimitBucket) consume(operationCost int, refillAllowed bool) (satisfied bool, refill *rateLimitRefill, waiter *rateLimitWaiter) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.disabled {
		return false, nil, nil
	}

	if bucket.available >= operationCost {
		bucket.available -= operationCost
		satisfied = true
	} else if operationCost == 1 && refillAllowed && bucket.refill == nil && bucket.allowInitialOverdraft {
		// Ingestion may serve its first single event immediately before publishing the
		// cold bucket's initial refill.
		bucket.available--
		bucket.allowInitialOverdraft = false
		satisfied = true
	} else if operationCost == 1 && refillAllowed && bucket.available > -apiRateLimitOverdraftLimit &&
		bucket.refill != nil && bucket.refill.active && bucket.refill.waiters.Len() == 0 {
		bucket.available--
		satisfied = true
	} else if refillAllowed && bucket.refill != nil && bucket.refill.active {
		return false, nil, bucket.admitWaiterLocked(bucket.refill, operationCost)
	}

	if !satisfied && bucket.refill != nil {
		// Requests racing with the short publishing phase are deliberately not
		// admitted. Callers may retry after publication has been confirmed.
		return false, nil, nil
	}
	needsRefill := !satisfied || bucket.available < bucket.threshold || bucket.available <= operationCost
	if needsRefill && bucket.refill == nil && refillAllowed {
		refill = bucket.newRefillLocked()
	}
	return satisfied, refill, nil
}

func (bucket *rateLimitBucket) newRefillLocked() *rateLimitRefill {
	requestedUnits := min(bucket.leaseSize, bucket.target-bucket.available)
	if bucket.target == 0 {
		requestedUnits = bucket.leaseSize
	}
	refill := &rateLimitRefill{
		bucket: bucket,
		request: apiRateLimitLeaseRequest{
			SubjectKind:    bucket.subjectKind,
			SubjectID:      bucket.subjectID,
			RequestedUnits: requestedUnits,
		},
		published: make(chan struct{}),
		done:      make(chan struct{}),
	}
	bucket.refill = refill
	return refill
}

// activateRefill confirms successful queue publication and optionally admits
// the request that created the refill. Activation and admission are one atomic
// bucket transition, so the batcher cannot complete the refill between them.
func (bucket *rateLimitBucket) activateRefill(refill *rateLimitRefill, operationCost int, admit bool, refillAllowed bool) *rateLimitWaiter {
	bucket.mu.Lock()
	if bucket.refill != refill || refill.active {
		bucket.mu.Unlock()
		return nil
	}
	if bucket.disabled || !refillAllowed {
		rejected := bucket.rejectRefillLocked(refill)
		bucket.mu.Unlock()
		if rejected {
			closeRejectedRefill(refill)
		}
		return nil
	}
	refill.active = true
	var waiter *rateLimitWaiter
	if admit {
		waiter = bucket.admitWaiterLocked(refill, operationCost)
	}
	bucket.mu.Unlock()
	close(refill.published)
	return waiter
}

func (bucket *rateLimitBucket) admitWaiterLocked(refill *rateLimitRefill, operationCost int) *rateLimitWaiter {
	debt := max(0, -bucket.available)
	if refill.pendingCost+operationCost > refill.request.RequestedUnits-debt {
		return nil
	}
	waiter := &rateLimitWaiter{refill: refill, cost: operationCost}
	waiter.element = refill.waiters.PushBack(waiter)
	refill.pendingCost += operationCost
	return waiter
}

// disable prevents further consumption and refills after the subject is
// removed from State, clears local capacity, and rejects pending waiters. A
// queued pointer remains safe because Go keeps the bucket alive; refillRequest
// and completeRefill discard later work for disabled buckets.
func (bucket *rateLimitBucket) disable() {
	bucket.mu.Lock()
	bucket.disabled = true
	bucket.available = 0
	refill := bucket.refill
	rejected := bucket.rejectRefillLocked(refill)
	bucket.mu.Unlock()
	if rejected {
		closeRejectedRefill(refill)
	}
}

// rejectRefill rejects every remaining waiter and detaches the generation.
func (bucket *rateLimitBucket) rejectRefill(refill *rateLimitRefill) {
	bucket.mu.Lock()
	rejected := bucket.rejectRefillLocked(refill)
	bucket.mu.Unlock()
	if rejected {
		closeRejectedRefill(refill)
	}
}

func (bucket *rateLimitBucket) rejectRefillLocked(refill *rateLimitRefill) bool {
	if refill == nil || bucket.refill != refill {
		return false
	}
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*rateLimitWaiter)
		waiter.err = ErrAPICapacityExceeded
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingCost = 0
	bucket.refill = nil
	return true
}

func closeRejectedRefill(refill *rateLimitRefill) {
	if !refill.active {
		close(refill.published)
	}
	close(refill.done)
}

// refillRequest returns the immutable request frozen before queue publication.
func (bucket *rateLimitBucket) refillRequest(refill *rateLimitRefill) (apiRateLimitLeaseRequest, bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.disabled || bucket.refill != refill || !refill.active {
		return apiRateLimitLeaseRequest{}, false
	}
	return refill.request, true
}

// completeRefill applies a valid grant and serves the longest satisfiable FIFO
// prefix while holding the bucket mutex. Capacity is deducted before any waiter
// is awakened, so later requests cannot consume assigned units.
func (bucket *rateLimitBucket) completeRefill(refill *rateLimitRefill, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	if bucket.disabled || bucket.refill != refill || !refill.active {
		bucket.mu.Unlock()
		return
	}
	bucket.applyLeaseLocked(grantedUnits, capacityUnits)
	serve := true
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*rateLimitWaiter)
		if serve && bucket.available >= waiter.cost {
			bucket.available -= waiter.cost
			waiter.err = nil
		} else {
			serve = false
			waiter.err = ErrAPICapacityExceeded
		}
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingCost = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	close(refill.done)
}

// cancelWaiter serializes cancellation with refill completion. If completion
// acquired the bucket mutex first, its already-final decision is returned.
func (bucket *rateLimitBucket) cancelWaiter(waiter *rateLimitWaiter, cancellation error) error {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if waiter.element == nil {
		return waiter.err
	}
	refill := waiter.refill
	refill.waiters.Remove(waiter.element)
	refill.pendingCost -= waiter.cost
	waiter.element = nil
	waiter.err = cancellation
	return cancellation
}

// apiRateLimitSubjectKey identifies one rate-limit subject in a batch response.
type apiRateLimitSubjectKey struct {
	kind string
	id   string
}

// apiRateLimitLeaseRequest is one input entry for the PostgreSQL lease
// acquisition function.
type apiRateLimitLeaseRequest struct {
	SubjectKind    string `json:"subject_kind"`
	SubjectID      string `json:"subject_id"`
	RequestedUnits int    `json:"requested_units"`
}

// apiRateLimitLeaseResult is one result returned by the PostgreSQL lease
// acquisition function.
type apiRateLimitLeaseResult struct {
	SubjectKind   string
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
}

// failCollectedRefills rejects every waiter in a collected batch without
// applying capacity.
func failCollectedRefills(pendingRefills map[*rateLimitRefill]apiRateLimitLeaseRequest) {
	for refill := range pendingRefills {
		refill.bucket.rejectRefill(refill)
	}
}

// newAPIRateLimitLeaseAcquirer returns an adapter that calls the PostgreSQL
// function acquire_api_rate_limit_leases.
func newAPIRateLimitLeaseAcquirer(database *db.DB) apiRateLimitLeaseAcquirer {
	return func(ctx context.Context, leaseRequests []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
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

		leaseResults := make([]apiRateLimitLeaseResult, 0, len(leaseRequests))
		for rows.Next() {
			var result apiRateLimitLeaseResult
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

// Metrics are process-wide. Reuse an existing collector if another State has
// already registered it, and never unregister it while another State may still
// be running.
func registerAPIRateLimiterCounter(counterOptions prometheus.CounterOpts) prometheus.Counter {
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
