//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"chichi/apis/events/pipe"

	"github.com/cenkalti/backoff/v4"
)

// queue represents an event queue.
type queue struct {
	id             queueID         // identifier.
	head           int             // offset of the head of the queue.
	events         []*pipe.Event   // events in the queue.
	sendingOffsets map[string]int  // maps the anonymousId of a sending event to its offset in the queue.
	backoff        backoff.BackOff // backoff policy for retries.
}

// newQueue returns a new empty queue.
func newQueue(id queueID) *queue {
	return &queue{
		id:             id,
		events:         []*pipe.Event{},
		sendingOffsets: map[string]int{},
	}
}

// Ack acks an event in the queue.
func (q *queue) Ack(event *pipe.Event, remove bool) {
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
func (q *queue) Pop() *pipe.Event {
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
func (q *queue) Push(event *pipe.Event) {
	q.events = append(q.events, event)
	return
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
