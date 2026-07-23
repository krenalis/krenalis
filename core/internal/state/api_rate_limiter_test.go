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
		acquireLeases: acquire,
		now:           time.Now,
		refillQueue:   make(chan *rateLimitBucket, apiRateLimitQueueSize),
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
	return bucket.available, bucket.target, bucket.threshold, bucket.refillQueued, bucket.disabled
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
	bucket.applyLease(10, 10)
	bucket.disable()

	exceeded, queueRefill := bucket.consume(1, true)
	if !exceeded || queueRefill {
		t.Fatalf("disabled bucket consume = exceeded:%t queue:%t, want true:false", exceeded, queueRefill)
	}
	if _, ok := bucket.refillRequest(); ok {
		t.Fatal("disabled bucket requested a refill")
	}
	bucket.applyLease(10, 10)
	available, _, _, _, disabled := bucketSnapshot(bucket)
	if available != 0 || !disabled {
		t.Fatalf("disabled bucket state = available:%d disabled:%t, want 0:true", available, disabled)
	}
}

// TestIngestionRateLimitBucketAllowsInitialOverdraft verifies that a first
// single-event request can queue its initial refill without being rejected.
func TestIngestionRateLimitBucketAllowsInitialOverdraft(t *testing.T) {
	bucket := newIngestionBucket("222222222222")

	exceeded, queueRefill := bucket.consume(1, true)
	if exceeded || !queueRefill {
		t.Fatalf("initial ingestion bucket consume = exceeded:%t queue:%t, want false:true", exceeded, queueRefill)
	}
	bucket.finishRefill()
	exceeded, queueRefill = bucket.consume(1, true)
	if !exceeded || !queueRefill {
		t.Fatalf("second ingestion bucket consume without a refill = exceeded:%t queue:%t, want true:true", exceeded, queueRefill)
	}
}

