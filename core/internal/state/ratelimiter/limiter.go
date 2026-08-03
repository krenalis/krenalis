// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	// batchSize is the maximum number of refill generations acquired in a
	// single PostgreSQL call.
	batchSize = 64

	// batchDelay is the collection interval used to coalesce refill and
	// restoration work into batches.
	batchDelay = 2 * time.Millisecond

	// queueSize is the maximum number of refill generations waiting to be
	// collected.
	queueSize = batchSize * 4

	// refillBackoffDuration is the delay during which new refills are rejected
	// after an acquisition error or invalid response.
	refillBackoffDuration = 250 * time.Millisecond

	// bucketCompactionInterval controls how often weak references to
	// unreachable buckets are removed.
	bucketCompactionInterval = 5 * time.Minute
)

// defaultAcquireTimeout limits the duration of a single PostgreSQL
// lease-acquisition operation. It is a variable so tests can use a shorter
// duration.
var defaultAcquireTimeout = 5 * time.Second

// ErrLimiterUnavailable is returned when the limiter cannot determine
// whether the requested capacity is available because of a temporary
// condition.
var ErrLimiterUnavailable = errors.New("rate limiter unavailable")

// CapacityExceededError is returned when the requested capacity is unavailable.
type CapacityExceededError struct {
	RetryAfter time.Duration
}

func (err CapacityExceededError) Error() string {
	return "requested rate-limit capacity is unavailable"
}

// Metrics contains optional counters for rate-limiter events.
type Metrics struct {
	AcquisitionErrors Counter
	QueueFull         Counter
}

// Counter records a rate-limiter event.
type Counter interface {
	Inc()
}

// SubjectKind identifies a class of rate-limit buckets.
type SubjectKind string

// Limiter manages node-local rate-limit capacity for the buckets it creates.
//
// Create one limiter for each application node, then create a bucket for every
// subject that needs to be rate-limited:
//
//	limiter := New(db, metrics)
//	bucket := limiter.NewBucket(subjectKind, subjectID, leaseSize, maxUnits)
type Limiter struct {
	db           *db.DB           // PostgreSQL backing store
	acquire      acquireFunc      // lease acquisition function
	restoreBatch restoreBatchFunc // capacity restoration function

	queue           chan *refill                 // pending refill generations
	queueFullLogged atomic.Bool                  // whether the current queue-full condition was logged
	backoff         atomic.Pointer[backoffState] // current acquisition backoff, if any
	restorations    restorationQueue             // capacity waiting to be restored

	metrics Metrics // event counters

	buckets struct { // tracked buckets
		sync.Mutex
		refs []weak.Pointer[Bucket] // weak references to created buckets; protected by the embedded mutex
	}

	close struct {
		ctx            context.Context    // canceled when shutdown starts
		cancel         context.CancelFunc // cancels ctx
		atomic.Bool                       // whether shutdown has started
		sync.WaitGroup                    // active background workers
	}
}

// New creates and starts a limiter backed by the rate-limit lease store in db.
// Call Close when the limiter is no longer needed.
func New(db *db.DB, metrics Metrics) *Limiter {
	limiter := &Limiter{
		db:      db,
		queue:   make(chan *refill, queueSize),
		metrics: metrics,
	}
	limiter.restorations = newRestorationQueue()
	limiter.acquire = limiter.acquireLeases
	limiter.restoreBatch = limiter.restoreCapacityBatch
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.Go(limiter.runRefiller)
	limiter.close.Go(limiter.runRestorer)
	limiter.close.Go(limiter.runCompactor)
	return limiter
}

