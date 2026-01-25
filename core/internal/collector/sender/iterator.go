// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"iter"
	"time"

	"github.com/meergo/meergo/connectors"
)

// iterator implements the connectors.Events interface to iterate over a
// sequence of events. Only one iterator reads from the queue at a time.
type iterator struct {
	sender *Sender
	index  int // read index in sender.events, set by the sender

	sameUser struct {
		enabled bool  // true if events must belong to the same user
		user    *user // user to match; set by the sender when enabled
	}

	numConsumed int // number of consumed events

	consumed   bool // true if at least one event has been consumed
	iterating  bool // true while iterating over events
	firstEvent bool // true if the current event is the first in the sequence
	postponed  bool // true if the last consumed event was postponed
	discarded  bool // true if the last consumed event was discarded
}

// newIterator returns a new iterator.
func newIterator(s *Sender) *iterator {
	it := iterator{sender: s}
	return &it
}

func (it *iterator) All() iter.Seq[*connectors.Event] {
	if it.consumed {
		panic(it.sender.connector + " connector: SendEvents method called Events.All after the events were consumed")
	}
	it.consumed = true
	return it.seq()
}

func (it *iterator) Discard(err error) {
	if !it.iterating {
		panic(it.sender.connector + " connector: SendEvents method called Events.Discard outside an iteration")
	}
	if it.postponed {
		panic(it.sender.connector + " connector: SendEvents method called Events.Discard on a postponed event")
	}
	if it.discarded {
		panic(it.sender.connector + " connector: SendEvents method called Events.Discard on a discarded event")
	}
	if err == nil {
		panic(it.sender.connector + " connector: SendEvents method called Events.Discard passing a nil error")
	}
	trace("iterator.Discard: iterator %p discarded an event with error %q\n", it, err)
	it.discarded = true
	it.sender.discard(err)
}

func (it *iterator) First() *connectors.Event {
	if it.consumed {
		panic(it.sender.connector + " connector: SendEvents method called Events.First after the events were consumed")
	}
	it.consumed = true
	trace("iterator.First: iterator %p reads only the first event\n", it)
	event, ok := it.sender.read(true)
	it.sender.complete()
	if !ok {
		panic("core/events/collector/sender: iterator has called Sender.read, but no events are available")
	}
	return &event.Event
}

func (it *iterator) Peek() (*connectors.Event, bool) {
	if it.consumed && !it.iterating {
		panic(it.sender.connector + " connector: SendEvents method called Events.Peek outside of an iteration")
	}
	trace("iterator.Peek: iterator %p peeked an event\n", it)
	event, ok := it.sender.read(false)
	if !ok {
		return nil, false
	}
	return &event.Event, true
}

func (it *iterator) Postpone() {
	if !it.iterating {
		panic(it.sender.connector + " connector: SendEvents method called Events.Postpone outside an iteration")
	}
	if it.discarded {
		panic(it.sender.connector + " connector: SendEvents method called Events.Postpone on a discarded event")
	}
	if it.postponed {
		return
	}
	if it.firstEvent {
		panic(it.sender.connector + " connector: SendEvents method called Events.Postpone on the first event")
	}
	trace("iterator.Postpone: iterator %p postponed an event\n", it)
	it.postponed = true
	it.sender.postpone()
}

func (it *iterator) SameUser() iter.Seq[*connectors.Event] {
	if it.consumed {
		panic(it.sender.connector + " connector: SendEvents method called Events.SameUser after the events were consumed")
	}
	it.consumed = true
	it.sameUser.enabled = true
	return it.seq()
}

// seq returns a sequence of events.
func (it *iterator) seq() iter.Seq[*connectors.Event] {
	return func(yield func(event *connectors.Event) bool) {
		if it.sameUser.enabled {
			trace("iterator.seq: iterator %p starting to read the events of a single user\n", it)
		} else {
			trace("iterator.seq: iterator %p starting to read events of all users", it)
		}
		it.iterating = true
		it.firstEvent = true
		for {
			it.postponed = false
			it.discarded = false
			e, ok := it.sender.read(true)
			if !ok {
				trace("iterator.seq: iterator %p finished reading the events; no more are available\n", it)
				break
			}
			ok = yield(&e.Event)
			if !it.postponed {
				wait := time.Since(e.EnqueuedAt).Seconds()
				it.sender.prometheus.queueWait.Observe(wait)
			}
			if !ok {
				trace("iterator.seq: iterator %p broke out of the loop while reading events\n", it)
				break
			}
			it.firstEvent = false
		}
		it.sender.prometheus.queueWait.Consolidate()
		it.iterating = false
		it.firstEvent = false
		it.sender.complete()
	}
}
