// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"
)

const (
	testSubjectKind SubjectKind = "test"
	testRateLimitID             = "111111111111"
	testLeaseSize               = 100
)

func isCapacityExceeded(err error) bool {
	_, ok := err.(CapacityExceededError)
	return ok
}

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
	if acquire != nil {
		limiter.acquire = func(ctx context.Context, requests []leaseRequest) ([]leaseResult, error) {
			results, err := acquire(ctx, requests)
			for i := range results {
				if results[i].CapacityUnits > 0 && results[i].RatePerMinute == 0 {
					results[i].RatePerMinute = 60
				}
			}
			return results, err
		}
	}
	limiter.restoreBatch = func(context.Context, []unusedCapacity) error { return nil }
	t.Cleanup(func() { limiter.Close(context.Background()) })
	return limiter
}

func testLeaseResult(granted, capacity int) leaseResult {
	return leaseResult{
		GrantedUnits:   granted,
		CapacityUnits:  capacity,
		RatePerMinute:  60,
		AvailableUnits: 0,
	}
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

func testUnusedCapacity(count int) []unusedCapacity {
	unused := make([]unusedCapacity, count)
	for i := range unused {
		unused[i] = unusedCapacity{
			SubjectKind: testSubjectKind,
			SubjectID:   fmt.Sprintf("subject-%d", i),
			Units:       1,
		}
	}
	return unused
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

// TestLimiterCloseRestoresUnusedCapacity verifies that Close restores unused
// capacity from reachable buckets.
func TestLimiterCloseRestoresUnusedCapacity(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	restored := make(chan []unusedCapacity, 1)
	limiter.restoreBatch = func(_ context.Context, unused []unusedCapacity) error {
		restored <- append([]unusedCapacity(nil), unused...)
		return nil
	}

	requests := limiter.NewBucket("requests", testRateLimitID, testLeaseSize, testLeaseSize)
	events := limiter.NewBucket("events", testRateLimitID, testLeaseSize, testLeaseSize)
	applyTestLease(requests, 40, 100)
	applyTestLease(events, 20, 100)
	if err := requests.Consume(t.Context(), 10); err != nil {
		t.Fatalf("consume requests: %v", err)
	}

	limiter.Close(t.Context())
	runtime.KeepAlive(requests)
	runtime.KeepAlive(events)
	unused := <-restored
	sort.Slice(unused, func(i, j int) bool { return unused[i].SubjectKind < unused[j].SubjectKind })
	want := []unusedCapacity{
		{SubjectKind: "events", SubjectID: testRateLimitID, Units: 20},
		{SubjectKind: "requests", SubjectID: testRateLimitID, Units: 30},
	}
	if !reflect.DeepEqual(unused, want) {
		t.Fatalf("expected restored capacity %#v, got %#v", want, unused)
	}
}

// TestCollectUnusedCapacityIncludesQueuedRestorations verifies that shutdown
// combines local capacity with excess that the background restorer has not yet
// attempted.
func TestCollectUnusedCapacityIncludesQueuedRestorations(t *testing.T) {
	limiter := new(Limiter)
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 10, 100)
	limiter.queueCapacityRestoration(testSubjectKind, testRateLimitID, 5)
	limiter.queueCapacityRestoration(testSubjectKind, testRateLimitID, 15)

	unused := limiter.collectUnusedCapacity()
	want := []unusedCapacity{{SubjectKind: testSubjectKind, SubjectID: testRateLimitID, Units: 30}}
	if !reflect.DeepEqual(unused, want) {
		t.Fatalf("expected unused capacity %#v, got %#v", want, unused)
	}
	runtime.KeepAlive(bucket)
}

// TestLimiterCloseIsIdempotent verifies that only the first Close restores
// unused capacity.
func TestLimiterCloseIsIdempotent(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	var calls atomic.Int32
	limiter.restoreBatch = func(context.Context, []unusedCapacity) error {
		calls.Add(1)
		return nil
	}
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 10, 10)

	limiter.Close(t.Context())
	limiter.Close(t.Context())
	runtime.KeepAlive(bucket)
	if got := calls.Load(); got != 1 {
		t.Fatalf("expected one capacity restore, got %d", got)
	}
}

