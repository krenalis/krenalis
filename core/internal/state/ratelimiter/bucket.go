// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import "sync"

const maxRefillThreshold = 25

// SubjectKind identifies a class of rate-limit buckets.
//
// The limiter treats subject kinds as opaque values. The caller defines which
// kinds are supported and how their leases are acquired.
type SubjectKind string

// Bucket stores process-local capacity for one subject kind and identifier.
//
// mu protects local capacity and refill state.
type Bucket struct {
	subjectKind SubjectKind
	subjectID   string
	leaseSize   int
	maxCost     int

	mu              sync.Mutex
	available       int
	localTarget     int
	refillThreshold int
	refill          *refill
	closed          bool
}

// NewBucket creates an empty local bucket for subjectKind and subjectID.
//
// leaseSize and maxCost must be positive, and maxCost must not exceed
// leaseSize. NewBucket panics if these conditions are not met.
func NewBucket(subjectKind SubjectKind, subjectID string, leaseSize, maxCost int) *Bucket {
	if leaseSize < 1 || maxCost < 1 || maxCost > leaseSize {
		panic("invalid rate-limit bucket configuration")
	}
	return &Bucket{
		subjectKind: subjectKind,
		subjectID:   subjectID,
		leaseSize:   leaseSize,
		maxCost:     maxCost,
	}
}

// Close closes the bucket, preventing further consumption and refills.
// It clears local capacity and rejects pending waiters. A queued pointer
// remains safe because Go keeps the bucket alive; refillRequest and
// completeRefill discards later work for closed buckets.
func (bucket *Bucket) Close() {
	bucket.mu.Lock()
	bucket.closed = true
	bucket.available = 0
	refill := bucket.refill
	rejected := bucket.rejectRefillLocked(refill)
	bucket.mu.Unlock()
	if rejected {
		closeRejectedRefill(refill)
	}
}

// Restore restores previously consumed capacity to this bucket. It does not
// affect the lease acquirer, pending refills, or admitted waiters.
func (bucket *Bucket) Restore(cost int) {
	if cost < 1 || cost > bucket.maxCost {
		return
	}
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.closed {
		return
	}
	bucket.available = min(bucket.localTarget, bucket.available+cost)
}

// activateRefill activates a successfully queued refill and optionally admits
// the request that created it. Activation and admission are one atomic bucket
// transition, so the batcher cannot complete the refill between them.
func (bucket *Bucket) activateRefill(refill *refill, cost int, admit bool, refillAllowed bool) *waiter {
	bucket.mu.Lock()
	if bucket.refill != refill || refill.active {
		bucket.mu.Unlock()
		return nil
	}
	if bucket.closed || !refillAllowed {
		rejected := bucket.rejectRefillLocked(refill)
		bucket.mu.Unlock()
		if rejected {
			closeRejectedRefill(refill)
		}
		return nil
	}
	refill.active = true
	var waiter *waiter
	if admit {
		waiter = bucket.admitWaiterLocked(refill, cost)
	}
	bucket.mu.Unlock()
	close(refill.published)
	return waiter
}

// admitWaiterLocked admits a waiter to the refill's FIFO queue.
//
// The caller must hold bucket.mu. It returns nil if the waiter's cost would
// exceed the refill's requested units.
func (bucket *Bucket) admitWaiterLocked(refill *refill, cost int) *waiter {
	if refill.pendingCost+cost > refill.request.RequestedUnits {
		return nil
	}
	waiter := &waiter{refill: refill, cost: cost}
	waiter.element = refill.waiters.PushBack(waiter)
	refill.pendingCost += cost
	return waiter
}

// applyLeaseLocked applies granted capacity, capped at the local target.
func (bucket *Bucket) applyLeaseLocked(grantedUnits, capacityUnits int) {
	bucket.localTarget = min(bucket.leaseSize, capacityUnits)
	// A fixed threshold would trigger a refill after almost every request when
	// the local target is small. Scale it with the target, with one unit as the
	// minimum.
	bucket.refillThreshold = max(1, min(maxRefillThreshold, bucket.localTarget/4))
	bucket.available = min(bucket.localTarget, bucket.available+grantedUnits)
}

