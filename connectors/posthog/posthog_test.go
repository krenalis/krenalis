// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package posthog

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/testconnector"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/validation"

	"github.com/google/uuid"
)

func TestSendEvents(t *testing.T) {

	posthog := newPostHogForTests(t)
	endpoint := endpointURLForTests(t, posthog.settings)

	timestamp := time.Now().UTC().Truncate(time.Millisecond)

	sendAndTestEvents := func(t *testing.T, events []*connectors.Event, expectedBody json.Value) {
		t.Helper()

		req := new(http.Request)
		ctx := context.WithValue(t.Context(), testconnector.CaptureRequestContextKey, req)
		iter := testconnector.NewEventsIterator(events)

		err := posthog.SendEvents(ctx, iter)
		if err != nil {
			t.Fatalf("expected SendEvents to succeed, got %v", err)
		}

		if req.Method == "" {
			t.Fatal("expected SendEvents to capture the HTTP request, got empty request")
		}
		defer req.Body.Close()

		if req.URL.String() != endpoint {
			t.Fatalf("expected request URL %q, got %q", endpoint, req.URL.String())
		}
		if req.Method != http.MethodPost {
			t.Fatalf("expected method %q, got %q", http.MethodPost, req.Method)
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf(`expected header Content-Type to be "application/json", got %q`, ct)
		}
		if ce := req.Header.Get("Content-Encoding"); ce != "gzip" {
			t.Fatalf(`expected header Content-Encoding to be "gzip", got %q`, ce)
		}

		body := decodeRequestBody(t, req.Body)

		if !bytes.Equal(expectedBody, body) {
			t.Fatalf("unexpected request body:\nexpected %s,\ngot      %s", expectedBody, body)
		}
	}

	t.Run("identify", func(t *testing.T) {

		anonymousID := uuid.NewString()
		userID := "user_891273"
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId": 187239,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"context": map[string]any{
				"ip": "203.0.113.9",
			},
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "identify",
			"traits": marshalJSON(map[string]any{
				"active":     true,
				"email":      "sam@example.com",
				"first_name": "Sam",
				"plan":       "enterprise",
			}),
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "identify")
		if err != nil {
			t.Fatalf("expected to obtain schema for identify event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 140261,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "identify",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$identify",
					"distinct_id": userID,
					"properties": map[string]any{
						"$anon_distinct_id": anonymousID,
						"$ip":               "203.0.113.9",
						"$set": map[string]any{
							"active":     true,
							"email":      "sam@example.com",
							"first_name": "Sam",
							"plan":       "enterprise",
						},
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("group", func(t *testing.T) {

		anonymousID := uuid.NewString()
		userID := "user_73155"
		groupID := "company-413"
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId": 276219,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"groupId":      groupID,
			"context": map[string]any{
				"ip": "198.51.100.24",
			},
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "group",
			"traits": marshalJSON(map[string]any{
				"employees": 48,
				"name":      "Globex",
				"tier":      "growth",
			}),
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "group")
		if err != nil {
			t.Fatalf("expected to obtain schema for group event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 194271,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "group",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$groupidentify",
					"distinct_id": userID,
					"properties": map[string]any{
						"$group_key":  groupID,
						"$group_set":  map[string]any{"employees": 48, "name": "Globex", "tier": "growth"},
						"$group_type": "company",
						"$ip":         "198.51.100.24",
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("track", func(t *testing.T) {

		anonymousID := uuid.NewString()
		userID := "user_4891"
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId": 962351,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"event":        "Checkout Started",
			"context": map[string]any{
				"ip": "192.0.2.23",
			},
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "track",
			"properties": marshalJSON(map[string]any{
				"cart_value": 147.95,
				"coupon":     "NEW10",
				"currency":   "USD",
			}),
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "track")
		if err != nil {
			t.Fatalf("expected to obtain schema for track event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 540261,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "track",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "Checkout Started",
					"distinct_id": userID,
					"properties": map[string]any{
						"$ip":        "192.0.2.23",
						"cart_value": 147.95,
						"coupon":     "NEW10",
						"currency":   "USD",
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("page", func(t *testing.T) {

		anonymousID := uuid.NewString()
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId": 408231,
			"anonymousId":  anonymousID,
			"context": map[string]any{
				"ip": "203.0.113.5",
				"page": map[string]any{
					"url": "https://app.example.com/dashboard?from=ad",
				},
			},
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "page",
			"properties": marshalJSON(map[string]any{
				"referrer": "https://ref.example.com/",
				"title":    "Dashboard",
			}),
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "page")
		if err != nil {
			t.Fatalf("expected to obtain schema for page event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 121731,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "page",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$pageview",
					"distinct_id": anonymousID,
					"properties": map[string]any{
						"$current_url": "https://app.example.com/dashboard?from=ad",
						"$ip":          "203.0.113.5",
						"referrer":     "https://ref.example.com/",
						"title":        "Dashboard",
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("screen", func(t *testing.T) {

		anonymousID := uuid.NewString()
		userID := "user_70351"
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId":      789351,
			"anonymousId":       anonymousID,
			"userId":            userID,
			"name":              "Transactions",
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "screen",
			"properties": marshalJSON(map[string]any{
				"hasCards": true,
				"section":  "wallet",
			}),
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "screen")
		if err != nil {
			t.Fatalf("expected to obtain schema for screen event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 209715,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "screen",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$screen",
					"distinct_id": userID,
					"properties": map[string]any{
						"$geoip_disable": true,
						"$screen_name":   "Transactions",
						"hasCards":       true,
						"section":        "wallet",
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("alias", func(t *testing.T) {

		anonymousID := "anon_492"
		userID := "user_982"
		messageID := uuid.NewString()

		received := map[string]any{
			"connectionId": 902351,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"previousId":   anonymousID,
			"context": map[string]any{
				"ip": "203.0.113.99",
			},
			"messageId":         messageID,
			"originalTimestamp": timestamp,
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"timestamp":         timestamp,
			"type":              "alias",
		}

		schema, err := posthog.EventTypeSchema(t.Context(), "alias")
		if err != nil {
			t.Fatalf("expected to obtain schema for alias event, got %v", err)
		}
		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("expected TransformEvent to succeed, got %v", err)
		}

		event := &connectors.Event{
			DestinationPipeline: 973511,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "alias",
				Schema: schema,
				Values: values,
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": posthog.settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$create_alias",
					"distinct_id": anonymousID,
					"properties": map[string]any{
						"$ip":   "203.0.113.99",
						"alias": userID,
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})
}

// marshalCanonicalJSON marshals data as canonical JSON and returns it.
// It panics if data cannot be marshalled.
func marshalCanonicalJSON(data any) json.Value {
	v, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	v, err = json.Canonicalize(v)
	if err != nil {
		panic(err)
	}
	return v
}

// marshalJSON marshals data as JSON and returns it.
// It panics if data cannot be marshalled.
func marshalJSON(data any) json.Value {
	v, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	return v
}

func decodeRequestBody(t *testing.T, body io.Reader) json.Value {
	t.Helper()

	gzr, err := gzip.NewReader(body)
	if err != nil {
		t.Fatalf("expected gzipped body, got %v", err)
	}
	defer gzr.Close()

	v, err := json.NewDecoder(gzr).ReadValue()
	if err != nil {
		t.Fatalf("expected valid JSON body, got %v", err)
	}
	v, err = json.Canonicalize(v)
	if err != nil {
		t.Fatalf("expected canonical JSON body, got %v", err)
	}

	return v
}

func newPostHogForTests(t *testing.T) *PostHog {
	t.Helper()

	var s innerSettings
	s.APIKey = os.Getenv("MEERGO_TEST_POSTHOG_API_KEY")
	if s.APIKey == "" {
		t.Fatal("expected env var MEERGO_TEST_POSTHOG_API_KEY to be set, got empty")
	}
	if len(s.APIKey) != 47 || !strings.HasPrefix(s.APIKey, "phc_") {
		t.Fatalf("expected MEERGO_TEST_POSTHOG_API_KEY to look like a PostHog project API key, got %q", s.APIKey)
	}

	region := os.Getenv("MEERGO_TEST_POSTHOG_PROJECT_REGION")
	selfHostedURL := os.Getenv("MEERGO_TEST_POSTHOG_SELF_HOSTED_URL")
	switch {
	case region != "" && selfHostedURL != "":
		t.Fatal("expected a single deployment setting, got both cloud region and self-hosted URL")
	case region != "":
		switch region {
		case "US", "EU":
			s.Cloud = &cloudSettings{ProjectRegion: region}
		default:
			t.Fatalf("expected MEERGO_TEST_POSTHOG_PROJECT_REGION to be either US or EU, got %q", region)
		}
	case selfHostedURL != "":
		url, err := validation.ParseURL(selfHostedURL, validation.NoPath|validation.NoQuery)
		if err != nil {
			t.Fatalf("expected MEERGO_TEST_POSTHOG_SELF_HOSTED_URL to be a valid base URL, got %v", err)
		}
		s.SelfHosted = &selfHostedSettings{URL: url}
	default:
		t.Fatal("expected MEERGO_TEST_POSTHOG_PROJECT_REGION or MEERGO_TEST_POSTHOG_SELF_HOSTED_URL to be set, got none")
	}

	api, err := testconnector.NewAPI("posthog", s)
	if err != nil {
		t.Fatalf("expected NewAPI to succeed, got %v", err)
	}

	return api.(*PostHog)
}

func endpointURLForTests(t *testing.T, settings *innerSettings) string {
	t.Helper()
	if settings == nil {
		t.Fatal("expected test settings to be loaded, got nil")
	}
	if cloud := settings.Cloud; cloud != nil {
		switch cloud.ProjectRegion {
		case "US":
			return "https://us.i.posthog.com/batch/"
		case "EU":
			return "https://eu.i.posthog.com/batch/"
		default:
			t.Fatalf("expected projectRegion to be either US or EU, got %q", cloud.ProjectRegion)
		}
	}
	if settings.SelfHosted == nil {
		t.Fatal("expected either cloud or self-hosted settings, got none")
	}
	return settings.SelfHosted.URL + "batch/"
}
