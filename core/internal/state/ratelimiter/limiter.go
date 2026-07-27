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
	"sync"
	"sync/atomic"
	"time"
	"weak"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	batchSize  = 64
	batchDelay = 2 * time.Millisecond
	queueSize  = batchSize * 4

	refillBackoffDuration    = 250 * time.Millisecond
	bucketCompactionInterval = 5 * time.Minute
	maxRestoreBatchSize      = 64
	maxRestoreWorkers        = 4
	maxRestoredUnits         = 100_000
)

var defaultAcquireTimeout = 5 * time.Second

var (
	// ErrCapacityExceeded is returned when a successful acquisition confirms
	// that the requested capacity is unavailable.
	ErrCapacityExceeded = errors.New("rate-limit capacity exceeded")

	// ErrLimiterUnavailable is returned when the limiter cannot determine
	// whether the requested capacity is available because of a temporary
	// condition.
	ErrLimiterUnavailable = errors.New("rate limiter unavailable")
)

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
	db           *db.DB
	acquire      acquireFunc
	restoreBatch restoreBatchFunc

	queue           chan *refill
	queueFullLogged atomic.Bool
	backoff         atomic.Pointer[backoffState]

	metrics Metrics

	buckets struct {
		sync.Mutex
		refs []weak.Pointer[Bucket]
	}

	shutdown struct {
		ctx    context.Context
		cancel context.CancelFunc
		done   chan struct{}

		closed atomic.Bool
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
	limiter.acquire = limiter.acquireLeases
	limiter.restoreBatch = limiter.restoreCapacityBatch
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	limiter.shutdown.done = make(chan struct{})
	go limiter.runRefiller()
	go limiter.runCompactor()
	return limiter
}

