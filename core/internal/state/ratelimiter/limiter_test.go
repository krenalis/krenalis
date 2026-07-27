// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

const (
	testSubjectKind SubjectKind = "test"
	testRateLimitID             = "111111111111"
	testLeaseSize               = 100
)

func newTestBucket(limiters ...*Limiter) *Bucket {
	limiter := new(Limiter)
	if len(limiters) != 0 {
		limiter = limiters[0]
	}
	return limiter.NewBucket(testSubjectKind, testRateLimitID, testLeaseSize, testLeaseSize)
}

func newTestRateLimiter(t *testing.T, acquire acquireFunc) *Limiter {
	t.Helper()
	limiter := New(nil, Metrics{})
	limiter.acquire = acquire
	t.Cleanup(func() { limiter.Close(context.Background()) })
	return limiter
}

func applyTestLease(bucket *Bucket, granted, capacity int) {
	bucket.mu.Lock()
	bucket.applyLeaseLocked(granted, capacity)
	bucket.mu.Unlock()
}

func bucketAvailable(bucket *Bucket) int {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	return bucket.available
}

func pendingUnits(bucket *Bucket) int {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.refill == nil {
		return 0
	}
	return bucket.refill.pendingUnits
}

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

type testCounter struct{ calls atomic.Int32 }

func (counter *testCounter) Inc() { counter.calls.Add(1) }

func TestLimiterCloseStopsBatcher(t *testing.T) {
	limiter := New(nil, Metrics{})
	limiter.Close(context.Background())
	select {
	case <-limiter.shutdown.done:
	default:
		t.Fatal("Close did not stop the batcher")
	}
}

func TestLimiterCloseHonorsContextDuringProactiveRefill(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(context.Context, []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-release
		return nil, context.Canceled
	})
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 100, 100)
	if err := bucket.Consume(context.Background(), 80); err != nil {
		t.Fatalf("consume that starts proactive refill: %v", err)
	}
	<-started

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	limiter.Close(ctx)
	select {
	case <-limiter.shutdown.done:
		t.Fatal("batcher stopped before the non-cooperative acquirer returned")
	default:
	}
	close(release)
	limiter.Close(context.Background())
	if backoff := limiter.backoff.Load(); backoff != nil {
		t.Fatalf("shutdown backoff = %v, want nil", backoff)
	}
	if got := bucketAvailable(bucket); got != 20 {
		t.Fatalf("capacity after shutdown = %d, want 20", got)
	}
}

func TestLimiterConsumesAndRefills(t *testing.T) {
	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		results := make([]leaseResult, len(requests))
		for i, request := range requests {
			results[i] = leaseResult{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, GrantedUnits: request.RequestedUnits, CapacityUnits: testLeaseSize}
		}
		return results, nil
	})
	bucket := newTestBucket(limiter)
	if err := bucket.Consume(context.Background(), 60); err != nil {
		t.Fatalf("initial consume: %v", err)
	}
	if err := bucket.Consume(context.Background(), 40); err != nil {
		t.Fatalf("second consume: %v", err)
	}
	if got := bucketAvailable(bucket); got != 0 {
		t.Fatalf("available capacity = %d, want 0", got)
	}
}

func TestLimiterCanceledContextConsumesLocalCapacity(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 10, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bucket.Consume(ctx, 1); err != nil {
		t.Fatalf("canceled local consume: %v", err)
	}
	if got := bucketAvailable(bucket); got != 9 {
		t.Fatalf("available capacity = %d, want 9", got)
	}
}

