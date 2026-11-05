// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// TestDecoderWorkspaceEventCatalog exercises the production catalog through
// the normal event decoder, retaining JSON as the pipeline representation.
func TestDecoderWorkspaceEventCatalog(t *testing.T) {
	large := strings.Repeat("9", 90)
	for _, test := range []struct {
		name       string
		body       string
		properties string
		traits     string
		errorPath  string
		badRequest bool
	}{
		{
			name: "order with optional data absent", body: `{"type":"track","event":"Order Completed","userId":"u"}`,
			properties: `{}`, traits: `{}`,
		},
		{
			name:       "order products and extra data",
			body:       `{"type":"track","event":"Order Completed","userId":"u","properties":{"total":30.125,"products":[{"quantity":2.5,"custom":null,"price":12.5,"product_id":"p"}],"currency":"EUR","extra":` + large + `}}`,
			properties: `{"currency":"EUR","products":[{"price":12.5,"product_id":"p","quantity":2.5,"custom":null}],"total":30.125,"extra":` + large + `}`, traits: `{}`,
		},
		{
			name:       "exact large monetary amount",
			body:       `{"type":"track","event":"Order Completed","userId":"u","properties":{"total":12345678901234.1234}}`,
			properties: `{"total":12345678901234.1234}`, traits: `{}`,
		},
		{
			name:       "removal does not change quantity sign",
			body:       `{"type":"track","event":"Product Removed","userId":"u","properties":{"quantity":-2.5}}`,
			properties: `{"quantity":-2.5}`, traits: `{}`,
		},
		{
			name:      "monetary overflow",
			body:      `{"type":"track","event":"Order Completed","userId":"u","properties":{"total":100000000000000}}`,
			errorPath: "properties.total",
		},
		{
			name:      "nullable does not apply to every field",
			body:      `{"type":"track","event":"Order Completed","userId":"u","properties":{"total":null}}`,
			errorPath: "properties.total",
		},
		{
			name:      "product element error",
			body:      `{"type":"track","event":"Order Completed","userId":"u","properties":{"products":[{"quantity":1},{"quantity":"2.5"}]}}`,
			errorPath: "properties.products[1].quantity",
		},
		{
			name:       "textual rating",
			body:       `{"type":"track","event":"Product Reviewed","userId":"u","properties":{"rating":"excellent"}}`,
			properties: `{"rating":"excellent"}`, traits: `{}`,
		},
		{
			name:      "numeric rating conversion remains deferred",
			body:      `{"type":"track","event":"Product Reviewed","userId":"u","properties":{"rating":3}}`,
			errorPath: "properties.rating",
		},
		{
			name:       "structured product search",
			body:       `{"type":"track","event":"Products Searched","userId":"u","properties":{"query":{"custom.key":["red",1,null]}}}`,
			properties: `{"query":{"custom.key":["red",1,null]}}`, traits: `{}`,
		},
		{
			name:       "textual product search",
			body:       `{"type":"track","event":"Products Searched","userId":"u","properties":{"query":"red shoes"}}`,
			properties: `{"query":"red shoes"}`, traits: `{}`,
		},
		{
			name:       "video live null",
			body:       `{"type":"track","event":"Video Playback Started","userId":"u","properties":{"total_length":null}}`,
			properties: `{"total_length":null}`, traits: `{}`,
		},
		{
			name:       "video absent duration",
			body:       `{"type":"track","event":"Video Playback Started","userId":"u"}`,
			properties: `{}`, traits: `{}`,
		},
		{
			name:       "video zero duration",
			body:       `{"type":"track","event":"Video Playback Started","userId":"u","properties":{"total_length":0}}`,
			properties: `{"total_length":0}`, traits: `{}`,
		},
		{
			name:       "video ad category array",
			body:       `{"type":"track","event":"Video Playback Started","userId":"u","properties":{"ad_type":["mid-roll","post-roll"]}}`,
			properties: `{"ad_type":["mid-roll","post-roll"]}`, traits: `{}`,
		},
		{
			name:       "video ad category scalar",
			body:       `{"type":"track","event":"Video Playback Started","userId":"u","properties":{"ad_type":"mid-roll"}}`,
			properties: `{"ad_type":"mid-roll"}`, traits: `{}`,
		},
		{
			name:       "fractional conversation duration",
			body:       `{"type":"track","event":"Live Chat Conversation Ended","userId":"u","properties":{"conversation_duration":1.250}}`,
			properties: `{"conversation_duration":1.25}`, traits: `{}`,
		},
		{
			name:      "nonfinite duration",
			body:      `{"type":"track","event":"Live Chat Conversation Ended","userId":"u","properties":{"conversation_duration":"NaN"}}`,
			errorPath: "properties.conversation_duration",
		},
		{
			name:      "duration conversion remains deferred",
			body:      `{"type":"track","event":"Live Chat Conversation Ended","userId":"u","properties":{"conversation_duration":"1.25"}}`,
			errorPath: "properties.conversation_duration",
		},
		{
			name:   "identify traits",
			body:   `{"type":"identify","userId":"u","traits":{"createdAt":"2026-11-04T12:00:00+01:00","birthday":"2000-01-02","extra":null}}`,
			traits: `{"birthday":"2000-01-02","createdAt":"2026-11-04T11:00:00Z","extra":null}`,
		},
		{
			name:   "group traits retain arbitrary extra keys",
			body:   `{"type":"group","userId":"u","groupId":"g","traits":{"employees":42,"total billed":100.25,"name":"Example"}}`,
			traits: `{"name":"Example","employees":42,"total billed":100.25}`,
		},
		{
			name:       "named page uses generic definition",
			body:       `{"type":"page","userId":"u","name":"Home","properties":{"title":"Welcome","extra":null}}`,
			properties: `{"title":"Welcome","extra":null}`, traits: `{}`,
		},
		{
			name:       "named screen preserves keys outside native grammar",
			body:       `{"type":"screen","userId":"u","name":"Home","properties":{"Feed Type":"news"}}`,
			properties: `{"Feed Type":"news"}`, traits: `{}`,
		},
		{
			name:       "proposed types remain unconstrained",
			body:       `{"type":"track","event":"Order Created","userId":"u","properties":{"total":"custom representation"}}`,
			properties: `{"total":"custom representation"}`, traits: `{}`,
		},
		{
			name:       "unknown track unchanged",
			body:       `{"type":"track","event":"Custom Event","userId":"u","properties":{"arbitrary.key":null}}`,
			properties: `{"arbitrary.key":null}`, traits: `{}`,
		},
		{
			name: "legacy catalog entry does not enable ingestion",
			body: `{"type":"merge","userId":"u"}`, badRequest: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(test.body))
			r.Header.Set("Content-Type", "application/json")
			dec, err := newDecoder(r)
			if err != nil {
				t.Fatalf("expected an event decoder, got %v", err)
			}

			var workspace state.Workspace
			count := 0
			for event, err := range dec.Events(decoderTestConnectionID, workspace.LookupEventDefinition, false) {
				count++
				if err != nil {
					if test.badRequest {
						if _, ok := errors.AsType[*errors.BadRequestError](err); !ok {
							t.Fatalf("expected a bad request, got %T: %v", err, err)
						}
						continue
					}
					validation, ok := errors.AsType[ValidationError](err)
					if !ok || test.errorPath == "" || validation.PropertyPath() != test.errorPath {
						t.Fatalf("expected validation error at %q, got %T: %v", test.errorPath, err, err)
					}
					if event != nil {
						t.Fatalf("expected no invalid event, got %#v", event)
					}
					continue
				}
				if test.badRequest || test.errorPath != "" {
					t.Fatal("expected invalid event data, got a decoded event")
				}
				for name, want := range map[string]string{"traits": test.traits, "properties": test.properties} {
					if want == "" {
						if _, ok := event[name]; ok {
							t.Fatalf("expected no %s property, got %#v", name, event[name])
						}
						continue
					}
					value, ok := event[name].(json.Value)
					if !ok || string(value) != want {
						t.Fatalf("expected %s=%s as json.Value, got %#v", name, want, event[name])
					}
				}
			}
			if count != 1 {
				t.Fatalf("expected one event result, got %d", count)
			}

		})
	}
}
