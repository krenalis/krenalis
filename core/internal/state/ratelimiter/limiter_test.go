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
	"testing/synctest"
	"time"
)

const (
	testSubjectKind    SubjectKind = "test"
	testRateLimitID                = "111111111111"
	testLeaseSize                  = 100
	testLargeLeaseSize             = 20_000
)

func newTestBucket() *Bucket {
	return NewBucket(testSubjectKind, testRateLimitID, testLeaseSize, testLeaseSize)
}

func newTestBucketFor(subjectKind SubjectKind, id string) *Bucket {
	return NewBucket(subjectKind, id, testLeaseSize, testLeaseSize)
}

func newTestLargeBucket() *Bucket {
	return NewBucket(testSubjectKind, testRateLimitID, testLargeLeaseSize, testLargeLeaseSize)
}

// newTestRateLimiter starts a limiter and registers its shutdown with the test.
func newTestRateLimiter(t *testing.T, acquire acquireFunc) *Limiter {
	t.Helper()
	limiter := New(context.Background(), nil, Metrics{})
	limiter.acquire = acquire
	t.Cleanup(limiter.Close)
	return limiter
}

// TestLimiterStopsWhenParentContextIsCanceled verifies that cancellation of
// the context passed to New stops the batcher.
func TestLimiterStopsWhenParentContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	limiter := New(ctx, nil, Metrics{})
	t.Cleanup(limiter.Close)
	done := make(chan struct{})
	go func() {
		limiter.shutdown.Wait()
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("batcher did not stop after parent context cancellation")
	}
}

// bucketState is a snapshot of a bucket's state.
type bucketState struct {
	available int
	hasRefill bool
	closed    bool
}

// bucketSnapshot reads bucket state while holding its mutex.
func bucketSnapshot(bucket *Bucket) bucketState {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	return bucketState{
		available: bucket.available,
		hasRefill: bucket.refill != nil,
		closed:    bucket.closed,
	}
}

// applyTestLease applies a test lease unless the bucket is closed.
func applyTestLease(bucket *Bucket, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if !bucket.closed {
		bucket.applyLeaseLocked(grantedUnits, capacityUnits)
	}
}

// limiterRetryAfter returns the current global backoff deadline.
func limiterRetryAfter(l *Limiter) time.Time {
	unixNano := l.retryAfter.Load()
	if unixNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, unixNano)
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

// TestBucketsConsumeIndependently verifies that buckets for distinct subject
// kinds and identifiers do not share local capacity.
func TestBucketsConsumeIndependently(t *testing.T) {
	limiter := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 2}}, nil
	})
	firstBucket := newTestBucketFor("first", "111111111111")
	secondBucket := newTestBucketFor("second", "222222222222")
	thirdBucket := newTestBucketFor("third", "222222222222")
	applyTestLease(firstBucket, 2, 2)
	applyTestLease(secondBucket, 2, 2)
	applyTestLease(thirdBucket, 2, 2)

	if err := limiter.Consume(context.Background(), firstBucket, 2); err != nil {
		t.Fatalf("first bucket consume: %v", err)
	}
	if err := limiter.Consume(context.Background(), firstBucket, 2); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("first bucket second consume: got %v, want ErrCapacityExceeded", err)
	}
	if err := limiter.Consume(context.Background(), secondBucket, 2); err != nil {
		t.Fatalf("second bucket consume: %v", err)
	}
	if err := limiter.Consume(context.Background(), thirdBucket, 2); err != nil {
		t.Fatalf("third bucket consume: %v", err)
	}
	thirdBucket.Restore(2)
	if err := limiter.Consume(context.Background(), thirdBucket, 2); err != nil {
		t.Fatalf("third bucket consume after restoration: %v", err)
	}
}

