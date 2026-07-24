// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testRateLimitID = "111111111111"

func newTestRateLimiter(t *testing.T, acquire apiRateLimitLeaseAcquirer) *rateLimiter {
	t.Helper()
	l := &rateLimiter{
		acquireLeases:  acquire,
		now:            time.Now,
		maxWait:        apiRateLimitMaxWaitDuration,
		acquireTimeout: apiRateLimitAcquireTimeout,
		refillQueue:    make(chan *rateLimitRefill, apiRateLimitQueueSize),
	}
	l.close.ctx, l.close.cancel = context.WithCancel(context.Background())
	l.close.Add(1)
	go l.runBatcher()
	t.Cleanup(l.Close)
	return l
}

func bucketSnapshot(bucket *rateLimitBucket) (available, target, threshold int, refillQueued, disabled bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	return bucket.available, bucket.target, bucket.threshold, bucket.refill != nil, bucket.disabled
}

func applyTestLease(bucket *rateLimitBucket, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if !bucket.disabled {
		bucket.applyLeaseLocked(grantedUnits, capacityUnits)
	}
}

func rateLimiterRetryAfter(l *rateLimiter) time.Time {
	unixNano := l.refillRetryAfter.Load()
	if unixNano == 0 {
		return time.Time{}
	}
	return time.Unix(0, unixNano)
}

type testRateLimitClock struct{ unixNano atomic.Int64 }

func newTestRateLimitClock() *testRateLimitClock {
	clock := &testRateLimitClock{}
	clock.Set(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC))
	return clock
}

func (clock *testRateLimitClock) Now() time.Time {
	return time.Unix(0, clock.unixNano.Load())
}

