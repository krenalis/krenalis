//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package sender

import (
	"context"
	"encoding/binary"
	"fmt"
	"iter"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

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

func zeroWaitTime(string) (time.Duration, error) { return 0, nil }

func Test_CreateEvent_DeterministicID(t *testing.T) {
	s := New("test", zeroWaitTime, func(context.Context, meergo.Events) error { return nil }, func([]Ack, error) {})
	src1 := rand.NewPCG(1, ^uint64(1))
	src2 := rand.NewPCG(1, ^uint64(1))
	e1 := s.CreateEvent(0, "t", types.Type{}, events.Event{"anonymousId": "u"}, src1)
	e2 := s.CreateEvent(0, "t", types.Type{}, events.Event{"anonymousId": "u"}, src2)
	if e1.ID != e2.ID {
		t.Fatalf("expected deterministic IDs, got %q and %q", e1.ID, e2.ID)
	}
	if e1.sequence != 0 || e2.sequence != 1 {
		t.Fatalf("unexpected sequence numbers %d and %d", e1.sequence, e2.sequence)
	}
	if e1.user == nil || e2.user == nil {
		t.Fatal("user should not be nil")
	}
}

func Test_iterator_invalidUsage(t *testing.T) {
	s := New("test", zeroWaitTime, func(context.Context, meergo.Events) error { return nil }, func([]Ack, error) {})
	expectPanic := func(f func()) {
		defer func() {
			if r := recover(); r == nil {
				t.Fatalf("expected panic")
			}
		}()
		f()
	}

	t.Run("PostponeOutsideIteration", func(t *testing.T) {
		it := newIterator(s)
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PostponeFirstEvent", func(t *testing.T) {
		it := newIterator(s)
		it.iterating = true
		it.firstEvent = true
		expectPanic(func() { it.Postpone() })
	})

	t.Run("PeekAfterConsumed", func(t *testing.T) {
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.Peek() })
	})

	t.Run("AllAfterConsumed", func(t *testing.T) {
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.All() })
	})

	t.Run("FirstAfterConsumed", func(t *testing.T) {
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.First() })
	})

	t.Run("SameUserAfterConsumed", func(t *testing.T) {
		it := newIterator(s)
		it.consumed = true
		expectPanic(func() { it.SameUser() })
	})

	t.Run("FirstNoEvents", func(t *testing.T) {
		it := newIterator(s)
		expectPanic(func() { it.First() })
	})
}

