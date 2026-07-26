// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"strconv"
	"testing"
)

// TestNewBucketRejectsInvalidConfiguration verifies that a bucket has a
// positive lease size and a supported maximum cost.
func TestNewBucketRejectsInvalidConfiguration(t *testing.T) {
	for _, test := range []struct {
		name      string
		leaseSize int
		maxCost   int
	}{
		{name: "zero lease size", leaseSize: 0, maxCost: 1},
		{name: "zero maximum cost", leaseSize: 1, maxCost: 0},
		{name: "maximum cost exceeds lease size", leaseSize: 1, maxCost: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("NewBucket did not panic")
				}
			}()
			NewBucket(testSubjectKind, testRateLimitID, test.leaseSize, test.maxCost)
		})
	}
}

// TestClosedBucketDoesNotRequestOrApplyCapacity verifies that a removed
// subject kind and identifier cannot consume, refill, or receive capacity.
func TestClosedBucketDoesNotRequestOrApplyCapacity(t *testing.T) {
	bucket := newTestBucketFor("test", "222222222222")
	applyTestLease(bucket, 10, 10)
	bucket.Close()

	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill != nil || waiter != nil {
		t.Fatalf("closed bucket consumed or requested capacity: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	applyTestLease(bucket, 10, 10)
	state := bucketSnapshot(bucket)
	if state.available != 0 || !state.closed {
		t.Fatalf("closed bucket state = available:%d closed:%t, want 0:true", state.available, state.closed)
	}
	bucket.Restore(1)
	state = bucketSnapshot(bucket)
	if state.available != 0 {
		t.Fatalf("restored capacity on a closed bucket = %d, want 0", state.available)
	}
}

// TestBucketRestoresLocalCapacity verifies that returned capacity is available
// immediately and remains capped at the local target.
func TestBucketRestoresLocalCapacity(t *testing.T) {
	bucket := newTestBucketFor("test", "222222222222")
	applyTestLease(bucket, 10, 10)

	satisfied, _, _ := bucket.consume(4, true)
	if !satisfied {
		t.Fatal("initial consumption was not satisfied")
	}
	bucket.Restore(3)
	state := bucketSnapshot(bucket)
	if state.available != 9 {
		t.Fatalf("restored capacity = %d, want 9", state.available)
	}

	bucket.Restore(3)
	state = bucketSnapshot(bucket)
	if state.available != 10 {
		t.Fatalf("capacity above the local target = %d, want 10", state.available)
	}
	bucket.Restore(0)
	state = bucketSnapshot(bucket)
	if state.available != 10 {
		t.Fatalf("zero restoration changed capacity to %d, want 10", state.available)
	}
	bucket.Restore(bucket.maxCost + 1)
	state = bucketSnapshot(bucket)
	if state.available != 10 {
		t.Fatalf("invalid restoration changed capacity to %d, want 10", state.available)
	}
}

// TestEmptyBucketWaitsForInitialRefill verifies that a request waits for its
// initial refill instead of using capacity that has not been acquired yet.
func TestEmptyBucketWaitsForInitialRefill(t *testing.T) {
	bucket := newTestLargeBucket()

	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("initial consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	if waiter := bucket.activateRefill(refill, 1, true, true); waiter == nil {
		t.Fatal("initial request was not admitted as a waiter")
	}
	if state := bucketSnapshot(bucket); state.available != 0 {
		t.Fatalf("available capacity = %d, want 0", state.available)
	}
	bucket.rejectRefill(refill)
}

// TestBucketAdmitsCostOneToPendingRefill verifies that a cost-one request
// waits for an active refill when local capacity is exhausted.
func TestBucketAdmitsCostOneToPendingRefill(t *testing.T) {
	bucket := newTestBucket()
	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("cold bucket consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	if waiter := bucket.activateRefill(refill, 0, false, true); waiter != nil {
		t.Fatal("unexpected waiter")
	}
	satisfied, queued, waiter := bucket.consume(1, true)
	if satisfied || queued != nil || waiter == nil {
		t.Fatalf("pending refill waiter: satisfied=%t refill=%p waiter=%p", satisfied, queued, waiter)
	}
	if state := bucketSnapshot(bucket); state.available != 0 {
		t.Fatalf("available capacity = %d, want 0", state.available)
	}
	bucket.rejectRefill(refill)
}

// TestBucketAdmitsWaitersOnlyWhileRefillsAreAllowed verifies that an active
// refill admits requests only while refills are allowed.
func TestBucketAdmitsWaitersOnlyWhileRefillsAreAllowed(t *testing.T) {
	bucket := newTestBucket()
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	satisfied, queued, waiter := bucket.consume(2, true)
	if satisfied || queued != nil || waiter == nil {
		t.Fatalf("cost-2 request was not admitted as waiter")
	}
	satisfied, queued, waiter = bucket.consume(1, false)
	if satisfied || queued != nil || waiter != nil {
		t.Fatal("closed refills allowed waiting")
	}
}

// TestBucketThresholdScalesWithTarget verifies that small leases use
// proportionally smaller refill thresholds.
func TestBucketThresholdScalesWithTarget(t *testing.T) {
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
			bucket := newTestBucket()
			applyTestLease(bucket, 0, test.target)
			if bucket.refillThreshold != test.threshold {
				t.Fatalf("refill threshold = %d, want %d", bucket.refillThreshold, test.threshold)
			}
		})
	}
}

// TestBucketQueuesRefillRelativeToOperationCost verifies that a variable-cost
// operation queues a refill while one similar operation remains.
func TestBucketQueuesRefillRelativeToOperationCost(t *testing.T) {
	bucket := newTestLargeBucket()
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
	if state := bucketSnapshot(bucket); state.available != 250 {
		t.Fatalf("available capacity = %d, want 250", state.available)
	}
}