func (clock *testRateLimitClock) Set(now time.Time) {
	clock.unixNano.Store(now.UnixNano())
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

func refillPendingCost(bucket *rateLimitBucket) int {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.refill == nil {
		return 0
	}
	return bucket.refill.pendingCost
}

func TestRateLimitBucketIsPreservedWhenEntitiesAreReplaced(t *testing.T) {
	organizationBucket := newNonspecificBucket("111111111111")
	workspaceBucket := newWorkspaceBucket("222222222222")
	ingestionBucket := newIngestionBucket("222222222222")
	organization := &Organization{
		mu:         new(sync.Mutex),
		workspaces: map[string]*Workspace{},
		bucket:     organizationBucket,
		ID:         "111111111111",
	}
	workspace := &Workspace{
		mu:              new(sync.Mutex),
		organization:    organization,
		apiBucket:       workspaceBucket,
		ingestionBucket: ingestionBucket,
		ID:              "222222222222",
	}
	organization.workspaces[workspace.ID] = workspace
	if organization.bucket == workspace.apiBucket {
		t.Fatal("organization and workspace share an API rate-limit bucket")
	}
	state := &State{
		mu:            new(sync.Mutex),
		organizations: map[string]*Organization{organization.ID: organization},
		workspaces:    map[string]*Workspace{workspace.ID: workspace},
	}

	updatedOrganization := state.replaceOrganization(organization.ID, func(organization *Organization) {
		organization.Name = "updated"
	})
	updatedWorkspace := state.replaceWorkspace(workspace.ID, func(workspace *Workspace) {
		workspace.Name = "updated"
	})

	if updatedOrganization.bucket != organizationBucket {
		t.Fatal("organization update replaced its API rate-limit bucket")
	}
	if updatedWorkspace.apiBucket != workspaceBucket {
		t.Fatal("workspace update replaced its API rate-limit bucket")
	}
	if updatedWorkspace.ingestionBucket != ingestionBucket {
		t.Fatal("workspace update replaced its ingestion rate-limit bucket")
	}
}

func TestDisabledRateLimitBucketDoesNotRequestOrApplyCapacity(t *testing.T) {
	bucket := newWorkspaceBucket("222222222222")
	applyTestLease(bucket, 10, 10)
	bucket.disable()

	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill != nil || waiter != nil {
		t.Fatalf("disabled bucket consumed or requested capacity: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	applyTestLease(bucket, 10, 10)
	available, _, _, _, disabled := bucketSnapshot(bucket)
	if available != 0 || !disabled {
		t.Fatalf("disabled bucket state = available:%d disabled:%t, want 0:true", available, disabled)
	}
}

// TestIngestionRateLimitBucketAllowsInitialOverdraft verifies that a first
// single-event request can queue its initial refill without being rejected.
func TestIngestionRateLimitBucketAllowsInitialOverdraft(t *testing.T) {
	bucket := newIngestionBucket("222222222222")

	satisfied, refill, waiter := bucket.consume(1, true)
	if !satisfied || refill == nil || waiter != nil {
		t.Fatalf("initial ingestion consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	bucket.rejectRefill(refill)
	satisfied, refill, waiter = bucket.consume(1, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("second ingestion consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
}

func TestRateLimitBucketAllowsOverdraftOnlyWithPendingRefill(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("cold bucket consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	if waiter := bucket.activateRefill(refill, 0, false, true); waiter != nil {
		t.Fatal("unexpected waiter")
	}
	satisfied, queued, waiter := bucket.consume(1, true)
	if !satisfied || queued != nil || waiter != nil {
		t.Fatalf("pending refill overdraft: satisfied=%t refill=%p waiter=%p", satisfied, queued, waiter)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != -1 {
		t.Fatalf("available capacity = %d, want -1", available)
	}
}

func TestRateLimitBucketDoesNotOverdraftHigherCostsOrClosedLimiter(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	satisfied, queued, waiter := bucket.consume(2, true)
	if satisfied || queued != nil || waiter == nil {
		t.Fatalf("cost-2 request was not admitted as waiter")
	}
	satisfied, queued, waiter = bucket.consume(1, false)
	if satisfied || queued != nil || waiter != nil {
		t.Fatal("closed limiter allowed overdraft or waiting")
	}
}

func TestRateLimitBucketLeaseRepaysOverdraft(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	applyTestLease(bucket, 0, 100)

	bucket.mu.Lock()
	bucket.available = -5
	bucket.mu.Unlock()
	applyTestLease(bucket, 10, 100)
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 5 {
		t.Fatalf("full debt repayment = %d, want 5", available)
	}

	bucket.mu.Lock()
	bucket.available = -5
	bucket.mu.Unlock()
	applyTestLease(bucket, 2, 100)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != -3 {
		t.Fatalf("partial debt repayment = %d, want -3", available)
	}
	applyTestLease(bucket, 0, 100)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != -3 {
		t.Fatalf("zero lease changed debt to %d, want -3", available)
	}
}

func TestRateLimitBucketCapsRefillRequestWithOverdraft(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	bucket.mu.Lock()
	bucket.target = apiRateLimitLeaseSize
	bucket.available = -apiRateLimitOverdraftLimit
	refill := bucket.newRefillLocked()
	bucket.mu.Unlock()
	bucket.activateRefill(refill, 0, false, true)

	request, ok := bucket.refillRequest(refill)
	if !ok || request.RequestedUnits != apiRateLimitLeaseSize {
		t.Fatalf("requested refill = %d, %t, want %d, true", request.RequestedUnits, ok, apiRateLimitLeaseSize)
	}
}

func TestRateLimitBucketThresholdScalesWithTarget(t *testing.T) {
	for _, test := range []struct {
		target    int
		threshold int
	}{
		{target: 100, threshold: 25},
		{target: 40, threshold: 10},
		{target: 10, threshold: 2},
		{target: 1, threshold: 1},
	} {
		t.Run(strconv.Itoa(test.target), func(t *testing.T) {
			bucket := newNonspecificBucket("111111111111")
			applyTestLease(bucket, 0, test.target)
			if bucket.threshold != test.threshold {
				t.Fatalf("threshold = %d, want %d", bucket.threshold, test.threshold)
			}
		})
	}
}

// TestRateLimitBucketQueuesRefillRelativeToOperationCost verifies that a
// variable-cost operation queues a refill while one similar operation remains.
func TestRateLimitBucketQueuesRefillRelativeToOperationCost(t *testing.T) {
	bucket := newIngestionBucket("222222222222")
	applyTestLease(bucket, 1_000, 1_000)

	for range 2 {
		satisfied, refill, waiter := bucket.consume(250, true)
		if !satisfied || refill != nil || waiter != nil {
			t.Fatalf("early consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
		}
	}
	satisfied, refill, waiter := bucket.consume(250, true)
	if !satisfied || refill == nil || waiter != nil {
		t.Fatalf("low-capacity consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 250 {
		t.Fatalf("available capacity = %d, want 250", available)
	}
}

func TestCanonicalEntitiesConsumeTheirOwnBuckets(t *testing.T) {
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		return []apiRateLimitLeaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 2}}, nil
	})
	organization := &Organization{
		bucket:      newNonspecificBucket("111111111111"),
		rateLimiter: l,
	}
	workspace := &Workspace{
		apiBucket:       newWorkspaceBucket("222222222222"),
		ingestionBucket: newIngestionBucket("222222222222"),
		organization:    organization,
	}
	applyTestLease(organization.bucket, 2, 2)
	applyTestLease(workspace.apiBucket, 2, 2)
	applyTestLease(workspace.ingestionBucket, 2, 2)

	if err := organization.ConsumeRateLimitCapacity(context.Background(), 2); err != nil {
		t.Fatalf("organization consume: %v", err)
	}
	if err := organization.ConsumeRateLimitCapacity(context.Background(), 2); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("organization second consume: got %v, want ErrAPICapacityExceeded", err)
	}
	if err := workspace.ConsumeRateLimitCapacity(context.Background(), 2); err != nil {
		t.Fatalf("workspace consume: %v", err)
	}
	if err := workspace.ConsumeIngestionRateLimitCapacity(context.Background(), 2); err != nil {
		t.Fatalf("workspace ingestion consume: %v", err)
	}
}

func TestAPIRateLimiterValidatesCost(t *testing.T) {
	l := newTestRateLimiter(t, nil)
	bucket := newNonspecificBucket(testRateLimitID)
	for _, cost := range []int{-1, 0, 101} {
		if err := l.consume(context.Background(), bucket, cost); !errors.Is(err, ErrInvalidAPICost) {
			t.Fatalf("cost %d: got %v, want ErrInvalidAPICost", cost, err)
		}
	}
}

// TestIngestionRateLimiterValidatesEventCount verifies the supported batch size.
func TestIngestionRateLimiterValidatesEventCount(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newIngestionBucket(testRateLimitID)
	for _, count := range []int{-1, 0, ingestionRateLimitMaxCost + 1} {
		if err := limiter.consume(context.Background(), bucket, count); !errors.Is(err, ErrInvalidAPICost) {
			t.Fatalf("event count %d: got %v, want ErrInvalidAPICost", count, err)
		}
	}
	applyTestLease(bucket, ingestionRateLimitMaxCost, ingestionRateLimitMaxCost)
	if err := limiter.consume(context.Background(), bucket, ingestionRateLimitMaxCost); err != nil {
		t.Fatalf("maximum event count: %v", err)
	}
}

func TestAPIRateLimiterConsumesLocalCapacity(t *testing.T) {
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		return []apiRateLimitLeaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 10}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	applyTestLease(bucket, 10, 10)

	if err := l.consume(context.Background(), bucket, 6); err != nil {
		t.Fatalf("consume: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 4 {
		t.Fatalf("available capacity = %d, want 4", available)
	}
	if err := l.consume(context.Background(), bucket, 5); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume exhausted capacity: got %v, want ErrAPICapacityExceeded", err)
	}
}

func TestAPIRateLimiterCanceledContextDoesNotConsumeLocalCapacity(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newNonspecificBucket(testRateLimitID)
	applyTestLease(bucket, 10, 10)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := limiter.consume(ctx, bucket, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("consume with canceled context: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 10 {
		t.Fatalf("available capacity = %d, want 10", available)
	}
}

func TestAPIRateLimiterCallerDeadlineTakesPrecedence(t *testing.T) {
	limiter := &rateLimiter{maxWait: 0}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.cancel()
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	for range 100 {
		bucket := newNonspecificBucket(testRateLimitID)
		_, refill, _ := bucket.consume(1, true)
		waiter := bucket.activateRefill(refill, 1, true, true)
		if err := limiter.waitForRefill(ctx, waiter); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("wait returned %v, want context deadline exceeded", err)
		}
		bucket.rejectRefill(refill)
	}
}

func TestAPIRateLimiterUsesRequestedUnitsAsAdmissionBudget(t *testing.T) {
	bucket := newNonspecificBucket(testRateLimitID)
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

func TestAPIRateLimiterSubtractsOverdraftFromAdmissionBudget(t *testing.T) {
	bucket := newNonspecificBucket(testRateLimitID)
	applyTestLease(bucket, 0, 100)
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	for range apiRateLimitOverdraftLimit {
		satisfied, _, waiter := bucket.consume(1, true)
		if !satisfied || waiter != nil {
			t.Fatal("cost-1 operation did not use the available overdraft")
		}
	}
	_, _, waiter := bucket.consume(95, true)
	if waiter == nil {
		t.Fatal("waiter matching the debt-adjusted budget was not admitted")
	}
	satisfied, _, waiter := bucket.consume(1, true)
	if satisfied || waiter != nil {
		t.Fatal("new overdraft or excess waiting was allowed with a waiter present")
	}
	bucket.rejectRefill(refill)
}

func TestAPIRateLimiterServesOnlySatisfiableFIFOPrefix(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 60, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)

	resultA := make(chan error, 1)
	resultB := make(chan error, 1)
	resultC := make(chan error, 1)
	go func() { resultA <- limiter.consume(context.Background(), bucket, 40) }()
	<-requests
	go func() { resultB <- limiter.consume(context.Background(), bucket, 30) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 70 })
	go func() { resultC <- limiter.consume(context.Background(), bucket, 20) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 90 })
	close(release)

	if err := <-resultA; err != nil {
		t.Fatalf("first waiter: %v", err)
	}
	if err := <-resultB; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("second waiter: %v", err)
	}
	if err := <-resultC; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("third waiter bypassed the FIFO head: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 20 {
		t.Fatalf("remaining capacity = %d, want 20", available)
	}
}

func TestAPIRateLimiterCancellationReturnsAdmissionBudget(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 100, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)

	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	second := make(chan error, 1)
	third := make(chan error, 1)
	go func() { first <- limiter.consume(ctx, bucket, 60) }()
	<-requests
	go func() { second <- limiter.consume(context.Background(), bucket, 40) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 100 })
	cancel()
	if err := <-first; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled waiter: %v", err)
	}
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 40 })
	go func() { third <- limiter.consume(context.Background(), bucket, 60) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(bucket) == 100 })
	close(release)

	if err := <-second; err != nil {
		t.Fatalf("second waiter: %v", err)
	}
	if err := <-third; err != nil {
		t.Fatalf("replacement waiter: %v", err)
	}
}

