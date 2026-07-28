// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"container/list"
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	maxRefillThreshold    = 25
	microsecondsPerMinute = 60 * 1_000_000
)

var maxWaitDuration = time.Second

// Bucket manages node-local rate-limit capacity for one subject, identified by
// a SubjectKind and an identifier. The capacity is used for rate-limited
// operations associated with that subject.
//
// Consume capacity before performing a rate-limited operation:
//
//	if err := bucket.Consume(ctx, units); err != nil {
//		return err
//	}
//
// Optionally restore any consumed capacity that was ultimately not used:
//
//	if err := bucket.Restore(unusedUnits); err != nil {
//		return err
//	}
type Bucket struct {
	limiter     *Limiter
	subjectKind SubjectKind
	subjectID   string
	leaseSize   int
	maxUnits    int

	mu              sync.Mutex
	available       int
	localTarget     int
	refillThreshold int
	refill          *refill
}

// Consume consumes the specified number of units from the bucket.
// units must be between 1 and the configured maximum, inclusive.
//
// Consume returns a CapacityExceededError when the requested capacity is
// unavailable. It returns ErrLimiterUnavailable when a temporary condition
// prevents the limiter from determining whether capacity is available.
func (bucket *Bucket) Consume(ctx context.Context, units int) error {
	if units < 1 || units > bucket.maxUnits {
		return fmt.Errorf("rate-limit units %d are not in range [1, %d]", units, bucket.maxUnits)
	}

	limiter := bucket.limiter
	backoffErr := limiter.backoffError()

	bucket.mu.Lock()

	// Consume locally and refill when little capacity remains, either
	// absolutely or relative to the current consumption.
	if units <= bucket.available {
		bucket.available -= units
		if backoffErr != nil {
			bucket.mu.Unlock()
			return nil
		}
		startRefill := bucket.refill == nil &&
			(bucket.available < bucket.refillThreshold || bucket.available <= units)
		if !startRefill {
			bucket.mu.Unlock()
			return nil
		}
		refill := bucket.newRefillLocked()
		bucket.mu.Unlock()
		// Refill failure does not affect capacity already consumed.
		limiter.publishRefill(refill)
		return nil
	}

	// Reject requests without local capacity during backoff.
	if backoffErr != nil {
		bucket.mu.Unlock()
		return backoffErr
	}

	// Try to admit the request to the current refill.
	if bucket.refill != nil {
		waiter := bucket.admitWaiterLocked(bucket.refill, units)
		bucket.mu.Unlock()
		if waiter == nil {
			return ErrLimiterUnavailable
		}
		return waitForRefill(ctx, waiter)
	}

	// Start a new refill and try to admit the request.
	refill := bucket.newRefillLocked()
	waiter := bucket.admitWaiterLocked(refill, units)
	bucket.mu.Unlock()
	if !limiter.publishRefill(refill) {
		return ErrLimiterUnavailable
	}
	if waiter == nil {
		return ErrLimiterUnavailable
	}

	return waitForRefill(ctx, waiter)
}

// Restore restores units previously consumed from the bucket.
// Restoring units never increases local capacity above the current limit.
//
// Restore returns an error if the units value is not between 1 and the
// configured maximum, inclusive.
func (bucket *Bucket) Restore(units int) error {
	if units < 1 || units > bucket.maxUnits {
		return fmt.Errorf("units (%d) are not in range [1, %d]", units, bucket.maxUnits)
	}
	bucket.mu.Lock()
	if units >= bucket.localTarget-bucket.available {
		bucket.available = bucket.localTarget
	} else {
		bucket.available += units
	}
	bucket.mu.Unlock()
	return nil
}

// admitWaiterLocked admits a waiter to the refill's FIFO queue. The caller
// must hold bucket.mu.
func (bucket *Bucket) admitWaiterLocked(refill *refill, units int) *waiter {
	if units > refill.request.RequestedUnits-refill.pendingUnits {
		return nil
	}
	waiter := &waiter{refill: refill, units: units}
	waiter.element = refill.waiters.PushBack(waiter)
	refill.pendingUnits += units
	return waiter
}

