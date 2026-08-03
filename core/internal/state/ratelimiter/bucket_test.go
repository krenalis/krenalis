// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestNewBucketRejectsInvalidConfiguration(t *testing.T) {
	if operation := os.Getenv("KRENALIS_RATE_LIMIT_INVALID_BUCKET"); operation != "" {
		switch operation {
		case "zero-lease-size":
			new(Limiter).NewBucket(testSubjectKind, testRateLimitID, 0, 1)
		case "zero-maximum-units":
			new(Limiter).NewBucket(testSubjectKind, testRateLimitID, 1, 0)
		case "maximum-units-exceed-lease-size":
			new(Limiter).NewBucket(testSubjectKind, testRateLimitID, 1, 2)
		default:
			t.Fatalf("unknown invalid-bucket operation %q", operation)
		}
		t.Fatalf("%s returned after an invalid configuration", operation)
	}
	for _, test := range []struct {
		name      string
		operation string
	}{
		{name: "zero lease size", operation: "zero-lease-size"},
		{name: "zero maximum units", operation: "zero-maximum-units"},
		{name: "maximum units exceed lease size", operation: "maximum-units-exceed-lease-size"},
	} {
		t.Run(test.name, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestNewBucketRejectsInvalidConfiguration$")
			command.Env = append(os.Environ(), "KRENALIS_RATE_LIMIT_INVALID_BUCKET="+test.operation)
			err := command.Run()
			exitError, ok := err.(*exec.ExitError)
			if !ok || exitError.ExitCode() != 1 {
				t.Fatalf("expected invalid configuration to exit with status 1, got %v", err)
			}
		})
	}
}

func TestBucketRestoresLocalCapacity(t *testing.T) {
	bucket := newTestBucket()
	applyTestLease(bucket, 10, 10)

	if err := bucket.Consume(context.Background(), 4); err != nil {
		t.Fatalf("initial consumption: %v", err)
	}
	if err := bucket.Restore(3); err != nil {
		t.Fatalf("first restoration: %v", err)
	}
	if got := bucketAvailable(bucket); got != 9 {
		t.Fatalf("expected restored capacity 9, got %d", got)
	}
	if err := bucket.Restore(3); err != nil {
		t.Fatalf("second restoration: %v", err)
	}
	if got := bucketAvailable(bucket); got != 10 {
		t.Fatalf("expected capacity capped at 10, got %d", got)
	}
}

// TestBucketCapacityReductionRevokesLocalExcess verifies that reducing the
// local target clamps existing capacity without making it restorable.
func TestBucketCapacityReductionRevokesLocalExcess(t *testing.T) {
	bucket := newTestBucket()
	applyTestLease(bucket, 100, 100)

	bucket.mu.Lock()
	restorable := bucket.applyLeaseLocked(0, 10, 10)
	available := bucket.available
	bucket.mu.Unlock()

	if available != 10 {
		t.Fatalf("expected clamped local capacity 10, got %d", available)
	}
	if restorable != 0 {
		t.Fatalf("expected no restorable revoked capacity, got %d", restorable)
	}
}

// TestBucketAdaptiveTargetReductionPreservesLocalCapacity verifies that a lower
// adaptive target does not revoke capacity already held locally.
func TestBucketAdaptiveTargetReductionPreservesLocalCapacity(t *testing.T) {
	bucket := newTestBucket()
	applyTestLease(bucket, 100, 100)

	bucket.mu.Lock()
	restorable := bucket.applyLeaseLocked(0, 100, initialLeaseTarget)
	available := bucket.available
	target := bucket.localTarget
	bucket.mu.Unlock()

	if available != 100 || target != 100 {
		t.Fatalf("expected existing capacity and target to remain 100, got capacity %d and target %d", available, target)
	}
	if restorable != 0 {
		t.Fatalf("expected no restorable capacity, got %d", restorable)
	}
}

func TestBucketThresholdScalesWithTarget(t *testing.T) {
	for _, test := range []struct {
		target, threshold int
	}{
		{target: 100, threshold: 25},
		{target: 40, threshold: 10},
		{target: 10, threshold: 2},
		{target: 1, threshold: 1},
	} {
		t.Run(strconv.Itoa(test.target), func(t *testing.T) {
			bucket := newTestBucket()
			applyTestLease(bucket, 0, test.target)
			if bucket.refillThreshold != test.threshold {
				t.Fatalf("expected refill threshold %d, got %d", test.threshold, bucket.refillThreshold)
			}
		})
	}
}

