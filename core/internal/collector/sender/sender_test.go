// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"iter"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
)

// uuidDeterministicNS defines the namespace used to generate deterministic UUIDv5 values.
var uuidDeterministicNS = uuid.MustParse("00000000-0000-0000-0000-000000000000")

// testApplication is a configurable Application implementation for tests.
// It defaults to no-op behavior when hooks are not provided.
type testApplication struct {
	IDValue        int
	ConnectorValue string
	WaitTimeFunc   func(string) (time.Duration, error)
	SendEventsFunc func(context.Context, connectors.Events) error
}

func newTestApplication() *testApplication {
	return &testApplication{
		IDValue:        1,
		ConnectorValue: "nop",
	}
}

func (a *testApplication) ID() int {
	if a.IDValue == 0 {
		return 1
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
func nopAck() {}

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

		event0 := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user",
				"messageId":   "msg-0",
			},
			Ack: nopAck,
		})
		event1 := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user",
				"messageId":   "msg-1",
			},
			Ack: nopAck,
		})

		s.DiscardEvent(event1)
		s.SendEvent(event0)
		time.Sleep(maxQueueDelay)

		s.Close(t.Context())
		if !consumed {
			t.Fatalf("event was not consumed")
		}

	})
}

// Test_Sender_SequenceOverflowRescale verifies that per-user ordering holds
// across sequence overflow.
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

		// Force the per-user sequence near overflow without creating a huge number
		// of events, while still asserting only on externally visible ordering.
		const userID = "user-overflow"
		s.mu.Lock()
		u := users.Get()
		u.anonymousID = userID
		u.queue.sequence.next = math.MaxInt - 1
		u.queue.sequence.expected = math.MaxInt - 1
		s.users[userID] = u
		s.mu.Unlock()

		makeEvent := func(messageID string) *Event {
			t.Helper()
			return s.CreateEvent(1, "Click", types.Type{}, streams.Event{
				Attributes: map[string]any{
					"anonymousId": userID,
					"messageId":   messageID,
				},
				Ack: nopAck,
			})
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

		event := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user",
				"messageId":   "msg-0",
			},
			Ack: nopAck,
		})
		s.SendEvent(event)

		time.Sleep(maxQueueDelay)
		time.Sleep(1) // TODO(marco): remove the following line. See issue https://github.com/meergo/meergo/issues/2122

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

		MaxQueuedEvents = 1
		defer func() {
			MaxQueuedEvents = 5_000
		}()

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
		for i := 0; i < total; i++ {
			s.SendEvent(createTestEvent(s, i))
		}

		time.Sleep(1) // TODO(marco): remove the following line. See issue https://github.com/meergo/meergo/issues/2122
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

		MaxQueuedEvents = 100
		defer func() {
			MaxQueuedEvents = 5_000
		}()

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

		MaxQueuedEvents = 100
		defer func() {
			MaxQueuedEvents = 5_000
		}()

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

		MaxQueuedEvents = 100
		defer func() {
			MaxQueuedEvents = 5_000
		}()

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

// Test_Sender_UserRemoval verifies that users are removed after their last
// event is discarded or sent.
func Test_Sender_UserRemoval(t *testing.T) {
	t.Run("DiscardBeforeEnqueue", func(t *testing.T) {
		app := newTestApplication()
		s := New(app, nil)
		defer s.Close(t.Context())

		event := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user-1",
				"messageId":   "msg-1",
			},
			Ack: nopAck,
		})

		s.DiscardEvent(event)

		s.mu.Lock()
		_, ok := s.users["user-1"]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected user to be removed after discarding the only event before enqueue, got present")
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

		event := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user-2",
				"messageId":   "msg-2",
			},
			Ack: nopAck,
		})
		s.SendEvent(event)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected discard to complete, got timeout")
		}

		s.mu.Lock()
		_, ok := s.users["user-2"]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected user to be removed after discarding during iteration, got present")
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

		event := s.CreateEvent(1, "Click", types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": "user-3",
				"messageId":   "msg-3",
			},
			Ack: nopAck,
		})
		s.SendEvent(event)

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("expected send to complete, got timeout")
		}

		s.mu.Lock()
		_, ok := s.users["user-3"]
		s.mu.Unlock()
		if ok {
			t.Fatal("expected user to be removed after sending the only event, got present")
		}
	})
}

