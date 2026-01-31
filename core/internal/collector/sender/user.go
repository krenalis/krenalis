// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"cmp"
	"slices"
)

// user represents the per-user state used during event processing.
type user struct {
	queue    userQueue // per-user queue holding out-of-order events.
	iterator *iterator // iterator over the user's events; nil when no iteration is active.
	consumed int       // number of events already consumed from iterator, if any.
	totals   int       // total number of the user's events in the sender queue.
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
func (q *userQueue) enqueue(event *Event, forward func(event *Event)) {
	if event.sequence != q.sequence.expected {
		i, _ := slices.BinarySearchFunc(q.events, event, func(a, b *Event) int {
			return cmp.Compare(b.sequence, a.sequence)
		})
		q.events = slices.Insert(q.events, i, event)
		return
	}
	if !event.enqueuedAt.IsZero() {
		forward(event)
	}
	expected := q.sequence.expected + 1
	for len(q.events) > 0 {
		last := len(q.events) - 1
		if q.events[last].sequence != expected {
			break
		}
		// Append the event to the ready queue unless it has been discarded.
		if !q.events[last].enqueuedAt.IsZero() {
			forward(q.events[last])
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
	return next
}