func TestAPIRateLimiterWaitHasFiniteInternalTimeout(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			CapacityUnits: 100,
		}}, nil
	})
	limiter.maxWait = 10 * time.Millisecond
	bucket := newNonspecificBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- limiter.consume(context.Background(), bucket, 1) }()
	<-started
	if err := <-result; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("internal timeout: %v", err)
	}
	if pending := refillPendingCost(bucket); pending != 0 {
		t.Fatalf("pending cost after timeout = %d, want 0", pending)
	}
	close(release)
}

func TestAPIRateLimiterLeaseAcquisitionHasFiniteTimeout(t *testing.T) {
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-ctx.Done()
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 1, CapacityUnits: 100,
		}}, nil
	})
	limiter.acquireTimeout = 10 * time.Millisecond
	bucket := newNonspecificBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- limiter.consume(context.Background(), bucket, 1) }()
	<-started

	select {
	case err := <-result:
		if !errors.Is(err, ErrAPICapacityExceeded) {
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

func TestAPIRateLimiterCancellationAndGrantResolveOnce(t *testing.T) {
	for range 100 {
		bucket := newNonspecificBucket(testRateLimitID)
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

func TestAPIRateLimiterIgnoresStaleGenerationResult(t *testing.T) {
	bucket := newNonspecificBucket(testRateLimitID)
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

func TestAPIRateLimiterQueuesOneRefill(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		requests <- request
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 10, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	results := make(chan error, 3)
	go func() { results <- l.consume(context.Background(), bucket, 1) }()
	if request := <-requests; len(request) != 1 {
		t.Fatalf("batch contains %d requests, want 1", len(request))
	}
	for range 2 {
		go func() { results <- l.consume(context.Background(), bucket, 1) }()
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

func TestAPIRateLimiterBatchesOrganizationsAndWorkspaces(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	limiter := &rateLimiter{
		now:         time.Now,
		maxWait:     apiRateLimitMaxWaitDuration,
		refillQueue: make(chan *rateLimitRefill, apiRateLimitQueueSize),
		acquireLeases: func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
			requests <- request
			results := make([]apiRateLimitLeaseResult, len(request))
			for i, r := range request {
				results[i] = apiRateLimitLeaseResult{SubjectKind: r.SubjectKind, SubjectID: r.SubjectID, CapacityUnits: 100}
			}
			return results, nil
		},
	}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	limiter.close.Add(1)
	t.Cleanup(limiter.Close)

	refills := make([]*rateLimitRefill, 0, 2)
	waiters := make([]*rateLimitWaiter, 0, 2)
	for _, bucket := range []*rateLimitBucket{newNonspecificBucket("111111111111"), newWorkspaceBucket("222222222222")} {
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
		if !errors.Is(waiters[i].err, ErrAPICapacityExceeded) {
			t.Fatalf("waiter with zero grant: %v", waiters[i].err)
		}
	}
}

func TestAPIRateLimiterAddsLeaseAfterConcurrentConsumption(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 20, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	applyTestLease(bucket, 10, 100)
	if err := l.consume(context.Background(), bucket, 1); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := l.consume(context.Background(), bucket, 5); err != nil {
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

func TestAPIRateLimiterStartsGlobalBackoffAfterRefillError(t *testing.T) {
	clock := newTestRateLimitClock()
	var calls atomic.Int32
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		if calls.Add(1) == 1 {
			return nil, errors.New("PostgreSQL is unavailable")
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			CapacityUnits: 100,
		}}, nil
	})
	l.now = clock.Now
	bucket := newNonspecificBucket(testRateLimitID)
	applyTestLease(bucket, 2, 100)
	if err := l.consume(context.Background(), bucket, 1); err != nil {
		t.Fatalf("initial consume: %v", err)
	}
	waitForRateLimit(t, func() bool {
		return !rateLimiterRetryAfter(l).IsZero() && calls.Load() == 1
	})
	retryAt := rateLimiterRetryAfter(l)
	if want := clock.Now().Add(apiRateLimitRefillBackoffDuration); !retryAt.Equal(want) {
		t.Fatalf("retry after = %s, want %s", retryAt, want)
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("failed refill remained queued")
	}

	if err := l.consume(context.Background(), bucket, 1); err != nil {
		t.Fatalf("consume local capacity during backoff: %v", err)
	}
	if err := l.consume(context.Background(), bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume overdraft during backoff: got %v, want ErrAPICapacityExceeded", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("refill attempts during backoff = %d, want 1", calls.Load())
	}
	overdraftBucket := newWorkspaceBucket("222222222222")
	_, refill, _ := overdraftBucket.consume(2, true)
	overdraftBucket.activateRefill(refill, 0, false, true)
	if err := l.consume(context.Background(), overdraftBucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("overdraft during global backoff: got %v, want ErrAPICapacityExceeded", err)
	}

	clock.Set(retryAt)
	if err := l.consume(context.Background(), bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume after backoff: got %v, want ErrAPICapacityExceeded", err)
	}
	waitForRateLimit(t, func() bool {
		return calls.Load() == 2 && rateLimiterRetryAfter(l).IsZero()
	})
}

func TestAPIRateLimiterBacksOffAfterInvalidLeaseResults(t *testing.T) {
	for _, test := range []struct {
		name    string
		results func(apiRateLimitLeaseRequest) []apiRateLimitLeaseResult
	}{
		{
			name: "incomplete",
			results: func(apiRateLimitLeaseRequest) []apiRateLimitLeaseResult {
				return nil
			},
		},
		{
			name: "duplicate",
			results: func(request apiRateLimitLeaseRequest) []apiRateLimitLeaseResult {
				result := apiRateLimitLeaseResult{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, CapacityUnits: 100}
				return []apiRateLimitLeaseResult{result, result}
			},
		},
		{
			name: "unmatched subject",
			results: func(request apiRateLimitLeaseRequest) []apiRateLimitLeaseResult {
				return []apiRateLimitLeaseResult{{
					SubjectKind:   request.SubjectKind,
					SubjectID:     "222222222222",
					CapacityUnits: 100,
				}}
			},
		},
		{
			name: "invalid grant",
			results: func(request apiRateLimitLeaseRequest) []apiRateLimitLeaseResult {
				return []apiRateLimitLeaseResult{{
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
			l := newTestRateLimiter(t, func(_ context.Context, requests []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
				return test.results(requests[0]), nil
			})
			l.now = clock.Now
			bucket := newNonspecificBucket(testRateLimitID)
			_ = l.consume(context.Background(), bucket, 1)
			waitForRateLimit(t, func() bool {
				return !rateLimiterRetryAfter(l).IsZero()
			})
		})
	}
}

func TestAPIRateLimiterGlobalBackoffBlocksQueuedAndConcurrentRefills(t *testing.T) {
	clock := newTestRateLimitClock()
	var calls atomic.Int32
	l := newTestRateLimiter(t, func(_ context.Context, _ []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		calls.Add(1)
		return nil, nil
	})
	l.now = clock.Now
	l.refillRetryAfter.Store(clock.Now().Add(apiRateLimitRefillBackoffDuration).UnixNano())

	queued := newNonspecificBucket(testRateLimitID)
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

	bucket := newWorkspaceBucket("222222222222")
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			if err := l.consume(context.Background(), bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
				t.Errorf("consume during global backoff: got %v, want ErrAPICapacityExceeded", err)
			}
		})
	}
	wg.Wait()
	_, _, _, refillQueued, _ := bucketSnapshot(bucket)
	if refillQueued {
		t.Fatal("global backoff queued a refill")
	}
}

func TestAPIRateLimiterLimitsConcurrentOverdraft(t *testing.T) {
	bucket := newNonspecificBucket(testRateLimitID)
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	var successful atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			satisfied, _, _ := bucket.consume(1, true)
			if satisfied {
				successful.Add(1)
			}
		})
	}
	wg.Wait()
	available, _, _, _, _ := bucketSnapshot(bucket)
	if successful.Load() != apiRateLimitOverdraftLimit || available != -apiRateLimitOverdraftLimit {
		t.Fatalf("successful=%d available=%d, want %d and %d", successful.Load(), available, apiRateLimitOverdraftLimit, -apiRateLimitOverdraftLimit)
	}
}

func TestAPIRateLimiterDisabledBucketCompletesInFlightRefill(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		return []apiRateLimitLeaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  10,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- l.consume(context.Background(), bucket, 1) }()
	<-started
	bucket.disable()
	if err := <-result; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("disabled waiter: %v", err)
	}
	close(release)
	waitForRateLimit(t, func() bool {
		available, _, _, queued, disabled := bucketSnapshot(bucket)
		return disabled && !queued && available == 0
	})
}