func TestLimiterCanceledContextCancelsWaiterButStartsRefill(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-release
		return []leaseResult{{
			SubjectKind:   requests[0].SubjectKind,
			SubjectID:     requests[0].SubjectID,
			GrantedUnits:  1,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newTestBucket(limiter)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := bucket.Consume(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled consume: %v", err)
	}
	<-started
	if got := pendingUnits(bucket); got != 0 {
		t.Fatalf("pending units after cancellation = %d, want 0", got)
	}
	close(release)
}

func TestLimiterCallerDeadlineTakesPrecedence(t *testing.T) {
	previous := maxWaitDuration
	maxWaitDuration = 0
	t.Cleanup(func() { maxWaitDuration = previous })
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	for range 100 {
		bucket := newTestBucket()
		bucket.mu.Lock()
		refill := bucket.newRefillLocked()
		waiter := bucket.admitWaiterLocked(refill, 1)
		bucket.mu.Unlock()
		if err := waitForRefill(ctx, waiter); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("wait returned %v, want context deadline exceeded", err)
		}
		bucket.rejectRefill(refill, ErrLimiterUnavailable)
	}
}

func TestLimiterKeepsPositiveCapacityForSmallerRequest(t *testing.T) {
	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: requests[0].SubjectKind, SubjectID: requests[0].SubjectID, GrantedUnits: 10, CapacityUnits: 100}}, nil
	})
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 90, 100)
	if err := bucket.Consume(context.Background(), 100); !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("large request = %v, want ErrLimiterUnavailable", err)
	}
	if err := bucket.Consume(context.Background(), 90); err != nil {
		t.Fatalf("smaller request did not use positive local capacity: %v", err)
	}
}

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
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, GrantedUnits: 60, CapacityUnits: 100}}, nil
	})
	bucket := newTestBucket(limiter)
	first := make(chan error, 1)
	second := make(chan error, 1)
	third := make(chan error, 1)
	go func() { first <- bucket.Consume(context.Background(), 40) }()
	<-requests
	go func() { second <- bucket.Consume(context.Background(), 30) }()
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 70 })
	go func() { third <- bucket.Consume(context.Background(), 20) }()
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 90 })
	close(release)
	if err := <-first; err != nil {
		t.Fatalf("first waiter: %v", err)
	}
	if err := <-second; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("second waiter: %v", err)
	}
	if err := <-third; !errors.Is(err, ErrCapacityExceeded) {
		t.Fatalf("third waiter: %v", err)
	}
}

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
		return []leaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, GrantedUnits: 100, CapacityUnits: 100}}, nil
	})
	bucket := newTestBucket(limiter)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- bucket.Consume(ctx, 60) }()
	<-requests
	go func() { second <- bucket.Consume(context.Background(), 40) }()
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 100 })
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter: %v", err)
	}
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 40 })
	close(release)
	if err := <-second; err != nil {
		t.Fatalf("remaining waiter: %v", err)
	}
}

func TestLimiterCancellationAndGrantResolveOnce(t *testing.T) {
	for range 100 {
		bucket := newTestBucket()
		applyTestLease(bucket, 0, 100)
		bucket.mu.Lock()
		refill := bucket.newRefillLocked()
		waiter := bucket.admitWaiterLocked(refill, 1)
		bucket.mu.Unlock()

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

		bucket.mu.Lock()
		available := bucket.available
		hasRefill := bucket.refill != nil
		bucket.mu.Unlock()
		if hasRefill {
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

func TestLimiterIgnoresStaleGenerationResult(t *testing.T) {
	bucket := newTestBucket()
	applyTestLease(bucket, 0, 100)
	bucket.mu.Lock()
	first := bucket.newRefillLocked()
	bucket.mu.Unlock()
	bucket.rejectRefill(first, ErrLimiterUnavailable)

	bucket.mu.Lock()
	second := bucket.newRefillLocked()
	waiter := bucket.admitWaiterLocked(second, 1)
	bucket.mu.Unlock()
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
	<-second.done
	if waiter.err != nil {
		t.Fatalf("second-generation waiter: %v", waiter.err)
	}
}

func TestLimiterWaitHasFiniteInternalTimeout(t *testing.T) {
	previous := maxWaitDuration
	maxWaitDuration = 5 * time.Millisecond
	t.Cleanup(func() { maxWaitDuration = previous })
	started := make(chan struct{})
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return nil, context.Canceled
	})
	result := make(chan error, 1)
	go func() { result <- newTestBucket(limiter).Consume(context.Background(), 1) }()
	<-started
	if err := <-result; !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("internal timeout: %v", err)
	}
	close(release)
}

func TestLimiterLeaseAcquisitionHasFiniteTimeout(t *testing.T) {
	previous := defaultAcquireTimeout
	defaultAcquireTimeout = 10 * time.Millisecond
	t.Cleanup(func() { defaultAcquireTimeout = previous })
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return []leaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  1,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newTestBucket(limiter)
	result := make(chan error, 1)
	go func() { result <- bucket.Consume(context.Background(), 1) }()
	<-started

	select {
	case err := <-result:
		if !errors.Is(err, ErrLimiterUnavailable) {
			t.Fatalf("acquisition timeout: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("lease acquisition did not time out")
	}
	bucket.mu.Lock()
	available := bucket.available
	hasRefill := bucket.refill != nil
	bucket.mu.Unlock()
	if hasRefill || available != 0 {
		t.Fatalf("state after acquisition timeout: pending=%t available=%d", hasRefill, available)
	}
}

func TestLimiterQueuesOneRefill(t *testing.T) {
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
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  10,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newTestBucket(limiter)
	results := make(chan error, 3)
	go func() { results <- bucket.Consume(context.Background(), 1) }()
	if request := <-requests; len(request) != 1 {
		t.Fatalf("batch contains %d requests, want 1", len(request))
	}
	for range 2 {
		go func() { results <- bucket.Consume(context.Background(), 1) }()
	}
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 3 })
	close(release)
	for range 3 {
		if err := <-results; err != nil {
			t.Fatalf("consume after refill: %v", err)
		}
	}
	if got := bucketAvailable(bucket); got != 7 {
		t.Fatalf("available capacity = %d, want 7", got)
	}
}

