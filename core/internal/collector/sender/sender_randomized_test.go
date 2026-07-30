// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package sender

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math/rand/v2"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/tools/types"
)

var (
	errSenderRandomInvalidEvent   = errors.New("event is invalid")
	errSenderRandomMalformedEvent = errors.New("event returned by sender is malformed")
)

type senderRandomTest struct {
	events      int
	seed        uint64
	shuffle     bool
	users       int
	invalidRate float64
}

// Test_Sender_Randomized tests the sender using seeded pseudo-random workloads,
// including out-of-order event submission, multiple users, invalid events, and
// all methods provided by connectors.Events.
//
// It verifies that each event is finalized exactly once and that all valid
// events are consumed exactly once, in creation order for each user. Each
// scenario runs inside a synctest bubble, using virtual time and explicit
// completion synchronization instead of real-time polling.
func Test_Sender_Randomized(t *testing.T) {
	tests := []senderRandomTest{
		{events: 0, seed: 0, users: 1},
		{events: 1, seed: 25, users: 1},
		{events: 1, seed: 92, users: 1, invalidRate: 0.1},
		{events: 4, seed: 40, users: 1},
		{events: 4, seed: 40, shuffle: true, users: 1, invalidRate: 0.1},
		{events: 4, seed: 40, shuffle: true, users: 1, invalidRate: 1},
		{events: 500, seed: 63, users: 76, invalidRate: 0.008},
		{events: 500, seed: 11, shuffle: true, users: 55, invalidRate: 0.12},
		{events: 333, seed: 47, users: 100, invalidRate: 0.075},
		{events: 8_000, seed: 90, shuffle: true, users: 111, invalidRate: 0.187},
		{events: 15_000, seed: 142, users: 333, invalidRate: 0.09},
		{events: 20_000, seed: 28, shuffle: true, users: 200, invalidRate: 0.045},
	}

	var coverage senderRandomCoverage
	for _, test := range tests {
		name := fmt.Sprintf("events=%d,seed=%d,users=%d,shuffle=%t,invalid=%g",
			test.events, test.seed, test.users, test.shuffle, test.invalidRate)
		t.Run(name, func(t *testing.T) {
			var testCoverage senderRandomCoverage
			synctest.Test(t, func(t *testing.T) {
				testCoverage = testSenderRandomScenario(t, test)
			})
			coverage.add(testCoverage)
		})
	}
	coverage.assert(t)
}

func testSenderRandomScenario(t *testing.T, test senderRandomTest) senderRandomCoverage {
	rng := rand.New(rand.NewPCG(test.seed, ^test.seed))

	app := newSenderRandomApplication(test.seed, test.events)
	s := New(app, nil)
	t.Cleanup(func() {
		s.Close(t.Context())
	})
	s.setSentFunc(app.sent)

	// Generate deterministic users.
	anonymousIDs := make([]string, test.users)
	for i := range anonymousIDs {
		anonymousIDs[i] = fmt.Sprintf("user-%d", i)
	}

	expectedEvents := make(map[string]senderRandomExpectedEvent, test.events)
	validEventsByUser := make(map[string][]string)
	events := make([]*Event, 0, test.events)

	for i := range test.events {
		anonymousID := anonymousIDs[rng.IntN(test.users)]
		messageID := fmt.Sprintf("message-%d", i)
		valid := rng.Float64() >= test.invalidRate
		typ := "Valid"
		if !valid {
			typ = "Invalid"
		}
		event := s.CreateEvent(testPipelineID, typ, types.Type{}, streams.Event{
			Attributes: map[string]any{
				"anonymousId": anonymousID,
				"messageId":   messageID,
			},
			Ack: nopAck,
		})
		expectedEvents[messageID] = senderRandomExpectedEvent{
			anonymousID: anonymousID,
			valid:       valid,
		}
		if valid {
			validEventsByUser[anonymousID] = append(validEventsByUser[anonymousID], messageID)
		}
		events = append(events, event)
	}

	if test.shuffle {
		rng.Shuffle(len(events), func(i, j int) {
			events[i], events[j] = events[j], events[i]
		})
	}
	for _, event := range events {
		s.SendEvent(event)
	}

	<-app.done
	s.Close(t.Context())

	snapshot := app.snapshot()
	assertSenderRandomScenario(t, test, snapshot, expectedEvents, validEventsByUser)
	return snapshot.coverage
}

type senderRandomExpectedEvent struct {
	anonymousID string
	valid       bool
}

