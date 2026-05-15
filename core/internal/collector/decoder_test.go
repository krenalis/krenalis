// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/events"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"

	"github.com/google/go-cmp/cmp"
)

func Test_Decoder(t *testing.T) {

	writeKey := "vjJCb9lilU1GABTrSQ5qOkY7ddTW1uBQ"

	ip := "192.168.1.1"
	library := map[string]any{"name": "krenalis.js", "version": "0.0.0"}

	// These non-read optional properties are not tested if they are not present as expected.
	var nonReadOptionalProperties = []string{
		"connectionId", "anonymousId", "context", "messageId", "receivedAt", "sentAt", "originalTimestamp", "timestamp", "type",
	}

	type expectedEvent struct {
		// Expected decoded event.
		//   - ConnectionId: do not set; it will use the connection Id from the test.
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
		connectionId   int                 // Can be any value.
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
		{body: `{"batch":[],"connectionId":-2}`, err: errors.BadRequest("property 'connectionId' is not a valid connection identifier")},
		{body: `{"batch":[],"connectionId":264826420}`},

		{typ: "track", body: ``, err: errors.BadRequest("request's body is empty")},
		{typ: "track", body: `{`, expected: []expectedEvent{{err: errors.BadRequest("unexpected invalid token while decoding an event")}}},
		{typ: "track", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("either 'anonymousId' or 'userId' properties are required for a track event")}}},
		{typ: "page", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("either 'anonymousId' or 'userId' properties are required for a page event")}}},
		{typ: "identify", body: `{}`, expected: []expectedEvent{{err: errors.BadRequest("property 'userId' is required for an identify event")}}},

		// krenalis.track('click'); anonymous
		{
			body:         `{"batch":[{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}]}`,
			connectionId: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     map[string]any{"ip": ip},
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"traits":      json.Value("{}"),
					"type":        "track",
					"event":       "click",
				},
			}},
		},
		{
			body:         `[{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}]`,
			connectionId: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     map[string]any{"ip": ip},
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"traits":      json.Value("{}"),
					"type":        "track",
					"event":       "click",
				},
			}},
		},
		{
			typ:          "track",
			body:         `{"type":"track","event":"click","messageId":"90112b1f-1d2d-4566-a86f-27efae53530c","anonymousId":"d6e77158-a417-4571-9ec7-8ee0a7d169ad"}`,
			connectionId: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d6e77158-a417-4571-9ec7-8ee0a7d169ad",
					"context":     map[string]any{"ip": ip},
					"messageId":   "90112b1f-1d2d-4566-a86f-27efae53530c",
					"properties":  json.Value("{}"),
					"traits":      json.Value("{}"),
					"type":        "track",
					"event":       "click",
				},
			}},
		},

		// krenalis.identify('bob', {name: 'bob', age: 19})
		{
			body: `{"batch":[{"type":"identify","messageId":"9677e303-6a57-45e4-9c94-e47ec550a261","userId":"bob","groupId":null,"traits":{"name":"bob","age":19}}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":   map[string]any{"ip": ip},
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
					"context":   map[string]any{"ip": ip},
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
					"context":   map[string]any{"ip": ip},
					"messageId": "9677e303-6a57-45e4-9c94-e47ec550a261",
					"traits":    json.Value(`{"name":"bob","age":19}`),
					"type":      "identify",
					"userId":    "bob",
				}},
			},
		},

		// krenalis.track('page')
		{
			body: `{"batch":[{"type":"page","context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "ip": ip},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"traits":      json.Value(`{}`),
					"type":        "page",
				}},
			},
		},
		{
			body: `[{"type":"page","context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "ip": ip},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"traits":      json.Value(`{}`),
					"type":        "page",
				}},
			},
		},
		{
			typ:  "page",
			body: `{"context":{"page":{"path":"/boo","referrer":"https://example.com/","search":"id=5","title":"boo","url":"https://example.com/boo?id=5"}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":     map[string]any{"page": map[string]any{"path": "/boo", "referrer": "https://example.com/", "search": "id=5", "title": "boo", "url": "https://example.com/boo?id=5"}, "ip": ip},
					"anonymousId": "82281550-c0fc-4d69-bcf9-db1e43f9a76a",
					"properties":  json.Value(`{}`),
					"traits":      json.Value(`{}`),
					"type":        "page",
				}},
			},
		},

		// krenalis.screen('login', {}, {traits: {name: 'Bob'}})
		{
			body: `{"batch":[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635},"traits":{"name":"Bob"}},"name":"login","userId":"bob"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
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
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
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
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
					"properties": json.Value(`{}`),
					"name":       "login",
					"traits":     json.Value(`{"name":"Bob"}`),
					"type":       "screen",
					"userId":     "bob",
				}},
			},
		},

		// krenalis.screen('login')
		{
			body: `{"batch":[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","channel":"web","name":"login"}]}`,
			expected: []expectedEvent{{
				event: events.Event{
					"channel":    "web",
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
					"properties": json.Value(`{}`),
					"traits":     json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
				}},
			},
		},
		{
			body: `[{"type":"screen","context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}]`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
					"properties": json.Value(`{}`),
					"traits":     json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
				}},
			},
		},
		{
			typ:  "screen",
			body: `{"context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context":    map[string]any{"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")}, "ip": ip},
					"properties": json.Value(`{}`),
					"traits":     json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
				}},
			},
		},

		// krenalis.track('click'); krenalis.track('click');
		{
			// The 'integrations' field is included in the event's body even if
			// Krenalis ignores it, to test that when an SDK sends this field, no
			// errors are returned by the decoder.
			body: `{"batch":[` +
				`{"type":"track","event":"click","timestamp":"2024-10-31T14:39:06.050Z","properties":{},"userId":null,"messageId":"8071f50d-5a69-45f7-bb31-70e111aa8aed","anonymousId":"5d60ebba-cbf6-463c-8d55-fc7a6f66183f","context":{"browser":{"name":"Chrome","version":"138.0"},"library":{"name":"krenalis.js","version":"0.0.0"},"locale":"it-IT","page":{"path":"/catalog/","referrer":"https://listing.sample.com/","title":"Test website","url":"https://sample.com/catalog/"},"screen":{"width":2816,"height":1584,"density":1.3636363636363635},"userAgent":"Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0","sessionId":1730384955277,"sessionStart":true},"integrations":{}},` +
				`{"type":"track","event":"click","timestamp":"2024-10-31T14:39:12.319Z","properties":{},"userId":null,"messageId":"1935c955-45f8-44a3-b835-ced93138e8b3","anonymousId":"5d60ebba-cbf6-463c-8d55-fc7a6f66183f","context":{"os":{"name":"macOS","version":"15"},"library":{"name":"krenalis.js","version":"0.0.0"},"locale":"it-IT","page":{"path":"/catalog/","referrer":"https://listing.sample.com/","title":"Test website","url":"https://sample.com/catalog/"},"screen":{"width":2816,"height":1584,"density":1.3636363636363635},"userAgent":"Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0","sessionId":1730384955277,"sessionStart":false},"integrations":{}}` +
				`],"sentAt":"2024-10-31T14:39:12.647Z","writeKey":"qWqwaP3zGZOazQUmuFRuRMfW3lMCqjUa"}`,
			connectionId: 830163006,
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "5d60ebba-cbf6-463c-8d55-fc7a6f66183f",
					"context": map[string]any{
						"library": library,
						"locale":  "it-IT",
						"page": map[string]any{
							"path":     "/catalog/",
							"referrer": "https://listing.sample.com/",
							"title":    "Test website",
							"url":      "https://sample.com/catalog/",
						},
						"browser": map[string]any{
							"name":    "Chrome",
							"version": "138.0",
						},
						"ip": ip,
						"os": map[string]any{
							"name":    "Linux",
							"version": "132.0.0",
						},
						"screen":    map[string]any{"width": 2816, "height": 1584, "density": 1.36},
						"userAgent": "Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",
						"session": map[string]any{
							"id":    1730384955277,
							"start": true,
						},
					},
					"originalTimestamp": time.Date(2024, 10, 31, 14, 39, 06, 50000000, time.UTC),
					"sentAt":            time.Date(2024, 10, 31, 14, 39, 12, 647000000, time.UTC),
					"messageId":         "8071f50d-5a69-45f7-bb31-70e111aa8aed",
					"properties":        json.Value("{}"),
					"traits":            json.Value("{}"),
					"type":              "track",
					"event":             "click",
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
							"title":    "Test website",
							"url":      "https://sample.com/catalog/",
						},
						"browser": map[string]any{
							"name":    "Firefox",
							"version": "132.0.0",
						},
						"ip": ip,
						"os": map[string]any{
							"name":    "macOS",
							"version": "15",
						},
						"screen":    map[string]any{"width": 2816, "height": 1584, "density": 1.36},
						"userAgent": "Mozilla/5.0 (X11; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",
						"session": map[string]any{
							"id": 1730384955277,
						},
					},
					"originalTimestamp": time.Date(2024, 10, 31, 14, 39, 12, 319000000, time.UTC),
					"sentAt":            time.Date(2024, 10, 31, 14, 39, 12, 647000000, time.UTC),
					"messageId":         "1935c955-45f8-44a3-b835-ced93138e8b3",
					"properties":        json.Value("{}"),
					"traits":            json.Value("{}"),
					"type":              "track",
					"event":             "click",
				},
			}},
		},

		// Browser and OS.
		{
			typ:  "track",
			body: `{"context":{"userAgent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36"},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","event":"Product View"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context": map[string]any{
						"browser":   map[string]any{"name": "Chrome", "version": "117.0.0"},
						"ip":        ip,
						"os":        map[string]any{"name": "Windows", "version": "10.0.0"},
						"userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
					},
					"properties": json.Value(`{}`),
					"traits":     json.Value(`{}`),
					"type":       "track",
					"event":      "Product View",
				}},
			},
		},
		{
			body:     `{"batch":[{"type":"track","event":"Java Test","messageId":"249e67fd-2236-4e44-b79c-a82fa63ad214","timestamp":"2026-03-31T09:51:27.802Z","anonymousId":"d4c3aee6-00a9-4863-a2c8-dd0f18ef2145","userId":"andrea","properties":{"foo":"bar"}}],"sentAt":"2026-03-31T09:51:27.824Z","context":{"library":{"name":"analytics-java","version":"0.0.2"},"locale":"it-IT"},"writeKey":"8q8MWIB6iWKk2ffU2yyRrnVdoTYicULj"}`,
			writeKey: "8q8MWIB6iWKk2ffU2yyRrnVdoTYicULj",
			expected: []expectedEvent{{
				event: events.Event{
					"anonymousId": "d4c3aee6-00a9-4863-a2c8-dd0f18ef2145",
					"context": map[string]any{
						"ip": ip,
						"library": map[string]any{
							"name":    "analytics-java",
							"version": "0.0.2",
						},
						"locale": "it-IT",
					},
					"event":             "Java Test",
					"messageId":         "249e67fd-2236-4e44-b79c-a82fa63ad214",
					"originalTimestamp": time.Date(2026, 3, 31, 9, 51, 27, 802000000, time.UTC),
					"properties":        json.Value(`{"foo":"bar"}`),
					"sentAt":            time.Date(2026, 3, 31, 9, 51, 27, 824000000, time.UTC),
					"traits":            json.Value(`{}`),
					"type":              "track",
					"userId":            "andrea",
				},
			}},
		},
		// Location.
		{
			typ:  "screen",
			body: `{"context":{"screen":{"width":2600,"height":1550,"density":1.3636363636363635},"location":{"city":"London","country":"GB","latitude":51.5074,"longitude":-0.1278,"speed":25.562}},"anonymousId":"82281550-c0fc-4d69-bcf9-db1e43f9a76a","name":"login"}`,
			expected: []expectedEvent{{
				event: events.Event{
					"context": map[string]any{
						"screen": map[string]any{"width": 2600, "height": 1550, "density": decimal.MustParse("1.36")},
						"ip":     ip,
						"location": map[string]any{
							"city":      "London",
							"country":   "GB",
							"latitude":  51.5074,
							"longitude": -0.1278,
							"speed":     25.562,
						},
					},
					"properties": json.Value(`{}`),
					"traits":     json.Value(`{}`),
					"type":       "screen",
					"name":       "login",
				}},
			},
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
					"context":    map[string]any{"ip": ip},
					"type":       "page",
					"properties": json.Value("{}"),
					"traits":     json.Value("{}"),
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
			i := 0
			for got, err := range dec.Events(test.connectionId, true) {
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
				buf.Truncate(0)
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

// Test_mergeDefaultContext verifies the merge semantics for event-level and
// batch-level contexts, including precedence and cloning of nested sections.
func Test_mergeDefaultContext(t *testing.T) {
	t.Parallel()

	t.Run("clones nested default sections", func(t *testing.T) {
		t.Parallel()

		defaultContext := map[string]any{
			"browser": map[string]any{
				"name":    "chrome",
				"version": "138.0",
			},
			"library": map[string]any{
				"name":    "analytics-java",
				"version": "0.0.2",
			},
		}

		got := mergeDefaultContext(nil, defaultContext)
		gotBrowser, ok := got["browser"].(map[string]any)
		if !ok {
			t.Fatalf("expected merged browser section, got %T", got["browser"])
		}
		gotBrowser["name"] = "firefox"
		gotBrowser["extra"] = "value"

		defaultBrowser, ok := defaultContext["browser"].(map[string]any)
		if !ok {
			t.Fatalf("expected default browser section, got %T", defaultContext["browser"])
		}
		if defaultBrowser["name"] != "chrome" {
			t.Fatalf("expected default browser name to remain %q, got %v", "chrome", defaultBrowser["name"])
		}
		if _, ok := defaultBrowser["extra"]; ok {
			t.Fatalf("expected default browser section to remain untouched, got %v", defaultBrowser)
		}
	})

	t.Run("event context overrides default and receives missing nested keys", func(t *testing.T) {
		t.Parallel()

		eventContext := map[string]any{
			"browser": map[string]any{
				"name": "firefox",
			},
		}
		defaultContext := map[string]any{
			"browser": map[string]any{
				"name":    "chrome",
				"version": "138.0",
			},
			"library": map[string]any{
				"name":    "analytics-java",
				"version": "0.0.2",
			},
		}

		got := mergeDefaultContext(eventContext, defaultContext)
		expected := map[string]any{
			"browser": map[string]any{
				"name":    "firefox",
				"version": "138.0",
			},
			"library": map[string]any{
				"name":    "analytics-java",
				"version": "0.0.2",
			},
		}

		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatalf("unexpected merged context (-want +got):\n%s", diff)
		}
	})
}

func TestDecoderDefaultContextNotMutatedAcrossEvents(t *testing.T) {
	t.Parallel()

	body := `{"batch":[` +
		`{"type":"track","event":"first","anonymousId":"anon-1"},` +
		`{"type":"track","event":"second","anonymousId":"anon-2"}` +
		`],"context":{"browser":{"name":"custombrowser","version":"138.0"}}}`

	requestURL, err := url.Parse("/events")
	if err != nil {
		t.Fatalf("failed to parse request URL: %v", err)
	}

	r := &http.Request{
		Method: "POST",
		Header: http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
			"User-Agent":   []string{"DecoderDefaultContextNotMutatedAcrossEvents/1.0"},
		},
		RemoteAddr: "192.168.1.1:7048",
		URL:        requestURL,
		Body:       io.NopCloser(strings.NewReader(body)),
	}

	dec, err := newDecoder(r)
	if err != nil {
		t.Fatalf("newDecoder returned error: %v", err)
	}

	i := 0
	for event, err := range dec.Events(42, true) {
		if err != nil {
			t.Fatalf("unexpected event error: %v", err)
		}
		if event == nil {
			t.Fatal("expected non-nil event, got nil")
		}
		i++
		if i == 1 {
			browser, ok := dec.context["browser"].(map[string]any)
			if !ok {
				t.Fatalf("expected default browser section, got %T", dec.context["browser"])
			}
			expected := map[string]any{
				"name":    "custombrowser",
				"version": "138.0",
			}
			if diff := cmp.Diff(expected, browser); diff != "" {
				t.Fatalf("default context was mutated after first event (-want +got):\n%s", diff)
			}
		}
	}
	if i != 2 {
		t.Fatalf("expected 2 events, got %d", i)
	}
}