func Test_Sender(t *testing.T) {

	tests := []struct {
		num         int     // number of events to process
		seed        uint64  // seed value to deterministically pseudo-randomize the test
		shuffle     bool    // whether to shuffle the events
		users       int     // number of users, must be > 0
		discardRate float64 // rate [0,1] at which events are discarded
	}{
		{num: 0, seed: 0, users: 1},
		{num: 0, seed: 0, users: 1, discardRate: 1},
		{num: 1, seed: 25, users: 1},
		{num: 1, seed: 92, users: 1, discardRate: 0.1},
		{num: 4, seed: 40, users: 1},
		{num: 4, seed: 40, shuffle: true, users: 1, discardRate: 0.1},
		{num: 4, seed: 40, shuffle: true, users: 1, discardRate: 1},
		{num: 1000 / 2, seed: 63, shuffle: false, users: 1000 / 13, discardRate: 0.008},
		{num: 1000 / 2, seed: 11, shuffle: true, users: 1000 / 18, discardRate: 0.12},
		{num: 1000 / 3, seed: 47, shuffle: false, users: 1000 / 10, discardRate: 0.075},
		{num: 1000 * 8, seed: 90, shuffle: true, users: 1000 / 9, discardRate: 0.187},
		{num: 1000 * 15, seed: 142, shuffle: false, users: 1000 / 3, discardRate: 0.09},
		{num: 1000 * 20, seed: 28, shuffle: true, users: 1000 / 5, discardRate: 0.045},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d/%d/%d", test.num, test.seed, test.users), func(t *testing.T) {

			src := rand.NewPCG(test.seed, ^test.seed)
			rng := rand.New(src)

			app := newApplication(t, test.seed)
			s := New(app, nil)
			s.setSentFunc(app.sent)

			ctx := context.Background()

			// Generate random users.
			anonymousIDs := make([]string, test.users)
			for i := 0; i < test.users; i++ {
				// Create a pseudo-random UUID v5.
				n := src.Uint64()
				data := make([]byte, 8)
				binary.BigEndian.PutUint64(data, n)
				anonymousIDs[i] = uuid.NewSHA1(uuidDeterministicNS, data).String()
			}

			userByEvent := map[string]string{}
			validEventsByUser := map[string][]string{}
			isValid := map[string]bool{}
			receivedAck := map[string]bool{}

			// Create the events.
			var evs []*Event
			for range test.num {
				// Choose an Anonymous ID deterministically.
				anonymousId := anonymousIDs[rng.IntN(test.users)]
				// Generate a deterministic UUIDv5 from src.
				n := src.Uint64()
				data := make([]byte, 8)
				binary.BigEndian.PutUint64(data, n)
				messageId := uuid.NewSHA1(uuidDeterministicNS, data).String()
				// Deterministically decide whether the event should be valid.
				valid := rng.Float64() >= test.discardRate
				typ := "Valid"
				if !valid {
					typ = "Invalid"
				}
				event := s.CreateEvent(1, typ, types.Type{}, streams.Event{
					Attributes: map[string]any{"anonymousId": anonymousId, "messageId": messageId},
					Ack:        nopAck,
				})
				userByEvent[messageId] = anonymousId
				if valid {
					if ids, ok := validEventsByUser[anonymousId]; ok {
						validEventsByUser[anonymousId] = append(ids, messageId)
					} else {
						validEventsByUser[anonymousId] = []string{messageId}
					}
				}
				if _, ok := receivedAck[messageId]; ok {
					t.Fatal("CreateEvent has returned a duplicated ID")
				}
				isValid[messageId] = valid
				receivedAck[messageId] = false
				evs = append(evs, event)
			}

			// Shuffle the events.
			if test.shuffle {
				rng.Shuffle(len(evs), func(i, j int) {
					evs[i], evs[j] = evs[j], evs[i]
				})
			}

			// Queue the events.
			for _, event := range evs {
				s.SendEvent(event)
			}

			for n := 0; n != test.num; {
				time.Sleep(1 * time.Second)
				n = app.N()
				trace("acks: %d\n", n)
			}

			// Close the sender.
			s.Close(context.Background())

			// Check that all valid events have been consumed and in the correct order.
			expectedByUser := map[string]int{}
			for i, id := range app.Consumed() {
				u, ok := userByEvent[id]
				if !ok {
					t.Fatalf("ack %d/%d: unexpected non-existent event %q", i+1, test.num, id)
				}
				ids := validEventsByUser[u]
				expected := expectedByUser[u]
				expectedByUser[u]++
				if expected >= len(ids) {
					t.Fatalf("ack %d/%d: unexpected consumed event %q", i+1, test.num, id)
				}
				if ids[expected] != id {
					t.Fatalf("ack %d/%d: expected consumed event %q, got %q", i+1, test.num, ids[expected], id)
				}
			}
			for u, ids := range validEventsByUser {
				expected := expectedByUser[u]
				if expected < len(ids) {
					t.Fatalf("ack: ID %q has not been received", ids[0])
				}
			}

			// Check that all sends were completed.
			for i, send := range app.Sends() {
				id := send.messageID
				if r, ok := receivedAck[id]; !ok {
					t.Fatalf("ack %d/%d: unexpected ID %q", i+1, test.num, id)
				} else if r {
					t.Fatalf("ack %d/%d: ID %q has already been received", i+1, test.num, id)
				}
				if send.err == nil {
					if !isValid[id] {
						t.Fatalf("ack %d/%d: expected error for ID %q, got none", i+1, test.num, id)
					}
				} else {
					if isValid[id] {
						t.Fatalf("ack %d/%d: expected no error for ID %q, got an error", i+1, test.num, id)
					}
				}
				receivedAck[id] = true
			}
			for id, r := range receivedAck {
				if !r {
					t.Fatalf("ack: ID %q has not been received", id)
				}
			}

			s.Close(ctx)

		})
	}

}