func TestAPIRateLimiterQueueFullDoesNotEnableOverdraft(t *testing.T) {
	l := &rateLimiter{
		now:         time.Now,
		refillQueue: make(chan *rateLimitRefill, 1),
	}
	l.refillQueue <- &rateLimitRefill{}
	bucket := newNonspecificBucket(testRateLimitID)

	if err := l.consume(context.Background(), bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume with a full queue: got %v, want ErrAPICapacityExceeded", err)
	}
	if err := l.consume(context.Background(), bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("overdraft after a full queue: got %v, want ErrAPICapacityExceeded", err)
	}
	if !rateLimiterRetryAfter(l).IsZero() {
		t.Fatal("full queue started backoff")
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("full queue left refill queued")
	}
}

func TestAPIRateLimiterShutdownDiscardsLateResultWithoutBackoff(t *testing.T) {
	started := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-ctx.Done()
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 1, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	result := make(chan error, 1)
	go func() { result <- l.consume(context.Background(), bucket, 1) }()
	<-started
	l.Close()
	if err := <-result; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("waiter after shutdown: %v", err)
	}

	if !rateLimiterRetryAfter(l).IsZero() {
		t.Fatal("shutdown started global backoff")
	}
	available, _, _, queued, _ := bucketSnapshot(bucket)
	if queued || available != 0 {
		t.Fatalf("state after shutdown: queued=%t available=%d", queued, available)
	}
}

func TestAPIRateLimiterShutdownUnblocksBufferedWaiter(t *testing.T) {
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, _ []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	firstBucket := newNonspecificBucket(testRateLimitID)
	secondBucket := newWorkspaceBucket("222222222222")
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- limiter.consume(context.Background(), firstBucket, 1) }()
	<-started
	go func() { second <- limiter.consume(context.Background(), secondBucket, 1) }()
	waitForRateLimit(t, func() bool { return refillPendingCost(secondBucket) == 1 })

	limiter.Close()
	if err := <-first; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("in-flight waiter after shutdown: %v", err)
	}
	if err := <-second; !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("buffered waiter after shutdown: %v", err)
	}
}

func TestAPIRateLimiterShutdownFinishesCollectedRefill(t *testing.T) {
	l := &rateLimiter{}
	l.close.ctx, l.close.cancel = context.WithCancel(context.Background())
	l.close.cancel()

	bucket := newNonspecificBucket(testRateLimitID)
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