// TestLimiterCloseSkipsCapacityRestoreWhenCanceled verifies that a canceled
// Close does not begin restoring capacity.
func TestLimiterCloseSkipsCapacityRestoreWhenCanceled(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	called := atomic.Bool{}
	limiter.restoreBatch = func(context.Context, []unusedCapacity) error {
		called.Store(true)
		return nil
	}
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 10, 10)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	limiter.Close(ctx)
	if called.Load() {
		t.Fatal("expected canceled Close to skip restoring capacity, got a restore attempt")
	}
}

// TestCompactBucketsStopsDuringShutdown verifies that shutdown leaves the
// reference slice unchanged.
func TestCompactBucketsStopsDuringShutdown(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newTestBucket(limiter)
	limiter.buckets.Lock()
	limiter.buckets.refs = append(limiter.buckets.refs, weak.Pointer[Bucket]{})
	limiter.buckets.Unlock()

	limiter.close.cancel()
	limiter.compactBuckets()
	limiter.buckets.Lock()
	refs := limiter.buckets.refs
	limiter.buckets.Unlock()
	runtime.KeepAlive(bucket)
	if len(refs) != 2 || refs[0].Value() != bucket || refs[1].Value() != nil {
		t.Fatal("expected interrupted compaction to leave references unchanged")
	}
}

// TestCompactBucketsRemovesUnavailableReferences verifies that compaction
// removes unavailable weak references and retains reachable buckets.
func TestCompactBucketsRemovesUnavailableReferences(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	bucket := newTestBucket(limiter)
	limiter.buckets.Lock()
	limiter.buckets.refs = append(limiter.buckets.refs, weak.Pointer[Bucket]{})
	limiter.buckets.Unlock()

	limiter.compactBuckets()
	limiter.buckets.Lock()
	refs := limiter.buckets.refs
	limiter.buckets.Unlock()
	runtime.KeepAlive(bucket)
	if len(refs) != 1 || refs[0].Value() != bucket {
		t.Fatal("expected compaction to retain the reachable bucket")
	}
}