// Close shuts down the limiter and waits for all its operations to stop, even
// if ctx is canceled.
//
// When Close is called, no other calls to Limiter's methods or to methods of
// buckets created by it should be in progress and no other shall be made. The
// caller must keep the buckets for existing subjects reachable until Close
// returns. Subsequent calls to Close return immediately.
func (limiter *Limiter) Close(ctx context.Context) {
	if limiter.close.Swap(true) {
		return
	}
	limiter.close.cancel()
	limiter.close.Wait()
	if ctx.Err() != nil {
		return
	}
	err := limiter.restoreUnusedCapacity(ctx, limiter.collectUnusedCapacity())
	if err != nil && ctx.Err() == nil {
		slog.Warn("cannot restore unused rate-limit capacity", "error", err)
	}
}

// NewBucket creates an empty node-local bucket. subjectKind identifies the
// class of rate-limited subject, and subjectID identifies the subject within
// that class. Together, they identify the budget represented by the bucket.
//
// leaseSize is the maximum number of units reserved locally at one time. Larger
// values require less frequent acquisitions, while smaller values leave less
// unused capacity reserved by one process. maxUnits is the maximum number of
// units that a single Consume or Restore call accepts. Both values must be at
// least 1, and maxUnits must not exceed leaseSize.
func (limiter *Limiter) NewBucket(subjectKind SubjectKind, subjectID string, leaseSize, maxUnits int) *Bucket {
	if leaseSize < 1 || maxUnits < 1 || maxUnits > leaseSize {
		slog.Error("core/internal/state/ratelimiter: invalid bucket configuration", "lease_size", leaseSize, "max_units", maxUnits)
		os.Exit(1)
	}
	bucket := &Bucket{
		limiter:     limiter,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		leaseSize:   leaseSize,
		maxUnits:    maxUnits,
	}
	limiter.buckets.Lock()
	limiter.buckets.refs = append(limiter.buckets.refs, weak.Make(bucket))
	limiter.buckets.Unlock()
	return bucket
}