func TestBucketStartsProactiveRefillRelativeToConsumption(t *testing.T) {
	requests := make(chan []leaseRequest, 1)
	limiter := newTestRateLimiter(t, func(_ context.Context, request []leaseRequest) ([]leaseResult, error) {
		requests <- request
		return []leaseResult{{
			SubjectKind:   request[0].SubjectKind,
			SubjectID:     request[0].SubjectID,
			GrantedUnits:  request[0].RequestedUnits,
			CapacityUnits: testLeaseSize,
		}}, nil
	})
	bucket := newTestBucket(limiter)
	applyTestLease(bucket, testLeaseSize, testLeaseSize)

	// The 40 remaining units are above the absolute threshold but insufficient
	// for another operation consuming the same units.
	if err := bucket.Consume(context.Background(), 60); err != nil {
		t.Fatalf("consume: %v", err)
	}
	select {
	case request := <-requests:
		if got := request[0].RequestedUnits; got != 60 {
			t.Fatalf("expected requested units 60, got %d", got)
		}
	case <-time.After(time.Second):
		t.Fatal("expected relative consumption to start a refill, got none")
	}
}

// TestBucketAdaptsLeaseTargetToDemand verifies that lease targets grow after
// recent demand, reset after slower demand, and include triggering operations.
func TestBucketAdaptsLeaseTargetToDemand(t *testing.T) {
	for _, test := range []struct {
		name          string
		leaseSize     int
		localTarget   int
		available     int
		growthGrantAt time.Time
		minimumUnits  int
		wantTarget    int
		wantRequested int
		wantRefill    bool
	}{
		{
			name:          "cold bucket",
			leaseSize:     100,
			minimumUnits:  1,
			wantTarget:    initialLeaseTarget + 1,
			wantRequested: initialLeaseTarget + 1,
			wantRefill:    true,
		},
		{
			name:          "recent demand doubles target",
			leaseSize:     100,
			localTarget:   10,
			growthGrantAt: time.Now(),
			wantTarget:    20,
			wantRequested: 20,
			wantRefill:    true,
		},
		{
			name:          "growth stops at lease size",
			leaseSize:     100,
			localTarget:   80,
			growthGrantAt: time.Now(),
			wantTarget:    100,
			wantRequested: 100,
			wantRefill:    true,
		},
		{
			name:          "large operation raises target",
			leaseSize:     20_000,
			minimumUnits:  15_000,
			wantTarget:    15_000 + initialLeaseTarget,
			wantRequested: 15_000 + initialLeaseTarget,
			wantRefill:    true,
		},
		{
			name:          "admission headroom stops at lease size",
			leaseSize:     20_000,
			minimumUnits:  19_995,
			wantTarget:    20_000,
			wantRequested: 20_000,
			wantRefill:    true,
		},
		{
			name:          "slow demand resets target",
			leaseSize:     100,
			localTarget:   100,
			growthGrantAt: time.Now().Add(-leaseGrowthWindow - time.Second),
			wantTarget:    initialLeaseTarget,
			wantRequested: initialLeaseTarget,
			wantRefill:    true,
		},
		{
			name:          "slow demand retains existing capacity without refill",
			leaseSize:     100,
			localTarget:   100,
			available:     24,
			growthGrantAt: time.Now().Add(-leaseGrowthWindow - time.Second),
			wantRefill:    false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			bucket := new(Limiter).NewBucket(testSubjectKind, testRateLimitID, test.leaseSize, test.leaseSize)
			bucket.mu.Lock()
			bucket.localTarget = test.localTarget
			bucket.available = test.available
			bucket.growthGrantAt = test.growthGrantAt
			refill := bucket.newRefillWithAdmissionLocked(test.minimumUnits)
			available := bucket.available
			bucket.mu.Unlock()

			if !test.wantRefill {
				if refill != nil {
					t.Fatalf("expected no refill, got target %d and request %d", refill.targetUnits, refill.request.RequestedUnits)
				}
				if available != test.available {
					t.Fatalf("expected existing capacity %d to remain available, got %d", test.available, available)
				}
				return
			}
			if refill == nil {
				t.Fatal("expected refill, got nil")
			}
			if refill.targetUnits != test.wantTarget {
				t.Fatalf("expected target %d, got %d", test.wantTarget, refill.targetUnits)
			}
			if refill.request.RequestedUnits != test.wantRequested {
				t.Fatalf("expected request %d, got %d", test.wantRequested, refill.request.RequestedUnits)
			}
		})
	}
}