func assertSenderRandomScenario(
	t *testing.T,
	test senderRandomTest,
	snapshot senderRandomSnapshot,
	expectedEvents map[string]senderRandomExpectedEvent,
	validEventsByUser map[string][]string,
) {
	t.Helper()

	for _, failure := range snapshot.failures {
		t.Errorf("application: %s", failure)
	}
	if len(snapshot.results) != test.events {
		t.Errorf("finalized %d events, want %d", len(snapshot.results), test.events)
	}

	// Events may be interleaved across users, but each user's valid events must
	// be consumed exactly once and in creation order.
	consumedByUser := make(map[string]int)
	for i, messageID := range snapshot.consumed {
		expectedEvent, exists := expectedEvents[messageID]
		if !exists {
			t.Errorf("consumed event %d: unexpected message ID %q", i, messageID)
			continue
		}
		anonymousID := expectedEvent.anonymousID
		expected := consumedByUser[anonymousID]
		ids := validEventsByUser[anonymousID]
		if expected >= len(ids) {
			t.Errorf("consumed event %d: unexpected message ID %q for user %q", i, messageID, anonymousID)
			continue
		}
		if ids[expected] != messageID {
			t.Errorf("consumed event %d for user %q: got %q, want %q",
				i, anonymousID, messageID, ids[expected])
		}
		consumedByUser[anonymousID]++
	}
	for anonymousID, ids := range validEventsByUser {
		if got := consumedByUser[anonymousID]; got != len(ids) {
			t.Errorf("user %q consumed %d valid events, want %d", anonymousID, got, len(ids))
		}
	}

	// The sent callback is unordered, so treat its results as a set and verify
	// finalization, uniqueness, and error classification independently of order.
	finalized := make(map[string]bool, len(expectedEvents))
	for i, result := range snapshot.results {
		expectedEvent, exists := expectedEvents[result.messageID]
		if !exists {
			t.Errorf("finalization %d: unexpected message ID %q", i, result.messageID)
			continue
		}
		if finalized[result.messageID] {
			t.Errorf("finalization %d: message ID %q was finalized more than once", i, result.messageID)
			continue
		}
		if expectedEvent.valid && result.err != nil {
			t.Errorf("finalization %d: valid event %q failed: %v", i, result.messageID, result.err)
		}
		if !expectedEvent.valid && result.err == nil {
			t.Errorf("finalization %d: invalid event %q succeeded", i, result.messageID)
		}
		finalized[result.messageID] = true
	}
	for messageID := range expectedEvents {
		if !finalized[messageID] {
			t.Errorf("event %q was not finalized", messageID)
		}
	}
}

type senderRandomResult struct {
	messageID string
	err       error
}

type senderRandomCoverage struct {
	peek       int
	first      int
	all        int
	sameUser   int
	postpone   int
	discard    int
	breakEarly int
}

func (c *senderRandomCoverage) add(other senderRandomCoverage) {
	c.peek += other.peek
	c.first += other.first
	c.all += other.all
	c.sameUser += other.sameUser
	c.postpone += other.postpone
	c.discard += other.discard
	c.breakEarly += other.breakEarly
}

func (c senderRandomCoverage) assert(t *testing.T) {
	t.Helper()
	operations := []struct {
		name  string
		count int
	}{
		{name: "Peek", count: c.peek},
		{name: "First", count: c.first},
		{name: "All", count: c.all},
		{name: "SameUser", count: c.sameUser},
		{name: "Postpone", count: c.postpone},
		{name: "Discard", count: c.discard},
		{name: "early break", count: c.breakEarly},
	}
	for _, operation := range operations {
		if operation.count == 0 {
			t.Errorf("random scenarios did not exercise %s", operation.name)
		}
	}
}

type senderRandomSnapshot struct {
	consumed []string
	results  []senderRandomResult
	failures []string
	coverage senderRandomCoverage
}

type senderRandomApplication struct {
	seed     uint64
	expected int
	done     chan struct{}

	mu        sync.Mutex
	iteration uint64
	finalized int
	consumed  []string
	results   []senderRandomResult
	failures  []string
	coverage  senderRandomCoverage
}

func newSenderRandomApplication(seed uint64, expected int) *senderRandomApplication {
	app := &senderRandomApplication{
		seed:     seed,
		expected: expected,
		done:     make(chan struct{}),
	}
	if expected == 0 {
		close(app.done)
	}
	return app
}

func (app *senderRandomApplication) ID() string {
	return testConnectionID
}

func (app *senderRandomApplication) Connector() string {
	return "test"
}

