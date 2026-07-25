// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testRateLimitID is a valid subject identifier shared by bucket tests.
const testRateLimitID = "111111111111"

// newTestRateLimiter starts a limiter and registers its shutdown with the test.
func newTestRateLimiter(t *testing.T, acquire leaseAcquirer) *Limiter {
	t.Helper()
	l := &Limiter{
		acquireLeases:  acquire,
		now:            time.Now,
		maxWait:        maxWaitDuration,
		acquireTimeout: defaultAcquireTimeout,
		refillQueue:    make(chan *refill, queueSize),
	}
	l.close.ctx, l.close.cancel = context.WithCancel(context.Background())
	l.close.Add(1)
	go l.runBatcher()
	t.Cleanup(l.Close)
	return l
}

// bucketSnapshot reads bucket state while holding its mutex.
func bucketSnapshot(bucket *Bucket) (available, target, threshold int, refillQueued, disabled bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	return bucket.available, bucket.target, bucket.threshold, bucket.refill != nil, bucket.disabled
}

// applyTestLease applies a test lease unless the bucket is disabled.
func applyTestLease(bucket *Bucket, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if !bucket.disabled {
		bucket.applyLeaseLocked(grantedUnits, capacityUnits)
	}
}

// limiterRetryAfter returns the current global backoff deadline.
func limiterRetryAfter(l *Limiter) time.Time {
	unixNano := l.refillRetryAfter.Load()
	if unixNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, unixNano)
}

// testRateLimitClock is an atomically readable clock controlled by a test.
type testRateLimitClock struct{ unixNano atomic.Int64 }

// newTestRateLimitClock returns a clock with a stable initial value.
func newTestRateLimitClock() *testRateLimitClock {
	clock := &testRateLimitClock{}
	clock.Set(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	return clock
}

// Now returns the clock's current value.
func (clock *testRateLimitClock) Now() time.Time {
	return time.Unix(0, clock.unixNano.Load())
}

// Set updates the clock's current value.
func (clock *testRateLimitClock) Set(now time.Time) {
	clock.unixNano.Store(now.UnixNano())
}

// waitForRateLimit waits until condition succeeds or fails the test after one second.
func waitForRateLimit(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return
		}
		select {
		case <-deadline.C:
			t.Fatal("timed out waiting for rate limiter")
		case <-ticker.C:
		}
	}
}

// refillPendingCost returns the total cost of waiters attached to the refill.
func refillPendingCost(bucket *Bucket) int {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.refill == nil {
		return 0
	}
	return bucket.refill.pendingCost
}

// TestBucketsConsumeIndependently verifies that organization, workspace, and
// ingestion buckets do not share local capacity.
func TestBucketsConsumeIndependently(t *testing.T) {
	limiter := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 2}}, nil
	})
	organizationBucket := NewOrganizationBucket("111111111111")
	workspaceBucket := NewWorkspaceBucket("222222222222")
	ingestionBucket := NewIngestionBucket("222222222222")
	applyTestLease(organizationBucket, 2, 2)
	applyTestLease(workspaceBucket, 2, 2)
	applyTestLease(ingestionBucket, 2, 2)

	if err := limiter.Consume(context.Background(), organizationBucket, 2); err != nil {
		t.Fatalf("organization consume: %v", err)
	}
	if err := limiter.Consume(context.Background(), organizationBucket, 2); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("organization second consume: got %v, want ErrCapacityExceeded", err)
	}
	if err := limiter.Consume(context.Background(), workspaceBucket, 2); err != nil {
		t.Fatalf("workspace consume: %v", err)
	}
	if err := limiter.Consume(context.Background(), ingestionBucket, 2); err != nil {
		t.Fatalf("ingestion consume: %v", err)
	}
	ingestionBucket.Restore(2)
	if err := limiter.Consume(context.Background(), ingestionBucket, 2); err != nil {
		t.Fatalf("ingestion consume after restoration: %v", err)
	}
}

