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
	// maxRefillThreshold caps only the absolute local-capacity threshold for
	// starting a proactive refill. Consume may also start a refill when the
	// remaining capacity is no greater than the units just consumed. Capacity
	// below this threshold remains available for consumption while the refill is
	// in progress.
	maxRefillThreshold = 25

	// initialLeaseTarget is the baseline local target used before a subject has
	// shown sustained demand on this node. A larger operation raises the target
	// for the refill that admits it.
	initialLeaseTarget = 10

	// leaseGrowthWindow is the interval after a positive grant during which the
	// next refill doubles the local target.
	leaseGrowthWindow = 5 * time.Second

	// microsecondsPerMinute is the denominator used by PostgreSQL's fractional
	// per-minute refill calculation.
	microsecondsPerMinute = 60 * 1_000_000
)

// maxWaitDuration limits how long a caller waits for a single refill
// generation.
// It is a variable so tests can use a shorter duration.
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
	limiter     *Limiter    // limiter that owns the bucket
	subjectKind SubjectKind // subject class
	subjectID   string      // subject identifier within its class
	leaseSize   int         // maximum capacity reserved locally
	maxUnits    int         // maximum units accepted by a single operation

	mu              sync.Mutex
	available       int       // locally available capacity; protected by mu
	localTarget     int       // current local capacity limit; protected by mu
	refillThreshold int       // capacity below which a proactive refill starts; protected by mu
	growthGrantAt   time.Time // positive grant time used to determine target growth; protected by mu
	refill          *refill   // current refill generation, if any; protected by mu
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
		if refill == nil {
			bucket.mu.Unlock()
			return nil
		}
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
	refill := bucket.newRefillWithAdmissionLocked(units)
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
// Restoring units never increases local capacity above the current limit. Any
// excess is scheduled for best-effort restoration to PostgreSQL.
//
// Restore returns an error if the number of units is not between 1 and the
// configured maximum, inclusive. A nil error does not guarantee that an
// asynchronous PostgreSQL restoration will succeed.
func (bucket *Bucket) Restore(units int) error {
	if units < 1 || units > bucket.maxUnits {
		return fmt.Errorf("units (%d) are not in range [1, %d]", units, bucket.maxUnits)
	}
	bucket.mu.Lock()
	availableSpace := max(0, bucket.localTarget-bucket.available)
	localUnits := min(units, availableSpace)
	bucket.available += localUnits
	excessUnits := units - localUnits
	bucket.mu.Unlock()
	bucket.limiter.queueCapacityRestoration(bucket.subjectKind, bucket.subjectID, excessUnits)
	return nil
}

// admitWaiterLocked adds a waiter to the refill generation's FIFO queue. It
// returns nil if the requested units exceed the generation's remaining
// admission budget. This indicates local saturation, not authoritative
// capacity exhaustion. The caller must hold bucket.mu.
func (bucket *Bucket) admitWaiterLocked(refill *refill, units int) *waiter {
	if units > refill.request.RequestedUnits-refill.pendingUnits {
		return nil
	}
	waiter := &waiter{refill: refill, units: units}
	waiter.element = refill.waiters.PushBack(waiter)
	refill.pendingUnits += units
	return waiter
}

// applyLeaseLocked applies the granted capacity up to the local target and
// returns the number of newly granted units that do not fit in the bucket.
func (bucket *Bucket) applyLeaseLocked(grantedUnits, capacityUnits, targetUnits int) int {
	// An adaptive target decrease drains capacity already held locally instead
	// of revoking it. A decrease in authoritative capacity may still revoke
	// local capacity immediately.
	bucket.localTarget = min(capacityUnits, max(targetUnits, bucket.available))
	bucket.refillThreshold = max(1, min(maxRefillThreshold, bucket.localTarget/4))
	// A lower target revokes previously leased local capacity above the new
	// limit. Revoked capacity is discarded rather than returned to PostgreSQL.
	if bucket.available > bucket.localTarget {
		bucket.available = bucket.localTarget
	}
	availableSpace := bucket.localTarget - bucket.available
	localUnits := min(grantedUnits, availableSpace)
	bucket.available += localUnits
	return grantedUnits - localUnits
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
	unappliedGrant := bucket.applyLeaseLocked(result.GrantedUnits, result.CapacityUnits, refill.targetUnits)
	if result.GrantedUnits > 0 {
		bucket.growthGrantAt = time.Now()
	} else {
		bucket.growthGrantAt = time.Time{}
	}
	// Units that do not fit within the local target may still be consumed by
	// admitted waiters. Keep those units outside bucket.available until the
	// satisfiable prefix has been served, then restore only the unused remainder.
	availableUnits := bucket.available + unappliedGrant
	element := refill.waiters.Front()
	for ; element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		if waiter.units > result.CapacityUnits || availableUnits < waiter.units {
			break
		}
		availableUnits -= waiter.units
		waiter.err = nil
		waiter.element = nil
	}
	excessUnits := max(0, availableUnits-bucket.localTarget)
	bucket.available = availableUnits - excessUnits

	retryUnits := 0
	for ; element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		err := CapacityExceededError{}
		if waiter.units <= result.CapacityUnits {
			retryUnits += waiter.units
			requiredUnits := max(0, retryUnits-bucket.available)
			err.RetryAfter = calculateRetryAfter(requiredUnits, result)
		}
		waiter.err = err
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingUnits = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	bucket.limiter.queueCapacityRestoration(bucket.subjectKind, bucket.subjectID, excessUnits)
	close(refill.done)
}

// newRefillLocked creates the immutable lease request for the next refill. It
// returns nil when the existing local capacity already satisfies the next
// adaptive target. The caller must hold bucket.mu.
func (bucket *Bucket) newRefillLocked() *refill {
	return bucket.newRefillWithAdmissionLocked(0)
}

// newRefillWithAdmissionLocked creates the immutable lease request for the next
// refill and ensures an admission budget of at least minAdmissionUnits. It
// returns nil when minAdmissionUnits is zero and the existing local capacity
// already satisfies the next adaptive target. The caller must hold bucket.mu.
func (bucket *Bucket) newRefillWithAdmissionLocked(minAdmissionUnits int) *refill {
	baselineTarget := min(initialLeaseTarget, bucket.leaseSize)
	targetUnits := baselineTarget
	if !bucket.growthGrantAt.IsZero() && time.Since(bucket.growthGrantAt) <= leaseGrowthWindow {
		grownTarget := bucket.leaseSize
		if bucket.localTarget <= bucket.leaseSize/2 {
			grownTarget = bucket.localTarget * 2
		}
		targetUnits = max(targetUnits, grownTarget)
	}
	admissionUnits := minAdmissionUnits
	if minAdmissionUnits > 0 {
		// Include as much baseline admission headroom as the lease size permits
		// after accounting for the operation that triggered the refill.
		admissionUnits += min(baselineTarget, bucket.leaseSize-minAdmissionUnits)
		targetUnits = max(targetUnits, admissionUnits)
	}
	requestedUnits := max(admissionUnits, targetUnits-bucket.available)
	if requestedUnits == 0 {
		return nil
	}
	refill := &refill{
		bucket:      bucket,
		targetUnits: targetUnits,
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
	targetUnits  int
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
