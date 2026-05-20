// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package posthog

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/validation"

	"github.com/google/uuid"
)

func TestSendEvents(t *testing.T) {

	posthog, settings := newPostHogForTests(t)
	endpoint := endpointURLForTests(t, settings)

	timestamp := time.Now().UTC().Truncate(time.Millisecond)
	const sessionID int64 = 1736972405123

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

		anonymousID := "anon-identify-01"
		userID := "user_891273"
		messageID := uuid.NewString()
		sessionUUID, err := makeSessionUUIDv7(anonymousID, sessionID)
		if err != nil {
			t.Fatalf("expected session UUID generation to succeed, got %v", err)
		}

		received := map[string]any{
			"connectionId": 187239,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"context": map[string]any{
				"ip": "2001:db8:face:12::1",
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
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
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$identify",
					"distinct_id": userID,
					"properties": map[string]any{
						"$anon_distinct_id": anonymousID,
						"$ip":               "2001:db8:face:12::1",
						"$set": map[string]any{
							"active":     true,
							"email":      "sam@example.com",
							"first_name": "Sam",
							"plan":       "enterprise",
						},
						"$session_id": sessionUUID,
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})

	t.Run("group", func(t *testing.T) {

		anonymousID := "anon-group-01"
		userID := "user_73155"
		groupID := "company-413"
		messageID := uuid.NewString()
		sessionUUID, err := makeSessionUUIDv7(anonymousID, sessionID)
		if err != nil {
			t.Fatalf("expected session UUID generation to succeed, got %v", err)
		}

		received := map[string]any{
			"connectionId": 276219,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"groupId":      groupID,
			"context": map[string]any{
				"ip": "198.51.100.24",
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
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
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$groupidentify",
					"distinct_id": userID,
					"properties": map[string]any{
						"$group_key":  groupID,
						"$group_set":  map[string]any{"employees": 48, "name": "Globex", "tier": "growth"},
						"$group_type": "company",
						"$ip":         "198.51.100.24",
						"$session_id": sessionUUID,
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
		const explicitSessionID = "01946b9f-859b-7cce-ab5c-f9e68680be6e"

		received := map[string]any{
			"connectionId": 962351,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"event":        "Checkout Started",
			"context": map[string]any{
				"ip": "192.0.2.23",
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
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
		event.Type.Values["session_id"] = explicitSessionID

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "Checkout Started",
					"distinct_id": userID,
					"properties": map[string]any{
						"$ip":         "192.0.2.23",
						"$session_id": explicitSessionID,
						"cart_value":  147.95,
						"coupon":      "NEW10",
						"currency":    "USD",
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
		sessionUUID, err := makeSessionUUIDv7(anonymousID, sessionID)
		if err != nil {
			t.Fatalf("expected session UUID generation to succeed, got %v", err)
		}

		received := map[string]any{
			"connectionId": 408231,
			"anonymousId":  anonymousID,
			"context": map[string]any{
				"ip": "203.0.113.5",
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
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
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$pageview",
					"distinct_id": anonymousID,
					"properties": map[string]any{
						"$current_url": "https://app.example.com/dashboard?from=ad",
						"$ip":          "203.0.113.5",
						"$session_id":  sessionUUID,
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
		sessionUUID, err := makeSessionUUIDv7(anonymousID, sessionID)
		if err != nil {
			t.Fatalf("expected session UUID generation to succeed, got %v", err)
		}

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
			"context": map[string]any{
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
			},
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
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$screen",
					"distinct_id": userID,
					"properties": map[string]any{
						"$geoip_disable": true,
						"$screen_name":   "Transactions",
						"$session_id":    sessionUUID,
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
		sessionUUID, err := makeSessionUUIDv7(anonymousID, sessionID)
		if err != nil {
			t.Fatalf("expected session UUID generation to succeed, got %v", err)
		}

		received := map[string]any{
			"connectionId": 902351,
			"anonymousId":  anonymousID,
			"userId":       userID,
			"previousId":   anonymousID,
			"context": map[string]any{
				"ip": "203.0.113.99",
				"session": map[string]any{
					"id":    int(sessionID),
					"start": true,
				},
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

		event := &connectors.Event{
			DestinationPipeline: 973511,
			Received:            testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "alias",
				Schema: schema,
				Values: map[string]any{},
			},
		}

		expectedBody := marshalCanonicalJSON(map[string]any{
			"api_key": settings.APIKey,
			"batch": []any{
				map[string]any{
					"event":       "$create_alias",
					"distinct_id": anonymousID,
					"properties": map[string]any{
						"$ip":         "203.0.113.99",
						"$session_id": sessionUUID,
						"alias":       userID,
					},
					"timestamp": timestamp.Format(time.RFC3339),
					"uuid":      messageID,
				},
			},
		})

		sendAndTestEvents(t, []*connectors.Event{event}, expectedBody)
	})
}

func TestMakeSessionUUIDv7(t *testing.T) {

	cases := []struct {
		name        string
		anonymousID string
		sessionID   int64
		want        string
	}{
		{
			name:        "uuid_input_anonymous_id",
			anonymousID: "9f3c2c2a-1d4b-4b7c-9c2a-8a3f6d3c0e91",
			sessionID:   1734260905123,
			want:        "0193ca01-50bb-7b17-acea-34706793f387",
		},
		{
			name:        "regression_known_output",
			anonymousID: "anon-123456789",
			sessionID:   1736972405123,
			want:        "01946b9f-859b-7cce-ab5c-f9e68680be6e",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Helper()

			got, err := makeSessionUUIDv7(tc.anonymousID, tc.sessionID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tc.want != "" && got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}

			parsed, err := uuid.Parse(got)
			if err != nil {
				t.Fatalf("expected a valid UUID, got %v", err)
			}
			if parsed.Version() != 7 {
				t.Fatalf("expected version 7, got %d", parsed.Version())
			}
			if parsed.Variant() != uuid.RFC4122 {
				t.Fatalf("expected RFC4122 variant, got %d", parsed.Variant())
			}

			if ts := uuidTimestamp(parsed); ts != tc.sessionID-1000 {
				t.Fatalf("expected timestamp %d, got %d", tc.sessionID-1000, ts)
			}

			got2, err := makeSessionUUIDv7(tc.anonymousID, tc.sessionID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got2 != got {
				t.Fatalf("expected deterministic UUIDs, got %s and %s", got, got2)
			}

			seed := sha256.Sum256([]byte(tc.anonymousID + "|" + strconv.FormatInt(tc.sessionID, 10)))
			var expectedUUID uuid.UUID
			copy(expectedUUID[6:], seed[:10])
			expectedUUID[6] = (expectedUUID[6] & 0x0f) | 0x70
			expectedUUID[8] = (expectedUUID[8] & 0x3f) | 0x80
			if !bytes.Equal(parsed[6:], expectedUUID[6:]) {
				t.Fatalf("expected random portion %x, got %x", expectedUUID[6:], parsed[6:])
			}
		})
	}
}

func TestMakeSessionUUIDv7InvalidSession(t *testing.T) {
	_, err := makeSessionUUIDv7("anon", 999)
	if err == nil {
		t.Fatal("expected error for invalid session ID, got nil")
	}
}

func TestMakeSessionUUIDv7DifferentInputs(t *testing.T) {
	baseAnon := "9f3c2c2a-1d4b-4b7c-9c2a-8a3f6d3c0e91"
	baseSession := int64(1734260905123)

	base, err := makeSessionUUIDv7(baseAnon, baseSession)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	cases := []struct {
		name        string
		anonymousID string
		sessionID   int64
	}{
		{
			name:        "different_anonymous_id",
			anonymousID: "9f3c2c2a-1d4b-4b7c-9c2a-8a3f6d3c0e92",
			sessionID:   baseSession,
		},
		{
			name:        "different_session_id",
			anonymousID: baseAnon,
			sessionID:   baseSession + 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := makeSessionUUIDv7(tc.anonymousID, tc.sessionID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got == base {
				t.Fatalf("expected different inputs to yield different UUIDs; got same %s", got)
			}
		})
	}
}

func uuidTimestamp(u uuid.UUID) int64 {
	return int64(u[0])<<40 |
		int64(u[1])<<32 |
		int64(u[2])<<24 |
		int64(u[3])<<16 |
		int64(u[4])<<8 |
		int64(u[5])
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

func newPostHogForTests(t *testing.T) (*PostHog, *innerSettings) {
	t.Helper()

	var s innerSettings
	s.APIKey = os.Getenv("KRENALIS_TEST_POSTHOG_API_KEY")
	if s.APIKey == "" {
		t.Skip("the KRENALIS_TEST_POSTHOG_API_KEY environment variable is not present")
	}
	if len(s.APIKey) != 47 || !strings.HasPrefix(s.APIKey, "phc_") {
		t.Fatalf("expected KRENALIS_TEST_POSTHOG_API_KEY to look like a PostHog project API key, got %q", s.APIKey)
	}

	region := os.Getenv("KRENALIS_TEST_POSTHOG_PROJECT_REGION")
	selfHostedURL := os.Getenv("KRENALIS_TEST_POSTHOG_SELF_HOSTED_URL")
	switch {
	case region != "" && selfHostedURL != "":
		t.Fatal("expected a single deployment setting, got both cloud region and self-hosted URL")
	case region != "":
		switch region {
		case "US", "EU":
			s.Cloud = &cloudSettings{ProjectRegion: region}
		default:
			t.Fatalf("expected KRENALIS_TEST_POSTHOG_PROJECT_REGION to be either US or EU, got %q", region)
		}
	case selfHostedURL != "":
		url, err := validation.ParseURL(selfHostedURL, validation.NoPath|validation.NoQuery)
		if err != nil {
			t.Fatalf("expected KRENALIS_TEST_POSTHOG_SELF_HOSTED_URL to be a valid base URL, got %v", err)
		}
		s.SelfHosted = &selfHostedSettings{URL: url}
	default:
		t.Skip("the KRENALIS_TEST_POSTHOG_PROJECT_REGION and KRENALIS_TEST_POSTHOG_SELF_HOSTED_URL environment variables are not present")
	}

	ph, err := testconnector.NewApplication[*PostHog]("posthog", s)
	if err != nil {
		t.Fatalf("expected NewApplication to succeed, got %v", err)
	}

	return ph, &s
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
