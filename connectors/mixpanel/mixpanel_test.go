// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package mixpanel

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/testconnector"

	"github.com/google/uuid"
)

func TestSendEvents(t *testing.T) {

	mixpanel := newMixpanelForTests(t)

	timestamp := time.Now().UTC().Truncate(time.Millisecond)
	sessionID := int(timestamp.Add(-5 * time.Minute).UnixMilli())

	sendAndTestEvent := func(t *testing.T, event *connectors.Event, expected []json.Value) {
		req := new(http.Request)
		ctx := context.WithValue(t.Context(), testconnector.CaptureRequestContextKey, req)
		iter := testconnector.NewEventsIterator([]*connectors.Event{event})
		err := mixpanel.SendEvents(ctx, iter)
		if err != nil {
			t.Fatal(err)
		}
		defer req.Body.Close()
		got, err := testconnector.DecodeNDJSON(req.Body, contentEncoding)
		if err != nil {
			t.Fatal(err)
		}
		if len(expected) != len(got) {
			t.Fatalf("expected %d JSON objects, got %d objects", len(expected), len(got))
		}
		for i, v := range got {
			got, err := json.Canonicalize(v)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(expected[i], got) {
				t.Fatalf("unexpected request to Mixpanel:\nexpected %s,\ngot      %s", string(expected[i]), got)
			}
		}
	}

	t.Run("order_completed", func(t *testing.T) {

		received := map[string]any{
			"connectionId": 1323607634,
			"anonymousId":  uuid.NewString(),
			"context": map[string]any{
				"browser": map[string]any{
					"name":    "Safari",
					"version": "18.5",
				},
				"device": map[string]any{
					"advertisingId": "6D92078A-8246-4BA4-AE5B-76104861E7DC",
				},
				"ip": "192.0.2.1",
				"os": map[string]any{
					"name":    "macOS",
					"version": "15.5",
				},
				"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1",
				"session": map[string]any{
					"id": sessionID,
				},
			},
			"event":     "Order Completed",
			"messageId": uuid.NewString(),
			"properties": marshalJSON(map[string]any{
				"order_id":    "703924",
				"affiliation": "AP3383",
				"currency":    "USD",
				"revenue":     198.45,
				"coupon":      "PROMO",
				"discount":    20.78,
				"shipping":    5.0,
				"tax":         28.05,
				"value":       405.99,
				"products":    []map[string]any{{"sku": "G7NZ0I5"}, {"sku": "QN72LVRA"}},
				"other":       true,
			}),
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"originalTimestamp": timestamp,
			"timestamp":         timestamp,
			"type":              "track",
		}

		schema, err := mixpanel.EventTypeSchema(t.Context(), "order_completed")
		if err != nil {
			t.Fatal(err)
		}

		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("cannot transform the 'order_completed' event: %s", err)
		}

		event := &connectors.Event{
			DestinationAction: 242809157,
			Received:          testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "order_completed",
				Schema: schema,
				Values: values,
			},
		}

		expected := []json.Value{
			marshalCanonicalJSON(map[string]any{
				"event": "Order Completed",
				"properties": map[string]any{
					"$browser":         "Safari",
					"$browser_version": "18.5",
					"$device_id":       event.Received.AnonymousId(),
					"$insert_id":       "242809157*" + event.Received.MessageId(),
					"$ios_ifa":         "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$os":              "macOS",
					"$os_version":      "15.5",
					"$source":          "meergo",
					"affiliation":      "AP3383",
					"coupon":           "PROMO",
					"currency":         "USD",
					"discount":         20.78,
					"distinct_id":      event.Received.AnonymousId(),
					"ip":               "192.0.2.1",
					"order_id":         "703924",
					"products":         []map[string]any{{"sku": "G7NZ0I5"}, {"sku": "QN72LVRA"}},
					"revenue":          198.45,
					"session_id":       sessionID,
					"shipping":         5.0,
					"tax":              28.05,
					"time":             timestamp.UnixMilli(),
					"value":            405.99,
				},
			}),
		}

		// Send the event and test the request body.
		sendAndTestEvent(t, event, expected)

	})

	t.Run("product_purchased", func(t *testing.T) {

		received := map[string]any{
			"connectionId": 1323607634,
			"anonymousId":  uuid.NewString(),
			"context": map[string]any{
				"browser": map[string]any{
					"name":    "Safari",
					"version": "18.5",
				},
				"device": map[string]any{
					"advertisingId": "6D92078A-8246-4BA4-AE5B-76104861E7DC",
				},
				"ip": "192.0.2.1",
				"os": map[string]any{
					"name":    "macOS",
					"version": "15.5",
				},
				"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1",
				"session": map[string]any{
					"id": sessionID,
				},
			},
			"event":     "Order Completed",
			"messageId": uuid.NewString(),
			"properties": marshalJSON(map[string]any{
				"order_id":    "703924",
				"affiliation": "AP3383",
				"currency":    "USD",
				"revenue":     198.45,
				"coupon":      "PROMO",
				"discount":    20.78,
				"shipping":    5.0,
				"tax":         28.05,
				"value":       405.99,
				"products":    []map[string]any{{"sku": "G7NZ0I5"}, {"sku": "QN72LVRA"}},
				"other":       true,
			}),
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"originalTimestamp": timestamp,
			"timestamp":         timestamp,
			"type":              "track",
		}

		schema, err := mixpanel.EventTypeSchema(t.Context(), "product_purchased")
		if err != nil {
			t.Fatal(err)
		}

		values, err := testconnector.TransformEvent(schema, received, nil)
		if err != nil {
			t.Fatalf("cannot transform the 'product_purchased' event: %s", err)
		}

		event := &connectors.Event{
			DestinationAction: 148606728,
			Received:          testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "product_purchased",
				Schema: schema,
				Values: values,
			},
		}

		expected := []json.Value{
			marshalCanonicalJSON(map[string]any{
				"event": "Product Purchased",
				"properties": map[string]any{
					"$browser":         "Safari",
					"$browser_version": "18.5",
					"$device_id":       event.Received.AnonymousId(),
					"$insert_id":       "1#148606728*" + event.Received.MessageId(),
					"$ios_ifa":         "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$os":              "macOS",
					"$os_version":      "15.5",
					"$source":          "meergo",
					"distinct_id":      event.Received.AnonymousId(),
					"ip":               "192.0.2.1",
					"session_id":       sessionID,
					"sku":              "G7NZ0I5",
					"time":             timestamp.UnixMilli() + 1,
				},
			}),
			marshalCanonicalJSON(map[string]any{
				"event": "Product Purchased",
				"properties": map[string]any{
					"$browser":         "Safari",
					"$browser_version": "18.5",
					"$device_id":       event.Received.AnonymousId(),
					"$insert_id":       "2#148606728*" + event.Received.MessageId(),
					"$ios_ifa":         "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$os":              "macOS",
					"$os_version":      "15.5",
					"$source":          "meergo",
					"distinct_id":      event.Received.AnonymousId(),
					"ip":               "192.0.2.1",
					"session_id":       sessionID,
					"sku":              "QN72LVRA",
					"time":             timestamp.UnixMilli() + 2,
				},
			}),
		}

		// Send the event and test the request body.
		sendAndTestEvent(t, event, expected)

	})

	t.Run("track", func(t *testing.T) {

		received := map[string]any{
			"connectionId": 1323607634,
			"anonymousId":  uuid.NewString(),
			"context": map[string]any{
				"browser": map[string]any{
					"name":    "Safari",
					"version": "18.5",
				},
				"device": map[string]any{
					"advertisingId": "6D92078A-8246-4BA4-AE5B-76104861E7DC",
				},
				"ip": "192.0.2.1",
				"os": map[string]any{
					"name":    "macOS",
					"version": "15.5",
				},
				"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1",
				"session": map[string]any{
					"id": sessionID,
				},
			},
			"event":     "Product Viewed",
			"messageId": uuid.NewString(),
			"properties": marshalJSON(map[string]any{
				"product_id": 803916,
			}),
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"originalTimestamp": timestamp,
			"timestamp":         timestamp,
			"type":              "track",
		}

		schema, err := mixpanel.EventTypeSchema(t.Context(), "track")
		if err != nil {
			t.Fatal(err)
		}

		values, err := testconnector.TransformEvent(schema, received, map[string]string{
			"event":      "event",
			"properties": `map("product_id",properties.product_id)`,
		})
		if err != nil {
			t.Fatalf("cannot transform the 'track' event: %s", err)
		}

		event := &connectors.Event{
			DestinationAction: 140861001,
			Received:          testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "track",
				Schema: schema,
				Values: values,
			},
		}

		expected := []json.Value{
			marshalCanonicalJSON(map[string]any{
				"event": "Product Viewed",
				"properties": map[string]any{
					"$browser":         "Safari",
					"$browser_version": "18.5",
					"$device_id":       event.Received.AnonymousId(),
					"$insert_id":       "140861001*" + event.Received.MessageId(),
					"$ios_ifa":         "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$os":              "macOS",
					"$os_version":      "15.5",
					"$source":          "meergo",
					"distinct_id":      event.Received.AnonymousId(),
					"ip":               "192.0.2.1",
					"product_id":       803916,
					"session_id":       sessionID,
					"time":             timestamp.UnixMilli(),
				},
			}),
		}

		// Send the event and test the request body.
		sendAndTestEvent(t, event, expected)

	})

	t.Run("page", func(t *testing.T) {

		received := map[string]any{
			"connectionId": 1323607634,
			"anonymousId":  uuid.NewString(),
			"context": map[string]any{
				"browser": map[string]any{
					"name":    "Safari",
					"version": "18.5",
				},
				"device": map[string]any{
					"advertisingId": "6D92078A-8246-4BA4-AE5B-76104861E7DC",
				},
				"ip": "192.0.2.1",
				"os": map[string]any{
					"name":    "macOS",
					"version": "15.5",
				},
				"userAgent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 15_5) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Safari/605.1",
				"session": map[string]any{
					"id": sessionID,
				},
			},
			"name":      "Wireless Headphones",
			"messageId": uuid.NewString(),
			"properties": marshalJSON(map[string]any{
				"productId": "WH-001",
				"name":      "Wireless Headphones",
				"category":  "Electronics",
				"price":     99.99,
				"currency":  "USD",
			}),
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"originalTimestamp": timestamp,
			"timestamp":         timestamp,
			"type":              "page",
		}

		schema, err := mixpanel.EventTypeSchema(t.Context(), "page")
		if err != nil {
			t.Fatal(err)
		}

		values, err := testconnector.TransformEvent(schema, received, map[string]string{
			"event": `"Viewed " name`,
			"properties": `map(` +
				`"category",properties.category,` +
				`"currency",properties.currency,` +
				`"name",properties.name,` +
				`"price",properties.price,` +
				`"product_id",properties.productId)`,
		})
		if err != nil {
			t.Fatalf("cannot transform the 'page' event: %s", err)
		}

		event := &connectors.Event{
			DestinationAction: 2094515358,
			Received:          testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "page",
				Schema: schema,
				Values: values,
			},
		}

		expected := []json.Value{
			marshalCanonicalJSON(map[string]any{
				"event": "Viewed Wireless Headphones",
				"properties": map[string]any{
					"$browser":         "Safari",
					"$browser_version": "18.5",
					"$device_id":       event.Received.AnonymousId(),
					"$ios_ifa":         "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$insert_id":       "2094515358*" + event.Received.MessageId(),
					"$os":              "macOS",
					"$os_version":      "15.5",
					"$source":          "meergo",
					"category":         "Electronics",
					"currency":         "USD",
					"distinct_id":      event.Received.AnonymousId(),
					"ip":               "192.0.2.1",
					"name":             "Wireless Headphones",
					"price":            99.99,
					"product_id":       "WH-001",
					"session_id":       sessionID,
					"time":             timestamp.UnixMilli(),
				},
			}),
		}

		// Send the event and test the request body.
		sendAndTestEvent(t, event, expected)

	})

	t.Run("screen", func(t *testing.T) {

		received := map[string]any{
			"connectionId": 1323607634,
			"anonymousId":  uuid.NewString(),
			"context": map[string]any{
				"app": map[string]any{
					"name":      "MyFinance",
					"version":   "3.4.1",
					"build":     "3410",
					"namespace": "com.mycompany.myfinance",
				},
				"device": map[string]any{
					"id":            "AEBE52E7-03EE-455A-B3C4-E57283966239",
					"manufacturer":  "Apple",
					"model":         "iPhone 16 Pro",
					"name":          "iPhone",
					"type":          "ios",
					"advertisingId": "6D92078A-8246-4BA4-AE5B-76104861E7DC",
				},
				"ip": "192.0.2.1",
				"os": map[string]any{
					"name":    "iOS",
					"version": "18",
				},
				"session": map[string]any{
					"id":    sessionID,
					"start": true,
				},
			},
			"name":      "Transaction History",
			"messageId": uuid.NewString(),
			"properties": marshalJSON(map[string]any{
				"filter":            "Last 30 days",
				"totalTransactions": 42,
				"page":              1,
			}),
			"receivedAt":        timestamp,
			"sentAt":            timestamp.Add(-10 * time.Millisecond),
			"originalTimestamp": timestamp,
			"timestamp":         timestamp,
			"type":              "screen",
			"userId":            "BN8204913066K",
		}

		schema, err := mixpanel.EventTypeSchema(t.Context(), "screen")
		if err != nil {
			t.Fatal(err)
		}

		values, err := testconnector.TransformEvent(schema, received, map[string]string{
			"event":      `"Viewed " name`,
			"properties": `map("filter",properties.filter,"transactionCount",properties.totalTransactions)`,
		})
		if err != nil {
			t.Fatalf("cannot transform the 'screen' event: %s", err)
		}

		event := &connectors.Event{
			DestinationAction: 2023196674,
			Received:          testconnector.ReceivedEvent(received),
			Type: connectors.EventTypeInfo{
				ID:     "screen",
				Schema: schema,
				Values: values,
			},
		}

		expected := []json.Value{
			marshalCanonicalJSON(map[string]any{
				"event": "Viewed Transaction History",
				"properties": map[string]any{
					"$app_build_number":   "3410",
					"$app_name":           "MyFinance",
					"$app_namespace":      "com.mycompany.myfinance",
					"$app_version_string": "3.4.1",
					"$device":             "iPhone",
					"$device_id":          event.Received.AnonymousId(),
					"$device_name":        "iPhone",
					"$device_type":        "ios",
					"$ios_ifa":            "6D92078A-8246-4BA4-AE5B-76104861E7DC",
					"$insert_id":          "2023196674*" + event.Received.MessageId(),
					"$manufacturer":       "Apple",
					"$model":              "iPhone 16 Pro",
					"$os":                 "iOS",
					"$os_version":         "18",
					"$source":             "meergo",
					"$user_id":            "BN8204913066K",
					"device_id":           "AEBE52E7-03EE-455A-B3C4-E57283966239",
					"distinct_id":         "BN8204913066K",
					"filter":              "Last 30 days",
					"ip":                  "192.0.2.1",
					"session_id":          sessionID,
					"time":                timestamp.UnixMilli(),
					"transactionCount":    42,
				},
			}),
		}

		// Send the event and test the request body.
		sendAndTestEvent(t, event, expected)

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

func newMixpanelForTests(t *testing.T) *Mixpanel {
	var s innerSettings
	s.ProjectID = os.Getenv("MEERGO_TEST_MIXPANEL_PROJECT_ID")
	if s.ProjectID == "" {
		t.Fatal("env var MEERGO_TEST_MIXPANEL_PROJECT_ID is required but not provided")
	}
	s.ProjectToken = os.Getenv("MEERGO_TEST_MIXPANEL_PROJECT_TOKEN")
	if s.ProjectToken == "" {
		t.Fatal("env var MEERGO_TEST_MIXPANEL_PROJECT_TOKEN is required but not provided")
	}
	s.DataResidency = os.Getenv("MEERGO_TEST_MIXPANEL_DATA_RESIDENCY")
	switch s.DataResidency {
	case "US", "EU", "IN":
	case "":
		t.Fatal("env var MEERGO_TEST_MIXPANEL_DATA_RESIDENCY is required but not provided")
	default:
		t.Fatal("env var MEERGO_TEST_MIXPANEL_DATA_RESIDENCY can only be either US, EU, or IN")
	}
	api, err := testconnector.NewAPI("mixpanel", s)
	if err != nil {
		t.Fatal(err)
	}
	return api.(*Mixpanel)
}
