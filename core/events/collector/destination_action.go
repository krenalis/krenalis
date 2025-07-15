//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package collector

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/events/collector/sender"
	"github.com/meergo/meergo/core/filters"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"
)

// minQueuedEventSize is the minimum number of events in the queue required to
// trigger a new transformation.
const minQueuedEventSize = 100

// maxQueuedEventTime is the maximum time an event can stay in the queue before
// being transformed.
const maxQueuedEventTime = 200 * time.Millisecond

// destinationAction represents an active destination action on events,
// optionally with an associated transformation.
//
// All fields are read-only. If the corresponding action is modified and any
// of the fields need to change, a new instance is created with the updated
// values.
type destinationAction struct {
	id             int                       // ID of the action
	eventType      string                    // type of event the action handles
	filter         *state.Where              // filter applied to incoming events
	schema         types.Type                // schema of the event type.
	transformation state.Transformation      // transformation applied to events
	transformer    *transformers.Transformer // transformer; nil if no transformation is set

	// Queue of events to be transformed.
	// If an action had a transformation but no longer does, new events are sent
	// directly to the sender, bypassing the queue. However, events already in
	// the queue are still transformed before being passed to the sender.
	queue *destinationActionQueue
}

// queuedEvent represents a queued event.
type queuedEvent struct {
	eventsEvent events.Event
	senderEvent *sender.Event
}

// destinationActionQueue holds the events of an action that need to be
// transformed. Each destinationAction instance has its own
// destinationActionQueue, even if the action has no transformation.
type destinationActionQueue struct {
	metrics *metrics.Collector // metrics collector
	sender  *sender.Sender     // sender associated with the connection
	ctx     context.Context
	cancel  context.CancelCauseFunc

	mu     sync.Mutex
	events []queuedEvent // events to be transformed; protected by mu
	timer  *time.Timer   // timer to trigger transformation
}

// newDestinationAction returns a new destination action for the provided
// action with the provided schema, provider, and queue.
func newDestinationAction(action *state.Action, schema types.Type, provider transformers.FunctionProvider, queue *destinationActionQueue) *destinationAction {
	da := &destinationAction{
		id:             action.ID,
		eventType:      action.EventType,
		filter:         action.Filter,
		schema:         schema,
		transformation: action.Transformation,
		queue:          queue,
	}
	if t := da.transformation; t.Mapping != nil || t.Function != nil {
		da.transformer, _ = transformers.New(action, provider, nil)
	}
	return da
}

// Discard discards all events in the queue and cancels any ongoing
// transformations, passing the provided error as the cancellation cause.
// It is called when the associated action is disabled or deleted.
func (da *destinationAction) Discard(cause error) {
	da.queue.mu.Lock()
	if len(da.queue.events) > 0 {
		da.queue.metrics.TransformationFailed(da.id, len(da.queue.events), cause.Error())
	}
	clear(da.queue.events)
	da.queue.events = da.queue.events[0:0]
	da.queue.resetTimerLocked()
	da.queue.mu.Unlock()
	da.queue.cancel(cause)
}

// QueueEvent queues an event for the action.
//
// If the event does not match the action's filter, it is ignored.
// If the action has a transformation, the event is transformed before being
// queued.
func (da *destinationAction) QueueEvent(event events.Event) {
	if !filters.Applies(da.filter, event) {
		da.queue.metrics.FilterFailed(da.id, 1)
		meergoMetrics.Increment("Collector.serveEvents.discarded_user_identities", 1)
		return
	}
	da.queue.metrics.FilterPassed(da.id, 1)
	se := da.queue.sender.CreateEvent(da.id, da.eventType, da.schema, event, nil)
	if da.transformer == nil {
		da.queue.metrics.TransformationPassed(da.id, 1)
		da.queue.metrics.OutputValidationPassed(da.id, 1)
		da.queue.sender.QueueEvent(se)
		return
	}
	da.queue.mu.Lock()
	da.queue.events = append(da.queue.events, queuedEvent{eventsEvent: event, senderEvent: se})
	n := len(da.queue.events)
	if n == 1 || n == minQueuedEventSize {
		da.queue.resetTimerLocked()
	}
	da.queue.mu.Unlock()
}

// resetTimerLocked schedules the timer so that the oldest queued event is
// transformed within maxQueuedEventTime.
//
// The caller must hold q.mu.
func (q *destinationActionQueue) resetTimerLocked() {
	if len(q.events) == 0 {
		q.timer.Stop()
		return
	}
	if len(q.events) >= minQueuedEventSize {
		q.timer.Reset(0)
		return
	}
	elapsed := time.Since(q.events[0].senderEvent.CreatedAt)
	if elapsed < maxQueuedEventTime {
		q.timer.Reset(maxQueuedEventTime - elapsed)
	} else {
		q.timer.Reset(0)
	}
}

// transform transforms the queued events.
func (da *destinationAction) transform() {

	var events []queuedEvent
	da.queue.mu.Lock()
	n := min(len(da.queue.events), minQueuedEventSize)
	if n == 0 {
		da.queue.mu.Unlock()
		return
	}
	events = make([]queuedEvent, n)
	copy(events, da.queue.events[:n])
	da.queue.events = slices.Delete(da.queue.events, 0, n)
	da.queue.resetTimerLocked()
	da.queue.mu.Unlock()

	records := make([]transformers.Record, n)
	for i := 0; i < n; i++ {
		records[i].Purpose = transformers.Create
		records[i].Properties = events[i].eventsEvent
	}

	// Transform the events.
	err := da.transformer.Transform(da.queue.ctx, records)
	if err != nil {
		var msg string
		if _, ok := err.(transformers.FunctionExecError); ok {
			msg = err.Error()
		} else if da.queue.ctx.Err() != nil {
			msg = context.Cause(da.queue.ctx).Error()
		} else {
			msg = "an internal error has occurred"
			slog.Error("core/events/collector: cannot transform events", "action", da.id, "err", err)
		}
		da.queue.metrics.TransformationFailed(da.id, n, msg)
		return
	}

	for i, record := range records {
		if err := record.Err; err != nil {
			switch err.(type) {
			case transformers.RecordTransformationError:
				da.queue.metrics.TransformationFailed(da.id, 1, err.Error())
			case transformers.RecordValidationError:
				da.queue.metrics.TransformationPassed(da.id, 1)
				da.queue.metrics.OutputValidationFailed(da.id, 1, err.Error())
			}
			da.queue.sender.DiscardEvent(events[i].senderEvent)
			continue
		}
		da.queue.metrics.TransformationPassed(da.id, 1)
		da.queue.metrics.OutputValidationPassed(da.id, 1)
		events[i].senderEvent.Type.Values = record.Properties
		da.queue.sender.QueueEvent(events[i].senderEvent)
	}

}

// newStoppedTimer returns a new stopped timer.
func newStoppedTimer() *time.Timer {
	t := time.NewTimer(math.MaxInt64)
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	return t
}
