// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/types"
)

const (
	testConnectionID = "B2tw4V9aGM2n"
	testPipelineID   = "8QaT3mN7KxP5"
)

// testApplication is a configurable Application implementation for tests.
// It defaults to no-op behavior when hooks are not provided.
type testApplication struct {
	IDValue        string
	ConnectorValue string
	WaitTimeFunc   func(string) (time.Duration, error)
	SendEventsFunc func(context.Context, connectors.Events) error
}

func newTestApplication() *testApplication {
	return &testApplication{
		IDValue:        testConnectionID,
		ConnectorValue: "nop",
	}
}

func (a *testApplication) ID() string {
	if a.IDValue == "" {
		return testConnectionID
	}
	return a.IDValue
}

func (a *testApplication) Connector() string {
	if a.ConnectorValue == "" {
		return "nop"
	}
	return a.ConnectorValue
}

func (a *testApplication) WaitTime(pattern string) (time.Duration, error) {
	if a.WaitTimeFunc == nil {
		return 0, nil
	}
	return a.WaitTimeFunc(pattern)
}

func (a *testApplication) SendEvents(ctx context.Context, events connectors.Events) error {
	if a.SendEventsFunc == nil {
		return nil
	}
	return a.SendEventsFunc(ctx, events)
}

func Test_newStoppedTimer(t *testing.T) {
	tm := newSchedule()
	select {
	case <-tm.C:
		t.Fatal("timer should be stopped")
	default:
	}
	if tm.Stop() {
		t.Fatal("Stop should return false on an already stopped timer")
	}
}

func Test_iterator_invalidUsage(t *testing.T) {

	expectPanic := func(f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()
		f()
	}

	t.Run("PostponeOutsideIteration", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PostponeFirstEvent", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.firstEvent = true
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PostponeDiscardedEvent", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.discarded = true
		expectPanic(func() { it.Postpone() })
	})

	t.Run("DiscardDiscardedEvent", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.discarded = true
		expectPanic(func() { it.Discard(errors.New("event is invalid")) })
	})

	t.Run("DiscardPostponedEvent", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.postponed = true
		expectPanic(func() { it.Discard(errors.New("event is invalid")) })
	})

	t.Run("PeekAfterConsumed", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.Peek() })
	})

	t.Run("AllAfterConsumed", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.All() })
	})

	t.Run("FirstAfterConsumed", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.First() })
	})

	t.Run("SameUserAfterConsumed", func(t *testing.T) {
		s := New(newTestApplication(), nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.SameUser() })
	})

}

// nopAck is a no-op streams.Ack implementation.
type nopAck struct{}

// Acknowledge implements streams.Ack.
func (nopAck) Acknowledge() {}

// Test_Sender_DiscardedOutOfOrderEvent verifies that discarding an out-of-order
// event does not prevent delivering the next event exactly once.
func Test_Sender_DiscardedOutOfOrderEvent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		var consumed bool

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.All() {
				if consumed || event.Received.MessageID() != "msg-0" {
					t.Fatalf("unexpected consumed event %q", event.Received.MessageID())
				}
				consumed = true
			}
			return nil
		}
		s := New(app, nil)

		event0 := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user",
			"messageId":   "msg-0",
		}, nopAck{})
		event1 := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user",
			"messageId":   "msg-1",
		}, nopAck{})

		s.DiscardEvent(event1)
		s.SendEvent(event0)
		time.Sleep(maxQueueDelay)

		s.Close(t.Context())
		if !consumed {
			t.Fatalf("event was not consumed")
		}

	})
}

// Test_Sender_OrderingGroupPreservesOrder verifies that events for the same
// user and ordering group are delivered in creation order even when associated
// with different pipelines.
func Test_Sender_OrderingGroupPreservesOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var delivered []string

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.All() {
				delivered = append(delivered, event.Received.MessageID())
			}
			return nil
		}
		s := New(app, nil)

		newEvent := func(pipeline, messageID string) *Event {
			return s.CreateEvent(pipeline, "Click", "contacts", types.Type{}, map[string]any{
				"anonymousId": "user",
				"messageId":   messageID,
			}, nopAck{})
		}
		first := newEvent("pipeline-1", "msg-1")
		second := newEvent("pipeline-2", "msg-2")

		// Make the second event ready first. It must wait for the first event
		// because both pipelines share the same ordering group.
		s.SendEvent(second)
		s.SendEvent(first)
		time.Sleep(maxQueueDelay)
		s.Close(t.Context())

		expected := []string{"msg-1", "msg-2"}
		if !slices.Equal(delivered, expected) {
			t.Fatalf("expected delivered events %v, got %v", expected, delivered)
		}
	})
}

