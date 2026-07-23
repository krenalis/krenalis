// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
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

// apiRateLimitLeaseAcquirer acquires leases for a batch containing
// organization, workspace, and ingestion buckets.
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
)

// ErrAPICapacityExceeded is returned when a subject does not currently have
// enough capacity in its API rate-limit bucket.
var ErrAPICapacityExceeded = errors.New("API rate-limit capacity exceeded")

// ErrInvalidAPICost is returned when an API operation has an unsupported
// cost.
var ErrInvalidAPICost = errors.New("invalid API cost")

// rateLimiter coordinates refills for every local bucket owned by a State.
// Organization and Workspace instances own their buckets, while rateLimiter
// owns the refill queue, batching, PostgreSQL lease acquisition, metrics, and
// close lifecycle. Public consumption never accesses PostgreSQL directly.
type rateLimiter struct {
	acquireLeases apiRateLimitLeaseAcquirer
	now           func() time.Time

	refillQueue chan *rateLimitBucket // bounded so consumption never blocks.
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
		acquireLeases: newAPIRateLimitLeaseAcquirer(database),
		now:           time.Now,
		refillQueue:   make(chan *rateLimitBucket, apiRateLimitQueueSize),
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
// not drained.
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

// collectAndRefillBatch collects distinct buckets for a short window, then
// requests their leases in one PostgreSQL call. It returns false when close
// interrupts collection and the batcher should stop.
func (limiter *rateLimiter) collectAndRefillBatch(firstBucket *rateLimitBucket) bool {
	pendingRefills := map[*rateLimitBucket]apiRateLimitLeaseRequest{firstBucket: {}}
	timer := time.NewTimer(apiRateLimitBatchDelay)
	defer timer.Stop()
	for len(pendingRefills) < apiRateLimitBatchSize {
		select {
		case <-limiter.close.ctx.Done():
			finishCollectedRefills(pendingRefills)
			return false
		case bucket := <-limiter.refillQueue:
			if _, exists := pendingRefills[bucket]; !exists {
				pendingRefills[bucket] = apiRateLimitLeaseRequest{}
			}
		case <-timer.C:
			limiter.refillBatch(pendingRefills)
			return true
		}
	}
	limiter.refillBatch(pendingRefills)
	return true
}

// consume validates an operation cost, consumes from the local bucket, and
// queues a refill when requested by the bucket. It never waits for PostgreSQL.
func (limiter *rateLimiter) consume(bucket *rateLimitBucket, operationCost int) error {
	if operationCost < 1 || operationCost > bucket.maxCost {
		return ErrInvalidAPICost
	}
	now := limiter.now()
	refillAllowed := !limiter.closed.Load() && !limiter.refillBackoffActive(now)
	capacityExceeded, shouldQueueRefill := bucket.consume(operationCost, refillAllowed)
	if shouldQueueRefill {
		queued, queueFull := limiter.queueRefill(bucket)
		if !queued {
			bucket.finishRefill()
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
		}
	}
	if capacityExceeded {
		return ErrAPICapacityExceeded
	}
	return nil
}

// failRefillBatch keeps the current local capacity, including any overdraft,
// and prevents new refill attempts until the fixed backoff has elapsed.
// The shared deadline applies to every bucket, including those outside this
// batch, so a PostgreSQL outage cannot trigger rapid retries across the queue.
// Shutdown only finishes pending refills; it does not start a backoff.
func (limiter *rateLimiter) failRefillBatch(pendingRefills map[*rateLimitBucket]apiRateLimitLeaseRequest) {
	if limiter.close.ctx.Err() != nil {
		finishCollectedRefills(pendingRefills)
		return
	}
	now := limiter.now()
	limiter.refillRetryAfter.Store(now.Add(apiRateLimitRefillBackoffDuration).UnixNano())
	finishCollectedRefills(pendingRefills)
}

// queueRefill adds a bucket to the refill queue without blocking the request
// path.
func (limiter *rateLimiter) queueRefill(bucket *rateLimitBucket) (queued, queueFull bool) {
	if limiter.closed.Load() {
		return false, false
	}
	select {
	case limiter.refillQueue <- bucket:
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
func (limiter *rateLimiter) refillBatch(pendingRefills map[*rateLimitBucket]apiRateLimitLeaseRequest) {

	if limiter.refillBackoffActive(limiter.now()) {
		// Buckets queued before a failed batch may still be in the channel. They
		// are not sent to PostgreSQL during the global backoff, and clearing their
		// pending marker also prevents them from using overdraft in the meantime.
		finishCollectedRefills(pendingRefills)
		return
	}

	leaseRequests := make([]apiRateLimitLeaseRequest, 0, len(pendingRefills))
	for bucket := range pendingRefills {
		request, needed := bucket.refillRequest()
		if !needed {
			delete(pendingRefills, bucket)
			continue
		}
		pendingRefills[bucket] = request
		leaseRequests = append(leaseRequests, request)
	}
	if len(leaseRequests) == 0 {
		return
	}

	leaseResults, err := limiter.acquireLeases(limiter.close.ctx, leaseRequests)
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

	for bucket, request := range pendingRefills {
		key := apiRateLimitSubjectKey{kind: request.SubjectKind, id: request.SubjectID}
		result := resultsBySubject[key]
		bucket.applyLease(result.GrantedUnits, result.CapacityUnits)
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
		case firstBucket := <-limiter.refillQueue:
			if !limiter.collectAndRefillBatch(firstBucket) {
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
	refillQueued          bool
	disabled              bool
	allowInitialOverdraft bool
	leaseSize             int
	maxCost               int
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

// newNonspecificBucket creates the empty local bucket for nonspecific requests
// in an organization.
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

// applyLease adds capacity that PostgreSQL has already removed from the
// corresponding authoritative bucket. Local requests may consume capacity while
// query is running, so the granted units are added to the current value rather
// than replacing it.
//
// The upper bound is capped at the local target. The lower bound is not capped:
// a partial lease may leave a negative balance that later leases must repay.
func (bucket *rateLimitBucket) applyLease(grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.disabled {
		bucket.refillQueued = false
		return
	}
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
	bucket.refillQueued = false
}

// consume attempts to deduct an operation cost from the local bucket. It also
// marks a refill for queueing when capacity is low relative to either the
// bucket threshold or the operation cost, or when the operation cannot be
// served.
//
// A cost-1 operation may temporarily make the bucket negative, but only while a
// refill is already queued or in progress and refills are currently allowed.
func (bucket *rateLimitBucket) consume(operationCost int, refillAllowed bool) (capacityExceeded, shouldQueueRefill bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.disabled {
		return true, false
	}
	usingInitialOverdraft := operationCost == 1 && !bucket.refillQueued && bucket.allowInitialOverdraft
	if bucket.available >= operationCost {
		bucket.available -= operationCost
	} else if operationCost == 1 && refillAllowed && bucket.available > -apiRateLimitOverdraftLimit &&
		(bucket.refillQueued || usingInitialOverdraft) {
		bucket.available--
		if usingInitialOverdraft {
			bucket.allowInitialOverdraft = false
		}
	} else {
		capacityExceeded = true
	}
	needsRefill := capacityExceeded || bucket.available < bucket.threshold || bucket.available <= operationCost
	if needsRefill && !bucket.refillQueued && refillAllowed {
		bucket.refillQueued = true
		shouldQueueRefill = true
	}
	return capacityExceeded, shouldQueueRefill
}

// disable prevents further consumption and refills after the subject is
// removed from State. A queued pointer remains safe because Go keeps the bucket
// alive; refillRequest and applyLease both discard work for disabled buckets.
func (bucket *rateLimitBucket) disable() {
	bucket.mu.Lock()
	bucket.disabled = true
	bucket.available = 0
	bucket.refillQueued = false
	bucket.mu.Unlock()
}

// finishRefill clears the pending marker when a refill finishes without
// applying a lease.
func (bucket *rateLimitBucket) finishRefill() {
	bucket.mu.Lock()
	bucket.refillQueued = false
	bucket.mu.Unlock()
}

// refillRequest builds the PostgreSQL lease request for a queued refill. A
// disabled bucket clears its pending marker without accessing PostgreSQL.
func (bucket *rateLimitBucket) refillRequest() (apiRateLimitLeaseRequest, bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.disabled {
		bucket.refillQueued = false
		return apiRateLimitLeaseRequest{}, false
	}
	// A negative local balance can make the amount missing from the target
	// larger than one standard lease. Each batch entry still requests at most
	// one lease.
	requestedUnits := min(bucket.leaseSize, bucket.target-bucket.available)
	if bucket.target == 0 {
		requestedUnits = bucket.leaseSize
	}
	if requestedUnits <= 0 {
		bucket.refillQueued = false
		return apiRateLimitLeaseRequest{}, false
	}
	return apiRateLimitLeaseRequest{
		SubjectKind:    bucket.subjectKind,
		SubjectID:      bucket.subjectID,
		RequestedUnits: requestedUnits,
	}, true
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

// finishCollectedRefills clears the pending marker for every bucket in a
// collected batch without applying capacity.
func finishCollectedRefills(pendingRefills map[*rateLimitBucket]apiRateLimitLeaseRequest) {
	for bucket := range pendingRefills {
		bucket.finishRefill()
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
