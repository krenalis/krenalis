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
	"github.com/krenalis/krenalis/core/internal/metrics"
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
	streamEvent streams.Event
	senderEvent *sender.Event
}

// destinationPipelineQueue holds the events of a pipeline that need to be
// transformed. Each destinationPipeline instance has its own
// destinationPipelineQueue, even if the pipeline has no transformation.
type destinationPipelineQueue struct {
	metrics *metrics.Collector // metrics collector
	sender  *sender.Sender     // sender associated with the connection

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
		dp.transformer, _ = transformers.New(pipeline, provider, nil)
	}
	return dp
}

// Close closes dp by discarding all queued events and canceling any in-progress
// transformations, using the provided error as the cancellation cause.
// It is called when the associated pipeline is disabled or deleted.
func (dp *destinationPipeline) Close(cause error) {
	dp.queue.mu.Lock()
	dp.queue.close.closed = true
	if len(dp.queue.events) > 0 {
		dp.queue.metrics.TransformationFailed(dp.id, len(dp.queue.events), cause.Error())
	}
	clear(dp.queue.events)
	dp.queue.resetTimerLocked()
	dp.queue.cond.Broadcast()
	dp.queue.close.cancel(cause)
	dp.queue.mu.Unlock()
}

// QueueEvent queues an event for the pipeline.
//
// If the pipeline has a transformation, the event is transformed before being
// queued.
func (dp *destinationPipeline) QueueEvent(event streams.Event) {
	se := dp.queue.sender.CreateEvent(dp.id, dp.eventType, dp.schema, event)
	if dp.transformer == nil {
		dp.queue.metrics.TransformationPassed(dp.id, 1)
		dp.queue.metrics.OutputValidationPassed(dp.id, 1)
		dp.queue.sender.SendEvent(se)
		return
	}
	dp.queue.mu.Lock()
	for !dp.queue.close.closed && len(dp.queue.events) >= maxQueuedEventSize {
		dp.queue.cond.Wait()
	}
	if dp.queue.close.closed {
		dp.queue.mu.Unlock()
		return
	}
	dp.queue.events = append(dp.queue.events, queuedEvent{streamEvent: event, senderEvent: se})
	n := len(dp.queue.events)
	if n == 1 || n == minQueuedEventSize {
		dp.queue.resetTimerLocked()
	}
	dp.queue.mu.Unlock()
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
		records[i].Attributes = events[i].streamEvent.Attributes
	}

	// Transform the events.
	err := dp.transformer.Transform(dp.queue.close.ctx, records)
	if err != nil {
		for i := range n {
			dp.queue.sender.DiscardEvent(events[i].senderEvent)
			events[i].streamEvent.Ack()
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
			dp.queue.sender.DiscardEvent(events[i].senderEvent)
			events[i].streamEvent.Ack()
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