// TestDecoderContextIPHandling verifies the context.ip normalization and the
// fallback-to-request-ip switch.
func TestDecoderContextIPHandling(t *testing.T) {
	t.Parallel()

	const remoteIP = "198.51.100.23"
	const remoteIPv6 = "2001:db8:face:12::1"
	const remoteIPv6Prefix48 = "2001:db8:face::"
	const remoteIPv6Prefix32 = "2001:db8::"

	remoteParts := strings.Split(remoteIP, ".")
	if len(remoteParts) != 4 {
		t.Fatalf("expected 4 parts for remote IP %q, got %d parts", remoteIP, len(remoteParts))
	}
	remoteIP24 := remoteParts[0] + "." + remoteParts[1] + "." + remoteParts[2] + ".0"
	remoteIP16 := remoteParts[0] + "." + remoteParts[1] + ".0.0"

	requestURL, err := url.Parse("/events/track")
	if err != nil {
		t.Fatalf("failed to parse request URL: %v", err)
	}

	makeBody := func(contextJSON string) string {
		const base = `{"type":"track","event":"click","anonymousId":"anon-1"`
		if contextJSON == "" {
			return base + `}`
		}
		return base + `,"context":` + contextJSON + `}`
	}

	decode := func(t *testing.T, body string, fallback bool, remoteAddr string, forwardedFor string) (events.Event, error) {
		t.Helper()

		header := http.Header{
			"Content-Type": []string{"application/json; charset=utf-8"},
			"User-Agent":   []string{"DecoderContextIPTest/1.0"},
		}
		if forwardedFor != "" {
			header.Set("X-Forwarded-For", forwardedFor)
		}

		r := &http.Request{
			Method:     "POST",
			Header:     header,
			RemoteAddr: remoteAddr,
			URL:        requestURL,
			Body:       io.NopCloser(strings.NewReader(body)),
		}

		dec, err := newDecoder(r)
		if err != nil {
			t.Fatalf("newDecoder returned error: %v", err)
		}

		var (
			gotEvent events.Event
			gotErr   error
			count    int
		)

		for event, err := range dec.Events(42, fallback) {
			gotEvent = event
			gotErr = err
			count++
		}

		if count != 1 {
			t.Fatalf("expected 1 event, got %d", count)
		}
		if gotErr == nil && gotEvent == nil {
			t.Fatal("expected non-nil event, got nil")
		}

		return gotEvent, gotErr
	}

	type expectedIP struct {
		present bool
		value   string
	}

	tests := []struct {
		name         string
		contextJSON  string
		fallback     bool
		remoteIP     string
		forwardedFor string
		wantIP       expectedIP
		wantErr      error
	}{
		{
			name:        "no-context-ip-fallback-disabled",
			contextJSON: "",
			fallback:    false,
			wantIP:      expectedIP{present: false},
		},
		{
			name:        "no-context-ip-fallback-enabled",
			contextJSON: "",
			fallback:    true,
			wantIP:      expectedIP{present: true, value: remoteIP},
		},
		{
			name:        "context-without-ip-fallback-enabled",
			contextJSON: `{"locale":"en-US"}`,
			fallback:    true,
			wantIP:      expectedIP{present: true, value: remoteIP},
		},
		{
			name:        "context-regular-ip",
			contextJSON: `{"ip":"198.18.0.1"}`,
			fallback:    true,
			wantIP:      expectedIP{present: true, value: "198.18.0.1"},
		},
		{
			name:        "context-regular-ipv6",
			contextJSON: `{"ip":"2001:db8:face:12::1"}`,
			fallback:    true,
			wantIP:      expectedIP{present: true, value: "2001:db8:face:12::1"},
		},
		{
			name:        "context-ipv4-mapped-ipv6",
			contextJSON: `{"ip":"::ffff:192.0.2.1"}`,
			fallback:    true,
			wantIP:      expectedIP{present: true, value: "192.0.2.1"},
		},
		{
			name:        "context-scoped-ipv6",
			contextJSON: `{"ip":"fe80::1%eth0"}`,
			fallback:    true,
			wantErr:     errors.BadRequest("property 'ip' is not a valid IP address"),
		},
		{
			name:        "context-multicast-ipv6",
			contextJSON: `{"ip":"ff02::1"}`,
			fallback:    true,
			wantErr:     errors.BadRequest("property 'ip' cannot be a multicast IP address"),
		},
		{
			name:        "context-ip-zero",
			contextJSON: `{"ip":"0.0.0.0"}`,
			fallback:    true,
			wantIP:      expectedIP{present: false},
		},
		{
			name:        "context-ip-mask-32",
			contextJSON: `{"ip":"255.255.255.255"}`,
			fallback:    false,
			wantIP:      expectedIP{present: true, value: remoteIP},
		},
		{
			name:        "context-ip-mask-24",
			contextJSON: `{"ip":"255.255.255.0"}`,
			fallback:    false,
			wantIP:      expectedIP{present: true, value: remoteIP24},
		},
		{
			name:        "context-ip-mask-16",
			contextJSON: `{"ip":"255.255.0.0"}`,
			fallback:    false,
			wantIP:      expectedIP{present: true, value: remoteIP16},
		},
		{
			name:        "context-ipv4-mapped-ipv6-mask-24",
			contextJSON: `{"ip":"::ffff:255.255.255.0"}`,
			fallback:    false,
			wantIP:      expectedIP{present: true, value: "255.255.255.0"},
		},
		{
			name:        "context-ipv4-mapped-ipv6-multicast",
			contextJSON: `{"ip":"::ffff:224.0.0.1"}`,
			fallback:    false,
			wantErr:     errors.BadRequest("property 'ip' cannot be a multicast IP address"),
		},
		{
			name:        "no-context-ip-ipv6-fallback-enabled",
			contextJSON: "",
			fallback:    true,
			remoteIP:    remoteIPv6,
			wantIP:      expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:        "context-ip-mask-32-with-ipv6-remote",
			contextJSON: `{"ip":"255.255.255.255"}`,
			fallback:    false,
			remoteIP:    remoteIPv6,
			wantIP:      expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:        "context-ip-mask-24-with-ipv6-remote",
			contextJSON: `{"ip":"255.255.255.0"}`,
			fallback:    false,
			remoteIP:    remoteIPv6,
			wantIP:      expectedIP{present: true, value: remoteIPv6Prefix48},
		},
		{
			name:        "context-ip-mask-16-with-ipv6-remote",
			contextJSON: `{"ip":"255.255.0.0"}`,
			fallback:    false,
			remoteIP:    remoteIPv6,
			wantIP:      expectedIP{present: true, value: remoteIPv6Prefix32},
		},
		{
			name:         "x-forwarded-for-ipv6-overrides-remote-addr",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: remoteIPv6,
			wantIP:       expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:         "x-forwarded-for-ipv6-list",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: remoteIPv6 + ", " + remoteIP,
			wantIP:       expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:         "x-forwarded-for-ipv4",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: remoteIP,
			wantIP:       expectedIP{present: true, value: remoteIP},
		},
		{
			name:         "x-forwarded-for-bracketed-ipv6",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: "[" + remoteIPv6 + "]",
			wantIP:       expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:         "x-forwarded-for-bracketed-ipv6-with-port",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: "[" + remoteIPv6 + "]:1234",
			wantIP:       expectedIP{present: true, value: remoteIPv6},
		},
		{
			name:         "x-forwarded-for-ipv4-with-port",
			contextJSON:  "",
			fallback:     true,
			forwardedFor: remoteIP + ":1234",
			wantIP:       expectedIP{present: true, value: remoteIP},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			body := makeBody(tt.contextJSON)
			remoteAddr := remoteIP
			if tt.remoteIP != "" {
				remoteAddr = tt.remoteIP
			}
			event, err := decode(t, body, tt.fallback, net.JoinHostPort(remoteAddr, "9000"), tt.forwardedFor)
			if !reflect.DeepEqual(tt.wantErr, err) {
				t.Fatalf("expected error %#v, got error %#v", tt.wantErr, err)
			}
			if err != nil {
				return
			}

			ctx, ok := event["context"].(map[string]any)
			if ctx == nil {
				ctx = map[string]any{}
			}

			ipVal, ok := ctx["ip"]
			if tt.wantIP.present {
				if !ok {
					t.Fatalf("expected context.ip %q, got missing value", tt.wantIP.value)
				}
				gotIP, ok := ipVal.(string)
				if !ok {
					t.Fatalf("expected context.ip to be string, got %T", ipVal)
				}
				if gotIP != tt.wantIP.value {
					t.Fatalf("expected context.ip %q, got %q", tt.wantIP.value, gotIP)
				}
			} else {
				if ok {
					t.Fatalf("expected context.ip to be absent, got %v", ipVal)
				}
			}
		})
	}
}

