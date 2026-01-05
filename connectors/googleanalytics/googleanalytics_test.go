// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package googleanalytics

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/testconnector"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

func TestSendEvents(t *testing.T) {

	// Read Google Analytics settings from environment variables, then prepare
	// the settings that will be passed to the connector.
	var s innerSettings
	s.MeasurementID = os.Getenv("MEERGO_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID")
	if s.MeasurementID == "" {
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID is required but not provided")
	}
	s.APISecret = os.Getenv("MEERGO_TEST_GOOGLE_ANALYTICS_API_SECRET")
	if s.APISecret == "" {
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_API_SECRET is required but not provided")
	}
	s.CollectionEndpoint = os.Getenv("MEERGO_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT")
	switch s.CollectionEndpoint {
	case "Global", "EU":
	case "":
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT is required but not provided")
	default:
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT can only be either Global or EU")

	}
	settings, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate the connector for Google Analytics, with a specific configuration for testing.
	app, err := testconnector.NewApplication("google-analytics", settings)
	if err != nil {
		t.Fatal(err)
	}

	ga := app.(*Analytics)

	ctx := context.Background()

	now := time.Now().UTC()

	tests := []struct {
		events              []*connectors.Event
		expectedRequestBody map[string]any
	}{
		{
			events: []*connectors.Event{
				{
					Received: testconnector.ReceivedEvent(map[string]any{
						"anonymousId":  "17fba6ee-8673-4ebc-afd6-69e62124e017",
						"connectionId": 1323607634,
						"context": map[string]any{
							"browser": map[string]any{
								"name":  "Other",
								"other": "Unknown",
							},
							"ip": "127.0.0.1",
							"os": map[string]any{
								"name": "Other",
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
						ID: "ad_impression",
						Schema: types.Object([]types.Property{
							{Name: "ad_platform", Type: types.String()},
							{Name: "ad_source", Type: types.String()},
							{Name: "ad_format", Type: types.String()},
							{Name: "ad_unit_name", Type: types.String()},
							{Name: "currency", Type: types.String()},
							{Name: "value", Type: genericNumberType},
						}),
						Values: map[string]any{},
					},
				},
			},
			expectedRequestBody: map[string]any{
				"client_id": "17fba6ee-8673-4ebc-afd6-69e62124e017",
				"events": []any{
					map[string]any{
						"name":             "ad_impression",
						"params":           map[string]any{},
						"timestamp_micros": float64(now.UnixMicro()),
					},
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
			events := testconnector.NewEventsIterator(test.events)

			// Actually sends the events.
			t.Log("calling SendEvents")
			err = ga.SendEvents(ctx, events)
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
			var jsonRequest map[string]any
			err = json.Decode(req.Body, &jsonRequest)
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
