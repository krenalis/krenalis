//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"github.com/open2b/chichi/backoff"
)

// queue represents an event queue.
type queue struct {
	destination    int                 // destination connection.
	endpoint       int                 // endpoint.
	head           int                 // offset of the head of the queue.
	events         []*dispatchingEvent // events in the queue.
	sendingOffsets map[string]int      // maps the anonymousId of a sending event to its offset in the queue.
	backoff        *backoff.Backoff    // backoff policy for retries.
}

// newQueue returns a new empty queue for the given destination connection and
// action's endpoint.
func newQueue(destination, endpoint int) *queue {
	return &queue{
		destination:    destination,
		endpoint:       endpoint,
		events:         []*dispatchingEvent{},
		sendingOffsets: map[string]int{},
	}
}

// Ack acks an event in the queue.
func (q *queue) Ack(event *dispatchingEvent, remove bool) {
	off := q.sendingOffsets[event.AnonymousId]
	delete(q.sendingOffsets, event.AnonymousId)
	if !remove {
		q.head = min(q.head, off)
		return
	}
	q.events[off] = nil
	q.head = min(q.head, off+1)
	q.compact()
}

// Len returns the length of q.
func (q *queue) Len() int {
	return len(q.events) - q.head
}

// Pop pops an event from the queue.
// It panics if the state of q is not ready.
func (q *queue) Pop() *dispatchingEvent {
	for i := q.head; i < len(q.events); i++ {
		event := q.events[i]
		if event == nil {
			continue
		}
		if _, ok := q.sendingOffsets[event.AnonymousId]; !ok {
			q.sendingOffsets[event.AnonymousId] = i
			q.head = i + 1
			return event
		}
	}
	q.head = len(q.events)
	return nil
}

// Push pushes an event into the queue.
func (q *queue) Push(event *dispatchingEvent) {
	q.events = append(q.events, event)
}

// compact compacts the queue.
func (q *queue) compact() {
	var i int
	for i = 0; i < q.head; i++ {
		if q.events[i] != nil {
			break
		}
	}
	if i > 0 {
		copy(q.events, q.events[i:])
		q.events = q.events[:len(q.events)-i]
		q.head -= i
		for id, off := range q.sendingOffsets {
			q.sendingOffsets[id] = off - i
		}
	}
}

// min returns the minimum value between x and y.
func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}
