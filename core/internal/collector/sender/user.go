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

// orderingKey identifies an independent event ordering for a user.
type orderingKey struct {
	group       string // ordering group.
	anonymousID string // anonymous ID.
}

// ordering owns the queued and in-flight events associated with one ordering
// key.
type ordering struct {
	key      orderingKey   // key identifying the event ordering.
	queue    orderingQueue // events waiting for preceding events.
	iterator *iterator     // iterator over the ordering's events; nil when no iteration is active.
	consumed int           // number of events consumed by the active iterator.
	total    int           // total number of the ordering's events in the sender queue.
}

// disposable reports whether the ordering has no remaining events and can be
// safely removed from the sender.
//
// The owning Sender's mutex must be held when calling this method.
func (o *ordering) disposable() bool {
	seq := o.queue.sequence
	return seq.expected == seq.next && o.total == 0 && o.consumed == 0
}

// orderingQueue buffers events processed out of creation order for one
// ordering key.
//
// An event is buffered until all earlier-created events have been processed.
// It is then either appended to the sender queue or discarded. Buffered events
// are stored from newest to oldest.
type orderingQueue struct {
	events   []*Event // buffered events, newest first.
	sequence struct {
		expected int // sequence number expected to be processed next.
		next     int // next sequence number to assign to a newly created event for this ordering.
	}
}

// enqueue adds an event to the ordering queue or passes it to forward when
// possible.
//
// If the event has the expected sequence number, enqueue passes it and any
// queued events that become unblocked to forward. Discarded events are not
// forwarded, but they still advance the expected sequence and may unblock
// later events.
//
// If the event is out of order, enqueue inserts it into the ordering queue
// until all earlier sequence numbers have been processed.
//
// If the sender is closed, forward returns false and enqueue returns
// immediately.
//
// The owning Sender's mutex must be held when calling this method.
func (q *orderingQueue) enqueue(event *Event, forward func(event *Event) bool) {
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

// next returns the next sequence number for this ordering.
// It is called once for each new event with the same ordering key.
//
// The owning Sender's mutex must be held when calling this method.
func (q *orderingQueue) next() int {
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

// orderingsPool is shared by all senders and reuses orderings to reduce
// allocations and GC pressure.
var orderingsPool orderingPool

// orderingPool stores orderings for reuse by senders.
type orderingPool struct {
	p sync.Pool
}

// Get returns an ordering from the pool, allocating a new one if necessary.
//
// The returned ordering is reset for reuse while preserving its event buffer.
func (p *orderingPool) Get() *ordering {
	v := p.p.Get()
	if v == nil {
		return new(ordering)
	}
	o := v.(*ordering)
	events := o.queue.events[:0]
	*o = ordering{}
	o.queue.events = events
	return o
}

// Put returns an ordering to the pool for reuse.
//
// The caller must ensure that the ordering is no longer in use.
func (p *orderingPool) Put(o *ordering) {
	p.p.Put(o)
}
