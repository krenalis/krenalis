// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/collector/sender"
	"github.com/meergo/meergo/core/internal/events"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/tools/types"
)

// minQueuedEventSize is the minimum number of events in the queue required to
// trigger a new transformation.
const minQueuedEventSize = 100

// maxQueuedEventTime is the maximum time an event can stay in the queue before
// being transformed.
const maxQueuedEventTime = 200 * time.Millisecond

// destinationPipeline represents an active destination pipeline on events,
// optionally with an associated transformation.
//
// All fields are read-only. If the corresponding pipeline is modified and any
// of the fields need to change, a new instance is created with the updated
// values.
type destinationPipeline struct {
	id             int                       // ID of the pipeline
	eventType      string                    // type of event the pipeline handles
	filter         *state.Where              // filter applied to incoming events
	schema         types.Type                // schema of the event type.
	transformation state.Transformation      // transformation applied to events
	transformer    *transformers.Transformer // transformer; nil if no transformation is set

	// Queue of events to be transformed.
	// If a pipeline had a transformation but no longer does, new events are
	// sent directly to the sender, bypassing the queue. However, events already
	// in the queue are still transformed before being passed to the sender.
	queue *destinationPipelineQueue
}

// queuedEvent represents a queued event.
type queuedEvent struct {
	eventsEvent events.Event
	senderEvent *sender.Event
}

// destinationPipelineQueue holds the events of a pipeline that need to be
// transformed. Each destinationPipeline instance has its own
// destinationPipelineQueue, even if the pipeline has no transformation.
type destinationPipelineQueue struct {
	metrics *metrics.Collector // metrics collector
	sender  *sender.Sender     // sender associated with the connection
	ctx     context.Context
	cancel  context.CancelCauseFunc

	mu     sync.Mutex
	events []queuedEvent // events to be transformed; protected by mu
	timer  *time.Timer   // timer to trigger transformation
}

// newDestinationPipeline returns a new destination pipeline for the provided
// pipeline with the provided schema, provider, and queue.
func newDestinationPipeline(pipeline *state.Pipeline, schema types.Type, provider transformers.FunctionProvider, queue *destinationPipelineQueue) *destinationPipeline {
	da := &destinationPipeline{
		id:             pipeline.ID,
		eventType:      pipeline.EventType,
		filter:         pipeline.Filter,
		schema:         schema,
		transformation: pipeline.Transformation,
		queue:          queue,
	}
	if t := da.transformation; t.Mapping != nil || t.Function != nil {
		da.transformer, _ = transformers.New(pipeline, provider, nil)
	}
	return da
}

// Discard discards all events in the queue and cancels any ongoing
// transformations, passing the provided error as the cancellation cause.
// It is called when the associated pipeline is disabled or deleted.
func (da *destinationPipeline) Discard(cause error) {
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

// QueueEvent queues an event for the pipeline.
//
// If the pipeline has a transformation, the event is transformed before being
// queued.
func (da *destinationPipeline) QueueEvent(event events.Event) {
	se := da.queue.sender.CreateEvent(da.id, da.eventType, da.schema, event)
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
func (q *destinationPipelineQueue) resetTimerLocked() {
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
func (da *destinationPipeline) transform() {

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
		records[i].Attributes = events[i].eventsEvent
	}

	// Transform the events.
	err := da.transformer.Transform(da.queue.ctx, records)
	if err != nil {
		for i := 0; i < n; i++ {
			da.queue.sender.DiscardEvent(events[i].senderEvent)
		}
		var msg string
		if _, ok := err.(transformers.FunctionExecError); ok {
			msg = err.Error()
		} else if da.queue.ctx.Err() != nil {
			msg = context.Cause(da.queue.ctx).Error()
		} else {
			msg = "an internal error has occurred"
			slog.Error("core/events/collector: cannot transform events", "pipeline", da.id, "err", err)
		}
		da.queue.metrics.TransformationFailed(da.id, n, msg)
		return
	}

	for i, record := range records {
		if err := record.Err; err != nil {
			da.queue.sender.DiscardEvent(events[i].senderEvent)
			switch err.(type) {
			case transformers.RecordTransformationError:
				da.queue.metrics.TransformationFailed(da.id, 1, err.Error())
			case transformers.RecordValidationError:
				da.queue.metrics.TransformationPassed(da.id, 1)
				da.queue.metrics.OutputValidationFailed(da.id, 1, err.Error())
			}
			continue
		}
		da.queue.metrics.TransformationPassed(da.id, 1)
		da.queue.metrics.OutputValidationPassed(da.id, 1)
		events[i].senderEvent.Type.Values = record.Attributes
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