// TestLimiterValidatesCost verifies that normal API costs stay within
// the supported range.
func TestLimiterValidatesCost(t *testing.T) {
	l := newTestRateLimiter(t, nil)
	bucket := NewOrganizationBucket(testRateLimitID)
	for _, cost := range []int{-1, 0, 101} {
		if err := l.Consume(context.Background(), bucket, cost); !errors.Is(err, ErrInvalidCost) {
			t.Fatalf("cost %d: got %v, want ErrInvalidCost", cost, err)
		}
	}
}

// TestIngestionValidatesEventCount verifies the supported batch size.
func TestIngestionValidatesEventCount(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := NewIngestionBucket(testRateLimitID)
	for _, count := range []int{-1, 0, ingestionMaxCost + 1} {
		if err := limiter.Consume(context.Background(), bucket, count); !errors.Is(err, ErrInvalidCost) {
			t.Fatalf("event count %d: got %v, want ErrInvalidCost", count, err)
		}
	}
	applyTestLease(bucket, ingestionMaxCost, ingestionMaxCost)
	if err := limiter.Consume(context.Background(), bucket, ingestionMaxCost); err != nil {
		t.Fatalf("maximum event count: %v", err)
	}
}

// TestLimiterConsumesLocalCapacity verifies immediate local consumption
// and rejection when the remaining capacity is insufficient.
func TestLimiterConsumesLocalCapacity(t *testing.T) {
	l := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 10}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 10, 10)

	if err := l.Consume(context.Background(), bucket, 6); err != nil {
		t.Fatalf("consume: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 4 {
		t.Fatalf("available capacity = %d, want 4", available)
	}
	if err := l.Consume(context.Background(), bucket, 5); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume exhausted capacity: got %v, want ErrCapacityExceeded", err)
	}
}

// TestLimiterCanceledContextDoesNotConsumeLocalCapacity verifies that
// an already canceled request leaves local capacity unchanged.
func TestLimiterCanceledContextDoesNotConsumeLocalCapacity(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 10, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Consume(ctx, bucket, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("consume with canceled context: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 10 {
		t.Fatalf("available capacity = %d, want 10", available)
	}
}

// TestLimiterCallerDeadlineTakesPrecedence verifies that a caller
// deadline wins over shutdown and internal timeout.
func TestLimiterCallerDeadlineTakesPrecedence(t *testing.T) {
	limiter := &Limiter{maxWait: 0}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.cancel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	for range 100 {
		bucket := NewOrganizationBucket(testRateLimitID)
		_, refill, _ := bucket.consume(1, true)
		waiter := bucket.activateRefill(refill, 1, true, true)
		if err := limiter.waitForRefill(ctx, waiter); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("wait returned %v, want context deadline exceeded", err)
		}
		bucket.rejectRefill(refill)
	}
}

// TestLimiterUsesRequestedUnitsAsAdmissionBudget verifies that waiter
// admission uses the frozen refill request rather than the lease size.
func TestLimiterUsesRequestedUnitsAsAdmissionBudget(t *testing.T) {
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 90, 100)

	satisfied, refill, waiter := bucket.consume(100, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("insufficient consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	if refill.request.RequestedUnits != 10 {
		t.Fatalf("requested units = %d, want 10", refill.request.RequestedUnits)
	}
	if waiter := bucket.activateRefill(refill, 100, true, true); waiter != nil {
		t.Fatal("operation larger than the refill request was admitted")
	}

	satisfied, _, _ = bucket.consume(90, true)
	if !satisfied {
		t.Fatal("positive local capacity was not left available for immediate consumption")
	}
	_, _, waiter = bucket.consume(10, true)
	if waiter == nil {
		t.Fatal("operation matching the refill request was not admitted")
	}
	_, _, waiter = bucket.consume(1, true)
	if waiter != nil {
		t.Fatal("waiter cost exceeded the requested-units admission budget")
	}
	bucket.rejectRefill(refill)
}