// TestDecoderRemoteAddrValidation verifies request-level validation of IP
// addresses provided by RemoteAddr and X-Forwarded-For.
func TestDecoderRemoteAddrValidation(t *testing.T) {
	t.Parallel()

	requestURL, err := url.Parse("/events/track")
	if err != nil {
		t.Fatalf("failed to parse request URL: %v", err)
	}

	tests := []struct {
		name         string
		remoteAddr   string
		forwardedFor string
		wantErr      error
	}{
		{
			name:         "scoped-ipv6-x-forwarded-for",
			remoteAddr:   net.JoinHostPort("198.51.100.23", "9000"),
			forwardedFor: "fe80::1%eth0",
			wantErr:      errors.BadRequest("address specified in the 'X-Forwarded-For' header is not a valid IP address"),
		},
		{
			name:         "unclosed-bracket-ipv6-x-forwarded-for",
			remoteAddr:   net.JoinHostPort("198.51.100.23", "9000"),
			forwardedFor: "[2001:db8:face:12::1",
			wantErr:      errors.BadRequest("address specified in the 'X-Forwarded-For' header is not a valid IP address"),
		},
		{
			name:       "scoped-ipv6-remote-addr",
			remoteAddr: net.JoinHostPort("fe80::1%eth0", "9000"),
			wantErr:    errors.New("unexpected IP address from RemoteAddr"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{
				"Content-Type": []string{"application/json; charset=utf-8"},
				"User-Agent":   []string{"DecoderRemoteAddrValidationTest/1.0"},
			}
			if tt.forwardedFor != "" {
				header.Set("X-Forwarded-For", tt.forwardedFor)
			}

			r := &http.Request{
				Method:     "POST",
				Header:     header,
				RemoteAddr: tt.remoteAddr,
				URL:        requestURL,
				Body:       io.NopCloser(strings.NewReader(`{"type":"track","event":"click","anonymousId":"anon-1"}`)),
			}

			dec, err := newDecoder(r)
			if !reflect.DeepEqual(tt.wantErr, err) {
				t.Fatalf("expected error %#v, got error %#v", tt.wantErr, err)
			}
			if dec != nil {
				t.Fatal("unexpected non-nil decoder")
			}
		})
	}
}

