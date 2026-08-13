// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package testconnector

import (
	"slices"
	"testing"

	"github.com/krenalis/krenalis/connectors"
)

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