// applyLeaseLocked applies granted capacity, capped at the local target.
func (bucket *Bucket) applyLeaseLocked(grantedUnits, capacityUnits int) {
	bucket.localTarget = min(bucket.leaseSize, capacityUnits)
	bucket.refillThreshold = max(1, min(maxRefillThreshold, bucket.localTarget/4))
	if grantedUnits >= bucket.localTarget-bucket.available {
		bucket.available = bucket.localTarget
	} else {
		bucket.available += grantedUnits
	}
}

// calculateRetryAfter calculates when PostgreSQL can regenerate requiredUnits,
// after accounting for the capacity that remains authoritative in result.
func calculateRetryAfter(requiredUnits int, result leaseResult) time.Duration {
	missing := max(0, requiredUnits-result.AvailableUnits)
	if missing == 0 {
		return 0
	}
	numerator := max(int64(missing)*microsecondsPerMinute-int64(result.RefillRemainder), 1)
	microseconds := (numerator + int64(result.RatePerMinute) - 1) / int64(result.RatePerMinute)
	return time.Duration(microseconds) * time.Microsecond
}

// cancelWaiter serializes caller cancellation with refill completion.
func (bucket *Bucket) cancelWaiter(waiter *waiter, cancellation error) error {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if waiter.element == nil {
		return waiter.err
	}
	refill := waiter.refill
	refill.waiters.Remove(waiter.element)
	refill.pendingUnits -= waiter.units
	waiter.element = nil
	waiter.err = cancellation
	return cancellation
}

// completeRefill applies a valid grant and serves the longest satisfiable FIFO
// prefix. Capacity is assigned before waiters are notified.
func (bucket *Bucket) completeRefill(refill *refill, result leaseResult) {
	bucket.mu.Lock()
	if bucket.refill != refill {
		bucket.mu.Unlock()
		return
	}
	bucket.applyLeaseLocked(result.GrantedUnits, result.CapacityUnits)
	serve := true
	retryUnits := 0
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		if serve && bucket.available >= waiter.units {
			bucket.available -= waiter.units
			waiter.err = nil
		} else {
			serve = false
			err := CapacityExceededError{}
			if waiter.units <= result.CapacityUnits {
				retryUnits += waiter.units
				requiredUnits := max(0, retryUnits-bucket.available)
				err.RetryAfter = calculateRetryAfter(requiredUnits, result)
			}
			waiter.err = err
		}
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingUnits = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	close(refill.done)
}

// newRefillLocked creates the immutable lease request for the next refill.
// The caller must hold bucket.mu.
func (bucket *Bucket) newRefillLocked() *refill {
	requestedUnits := bucket.leaseSize
	if bucket.localTarget > 0 {
		requestedUnits = min(bucket.leaseSize, bucket.localTarget-bucket.available)
	}
	refill := &refill{
		bucket: bucket,
		request: leaseRequest{
			SubjectKind:    bucket.subjectKind,
			SubjectID:      bucket.subjectID,
			RequestedUnits: requestedUnits,
		},
		done: make(chan struct{}),
	}
	bucket.refill = refill
	return refill
}

// rejectRefill rejects all remaining waiters and detaches the generation.
func (bucket *Bucket) rejectRefill(refill *refill, err error) {
	bucket.mu.Lock()
	if bucket.refill != refill {
		bucket.mu.Unlock()
		return
	}
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		waiter.err = err
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingUnits = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	close(refill.done)
}

// waitForRefill waits until the refill resolves, the caller cancels, or the
// internal wait deadline expires.
func waitForRefill(ctx context.Context, waiter *waiter) error {
	timer := time.NewTimer(maxWaitDuration)
	defer timer.Stop()
	select {
	case <-waiter.refill.done:
		return waiter.err
	case <-ctx.Done():
		return waiter.refill.bucket.cancelWaiter(waiter, ctx.Err())
	case <-timer.C:
		cancellation := ErrLimiterUnavailable
		if err := ctx.Err(); err != nil {
			cancellation = err
		}
		return waiter.refill.bucket.cancelWaiter(waiter, cancellation)
	}
}

// refill represents one immutable lease request and its admitted waiters.
// Its mutable state is protected by its bucket mutex.
type refill struct {
	bucket       *Bucket
	request      leaseRequest
	done         chan struct{}
	waiters      list.List
	pendingUnits int
}

// waiter represents one request admitted to a refill. element is non-nil only
// while the waiter belongs to the refill's FIFO queue.
type waiter struct {
	refill  *refill
	units   int
	element *list.Element
	err     error
}