// TestRestoreUnusedCapacityBatchesRequests verifies batching and bounded
// concurrency when restoring unused capacity.
func TestRestoreUnusedCapacityBatchesRequests(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	unused := testUnusedCapacity(maxRestoreBatchSize*maxRestoreWorkers + 1)
	started := make(chan struct{}, maxRestoreWorkers)
	release := make(chan struct{})
	var mu sync.Mutex
	var batches [][]unusedCapacity
	active := 0
	maxActive := 0
	limiter.restoreBatch = func(_ context.Context, batch []unusedCapacity) error {
		mu.Lock()
		batches = append(batches, append([]unusedCapacity(nil), batch...))
		active++
		maxActive = max(maxActive, active)
		mu.Unlock()
		started <- struct{}{}
		<-release
		mu.Lock()
		active--
		mu.Unlock()
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- limiter.restoreUnusedCapacity(t.Context(), unused) }()
	for range maxRestoreWorkers {
		<-started
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatalf("restore unused capacity: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if maxActive != maxRestoreWorkers {
		t.Fatalf("expected %d concurrent batches, got %d", maxRestoreWorkers, maxActive)
	}
	if got, want := len(batches), maxRestoreWorkers+1; got != want {
		t.Fatalf("expected %d batches, got %d", want, got)
	}
	seen := make(map[string]bool, len(unused))
	for _, batch := range batches {
		if len(batch) > maxRestoreBatchSize {
			t.Fatalf("expected batch size at most %d, got %d", maxRestoreBatchSize, len(batch))
		}
		for _, capacity := range batch {
			if seen[capacity.SubjectID] {
				t.Fatalf("expected subject %q once, got it more than once", capacity.SubjectID)
			}
			seen[capacity.SubjectID] = true
		}
	}
	if got, want := len(seen), len(unused); got != want {
		t.Fatalf("expected %d restored subjects, got %d", want, got)
	}
}

// TestRestoreUnusedCapacityContinuesAfterBatchError verifies that one failed
// batch does not prevent later batches from being attempted.
func TestRestoreUnusedCapacityContinuesAfterBatchError(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	unused := testUnusedCapacity(maxRestoreBatchSize*maxRestoreWorkers + 1)
	var calls atomic.Int32
	limiter.restoreBatch = func(context.Context, []unusedCapacity) error {
		if calls.Add(1) == 1 {
			return errors.New("restore failed")
		}
		return nil
	}

	err := limiter.restoreUnusedCapacity(t.Context(), unused)
	if err == nil {
		t.Fatal("expected restore to return an error, got nil")
	}
	if got, want := int(calls.Load()), maxRestoreWorkers+1; got != want {
		t.Fatalf("expected %d restore attempts, got %d", want, got)
	}
}

// TestRestoreUnusedCapacityHonorsCancellation verifies that cancellation stops
// new restore batches.
func TestRestoreUnusedCapacityHonorsCancellation(t *testing.T) {
	limiter := newTestRateLimiter(t, nil)
	unused := testUnusedCapacity(maxRestoreBatchSize*maxRestoreWorkers + 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{}, maxRestoreWorkers)
	var calls atomic.Int32
	limiter.restoreBatch = func(ctx context.Context, _ []unusedCapacity) error {
		calls.Add(1)
		started <- struct{}{}
		<-ctx.Done()
		return ctx.Err()
	}

	done := make(chan error, 1)
	go func() { done <- limiter.restoreUnusedCapacity(ctx, unused) }()
	for range maxRestoreWorkers {
		<-started
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if got := calls.Load(); got != maxRestoreWorkers {
		t.Fatalf("expected %d restore attempts before cancellation, got %d", maxRestoreWorkers, got)
	}
}

// TestLimiterCloseWaitsForProactiveRefill verifies that Close waits for an
// in-progress acquisition even when its context is canceled.
func TestLimiterCloseWaitsForProactiveRefill(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	restored := atomic.Int32{}
	limiter := newTestRateLimiter(t, func(context.Context, []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-release
		return nil, context.Canceled
	})
	limiter.restoreBatch = func(context.Context, []unusedCapacity) error {
		restored.Add(1)
		return nil
	}
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 100, 100)
	if err := bucket.Consume(context.Background(), 80); err != nil {
		t.Fatalf("consume that starts proactive refill: %v", err)
	}
	<-started
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	closed := make(chan struct{})
	go func() {
		limiter.Close(ctx)
		close(closed)
	}()
	waitForRateLimit(t, limiter.close.Load)
	select {
	case <-closed:
		t.Fatal("expected Close to wait for the acquisition, got an early return")
	default:
	}
	close(release)
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for Close")
	}
	if backoff := limiter.backoff.Load(); backoff != nil {
		t.Fatalf("expected shutdown backoff nil, got %v", backoff)
	}
	if got := bucketAvailable(bucket); got != 20 {
		t.Fatalf("expected capacity after shutdown 20, got %d", got)
	}
	if got := restored.Load(); got != 0 {
		t.Fatalf("expected canceled Close not to retry restoring capacity, got %d attempts", got)
	}
}

// TestLimiterConsumesAndRefills verifies local consumption followed by refill.
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
		t.Fatalf("expected available capacity 0, got %d", got)
	}
}

// TestLimiterFullBucketChecksCapacityAboveKnownTarget verifies that an
// operation larger than a full bucket's known target starts a refill and waits
// for the authoritative capacity result.
func TestLimiterFullBucketChecksCapacityAboveKnownTarget(t *testing.T) {
	requests := make(chan leaseRequest, 1)
	limiter := newTestRateLimiter(t, func(_ context.Context, batch []leaseRequest) ([]leaseResult, error) {
		request := batch[0]
		requests <- request
		return []leaseResult{{
			SubjectKind:    request.SubjectKind,
			SubjectID:      request.SubjectID,
			CapacityUnits:  1_000,
			RatePerMinute:  1_000,
			AvailableUnits: 0,
		}}, nil
	})
	bucket := limiter.NewBucket("events", testRateLimitID, 20_000, 20_000)
	applyTestLease(bucket, 1_000, 1_000)

	err := bucket.Consume(context.Background(), 2_000)
	if !isCapacityExceeded(err) {
		t.Fatalf("expected capacity exceeded, got %v", err)
	}
	request := <-requests
	if request.RequestedUnits != 20_000 {
		t.Fatalf("expected refill request of 20000 units, got %d", request.RequestedUnits)
	}
}

