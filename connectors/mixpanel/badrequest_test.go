// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mixpanel

import (
	"context"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/types"
)

// TestBadRequest verifies that the Mixpanel connector correctly deserializes
// the response body - which contains error details - when the Mixpanel server
// replies with an HTTP 400 Bad Request.
func TestBadRequest(t *testing.T) {

	mixpanel := newMixpanelForTests(t)

	ctx := context.WithValue(context.Background(), sendBadRequestContextKey, true)

	now := time.Now().UTC()

	event := &connectors.Event{
		Received: testconnector.ReceivedEvent(map[string]any{
			"anonymousId":  "17fba6ee-8673-4ebc-afd6-69e62124e017",
			"connectionId": 1323607634,
			"context": map[string]any{
				"browser": map[string]any{
					"name":    "Other",
					"other":   "Unknown",
					"version": "0.0",
				},
				"ip": "127.0.0.1",
				"os": map[string]any{
					"name":    "Other",
					"version": "0.0",
				},
				"userAgent": "python-requests/2.32.4",
			},
			"messageId":         "1427b912-438f-46a8-ae7f-b276ee5345ee",
			"originalTimestamp": now,
			"previousId":        "IAJVLPBEZJ",
			"receivedAt":        now,
			"sentAt":            now.Add(-10 * time.Millisecond),
			"timestamp":         now,
			"type":              "alias",
			"userId":            nil,
		}),
		Type: connectors.EventTypeInfo{
			ID: "track",
			Schema: types.Object([]types.Property{
				{Name: "event", Prefilled: "event", Type: types.String().WithMaxLength(255), CreateRequired: true, Description: "Event Name"},
				{Name: "properties", Type: types.Map(types.JSON()), CreateRequired: true, Description: "Your Properties"},
			}),
			Values: map[string]any{
				"event": "Test Event",
				"properties": map[string]any{
					"X": 42,
				},
			},
		},
	}

	// Create an iterator over the test events.
	iter := testconnector.NewEventsIterator([]*connectors.Event{event, event})

	// Actually sends the events.
	t.Log("calling SendEvents")
	err := mixpanel.SendEvents(ctx, iter)
	if err == nil {
		t.Fatal("test should fail, but it returned no errors")
	}
	gotKrenalisError, ok := err.(connectors.EventsError)
	if !ok {
		t.Fatalf("expected a connectors.EventsError error, got %T instead", err)
	}
	if len(gotKrenalisError) != 2 {
		t.Fatalf("expected a connectors.EventsError with 2 event(s) inside, have %d instead", len(gotKrenalisError))
	}
	gotErr1 := gotKrenalisError[0].Error()
	gotErr2 := gotKrenalisError[1].Error()
	if gotErr1 != gotErr2 {
		t.Fatal("the two errors should be the same")
	}
	const expectedErr = `properties.time: 'properties.time' is invalid: must not be missing`
	if gotErr1 != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, gotErr1)
	}
	t.Log("the connector has returned the correct error")
}