// TestLimiterServesOnlySatisfiableFIFOPrefix verifies FIFO head-of-line
// blocking after a partial grant.
func TestLimiterServesOnlySatisfiableFIFOPrefix(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 60, CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)

	resultA := make(chan error, 1)
	resultB := make(chan error, 1)
	resultC := make(chan error, 1)
	go func() { resultA <- limiter.Consume(context.Background(), bucket, 40) }()
	<-requests
	go func() { resultB <- limiter.Consume(context.Background(), bucket, 30) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 70 })
	go func() { resultC <- limiter.Consume(context.Background(), bucket, 20) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 90 })
	close(release)

	if err := <-resultA; err != nil {
		t.Fatalf("first waiter: %v", err)
	}
	if err := <-resultB; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("second waiter: %v", err)
	}
	if err := <-resultC; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("third waiter bypassed the FIFO head: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 20 {
		t.Fatalf("remaining capacity = %d, want 20", available)
	}
}

// TestLimiterCancellationReturnsAdmissionBudget verifies that a
// canceled waiter makes its admission budget available to a later request.
func TestLimiterCancellationReturnsAdmissionBudget(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 100, CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	second := make(chan error, 1)
	third := make(chan error, 1)
	go func() { first <- limiter.Consume(ctx, bucket, 60) }()
	<-requests
	go func() { second <- limiter.Consume(context.Background(), bucket, 40) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 100 })
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter: %v", err)
	}
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 40 })
	go func() { third <- limiter.Consume(context.Background(), bucket, 60) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 100 })
	close(release)

	if err := <-second; err != nil {
		t.Fatalf("second waiter: %v", err)
	}
	if err := <-third; err != nil {
		t.Fatalf("replacement waiter: %v", err)
	}
}

// TestLimiterWaitHasFiniteInternalTimeout verifies that waiters do not
// block indefinitely without a caller deadline.
func TestLimiterWaitHasFiniteInternalTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			CapacityUnits: 100,
		}}, nil
	})
	limiter.maxWait = 10 * time.Millisecond
	bucket := NewOrganizationBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- limiter.Consume(context.Background(), bucket, 1) }()
	<-started
	if err := <-result; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("internal timeout: %v", err)
	}
	if pending := refillPendingCost(bucket); pending != 0 {
		t.Fatalf("pending cost after timeout = %d, want 0", pending)
	}
	close(release)
}

// TestLimiterLeaseAcquisitionHasFiniteTimeout verifies that a late
// acquisition result is discarded after the batch deadline.
func TestLimiterLeaseAcquisitionHasFiniteTimeout(t *testing.T) {
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 1, CapacityUnits: 100,
		}}, nil
	})
	limiter.acquireTimeout = 10 * time.Millisecond
	bucket := NewOrganizationBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- limiter.Consume(context.Background(), bucket, 1) }()
	<-started

	select {
	case err := <-result:
		if !errors.Is(err, ErrCapacityExceeded) {
			t.Fatalf("acquisition timeout: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("lease acquisition did not time out")
	}
	available, _, _, refillPending, _ := bucketSnapshot(bucket)
	if refillPending || available != 0 {
		t.Fatalf("state after acquisition timeout: pending=%t available=%d", refillPending, available)
	}
}

// TestLimiterCancellationAndGrantResolveOnce verifies that cancellation
// and completion resolve a waiter once and leave capacity consistent with the
// winner.
func TestLimiterCancellationAndGrantResolveOnce(t *testing.T) {
	for range 100 {
		bucket := NewOrganizationBucket(testRateLimitID)
		applyTestLease(bucket, 0, 100)
		_, refill, _ := bucket.consume(2, true)
		waiter := bucket.activateRefill(refill, 1, true, true)
		if waiter == nil {
			t.Fatal("waiter was not admitted")
		}

		start := make(chan struct{})
		cancelResult := make(chan error, 1)
		completed := make(chan struct{})
		go func() {
			<-start
			cancelResult <- bucket.cancelWaiter(waiter, context.Canceled)
		}()
		go func() {
			<-start
			bucket.completeRefill(refill, 1, 100)
			close(completed)
		}()
		close(start)
		err := <-cancelResult
		<-completed
		available, _, _, queued, _ := bucketSnapshot(bucket)
		if queued {
			t.Fatal("completed generation remained attached")
		}
		switch {
		case errors.Is(err, context.Canceled):
			if available != 1 {
				t.Fatalf("cancellation won but available capacity = %d, want 1", available)
			}
		case err == nil:
			if available != 0 {
				t.Fatalf("grant won but available capacity = %d, want 0", available)
			}
		default:
			t.Fatalf("race result: %v", err)
		}
	}
}

// TestLimiterIgnoresStaleGenerationResult verifies that an old refill
// cannot change a newer generation or its waiter.
func TestLimiterIgnoresStaleGenerationResult(t *testing.T) {
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 0, 100)
	_, first, _ := bucket.consume(2, true)
	bucket.activateRefill(first, 0, false, true)
	bucket.rejectRefill(first)

	_, second, _ := bucket.consume(2, true)
	waiter := bucket.activateRefill(second, 1, true, true)
	if waiter == nil {
		t.Fatal("second-generation waiter was not admitted")
	}
	bucket.completeRefill(first, 100, 100)
	bucket.mu.Lock()
	current := bucket.refill
	available := bucket.available
	pending := waiter.element != nil
	bucket.mu.Unlock()
	if current != second || available != 0 || !pending {
		t.Fatalf("stale result changed current generation: current=%p available=%d pending=%t", current, available, pending)
	}

	bucket.completeRefill(second, 1, 100)
	<-waiter.refill.done
	if waiter.err != nil {
		t.Fatalf("second-generation waiter: %v", waiter.err)
	}
}