// TestLimiterRestoresCapacityThatNoLongerFitsLocally verifies that capacity is
// not lost when Restore and a proactive refill both fill the same local space.
func TestLimiterRestoresCapacityThatNoLongerFitsLocally(t *testing.T) {
	for _, restoreBeforeGrant := range []bool{true, false} {
		name := "after grant"
		if restoreBeforeGrant {
			name = "before grant"
		}
		t.Run(name, func(t *testing.T) {
			requests := make(chan leaseRequest, 1)
			release := make(chan struct{})
			restored := make(chan []unusedCapacity, 2)
			var authoritative atomic.Int64
			authoritative.Store(900)

			limiter := newTestRateLimiter(t, func(_ context.Context, batch []leaseRequest) ([]leaseResult, error) {
				request := batch[0]
				requests <- request
				<-release
				remaining := authoritative.Add(int64(-request.RequestedUnits))
				return []leaseResult{{
					SubjectKind:    request.SubjectKind,
					SubjectID:      request.SubjectID,
					GrantedUnits:   request.RequestedUnits,
					CapacityUnits:  1000,
					AvailableUnits: int(remaining),
					RatePerMinute:  60,
				}}, nil
			})
			limiter.restoreBatch = func(_ context.Context, batch []unusedCapacity) error {
				for _, capacity := range batch {
					authoritative.Add(int64(capacity.Units))
				}
				restored <- append([]unusedCapacity(nil), batch...)
				return nil
			}
			bucket := newTestBucket(limiter)
			applyTestLease(bucket, 100, 1000)

			if err := bucket.Consume(context.Background(), 60); err != nil {
				t.Fatalf("consume: %v", err)
			}
			request := <-requests
			if request.RequestedUnits != 60 {
				t.Fatalf("expected refill request of 60 units, got %d", request.RequestedUnits)
			}
			if restoreBeforeGrant {
				if err := bucket.Restore(60); err != nil {
					t.Fatalf("restore before grant: %v", err)
				}
				close(release)
			} else {
				close(release)
				waitForRateLimit(t, func() bool {
					bucket.mu.Lock()
					defer bucket.mu.Unlock()
					return bucket.refill == nil
				})
				if err := bucket.Restore(60); err != nil {
					t.Fatalf("restore after grant: %v", err)
				}
			}

			select {
			case batch := <-restored:
				want := []unusedCapacity{{SubjectKind: testSubjectKind, SubjectID: testRateLimitID, Units: 60}}
				if !reflect.DeepEqual(batch, want) {
					t.Fatalf("expected excess restoration %#v, got %#v", want, batch)
				}
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for excess capacity restoration")
			}
			if got := bucketAvailable(bucket); got != 100 {
				t.Fatalf("expected local capacity 100, got %d", got)
			}
			if got := authoritative.Load(); got != 900 {
				t.Fatalf("expected authoritative capacity 900, got %d", got)
			}
			runtime.KeepAlive(bucket)
		})
	}
}

// TestLimiterCanceledContextConsumesLocalCapacity verifies that cancellation
// does not prevent immediate local consumption.
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
		t.Fatalf("expected available capacity 9, got %d", got)
	}
}

// TestLimiterCanceledContextCancelsWaiterButStartsRefill verifies that a
// canceled waiter does not cancel its published refill.
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
		t.Fatalf("expected pending units after cancellation 0, got %d", got)
	}
	close(release)
}

// TestLimiterCallerDeadlineTakesPrecedence verifies that the caller deadline
// takes precedence over the internal wait timeout.
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
			t.Fatalf("expected wait error context deadline exceeded, got %v", err)
		}
		bucket.rejectRefill(refill, ErrLimiterUnavailable)
	}
}

