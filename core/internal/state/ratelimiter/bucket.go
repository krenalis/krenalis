// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package ratelimiter

import (
	"container/list"
	"context"
	"log/slog"
	"os"
	"sync"
	"time"
)

const maxRefillThreshold = 25

var maxWaitDuration = time.Second

// Bucket stores process-local capacity for one subject kind and identifier.
//
// A bucket does not need to be closed. Once its owner stops using it, it is
// collected after any queued or in-progress refill releases its last reference.
// mu protects local capacity and refill state.
type Bucket struct {
	limiter     *Limiter
	subjectKind SubjectKind
	subjectID   string
	leaseSize   int
	maxCost     int

	mu              sync.Mutex
	available       int
	localTarget     int
	refillThreshold int
	refill          *refill
}

// refill represents one immutable lease request and its admitted waiters.
// Its mutable state is protected by its bucket mutex.
type refill struct {
	bucket      *Bucket
	request     leaseRequest
	done        chan struct{}
	waiters     list.List
	pendingCost int
}

// Consume consumes cost from this bucket. An invalid cost is a programming
// error and terminates the process. If local capacity is insufficient, Consume
// may wait for the refill generation to which it was admitted.
func (bucket *Bucket) Consume(ctx context.Context, cost int) error {
	if cost < 1 || cost > bucket.maxCost {
		slog.Error("core/internal/state/ratelimiter: cost is not in range [1, max cost]", "cost", cost, "max_cost", bucket.maxCost)
		os.Exit(1)
	}

	limiter := bucket.limiter
	refillAllowed := !limiter.backoffActive()

	bucket.mu.Lock()

	// Consume locally and refill when little capacity remains, either
	// absolutely or relative to the current cost.
	if cost <= bucket.available {
		bucket.available -= cost
		if !refillAllowed {
			bucket.mu.Unlock()
			return nil
		}
		startRefill := bucket.refill == nil &&
			(bucket.available < bucket.refillThreshold || bucket.available <= cost)
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
	if !refillAllowed {
		bucket.mu.Unlock()
		return ErrCapacityExceeded
	}

	// Try to admit the request to the current refill.
	if bucket.refill != nil {
		waiter := bucket.admitWaiterLocked(bucket.refill, cost)
		bucket.mu.Unlock()
		if waiter == nil {
			return ErrCapacityExceeded
		}
		return waitForRefill(ctx, waiter)
	}

	// Start a new refill and try to admit the request.
	refill := bucket.newRefillLocked()
	waiter := bucket.admitWaiterLocked(refill, cost)
	bucket.mu.Unlock()
	if !limiter.publishRefill(refill) {
		return ErrCapacityExceeded
	}
	if waiter == nil {
		return ErrCapacityExceeded
	}

	return waitForRefill(ctx, waiter)
}

// Restore restores previously consumed capacity to this bucket. It does not
// affect the lease acquirer or pending refills.
func (bucket *Bucket) Restore(cost int) {
	if cost < 1 || cost > bucket.maxCost {
		slog.Error("core/internal/state/ratelimiter: cost is not in range [1, max cost]", "cost", cost, "max_cost", bucket.maxCost)
		os.Exit(1)
	}
	bucket.mu.Lock()
	if cost >= bucket.localTarget-bucket.available {
		bucket.available = bucket.localTarget
	} else {
		bucket.available += cost
	}
	bucket.mu.Unlock()
}

// admitWaiterLocked admits a waiter to the refill's FIFO queue. The caller
// must hold bucket.mu.
func (bucket *Bucket) admitWaiterLocked(refill *refill, cost int) *waiter {
	if cost > refill.request.RequestedUnits-refill.pendingCost {
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
	bucket.refillThreshold = max(1, min(maxRefillThreshold, bucket.localTarget/4))
	if grantedUnits >= bucket.localTarget-bucket.available {
		bucket.available = bucket.localTarget
	} else {
		bucket.available += grantedUnits
	}
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
	refill.pendingCost -= waiter.cost
	waiter.element = nil
	waiter.err = cancellation
	return cancellation
}

func waitForRefill(ctx context.Context, waiter *waiter) error {
	timer := time.NewTimer(maxWaitDuration)
	defer timer.Stop()
	select {
	case <-waiter.refill.done:
		return waiter.err
	case <-ctx.Done():
		return waiter.refill.bucket.cancelWaiter(waiter, ctx.Err())
	case <-timer.C:
		cancellation := ErrCapacityExceeded
		if err := ctx.Err(); err != nil {
			cancellation = err
		}
		return waiter.refill.bucket.cancelWaiter(waiter, cancellation)
	}
}

// completeRefill applies a valid grant and serves the longest satisfiable FIFO
// prefix. Capacity is assigned before waiters are notified.
func (bucket *Bucket) completeRefill(refill *refill, grantedUnits, capacityUnits int) {
	bucket.mu.Lock()
	if bucket.refill != refill {
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
func (bucket *Bucket) rejectRefill(refill *refill) {
	bucket.mu.Lock()
	if bucket.refill != refill {
		bucket.mu.Unlock()
		return
	}
	for element := refill.waiters.Front(); element != nil; element = element.Next() {
		waiter := element.Value.(*waiter)
		waiter.err = ErrCapacityExceeded
		waiter.element = nil
	}
	refill.waiters.Init()
	refill.pendingCost = 0
	bucket.refill = nil
	bucket.mu.Unlock()
	close(refill.done)
}

// waiter represents one request admitted to a refill. element is non-nil only
// while the waiter belongs to the refill's FIFO queue.
type waiter struct {
	refill  *refill
	cost    int
	element *list.Element
	err     error
}
