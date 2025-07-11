//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package googleanalytics

import (
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
)

func TestSendEvents(t *testing.T) {

	// Read Google Analytics settings from environment variables, then prepare
	// the settings that will be passed to the connector.
	measurementID := os.Getenv("MEERGO_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID")
	if measurementID == "" {
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID is required but not provided")
	}
	apiSecret := os.Getenv("MEERGO_TEST_GOOGLE_ANALYTICS_API_SECRET")
	if apiSecret == "" {
		t.Fatal("env var MEERGO_TEST_GOOGLE_ANALYTICS_API_SECRET is required but not provided")
	}
	sett := innerSettings{
		MeasurementID: measurementID,
		APISecret:     apiSecret,
	}
	settings, err := json.Marshal(sett)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate the Google Analytics connector, with a specific configuration
	// for testing.
	app, err := testutils.NewAppConnectorForTests("Google Analytics", settings)
	if err != nil {
		t.Fatal(err)
	}

	ga := app.(*Analytics)

	ctx := context.Background()

	now := time.Now().UTC()

	tests := []struct {
		events              []*meergo.Event
		expectedRequestBody map[string]any
	}{
		{
			events: []*meergo.Event{
				{
					ID:   "7ba8676a-3182-4d76-bf6e-21483fc63893",
					Type: "ad_impression",
					Schema: types.Object([]types.Property{
						{Name: "ad_platform", Type: types.Text()},
						{Name: "ad_source", Type: types.Text()},
						{Name: "ad_format", Type: types.Text()},
						{Name: "ad_unit_name", Type: types.Text()},
						{Name: "currency", Type: types.Text()},
						{Name: "value", Type: genericNumberType},
					}),
					Properties: map[string]any{},
					Raw: events.RawEvent(map[string]any{
						"anonymousId": "17fba6ee-8673-4ebc-afd6-69e62124e017",
						"connection":  1323607634,
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
				},
			},
			expectedRequestBody: map[string]any{
				"client_id": "17fba6ee-8673-4ebc-afd6-69e62124e017",
				"events": []any{
					map[string]any{
						"name":             "ad_impression",
						"params":           map[string]any{},
						"timestamp_micros": json.Number(strconv.Itoa(int(now.UnixMicro()))),
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
			events := testutils.NewEventsIterator(test.events)

			// Actually sends the events.
			t.Log("calling SendEvents")
			err = ga.SendEvents(ctx, events)
			if err != nil {
				t.Fatal(err)
			}
			t.Log("SendEvent returned no errors")

			// Check that the HTTP request was actually set by the test.
			if req.Method == "" {
				t.Fatal("the 'SendEvents' function did not properly stored the HTTP request")
			}

			// Decode the request body and ensure it matches the expected result.
			var jsonRequest map[string]any
			dec := json.NewDecoder(req.Body)
			dec.UseNumber()
			err := dec.Decode(&jsonRequest)
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