// Close starts shutting down the limiter and waits until the batcher stops or
// ctx is done. After the batcher stops, it makes one best-effort attempt to
// restore unused node-local capacity to PostgreSQL. If ctx is done first, the
// batcher continues in the background and no capacity is restored.
//
// When Close is called, no other calls to Limiter's methods or to methods of
// buckets created by it should be in progress and no other shall be made. The
// caller must keep the buckets for existing subjects reachable until Close
// returns.
func (limiter *Limiter) Close(ctx context.Context) {
	if !limiter.shutdown.closed.CompareAndSwap(false, true) {
		return
	}
	limiter.shutdown.cancel()
	select {
	case <-limiter.shutdown.done:
	case <-ctx.Done():
	}
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
// that class.
// Together they identify the budget represented by the bucket.
//
// leaseSize is the maximum number of units reserved locally at one time. Larger
// values require less frequent acquisitions, while smaller values leave less
// unused capacity reserved to one process. maxUnits is the maximum number of
// units accepted by a single Consume or Restore call. Both values must be at
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

// backoffError returns the stored error while a refill backoff is active.
func (limiter *Limiter) backoffError() error {
	backoff := limiter.backoff.Load()
	if backoff == nil || !time.Now().Before(backoff.until) {
		return nil
	}
	return backoff.err
}

// collectAndRefill collects a batch starting with first, then processes it. It
// returns false if shutdown starts while collecting.
func (limiter *Limiter) collectAndRefill(first *refill) bool {
	pending := []*refill{first}
	timer := time.NewTimer(batchDelay)
	defer timer.Stop()
	for len(pending) < batchSize {
		select {
		case <-limiter.shutdown.ctx.Done():
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

// collectUnusedCapacity gathers unused node-local capacity from buckets that
// are still reachable during shutdown.
func (limiter *Limiter) collectUnusedCapacity() []unusedCapacity {
	// The compactor can still replace the reference slice after the batcher
	// stops. Copy the slice header while holding the same mutex it uses.
	limiter.buckets.Lock()
	refs := limiter.buckets.refs
	limiter.buckets.Unlock()

	available := make(map[subjectKey]int)
	for _, ref := range refs {
		bucket := ref.Value()
		if bucket == nil || bucket.available == 0 {
			continue
		}
		key := subjectKey{kind: bucket.subjectKind, id: bucket.subjectID}
		available[key] = min(maxRestoredUnits, available[key]+bucket.available)
	}
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
		if limiter.shutdown.ctx.Err() != nil {
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
	if limiter.shutdown.ctx.Err() != nil {
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
	if limiter.shutdown.ctx.Err() == nil {
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

// refill processes a batch by acquiring capacity and resolving each refill.
func (limiter *Limiter) refill(pending []*refill) {
	if err := limiter.backoffError(); err != nil {
		rejectRefills(pending, err)
		return
	}

	requests := make([]leaseRequest, 0, len(pending))
	for _, refill := range pending {
		requests = append(requests, refill.request)
	}
	acquireCtx, cancel := context.WithTimeout(limiter.shutdown.ctx, defaultAcquireTimeout)
	results, err := limiter.acquire(acquireCtx, requests)
	if err == nil {
		err = acquireCtx.Err()
	}
	cancel()
	if err != nil {
		if limiter.shutdown.ctx.Err() == nil {
			slog.Error("cannot acquire rate-limit leases", "error", err)
			if limiter.metrics.AcquisitionErrors != nil {
				limiter.metrics.AcquisitionErrors.Inc()
			}
		}
		limiter.failBatch(pending, ErrLimiterUnavailable)
		return
	}

	requestsBySubject := make(map[subjectKey]leaseRequest, len(pending))
	for _, refill := range pending {
		request := refill.request
		key := subjectKey{kind: request.SubjectKind, id: request.SubjectID}
		if _, duplicate := requestsBySubject[key]; duplicate {
			slog.Error("core/internal/state/ratelimiter: batch contains duplicate subject", "subject_kind", request.SubjectKind, "subject_id", request.SubjectID)
			os.Exit(1)
		}
		requestsBySubject[key] = request
	}
	resultsBySubject := make(map[subjectKey]leaseResult, len(results))
	for _, result := range results {
		key := subjectKey{kind: result.SubjectKind, id: result.SubjectID}
		request, ok := requestsBySubject[key]
		if _, duplicate := resultsBySubject[key]; !ok || duplicate ||
			result.GrantedUnits < 0 || result.GrantedUnits > request.RequestedUnits ||
			result.CapacityUnits <= 0 || result.GrantedUnits > result.CapacityUnits {
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
		refill.bucket.completeRefill(refill, result.GrantedUnits, result.CapacityUnits)
	}
	limiter.backoff.Store(nil)
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

// restoreCapacityBatch restores unused capacity for one batch of subjects.
func (limiter *Limiter) restoreCapacityBatch(ctx context.Context, unused []unusedCapacity) error {
	encoded, err := json.Marshal(unused)
	if err != nil {
		return fmt.Errorf("cannot encode unused rate-limit capacity: %w", err)
	}
	_, err = limiter.db.Exec(ctx, "SELECT restore_rate_limit_capacity($1::jsonb)", string(encoded))
	return err
}

// runRefiller processes queued refills until shutdown.
func (limiter *Limiter) runRefiller() {
	defer close(limiter.shutdown.done)
	defer limiter.discardQueuedRefills()
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
}

// runCompactor periodically cleans up references to buckets that have been
// garbage-collected by Go. It runs independently of the batcher, so Close does
// not wait for a scan to finish, though it may briefly block while the
// compactor updates the reference slice.
func (limiter *Limiter) runCompactor() {
	compaction := time.NewTicker(bucketCompactionInterval)
	for {
		select {
		case <-limiter.shutdown.ctx.Done():
			return
		case <-compaction.C:
			limiter.compactBuckets()
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
	SubjectKind   SubjectKind
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
}

// restoreBatchFunc restores unused capacity for one batch of subjects.
type restoreBatchFunc func(context.Context, []unusedCapacity) error

// subjectKey identifies a subject in a batch.
type subjectKey struct {
	kind SubjectKind
	id   string
}

// unusedCapacity identifies unused node-local capacity for one subject.
type unusedCapacity struct {
	SubjectKind SubjectKind `json:"subject_kind"`
	SubjectID   string      `json:"subject_id"`
	Units       int         `json:"units"`
}