// Test_Sender_OrderingGroupsAreIndependent verifies that an event waiting for
// an earlier event does not block the same user's events in another group.
func Test_Sender_OrderingGroupsAreIndependent(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var delivered []string

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.All() {
				delivered = append(delivered, event.Received.MessageID())
			}
			return nil
		}
		s := New(app, nil)

		newEvent := func(group, messageID string) *Event {
			return s.CreateEvent("pipeline-"+group, group, group, types.Type{}, map[string]any{
				"anonymousId": "user",
				"messageId":   messageID,
			}, nopAck{})
		}
		contactsFirst := newEvent("contacts", "contacts-1")
		contactsSecond := newEvent("contacts", "contacts-2")
		ordersFirst := newEvent("orders", "orders-1")

		// contacts-2 waits for contacts-1, while orders-1 remains available
		// because it belongs to an independent ordering group.
		s.SendEvent(contactsSecond)
		s.SendEvent(ordersFirst)
		time.Sleep(maxQueueDelay)
		synctest.Wait()

		expected := []string{"orders-1"}
		if !slices.Equal(delivered, expected) {
			t.Fatalf("expected delivered events %v while contacts are blocked, got %v", expected, delivered)
		}

		s.DiscardEvent(contactsFirst)
		time.Sleep(maxQueueDelay)
		s.Close(t.Context())

		expected = []string{"orders-1", "contacts-2"}
		if !slices.Equal(delivered, expected) {
			t.Fatalf("expected delivered events %v after contacts are unblocked, got %v", expected, delivered)
		}
	})
}

// Test_Sender_SameUserIncludesOrderingGroups verifies that SameUser returns
// events for one user even when they belong to different ordering groups.
func Test_Sender_SameUserIncludesOrderingGroups(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var batches [][]string

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			var batch []string
			for event := range events.SameUser() {
				batch = append(batch, event.Received.MessageID())
			}
			batches = append(batches, batch)
			return nil
		}
		s := New(app, nil)

		newEvent := func(anonymousID, group, messageID string) *Event {
			return s.CreateEvent("pipeline-"+group, group, group, types.Type{}, map[string]any{
				"anonymousId": anonymousID,
				"messageId":   messageID,
			}, nopAck{})
		}
		s.SendEvent(newEvent("user-1", "contacts", "contacts-1"))
		s.SendEvent(newEvent("user-2", "contacts", "other-user"))
		s.SendEvent(newEvent("user-1", "orders", "orders-1"))

		time.Sleep(maxQueueDelay)
		synctest.Wait()
		s.Close(t.Context())

		if len(batches) == 0 {
			t.Fatal("expected at least one delivered batch, got none")
		}
		expected := []string{"contacts-1", "orders-1"}
		if !slices.Equal(batches[0], expected) {
			t.Fatalf("expected first user batch %v, got %v", expected, batches[0])
		}
	})
}

