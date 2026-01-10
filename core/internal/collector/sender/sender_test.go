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
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
)

// nopApplication is a no-op application that returns zero wait time and skips
// sending events.
type nopApplication struct{}

func (nopApplication) ID() int { return 1 }

func (nopApplication) Connector() string { return "nop" }

func (nopApplication) WaitTime(string) (time.Duration, error) {
	return 0, nil
}

func (nopApplication) SendEvents(context.Context, connectors.Events) error {
	return nil
}

func Test_newStoppedTimer(t *testing.T) {
	tm := newStoppedTimer()
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
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PostponeFirstEvent", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.firstEvent = true
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PostponeDiscardedEvent", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.discarded = true
		expectPanic(func() { it.Postpone() })
	})

	t.Run("DiscardDiscardedEvent", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.discarded = true
		expectPanic(func() { it.Discard(errors.New("event is invalid")) })
	})

	t.Run("DiscardPostponedEvent", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.iterating = true
		it.postponed = true
		expectPanic(func() { it.Discard(errors.New("event is invalid")) })
	})

	t.Run("PeekAfterConsumed", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.Peek() })
	})

	t.Run("AllAfterConsumed", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.All() })
	})

	t.Run("FirstAfterConsumed", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.First() })
	})

	t.Run("SameUserAfterConsumed", func(t *testing.T) {
		s := New(nopApplication{}, nil)
		defer s.Close(t.Context())
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.SameUser() })
	})

}

// nopAck is a no-op streams.Ack implementation.
func nopAck() {}

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
				s.QueueEvent(event)
			}

			for n := 0; n != test.num; {
				time.Sleep(1 * time.Second)
				n = app.N()
				trace("acks: %d\n", n)
			}

			// Close the sender.
			err := s.Close(context.Background())
			if err != nil {
				t.Fatalf("cannot close the sender: %s", err)
			}

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

			if err := s.Close(ctx); err != nil {
				t.Fatalf("Close: expected no error, got error %q", err)
			}

		})
	}

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
