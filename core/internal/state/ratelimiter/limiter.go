// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
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

type backoffState struct {
	until time.Time
	err   error
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

type acquireFunc func(context.Context, []leaseRequest) ([]leaseResult, error)

// Limiter manages rate-limit capacity for the buckets it creates.
type Limiter struct {
	db      *db.DB
	acquire acquireFunc

	queue           chan *refill
	queueFullLogged atomic.Bool
	backoff         atomic.Pointer[backoffState]

	metrics Metrics

	shutdown struct {
		ctx    context.Context
		cancel context.CancelFunc
		done   chan struct{}
	}
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
	go limiter.run()
	return limiter
}

func (limiter *Limiter) run() {
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

// Close starts shutting down the limiter and waits until shutdown completes or
// ctx is done. If ctx is done first, shutdown continues in the background.
//
// When Close is called, no other calls to Limiter's methods or to methods of
// buckets created by it should be in progress and no other shall be made.
func (limiter *Limiter) Close(ctx context.Context) {
	limiter.shutdown.cancel()
	select {
	case <-limiter.shutdown.done:
	case <-ctx.Done():
	}
}

// NewBucket creates an empty local bucket. subjectKind identifies the class of
// rate-limited subject, and subjectID identifies the subject within that class.
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
	return &Bucket{
		limiter:     limiter,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		leaseSize:   leaseSize,
		maxUnits:    maxUnits,
	}
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

func (limiter *Limiter) backoffError() error {
	backoff := limiter.backoff.Load()
	if backoff == nil || !time.Now().Before(backoff.until) {
		return nil
	}
	return backoff.err
}

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

func (limiter *Limiter) failBatch(pending []*refill, err error) {
	if limiter.shutdown.ctx.Err() == nil {
		limiter.backoff.Store(&backoffState{until: time.Now().Add(refillBackoffDuration), err: err})
	}
	rejectRefills(pending, err)
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

func (limiter *Limiter) invalidBatch(pending []*refill, result leaseResult) {
	err := fmt.Errorf("rate limiter lease batch returned an invalid result for subject %q of kind %q", result.SubjectID, result.SubjectKind)
	slog.Error("cannot apply rate-limit lease batch", "error", err)
	limiter.failBatch(pending, err)
}

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

func rejectRefills(pending []*refill, err error) {
	for _, refill := range pending {
		refill.bucket.rejectRefill(refill, err)
	}
}

type leaseRequest struct {
	SubjectKind    SubjectKind `json:"subject_kind"`
	SubjectID      string      `json:"subject_id"`
	RequestedUnits int         `json:"requested_units"`
}

type leaseResult struct {
	SubjectKind   SubjectKind
	SubjectID     string
	GrantedUnits  int
	CapacityUnits int
}

type subjectKey struct {
	kind SubjectKind
	id   string
}
