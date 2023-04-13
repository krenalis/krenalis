//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"github.com/cenkalti/backoff/v4"
)

// dispatcherQueue represents an event dispatcher warehouseQueue.
type dispatcherQueue struct {
	destination    int               // destination connection.
	endpoint       int               // endpoint.
	head           int               // offset of the head of the warehouseQueue.
	events         []*processedEvent // events in the warehouseQueue.
	sendingOffsets map[string]int    // maps the anonymousId of a sending event to its offset in the warehouseQueue.
	backoff        backoff.BackOff   // backoff policy for retries.
}

// newDispatcherQueue returns a new empty dispatcher warehouseQueue for the given
// destination connection and action's endpoint.
func newDispatcherQueue(destination, endpoint int) *dispatcherQueue {
	return &dispatcherQueue{
		destination:    destination,
		endpoint:       endpoint,
		events:         []*processedEvent{},
		sendingOffsets: map[string]int{},
	}
}

// Ack acks an event in the warehouseQueue.
func (q *dispatcherQueue) Ack(event *processedEvent, remove bool) {
	off := q.sendingOffsets[event.AnonymousID]
	delete(q.sendingOffsets, event.AnonymousID)
	if !remove {
		q.head = min(q.head, off)
		return
	}
	q.events[off] = nil
	q.head = min(q.head, off+1)
	q.compact()
}

// Len returns the length of q.
func (q *dispatcherQueue) Len() int {
	return len(q.events) - q.head
}

// Pop pops an event from the dispatcherQueue.
// It panics if the state of q is not ready.
func (q *dispatcherQueue) Pop() *processedEvent {
	for i := q.head; i < len(q.events); i++ {
		event := q.events[i]
		if event == nil {
			continue
		}
		if _, ok := q.sendingOffsets[event.AnonymousID]; !ok {
			q.sendingOffsets[event.AnonymousID] = i
			q.head = i + 1
			return event
		}
	}
	q.head = len(q.events)
	return nil
}

// Push pushes an event into the dispatcherQueue.
func (q *dispatcherQueue) Push(event *processedEvent) {
	q.events = append(q.events, event)
	return
}

// compact compacts the dispatcherQueue.
func (q *dispatcherQueue) compact() {
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