// TestLimiterKeepsPositiveCapacityForSmallerRequest verifies that a failed
// larger request does not consume local capacity needed by a smaller request.
func TestLimiterKeepsPositiveCapacityForSmallerRequest(t *testing.T) {
	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		return []leaseResult{{SubjectKind: requests[0].SubjectKind, SubjectID: requests[0].SubjectID, CapacityUnits: 100}}, nil
	})
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, 90, 100)
	if err := bucket.Consume(context.Background(), 100); !isCapacityExceeded(err) {
		t.Fatalf("expected large request capacity error, got %v", err)
	}
	if err := bucket.Consume(context.Background(), 90); err != nil {
		t.Fatalf("expected smaller request to use positive local capacity, got %v", err)
	}
}

// TestLimiterAdmissionBudgetIncludesTriggeringOperation verifies that a refill
// triggered by an operation requests enough units to include that operation in
// its admission budget, even when positive local capacity would otherwise make
// the refill request smaller.
func TestLimiterAdmissionBudgetIncludesTriggeringOperation(t *testing.T) {
	const (
		leaseSize      = 20_000
		localCapacity  = 11_000
		operationUnits = 15_000
	)
	requests := make(chan leaseRequest, 1)
	limiter := newTestRateLimiter(t, func(_ context.Context, batch []leaseRequest) ([]leaseResult, error) {
		request := batch[0]
		requests <- request
		return []leaseResult{{
			SubjectKind:   request.SubjectKind,
			SubjectID:     request.SubjectID,
			GrantedUnits:  request.RequestedUnits,
			CapacityUnits: leaseSize,
		}}, nil
	})
	bucket := limiter.NewBucket(testSubjectKind, testRateLimitID, leaseSize, leaseSize)
	applyTestLease(bucket, localCapacity, leaseSize)

	if err := bucket.Consume(context.Background(), operationUnits); err != nil {
		t.Fatalf("consume: %v", err)
	}
	request := <-requests
	if request.RequestedUnits != operationUnits {
		t.Fatalf("expected refill request for %d units, got %d", operationUnits, request.RequestedUnits)
	}
	if available := bucketAvailable(bucket); available != leaseSize-operationUnits {
		t.Fatalf("expected %d locally available units, got %d", leaseSize-operationUnits, available)
	}
}

// TestLimiterServesOnlySatisfiableFIFOPrefix verifies that a partial grant
// serves only the satisfiable FIFO prefix.
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
	if err := <-second; !isCapacityExceeded(err) {
		t.Fatalf("second waiter: %v", err)
	} else {
		capacityErr, ok := err.(CapacityExceededError)
		if !ok || capacityErr.RetryAfter != 10*time.Second {
			t.Fatalf("second waiter retry-after: %#v", err)
		}
	}
	if err := <-third; !isCapacityExceeded(err) {
		t.Fatalf("third waiter: %v", err)
	} else {
		capacityErr, ok := err.(CapacityExceededError)
		if !ok || capacityErr.RetryAfter != 30*time.Second {
			t.Fatalf("third waiter retry-after: %#v", err)
		}
	}
}

func TestLimiterOmitsRetryAfterAboveCapacity(t *testing.T) {
	bucket := newTestBucket()
	bucket.mu.Lock()
	refill := bucket.newRefillLocked()
	waiter := bucket.admitWaiterLocked(refill, 20)
	bucket.mu.Unlock()

	bucket.completeRefill(refill, leaseResult{CapacityUnits: 10, RatePerMinute: 60})
	if err, ok := waiter.err.(CapacityExceededError); !ok || err.RetryAfter != 0 {
		t.Fatalf("capacity error for request above maximum capacity: %#v", waiter.err)
	}
}

func TestLimiterRejectsInvalidRetryMetadata(t *testing.T) {
	for _, test := range []struct {
		name   string
		result leaseResult
	}{
		{name: "negative available units", result: leaseResult{CapacityUnits: 1, AvailableUnits: -1, RatePerMinute: 60}},
		{name: "available units above capacity", result: leaseResult{CapacityUnits: 1, AvailableUnits: 2, RatePerMinute: 60}},
		{name: "zero rate", result: leaseResult{CapacityUnits: 1}},
		{name: "invalid remainder", result: leaseResult{CapacityUnits: 1, RatePerMinute: 60, RefillRemainder: microsecondsPerMinute}},
	} {
		t.Run(test.name, func(t *testing.T) {
			limiter := newTestRateLimiter(t, nil)
			limiter.acquire = func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
				result := test.result
				result.SubjectKind = requests[0].SubjectKind
				result.SubjectID = requests[0].SubjectID
				return []leaseResult{result}, nil
			}
			if err := newTestBucket(limiter).Consume(context.Background(), 1); err == nil || isCapacityExceeded(err) || errors.Is(err, ErrLimiterUnavailable) {
				t.Fatalf("expected generic invalid-result error, got %v", err)
			}
		})
	}
}