// Test_Sender_SameUserRebindPreservesOrder verifies that, after Peek skips an
// event from another user, discarding the iterator's only consumed event allows
// it to bind to the skipped event's user without consuming events out of order.
func Test_Sender_SameUserRebindPreservesOrder(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		var delivered []string
		var peeked string
		var peekedOK bool
		var unexpected []string
		done := make(chan struct{})
		discardErr := errors.New("event is invalid")

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.SameUser() {
				switch id := event.Received.MessageID(); id {
				case "w-0":
					// Peek the next event of the same user, reading past the
					// event of the other user that precedes it.
					peekedEvent, ok := events.Peek()
					peekedOK = ok
					if ok {
						peeked = peekedEvent.Received.MessageID()
					}
					events.Discard(discardErr)
				case "w-1":
					events.Discard(discardErr)
				case "u-0", "u-1":
					delivered = append(delivered, id)
					if len(delivered) == 2 {
						close(done)
					}
				default:
					unexpected = append(unexpected, id)
				}
			}
			return nil
		}
		s := New(app, nil)
		defer s.Close(t.Context())

		newEvent := func(anonymousID, messageID string) *Event {
			return s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
				"anonymousId": anonymousID,
				"messageId":   messageID,
			}, nopAck{})
		}

		// Discarding w-0 releases user-w, so the iteration binds to user-u
		// after Peek has already read past u-0 to find w-1.
		s.SendEvent(newEvent("user-w", "w-0"))
		s.SendEvent(newEvent("user-u", "u-0"))
		s.SendEvent(newEvent("user-w", "w-1"))
		s.SendEvent(newEvent("user-u", "u-1"))

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for events")
		}

		s.Close(t.Context())

		if !peekedOK {
			t.Fatal("expected Peek to return an event")
		}
		if peeked != "w-1" {
			t.Fatalf("expected Peek to return event %q, got %q", "w-1", peeked)
		}
		if len(unexpected) != 0 {
			t.Fatalf("unexpected delivered events: %v", unexpected)
		}

		want := []string{"u-0", "u-1"}
		if !slices.Equal(delivered, want) {
			t.Fatalf("expected the events delivered in order %v, got %v", want, delivered)
		}

	})
}

// Test_Sender_SequenceOverflowRescale verifies that an ordering preserves
// creation order across sequence overflow.
func Test_Sender_SequenceOverflowRescale(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var seen []string

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.All() {
				seen = append(seen, event.Received.MessageID())
			}
			return nil
		}
		s := New(app, nil)

		// Force the ordering sequence near overflow without creating a huge number
		// of events, while still asserting only on externally visible ordering.
		const userID = "user-overflow"
		s.mu.Lock()
		key := orderingKey{group: "events", anonymousID: userID}
		o := orderingsPool.Get()
		o.key = key
		o.queue.sequence.next = math.MaxInt - 1
		o.queue.sequence.expected = math.MaxInt - 1
		s.orderings[key] = o
		s.mu.Unlock()

		makeEvent := func(messageID string) *Event {
			t.Helper()
			return s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
				"anonymousId": userID,
				"messageId":   messageID,
			}, nopAck{})
		}

		e0 := makeEvent("msg-0")
		e1 := makeEvent("msg-1")
		e2 := makeEvent("msg-2")

		// Send out of order; delivery must follow creation order even across overflow.
		s.SendEvent(e1)
		s.SendEvent(e2)
		s.SendEvent(e0)

		time.Sleep(maxQueueDelay)

		s.Close(t.Context())

		want := []string{"msg-0", "msg-1", "msg-2"}
		if !slices.Equal(seen, want) {
			t.Fatalf("unexpected order: got %v, want %v", seen, want)
		}
	})
}

// Test_Sender_RetryAfterSendEventsErrorWithoutIteration verifies that a send
// error without iteration is retried and then consumes the events.
func Test_Sender_RetryAfterSendEventsErrorWithoutIteration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		var called bool
		var consumed bool

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			if !called {
				called = true
				return errors.New("an error occurred")
			}
			for event := range events.All() {
				if consumed || event.Received.MessageID() != "msg-0" {
					t.Fatalf("unexpected consumed event %q", event.Received.MessageID())
				}
				consumed = true
			}
			return nil
		}
		s := New(app, nil)

		event := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user",
			"messageId":   "msg-0",
		}, nopAck{})
		s.SendEvent(event)

		time.Sleep(maxQueueDelay)
		time.Sleep(1) // TODO(marco): remove the following line. See issue https://github.com/krenalis/krenalis/issues/2122

		s.Close(t.Context())

		if !consumed {
			t.Fatalf("event was not consumed")
		}
	})
}