// acquireLeases acquires rate-limit capacity from PostgreSQL.
func (limiter *Limiter) acquireLeases(ctx context.Context, requests []leaseRequest) ([]leaseResult, error) {
	encoded, err := json.Marshal(requests)
	if err != nil {
		return nil, fmt.Errorf("cannot encode rate-limit lease requests: %w", err)
	}
	rows, err := limiter.db.Query(ctx, `
		SELECT subject_kind, subject_id, granted_units, capacity_units,
			available_units, rate_per_minute, refill_remainder
		FROM acquire_rate_limit_leases($1::jsonb)`, string(encoded))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]leaseResult, 0, len(requests))
	for rows.Next() {
		var result leaseResult
		if err := rows.Scan(
			&result.SubjectKind,
			&result.SubjectID,
			&result.GrantedUnits,
			&result.CapacityUnits,
			&result.AvailableUnits,
			&result.RatePerMinute,
			&result.RefillRemainder,
		); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// backoffError returns the stored error while a refill backoff is active.
func (limiter *Limiter) backoffError() error {
	backoff := limiter.backoff.Load()
	if backoff == nil || !time.Now().Before(backoff.until) {
		return nil
	}
	return backoff.err
}

// collectAndRefill collects a batch starting with first, then processes it.
// It returns false if shutdown starts while collecting.
func (limiter *Limiter) collectAndRefill(first *refill) bool {
	pending := []*refill{first}
	timer := time.NewTimer(batchDelay)
	defer timer.Stop()
	for len(pending) < batchSize {
		select {
		case <-limiter.close.ctx.Done():
			rejectRefills(pending, ErrLimiterUnavailable)
			return false
		case refill := <-limiter.queue:
			pending = append(pending, refill)
		case <-timer.C:
			limiter.refill(pending)
			return true
		}
	}
	limiter.refill(pending)
	return true
}

// collectUnusedCapacity gathers unused node-local capacity and capacity queued
// for restoration during shutdown. Close calls it after all background workers
// have stopped.
func (limiter *Limiter) collectUnusedCapacity() []unusedCapacity {
	limiter.buckets.Lock()
	refs := slices.Clone(limiter.buckets.refs)
	limiter.buckets.Unlock()

	available := make(map[subjectKey]int)
	for _, ref := range refs {
		bucket := ref.Value()
		if bucket == nil {
			continue
		}
		bucket.mu.Lock()
		units := bucket.available
		bucket.mu.Unlock()
		if units == 0 {
			continue
		}
		key := subjectKey{kind: bucket.subjectKind, id: bucket.subjectID}
		available[key] = min(maxRestoredUnits, available[key]+units)
	}
	limiter.restorations.mergeInto(available)

	unused := make([]unusedCapacity, 0, len(available))
	for key, units := range available {
		unused = append(unused, unusedCapacity{
			SubjectKind: key.kind,
			SubjectID:   key.id,
			Units:       units,
		})
	}
	return unused
}

// compactBuckets removes references to buckets that are no longer reachable.
// It scans a snapshot without holding the mutex, avoiding blocking NewBucket
// for the entire scan. If shutdown begins during the scan, the existing slice
// is left unchanged.
func (limiter *Limiter) compactBuckets() {
	limiter.buckets.Lock()
	refs := limiter.buckets.refs
	limiter.buckets.Unlock()

	compacted := make([]weak.Pointer[Bucket], 0, len(refs))
	for _, ref := range refs {
		if limiter.close.ctx.Err() != nil {
			return
		}
		if ref.Value() != nil {
			compacted = append(compacted, ref)
		}
	}
	if len(compacted) == len(refs) {
		return
	}

	limiter.buckets.Lock()
	defer limiter.buckets.Unlock()
	if limiter.close.ctx.Err() != nil {
		return
	}
	compacted = append(compacted, limiter.buckets.refs[len(refs):]...)
	limiter.buckets.refs = compacted
}

// discardQueuedRefills rejects refills still queued during shutdown.
func (limiter *Limiter) discardQueuedRefills() {
	for {
		select {
		case refill := <-limiter.queue:
			refill.bucket.rejectRefill(refill, ErrLimiterUnavailable)
		default:
			return
		}
	}
}

// failBatch starts backoff unless shutdown is in progress, then rejects the
// pending refills.
func (limiter *Limiter) failBatch(pending []*refill, err error) {
	if limiter.close.ctx.Err() == nil {
		limiter.backoff.Store(&backoffState{until: time.Now().Add(refillBackoffDuration), err: err})
	}
	rejectRefills(pending, err)
}

// invalidBatch logs and rejects a batch with an invalid lease result.
func (limiter *Limiter) invalidBatch(pending []*refill, result leaseResult) {
	err := fmt.Errorf("rate limiter lease batch returned an invalid result for subject %q of kind %q", result.SubjectID, result.SubjectKind)
	slog.Error("cannot apply rate-limit lease batch", "error", err)
	limiter.failBatch(pending, err)
}

// publishRefill queues a refill or rejects it if the queue is full.
func (limiter *Limiter) publishRefill(refill *refill) bool {
	select {
	case limiter.queue <- refill:
		limiter.queueFullLogged.Store(false)
		return true
	default:
		refill.bucket.rejectRefill(refill, ErrLimiterUnavailable)
		if limiter.metrics.QueueFull != nil {
			limiter.metrics.QueueFull.Inc()
		}
		if limiter.queueFullLogged.CompareAndSwap(false, true) {
			slog.Warn("rate limiter refill queue is full")
		}
		return false
	}
}

// queueCapacityRestoration schedules capacity that does not fit in a local
// bucket for best-effort restoration to PostgreSQL.
func (limiter *Limiter) queueCapacityRestoration(kind SubjectKind, id string, units int) {
	if units > 0 {
		limiter.restorations.add(subjectKey{kind: kind, id: id}, units)
	}
}

// refill processes a batch by acquiring capacity and resolving each refill.
func (limiter *Limiter) refill(pending []*refill) {
	if err := limiter.backoffError(); err != nil {
		rejectRefills(pending, err)
		return
	}

	requests := make([]leaseRequest, 0, len(pending))
	requestsBySubject := make(map[subjectKey]leaseRequest, len(pending))
	for _, refill := range pending {
		request := refill.request
		key := subjectKey{kind: request.SubjectKind, id: request.SubjectID}
		if _, duplicate := requestsBySubject[key]; duplicate {
			err := fmt.Errorf("rate limiter refill batch contains duplicate subject %q of kind %q", request.SubjectID, request.SubjectKind)
			slog.Error("cannot prepare rate-limit lease batch", "error", err)
			limiter.failBatch(pending, err)
			return
		}
		requestsBySubject[key] = request
		requests = append(requests, request)
	}
	acquireCtx, cancel := context.WithTimeout(limiter.close.ctx, defaultAcquireTimeout)
	results, err := limiter.acquire(acquireCtx, requests)
	cancel()
	if err != nil {
		if limiter.close.ctx.Err() == nil {
			slog.Error("cannot acquire rate-limit leases", "error", err)
			if limiter.metrics.AcquisitionErrors != nil {
				limiter.metrics.AcquisitionErrors.Inc()
			}
		}
		limiter.failBatch(pending, ErrLimiterUnavailable)
		return
	}

	resultsBySubject := make(map[subjectKey]leaseResult, len(results))
	for _, result := range results {
		key := subjectKey{kind: result.SubjectKind, id: result.SubjectID}
		request, ok := requestsBySubject[key]
		_, duplicate := resultsBySubject[key]
		// The all-zero encoding is unambiguous because schema constraints require
		// existing subjects to have positive refill rates and maximum capacities.
		missingSubject :=
			result.GrantedUnits == 0 &&
				result.CapacityUnits == 0 &&
				result.AvailableUnits == 0 &&
				result.RatePerMinute == 0 &&
				result.RefillRemainder == 0
		invalidValues :=
			result.GrantedUnits < 0 ||
				result.GrantedUnits > request.RequestedUnits ||
				result.CapacityUnits <= 0 ||
				result.GrantedUnits > result.CapacityUnits ||
				result.AvailableUnits < 0 ||
				result.AvailableUnits > result.CapacityUnits-result.GrantedUnits ||
				result.RatePerMinute <= 0 ||
				result.RefillRemainder < 0 ||
				result.RefillRemainder >= microsecondsPerMinute
		invalidBatch := !ok || duplicate || (!missingSubject && invalidValues)
		if invalidBatch {
			limiter.invalidBatch(pending, result)
			return
		}
		resultsBySubject[key] = result
	}
	if len(resultsBySubject) != len(pending) {
		err := errors.New("rate limiter lease batch returned incomplete results")
		slog.Error("cannot apply rate-limit lease batch", "error", err)
		limiter.failBatch(pending, err)
		return
	}
	for _, refill := range pending {
		request := refill.request
		result := resultsBySubject[subjectKey{kind: request.SubjectKind, id: request.SubjectID}]
		if result.CapacityUnits == 0 {
			refill.bucket.rejectRefill(refill, ErrLimiterUnavailable)
			continue
		}
		refill.bucket.completeRefill(refill, result)
	}
	limiter.backoff.Store(nil)
}

// restoreCapacityBatch restores unused capacity for a batch of subjects.
func (limiter *Limiter) restoreCapacityBatch(ctx context.Context, unused []unusedCapacity) error {
	encoded, err := json.Marshal(unused)
	if err != nil {
		return fmt.Errorf("cannot encode unused rate-limit capacity: %w", err)
	}
	_, err = limiter.db.Exec(ctx, "SELECT restore_rate_limit_capacity($1::jsonb)", string(encoded))
	return err
}

// restoreUnusedCapacity restores node-local capacity that remains available at
// shutdown.
func (limiter *Limiter) restoreUnusedCapacity(ctx context.Context, unused []unusedCapacity) error {
	if len(unused) == 0 {
		return nil
	}
	// Shuffle buckets so an interrupted shutdown does not consistently favor the
	// same buckets.
	rand.Shuffle(len(unused), func(i, j int) {
		unused[i], unused[j] = unused[j], unused[i]
	})
	batches := (len(unused) + maxRestoreBatchSize - 1) / maxRestoreBatchSize
	workers := min(maxRestoreWorkers, batches)
	// Batches are independent, so a few workers can restore capacity promptly
	// during shutdown without placing excessive load on PostgreSQL.
	var next atomic.Int64
	var wait sync.WaitGroup
	var errOnce sync.Once
	var firstErr error
	for range workers {
		wait.Go(func() {
			for {
				first := int(next.Add(1)-1) * maxRestoreBatchSize
				if first >= len(unused) || ctx.Err() != nil {
					return
				}
				last := min(first+maxRestoreBatchSize, len(unused))
				err := limiter.restoreBatch(ctx, unused[first:last])
				if err != nil {
					errOnce.Do(func() {
						firstErr = fmt.Errorf("cannot restore unused rate-limit capacity: %w", err)
					})
				}
			}
		})
	}
	wait.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	return firstErr
}

// runCompactor periodically removes references to buckets collected by Go's
// garbage collector.
func (limiter *Limiter) runCompactor() {
	compaction := time.NewTicker(bucketCompactionInterval) // Go 1.23+ GC reclaims unreachable tickers; no Stop needed.
	for {
		select {
		case <-limiter.close.ctx.Done():
			return
		case <-compaction.C:
			limiter.compactBuckets()
		}
	}
}

// runRefiller processes queued refills until shutdown.
func (limiter *Limiter) runRefiller() {
	defer limiter.discardQueuedRefills()
	for {
		select {
		case <-limiter.close.ctx.Done():
			return
		case refill := <-limiter.queue:
			if !limiter.collectAndRefill(refill) {
				return
			}
		}
	}
}

// runRestorer restores capacity that could not be retained locally to
// PostgreSQL. Failed batches are not retried because the database operation may
// have committed even when the caller receives an error.
func (limiter *Limiter) runRestorer() {
	for {
		select {
		case <-limiter.close.ctx.Done():
			return
		case <-limiter.restorations.wake:
		}
		if limiter.close.ctx.Err() != nil {
			return
		}
		timer := time.NewTimer(batchDelay)
		select {
		case <-limiter.close.ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		for limiter.close.ctx.Err() == nil {
			batch := limiter.restorations.takeBatch()
			if len(batch) == 0 {
				break
			}
			restoreCtx, cancel := context.WithTimeout(limiter.close.ctx, defaultRestorationTimeout)
			err := limiter.restoreBatch(restoreCtx, batch)
			cancel()
			if err != nil && limiter.close.ctx.Err() == nil {
				slog.Warn("cannot restore excess rate-limit capacity", "error", err)
			}
		}
	}
}

// rejectRefills rejects every refill in pending with err.
func rejectRefills(pending []*refill, err error) {
	for _, refill := range pending {
		refill.bucket.rejectRefill(refill, err)
	}
}

// acquireFunc obtains capacity for a batch of lease requests.
type acquireFunc func(context.Context, []leaseRequest) ([]leaseResult, error)

// backoffState stores a temporary failure and its expiration time.
type backoffState struct {
	until time.Time
	err   error
}

// leaseRequest identifies one subject and the capacity requested for it.
type leaseRequest struct {
	SubjectKind    SubjectKind `json:"subject_kind"`
	SubjectID      string      `json:"subject_id"`
	RequestedUnits int         `json:"requested_units"`
}

// leaseResult contains PostgreSQL's capacity result for one subject.
type leaseResult struct {
	SubjectKind     SubjectKind
	SubjectID       string
	GrantedUnits    int
	CapacityUnits   int
	AvailableUnits  int
	RatePerMinute   int
	RefillRemainder int
}

// subjectKey identifies a subject in a batch.
type subjectKey struct {
	kind SubjectKind
	id   string
}