func TestRateLimitBucketAllowsOverdraftOnlyWithPendingRefill(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	exceeded, queueRefill := bucket.consume(1, true)
	if !exceeded || !queueRefill {
		t.Fatalf("cold bucket consume = exceeded:%t queue:%t, want true:true", exceeded, queueRefill)
	}
	exceeded, queueRefill = bucket.consume(1, true)
	if exceeded || queueRefill {
		t.Fatalf("pending refill overdraft = exceeded:%t queue:%t, want false:false", exceeded, queueRefill)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != -1 {
		t.Fatalf("available capacity = %d, want -1", available)
	}
}

func TestRateLimitBucketDoesNotOverdraftHigherCostsOrClosedLimiter(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	bucket.mu.Lock()
	bucket.refillQueued = true
	bucket.mu.Unlock()

	exceeded, queueRefill := bucket.consume(2, true)
	if !exceeded || queueRefill {
		t.Fatalf("cost-2 overdraft = exceeded:%t queue:%t, want true:false", exceeded, queueRefill)
	}
	exceeded, queueRefill = bucket.consume(1, false)
	if !exceeded || queueRefill {
		t.Fatalf("closed limiter overdraft = exceeded:%t queue:%t, want true:false", exceeded, queueRefill)
	}
}

func TestRateLimitBucketLeaseRepaysOverdraft(t *testing.T) {
	bucket := newNonspecificBucket("111111111111")
	bucket.applyLease(0, 100)

	bucket.mu.Lock()
	bucket.available = -5
	bucket.mu.Unlock()
	bucket.applyLease(10, 100)
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 5 {
		t.Fatalf("full debt repayment = %d, want 5", available)
	}

	bucket.mu.Lock()
	bucket.available = -5
	bucket.mu.Unlock()
	bucket.applyLease(2, 100)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != -3 {
		t.Fatalf("partial debt repayment = %d, want -3", available)
	}
	bucket.applyLease(0, 100)
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
	bucket.refillQueued = true
	bucket.mu.Unlock()

	request, ok := bucket.refillRequest()
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
			bucket.applyLease(0, test.target)
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
	bucket.applyLease(1_000, 1_000)

	for range 2 {
		exceeded, queueRefill := bucket.consume(250, true)
		if exceeded || queueRefill {
			t.Fatalf("early consume = exceeded:%t queue:%t, want false:false", exceeded, queueRefill)
		}
	}
	exceeded, queueRefill := bucket.consume(250, true)
	if exceeded || !queueRefill {
		t.Fatalf("low-capacity consume = exceeded:%t queue:%t, want false:true", exceeded, queueRefill)
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
	organization.bucket.applyLease(2, 2)
	workspace.apiBucket.applyLease(2, 2)
	workspace.ingestionBucket.applyLease(2, 2)

	if err := organization.ConsumeRateLimitCapacity(2); err != nil {
		t.Fatalf("organization consume: %v", err)
	}
	if err := organization.ConsumeRateLimitCapacity(2); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("organization second consume: got %v, want ErrAPICapacityExceeded", err)
	}
	if err := workspace.ConsumeRateLimitCapacity(2); err != nil {
		t.Fatalf("workspace consume: %v", err)
	}
	if err := workspace.ConsumeIngestionRateLimitCapacity(2); err != nil {
		t.Fatalf("workspace ingestion consume: %v", err)
	}
}

func TestAPIRateLimiterValidatesCost(t *testing.T) {
	l := newTestRateLimiter(t, nil)
	bucket := newNonspecificBucket(testRateLimitID)
	for _, cost := range []int{-1, 0, 101} {
		if err := l.consume(bucket, cost); !errors.Is(err, ErrInvalidAPICost) {
			t.Fatalf("cost %d: got %v, want ErrInvalidAPICost", cost, err)
		}
	}
}

// TestIngestionRateLimiterValidatesEventCount verifies the supported batch size.
func TestIngestionRateLimiterValidatesEventCount(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newIngestionBucket(testRateLimitID)
	for _, count := range []int{-1, 0, ingestionRateLimitMaxCost + 1} {
		if err := limiter.consume(bucket, count); !errors.Is(err, ErrInvalidAPICost) {
			t.Fatalf("event count %d: got %v, want ErrInvalidAPICost", count, err)
		}
	}
	bucket.applyLease(ingestionRateLimitMaxCost, ingestionRateLimitMaxCost)
	if err := limiter.consume(bucket, ingestionRateLimitMaxCost); err != nil {
		t.Fatalf("maximum event count: %v", err)
	}
}

func TestAPIRateLimiterConsumesLocalCapacity(t *testing.T) {
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		return []apiRateLimitLeaseResult{{SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID, CapacityUnits: 10}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	bucket.applyLease(10, 10)

	if err := l.consume(bucket, 6); err != nil {
		t.Fatalf("consume: %v", err)
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 4 {
		t.Fatalf("available capacity = %d, want 4", available)
	}
	if err := l.consume(bucket, 5); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume exhausted capacity: got %v, want ErrAPICapacityExceeded", err)
	}
}

func TestAPIRateLimiterQueuesOneRefill(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		requests <- request
		<-release
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 10, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("first cold bucket consume: got %v, want ErrAPICapacityExceeded", err)
	}
	for range 2 {
		if err := l.consume(bucket, 1); err != nil {
			t.Fatalf("overdraft consume: %v", err)
		}
	}
	if request := <-requests; len(request) != 1 {
		t.Fatalf("batch contains %d requests, want 1", len(request))
	}
	close(release)
	waitForRateLimit(t, func() bool {
		available, _, _, queued, _ := bucketSnapshot(bucket)
		return !queued && available == 8
	})
}

func TestAPIRateLimiterBatchesOrganizationsAndWorkspaces(t *testing.T) {
	requests := make(chan []apiRateLimitLeaseRequest, 1)
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		requests <- request
		results := make([]apiRateLimitLeaseResult, len(request))
		for i, r := range request {
			results[i] = apiRateLimitLeaseResult{SubjectKind: r.SubjectKind, SubjectID: r.SubjectID, CapacityUnits: 100}
		}
		return results, nil
	})
	if err := l.consume(newNonspecificBucket("111111111111"), 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatal(err)
	}
	if err := l.consume(newWorkspaceBucket("222222222222"), 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatal(err)
	}
	if request := <-requests; len(request) != 2 {
		t.Fatalf("batch contains %d requests, want 2", len(request))
	}
}

