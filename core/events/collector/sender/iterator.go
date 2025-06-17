//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package sender

import (
	"iter"

	"github.com/meergo/meergo"
)

// iterator implements the meergo.Events interface to iterate over a sequence of events.
// Only one iterator reads from the queue at a time.
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
	skipped    bool // true if the last consumed event was skipped
}

// newIterator returns a new iterator.
func newIterator(s *Sender) *iterator {
	it := iterator{sender: s}
	return &it
}

func (it *iterator) All() iter.Seq2[int, *meergo.Event] {
	if it.consumed {
		panic(it.sender.connector + " connector: SendEvents method called Events.All after the events were consumed")
	}
	it.consumed = true
	return it.seq()
}

func (it *iterator) First() *meergo.Event {
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
	return event
}

func (it *iterator) Peek() (*meergo.Event, bool) {
	if it.consumed && !it.iterating {
		panic(it.sender.connector + " connector: SendEvents method called Events.Peek outside of an iteration")
	}
	trace("iterator.Peek: iterator %p peeked an event\n", it)
	event, ok := it.sender.read(false)
	if !ok {
		return nil, false
	}
	return event, true
}

func (it *iterator) SameUser() iter.Seq2[int, *meergo.Event] {
	if it.consumed {
		panic(it.sender.connector + " connector: SendEvents method called Events.SameUser after the events were consumed")
	}
	it.consumed = true
	it.sameUser.enabled = true
	return it.seq()
}

func (it *iterator) Skip() {
	if !it.iterating {
		panic(it.sender.connector + " connector: SendEvents method called Events.Skip outside an iteration")
	}
	if it.skipped {
		return
	}
	if it.firstEvent {
		panic(it.sender.connector + " connector: SendEvents method called Events.Skip on the first event")
	}
	trace("iterator.Skip: iterator %p skipped an event\n", it)
	it.skipped = true
	it.sender.skip()
}

// seq returns a sequence of events.
func (it *iterator) seq() iter.Seq2[int, *meergo.Event] {
	return func(yield func(i int, event *meergo.Event) bool) {
		if it.sameUser.enabled {
			trace("iterator.seq: iterator %p starting to read the events of a single user\n", it)
		} else {
			trace("iterator.seq: iterator %p starting to read events of all users", it)
		}
		it.iterating = true
		it.firstEvent = true
		i := 0
		for {
			it.skipped = false
			e, ok := it.sender.read(true)
			if !ok {
				trace("iterator.seq: iterator %p finished reading the events; no more are available\n", it)
				break
			}
			if !yield(i, e) {
				trace("iterator.seq: iterator %p broke out of the loop while reading events\n", it)
				break
			}
			if !it.skipped {
				i++
			}
			it.firstEvent = false
		}
		it.iterating = false
		it.firstEvent = false
		it.sender.complete()
	}
}
