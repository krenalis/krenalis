//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"context"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
	"github.com/open2b/chichi/backoff"
)

const debug = false

type dispatchingEvent struct {
	*events.Event
	connection int
	action     *state.Action
	request    *connectors.EventRequest
	err        error
}

type Dispatcher struct {
	events struct {
		in  chan *dispatchingEvent
		out chan *dispatchingEvent
	}
	sent        chan *dispatchingEvent
	results     chan Result
	db          *postgres.DB
	state       *state.State
	processor   *processor
	stopSenders chan<- struct{}
}

type Result struct {
	Event  *events.Event
	Action int
	Sent   bool
}

// New returns new dispatcher.
func New(db *postgres.DB, st *state.State, transformer transformers.Function, connectors *connectors.Connectors) (*Dispatcher, error) {

	processor, err := newProcessor(db, st, connectors, transformer)
	if err != nil {
		return nil, err
	}

	dispatcher := &Dispatcher{
		db:        db,
		state:     st,
		sent:      make(chan *dispatchingEvent, cap(processor.events.out)),
		results:   make(chan Result, cap(processor.events.out)),
		processor: processor,
	}
	dispatcher.events.in = processor.events.out
	dispatcher.events.out = make(chan *dispatchingEvent, cap(processor.events.out))

	dispatcher.stopSenders = startSenders(dispatcher.events.out, dispatcher.sent, connectors)

	go dispatcher.dispatch()

	return dispatcher, nil
}

// Close closes the dispatcher.
func (d *Dispatcher) Close() {
	d.processor.Close()
	close(d.stopSenders)
}

// Dispatch dispatches an event to a destination action.
func (d *Dispatcher) Dispatch(event *events.Event, action *state.Action) {
	ev := &dispatchingEvent{Event: event, action: action}
	d.processor.events.in <- ev
}

// Results returns a channel from which to read the result of event dispatch.
func (d *Dispatcher) Results() <-chan Result {
	return d.results
}

// queueKey represents a key in queues map.
type queueKey struct {
	destination int // destination connection.
	endpoint    int // endpoint.
}

// dispatch dispatches the events. It is called in its own goroutine by the
// newDispatcher function.
func (d *Dispatcher) dispatch() {

	// a queue for each action.
	queues := map[queueKey]*queue{}
	readyQueues := map[queueKey]*queue{}

	var sendingEvent *dispatchingEvent

	var pop = make(chan struct{}, 1)
	var send chan<- *dispatchingEvent
	var ready = make(chan *queue, 1)

	// endpoints maps request endpoints to their integer representations.
	endpoints := endpoints{}

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
					slog.Debug("dispatcher: events channel closed")
				}
				if numEvents == 0 {
					break dispatch
				}
				d.events.in = nil // don't receive events anymore
				continue
			}
			if debug {
				slog.Debug("dispatcher: receive event", "id", hex.EncodeToString(event.Event.Id[:]))
			}
			// push.
			key := queueKey{
				destination: event.connection,
				endpoint:    endpoints.int(event.request.Endpoint),
			}
			q, ok := queues[key]
			if !ok {
				q = newQueue(key.destination, key.endpoint)
				queues[key] = q
			}
			readyQueues[key] = q
			q.Push(event)
			numEvents++
			enablePop()

		// Pop an event to send to the senders pool.
		// pop is empty if there are no events to pop or there is a sending event.
		case <-pop:
			var event *dispatchingEvent
			for id, q := range readyQueues {
				if event = q.Pop(); event != nil {
					break
				}
				delete(readyQueues, id)
			}
			if debug {
				if event == nil {
					slog.Debug("dispatcher: no event to pop")
				} else {
					slog.Debug("dispatcher: pop event", "id", hex.EncodeToString(event.Event.Id[:]))
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
				slog.Debug("dispatcher: sent event", "id", hex.EncodeToString(sendingEvent.Event.Id[:]))
			}
			send = nil // there are no more requests to send
			sendingEvent = nil
			enablePop()

		// Receive an event from the senders pool.
		case event := <-d.sent:
			if debug {
				slog.Debug("dispatcher: receive response for event", "event", hex.EncodeToString(event.Event.Id[:]))
			}
			// ack.
			key := queueKey{
				destination: event.connection,
				endpoint:    endpoints.int(event.request.Endpoint),
			}
			q := queues[key]
			q.Ack(event, event.err == nil)
			if event.err == nil {
				if false {
					d.results <- Result{
						Event:  event.Event,
						Action: event.action.ID,
						Sent:   true,
					}
				}
				q.backoff = nil
				readyQueues[key] = q
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
				q.backoff = backoff.New(2)
				q.backoff.SetCap(10 * time.Second)
			}
			ctx := context.Background()
			q.backoff.AfterFunc(ctx, func(_ context.Context) {
				ready <- q
			})

		// Make a queue ready again.
		case queue := <-ready:
			if debug {
				endpoint := endpoints.string(queue.endpoint)
				slog.Debug("dispatcher: queue ready again", "destination", queue.destination, "endpoint", endpoint)
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
					s += hex.EncodeToString(e.Event.Id[:])
				}
				s += " "
			}
			s += "]"
			slog.Debug("dispatcher: queue",
				"ready", isReady,
				"total", len(q.events),
				"sending", len(q.sendingOffsets),
				"head", q.head,
				"pop", len(pop) > 0,
				"content", s)
		}

	}

	close(d.events.out)

	if debug {
		slog.Debug("dispatcher: exited")
	}

}

// endpoints is a mapping from event request endpoints to their corresponding
// integer used in queues.
type endpoints map[string]int

func (endpoints endpoints) int(endpoint string) int {
	if endpoint == "" {
		return 0
	}
	id, ok := endpoints[endpoint]
	if !ok {
		id = len(endpoints) + 1
		endpoints[endpoint] = id
	}
	return id
}

func (endpoints endpoints) string(id int) string {
	if id == 0 {
		return ""
	}
	for s, p := range endpoints {
		if p == id {
			return s
		}
	}
	panic("unexpected endpoint")
}