func Test_Sender(t *testing.T) {

	tests := []struct {
		num     int    // number of records to process
		seed    uint64 // seed value to deterministically pseudo-randomize the test
		shuffle bool   // whether to shuffle the events
		users   int    // number of users, must be > 0
	}{
		{num: 0, seed: 0, users: 1},
		{num: 1, seed: 25, users: 1},
		{num: 1, seed: 92, users: 1},
		{num: 4, seed: 40, users: 1},
		{num: 4, seed: 40, shuffle: true, users: 1},
		{num: 1000 / 2, seed: 63, shuffle: false, users: 1000 / 13},
		{num: 1000 / 2, seed: 11, shuffle: true, users: 1000 / 18},
		{num: 1000 / 3, seed: 47, shuffle: false, users: 1000 / 10},
		{num: 1000 * 8, seed: 90, shuffle: true, users: 1000 / 9},
		{num: 1000 * 15, seed: 142, shuffle: false, users: 1000 / 3},
		{num: 1000 * 20, seed: 28, shuffle: true, users: 1000 / 5},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%d/%d/%d", test.num, test.seed, test.users), func(t *testing.T) {

			src := rand.NewPCG(test.seed, ^test.seed)
			rng := rand.New(src)

			app := newApp(t, test.seed)
			s := New("test", zeroWaitTime, app.SendEvents, app.ack)

			ctx := context.Background()

			// Generate random users.
			anonymousIds := make([]string, test.users)
			for i := 0; i < test.users; i++ {
				// Create a pseudo-random UUID v5.
				n := src.Uint64()
				data := make([]byte, 8)
				binary.BigEndian.PutUint64(data, n)
				anonymousIds[i] = uuid.NewSHA1(uuidDeterministicNS, data).String()
			}

			userByEvent := map[string]string{}
			eventsByUser := map[string][]string{}
			receivedAck := map[string]bool{}

			// Create the events.
			var evs []*Event
			for range test.num {
				anonymousId := anonymousIds[rng.IntN(test.users)]
				event := s.CreateEvent(1, "test", types.Type{}, events.Event{"anonymousId": anonymousId}, src)
				userByEvent[event.ID] = anonymousId
				if ids, ok := eventsByUser[anonymousId]; ok {
					eventsByUser[anonymousId] = append(ids, event.ID)
				} else {
					eventsByUser[anonymousId] = []string{event.ID}
				}
				if _, ok := receivedAck[event.ID]; ok {
					t.Fatal("CreateEvent has returned a duplicated ID")
				}
				receivedAck[event.ID] = false
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

			// Check that all events have been delivered and in the correct order.
			expectedByUser := map[string]int{}
			for i, id := range app.Delivered() {
				u, ok := userByEvent[id]
				if !ok {
					t.Fatalf("ack %d/%d: unexpected non-existent event %q", i+1, test.num, id)
				}
				ids := eventsByUser[u]
				expected := expectedByUser[u]
				expectedByUser[u]++
				if ids[expected] != id {
					t.Fatalf("ack %d/%d: expected event %q, got %q", i+1, test.num, ids[expected], id)
				}
			}
			for u, ids := range eventsByUser {
				expected := expectedByUser[u]
				if expected < len(ids) {
					t.Fatalf("ack: ID %q has not been received", ids[0])
				}
			}

			// Check that all acks have been received without errors.
			for i, ack := range app.Acks() {
				if ack.err != nil {
					t.Fatalf("ack %d/%d: expected no error, got %#v", i+1, test.num, ack.err)
				}
				for _, id := range ack.ids {
					if r, ok := receivedAck[id]; !ok {
						t.Fatalf("ack %d/%d: unexpected ID %q", i+1, test.num, id)
					} else if r {
						t.Fatalf("ack %d/%d: ID %q has already been received", i+1, test.num, id)
					}
					receivedAck[id] = true
				}
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

type ack struct {
	ids []string
	err error
}

type app struct {
	t    *testing.T
	seed uint64

	mu        sync.Mutex
	iteration uint64
	n         int      // protected by mu
	delivered []string // ids of the delivered events; protected by mu
	acks      []ack    // protected by mu
}

func newApp(t *testing.T, seed uint64) *app {
	a := app{
		t:    t,
		seed: seed,
		acks: []ack{},
	}
	return &a
}

func (app *app) Acks() []ack {
	app.mu.Lock()
	acks := slices.Clone(app.acks)
	app.mu.Unlock()
	return acks
}

func (app *app) Delivered() []string {
	app.mu.Lock()
	delivered := slices.Clone(app.delivered)
	app.mu.Unlock()
	return delivered
}

func (app *app) N() int {
	app.mu.Lock()
	n := app.n
	app.mu.Unlock()
	return n
}

func (app *app) SendEvents(ctx context.Context, events meergo.Events) error {

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
		app.mu.Lock()
		app.delivered = append(app.delivered, event.ID)
		app.mu.Unlock()
		time.Sleep(time.Duration(rng.Int()%10) * time.Nanosecond)
		return nil
	}

	var seq iter.Seq[*meergo.Event]
	if rng.Int()%3 == 0 {
		seq = events.SameUser()
	} else {
		seq = events.All()
	}

	var n int
	var delivered []string
	for event := range seq {
		app.validateEvent(event)
		if n%4 == 0 {
			if p, ok := events.Peek(); ok {
				app.validateEvent(p)
			}
		}
		if n > 0 && rng.Int()%3 == 0 {
			events.Postpone()
		} else {
			delivered = append(delivered, event.ID)
		}
		if n == rng.Int()/2 {
			break
		}
		n++
	}

	time.Sleep(time.Duration(rng.Int()%10) * time.Microsecond)

	app.mu.Lock()
	app.delivered = append(app.delivered, delivered...)
	app.mu.Unlock()

	return nil
}

func (app *app) ack(acks []Ack, err error) {
	app.t.Helper()
	if len(acks) == 0 {
		app.t.Fatalf("ack: expected at least one ack, got none")
	}
	ids := make([]string, len(acks))
	for i, ack := range acks {
		ids[i] = ack.Event
	}
	app.mu.Lock()
	app.acks = append(app.acks, ack{ids: ids, err: err})
	app.n += len(ids)
	app.mu.Unlock()
}

func (app *app) validateEvent(e *meergo.Event) {
	app.t.Helper()
	if e.ID == "" {
		app.t.Fatal("SendEvents: expected non-empty message ID, got empty")
	}
	if e.Type == "" {
		app.t.Fatal("SendEvents: expected type, got empty")
	}
	if e.Schema.Valid() {
		if e.Properties == nil {
			app.t.Fatal("SendEvents: expected non-nil properties with a valid schema, got nil")
		}
	} else {
		if e.Properties != nil {
			app.t.Fatal("SendEvents: expected nil properties with an invalid schema, got non-nil")
		}
	}
	if e.Raw == nil {
		app.t.Fatal("SendEvents: expected non-nil raw event, got nil")
	}
}