// Test_Sender_MinQueuedEvents tests that the sender works correctly when
// MaxQueuedEvents is set to its minimum value (1).
func Test_Sender_MinQueuedEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		defer func(previous int) {
			MaxQueuedEvents = previous
		}(MaxQueuedEvents)
		MaxQueuedEvents = 1

		var total = 100
		var consumed int

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for range events.All() {
				consumed++
			}
			return nil
		}
		s := New(app, nil)

		// Send events.
		for i := range total {
			s.SendEvent(createTestEvent(s, i))
		}

		time.Sleep(1) // TODO(marco): remove the following line. See issue https://github.com/krenalis/krenalis/issues/2122
		synctest.Wait()
		if consumed != total {
			t.Fatalf("expected %d consumed events, got %d", total, consumed)
		}

		s.Close(context.Background())
	})

}

// Test_Sender_QueueEventBlocksWhenQueueFull verifies that QueueEvent blocks
// when the event queue is full and unblocks once space becomes available.
func Test_Sender_QueueEventBlocksWhenQueueFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		defer func(previous int) {
			MaxQueuedEvents = previous
		}(MaxQueuedEvents)
		MaxQueuedEvents = 100

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			// Consume the first event.
			events.First()
			return nil
		}
		s := New(app, nil)

		// Fill the queue up to its maximum capacity.
		for i := 0; i < MaxQueuedEvents; i++ {
			s.SendEvent(createTestEvent(s, i))
		}

		// Start a goroutine that attempts to enqueue one more event.
		// Since the queue is full, QueueEvent must block.
		var done bool
		go func() {
			s.SendEvent(createTestEvent(s, MaxQueuedEvents))
			done = true
		}()
		synctest.Wait()

		// Let one queue cycle proceed so space becomes available.
		// QueueEvent should then unblock and the goroutine must complete.
		time.Sleep(maxQueueDelay)
		synctest.Wait()
		if !done {
			t.Fatal("QueueEvent is still blocked after queue capacity was freed; expected it to unblock")
		}

		s.Close(context.Background())
	})

}

// Test_Sender_QueueEventUnblocksAfterCloseWhenFull verifies QueueEvent unblocks
// after Close is called while the queue is full.
func Test_Sender_QueueEventUnblocksAfterCloseWhenFull(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		defer func(previous int) {
			MaxQueuedEvents = previous
		}(MaxQueuedEvents)
		MaxQueuedEvents = 100

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, _ connectors.Events) error {
			return nil
		}
		s := New(app, nil)

		// Fill the queue up to its maximum capacity.
		for i := 0; i < MaxQueuedEvents; i++ {
			s.SendEvent(createTestEvent(s, i))
		}

		// Start a goroutine that attempts to enqueue one more event.
		// Since the queue is full, QueueEvent must block.
		var done bool
		go func() {
			s.SendEvent(createTestEvent(s, MaxQueuedEvents))
			done = true
		}()
		synctest.Wait()

		if done {
			t.Fatal("QueueEvent unexpectedly unblocked before Close")
		}

		s.Close(t.Context())
		synctest.Wait()
		if !done {
			t.Fatal("QueueEvent is still blocked after Close; expected it to unblock")
		}
	})

}

// Test_Sender_QueueEventUnblocksAfterDiscard verifies QueueEvent unblocks after
// a discard.
func Test_Sender_QueueEventUnblocksAfterDiscard(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		defer func(previous int) {
			MaxQueuedEvents = previous
		}(MaxQueuedEvents)
		MaxQueuedEvents = 100

		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			// Discard the first event.
			for range events.All() {
				events.Discard(errors.New("discard error"))
				break
			}
			// Pause to allow verification that an additional event is enqueued.
			time.Sleep(1)
			return nil
		}
		s := New(app, nil)

		// Fill the queue up to its maximum capacity.
		for i := 0; i < MaxQueuedEvents; i++ {
			s.SendEvent(createTestEvent(s, i))
		}

		// Start a goroutine that attempts to enqueue one more event.
		// Since the queue is full, QueueEvent must block.
		var done bool
		go func() {
			s.SendEvent(createTestEvent(s, MaxQueuedEvents))
			done = true
		}()
		synctest.Wait()

		// Let one queue cycle proceed so space becomes available.
		// QueueEvent should then unblock and the goroutine must complete.
		time.Sleep(maxQueueDelay)
		synctest.Wait()
		if !done {
			t.Fatal("QueueEvent is still blocked after queue capacity was freed; expected it to unblock")
		}

		s.Close(context.Background())
	})

}