// TestLimiterQueuesOneRefill verifies that concurrent requests share
// one active refill generation.
func TestLimiterQueuesOneRefill(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 10, CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)
	results := make(chan error, 3)
	go func() { results <- l.Consume(context.Background(), bucket, 1) }()
	if request := <-requests; len(request) != 1 {
		t.Fatalf("batch contains %d requests, want 1", len(request))
	}
	for range 2 {
		go func() { results <- l.Consume(context.Background(), bucket, 1) }()
	}
	waitForRateLimit(t, func() bool {
		bucket.mu.Lock()
		defer bucket.mu.Unlock()
		return bucket.refill != nil && bucket.refill.pendingCost == 3
	})
	close(release)
	for range 3 {
		if err := <-results; err != nil {
			t.Fatalf("consume after refill: %v", err)
		}
	}
	waitForRateLimit(t, func() bool {
		available, _, _, queued, _ := bucketSnapshot(bucket)
		return !queued && available == 7
	})
}

// TestLimiterBatchesOrganizationsAndWorkspaces verifies that distinct
// subject kinds are acquired in the same batch.
func TestLimiterBatchesOrganizationsAndWorkspaces(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	limiter := &Limiter{
		now:         time.Now,
		maxWait:     maxWaitDuration,
		refillQueue: make(chan *refill, queueSize),
		acquireLeases: func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
			requests <- request
			results := make([]leaseResult, len(request))
			for i, r := range request {
				results[i] = leaseResult{SubjectKind: r.SubjectKind, SubjectID: r.SubjectID, CapacityUnits: 100}
			}
			return results, nil
		},
	}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.Add(1)
	t.Cleanup(limiter.Close)

	refills := make([]*refill, 0, 2)
	waiters := make([]*waiter, 0, 2)
	for _, bucket := range []*Bucket{NewOrganizationBucket("111111111111"), NewWorkspaceBucket("222222222222")} {
		_, refill, _ := bucket.consume(1, true)
		waiter := bucket.activateRefill(refill, 1, true, true)
		if waiter == nil {
			t.Fatal("waiter was not admitted")
		}
		refills = append(refills, refill)
		waiters = append(waiters, waiter)
		limiter.refillQueue <- refill
	}
	go limiter.runBatcher()

	if request := <-requests; len(request) != 2 {
		t.Fatalf("batch contains %d requests, want 2", len(request))
	}
	for i, refill := range refills {
		<-refill.done
		if !errors.Is(waiters[i].err, ErrCapacityExceeded) {
			t.Fatalf("waiter with zero grant: %v", waiters[i].err)
		}
	}
}

// TestLimiterAddsLeaseAfterConcurrentConsumption verifies that a grant
// is added to capacity consumed while its refill was in flight.
func TestLimiterAddsLeaseAfterConcurrentConsumption(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 20, CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 10, 100)
	if err := l.Consume(context.Background(), bucket, 1); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := l.Consume(context.Background(), bucket, 5); err != nil {
		t.Fatal(err)
	}
	close(release)
	waitForRateLimit(t, func() bool {
		_, _, _, queued, _ := bucketSnapshot(bucket)
		return !queued
	})
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 24 {
		t.Fatalf("available capacity = %d, want 24", available)
	}
}

