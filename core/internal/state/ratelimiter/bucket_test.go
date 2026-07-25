// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"strconv"
	"testing"
)

// TestDisabledBucketDoesNotRequestOrApplyCapacity verifies that a removed
// subject cannot consume, refill, or receive capacity.
func TestDisabledBucketDoesNotRequestOrApplyCapacity(t *testing.T) {
	bucket := NewWorkspaceBucket("222222222222")
	applyTestLease(bucket, 10, 10)
	bucket.Disable()

	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill != nil || waiter != nil {
		t.Fatalf("disabled bucket consumed or requested capacity: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	applyTestLease(bucket, 10, 10)
	available, _, _, _, disabled := bucketSnapshot(bucket)
	if available != 0 || !disabled {
		t.Fatalf("disabled bucket state = available:%d disabled:%t, want 0:true", available, disabled)
	}
	bucket.Restore(1)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != 0 {
		t.Fatalf("restored capacity on a disabled bucket = %d, want 0", available)
	}
}

// TestBucketRestoresLocalCapacity verifies that returned capacity is available
// immediately and remains capped at the local target.
func TestBucketRestoresLocalCapacity(t *testing.T) {
	bucket := NewWorkspaceBucket("222222222222")
	applyTestLease(bucket, 10, 10)

	satisfied, _, _ := bucket.consume(4, true)
	if !satisfied {
		t.Fatal("initial consumption was not satisfied")
	}
	bucket.Restore(3)
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 9 {
		t.Fatalf("restored capacity = %d, want 9", available)
	}

	bucket.Restore(3)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != 10 {
		t.Fatalf("capacity above the local target = %d, want 10", available)
	}
	bucket.Restore(0)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != 10 {
		t.Fatalf("zero restoration changed capacity to %d, want 10", available)
	}
	bucket.Restore(bucket.maxCost + 1)
	available, _, _, _, _ = bucketSnapshot(bucket)
	if available != 10 {
		t.Fatalf("invalid restoration changed capacity to %d, want 10", available)
	}
}

// TestIngestionBucketWaitsForInitialRefill verifies that a first
// single-event request waits for its initial refill instead of using capacity
// that has not been leased yet.
func TestIngestionBucketWaitsForInitialRefill(t *testing.T) {
	bucket := NewIngestionBucket("222222222222")

	satisfied, refill, waiter := bucket.consume(1, true)
	if satisfied || refill == nil || waiter != nil {
		t.Fatalf("initial ingestion consume: satisfied=%t refill=%p waiter=%p", satisfied, refill, waiter)
	}
	if waiter := bucket.activateRefill(refill, 1, true, true); waiter == nil {
		t.Fatal("initial ingestion request was not admitted as a waiter")
	}
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 0 {
		t.Fatalf("available capacity = %d, want 0", available)
	}
	bucket.rejectRefill(refill)
}

// TestBucketAdmitsCostOneToPendingRefill verifies that a cost-one request
// waits for an active refill when local capacity is exhausted.
func TestBucketAdmitsCostOneToPendingRefill(t *testing.T) {
	bucket := NewOrganizationBucket("111111111111")
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
	available, _, _, _, _ := bucketSnapshot(bucket)
	if available != 0 {
		t.Fatalf("available capacity = %d, want 0", available)
	}
	bucket.rejectRefill(refill)
}

// TestBucketAdmitsWaitersOnlyWhileRefillsAreAllowed verifies that an active
// refill admits requests only while refills are allowed.
func TestBucketAdmitsWaitersOnlyWhileRefillsAreAllowed(t *testing.T) {
	bucket := NewOrganizationBucket("111111111111")
	_, refill, _ := bucket.consume(2, true)
	bucket.activateRefill(refill, 0, false, true)

	satisfied, queued, waiter := bucket.consume(2, true)
	if satisfied || queued != nil || waiter == nil {
		t.Fatalf("cost-2 request was not admitted as waiter")
	}
	satisfied, queued, waiter = bucket.consume(1, false)
	if satisfied || queued != nil || waiter != nil {
		t.Fatal("disabled refills allowed waiting")
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
			bucket := NewOrganizationBucket("111111111111")
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
	bucket := NewIngestionBucket("222222222222")
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