// TestLimiterValidatesCost verifies that costs stay within a bucket's
// supported range.
func TestLimiterValidatesCost(t *testing.T) {
	l := newTestRateLimiter(t, nil)
	bucket := newTestBucket()
	for _, cost := range []int{-1, 0, 101} {
		if err := l.Consume(context.Background(), bucket, cost); !errors.Is(err, ErrInvalidCost) {
			t.Fatalf("cost %d: got %v, want ErrInvalidCost", cost, err)
		}
	}
}

// TestLimiterValidatesLargeCost verifies a bucket can support a larger maximum
// cost.
func TestLimiterValidatesLargeCost(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newTestLargeBucket()
	for _, count := range []int{-1, 0, testLargeLeaseSize + 1} {
		if err := limiter.Consume(context.Background(), bucket, count); !errors.Is(err, ErrInvalidCost) {
			t.Fatalf("cost %d: got %v, want ErrInvalidCost", count, err)
		}
	}
	applyTestLease(bucket, testLargeLeaseSize, testLargeLeaseSize)
	if err := limiter.Consume(context.Background(), bucket, testLargeLeaseSize); err != nil {
		t.Fatalf("maximum cost: %v", err)
	}
}

// TestLimiterConsumesLocalCapacity verifies immediate local consumption
// and rejection when the remaining capacity is insufficient.
func TestLimiterConsumesLocalCapacity(t *testing.T) {
	l := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 10}}, nil
	})
	bucket := newTestBucket()
	applyTestLease(bucket, 10, 10)

	if err := l.Consume(context.Background(), bucket, 6); err != nil {
		t.Fatalf("consume: %v", err)
	}
	if state := bucketSnapshot(bucket); state.available != 4 {
		t.Fatalf("available capacity = %d, want 4", state.available)
	}
	if err := l.Consume(context.Background(), bucket, 5); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume exhausted capacity: got %v, want ErrCapacityExceeded", err)
	}
}

// TestLimiterCanceledContextDoesNotConsumeLocalCapacity verifies that
// an already canceled request leaves local capacity unchanged.
func TestLimiterCanceledContextDoesNotConsumeLocalCapacity(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newTestBucket()
	applyTestLease(bucket, 10, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.Consume(ctx, bucket, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("consume with canceled context: %v", err)
	}
	if state := bucketSnapshot(bucket); state.available != 10 {
		t.Fatalf("available capacity = %d, want 10", state.available)
	}
}

// TestLimiterCallerDeadlineTakesPrecedence verifies that a caller
// deadline wins over shutdown and internal timeout.
func TestLimiterCallerDeadlineTakesPrecedence(t *testing.T) {
	previousMaxWaitDuration := maxWaitDuration
	maxWaitDuration = 0
	t.Cleanup(func() { maxWaitDuration = previousMaxWaitDuration })
	limiter := &Limiter{}
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	limiter.shutdown.cancel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	for range 100 {
		bucket := newTestBucket()
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
	bucket := newTestBucket()
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
	bucket := newTestBucket()

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
	if state := bucketSnapshot(bucket); state.available != 20 {
		t.Fatalf("remaining capacity = %d, want 20", state.available)
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
	bucket := newTestBucket()

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
	previousMaxWaitDuration := maxWaitDuration
	maxWaitDuration = 10 * time.Millisecond
	t.Cleanup(func() { maxWaitDuration = previousMaxWaitDuration })
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
	bucket := newTestBucket()
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
	previousAcquireTimeout := defaultAcquireTimeout
	defaultAcquireTimeout = 10 * time.Millisecond
	t.Cleanup(func() { defaultAcquireTimeout = previousAcquireTimeout })
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return []leaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 1, CapacityUnits: 100,
		}}, nil
	})
	bucket := newTestBucket()
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
	state := bucketSnapshot(bucket)
	if state.hasRefill || state.available != 0 {
		t.Fatalf("state after acquisition timeout: pending=%t available=%d", state.hasRefill, state.available)
	}
}

