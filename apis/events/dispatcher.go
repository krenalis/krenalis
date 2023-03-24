//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
)

const debug = false

type Dispatcher struct {
	events struct {
		in  <-chan *processedEvent
		out chan *processedEvent
	}
	done     chan *processedEvent
	eventLog *eventsLog
}

// newDispatcher returns a dispatcher that reads the events from the events
// channel.
func newDispatcher(eventLog *eventsLog, events <-chan *processedEvent) *Dispatcher {
	dispatcher := &Dispatcher{
		eventLog: eventLog,
		done:     make(chan *processedEvent, cap(events)),
	}
	dispatcher.events.in = events
	dispatcher.events.out = make(chan *processedEvent, cap(events))
	go dispatcher.dispatch()
	return dispatcher
}

// Events returns the events channel.
func (d *Dispatcher) Events() <-chan *processedEvent {
	return d.events.out
}

// Done returns the done channel.
func (d *Dispatcher) Done() chan<- *processedEvent {
	return d.done
}

// queueKey represents a key in queues map.
type queueKey struct {
	destination int // destination connection.
	endpoint    int // action's endpoint.
}

// dispatch dispatches the events. It is called in its own goroutine by the newEventsLog
// function.
func (d *Dispatcher) dispatch() {

	// a dispatcherQueue for each action.
	queues := map[queueKey]*dispatcherQueue{}
	readyQueues := map[queueKey]*dispatcherQueue{}

	var sendingEvent *processedEvent

	var pop = make(chan struct{}, 1)
	var send chan<- *processedEvent
	var ready = make(chan *dispatcherQueue, 1)

	var numEvents int

	// enablePop enables the pop of an event from the queues.
	enablePop := func() {
		// If pop is not enabled and there is no event to send, enable it.
		if len(pop) == 0 && sendingEvent == nil {
			pop <- struct{}{}
		}
	}

dispatch:
	for {

		select {

		// Receive a new event.
		case event, ok := <-d.events.in:
			if !ok {
				if debug {
					log.Print("dispatcher: events channel closed")
				}
				if numEvents == 0 {
					break dispatch
				}
				d.events.in = nil // don't receive events anymore
				continue
			}
			if debug {
				log.Printf("dispatcher: receive event %s", event.id)
			}
			// push.
			key := queueKey{destination: event.destination, endpoint: event.endpoint}
			q, ok := queues[key]
			if !ok {
				q = newDispatcherQueue(event.destination, event.endpoint)
				queues[key] = q
			}
			readyQueues[key] = q
			q.Push(event)
			numEvents++
			enablePop()

		// Pop an event to send to the senders pool.
		// pop is empty if there are no events to pop or there is a sending event.
		case <-pop:
			var event *processedEvent
			for id, q := range readyQueues {
				if event = q.Pop(); event != nil {
					break
				}
				delete(readyQueues, id)
			}
			if debug {
				if event == nil {
					log.Print("dispatcher: no event to pop")
				} else {
					log.Printf("dispatcher: pop event %s", event.id)
				}
			}
			if event != nil {
				send = d.events.out
				sendingEvent = event
			}

		// Send an event to the senders pool.
		// send is nil if there is no sending event.
		case send <- sendingEvent:
			if debug {
				log.Printf("dispatcher: sent event %s", sendingEvent.id)
			}
			send = nil // there are no more requests to send
			sendingEvent = nil
			enablePop()

		// Receive an event from the senders pool.
		case event := <-d.done:
			if debug {
				log.Printf("dispatcher: receive response for event %s", event.id)
			}
			// ack.
			key := queueKey{destination: event.destination, endpoint: event.endpoint}
			q := queues[key]
			q.Ack(event, event.err == nil)
			if event.err == nil {
				q.backoff = nil
				readyQueues[key] = q
				d.eventLog.Delivered(event.id, event.action.ID)
				numEvents--
				if d.events.in == nil && numEvents == 0 {
					break dispatch
				}
				enablePop()
				break
			}
			event.err = nil
			delete(readyQueues, queueKey{destination: q.destination, endpoint: q.endpoint})
			if q.backoff == nil {
				q.backoff = backoff.NewExponentialBackOff()
			}
			time.AfterFunc(q.backoff.NextBackOff(), func() {
				ready <- q
			})

		// Make a dispatcherQueue ready again.
		case queue := <-ready:
			if debug {
				log.Printf("dispatcher: warehouseQueue %d-%d ready again", queue.destination, queue.endpoint)
			}
			key := queueKey{destination: queue.destination, endpoint: queue.endpoint}
			readyQueues[key] = queue
			enablePop()
		}

		if debug {
			q, isReady := readyQueues[queueKey{1, 1}]
			if !isReady {
				q = queues[queueKey{1, 1}]
			}
			s := "[ "
			for _, e := range q.events {
				if e == nil {
					s += "nil"
				} else {
					s += e.id.String()
				}
				s += " "
			}
			s += "]"
			log.Printf("dispatcher: warehouseQueue (ready: %t, total: %d, sending: %d, head: %d), pop: %t - %s", isReady, len(q.events), len(q.sendingOffsets), q.head, len(pop) > 0, s)
		}

	}

	close(d.events.out)

	if debug {
		log.Print("dispatcher: exited")
	}

}
