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
				t.Fatalf("invalid configuration exit = %v, want exit status 1", err)
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
		t.Fatalf("restored capacity = %d, want 9", got)
	}
	if err := bucket.Restore(3); err != nil {
		t.Fatalf("second restoration: %v", err)
	}
	if got := bucketAvailable(bucket); got != 10 {
		t.Fatalf("capacity above local target = %d, want 10", got)
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
				t.Fatalf("refill threshold = %d, want %d", bucket.refillThreshold, test.threshold)
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
			t.Fatalf("requested units = %d, want 60", got)
		}
	case <-time.After(time.Second):
		t.Fatal("relative consumption did not start a refill")
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
		if err := bucket.Consume(t.Context(), units); err == nil || errors.Is(err, ErrCapacityExceeded) || errors.Is(err, ErrLimiterUnavailable) {
			t.Fatalf("Consume(%d) = %v, want generic error", units, err)
		}
	}
}

func TestRestoreReturnsInvalidUnits(t *testing.T) {
	bucket := newTestBucket()
	for _, units := range []int{0, testLeaseSize + 1} {
		if err := bucket.Restore(units); err == nil {
			t.Fatalf("Restore(%d) returned nil", units)
		}
	}
}