// cancelWaiter cancels a waiter while serializing with refill completion.
// If completion acquired the bucket mutex first, its already-final decision is
// returned.
func (bucket *Bucket) cancelWaiter(waiter *waiter, cancellation error) error {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if waiter.element == nil {
		return waiter.err
	}
	refill := waiter.refill
	refill.waiters.Remove(waiter.element)
	refill.pendingCost -= waiter.cost
	waiter.element = nil
	waiter.err = cancellation
	return cancellation
}

// completeRefill completes a refill by applying a valid grant and serving the
// longest satisfiable FIFO prefix while holding the bucket mutex.
//
// Capacity is deducted before any waiter is awakened, so later requests cannot
// consume assigned units.
func (bucket *Bucket) completeRefill(refill *refill, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	if bucket.closed || bucket.refill != refill || !refill.active {
		bucket.mu.Unlock()
		return
	}
	bucket.applyLeaseLocked(grantedUnits, capacityUnits)
	serve := true
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		if serve && bucket.available >= waiter.cost {
			bucket.available -= waiter.cost
			waiter.err = nil
		} else {
			serve = false
			waiter.err = ErrCapacityExceeded
		}
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingCost = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	close(refill.done)
}

// consume consumes cost from local capacity, admits it to an active refill, or
// prepares a new refill. A returned refill is still being published and must be
// placed on the limiter queue before it can admit a waiter.
func (bucket *Bucket) consume(cost int, refillAllowed bool) (satisfied bool, refill *refill, waiter *waiter) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.closed {
		return false, nil, nil
	}

	if bucket.available >= cost {
		bucket.available -= cost
		satisfied = true
	} else if refillAllowed && bucket.refill != nil && bucket.refill.active {
		return false, nil, bucket.admitWaiterLocked(bucket.refill, cost)
	}

	if !satisfied && bucket.refill != nil {
		// Requests racing with the short publishing phase are deliberately not
		// admitted. Callers may retry after publication has been confirmed.
		return false, nil, nil
	}
	needsRefill := !satisfied || bucket.available < bucket.refillThreshold || bucket.available <= cost
	if needsRefill && bucket.refill == nil && refillAllowed {
		refill = bucket.newRefillLocked()
	}
	return satisfied, refill, nil
}

// newRefillLocked creates a refill generation for the missing local capacity.
// The caller must hold bucket.mu.
func (bucket *Bucket) newRefillLocked() *refill {
	requestedUnits := min(bucket.leaseSize, bucket.localTarget-bucket.available)
	if bucket.localTarget == 0 {
		requestedUnits = bucket.leaseSize
	}
	refill := &refill{
		bucket: bucket,
		request: leaseRequest{
			SubjectKind:    bucket.subjectKind,
			SubjectID:      bucket.subjectID,
			RequestedUnits: requestedUnits,
		},
		published: make(chan struct{}),
		done:      make(chan struct{}),
	}
	bucket.refill = refill
	return refill
}

// refillRequest returns the immutable request frozen before queue publication.
func (bucket *Bucket) refillRequest(refill *refill) (leaseRequest, bool) {
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if bucket.closed || bucket.refill != refill || !refill.active {
		return leaseRequest{}, false
	}
	return refill.request, true
}

// rejectRefill rejects every remaining waiter and detaches the refill
// generation.
func (bucket *Bucket) rejectRefill(refill *refill) {
	bucket.mu.Lock()
	rejected := bucket.rejectRefillLocked(refill)
	bucket.mu.Unlock()
	if rejected {
		closeRejectedRefill(refill)
	}
}

// rejectRefillLocked rejects every waiter and detaches the refill generation.
// The caller must hold bucket.mu. It reports whether the refill was attached.
func (bucket *Bucket) rejectRefillLocked(refill *refill) bool {
	if refill == nil || bucket.refill != refill {
		return false
	}
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		waiter.err = ErrCapacityExceeded
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingCost = 0
	bucket.refill = nil
	return true
}