func Test_parseUserAgent(t *testing.T) {

	tests := []struct {
		ua              string
		expectedBrowser map[string]any
		expectedOS      map[string]any
	}{
		{
			ua: "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1",
			expectedBrowser: map[string]any{
				"name":    "Firefox",
				"version": "15.0.1",
			},
			expectedOS: map[string]any{
				"name":    "Linux",
				"version": "15.0.1",
			},
		},
		{
			ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472 Safari/537.36",
			expectedBrowser: map[string]any{
				"name":    "Chrome",
				"version": "91.0.4472",
			},
			expectedOS: map[string]any{
				"name":    "Windows",
				"version": "10.0.0",
			},
		},
		{
			ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.1 Safari/605.1.15",
			expectedBrowser: map[string]any{
				"name":    "Safari",
				"version": "15.1.0",
			},
			expectedOS: map[string]any{
				"name":    "macOS",
				"version": "10.15.7",
			},
		},
		{
			ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1",
			expectedBrowser: map[string]any{
				"name":    "Safari",
				"version": "14.0.0",
			},
			expectedOS: map[string]any{
				"name":    "iOS",
				"version": "14.6.0",
			},
		},
		{
			ua: "Mozilla/5.0 (Linux; Android 11; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Mobile Safari/537.36",
			expectedBrowser: map[string]any{
				"name":    "Chrome",
				"version": "94.0.4606",
			},
			expectedOS: map[string]any{
				"name":    "Android",
				"version": "11.0.0",
			},
		},
		{
			ua: "Mozilla/5.0 (Linux; Android 1234123412341234123.0.864; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Mobile Safari/537.36",
			expectedBrowser: map[string]any{
				"name":    "Chrome",
				"version": "94.0.4606",
			},
			expectedOS: map[string]any{
				"name":    "Android",
				"version": "1234123412341234123.0.864",
			},
		},
		{
			ua: "Mozilla/5.0 (Linux; Android 12341234123412341231.0.864; Pixel 5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Mobile Safari/537.36",
			expectedBrowser: map[string]any{
				"name":    "Chrome",
				"version": "94.0.4606",
			},
			expectedOS: map[string]any{
				"name": "Android",
			},
		},
		{
			ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edg/91.0.864",
			expectedBrowser: map[string]any{
				"name":    "Edge",
				"version": "91.0.864",
			},
			expectedOS: map[string]any{
				"name":    "Windows",
				"version": "10.0.0",
			},
		},
		{
			ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edg/1234123412341234123.0.864",
			expectedBrowser: map[string]any{
				"name":    "Edge",
				"version": "1234123412341234123.0.864",
			},
			expectedOS: map[string]any{
				"name":    "Windows",
				"version": "10.0.0",
			},
		},
		{
			ua: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edg/123412341234123412341.0.864",
			expectedBrowser: map[string]any{
				"name": "Edge",
			},
			expectedOS: map[string]any{
				"name":    "Windows",
				"version": "10.0.0",
			},
		},
		{
			ua: "Mozilla/5.0 (X11; CrOS x86_64 16181.61.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.6998.198 Safari/537.36",
			expectedBrowser: map[string]any{
				"name":    "Chrome",
				"version": "134.0.6998",
			},
			expectedOS: map[string]any{
				"name":    "ChromeOS",
				"version": "134.0.6998",
			},
		},
		{
			ua: "SomeUnknownAgent/1.0",
			expectedBrowser: map[string]any{
				"name":  "Other",
				"other": "Unknown",
			},
			expectedOS: map[string]any{
				"name":  "Other",
				"other": "Unknown",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.ua, func(t *testing.T) {
			gotBrowser, gotOS := parseUserAgent(test.ua)
			if !reflect.DeepEqual(gotBrowser, test.expectedBrowser) {
				t.Fatalf("expected browser %#v, got %#v", test.expectedBrowser, gotBrowser)
			}
			if !reflect.DeepEqual(gotOS, test.expectedOS) {
				t.Fatalf("expected OS %#v, got %#v", test.expectedOS, gotOS)
			}
		})
	}

}

func Test_normalizeContextBrowser(t *testing.T) {
	tests := []struct {
		b        map[string]any
		expected map[string]any
	}{
		{
			b: map[string]any{
				"name": "Chrome",
			},
			expected: map[string]any{
				"name": "Chrome",
			},
		},
		{
			b: map[string]any{
				"name":    "chrome",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Chrome",
				"version": "123.456.789",
			},
		},
		{
			b: map[string]any{
				"name":    "samsung internet",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Samsung Internet",
				"version": "123.456.789",
			},
		},
		{
			b: map[string]any{
				"name": "My Strange Browser",
			},
			expected: map[string]any{
				"name":  "Other",
				"other": "My Strange Browser",
			},
		},
		{
			b: map[string]any{
				"name":    "My Strange Browser",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Other",
				"other":   "My Strange Browser",
				"version": "123.456.789",
			},
		},
		{
			b: map[string]any{
				"name":  "Chrome",
				"other": "X",
			},
			expected: map[string]any{
				"name": "Chrome",
			},
		},
		{
			b: map[string]any{
				"name":    "My Strange Browser",
				"version": "123.456.789",
				"other":   "x",
			},
			expected: map[string]any{
				"name":    "Other",
				"other":   "My Strange Browser",
				"version": "123.456.789",
			},
		},
		{
			b: map[string]any{
				"name": "CHROME FIREFOX",
			},
			expected: map[string]any{
				"name":  "Other",
				"other": "CHROME FIREFOX",
			},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			normalizeContextBrowser(test.b)
			if !reflect.DeepEqual(test.b, test.expected) {
				t.Fatalf("expected %#v, got %#v", test.expected, test.b)
			}
		})
	}
}

