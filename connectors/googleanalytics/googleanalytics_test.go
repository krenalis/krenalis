// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package googleanalytics

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

func TestSendEvents(t *testing.T) {

	// Read Google Analytics settings from environment variables, then prepare
	// the settings that will be passed to the connector.
	var s innerSettings
	s.MeasurementID = os.Getenv("KRENALIS_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID")
	if s.MeasurementID == "" {
		t.Fatal("env var KRENALIS_TEST_GOOGLE_ANALYTICS_MEASUREMENT_ID is required but not provided")
	}
	s.APISecret = os.Getenv("KRENALIS_TEST_GOOGLE_ANALYTICS_API_SECRET")
	if s.APISecret == "" {
		t.Fatal("env var KRENALIS_TEST_GOOGLE_ANALYTICS_API_SECRET is required but not provided")
	}
	s.CollectionEndpoint = os.Getenv("KRENALIS_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT")
	switch s.CollectionEndpoint {
	case "Global", "EU":
	case "":
		t.Fatal("env var KRENALIS_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT is required but not provided")
	default:
		t.Fatal("env var KRENALIS_TEST_GOOGLE_ANALYTICS_COLLECTION_ENDPOINT can only be either Global or EU")

	}
	settings, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate the connector for Google Analytics, with a specific configuration for testing.
	ga, err := testconnector.NewApplication[*Analytics]("google-analytics", settings)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()

	now := time.Now().UTC()

	pageViewReceived := map[string]any{
		"anonymousId":  "17fba6ee-8673-4ebc-afd6-69e62124e017",
		"connectionId": 1323607634,
		"context": map[string]any{
			"page": map[string]any{
				"referrer": "https://example.com/",
				"title":    "Analytics Academy",
				"url":      "https://example.com/academy/",
			},
		},
		"messageId":         "1427b912-438f-46a8-ae7f-b276ee5345ee",
		"originalTimestamp": now,
		"receivedAt":        now,
		"sentAt":            now.Add(-10 * time.Millisecond),
		"timestamp":         now,
		"type":              "page",
		"userId":            nil,
	}
	pageViewSchema := eventTypeByID["page_view"].Schema
	pageViewValues, err := testconnector.TransformEvent(pageViewSchema, pageViewReceived, nil)
	if err != nil {
		t.Fatal(err)
	}

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
		{
			events: []*connectors.Event{
				{
					Received: testconnector.ReceivedEvent(pageViewReceived),
					Type: connectors.EventTypeInfo{
						ID:     "page_view",
						Schema: pageViewSchema,
						Values: pageViewValues,
					},
				},
			},
			expectedRequestBody: map[string]any{
				"client_id": "17fba6ee-8673-4ebc-afd6-69e62124e017",
				"events": []any{
					map[string]any{
						"name": "page_view",
						"params": map[string]any{
							"page_location":        "https://example.com/academy/",
							"page_referrer":        "https://example.com/",
							"page_title":           "Analytics Academy",
							"engagement_time_msec": 1.0,
						},
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

func TestPageViewEventTypeDefinition(t *testing.T) {

	ga := &Analytics{}

	schema, err := ga.EventTypeSchema(context.Background(), "page_view")
	if err != nil {
		t.Fatal(err)
	}

	for name, prefilled := range map[string]string{
		"page_location":        "context.page.url",
		"page_referrer":        "context.page.referrer",
		"page_title":           "context.page.title",
		"engagement_time_msec": "1",
	} {
		property, ok := schema.Properties().ByName(name)
		if !ok {
			t.Fatalf("expected page_view schema to include %q", name)
		}
		if property.Prefilled != prefilled {
			t.Fatalf("expected %q prefilled to be %q, got %q", name, prefilled, property.Prefilled)
		}
		if property.CreateRequired {
			t.Fatalf("expected %q not to be required", name)
		}
	}

	received := map[string]any{
		"context": map[string]any{
			"page": map[string]any{
				"referrer": "https://example.com/",
				"title":    "Analytics Academy",
				"url":      "https://example.com/academy/",
			},
		},
		"type": "page",
	}
	values, err := testconnector.TransformEvent(schema, received, nil)
	if err != nil {
		t.Fatal(err)
	}
	params, err := types.Marshal(values, schema)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err = json.Decode(bytes.NewReader(params), &got); err != nil {
		t.Fatal(err)
	}

	expected := map[string]any{
		"page_location":        "https://example.com/academy/",
		"page_referrer":        "https://example.com/",
		"page_title":           "Analytics Academy",
		"engagement_time_msec": 1.0,
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected %#v, got %#v", expected, got)
	}
}
