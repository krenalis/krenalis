//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"chichi/apis/events/pipe"

	"github.com/cenkalti/backoff/v4"
)

const debug = false

// ErrDestinationDown is the error returned when a destination is down.
var ErrDestinationDown = errors.New("destination is down")

// queueID represents a queue identifier.
type queueID struct {
	connection int
	endpoint   int
}

func (id queueID) String() string {
	return fmt.Sprintf("%d-%d", id.connection, id.endpoint)
}

// Dispatch dispatches the events, received from the in Channel, to the
// returned Channel.
func Dispatch(in pipe.Channel) pipe.Channel {
	outEvents := make(chan *pipe.Event, cap(in.Events))
	outDone := make(chan *pipe.Event, cap(in.Done))
	go dispatch(in.Events, outDone, outEvents, in.Done)
	return pipe.Channel{Events: outEvents, Done: outDone}
}

// dispatch dispatches the events. It is called in its own goroutine by the
// Dispatch function.
func dispatch(inEvents, outDone <-chan *pipe.Event, outEvents, inDone chan<- *pipe.Event) {

	queues := map[queueID]*queue{}
	readyQueues := map[queueID]*queue{}

	var sendingEvent *pipe.Event

	var pop = make(chan struct{}, 1)
	var send chan<- *pipe.Event
	var ready = make(chan *queue, 1)

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
		case event, ok := <-inEvents:
			if !ok {
				if debug {
					log.Print("dispatcher: events channel closed")
				}
				if numEvents == 0 {
					break dispatch
				}
				inEvents = nil // don't receive events anymore
				continue
			}
			if debug {
				log.Printf("dispatcher: receive event %d", event.ID)
			}
			// push.
			id := queueID{connection: event.Connection, endpoint: event.Endpoint}
			q, ok := queues[id]
			if !ok {
				q = newQueue(id)
				queues[id] = q
			}
			readyQueues[id] = q
			q.Push(event)
			numEvents++
			enablePop()

		// Pop an event to send to the senders pool.
		// pop is empty if there are no events to pop or there is a sending event.
		case <-pop:
			var event *pipe.Event
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
					log.Printf("dispatcher: pop event %d", event.ID)
				}
			}
			if event != nil {
				send = outEvents
				sendingEvent = event
			}

		// Send an event to the senders pool.
		// send is nil if there is no sending event.
		case send <- sendingEvent:
			if debug {
				log.Printf("dispatcher: sent event %d", sendingEvent.ID)
			}
			send = nil // there are no more requests to send
			sendingEvent = nil
			enablePop()

		// Receive an event from the senders pool.
		case event := <-outDone:
			if debug {
				log.Printf("dispatcher: receive response for event %d", event.ID)
			}
			// ack.
			id := queueID{connection: event.Connection, endpoint: event.Endpoint}
			q := queues[id]
			q.Ack(event, event.Err == nil)
			if event.Err == nil {
				q.backoff = nil
				readyQueues[id] = q
				inDone <- event
				numEvents--
				if inEvents == nil && numEvents == 0 {
					break dispatch
				}
				enablePop()
				break
			}
			event.Err = nil
			delete(readyQueues, q.id)
			if q.backoff == nil {
				q.backoff = backoff.NewExponentialBackOff()
			}
			time.AfterFunc(q.backoff.NextBackOff(), func() {
				ready <- q
			})

		// Make a queue ready again.
		case queue := <-ready:
			if debug {
				log.Printf("dispatcher: queue %s ready again", queue.id)
			}
			readyQueues[queue.id] = queue
			enablePop()
		}

		if debug {
			id := queueID{connection: 1, endpoint: 1}
			q, isReady := readyQueues[id]
			if !isReady {
				q = queues[id]
			}
			s := "[ "
			for _, e := range q.events {
				if e == nil {
					s += "nil"
				} else {
					s += strconv.Itoa(e.ID)
				}
				s += " "
			}
			s += "]"
			log.Printf("dispatcher: queue (ready: %t, total: %d, sending: %d, head: %d), pop: %t - %s", isReady, len(q.events), len(q.sendingOffsets), q.head, len(pop) > 0, s)
		}

	}

	close(inDone)
	close(outEvents)

	if debug {
		log.Print("dispatcher: exited")
	}

}
