// Copyright 2026 Open2b. All rights reserved.
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

	"github.com/krenalis/krenalis/core/internal/collector/sender"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/types"
)

// minQueuedEventSize is the minimum number of events in the queue required to
// trigger a new transformation.
const (
	minQueuedEventSize = 100
	maxQueuedEventSize = 2 * minQueuedEventSize
)

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
	id             string                    // ID of the pipeline
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
	attributes  map[string]any
	ack         streams.Ack
	senderEvent *sender.Event
}

// destinationPipelineQueue holds the events of a pipeline that need to be
// transformed. Each destinationPipeline instance has its own
// destinationPipelineQueue, even if the pipeline has no transformation.
type destinationPipelineQueue struct {
	// metrics collector
	metrics interface {
		TransformationFailed(pipeline string, count int, message string)
		TransformationPassed(pipeline string, count int)
		OutputValidationFailed(pipeline string, count int, message string)
		OutputValidationPassed(pipeline string, count int)
	}
	sender *sender.Sender // sender associated with the connection

	mu     sync.Mutex
	cond   *sync.Cond
	events []queuedEvent // events to be transformed; protected by mu
	timer  *time.Timer   // timer to trigger transformation
	close  struct {
		closed bool
		ctx    context.Context
		cancel context.CancelCauseFunc
	}
}

// newDestinationPipeline returns a new destination pipeline for the provided
// pipeline with the provided schema, provider, and queue.
func newDestinationPipeline(pipeline *state.Pipeline, schema types.Type, provider transformers.FunctionProvider, queue *destinationPipelineQueue) *destinationPipeline {
	dp := &destinationPipeline{
		id:             pipeline.ID,
		eventType:      pipeline.EventType,
		filter:         pipeline.Filter,
		schema:         schema,
		transformation: pipeline.Transformation,
		queue:          queue,
	}
	if t := dp.transformation; t.Mapping != nil || t.Function != nil {
		dp.transformer, _ = transformers.New(pipeline.Organization().ID, pipeline, provider, nil)
	}
	return dp
}

// Close closes dp by discarding all queued events and canceling any in-progress
// transformations, using the provided error as the cancellation cause. It is
// idempotent and is called when the associated pipeline is disabled or deleted.
func (dp *destinationPipeline) Close(cause error) {
	dp.queue.mu.Lock()
	if dp.queue.close.closed {
		dp.queue.mu.Unlock()
		return
	}
	dp.queue.close.closed = true
	// Detach the queued events while holding the lock. A concurrent QueueEvent
	// either accepts its event before the pipeline is closed or observes the
	// closed state.
	events := dp.queue.events
	dp.queue.events = nil
	dp.queue.resetTimerLocked()
	dp.queue.cond.Broadcast()
	dp.queue.close.cancel(cause)
	dp.queue.mu.Unlock()

	if len(events) > 0 {
		dp.queue.metrics.TransformationFailed(dp.id, len(events), cause.Error())
		for _, event := range events {
			dp.queue.discardEvent(event)
		}
	}
}

// QueueEvent submits an event and its corresponding acknowledgment to the
// pipeline.
//
// If the pipeline has a transformation, the event is transformed before being
// passed to the sender.
func (dp *destinationPipeline) QueueEvent(attributes map[string]any, ack streams.Ack) {

	dp.queue.mu.Lock()
	if dp.queue.close.closed {
		dp.queue.mu.Unlock()
		ack.Acknowledge()
		return
	}

	if dp.transformer == nil {
		event := dp.queue.sender.CreateEvent(dp.id, dp.eventType, dp.schema, attributes, ack)
		dp.queue.mu.Unlock()
		dp.queue.metrics.TransformationPassed(dp.id, 1)
		dp.queue.metrics.OutputValidationPassed(dp.id, 1)
		dp.queue.sender.SendEvent(event)
		return
	}

	for len(dp.queue.events) >= maxQueuedEventSize && !dp.queue.close.closed {
		dp.queue.cond.Wait()
	}
	if dp.queue.close.closed {
		dp.queue.mu.Unlock()
		ack.Acknowledge()
		return
	}
	event := dp.queue.sender.CreateEvent(dp.id, dp.eventType, dp.schema, attributes, ack)
	dp.queue.events = append(dp.queue.events, queuedEvent{attributes: attributes, ack: ack, senderEvent: event})
	if n := len(dp.queue.events); n == 1 || n == minQueuedEventSize {
		dp.queue.resetTimerLocked()
	}
	dp.queue.mu.Unlock()

}

// discardEvent marks the event as discarded in the sender, allowing its
// per-user sequence to advance, before acknowledging its pipeline branch.
func (q *destinationPipelineQueue) discardEvent(event queuedEvent) {
	q.sender.DiscardEvent(event.senderEvent)
	event.ack.Acknowledge()
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
	elapsed := time.Since(q.events[0].senderEvent.CreatedAt())
	q.timer.Reset(max(0, maxQueuedEventTime-elapsed))
}

// transform transforms the queued events.
func (dp *destinationPipeline) transform() {

	var events []queuedEvent
	dp.queue.mu.Lock()
	n := min(len(dp.queue.events), minQueuedEventSize)
	if dp.queue.close.closed || n == 0 {
		dp.queue.mu.Unlock()
		return
	}
	events = make([]queuedEvent, n)
	copy(events, dp.queue.events[:n])
	dp.queue.events = slices.Delete(dp.queue.events, 0, n)
	dp.queue.resetTimerLocked()
	if len(dp.queue.events) < maxQueuedEventSize {
		dp.queue.cond.Broadcast()
	}
	dp.queue.mu.Unlock()

	records := make([]transformers.Record, n)
	for i := range n {
		records[i].Purpose = transformers.Create
		records[i].Attributes = events[i].attributes
	}

	// Transform the events.
	err := dp.transformer.Transform(dp.queue.close.ctx, records)
	if err != nil {
		for i := range n {
			dp.queue.discardEvent(events[i])
		}
		var msg string
		if _, ok := err.(transformers.FunctionExecError); ok {
			msg = err.Error()
		} else if dp.queue.close.ctx.Err() != nil {
			msg = context.Cause(dp.queue.close.ctx).Error()
		} else {
			msg = "an internal error has occurred"
			slog.Error("core/events/collector: cannot transform events", "pipeline", dp.id, "error", err)
		}
		dp.queue.metrics.TransformationFailed(dp.id, n, msg)
		return
	}

	for i, record := range records {
		if err := record.Err; err != nil {
			dp.queue.discardEvent(events[i])
			switch err.(type) {
			case transformers.RecordTransformationError:
				dp.queue.metrics.TransformationFailed(dp.id, 1, err.Error())
			case transformers.RecordValidationError:
				dp.queue.metrics.TransformationPassed(dp.id, 1)
				dp.queue.metrics.OutputValidationFailed(dp.id, 1, err.Error())
			}
			continue
		}
		dp.queue.metrics.TransformationPassed(dp.id, 1)
		dp.queue.metrics.OutputValidationPassed(dp.id, 1)
		events[i].senderEvent.Type.Values = record.Attributes
		dp.queue.sender.SendEvent(events[i].senderEvent)
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