// TestLimiterStartsGlobalBackoffAfterRefillError verifies that a failed
// acquisition suppresses refills and waiter admission until the retry deadline.
func TestLimiterStartsGlobalBackoffAfterRefillError(t *testing.T) {
	clock := newTestRateLimitClock()
	var calls atomic.Int32
	l := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("PostgreSQL is unavailable")
		}
		return []leaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			CapacityUnits: 100,
		}}, nil
	})
	l.now = clock.Now
	bucket := NewOrganizationBucket(testRateLimitID)
	applyTestLease(bucket, 2, 100)
	if err := l.Consume(context.Background(), bucket, 1); err != nil {
		t.Fatalf("initial consume: %v", err)
	}
	waitForRateLimit(t, func() bool {
		return !limiterRetryAfter(l).IsZero() && calls.Load() == 1
	})
	retryAt := limiterRetryAfter(l)
	if want := clock.Now().Add(refillBackoffDuration); !retryAt.Equal(want) {
		t.Fatalf("retry after = %s, want %s", retryAt, want)
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("failed refill remained queued")
	}

	if err := l.Consume(context.Background(), bucket, 1); err != nil {
		t.Fatalf("consume local capacity during backoff: %v", err)
	}
	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume without local capacity during backoff: got %v, want ErrCapacityExceeded", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("refill attempts during backoff = %d, want 1", calls.Load())
	}

	clock.Set(retryAt)
	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume after backoff: got %v, want ErrCapacityExceeded", err)
	}
	waitForRateLimit(t, func() bool {
		return calls.Load() == 2 && limiterRetryAfter(l).IsZero()
	})
}

// TestLimiterBacksOffAfterInvalidLeaseResults verifies that malformed
// batch responses fail without applying capacity.
func TestLimiterBacksOffAfterInvalidLeaseResults(t *testing.T) {
	for _, test := range []struct {
		name    string
		results func(leaseRequest) []leaseResult
	}{
		{
			name: "incomplete",
			results: func(leaseRequest) []leaseResult {
				return nil
			},
		},
		{
			name: "duplicate",
			results: func(request leaseRequest) []leaseResult {
				result := leaseResult{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, CapacityUnits: 100}
				return []leaseResult{result, result}
			},
		},
		{
			name: "unmatched subject",
			results: func(request leaseRequest) []leaseResult {
				return []leaseResult{{
					SubjectKind:   request.SubjectKind,
					SubjectID:     "222222222222",
					CapacityUnits: 100,
				}}
			},
		},
		{
			name: "invalid grant",
			results: func(request leaseRequest) []leaseResult {
				return []leaseResult{{
					SubjectKind:   request.SubjectKind,
					SubjectID:     request.SubjectID,
					GrantedUnits:  request.RequestedUnits + 1,
					CapacityUnits: 100,
				}}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clock := newTestRateLimitClock()
			l := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
				return test.results(requests[0]), nil
			})
			l.now = clock.Now
			bucket := NewOrganizationBucket(testRateLimitID)
			_ = l.Consume(context.Background(), bucket, 1)
			waitForRateLimit(t, func() bool {
				return !limiterRetryAfter(l).IsZero()
			})
		})
	}
}

// TestLimiterGlobalBackoffBlocksQueuedAndConcurrentRefills verifies
// that backoff rejects queued and concurrent refill attempts.
func TestLimiterGlobalBackoffBlocksQueuedAndConcurrentRefills(t *testing.T) {
	clock := newTestRateLimitClock()
	var calls atomic.Int32
	l := newTestRateLimiter(t, func(_ context.Context, _ []leaseRequest) ([]leaseResult, error) {
		calls.Add(1)
		return nil, nil
	})
	l.now = clock.Now
	l.refillRetryAfter.Store(clock.Now().Add(refillBackoffDuration).UnixNano())

	queued := NewOrganizationBucket(testRateLimitID)
	_, queuedRefill, _ := queued.consume(2, true)
	queued.activateRefill(queuedRefill, 0, false, true)
	l.refillQueue <- queuedRefill
	waitForRateLimit(t, func() bool {
		_, _, _, refillQueued, _ := bucketSnapshot(queued)
		return !refillQueued
	})
	if calls.Load() != 0 {
		t.Fatalf("refill attempts during global backoff = %d, want 0", calls.Load())
	}

	bucket := NewWorkspaceBucket("222222222222")
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
				t.Errorf("consume during global backoff: got %v, want ErrCapacityExceeded", err)
			}
		})
	}
	wg.Wait()
	_, _, _, refillQueued, _ := bucketSnapshot(bucket)
	if refillQueued {
		t.Fatal("global backoff queued a refill")
	}
}

