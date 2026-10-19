// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package testconnector

import (
	"iter"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/connectors"
)

// TestEventsIteratorPeek verifies Peek's behavior before, during, and after
// iterating over events.
func TestEventsIteratorPeek(t *testing.T) {

	newEvent := func(anonymousID, messageID string) *connectors.Event {
		return &connectors.Event{Received: ReceivedEvent(map[string]any{
			"anonymousId": anonymousID,
			"messageId":   messageID,
		})}
	}

	events := []*connectors.Event{
		newEvent("anonymous-a", "a-1"),
		newEvent("anonymous-b", "b-1"),
		newEvent("anonymous-a", "a-2"),
	}
	tests := []struct {
		name string
		seq  func(connectors.Events) iter.Seq[*connectors.Event]
		want []string
	}{
		{name: "All", seq: connectors.Events.All, want: []string{"a-1", "b-1", "a-2"}},
		{name: "SameUser", seq: connectors.Events.SameUser, want: []string{"a-1", "a-2"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			it := NewEventsIterator(events)
			first, ok := it.Peek()
			if !ok {
				t.Fatal("Peek before iteration: expected an event")
			}
			if got := first.Received.MessageID(); got != test.want[0] {
				t.Fatalf("Peek before iteration: expected message ID %q, got %q", test.want[0], got)
			}
			if got, ok := it.Peek(); !ok || got != first {
				t.Fatalf("repeated Peek before iteration: expected event %p and true, got %p and %t", first, got, ok)
			}

			yielded := 0
			for event := range test.seq(it) {
				if yielded >= len(test.want) {
					t.Fatalf("expected %d events, got more", len(test.want))
				}
				if yielded == 0 && event != first {
					t.Fatalf("first event: expected Peek event %p, got %p", first, event)
				}
				if got := event.Received.MessageID(); got != test.want[yielded] {
					t.Fatalf("event %d: expected message ID %q, got %q", yielded, test.want[yielded], got)
				}
				if yielded+1 < len(test.want) {
					got, ok := it.Peek()
					if !ok {
						t.Fatalf("Peek during iteration: expected message ID %q", test.want[yielded+1])
					}
					if messageID := got.Received.MessageID(); messageID != test.want[yielded+1] {
						t.Fatalf("Peek during iteration: expected message ID %q, got %q", test.want[yielded+1], messageID)
					}
				} else if got, ok := it.Peek(); ok || got != nil {
					t.Fatalf("Peek at the end of the iteration: expected nil and false, got %p and %t", got, ok)
				}
				yielded++
			}
			if yielded != len(test.want) {
				t.Fatalf("expected %d events, got %d", len(test.want), yielded)
			}

			func() {
				defer func() {
					r := recover()
					if r == nil {
						t.Fatal("Peek after iteration: expected panic")
					}
					msg, ok := r.(string)
					if !ok || !strings.Contains(msg, "Events.Peek outside of an iteration") {
						t.Fatalf("Peek after iteration: unexpected panic %v", r)
					}
				}()
				it.Peek()
			}()

		})
	}

}

// TestEventsIteratorSameUserUsesAnonymousID verifies that SameUser selects
// events with the same anonymous ID as the first event, rather than by user ID.
func TestEventsIteratorSameUserUsesAnonymousID(t *testing.T) {
	newEvent := func(anonymousID, messageID string) *connectors.Event {
		return &connectors.Event{Received: ReceivedEvent(map[string]any{
			"anonymousId": anonymousID,
			"userId":      "identified-user",
			"messageId":   messageID,
		})}
	}

	events := []*connectors.Event{
		newEvent("anonymous-a", "a-1"),
		newEvent("anonymous-b", "b-1"),
		newEvent("anonymous-a", "a-2"),
	}

	var got []string
	for event := range NewEventsIterator(events).SameUser() {
		got = append(got, event.Received.MessageID())
	}
	want := []string{"a-1", "a-2"}
	if !slices.Equal(got, want) {
		t.Fatalf("SameUser returned message IDs %v, want %v", got, want)
	}
}