func TestAPIRateLimiterAddsLeaseAfterConcurrentConsumption(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-release
		return []apiRateLimitLeaseResult{{
			SubjectKind: request[0].SubjectKind, SubjectID: request[0].SubjectID,
			GrantedUnits: 20, CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	bucket.applyLease(10, 100)
	if err := l.consume(bucket, 1); err != nil {
		t.Fatal(err)
	}
	<-started
	if err := l.consume(bucket, 5); err != nil {
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
	bucket.applyLease(2, 100)
	if err := l.consume(bucket, 1); err != nil {
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

	if err := l.consume(bucket, 1); err != nil {
		t.Fatalf("consume local capacity during backoff: %v", err)
	}
	if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume overdraft during backoff: got %v, want ErrAPICapacityExceeded", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("refill attempts during backoff = %d, want 1", calls.Load())
	}
	overdraftBucket := newWorkspaceBucket("222222222222")
	overdraftBucket.mu.Lock()
	overdraftBucket.refillQueued = true
	overdraftBucket.mu.Unlock()
	if err := l.consume(overdraftBucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("overdraft during global backoff: got %v, want ErrAPICapacityExceeded", err)
	}

	clock.Set(retryAt)
	if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
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
			_ = l.consume(bucket, 1)
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
	queued.mu.Lock()
	queued.refillQueued = true
	queued.mu.Unlock()
	l.refillQueue <- queued
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
			if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
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
	bucket.mu.Lock()
	bucket.refillQueued = true
	bucket.mu.Unlock()

	var successful atomic.Int32
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			exceeded, _ := bucket.consume(1, true)
			if !exceeded {
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
	l := newTestRateLimiter(t, func(_ context.Context, request []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-release
		return []apiRateLimitLeaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  10,
			CapacityUnits: 100,
		}}, nil
	})
	bucket := newNonspecificBucket(testRateLimitID)
	_ = l.consume(bucket, 1)
	<-started
	bucket.disable()
	close(release)
	waitForRateLimit(t, func() bool {
		available, _, _, queued, disabled := bucketSnapshot(bucket)
		return disabled && !queued && available == 0
	})
}

func TestAPIRateLimiterQueueFullDoesNotEnableOverdraft(t *testing.T) {
	l := &rateLimiter{
		now:         time.Now,
		refillQueue: make(chan *rateLimitBucket, 1),
	}
	l.refillQueue <- newNonspecificBucket("222222222222")
	bucket := newNonspecificBucket(testRateLimitID)

	if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
		t.Fatalf("consume with a full queue: got %v, want ErrAPICapacityExceeded", err)
	}
	if err := l.consume(bucket, 1); !errors.Is(err, ErrAPICapacityExceeded) {
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

func TestAPIRateLimiterShutdownDoesNotBackOff(t *testing.T) {
	started := make(chan struct{})
	l := newTestRateLimiter(t, func(ctx context.Context, _ []apiRateLimitLeaseRequest) ([]apiRateLimitLeaseResult, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	bucket := newNonspecificBucket(testRateLimitID)
	_ = l.consume(bucket, 1)
	<-started
	l.Close()

	if !rateLimiterRetryAfter(l).IsZero() {
		t.Fatal("shutdown started global backoff")
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("shutdown left refill queued")
	}
}

func TestAPIRateLimiterShutdownFinishesCollectedRefill(t *testing.T) {
	l := &rateLimiter{}
	l.close.ctx, l.close.cancel = context.WithCancel(context.Background())
	l.close.cancel()

	bucket := newNonspecificBucket(testRateLimitID)
	bucket.mu.Lock()
	bucket.refillQueued = true
	bucket.mu.Unlock()

	if l.collectAndRefillBatch(bucket) {
		t.Fatal("batch collection continued after shutdown")
	}
	_, _, _, queued, _ := bucketSnapshot(bucket)
	if queued {
		t.Fatal("shutdown left collected refill queued")
	}
}