// TestBucketColdRefillReservesAdmissionHeadroom verifies that a large
// triggering operation does not consume the entire admission budget when the
// lease size can also accommodate the baseline headroom.
func TestBucketColdRefillReservesAdmissionHeadroom(t *testing.T) {
	const operationUnits = 15_000
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
			GrantedUnits:  request[0].RequestedUnits,
			CapacityUnits: 20_000,
		}}, nil
	})
	bucket := limiter.NewBucket("events", testRateLimitID, 20_000, 20_000)
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- bucket.Consume(context.Background(), operationUnits) }()
	request := <-requests
	if requested := request[0].RequestedUnits; requested != operationUnits+initialLeaseTarget {
		t.Fatalf("expected requested units %d, got %d", operationUnits+initialLeaseTarget, requested)
	}
	go func() { second <- bucket.Consume(context.Background(), initialLeaseTarget) }()
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == operationUnits+initialLeaseTarget })
	close(release)
	if err := <-first; err != nil {
		t.Fatalf("first waiter: %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("headroom waiter: %v", err)
	}
}

func TestBucketAdmitsWaitersToPublishedRefill(t *testing.T) {
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
	setTestRecentLocalTarget(bucket, testLeaseSize)
	first := make(chan error, 1)
	second := make(chan error, 1)
	go func() { first <- bucket.Consume(context.Background(), 40) }()
	<-requests
	go func() { second <- bucket.Consume(context.Background(), 30) }()
	waitForRateLimit(t, func() bool { return pendingUnits(bucket) == 70 })
	close(release)
	if err := <-first; err != nil {
		t.Fatalf("first waiter: %v", err)
	}
	if err := <-second; err != nil {
		t.Fatalf("second waiter: %v", err)
	}
}

func TestConsumeReturnsInvalidUnits(t *testing.T) {
	bucket := newTestBucket()
	for _, units := range []int{0, testLeaseSize + 1} {
		if err := bucket.Consume(t.Context(), units); err == nil || isCapacityExceeded(err) || errors.Is(err, ErrLimiterUnavailable) {
			t.Fatalf("expected Consume(%d) to return a generic error, got %v", units, err)
		}
	}
}

func TestCalculateRetryAfter(t *testing.T) {
	for _, test := range []struct {
		name          string
		requiredUnits int
		result        leaseResult
		want          time.Duration
	}{
		{
			name:          "one unit",
			requiredUnits: 1,
			result:        leaseResult{CapacityUnits: 100, RatePerMinute: 60},
			want:          time.Second,
		},
		{
			name:          "fractional remainder",
			requiredUnits: 1,
			result:        leaseResult{CapacityUnits: 100, RatePerMinute: 60, RefillRemainder: 30_000_000},
			want:          500 * time.Millisecond,
		},
		{
			name:          "available capacity",
			requiredUnits: 10,
			result:        leaseResult{CapacityUnits: 100, AvailableUnits: 10, RatePerMinute: 60},
		},
		{
			name:          "cumulative demand above capacity",
			requiredUnits: 101,
			result:        leaseResult{CapacityUnits: 100, RatePerMinute: 60},
			want:          101 * time.Second,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := calculateRetryAfter(test.requiredUnits, test.result)
			if got != test.want {
				t.Fatalf("calculateRetryAfter() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestRestoreReturnsInvalidUnits(t *testing.T) {
	bucket := newTestBucket()
	for _, units := range []int{0, testLeaseSize + 1} {
		if err := bucket.Restore(units); err == nil {
			t.Fatalf("expected Restore(%d) to return an error, got nil", units)
		}
	}
}