// TestLimiterIsolatesMissingSubject verifies that a missing-subject sentinel
// rejects only its refill and does not start global backoff.
func TestLimiterIsolatesMissingSubject(t *testing.T) {
	const missingSubjectID = "222222222222"

	limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		results := make([]leaseResult, len(requests))
		for i, request := range requests {
			results[i].SubjectKind = request.SubjectKind
			results[i].SubjectID = request.SubjectID
			if request.SubjectID != missingSubjectID {
				results[i].GrantedUnits = 1
				results[i].CapacityUnits = 100
				results[i].AvailableUnits = 99
				results[i].RatePerMinute = 60
			}
		}
		return results, nil
	})
	missingBucket := limiter.NewBucket(testSubjectKind, missingSubjectID, testLeaseSize, testLeaseSize)
	validBucket := newTestBucket(limiter)
	buckets := []*Bucket{missingBucket, validBucket}
	refills := make([]*refill, len(buckets))
	waiters := make([]*waiter, len(buckets))
	for i, bucket := range buckets {
		bucket.mu.Lock()
		refills[i] = bucket.newRefillLocked()
		waiters[i] = bucket.admitWaiterLocked(refills[i], 1)
		bucket.mu.Unlock()
	}

	limiter.refill(refills)

	if !errors.Is(waiters[0].err, ErrLimiterUnavailable) {
		t.Fatalf("expected missing subject error %v, got %v", ErrLimiterUnavailable, waiters[0].err)
	}
	if waiters[1].err != nil {
		t.Fatalf("expected valid subject refill to succeed, got %v", waiters[1].err)
	}
	if backoff := limiter.backoff.Load(); backoff != nil {
		t.Fatalf("expected no global backoff, got %v", backoff.err)
	}
}

// TestLimiterCancellationReturnsAdmissionBudget verifies that cancellation
// makes reserved admission capacity available to later waiters.
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

// TestLimiterCancellationAndGrantResolveOnce verifies that concurrent
// cancellation and completion resolve a waiter once.
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
			bucket.completeRefill(refill, testLeaseResult(1, 100))
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
				t.Fatalf("expected available capacity 1 after cancellation, got %d", available)
			}
		case err == nil:
			if available != 0 {
				t.Fatalf("expected available capacity 0 after grant, got %d", available)
			}
		default:
			t.Fatalf("race result: %v", err)
		}
	}
}

// TestLimiterIgnoresStaleGenerationResult verifies that an older refill result
// cannot affect the current refill generation.
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
	bucket.completeRefill(first, testLeaseResult(100, 100))

	bucket.mu.Lock()
	current := bucket.refill
	available := bucket.available
	pending := waiter.element != nil
	bucket.mu.Unlock()
	if current != second || available != 0 || !pending {
		t.Fatalf("stale result changed current generation: current=%p available=%d pending=%t", current, available, pending)
	}

	bucket.completeRefill(second, testLeaseResult(1, 100))
	<-second.done
	if waiter.err != nil {
		t.Fatalf("second-generation waiter: %v", waiter.err)
	}
}

// TestLimiterWaitHasFiniteInternalTimeout verifies that an admitted waiter has
// a finite internal timeout.
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