func Test_normalizeContextOS(t *testing.T) {
	tests := []struct {
		os       map[string]any
		expected map[string]any
	}{
		{
			os: map[string]any{
				"name": "Android",
			},
			expected: map[string]any{
				"name": "Android",
			},
		},
		{
			os: map[string]any{
				"name":    "android",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Android",
				"version": "123.456.789",
			},
		},
		{
			os: map[string]any{
				"name":    "chrome os",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Chrome OS",
				"version": "123.456.789",
			},
		},
		{
			os: map[string]any{
				"name": "My Strange OS",
			},
			expected: map[string]any{
				"name":  "Other",
				"other": "My Strange OS",
			},
		},
		{
			os: map[string]any{
				"name":    "My Strange OS",
				"version": "123.456.789",
			},
			expected: map[string]any{
				"name":    "Other",
				"other":   "My Strange OS",
				"version": "123.456.789",
			},
		},
		{
			os: map[string]any{
				"name":  "Linux",
				"other": "X",
			},
			expected: map[string]any{
				"name": "Linux",
			},
		},
		{
			os: map[string]any{
				"name":    "My Strange OS",
				"version": "123.456.789",
				"other":   "x",
			},
			expected: map[string]any{
				"name":    "Other",
				"other":   "My Strange OS",
				"version": "123.456.789",
			},
		},
		{
			os: map[string]any{
				"name": "LINUX BAD Android",
			},
			expected: map[string]any{
				"name":  "Other",
				"other": "LINUX BAD Android",
			},
		},
		{
			os: map[string]any{
				"name": "macos",
			},
			expected: map[string]any{
				"name": "macOS",
			},
		},
		{
			os: map[string]any{
				"name": "darwin",
			},
			expected: map[string]any{
				"name": "macOS",
			},
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			normalizeContextOS(test.os)
			if !reflect.DeepEqual(test.os, test.expected) {
				t.Fatalf("expected %#v, got %#v", test.expected, test.os)
			}
		})
	}
}

