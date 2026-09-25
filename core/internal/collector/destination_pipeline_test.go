// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/collector/sender"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/types"
)

// countingAck records how many times it is acknowledged.
type countingAck struct {
	count atomic.Int64
}

// Acknowledge records an acknowledgment.
func (a *countingAck) Acknowledge() {
	a.count.Add(1)
}

// testPipelineMetrics discards metrics recorded by destination pipeline tests.
type testPipelineMetrics struct{}

// TransformationFailed discards a failed transformation metric.
func (*testPipelineMetrics) TransformationFailed(string, int, string) {}

// TransformationPassed discards a successful transformation metric.
func (*testPipelineMetrics) TransformationPassed(string, int) {}

// OutputValidationFailed discards a failed output validation metric.
func (*testPipelineMetrics) OutputValidationFailed(string, int, string) {}

// OutputValidationPassed discards a successful output validation metric.
func (*testPipelineMetrics) OutputValidationPassed(string, int) {}

// testDestinationApplication records events delivered by a sender.
type testDestinationApplication struct {
	delivered chan string
}

// ID returns the test connection ID.
func (*testDestinationApplication) ID() string {
	return "test-connection"
}

// Connector returns the test connector name.
func (*testDestinationApplication) Connector() string {
	return "test"
}

// WaitTime returns no additional delay for test deliveries.
func (*testDestinationApplication) WaitTime(string) (time.Duration, error) {
	return 0, nil
}

// SendEvents records every event consumed by the test application.
func (a *testDestinationApplication) SendEvents(_ context.Context, events connectors.Events) error {
	for event := range events.All() {
		a.delivered <- event.Received.MessageID()
	}
	return nil
}

// TestDestinationsQueueEventAcknowledgesMissingPipelines verifies that an
// event's destinations are acknowledged when none of their pipelines exists.
func TestDestinationsQueueEventAcknowledgesMissingPipelines(t *testing.T) {
	first := new(countingAck)
	second := new(countingAck)
	d := destinations{
		pipelines: map[string]destinationPipelines{
			"connection": nil,
		},
	}
	d.QueueEvent("connection", streams.Event{
		Destinations: []streams.Destination{
			{ID: "missing-1", Ack: first},
			{ID: "missing-2", Ack: second},
		},
	})

	if got := first.count.Load(); got != 1 {
		t.Fatalf("expected 1 acknowledgment for the first missing pipeline, got %d", got)
	}
	if got := second.count.Load(); got != 1 {
		t.Fatalf("expected 1 acknowledgment for the second missing pipeline, got %d", got)
	}
}

// TestDestinationPipelineQueueEventAfterCloseAcknowledges verifies that a
// closed pipeline completes an event without creating sender state for it.
func TestDestinationPipelineQueueEventAfterCloseAcknowledges(t *testing.T) {
	original := new(countingAck)
	q := &destinationPipelineQueue{}
	q.close.closed = true
	dp := &destinationPipeline{queue: q}

	dp.QueueEvent(nil, original)

	if got := original.count.Load(); got != 1 {
		t.Fatalf("expected 1 acknowledgment from the closed pipeline, got %d", got)
	}
}