// TestLimiterCancellationAndGrantResolveOnce verifies that cancellation
// and completion resolve a waiter once and leave capacity consistent with the
// winner.
func TestLimiterCancellationAndGrantResolveOnce(t *testing.T) {
	for range 100 {
		bucket := newTestBucket()
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
		state := bucketSnapshot(bucket)
		if state.hasRefill {
			t.Fatal("completed generation remained attached")
		}
		switch {
		case errors.Is(err, context.Canceled):
			if state.available != 1 {
				t.Fatalf("cancellation won but available capacity = %d, want 1", state.available)
			}
		case err == nil:
			if state.available != 0 {
				t.Fatalf("grant won but available capacity = %d, want 0", state.available)
			}
		default:
			t.Fatalf("race result: %v", err)
		}
	}
}

// TestLimiterIgnoresStaleGenerationResult verifies that an old refill
// cannot change a newer generation or its waiter.
func TestLimiterIgnoresStaleGenerationResult(t *testing.T) {
	bucket := newTestBucket()
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
	bucket := newTestBucket()
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
		state := bucketSnapshot(bucket)
		return !state.hasRefill && state.available == 7
	})
}

// TestLimiterBatchesSubjectKinds verifies that distinct subject kinds are
// acquired in the same batch.
func TestLimiterBatchesSubjectKinds(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	limiter := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		requests <- request
		results := make([]leaseResult, len(request))
		for i, r := range request {
			results[i] = leaseResult{SubjectKind: r.SubjectKind, SubjectID: r.SubjectID, CapacityUnits: 100}
		}
		return results, nil
	})

	refills := make([]*refill, 0, 2)
	waiters := make([]*waiter, 0, 2)
	for _, bucket := range []*Bucket{newTestBucketFor("first", "111111111111"), newTestBucketFor("second", "222222222222")} {
		_, refill, _ := bucket.consume(1, true)
		waiter := bucket.activateRefill(refill, 1, true, true)
		if waiter == nil {
			t.Fatal("waiter was not admitted")
		}
		refills = append(refills, refill)
		waiters = append(waiters, waiter)
		limiter.queue <- refill
	}
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
	bucket := newTestBucket()
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
		return !bucketSnapshot(bucket).hasRefill
	})
	if state := bucketSnapshot(bucket); state.available != 24 {
		t.Fatalf("available capacity = %d, want 24", state.available)
	}
}

// TestLimiterStartsGlobalBackoffAfterRefillError verifies that a failed
// acquisition suppresses refills and waiter admission until the retry deadline.
func TestLimiterStartsGlobalBackoffAfterRefillError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		l := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
			if calls.Add(1) == 1 {
				return nil, errors.New("lease acquirer is unavailable")
			}
			return []leaseResult{{
				SubjectKind:   request[0].SubjectKind,
				SubjectID:     request[0].SubjectID,
				CapacityUnits: 100,
			}}, nil
		})
		bucket := newTestBucket()
		applyTestLease(bucket, 2, 100)
		if err := l.Consume(context.Background(), bucket, 1); err != nil {
			t.Fatalf("initial consume: %v", err)
		}
		time.Sleep(batchDelay)
		synctest.Wait()
		retryAt := limiterRetryAfter(l)
		if want := time.Now().Add(refillBackoffDuration); !retryAt.Equal(want) {
			t.Fatalf("retry after = %s, want %s", retryAt, want)
		}
		if bucketSnapshot(bucket).hasRefill {
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

		time.Sleep(refillBackoffDuration)
		if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
			t.Fatalf("consume after backoff: got %v, want ErrCapacityExceeded", err)
		}
		synctest.Wait()
		if calls.Load() != 2 {
			t.Fatalf("refill attempts after backoff = %d, want 2", calls.Load())
		}
		if retryAt := limiterRetryAfter(l); !retryAt.IsZero() {
			t.Fatalf("retry after successful acquisition = %s, want zero", retryAt)
		}
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
			synctest.Test(t, func(t *testing.T) {
				l := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
					return test.results(requests[0]), nil
				})
				bucket := newTestBucket()
				_ = l.Consume(context.Background(), bucket, 1)
				synctest.Wait()
				if limiterRetryAfter(l).IsZero() {
					t.Fatal("invalid lease result did not start global backoff")
				}
			})
		})
	}
}