// TestParseRemoteAddr covers valid, invalid, and normalization cases for
// parseRemoteAddr.
func TestParseRemoteAddr(t *testing.T) {
	t.Parallel()

	// --- valid cases ---
	valid := []struct {
		in                     string
		wantIdentifiable       string
		wantPartiallyAnonymous string
		wantStronglyAnonymous  string
	}{
		// Common cases.
		{"192.168.1.42", "192.168.1.42", "192.168.1.0", "192.168.0.0"},
		{"10.0.0.1", "10.0.0.1", "10.0.0.0", "10.0.0.0"},
		{"172.16.5.123", "172.16.5.123", "172.16.5.0", "172.16.0.0"},
		{"8.8.8.8", "8.8.8.8", "8.8.8.0", "8.8.0.0"},
		{"::ffff:192.0.2.1", "192.0.2.1", "192.0.2.0", "192.0.0.0"},
		{"2001:db8:face:12::1", "2001:db8:face:12::1", "2001:db8:face::", "2001:db8::"},
		{"::1", "::1", "::", "::"},

		// Edge octet values.
		{"0.0.0.0", "0.0.0.0", "0.0.0.0", "0.0.0.0"},
		{"255.255.255.255", "255.255.255.255", "255.255.255.0", "255.255.0.0"},
		{"1.2.3.0", "1.2.3.0", "1.2.3.0", "1.2.0.0"},
		{"1.2.0.0", "1.2.0.0", "1.2.0.0", "1.2.0.0"},
	}

	for _, test := range valid {
		t.Run("valid/"+test.in, func(t *testing.T) {
			t.Parallel()

			var dec decoder
			err := dec.parseRemoteAddr(test.in)
			if err != nil {
				t.Fatalf("parseRemoteAddr(%q) returned error: %v", test.in, err)
			}

			ra := dec.remoteAddr
			if ra.identifiable != test.wantIdentifiable {
				t.Fatalf("identifiable: expected %q, got %q", test.wantIdentifiable, ra.identifiable)
			}
			if ra.partiallyAnonymous != test.wantPartiallyAnonymous {
				t.Fatalf("partiallyAnonymous: expected %q, got %q", test.wantPartiallyAnonymous, ra.partiallyAnonymous)
			}
			if ra.stronglyAnonymous != test.wantStronglyAnonymous {
				t.Fatalf("stronglyAnonymous: expected %q, got %q", test.wantStronglyAnonymous, ra.stronglyAnonymous)
			}

			wantIP := netip.MustParseAddr(test.wantIdentifiable)
			if wantIP != ra.ip {
				t.Fatalf("ip: expected %v, got %v", wantIP, ra.ip)
			}
		})
	}

	// --- invalid cases ---
	invalid := []string{
		"", "   ", "1.2.3", "1.2.3.4.5", "256.1.1.1", "-1.2.3.4", "1.2.3.-4",
		"abc.def.ghi.jkl", "fe80::1%eth0", "1.2.3.4 ", " 1.2.3.4",
	}

	for _, in := range invalid {
		t.Run("invalid/"+in, func(t *testing.T) {
			t.Parallel()

			var dec decoder
			err := dec.parseRemoteAddr(in)
			if err == nil {
				t.Fatalf("parseRemoteAddr(%q): expected error, got nil", in)
			}
			ra := dec.remoteAddr
			if ra.identifiable != "" || ra.partiallyAnonymous != "" || ra.stronglyAnonymous != "" || ra.ip.IsValid() {
				t.Fatalf("parseRemoteAddr(%q): expected zero-value remoteAddr on error, got %+v", in, ra)
			}
		})
	}

	// --- normalization case ---
	t.Run("normalization/leadingZeros", func(t *testing.T) {
		t.Parallel()

		var dec decoder
		err := dec.parseRemoteAddr("192.168.001.042")
		if err == nil {
			ra := dec.remoteAddr
			if ra.identifiable != "192.168.1.42" {
				t.Fatalf("normalization: expected %q, got %q", "192.168.1.42", ra.identifiable)
			}
			if ra.partiallyAnonymous != "192.168.1.0" || ra.stronglyAnonymous != "192.168.0.0" {
				t.Fatalf("masked normalization: expected partiallyAnonymous=%q stronglyAnonymous=%q, got partiallyAnonymous=%q stronglyAnonymous=%q",
					"192.168.1.0", "192.168.0.0", ra.partiallyAnonymous, ra.stronglyAnonymous)
			}
		}
	})
}
