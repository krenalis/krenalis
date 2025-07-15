//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mixpanel

import (
	"context"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/testutils"
	"github.com/meergo/meergo/types"
)

// TestBadRequest verifies that the Mixpanel connector correctly deserializes
// the response body - which contains error details - when the Mixpanel server
// replies with an HTTP 400 Bad Request.
func TestBadRequest(t *testing.T) {

	mixpanel := initMixpanelForTests(t)

	ctx := context.WithValue(context.Background(), connectorTestString("sendBadRequest"), true)

	now := time.Now().UTC()

	event := &meergo.Event{
		ID:   "7ba8676a-3182-4d76-bf6e-21483fc63893",
		Type: "track",
		Schema: types.Object([]types.Property{
			{Name: "event", Placeholder: "event", Type: types.Text().WithCharLen(255), CreateRequired: true, Description: "Event Name"},
			{Name: "properties", Type: types.Map(types.JSON()), CreateRequired: true, Description: "Your Properties"},
		}),
		Properties: map[string]any{
			"event": "Test Event",
			"properties": map[string]any{
				"X": 42,
			},
		},
		Received: events.ReceivedEvent(map[string]any{
			"anonymousId": "17fba6ee-8673-4ebc-afd6-69e62124e017",
			"connection":  1323607634,
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
			"id":                "03a7b3f6-7e5a-5933-96d8-81fcc9fdf696",
			"messageId":         "1427b912-438f-46a8-ae7f-b276ee5345ee",
			"originalTimestamp": now,
			"previousId":        "IAJVLPBEZJ",
			"receivedAt":        now,
			"sentAt":            now.Add(-10 * time.Millisecond),
			"timestamp":         now,
			"type":              "alias",
			"userId":            nil,
		}),
	}

	// Create an iterator over the test events.
	events := testutils.NewEventsIterator([]*meergo.Event{event, event})

	// Actually sends the events.
	t.Log("calling SendEvents")
	err := mixpanel.SendEvents(ctx, events)
	if err == nil {
		t.Fatal("test should fail, but it returned no errors")
	}
	gotMeergoError, ok := err.(meergo.EventsError)
	if !ok {
		t.Fatalf("expected a meergo.EventsError error, got %T instead", err)
	}
	if len(gotMeergoError) != 2 {
		t.Fatalf("expected a meergo.EventsError with 2 event(s) inside, have %d instead", len(gotMeergoError))
	}
	gotErr1 := gotMeergoError[0].Error()
	gotErr2 := gotMeergoError[1].Error()
	if gotErr1 != gotErr2 {
		t.Fatal("the two errors should be the same")
	}
	const expectedErr = `sending event "properties.time": 'properties.time' is invalid: must not be missing`
	if gotErr1 != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, gotErr1)
	}
	t.Log("the connector has returned the correct error")
}