// TestLimiterGlobalBackoffBlocksQueuedAndConcurrentRefills verifies
// that backoff rejects queued and concurrent refill attempts.
func TestLimiterGlobalBackoffBlocksQueuedAndConcurrentRefills(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int32
		l := newTestRateLimiter(t, func(_ context.Context, _ []leaseRequest) ([]leaseResult, error) {
			calls.Add(1)
			return nil, nil
		})
		l.retryAfter.Store(time.Now().Add(refillBackoffDuration).UnixNano())

		queued := newTestBucket()
		_, queuedRefill, _ := queued.consume(2, true)
		queued.activateRefill(queuedRefill, 0, false, true)
		l.queue <- queuedRefill
		time.Sleep(batchDelay)
		synctest.Wait()
		if bucketSnapshot(queued).hasRefill {
			t.Fatal("queued refill remained active during global backoff")
		}
		if calls.Load() != 0 {
			t.Fatalf("refill attempts during global backoff = %d, want 0", calls.Load())
		}

		bucket := newTestBucketFor("other", "222222222222")
		var wg sync.WaitGroup
		for range 100 {
			wg.Go(func() {
				if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
					t.Errorf("consume during global backoff: got %v, want ErrCapacityExceeded", err)
				}
			})
		}
		wg.Wait()
		if bucketSnapshot(bucket).hasRefill {
			t.Fatal("global backoff queued a refill")
		}
	})
}

// TestLimiterClosedBucketCompletesInFlightRefill verifies that
// disabling a bucket rejects waiters and discards its late grant.
func TestLimiterClosedBucketCompletesInFlightRefill(t *testing.T) {
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
	bucket := newTestBucket()
	result := make(chan error, 1)
	go func() { result <- l.Consume(context.Background(), bucket, 1) }()
	<-started
	bucket.Close()
	if err := <-result; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("closed waiter: %v", err)
	}
	close(release)
	waitForRateLimit(t, func() bool {
		state := bucketSnapshot(bucket)
		return state.closed && !state.hasRefill && state.available == 0
	})
}

// TestLimiterQueueFullRejectsRequestsWithoutLocalCapacity verifies that
// a rejected queue publication cannot authorize waiting.
func TestLimiterQueueFullRejectsRequestsWithoutLocalCapacity(t *testing.T) {
	l := &Limiter{
		queue: make(chan *refill, 1),
	}
	l.shutdown.ctx = context.Background()
	l.queue <- &refill{}
	bucket := newTestBucket()

	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("consume with a full queue: got %v, want ErrCapacityExceeded", err)
	}
	if err := l.Consume(context.Background(), bucket, 1); !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("second consume with a full queue: got %v, want ErrCapacityExceeded", err)
	}
	if !limiterRetryAfter(l).IsZero() {
		t.Fatal("full queue started backoff")
	}
	if bucketSnapshot(bucket).hasRefill {
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
	bucket := newTestBucket()
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
	state := bucketSnapshot(bucket)
	if state.hasRefill || state.available != 0 {
		t.Fatalf("state after shutdown: queued=%t available=%d", state.hasRefill, state.available)
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
	firstBucket := newTestBucket()
	secondBucket := newTestBucketFor("other", "222222222222")
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
	l.shutdown.ctx, l.shutdown.cancel = context.WithCancel(context.Background())
	l.shutdown.cancel()

	bucket := newTestBucket()
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	if l.collectAndRefill(refill) {
		t.Fatal("batch collection continued after shutdown")
	}
	if bucketSnapshot(bucket).hasRefill {
		t.Fatal("shutdown left collected refill queued")
	}
}
