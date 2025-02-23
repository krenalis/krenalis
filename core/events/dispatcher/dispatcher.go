//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package dispatcher

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"

	"github.com/google/uuid"
)

const debug = false

type dispatchingEvent struct {
	id          uuid.UUID
	connection  int // source connection.
	anonymousId string
	properties  map[string]any
	action      *state.Action // destination action.
	request     *meergo.EventRequest
	err         error
}

type Dispatcher struct {
	events struct {
		in  chan *dispatchingEvent
		out chan *dispatchingEvent
	}
	sent           chan *dispatchingEvent
	db             *db.DB
	state          *state.State
	processor      *processor
	operationStore events.OperationStore
	metrics        *metrics.Collector
	stopSenders    chan<- struct{}
}

// New returns new dispatcher.
func New(db *db.DB, st *state.State, opStore events.OperationStore, provider transformers.FunctionProvider, connectors *connectors.Connectors, metrics *metrics.Collector) (*Dispatcher, error) {

	processor, err := newProcessor(db, st, opStore, connectors, provider, metrics)
	if err != nil {
		return nil, err
	}

	dispatcher := &Dispatcher{
		db:             db,
		state:          st,
		sent:           make(chan *dispatchingEvent, cap(processor.events.out)),
		processor:      processor,
		operationStore: opStore,
		metrics:        metrics,
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

var errTooManyEvents = errors.New("too many events")

// Dispatch dispatches an event to a destination action.
// Returns errTooManyEvents if there are already too many events queued for
// dispatch.
func (d *Dispatcher) Dispatch(event events.Event, action *state.Action) error {
	meergoMetrics.Increment("Dispatcher.Dispatch.calls", 1)
	ev := &dispatchingEvent{
		id:          uuid.MustParse(event["id"].(string)),
		connection:  event["connection"].(int),
		anonymousId: event["anonymousId"].(string),
		properties:  event,
		action:      action,
	}
	select {
	case d.processor.events.in <- ev:
		meergoMetrics.Increment("Dispatcher.processor.written_event_in_events_in_channel", 1)
	default:
		meergoMetrics.Increment("Dispatcher.Dispatch.returned_errTooManyEvents'", 1)
		return errTooManyEvents
	}
	return nil
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
			meergoMetrics.Increment("Dispatcher.dispatch.new_events_received", 1)
			if debug {
				slog.Debug("dispatcher: receive event", "id", event.id)
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
					slog.Debug("dispatcher: pop event", "id", event.id)
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
				slog.Debug("dispatcher: sent event", "id", sendingEvent.id)
			}
			send = nil // there are no more requests to send
			sendingEvent = nil
			enablePop()

		// Receive an event from the senders pool.
		case event := <-d.sent:
			if debug {
				slog.Debug("dispatcher: receive response for event", "event", event.id)
			}
			// ack.
			key := queueKey{
				destination: event.connection,
				endpoint:    endpoints.int(event.request.Endpoint),
			}
			q := queues[key]
			q.Ack(event, event.err == nil)
			if event.err == nil {
				d.operationStore.Done(events.DoneEvent{Action: event.action.ID, ID: event.id.String()})
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
					s += e.id.String()
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
