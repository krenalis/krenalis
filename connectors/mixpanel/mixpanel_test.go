//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package mixpanel

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/testutils"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

func TestSendEvents(t *testing.T) {

	mixpanel := initMixpanelForTests(t)

	ctx := context.Background()

	messageID := uuid.NewString()
	anonymousID := uuid.NewString()

	now := time.Now().UTC()

	tests := []struct {
		events              []*meergo.Event
		expectedRequestBody map[string]any
	}{
		{
			events: []*meergo.Event{
				newEventForTest("track", anonymousID, messageID, now),
			},
			expectedRequestBody: map[string]any{
				"event": "Test Event",
				"properties": map[string]any{
					"$browser":    "Other",
					"$device_id":  anonymousID,
					"$insert_id":  messageID,
					"$os":         "Other",
					"X":           json.Number("42"),
					"distinct_id": anonymousID,
					"ip":          "127.0.0.1",
					"time":        json.Number(strconv.FormatInt(now.UnixMilli(), 10)),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			// Through the context, inform the 'SendEvents' method that the HTTP
			// request must also be copied to the memory location specified via
			// 'req', so its content can be verified in tests.
			var req http.Request
			ctx = context.WithValue(ctx, connectorTestString("storeSentHTTPRequest"), &req)

			// Create an iterator over the test events.
			events := testutils.NewEventsIterator(test.events)

			// Actually sends the events.
			t.Log("calling SendEvents")
			err := mixpanel.SendEvents(ctx, events)
			if err != nil {
				t.Fatal(err)
			}
			defer req.Body.Close()
			t.Log("SendEvent returned no errors")

			// Check that the HTTP request was actually set by the test.
			if req.Method == "" {
				t.Fatal("the 'SendEvents' function did not properly stored the HTTP request")
			}

			// Decode the request body and ensure it matches the expected result.
			body := req.Body
			if contentEncoding == meergo.Gzip {
				body, err = gzip.NewReader(body)
				if err != nil {
					t.Fatal(err)
				}
			}
			var jsonRequest map[string]any
			dec := json.NewDecoder(body)
			dec.UseNumber()
			err = dec.Decode(&jsonRequest)
			if err != nil {
				t.Fatal(err)
			}
			eq := reflect.DeepEqual(jsonRequest, test.expectedRequestBody)
			if !eq {
				t.Fatalf("expected %#v, got %#v", test.expectedRequestBody, jsonRequest)
			}
		})
	}

}

func initMixpanelForTests(t *testing.T) *Mixpanel {
	// Read Mixpanel settings from environment variables, then prepare
	// the settings that will be passed to the connector.
	projectID := os.Getenv("MEERGO_TEST_MIXPANEL_PROJECT_ID")
	if projectID == "" {
		t.Fatal("env var MEERGO_TEST_MIXPANEL_PROJECT_ID is required but not provided")
	}
	projectToken := os.Getenv("MEERGO_TEST_MIXPANEL_PROJECT_TOKEN")
	if projectToken == "" {
		t.Fatal("env var MEERGO_TEST_MIXPANEL_PROJECT_TOKEN is required but not provided")
	}

	sett := innerSettings{
		ProjectID:           projectID,
		ProjectToken:        projectToken,
		UseEuropeanEndpoint: false,
	}
	settings, err := json.Marshal(sett)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate the Mixpanel connector, with a specific settings for testing.
	app, err := testutils.NewAppConnectorForTests("Mixpanel", settings)
	if err != nil {
		t.Fatal(err)
	}

	return app.(*Mixpanel)
}

// newEventForTest returns a new *meergo.Event that can be used in tests.
// This utility function helps reduce test code and makes it easier to extend.
// The choice of parameters is convenience-driven and may change over time
// depending on how each test is parameterized.
func newEventForTest(eventType, anonymousID, messageID string, now time.Time) *meergo.Event {
	eventID := uuid.NewString()
	receivedEventID := uuid.NewString()

	return &meergo.Event{
		ID: eventID,
		Received: events.ReceivedEvent(map[string]any{
			"anonymousId": anonymousID,
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
			"id":                receivedEventID,
			"messageId":         messageID,
			"originalTimestamp": now,
			"previousId":        "IAJVLPBEZJ",
			"receivedAt":        now,
			"sentAt":            now.Add(-10 * time.Millisecond),
			"timestamp":         now,
			"type":              "alias",
			"userId":            nil,
		}),
		Type: struct {
			ID     string
			Schema types.Type
			Values map[string]any
		}{
			ID: "track",
			Schema: types.Object([]types.Property{
				{Name: "event", Prefilled: "event", Type: types.Text().WithCharLen(255), CreateRequired: true, Description: "Event Name"},
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
}
