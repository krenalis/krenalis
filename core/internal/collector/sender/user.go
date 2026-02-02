// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"cmp"
	"math"
	"slices"
	"sync"
)

// user represents the per-user state used during event processing.
type user struct {
	anonymousID string    // anonymous ID.
	queue       userQueue // per-user queue holding out-of-order events.
	iterator    *iterator // iterator over the user's events; nil when no iteration is active.
	consumed    int       // number of events already consumed from iterator, if any.
	totals      int       // total number of the user's events in the sender queue.
}

// disposable reports whether the user has no pending or queued events and can
// be safely removed from the sender.
func (u *user) disposable() bool {
	seq := u.queue.sequence
	return seq.expected == seq.next && u.totals == 0 && u.consumed == 0
}

// userQueue represents a per-user event queue.
//
// Normally, an event passed to sender.SendEvent is enqueued directly into the
// sender queue, and an event passed to sender.DiscardEvent is discarded
// immediately.
//
// When events are not processed in creation order—from the earliest created
// with sender.CreateEvent to the latest—they must first be buffered in a
// per-user queue. This allows events to be reordered before being enqueued into
// the sender queue in the correct order.
//
// userQueue therefore holds events, both to be sent and discarded, that are
// waiting for an earlier-created event to arrive.
//
// Events are ordered from the most recent to the least recent.
type userQueue struct {
	events   []*Event // events waiting for the event with the expected sequence to be enqueued first.
	sequence struct {
		expected int // sequence number expected to be added next to Sender.events.
		next     int // next sequence number to assign to a newly created event for this user.
	}
}

// enqueue adds event to the per-user queue or forwards it immediately.
//
// If event has the expected sequence number, enqueue advances the expected
// sequence and forwards event (and any subsequently unblocked queued events)
// by calling forward. Events that were discarded are not forwarded, but they
// still advance the expected sequence and may unblock later events.
//
// If event is out of order, it is inserted into the per-user queue to wait
// until all earlier sequence numbers have been processed.
//
// If sender is closed, forward returns false and enqueue returns immediately.
func (q *userQueue) enqueue(event *Event, forward func(event *Event) bool) {
	// If the sequence has been rescaled, realign the current event's number.
	if event.sequence > q.sequence.next {
		event.sequence += math.MinInt
	}
	if event.sequence != q.sequence.expected {
		i, _ := slices.BinarySearchFunc(q.events, event, func(a, b *Event) int {
			return cmp.Compare(b.sequence, a.sequence)
		})
		q.events = slices.Insert(q.events, i, event)
		return
	}
	if !event.discarded {
		if !forward(event) {
			return // The sender is closed.
		}
	}
	expected := q.sequence.expected + 1
	for len(q.events) > 0 {
		last := len(q.events) - 1
		if q.events[last].sequence != expected {
			break
		}
		// Append the event to the ready queue unless it has been discarded.
		if !q.events[last].discarded {
			if !forward(q.events[last]) {
				return // The sender is closed.
			}
		}
		expected++
		q.events[last] = nil
		q.events = q.events[:last]
	}
	q.sequence.expected = expected
	return
}

// next returns the next sequence number for this user.
// It is called for each new event of this user.
func (q *userQueue) next() int {
	next := q.sequence.next
	q.sequence.next++
	// On overflow (unlikely in practice),
	// shift sequence numbers to keep ordering consistent.
	if q.sequence.next < 0 {
		q.sequence.next = 0
		q.sequence.expected += math.MinInt
		for _, event := range q.events {
			event.sequence += math.MinInt
		}
	}
	return next
}

// users is a shared pool used by all senders.
// It allows reuse of *user instances to reduce allocations and GC pressure.
var users usersPool

type usersPool struct {
	p sync.Pool
}

// Get returns a *user from the pool.
// If the pool is empty, a new instance is allocated.
// The returned user is reset to a clean state while preserving internal
// buffers.
func (p *usersPool) Get() *user {
	v := p.p.Get()
	if v == nil {
		return new(user)
	}
	u := v.(*user)
	events := u.queue.events[:0]
	*u = user{}
	u.queue.events = events
	return u
}

// Put returns a *user to the pool for reuse.
// The caller must ensure the user is no longer in use.
func (p *usersPool) Put(u *user) {
	p.p.Put(u)
}
