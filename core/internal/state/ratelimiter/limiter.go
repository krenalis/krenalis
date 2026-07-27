// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
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

var defaultAcquireTimeout = 5 * time.Second

// ErrCapacityExceeded is returned when an operation cannot be served, including
// when it cannot be admitted or its refill fails, grants insufficient capacity,
// or does not complete before the internal timeout.
var ErrCapacityExceeded = errors.New("rate-limit capacity exceeded")

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

// Limiter coordinates refills for local buckets.
type Limiter struct {
	db      *db.DB
	acquire acquireFunc

	queue           chan *refill
	queueFullLogged atomic.Bool
	retryAfter      atomic.Int64

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

// Close closes the limiter, waiting for the batcher until ctx is done.
// The caller must stop using the limiter and all of its buckets before Close.
func (limiter *Limiter) Close(ctx context.Context) {
	limiter.shutdown.cancel()
	select {
	case <-limiter.shutdown.done:
	case <-ctx.Done():
	}
}

// NewBucket creates an empty local bucket for subjectKind and subjectID.
// leaseSize and maxCost must be positive, and maxCost must not exceed
// leaseSize. NewBucket panics if these conditions are not met.
func (limiter *Limiter) NewBucket(subjectKind SubjectKind, subjectID string, leaseSize, maxCost int) *Bucket {
	if leaseSize < 1 || maxCost < 1 || maxCost > leaseSize {
		panic("invalid rate-limit bucket configuration")
	}
	return &Bucket{
		limiter:     limiter,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		leaseSize:   leaseSize,
		maxCost:     maxCost,
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

func (limiter *Limiter) backoffActive() bool {
	return time.Now().UnixNano() < limiter.retryAfter.Load()
}

func (limiter *Limiter) collectAndRefill(first *refill) bool {
	pending := []*refill{first}
	timer := time.NewTimer(batchDelay)
	defer timer.Stop()
	for len(pending) < batchSize {
		select {
		case <-limiter.shutdown.ctx.Done():
			rejectRefills(pending)
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

func (limiter *Limiter) failBatch(pending []*refill) {
	if limiter.shutdown.ctx.Err() == nil {
		limiter.retryAfter.Store(time.Now().Add(refillBackoffDuration).UnixNano())
	}
	rejectRefills(pending)
}

// publishRefill queues a refill or rejects it if the queue is full.
func (limiter *Limiter) publishRefill(refill *refill) bool {
	select {
	case limiter.queue <- refill:
		limiter.queueFullLogged.Store(false)
		return true
	default:
		refill.bucket.rejectRefill(refill)
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
	if limiter.backoffActive() {
		rejectRefills(pending)
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
		limiter.failBatch(pending)
		return
	}

	requestsBySubject := make(map[subjectKey]leaseRequest, len(pending))
	for _, refill := range pending {
		request := refill.request
		key := subjectKey{kind: request.SubjectKind, id: request.SubjectID}
		if _, duplicate := requestsBySubject[key]; duplicate {
			slog.Error("rate limiter batch contains duplicate subjects", "subject_kind", request.SubjectKind, "subject_id", request.SubjectID)
			limiter.failBatch(pending)
			return
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
		slog.Error("rate limiter lease batch returned incomplete results")
		limiter.failBatch(pending)
		return
	}
	for _, refill := range pending {
		request := refill.request
		result := resultsBySubject[subjectKey{kind: request.SubjectKind, id: request.SubjectID}]
		refill.bucket.completeRefill(refill, result.GrantedUnits, result.CapacityUnits)
	}
	limiter.retryAfter.Store(0)
}

func (limiter *Limiter) invalidBatch(pending []*refill, result leaseResult) {
	slog.Error("rate limiter lease batch returned an invalid result", "subject_kind", result.SubjectKind, "subject_id", result.SubjectID)
	limiter.failBatch(pending)
}

func (limiter *Limiter) discardQueuedRefills() {
	for {
		select {
		case refill := <-limiter.queue:
			refill.bucket.rejectRefill(refill)
		default:
			return
		}
	}
}

func rejectRefills(pending []*refill) {
	for _, refill := range pending {
		refill.bucket.rejectRefill(refill)
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
