//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package collector

import (
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"

	"github.com/google/go-cmp/cmp"
)

func Test_Decoder(t *testing.T) {

	writeKey := "vjJCb9lilU1GABTrSQ5qOkY7ddTW1uBQ"

	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"
	browser := map[string]any{"name": "Chrome", "other": "", "version": "117.0"}
	ip := "192.168.1.1"
	os := map[string]any{"name": "Windows", "version": "10.0"}
	library := map[string]any{"name": "meergo.js", "version": "0.0.0"}
	context := map[string]any{"browser": browser, "ip": ip, "os": os, "userAgent": userAgent}

	// These non-read optional properties are not tested if they are not present as expected.
	var nonReadOptionalProperties = []string{
		"id", "connection", "anonymousId", "context", "messageId", "receivedAt", "sentAt", "originalTimestamp", "timestamp", "type",
	}

	type expectedEvent struct {
		// Expected decoded event.
		//   - Connection: do not set; it will use the connection from the test.
		//   - ReceivedAt: do not set; it will be verified that the returned value is within a specific range.
		//   - Properties: properties in nonReadOptionalProperties are not tested if they are not present.
		event events.Event

		// Expected error from event decoding.
		err error
	}

	tests := []struct {
		typ            string
		body           string
		writeKey       string              // Leave empty if you don't want to test it.
		connection     int                 // Can be any value.
		connectionType state.ConnectorType // Defaults to "Website" if not set.
		expected       []expectedEvent     // Can be empty or nil, if no events are expected.
		err            error               // Expected error from the newDecoder function.
	}{
		{body: ``, err: errors.BadRequest("request's body is empty")},
		{body: `{`, err: errors.BadRequest("error parsing the request body as JSON: unexpected EOF")},
		{body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("property 'type' is required for a single-event request")}}},
		{body: `{"batch":null}`, err: errors.BadRequest("property 'batch' is not a valid array")},
		{body: `{"batch":{}}`, err: errors.BadRequest("property 'batch' is not a valid array")},
		{body: `{"batch":[]"}`, err: errors.BadRequest("error parsing the request body as JSON: invalid character '\"' after object value (expecting ',' or '}')")},
		{body: `{"batch":[],"writeKey":true}`, err: errors.BadRequest("property 'writeKey' is not a valid string")},
		{body: `{"batch":[],"writeKey":""}`, err: errors.BadRequest("property 'writeKey' cannot be empty")},
		{body: `{"batch":[],"writeKey":"vjJCb9lilU1GABTrSQ5qOkY7ddTW1uBQ"}`, writeKey: writeKey},
		{body: `{"batch":[]}`},
		{body: `{"b\u0061tch":[]}`},
		{body: `{"batch":[],"sentAt":""}`, err: errors.BadRequest("property 'sentAt' is not a valid ISO 8601 timestamp")},
		{body: `{"batch":[],"sentAt":"0000-01-01T12:56:23"}`, err: errors.BadRequest("property 'sentAt' has an invalid year value")},
		{body: `{"batch":[],"sentAt":"10000-01-01T12:56:23"}`, err: errors.BadRequest("property 'sentAt' has an invalid year value")},
		{body: `{"batch":[],"sentAt":"2024-10-23T14:08:07.288305712"}`},
		{body: `{"batch":[],"sentAt":"2024-10-23T14:08:07.288305712"}`},
		{body: `{"batch":[],"foo":"boo"}`},
		{body: `{"batch":[],"context":null}`, err: errors.BadRequest("property 'context' is not a valid object")},
		{body: `{"batch":[],"context":{}}`},
		{body: `{"batch":[],"context":{"foo":"boo"}}`},
		{body: `{"batch":[],"connection":-2}`, err: errors.BadRequest("property 'connection' is not a valid connection identifier")},
		{body: `{"batch":[],"connection":264826420}`},

		{typ: "track", body: ``, err: errors.BadRequest("request's body is empty")},
		{typ: "track", body: `{`, expected: []expectedEvent{{err: errors.BadRequest("unexpected invalid token while decoding an event")}}},
		{typ: "track", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("either 'anonymousId' or 'userId' properties are required for a track event")}}},
		{typ: "page", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("either 'anonymousId' or 'userId' properties are required for a page event")}}},
		{typ: "identify", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("property 'userId' is required for an identify event")}}},

		// meergo.track('click'); anonymous
		{
			body:       `{"batch":[{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}]}`,
			connection: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     context,
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"type":        "track",
					"event":       "click",
					"userId":      nil,
				},
			}},
		},
		{
			body:       `[{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}]`,
			connection: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     context,
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"type":        "track",
					"event":       "click",
					"userId":      nil,
				},
			}},
		},
		{
			typ:        "track",
			body:       `{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}`,
			connection: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     context,
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"type":        "track",
					"event":       "click",
					"userId":      nil,
				},
			}},
		},

		// meergo.identify('bob', {name: 'bob', age: 19})
		{
			body: `{"batch":[{"type":"identify","messageId":"9677e303-6a57-45e4-9c94-e47ec550a261","userId":"bob","groupId":null,"traits":{"name":"bob","age":19}}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":   context,
					"messageId": "9677e303-6a57-45e4-9c94-e47ec550a261",
					"traits":    json.Value(`{"name":"bob","age":19}`),
					"type":      "identify",
					"userId":    "bob",
				}},
			},
		},
		{
			body: `[{"type":"identify","messageId":"9677e303-6a57-45e4-9c94-e47ec550a261","userId":"bob","groupId":null,"traits":{"name":"bob","age":19}}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":   context,
					"messageId": "9677e303-6a57-45e4-9c94-e47ec550a261",
					"traits":    json.Value(`{"name":"bob","age":19}`),
					"type":      "identify",
					"userId":    "bob",
				}},
			},
		},
		{
			typ:  "identify",
			body: `{"messageId":"9677e303-6a57-45e4-9c94-e47ec550a261","userId":"bob","groupId":null,"traits":{"name":"bob","age":19}}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":   context,
					"messageId": "9677e303-6a57-45e4-9c94-e47ec550a261",
					"traits":    json.Value(`{"name":"bob","age":19}`),
					"type":      "identify",
					"userId":    "bob",
				}},
			},
		},

		// meergo.track('page')
		{
			body: `{"batch":[{"type":"page","context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"type":        "page",
					"userId":      nil,
				}},
			},
		},
		{
			body: `[{"type":"page","context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"type":        "page",
					"userId":      nil,
				}},
			},
		},
		{
			typ:  "page",
			body: `{"context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"type":        "page",
					"userId":      nil,
				}},
			},
		},

		// meergo.screen('login', {}, {traits: {name: 'Bob'}})
		{
			body: `{"batch":[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635},"traits":{"name":"Bob"}},"name":"login","userId":"bob"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"name":       "login",
					"traits":     json.Value(`{"name":"Bob"}`),
					"type":       "screen",
					"userId":     "bob",
				}},
			},
		},
		{
			body: `[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635},"traits":{"name":"Bob"}},"name":"login","userId":"bob"}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"name":       "login",
					"traits":     json.Value(`{"name":"Bob"}`),
					"type":       "screen",
					"userId":     "bob",
				}},
			},
		},
		{
			typ:  "screen",
			body: `{"context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635},"traits":{"name":"Bob"}},"name":"login","userId":"bob"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"name":       "login",
					"traits":     json.Value(`{"name":"Bob"}`),
					"type":       "screen",
					"userId":     "bob",
				}},
			},
		},

		// meergo.screen('login')
		{
			body: `{"batch":[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
					"userId":     nil,
				}},
			},
		},
		{
			body: `[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
					"userId":     nil,
				}},
			},
		},
		{
			typ:  "screen",
			body: `{"context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "browser": browser, "ip": ip, "os": os, "userAgent": userAgent},
					"properties": json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
					"userId":     nil,
				}},
			},
		},

		// meergo.track('click'); meergo.track('click');
		{
			body: `{"batch":[` +
				`{"type":"track","event":"click","timestamp":"2024-10-31T14:39:06.050Z","properties":{},"userId":null,"messageId":"8071f50d-5a69-45f7-bb31-70e111aa8aed","anonymousId":"5d60ebba-cbf6-463c-8d55-fc7a6f66183f","context":{"library":{"name":"meergo.js","version":"0.0.0"},"locale":"it-IT","page":{"path":"/catalog/","referrer":"https://listing.sample.com/","search":"","title":"Test website","url":"https://sample.com/catalog/"},"screen":{"width":2816,"height":1584,"density":1.3636363636363635},"userAgent":"Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0","sessionId":1730384955277},"integrations":{}},` +
				`{"type":"track","event":"click","timestamp":"2024-10-31T14:39:12.319Z","properties":{},"userId":null,"messageId":"1935c955-45f8-44a3-b835-ced93138e8b3","anonymousId":"5d60ebba-cbf6-463c-8d55-fc7a6f66183f","context":{"library":{"name":"meergo.js","version":"0.0.0"},"locale":"it-IT","page":{"path":"/catalog/","referrer":"https://listing.sample.com/","search":"","title":"Test website","url":"https://sample.com/catalog/"},"screen":{"width":2816,"height":1584,"density":1.3636363636363635},"userAgent":"Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0","sessionId":1730384955277},"integrations":{}}` +
				`],"sentAt":"2024-10-31T14:39:12.647Z","writeKey":"qWqwaP3zGZOazQUmuFRuRMfW3lMCqjUa"}`,
			connection: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "5d60ebba-cbf6-463c-8d55-fc7a6f66183f",
					"context": map[string]any{
						"library": library,
						"locale":  "it-IT",
						"page": map[string]any{
							"path":     "/catalog/",
							"referrer": "https://listing.sample.com/",
							"search":   "",
							"title":    "Test website",
							"url":      "https://sample.com/catalog/",
						},
						"browser": map[string]any{
							"name":    "Firefox",
							"other":   "",
							"version": "132.0",
						},
						"ip": ip,
						"os": map[string]any{
							"name":    "Linux",
							"version": "0.0",
						},
						"screen":    map[string]any{"width": 2816, "height": 1584, "density": 1.36},
						"userAgent": "Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",
						"sessionId": 1730384955277,
					},
					"originalTimestamp": time.Date(2024, 10, 31, 14, 39, 06, 50000000, time.UTC),
					"sentAt":            time.Date(2024, 10, 31, 14, 39, 12, 647000000, time.UTC),
					"integrations":      json.Value("{}"),
					"messageId":         "8071f50d-5a69-45f7-bb31-70e111aa8aed",
					"properties":        json.Value("{}"),
					"type":              "track",
					"event":             "click",
					"userId":            nil,
				},
			}, {
				event: events.Event{
					"anonymousId": "5d60ebba-cbf6-463c-8d55-fc7a6f66183f",
					"context": map[string]any{
						"library": library,
						"locale":  "it-IT",
						"page": map[string]any{
							"path":     "/catalog/",
							"referrer": "https://listing.sample.com/",
							"search":   "",
							"title":    "Test website",
							"url":      "https://sample.com/catalog/",
						},
						"browser": map[string]any{
							"name":    "Firefox",
							"other":   "",
							"version": "132.0",
						},
						"ip": ip,
						"os": map[string]any{
							"name":    "Linux",
							"version": "0.0",
						},
						"screen":    map[string]any{"width": 2816, "height": 1584, "density": 1.36},
						"userAgent": "Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",
						"sessionId": 1730384955277,
					},
					"originalTimestamp": time.Date(2024, 10, 31, 14, 39, 12, 319000000, time.UTC),
					"sentAt":            time.Date(2024, 10, 31, 14, 39, 12, 647000000, time.UTC),
					"integrations":      json.Value("{}"),
					"messageId":         "1935c955-45f8-44a3-b835-ced93138e8b3",
					"properties":        json.Value("{}"),
					"type":              "track",
					"event":             "click",
					"userId":            nil,
				},
			}},
		},

		// Errors reading events.
		{
			body: `{"batch":[` +
				`{"type":"page","event":null,"messageId":"f65c2f55-e30a-4458-83ca-0e5266e0f31d","userId":"bob"},` +
				`12,` +
				`{"type":"page","messageId":"ce93dc4b-72f1-43ac-8b82-bfe7eaaf6fe9","userId":"bob"}` +
				`]}`,
			expected: []expectedEvent{{
				err: errors.BadRequest("property 'event' is not a valid string"),
			}, {
				err: errors.BadRequest("expected an object for the event, but found number instead"),
			}, {
				event: events.Event{
					"messageId":  "ce93dc4b-72f1-43ac-8b82-bfe7eaaf6fe9",
					"context":    context,
					"type":       "page",
					"properties": json.Value("{}"),
					"userId":     "bob",
				},
			}},
		},
		{
			body: `[{}]`,
			expected: []expectedEvent{{
				err: errors.BadRequest("property 'type' is required for a batch request"),
			}},
		},
		{
			body: `{"batch":[{}]}`,
			expected: []expectedEvent{{
				err: errors.BadRequest("property 'type' is required for a batch request"),
			}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var requestURL *url.URL
			if test.typ == "" {
				requestURL, _ = url.Parse("/events")
			} else {
				requestURL, _ = url.Parse("/events/" + test.typ)
			}
			r := &http.Request{
				Method: "POST",
				Header: http.Header{
					"Content-Type": []string{"application/json; charset=utf-8"},
					"User-Agent":   []string{"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},
				},
				RemoteAddr: ip + ":7048",
				URL:        requestURL,
				Body:       io.NopCloser(strings.NewReader(test.body)),
			}
			maxReceivedAt := time.Now().UTC().Truncate(time.Millisecond)
			dec, err := newDecoder(r)
			if !reflect.DeepEqual(test.err, err) {
				t.Fatalf("expected error %#v, got error %#v", test.err, err)
			}
			if err != nil {
				if dec != nil {
					t.Fatal("unexpected non-nil decoder")
				}
				return
			}
			if dec == nil {
				t.Fatal("unexpected nil decoder")
			}
			if test.writeKey != "" {
				if writeKey := dec.WriteKey(); test.writeKey != writeKey {
					t.Fatalf("expected collect key %q, got %q", test.writeKey, writeKey)
				}
			}
			connectionType := test.connectionType
			if connectionType == 0 {
				test.connectionType = state.Website
			}
			i := 0
			for got, err := range dec.Events(test.connection, connectionType) {
				if i == len(test.expected) {
					if err != nil {
						t.Fatalf("when parsing an unexpected event, got error %q", err)
					}
					t.Fatalf("expected %d events, got more events", len(test.expected))
				}
				expected := test.expected[i]
				i++
				if !reflect.DeepEqual(expected.err, err) {
					t.Fatalf("expected events error %#v, got error %#v", expected.err, err)
				}
				if got == nil {
					if err == nil {
						t.Fatal("expected not nil event, got nil")
					}
					continue
				}
				// Test ReceivedAt.
				receivedAt, ok := got["receivedAt"].(time.Time)
				if !ok {
					if _, ok := got["receivedAt"]; !ok {
						t.Fatal("expected property 'receivedAt', got no property")
					}
					t.Fatalf("expected property 'receivedAt' of type time.Time, got with type %T", got["receivedAt"])
				}
				if receivedAt.Location() != time.UTC {
					t.Fatal("unexpected receiveAt location")
				}
				minReceivedAt := time.Now().UTC()
				if receivedAt.After(minReceivedAt) || receivedAt.Before(maxReceivedAt) {
					t.Fatalf("unexpected receiveAt %q", receivedAt.Format(time.RFC3339Nano))
				}
				expected.event["receivedAt"] = receivedAt
				// Test Properties.
				var properties = expected.event
				for _, name := range nonReadOptionalProperties {
					if _, ok := properties[name]; !ok {
						delete(got, name)
					}
				}
				var buf json.Buffer
				_ = buf.EncodeIndent(expected.event, "", "\t")
				expectedJSON := buf.String()
				buf.Reset()
				err = buf.EncodeIndent(got, "", "\t")
				if err != nil {
					t.Fatalf("unexpected error encoding the event: %s", err)
				}
				gotJSON := buf.String()
				if err != nil {
					t.Fatalf("unexpected error marshalling the event: %s", err)
				}
				if expectedJSON != gotJSON {
					t.Fatalf("unexpected event.\n\n- expected: %#v\n+ got:      %#v\n\n%s", expected.event, got, cmp.Diff(expectedJSON, gotJSON))
				}
			}
			if i < len(test.expected) {
				t.Fatalf("expected %d events, got %d", len(test.expected), i)
			}
		})
	}

}
