// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package testconnector

import (
	"fmt"
	"iter"
	"sync"

	"github.com/krenalis/krenalis/connectors"
)

// iterator implements the connectors.Events interface to iterate over a
// sequence of events. Only one iterator reads from the queue at a time.
type iterator struct {
	events []*connectors.Event

	index int

	sameUser struct {
		on   bool    // true if only events from the same user should be consumed
		user *string // the user to match events against when 'on' is true
	}

	consumed  bool // true if the iterator has consumed at least one event
	iterating bool // true if the iteration has started
	first     bool // true if the current event is the first one in the iteration
	postponed bool // true if the last consumed event was postponed
	discarded bool // true if the last consumed event was discarded
}

// NewEventsIterator returns a new iterator over events that can be used in
// tests.
func NewEventsIterator(events []*connectors.Event) connectors.Events {
	it := iterator{events: events}
	return &it
}

func (it *iterator) All() iter.Seq[*connectors.Event] {
	if it.consumed {
		panic("SendEvents method called Events.All after the events were consumed")
	}
	it.consumed = true
	return it.seq()
}

func (it *iterator) Discard(err error) {
	if !it.iterating {
		panic("SendEvents method called Events.Discard outside an iteration")
	}
	if it.postponed {
		panic("SendEvents method called Events.Discard on a postponed event")
	}
	if it.discarded {
		panic("SendEvents method called Events.Discard on a discarded event")
	}
	if err == nil {
		panic("SendEvents method called Events.Discard passing a nil error")
	}
	trace("iterator.Discard: iterator %p discarded an event\n", it)
	it.discarded = true
}

func (it *iterator) First() *connectors.Event {
	if it.consumed {
		panic("SendEvents method called Events.First after the events were consumed")
	}
	it.consumed = true
	trace("iterator.First: iterator %p reads only the first event\n", it)
	event, ok := it.read(true)
	if !ok {
		panic("iterator has called Sender.read, but no events are available")
	}
	return event
}

func (it *iterator) Peek() (*connectors.Event, bool) {
	if it.consumed && !it.iterating {
		panic("SendEvents method called Events.Peek outside of an iteration")
	}
	trace("iterator.Peek: iterator %p peeked an event\n", it)
	event, ok := it.read(false)
	if !ok {
		return nil, false
	}
	return event, true
}

func (it *iterator) Postpone() {
	if !it.iterating {
		panic("SendEvents method called Events.Postpone outside an iteration")
	}
	if it.postponed {
		return
	}
	if it.discarded {
		panic("SendEvents method called Events.Postpone on a discarded event")
	}
	if it.first {
		panic("SendEvents method called Events.Postpone on the first event")
	}
	trace("iterator.Postpone: iterator %p postponed an event\n", it)
	it.postponed = true
}

func (it *iterator) SameUser() iter.Seq[*connectors.Event] {
	if it.consumed {
		panic("SendEvents method called Events.SameUser after the events were consumed")
	}
	it.consumed = true
	it.sameUser.on = true
	it.sameUser.user = nil
	return it.seq()
}

// seq returns a sequence of events.
func (it *iterator) seq() iter.Seq[*connectors.Event] {
	return func(yield func(event *connectors.Event) bool) {
		if it.sameUser.on {
			trace("iterator.seq: iterator %p starting to read the events of a single user\n", it)
		} else {
			trace("iterator.seq: iterator %p starting to read events of all users", it)
		}
		it.iterating = true
		it.first = true
		for {
			it.postponed = false
			it.discarded = false
			e, ok := it.read(true)
			if !ok {
				trace("iterator.seq: iterator %p finished reading the events; no more are available\n", it)
				break
			}
			if it.sameUser.on && it.first {
				u, _ := e.Received.UserID()
				it.sameUser.user = &u
			}
			if !yield(e) {
				trace("iterator.seq: iterator %p broke out of the loop while reading events\n", it)
				break
			}
			it.first = false
		}
		it.iterating = false
		it.first = false
	}
}

func (it *iterator) read(consume bool) (*connectors.Event, bool) {
	for {
		if it.index >= len(it.events) {
			return nil, false
		}
		event := it.events[it.index]
		if it.sameUser.on && it.sameUser.user != nil && *it.sameUser.user != event.Received.AnonymousID() {
			if consume {
				it.index += 1
			}
			continue
		}
		if consume {
			it.index += 1
		}
		return event, true
	}
}

const traces = false // set to true to trace execution flow

var tracesMu sync.Mutex

// trace prints a tracing message if traces is true.
func trace(msg string, a ...any) {
	if !traces {
		return
	}
	tracesMu.Lock()
	defer tracesMu.Unlock()
	fmt.Printf(msg, a...)
}