func TestLimiterBatchesSubjectKinds(t *testing.T) {
	limiter := &Limiter{queue: make(chan *refill, queueSize)}
	limiter.shutdown.ctx, limiter.shutdown.cancel = context.WithCancel(context.Background())
	t.Cleanup(limiter.shutdown.cancel)
	var acquired []leaseRequest
	limiter.acquire = func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		acquired = append(acquired, requests...)
		results := make([]leaseResult, len(requests))
		for i, request := range requests {
			results[i] = leaseResult{
				SubjectKind:   request.SubjectKind,
				SubjectID:     request.SubjectID,
				CapacityUnits: 100,
			}
		}
		return results, nil
	}

	buckets := []*Bucket{
		limiter.NewBucket("first", "111111111111", 100, 100),
		limiter.NewBucket("second", "222222222222", 100, 100),
	}
	refills := make([]*refill, len(buckets))
	for i, bucket := range buckets {
		bucket.mu.Lock()
		refills[i] = bucket.newRefillLocked()
		bucket.admitWaiterLocked(refills[i], 1)
		bucket.mu.Unlock()
	}
	limiter.queue <- refills[1]
	if !limiter.collectAndRefill(refills[0]) {
		t.Fatal("batcher stopped while collecting refills")
	}
	if len(acquired) != 2 {
		t.Fatalf("batch contains %d requests, want 2", len(acquired))
	}
}

func TestLimiterAddsLeaseAfterConcurrentConsumption(t *testing.T) {
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
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  20,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 10, 100)
	if err := bucket.Consume(context.Background(), 1); err != nil {
		t.Fatalf("first consume: %v", err)
	}
	<-started
	if err := bucket.Consume(context.Background(), 5); err != nil {
		t.Fatalf("concurrent consume: %v", err)
	}
	close(release)
	waitForRateLimit(t, func() bool {
		bucket.mu.Lock()
		defer bucket.mu.Unlock()
		return bucket.refill == nil
	})
	if got := bucketAvailable(bucket); got != 24 {
		t.Fatalf("available capacity = %d, want 24", got)
	}
}

func TestLimiterRejectsInvalidLeaseResults(t *testing.T) {
	for _, test := range []struct {
		name    string
		results func(leaseRequest) []leaseResult
	}{
		{name: "missing", results: func(leaseRequest) []leaseResult { return nil }},
		{name: "unexpected subject", results: func(request leaseRequest) []leaseResult {
			return []leaseResult{{SubjectKind: request.SubjectKind, SubjectID: "other", CapacityUnits: 1}}
		}},
		{name: "duplicate", results: func(request leaseRequest) []leaseResult {
			result := leaseResult{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, CapacityUnits: 1}
			return []leaseResult{result, result}
		}},
		{name: "grant exceeds request", results: func(request leaseRequest) []leaseResult {
			return []leaseResult{{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, GrantedUnits: request.RequestedUnits + 1, CapacityUnits: request.RequestedUnits + 1}}
		}},
		{name: "zero capacity", results: func(request leaseRequest) []leaseResult {
			return []leaseResult{{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
				return test.results(requests[0]), nil
			})
			bucket := newTestBucket(limiter)
			if err := bucket.Consume(context.Background(), 1); err == nil || errors.Is(err, ErrCapacityExceeded) || errors.Is(err, ErrLimiterUnavailable) {
				t.Fatalf("invalid lease result = %v, want generic internal error", err)
			}
			if backoff := limiter.backoff.Load(); backoff == nil {
				t.Fatal("invalid lease result did not start backoff")
			}
			if err := bucket.Consume(context.Background(), 1); err == nil || errors.Is(err, ErrCapacityExceeded) || errors.Is(err, ErrLimiterUnavailable) {
				t.Fatalf("invalid-response backoff = %v, want generic internal error", err)
			}
		})
	}
}

func TestLimiterRejectsWhenQueueIsFull(t *testing.T) {
	queueFull := new(testCounter)
	limiter := &Limiter{queue: make(chan *refill, 1), metrics: Metrics{QueueFull: queueFull}}
	limiter.queue <- new(refill)
	if err := newTestBucket(limiter).Consume(context.Background(), 1); !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("consume with full queue: %v", err)
	}
	if got := queueFull.calls.Load(); got != 1 {
		t.Fatalf("queue-full metric = %d, want 1", got)
	}
}

func TestLimiterRejectsRefillsDuringBackoff(t *testing.T) {
	var calls atomic.Int32
	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		calls.Add(1)
		return nil, errors.New("database unavailable")
	})
	bucket := newTestBucket(limiter)
	if err := bucket.Consume(context.Background(), 1); !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("failed refill: %v", err)
	}
	if err := bucket.Consume(context.Background(), 1); !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("backoff refill: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("acquisition calls = %d, want 1", got)
	}
}