// TestLimiterLeaseAcquisitionHasFiniteTimeout verifies that lease acquisition
// has a finite timeout.
func TestLimiterLeaseAcquisitionHasFiniteTimeout(t *testing.T) {
	previous := defaultAcquireTimeout
	defaultAcquireTimeout = 10 * time.Millisecond
	t.Cleanup(func() { defaultAcquireTimeout = previous })
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, _ []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	bucket := newTestBucket(limiter)
	result := make(chan error, 1)
	go func() { result <- bucket.Consume(context.Background(), 1) }()
	<-started

	select {
	case err := <-result:
		if !errors.Is(err, ErrLimiterUnavailable) {
			t.Fatalf("expected acquisition timeout error %v, got %v", ErrLimiterUnavailable, err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected lease acquisition to time out, got no timeout")
	}
	bucket.mu.Lock()
	available := bucket.available
	hasRefill := bucket.refill != nil
	bucket.mu.Unlock()
	if hasRefill || available != 0 {
		t.Fatalf("expected no refill and no available capacity after acquisition timeout, got pending=%t available=%d", hasRefill, available)
	}
}

// TestLimiterAppliesSuccessfulAcquisitionAfterDeadline verifies that a complete
// successful response remains authoritative after its context expires.
func TestLimiterAppliesSuccessfulAcquisitionAfterDeadline(t *testing.T) {
	previous := defaultAcquireTimeout
	defaultAcquireTimeout = 10 * time.Millisecond
	t.Cleanup(func() { defaultAcquireTimeout = previous })
	started := make(chan struct{})
	limiter := newTestRateLimiter(t, func(ctx context.Context, requests []leaseRequest) ([]leaseResult, error) {
		close(started)
		<-ctx.Done()
		return []leaseResult{{
			SubjectKind:   requests[0].SubjectKind,
			SubjectID:     requests[0].SubjectID,
			GrantedUnits:  1,
			CapacityUnits: 100,
		}}, nil
	})
	result := make(chan error, 1)
	go func() { result <- newTestBucket(limiter).Consume(context.Background(), 1) }()
	<-started

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expected successful consumption, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected successful acquisition after deadline, got no result")
	}
	if backoff := limiter.backoff.Load(); backoff != nil {
		t.Fatalf("expected no acquisition backoff, got %v", backoff.err)
	}
}

// TestLimiterExcessRestorationHasFiniteTimeout verifies that a stuck
// asynchronous restoration cannot block the restorer indefinitely.
func TestLimiterExcessRestorationHasFiniteTimeout(t *testing.T) {
	previous := defaultRestorationTimeout
	defaultRestorationTimeout = 10 * time.Millisecond
	t.Cleanup(func() { defaultRestorationTimeout = previous })
	started := make(chan struct{})
	completed := make(chan struct{})
	var calls atomic.Int32
	limiter := newTestRateLimiter(t, nil)
	limiter.restoreBatch = func(ctx context.Context, _ []unusedCapacity) error {
		if calls.Add(1) != 1 {
			return nil
		}
		close(started)
		<-ctx.Done()
		close(completed)
		return ctx.Err()
	}
	limiter.queueCapacityRestoration(testSubjectKind, testRateLimitID, 1)
	<-started

	select {
	case <-completed:
	case <-time.After(time.Second):
		t.Fatal("expected excess capacity restoration to time out")
	}
}

// TestLimiterQueuesOneRefill verifies that concurrent consumption shares one
// refill generation.
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
		t.Fatalf("expected batch to contain 1 request, got %d", len(request))
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
		t.Fatalf("expected available capacity 7, got %d", got)
	}
}