// TestDestinationPipelineCloseCompletesQueuedEvent verifies that closing a
// pipeline does not panic, completes its queued event, and preserves sender
// ordering for the following event of the same user.
func TestDestinationPipelineCloseCompletesQueuedEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := &testDestinationApplication{delivered: make(chan string, 1)}
		s := sender.New(app, nil)
		defer s.Close(t.Context())

		ctx, cancel := context.WithCancelCause(context.Background())
		q := &destinationPipelineQueue{
			metrics: new(testPipelineMetrics),
			sender:  s,
			timer:   newStoppedTimer(),
		}
		q.cond = sync.NewCond(&q.mu)
		q.close.ctx = ctx
		q.close.cancel = cancel
		dp := &destinationPipeline{
			id:          "pipeline",
			eventType:   "event",
			transformer: new(transformers.Transformer),
			queue:       q,
		}

		missingAck := new(countingAck)
		queuedAck := new(countingAck)
		first := streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user",
				"messageId":   "first",
			},
			Destinations: []streams.Destination{
				{ID: "missing", Ack: missingAck},
				{ID: dp.id, Ack: queuedAck},
			},
		}
		d := destinations{
			pipelines: map[string]destinationPipelines{
				"connection": {dp},
			},
		}
		d.QueueEvent("connection", first)

		if got := missingAck.count.Load(); got != 1 {
			t.Fatalf("expected 1 acknowledgment for the missing pipeline, got %d", got)
		}
		if got := queuedAck.count.Load(); got != 0 {
			t.Fatalf("expected 0 acknowledgments while the queued pipeline is pending, got %d", got)
		}

		dp.Close(errors.New("pipeline has been disabled"))

		if got := queuedAck.count.Load(); got != 1 {
			t.Fatalf("expected 1 acknowledgment for the queued event, got %d", got)
		}
		if got := len(q.events); got != 0 {
			t.Fatalf("expected 0 queued events after close, got %d", got)
		}

		second := map[string]any{
			"anonymousId": "user",
			"messageId":   "second",
		}
		s.SendEvent(s.CreateEvent(dp.id, dp.eventType, dp.orderingGroup, types.Type{}, second, new(countingAck)))

		select {
		case got := <-app.delivered:
			if got != "second" {
				t.Fatalf("expected delivered event %q, got %q", "second", got)
			}
		case <-time.After(time.Second):
			t.Fatal("expected the following event to be delivered, got timeout")
		}

		dp.Close(errors.New("pipeline has been disabled"))
		if got := queuedAck.count.Load(); got != 1 {
			t.Fatalf("expected 1 acknowledgment after closing twice, got %d", got)
		}
	})
}

// TestDestinationPipelineCloseUnblocksQueueEvent verifies that closing a full
// pipeline queue completes a waiting event without leaving a sender sequence
// gap.
func TestDestinationPipelineCloseUnblocksQueueEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		app := &testDestinationApplication{delivered: make(chan string, 1)}
		s := sender.New(app, nil)
		defer s.Close(t.Context())

		ctx, cancel := context.WithCancelCause(context.Background())
		q := &destinationPipelineQueue{
			metrics: new(testPipelineMetrics),
			sender:  s,
			timer:   newStoppedTimer(),
		}
		q.cond = sync.NewCond(&q.mu)
		q.close.ctx = ctx
		q.close.cancel = cancel
		dp := &destinationPipeline{
			id:          "full-pipeline",
			eventType:   "event",
			transformer: new(transformers.Transformer),
			queue:       q,
		}

		for i := range maxQueuedEventSize {
			dp.QueueEvent(map[string]any{
				"anonymousId": "user",
				"messageId":   fmt.Sprintf("queued-%d", i),
			}, new(countingAck))
		}

		waitingAck := new(countingAck)
		go dp.QueueEvent(map[string]any{
			"anonymousId": "user",
			"messageId":   "waiting",
		}, waitingAck)
		synctest.Wait()
		if got := waitingAck.count.Load(); got != 0 {
			t.Fatalf("expected 0 acknowledgments before closing the full queue, got %d", got)
		}

		dp.Close(errors.New("pipeline has been disabled"))
		synctest.Wait()
		if got := waitingAck.count.Load(); got != 1 {
			t.Fatalf("expected 1 acknowledgment after closing the full queue, got %d", got)
		}

		following := map[string]any{
			"anonymousId": "user",
			"messageId":   "following",
		}
		s.SendEvent(s.CreateEvent(dp.id, dp.eventType, dp.orderingGroup, types.Type{}, following, new(countingAck)))

		select {
		case got := <-app.delivered:
			if got != "following" {
				t.Fatalf("expected delivered event %q, got %q", "following", got)
			}
		case <-time.After(time.Second):
			t.Fatal("expected the following event to be delivered, got timeout")
		}
	})
}