// createTestEvent creates a minimal event for tests.
func createTestEvent(s *Sender, i int) *Event {
	return s.CreateEvent(1, "page", types.Type{}, streams.Event{
		Attributes: map[string]any{
			"anonymousId": "user123",
			"messageId":   fmt.Sprintf("msg-%d", i),
		},
		Ack: nopAck,
	})
}

type send struct {
	messageID string
	err       error
}

type application struct {
	t    *testing.T
	seed uint64

	mu        sync.Mutex
	iteration uint64
	n         int      // protected by mu
	consumed  []string // ids of the consumed events; protected by mu
	sends     []send   // protected by mu
}

func newApplication(t *testing.T, seed uint64) *application {
	app := application{
		t:     t,
		seed:  seed,
		sends: []send{},
	}
	return &app
}

func (app *application) Sends() []send {
	app.mu.Lock()
	sends := slices.Clone(app.sends)
	app.mu.Unlock()
	return sends
}

func (app *application) ID() int {
	return 1
}

func (app *application) Connector() string {
	return "test"
}

func (app *application) Consumed() []string {
	app.mu.Lock()
	consumed := slices.Clone(app.consumed)
	app.mu.Unlock()
	return consumed
}

func (app *application) N() int {
	app.mu.Lock()
	n := app.n
	app.mu.Unlock()
	return n
}

func (app *application) SendEvents(ctx context.Context, events connectors.Events) error {

	// Get the current iteration number.
	var iteration uint64
	app.mu.Lock()
	iteration = app.iteration
	app.iteration++
	app.mu.Unlock()

	if app.iteration == math.MaxUint64 {
		panic("iteration is out of range")
	}

	seed := app.seed + iteration
	src := rand.NewPCG(seed, ^seed)
	rng := rand.New(src)

	// Test Peek.
	if rng.Int()%8 == 0 {
		event, _ := events.Peek()
		app.validateEvent(event)
		if rng.Int()%4 == 0 {
			event, ok := events.Peek()
			if !ok {
				return nil
			}
			app.validateEvent(event)
		}
	}

	// Test First.
	if rng.Int()%5 == 0 {
		event := events.First()
		app.validateEvent(event)
		if event.Type.ID == "Valid" {
			app.mu.Lock()
			app.consumed = append(app.consumed, event.Received.MessageID())
			app.mu.Unlock()
			time.Sleep(time.Duration(rng.Int()%10) * time.Nanosecond)
			return nil
		}
		return errors.New("event is not valid")
	}

	var seq iter.Seq[*connectors.Event]
	if rng.Int()%3 == 0 {
		seq = events.SameUser()
	} else {
		seq = events.All()
	}

	var n int
	var consumed []string
	for event := range seq {
		app.validateEvent(event)
		if n%4 == 0 {
			if p, ok := events.Peek(); ok {
				app.validateEvent(p)
			}
		}
		if n > 0 && rng.Int()%3 == 0 {
			events.Postpone()
		} else if event.Type.ID == "Invalid" {
			events.Discard(errors.New("event is invalid"))
		} else {
			consumed = append(consumed, event.Received.MessageID())
		}
		if n == rng.Int()/2 {
			break
		}
		n++
	}

	if len(consumed) == 0 {
		return nil
	}

	app.mu.Lock()
	app.consumed = append(app.consumed, consumed...)
	app.mu.Unlock()
	time.Sleep(time.Duration(rng.Int()%10) * time.Microsecond)

	return nil
}

func (app *application) sent(messageID string, err error) {
	app.t.Helper()
	if messageID == "" {
		app.t.Fatalf("sent: message ID is empty")
	}
	app.mu.Lock()
	app.sends = append(app.sends, send{messageID: messageID, err: err})
	app.n += 1
	app.mu.Unlock()
}

func (app *application) validateEvent(e *connectors.Event) {
	app.t.Helper()
	if e.Received.MessageID() == "" {
		app.t.Fatal("SendEvents: expected non-empty message ID, got empty")
	}
	if e.Type.ID != "Valid" && e.Type.ID != "Invalid" {
		app.t.Fatalf(`SendEvents: expected type "Valid" or "Invalid", got %q`, e.Type)
	}
	if e.Type.Schema.Valid() {
		if e.Type.Values == nil {
			app.t.Fatal("SendEvents: expected non-nil values with a valid schema, got nil")
		}
	} else {
		if e.Type.Values != nil {
			app.t.Fatal("SendEvents: expected nil values with an invalid schema, got non-nil")
		}
	}
	if e.Received == nil {
		app.t.Fatal("SendEvents: expected non-nil received event, got nil")
	}
}

func (app *application) WaitTime(string) (time.Duration, error) {
	return 0, nil
}