func (app *senderRandomApplication) SendEvents(_ context.Context, events connectors.Events) error {
	app.mu.Lock()
	iteration := app.iteration
	app.iteration++
	app.mu.Unlock()

	seed := app.seed + iteration
	rng := rand.New(rand.NewPCG(seed, ^seed))
	var coverage senderRandomCoverage
	defer func() {
		app.mu.Lock()
		app.coverage.add(coverage)
		app.mu.Unlock()
	}()

	if rng.IntN(8) == 0 {
		coverage.peek++
		event, ok := events.Peek()
		if !ok {
			app.recordFailure("Peek returned no event at the start of an iteration")
			coverage.first++
			return app.sendFirst(events, rng)
		}
		app.validateEvent(event)
		if rng.IntN(4) == 0 {
			coverage.peek++
			if event, ok := events.Peek(); ok {
				app.validateEvent(event)
			} else {
				app.recordFailure("a repeated Peek returned no event")
			}
		}
	}

	if rng.IntN(5) == 0 {
		coverage.first++
		return app.sendFirst(events, rng)
	}

	var seq iter.Seq[*connectors.Event]
	if rng.IntN(3) == 0 {
		coverage.sameUser++
		seq = events.SameUser()
	} else {
		coverage.all++
		seq = events.All()
	}

	var consumed []string
	n := 0
	for event := range seq {
		valid := app.validateEvent(event)
		if n%4 == 0 {
			coverage.peek++
			if event, ok := events.Peek(); ok {
				app.validateEvent(event)
			}
		}
		if n > 0 && rng.IntN(3) == 0 {
			coverage.postpone++
			events.Postpone()
		} else if !valid {
			coverage.discard++
			events.Discard(errSenderRandomMalformedEvent)
		} else if event.Type.ID == "Invalid" {
			coverage.discard++
			events.Discard(errSenderRandomInvalidEvent)
		} else {
			consumed = append(consumed, event.Received.MessageID())
		}
		if rng.IntN(8) == 0 {
			coverage.breakEarly++
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
	time.Sleep(time.Duration(1+rng.IntN(10)) * time.Microsecond)
	return nil
}

func (app *senderRandomApplication) sendFirst(events connectors.Events, rng *rand.Rand) error {
	event := events.First()
	if !app.validateEvent(event) {
		return errSenderRandomMalformedEvent
	}
	if event.Type.ID == "Invalid" {
		return errSenderRandomInvalidEvent
	}
	app.mu.Lock()
	app.consumed = append(app.consumed, event.Received.MessageID())
	app.mu.Unlock()
	time.Sleep(time.Duration(1+rng.IntN(10)) * time.Nanosecond)
	return nil
}

func (app *senderRandomApplication) validateEvent(event *connectors.Event) bool {
	if event == nil {
		app.recordFailure("sender returned a nil event")
		return false
	}
	if event.Received == nil {
		app.recordFailure("sender returned an event with nil Received")
		return false
	}
	valid := true
	if event.Received.MessageID() == "" {
		app.recordFailure("sender returned an event with an empty message ID")
		valid = false
	}
	if event.Type.ID != "Valid" && event.Type.ID != "Invalid" {
		app.recordFailure("sender returned unexpected event type %q", event.Type.ID)
		valid = false
	}
	if event.Type.Schema.Valid() && event.Type.Values == nil {
		app.recordFailure("sender returned nil values with a valid schema")
		valid = false
	}
	if !event.Type.Schema.Valid() && event.Type.Values != nil {
		app.recordFailure("sender returned non-nil values with an invalid schema")
		valid = false
	}
	return valid
}

func (app *senderRandomApplication) recordFailure(format string, args ...any) {
	app.mu.Lock()
	app.failures = append(app.failures, fmt.Sprintf(format, args...))
	app.mu.Unlock()
}

func (app *senderRandomApplication) sent(messageID string, err error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if messageID == "" {
		app.failures = append(app.failures, "sent callback received an empty message ID")
	}
	app.results = append(app.results, senderRandomResult{messageID: messageID, err: err})
	app.finalized++
	if app.finalized == app.expected {
		close(app.done)
	} else if app.finalized > app.expected {
		app.failures = append(app.failures,
			fmt.Sprintf("sent callback finalized %d events, want %d", app.finalized, app.expected))
	}
}

func (app *senderRandomApplication) snapshot() senderRandomSnapshot {
	app.mu.Lock()
	defer app.mu.Unlock()
	return senderRandomSnapshot{
		consumed: slices.Clone(app.consumed),
		results:  slices.Clone(app.results),
		failures: slices.Clone(app.failures),
		coverage: app.coverage,
	}
}

func (app *senderRandomApplication) WaitTime(string) (time.Duration, error) {
	return 0, nil
}