// TestLimiterBatchesSubjectKinds verifies that one acquisition batch can
// contain different subject kinds.
func TestLimiterBatchesSubjectKinds(t *testing.T) {
	limiter := &Limiter{queue: make(chan *refill, queueSize)}
	limiter.close.ctx, limiter.close.cancel = context.WithCancel(context.Background())
	t.Cleanup(limiter.close.cancel)
	var acquired []leaseRequest
	limiter.acquire = func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
		acquired = append(acquired, requests...)
		results := make([]leaseResult, len(requests))
		for i, request := range requests {
			results[i] = leaseResult{
				SubjectKind:   request.SubjectKind,
				SubjectID:     request.SubjectID,
				CapacityUnits: 100,
				RatePerMinute: 60,
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
		t.Fatal("refiller stopped while collecting refills")
	}
	if len(acquired) != 2 {
		t.Fatalf("expected batch to contain 2 requests, got %d", len(acquired))
	}
	kinds := map[SubjectKind]bool{}
	for _, request := range acquired {
		kinds[request.SubjectKind] = true
	}
	if !kinds["first"] || !kinds["second"] {
		t.Fatalf("expected subject kinds first and second, got %#v", kinds)
	}
}

// TestLimiterRejectsDuplicateSubjectsBeforeAcquisition verifies that malformed
// local batches fail without reaching the lease store.
func TestLimiterRejectsDuplicateSubjectsBeforeAcquisition(t *testing.T) {
	var acquisitions atomic.Int32
	limiter := newTestRateLimiter(t, func(context.Context, []leaseRequest) ([]leaseResult, error) {
		acquisitions.Add(1)
		return nil, nil
	})
	buckets := []*Bucket{newTestBucket(limiter), newTestBucket(limiter)}
	refills := make([]*refill, len(buckets))
	waiters := make([]*waiter, len(buckets))
	for i, bucket := range buckets {
		bucket.mu.Lock()
		refills[i] = bucket.newRefillLocked()
		waiters[i] = bucket.admitWaiterLocked(refills[i], 1)
		bucket.mu.Unlock()
	}

	limiter.refill(refills)

	if got := acquisitions.Load(); got != 0 {
		t.Fatalf("expected no lease acquisitions, got %d", got)
	}
	for i, waiter := range waiters {
		if waiter.err == nil {
			t.Fatalf("expected waiter %d to receive a duplicate-subject error, got nil", i)
		}
	}
	if backoff := limiter.backoff.Load(); backoff == nil {
		t.Fatal("expected duplicate-subject backoff, got no backoff")
	}
}

// TestLimiterAddsLeaseAfterConcurrentConsumption verifies that a lease is
// added to capacity remaining after concurrent consumption.
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
		t.Fatalf("expected available capacity 24, got %d", got)
	}
}

// TestLimiterRejectsInvalidLeaseResults verifies that invalid acquisition
// results fail the batch and start internal-error backoff.
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
		{name: "grant plus available exceeds capacity", results: func(request leaseRequest) []leaseResult {
			return []leaseResult{{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, GrantedUnits: 1, CapacityUnits: 1, AvailableUnits: 1}}
		}},
		{name: "zero capacity with nonzero rate", results: func(request leaseRequest) []leaseResult {
			return []leaseResult{{SubjectKind: request.SubjectKind, SubjectID: request.SubjectID, RatePerMinute: 60}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			limiter := newTestRateLimiter(t, func(_ context.Context, requests []leaseRequest) ([]leaseResult, error) {
				return test.results(requests[0]), nil
			})
			bucket := newTestBucket(limiter)
			if err := bucket.Consume(context.Background(), 1); err == nil || isCapacityExceeded(err) || errors.Is(err, ErrLimiterUnavailable) {
				t.Fatalf("expected invalid lease result to return a generic internal error, got %v", err)
			}
			if backoff := limiter.backoff.Load(); backoff == nil {
				t.Fatal("expected invalid lease result to start backoff, got no backoff")
			}
			if err := bucket.Consume(context.Background(), 1); err == nil || isCapacityExceeded(err) || errors.Is(err, ErrLimiterUnavailable) {
				t.Fatalf("expected invalid-response backoff to return a generic internal error, got %v", err)
			}
		})
	}
}

// TestLimiterRejectsWhenQueueIsFull verifies rejection and metrics when the
// refill queue is full.
func TestLimiterRejectsWhenQueueIsFull(t *testing.T) {
	queueFull := new(testCounter)
	limiter := &Limiter{queue: make(chan *refill, 1), metrics: Metrics{QueueFull: queueFull}}
	limiter.queue <- new(refill)
	if err := newTestBucket(limiter).Consume(context.Background(), 1); !errors.Is(err, ErrLimiterUnavailable) {
		t.Fatalf("consume with full queue: %v", err)
	}
	if got := queueFull.calls.Load(); got != 1 {
		t.Fatalf("expected queue-full metric 1, got %d", got)
	}
}

// TestLimiterRejectsRefillsDuringBackoff verifies that backoff prevents new
// lease acquisitions.
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
		t.Fatalf("expected acquisition calls 1, got %d", got)
	}
}