// TestLimiterDisabledBucketCompletesInFlightRefill verifies that
// disabling a bucket rejects waiters and discards its late grant.
func TestLimiterDisabledBucketCompletesInFlightRefill(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []leaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  10,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- l.Consume(context.Background(), bucket, 1) }()
	<-started
	bucket.Disable()
	if err := <-result; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("disabled waiter: %v", err)
	}
	close(release)
	waitForRateLimit(t, func() bool {
		available, _, _, queued, disabled := bucketSnapshot(bucket)
		return disabled && !queued && available == 0
	})
}

// TestLimiterQueueFullRejectsRequestsWithoutLocalCapacity verifies that
// a rejected queue publication cannot authorize waiting.
func TestLimiterQueueFullRejectsRequestsWithoutLocalCapacity(t *testing.T) {
	l := &Limiter{
		now:         time.Now,
		refillQueue: make(chan *refill, 1),
	}
	l.refillQueue <- &refill{}
	bucket := NewOrganizationBucket(testRateLimitID)

	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume with a full queue: got %v, want ErrCapacityExceeded", err)
	}
	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("second consume with a full queue: got %v, want ErrCapacityExceeded", err)
	}
	if !limiterRetryAfter(l).IsZero() {
		t.Fatal("full queue started backoff")
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("full queue left refill queued")
	}
}

// TestLimiterShutdownDiscardsLateResultWithoutBackoff verifies that
// shutdown rejects a late result without starting global backoff.
func TestLimiterShutdownDiscardsLateResultWithoutBackoff(t *testing.T) {
	started := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 1, CapacityUnits: 100,
		}}, nil
	})
	bucket := NewOrganizationBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- l.Consume(context.Background(), bucket, 1) }()
	<-started
	l.Close()
	if err := <-result; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("waiter after shutdown: %v", err)
	}

	if !limiterRetryAfter(l).IsZero() {
		t.Fatal("shutdown started global backoff")
	}
	available, _, _, queued, _ := bucketSnapshot(bucket)
	if queued || available != 0 {
		t.Fatalf("state after shutdown: queued=%t available=%d", queued, available)
	}
}

// TestLimiterShutdownUnblocksBufferedWaiter verifies that shutdown
// resolves waiters even when their refill remains buffered in the queue.
func TestLimiterShutdownUnblocksBufferedWaiter(t *testing.T) {
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, _ []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	firstBucket := NewOrganizationBucket(testRateLimitID)
	secondBucket := NewWorkspaceBucket("222222222222")
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- limiter.Consume(context.Background(), firstBucket, 1) }()
	<-started
	go func() { second <- limiter.Consume(context.Background(), secondBucket, 1) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(secondBucket) == 1 })

	limiter.Close()
	if err := <-first; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("in-flight waiter after shutdown: %v", err)
	}
	if err := <-second; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("buffered waiter after shutdown: %v", err)
	}
}

// TestLimiterShutdownFinishesCollectedRefill verifies that shutdown
// rejects a refill already collected by the batcher.
func TestLimiterShutdownFinishesCollectedRefill(t *testing.T) {
	l := &Limiter{}
	l.close.ctx, l.close.cancel = context.WithCancel(context.Background())
	l.close.cancel()

	bucket := NewOrganizationBucket(testRateLimitID)
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	if l.collectAndRefillBatch(refill) {
		t.Fatal("batch collection continued after shutdown")
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("shutdown left collected refill queued")
	}
}