// Test_Sender_OrderingRemoval verifies that orderings are removed after
// their last event is discarded or sent.
func Test_Sender_OrderingRemoval(t *testing.T) {
	t.Run("DiscardBeforeEnqueue", func(t *testing.T) {
		app := newTestApplication()
		s := New(app, nil)
		defer s.Close(t.Context())

		event := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user-1",
			"messageId":   "msg-1",
		}, nopAck{})

		s.DiscardEvent(event)

		s.mu.Lock()
		_, ok := s.orderings[orderingKey{group: "events", anonymousID: "user-1"}]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected ordering to be removed after discarding the only event before enqueue, got present")
		}
	})

	t.Run("SameUserDifferentGroups", func(t *testing.T) {
		app := newTestApplication()
		s := New(app, nil)
		defer s.Close(t.Context())

		newEvent := func(group, messageID string) *Event {
			return s.CreateEvent(testPipelineID, "Click", group, types.Type{}, map[string]any{
				"anonymousId": "user",
				"messageId":   messageID,
			}, nopAck{})
		}
		contacts := newEvent("contacts", "contacts-1")
		orders := newEvent("orders", "orders-1")
		contactsKey := orderingKey{group: "contacts", anonymousID: "user"}
		ordersKey := orderingKey{group: "orders", anonymousID: "user"}

		hasOrdering := func(key orderingKey) bool {
			s.mu.Lock()
			defer s.mu.Unlock()
			_, ok := s.orderings[key]
			return ok
		}
		if !hasOrdering(contactsKey) || !hasOrdering(ordersKey) {
			t.Fatal("expected both orderings to be present after creating their events")
		}

		s.DiscardEvent(contacts)
		if hasOrdering(contactsKey) {
			t.Fatal("expected contacts ordering to be removed after discarding its only event")
		}
		if !hasOrdering(ordersKey) {
			t.Fatal("expected orders ordering to remain until its event is processed")
		}

		s.DiscardEvent(orders)
		if hasOrdering(ordersKey) {
			t.Fatal("expected orders ordering to be removed after discarding its only event")
		}
	})

	t.Run("DiscardDuringIteration", func(t *testing.T) {
		done := make(chan struct{})
		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for event := range events.All() {
				events.Discard(errors.New("discard"))
				if event == nil {
					t.Fatal("unexpected nil event")
				}
				break
			}
			return nil
		}
		s := New(app, nil)
		s.setSentFunc(func(messageID string, err error) {
			close(done)
		})
		defer s.Close(t.Context())

		event := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user-2",
			"messageId":   "msg-2",
		}, nopAck{})
		s.SendEvent(event)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected discard to complete, got timeout")
		}

		s.mu.Lock()
		_, ok := s.orderings[orderingKey{group: "events", anonymousID: "user-2"}]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected ordering to be removed after discarding during iteration, got present")
		}
	})

	t.Run("SendSuccess", func(t *testing.T) {
		done := make(chan struct{})
		app := newTestApplication()
		app.SendEventsFunc = func(_ context.Context, events connectors.Events) error {
			for range events.All() {
				break
			}
			return nil
		}
		s := New(app, nil)
		s.setSentFunc(func(messageID string, err error) {
			close(done)
		})
		defer s.Close(t.Context())

		event := s.CreateEvent(testPipelineID, "Click", "events", types.Type{}, map[string]any{
			"anonymousId": "user-3",
			"messageId":   "msg-3",
		}, nopAck{})
		s.SendEvent(event)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected send to complete, got timeout")
		}

		s.mu.Lock()
		_, ok := s.orderings[orderingKey{group: "events", anonymousID: "user-3"}]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected ordering to be removed after sending the only event, got present")
		}
	})
}

// createTestEvent creates a minimal event for tests.
func createTestEvent(s *Sender, i int) *Event {
	return s.CreateEvent(testPipelineID, "page", "events", types.Type{}, map[string]any{
		"anonymousId": "user123",
		"messageId":   fmt.Sprintf("msg-%d", i),
	}, nopAck{})
}
