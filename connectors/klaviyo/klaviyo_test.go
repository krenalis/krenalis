//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package klaviyo

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/testconnector"
)

func TestSendEvents(t *testing.T) {

	// Read Klaviyo settings from environment variables, then prepare
	// the settings that will be passed to the connector.
	privateAPIKey := os.Getenv("MEERGO_TEST_KLAVIYO_PRIVATE_API_KEY")
	if privateAPIKey == "" {
		t.Fatal("env var MEERGO_TEST_KLAVIYO_PRIVATE_API_KEY is required but not provided")
	}
	sett := innerSettings{
		PrivateAPIKey: privateAPIKey,
	}
	settings, err := json.Marshal(sett)
	if err != nil {
		t.Fatal(err)
	}

	// Instantiate the Klaviyo connector, with a specific settings for testing.
	app, err := testconnector.NewApp("klaviyo", settings)
	if err != nil {
		t.Fatal(err)
	}

	ky := app.(*Klaviyo)

	ctx := context.Background()

	now := time.Now().UTC().Truncate(time.Millisecond)

	schema, err := ky.EventTypeSchema(context.Background(), "create_event")
	if err != nil {
		t.Fatalf("EventTypeSchema returned error %q", err)
	}

	tests := []struct {
		events              []*meergo.Event
		expectedRequestBody map[string]any
	}{
		{
			events: []*meergo.Event{
				{
					ID: "019813cc-6cbb-77a5-9e13-e57724067288",
					Received: testconnector.ReceivedEvent(map[string]any{
						"anonymousId": "199c664f-66ad-49d8-a088-fadd0f1a7acf",
						"connection":  347182063,
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
						"id":                "34122265-1ad6-47e0-8934-ba50c5f395e5",
						"messageId":         "8f8e652a-2518-4953-a2ed-c70e0894c791",
						"originalTimestamp": now,
						"previousId":        "IBEN72PV6",
						"receivedAt":        now,
						"sentAt":            now.Add(-10 * time.Millisecond),
						"timestamp":         now,
						"type":              "alias",
						"userId":            nil,
					}),
					Type: meergo.EventTypeInfo{
						ID:     "create_event",
						Schema: schema,
						Values: map[string]any{
							"metric_name":    "Placed Order",
							"email":          "bill.joel@klaviyo-demo.com",
							"value":          9.99,
							"value_currency": "USD",
							"properties": map[string]any{
								"Brand":      json.Value(`"Kids Book"`),
								"Categories": json.Value(`["Fiction","Children"]`),
							},
						},
					},
				},
			},
			expectedRequestBody: map[string]any{
				"data": map[string]any{
					"type": "event-bulk-create-job",
					"attributes": map[string]any{
						"events-bulk-create": map[string]any{
							"data": []any{
								map[string]any{
									"type": "event-bulk-create",
									"attributes": map[string]any{
										"profile": map[string]any{
											"data": map[string]any{
												"type": "profile",
												"attributes": map[string]any{
													"email": "bill.joel@klaviyo-demo.com",
												},
											},
										},
										"events": map[string]any{
											"data": []any{
												map[string]any{
													"type": "event",
													"attributes": map[string]any{
														"properties": map[string]any{
															"Brand": "Kids Book",
															"Categories": []any{
																"Fiction",
																"Children",
															},
														},
														"time":           now.Format(time.RFC3339Nano),
														"value":          9.99,
														"value_currency": "USD",
														"unique_id":      "019813cc-6cbb-77a5-9e13-e57724067288",
														"metric": map[string]any{
															"data": map[string]any{
																"type": "metric",
																"attributes": map[string]any{
																	"name": "Placed Order",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		// Event without properties.
		{
			events: []*meergo.Event{
				{
					ID: "019813cc-6cbb-77a5-9e13-e57724067288",
					Received: testconnector.ReceivedEvent(map[string]any{
						"anonymousId": "199c664f-66ad-49d8-a088-fadd0f1a7acf",
						"connection":  347182063,
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
						"id":                "34122265-1ad6-47e0-8934-ba50c5f395e5",
						"messageId":         "8f8e652a-2518-4953-a2ed-c70e0894c791",
						"originalTimestamp": now,
						"previousId":        "IBEN72PV6",
						"receivedAt":        now,
						"sentAt":            now.Add(-10 * time.Millisecond),
						"timestamp":         now,
						"type":              "alias",
						"userId":            nil,
					}),
					Type: meergo.EventTypeInfo{
						ID:     "create_event",
						Schema: schema,
						Values: map[string]any{
							"metric_name":    "Placed Order",
							"email":          "bill.joel@klaviyo-demo.com",
							"value":          9.99,
							"value_currency": "USD",
						},
					},
				},
			},
			expectedRequestBody: map[string]any{
				"data": map[string]any{
					"type": "event-bulk-create-job",
					"attributes": map[string]any{
						"events-bulk-create": map[string]any{
							"data": []any{
								map[string]any{
									"type": "event-bulk-create",
									"attributes": map[string]any{
										"profile": map[string]any{
											"data": map[string]any{
												"type": "profile",
												"attributes": map[string]any{
													"email": "bill.joel@klaviyo-demo.com",
												},
											},
										},
										"events": map[string]any{
											"data": []any{
												map[string]any{
													"type": "event",
													"attributes": map[string]any{
														"properties":     map[string]any{},
														"time":           now.Format(time.RFC3339Nano),
														"value":          9.99,
														"value_currency": "USD",
														"unique_id":      "019813cc-6cbb-77a5-9e13-e57724067288",
														"metric": map[string]any{
															"data": map[string]any{
																"type": "metric",
																"attributes": map[string]any{
																	"name": "Placed Order",
																},
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
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
			err = ky.SendEvents(ctx, events)
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
			dec := json.NewDecoder(req.Body)
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

func TestValidateProperty(t *testing.T) {
	type test struct {
		input   string
		wantErr bool
		name    string
	}

	var maxInt64 = fmt.Sprintf("%d", math.MaxInt64)
	var minInt64 = fmt.Sprintf("%d", math.MinInt64)
	var overMaxInt64 = fmt.Sprintf("%d0", math.MaxInt64)
	var underMinInt64 = fmt.Sprintf("%d0", math.MinInt64)

	tests := []test{
		// Normal/Valid cases
		{`null`, false, "Null"},
		{`[1,2,3]`, false, "Valid small array"},
		{`{"foo": "bar"}`, false, "Valid simple object"},
		{`true`, false, "Boolean true"},
		{`false`, false, "Boolean false"},
		{`null`, false, "Null value"},
		{`"ciao"`, false, "Short string"},
		{`56`, false, "Integer"},
		{`2.891`, false, "Number"},
		{`"` + strings.Repeat("a", maxStringLen) + `"`, false, "Long string"},
		{`"` + strings.Repeat("🍕", maxStringLen) + `"`, false, "Long UTF-8 emoji string"},
		{maxInt64, false, "Max int64"},
		{minInt64, false, "Min int64"},
		{`[1, [2, [3, [4]]]]`, false, "Valid depth"},
		{`[1, [2, [3, [4, [5, [6, [7, [8, [9]]]]]]]]]`, false, "Not too deep (9)"},
		{`{"a": {"b": {"c": {"d": 1}}}}`, false, "Valid object depth"},
		{buildMixedDepth(9), false, "Valid mixed depth"},

		// Edge and invalid cases
		{`[` + genManyElems(4001) + `]`, true, "Array too long"},
		{`"` + strings.Repeat("a", maxStringLen+1) + `"`, true, "String too long"},
		{overMaxInt64, true, "Integer overflow positive"},
		{underMinInt64, true, "Integer overflow negative"},
		{`123.5`, false, "Float allowed"},
		{`[1, [2, [3, [4, [5, [6, [7, [8, [9, [10]]]]]]]]]]`, true, "Too deep (10)"},
		{`"àèìòù"`, false, "UTF-8 string valid"},
		{`"` + strings.Repeat("🍕", maxStringLen+1) + `"`, true, "Long UTF-8 emoji string"},
		{buildDeepObject(10), true, "Too deep object"},
		{buildMixedDepth(10), true, "Too deep mixed"},
	}

	for _, tc := range tests {
		err := validateProperty(json.Value(tc.input))
		if tc.wantErr && err == nil {
			t.Errorf("%s: expected error but got nil", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("%s: unexpected error: %v", tc.name, err)
		}
	}
}

func genManyElems(n int) string {
	var s strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			s.WriteByte(',')
		}
		s.WriteByte('1')
	}
	return s.String()
}

func buildDeepArray(depth int) string {
	s := "0"
	for i := 0; i < depth; i++ {
		s = "[" + s + "]"
	}
	return s
}

func buildDeepObject(depth int) string {
	s := "0"
	for i := 0; i < depth; i++ {
		s = `{"a":` + s + `}`
	}
	return s
}

func buildMixedDepth(depth int) string {
	s := "0"
	for i := 0; i < depth; i++ {
		if i%2 == 0 {
			s = `{"a":` + s + `}`
		} else {
			s = `[` + s + `]`
		}
	}
	return s
}
