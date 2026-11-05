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
	"github.com/krenalis/krenalis/tools/types"
)

// TestDecoderEventCatalog checks schema lookup after common event decoding.
func TestDecoderEventCatalog(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       string
		traits     string
		properties string
		errorPath  string
	}{
		{
			name:       "known track with late discriminators",
			body:       `{"properties":{"extra":true,"details":{"extra":null,"source":"sdk"},"occurredAt":"2026-11-04T12:00:00+01:00"},"context":{"traits":{"country":"IT","email":"user@example.org"}},"event":"Example Action","type":"track","userId":"u"}`,
			traits:     `{"email":"user@example.org","country":"IT"}`,
			properties: `{"occurredAt":"2026-11-04T11:00:00Z","details":{"source":"sdk","extra":null},"extra":true}`,
		},
		{
			name:   "known track with optional data absent",
			body:   `{"type":"track","event":"Example Action","userId":"u"}`,
			traits: `{}`, properties: `{}`,
		},
		{
			name:      "invalid property",
			body:      `{"type":"track","event":"Example Action","userId":"u","properties":{"quantity":"1"}}`,
			errorPath: "properties.quantity",
		},
		{
			name:      "invalid traits",
			body:      `{"type":"track","event":"Example Action","userId":"u","context":{"traits":{"country":"invalid"}}}`,
			errorPath: "traits.country",
		},
		{
			name:      "nested validation error",
			body:      `{"type":"track","event":"Example Action","userId":"u","properties":{"details":{"source":1}}}`,
			errorPath: "properties.details.source",
		},
		{
			name:   "unknown event",
			body:   `{"type":"track","event":"Unknown","userId":"u","properties":{"quantity": "1"}}`,
			traits: `{}`, properties: `{"quantity": "1"}`,
		},
		{
			name:   "event name case",
			body:   `{"type":"track","event":"example action","userId":"u","properties":{"quantity":"1"}}`,
			traits: `{}`, properties: `{"quantity":"1"}`,
		},
		{
			name:   "catalog page without schemas",
			body:   `{"type":"page","name":"Example Page","userId":"u","properties":{"quantity":"1"}}`,
			traits: `{}`, properties: `{"quantity":"1"}`,
		},
		{
			name:   "same name with different type",
			body:   `{"type":"page","name":"Example Action","userId":"u","properties":{"quantity":"1"}}`,
			traits: `{}`, properties: `{"quantity":"1"}`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(test.body))
			r.Header.Set("Content-Type", "application/json")
			dec, err := newDecoder(r)
			if err != nil {
				t.Fatalf("expected an event decoder, got %v", err)
			}
			count := 0
			for event, err := range dec.Events(decoderTestConnectionID, lookupTestEventDefinition, false) {
				count++
				if err != nil {
					validation, ok := errors.AsType[ValidationError](err)
					if !ok || test.errorPath == "" {
						t.Fatalf("expected path %q, got %T: %v", test.errorPath, err, err)
					}
					if validation.PropertyPath() != test.errorPath {
						t.Fatalf("expected path %q, got %q", test.errorPath, validation.PropertyPath())
					}
					if event != nil {
						t.Fatalf("expected no invalid event, got %#v", event)
					}
					continue
				}
				if test.errorPath != "" {
					t.Fatalf("expected validation error for %s, got nil", test.errorPath)
				}
				for name, want := range map[string]string{"traits": test.traits, "properties": test.properties} {
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

// TestDecoderEventCatalogBatch keeps validation failures local to their events
// and preserves shared context when normalizing another event's traits.
func TestDecoderEventCatalogBatch(t *testing.T) {

	body := `{"context":{"traits":{"extra":null,"country":"IT"}},"batch":[` +
		`{"type":"track","event":"Example Action","userId":"bad","properties":{"quantity":"1"}},` +
		`{"type":"track","event":"Example Action","userId":"valid","properties":{"quantity":2}},` +
		`{"type":"track","event":"Unknown","userId":"other"}]}`
	r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	dec, err := newDecoder(r)
	if err != nil {
		t.Fatalf("expected an event decoder, got %v", err)
	}
	index := 0
	for event, err := range dec.Events(decoderTestConnectionID, lookupTestEventDefinition, false) {
		index++
		if err != nil {
			validation, ok := errors.AsType[ValidationError](err)
			if !ok || index != 1 || validation.PropertyPath() != "properties.quantity" {
				t.Fatalf("expected first-event properties.quantity error, got event %d: %v", index, err)
			}
			continue
		}
		if index == 1 {
			t.Fatal("expected validation error for first event, got nil")
		}
		want := `{"country":"IT","extra":null}`
		if index == 3 {
			want = `{"extra":null,"country":"IT"}`
		}
		if got := string(event["traits"].(json.Value)); got != want {
			t.Fatalf("expected traits %s, got %s", want, got)
		}
	}
	if index != 3 {
		t.Fatalf("expected three event results, got %d", index)
	}

}

// TestDecoderEventCatalogRequiresLookup rejects a missing schema lookup.
func TestDecoderEventCatalogRequiresLookup(t *testing.T) {

	r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(`{"type":"page","userId":"u"}`))
	r.Header.Set("Content-Type", "application/json")
	dec, err := newDecoder(r)
	if err != nil {
		t.Fatalf("expected an event decoder, got %v", err)
	}
	count := 0
	for event, err := range dec.Events(decoderTestConnectionID, nil, false) {
		count++
		if err != nil {
			if _, ok := errors.AsType[ValidationError](err); ok {
				t.Fatalf("expected a technical error, got %T", err)
			}
			if _, ok := errors.AsType[*errors.BadRequestError](err); ok {
				t.Fatalf("expected a technical error, got %T", err)
			}
			if event != nil {
				t.Fatalf("expected no event, got %#v", event)
			}
			continue
		}
		t.Fatal("expected a missing schema lookup error, got nil")
	}
	if count != 1 {
		t.Fatalf("expected one error, got %d", count)
	}

}

// TestDecoderEventSchemaLookup checks the schemas supplied by the lookup.
func TestDecoderEventSchemaLookup(t *testing.T) {
	for _, test := range []struct {
		typ  string
		body string
	}{
		{"track", `{"type":"track","event":"Custom","userId":"u","context":{"traits":{"joinedAt":"2026-11-04T12:00:00+01:00"}},"properties":{"purchasedAt":"2026-11-04T14:00:00+01:00"}}`},
		{"page", `{"type":"page","name":"Custom","userId":"u","context":{"traits":{"joinedAt":"2026-11-04T12:00:00+01:00"}},"properties":{"purchasedAt":"2026-11-04T14:00:00+01:00"}}`},
	} {
		t.Run(test.typ, func(t *testing.T) {

			r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(test.body))
			r.Header.Set("Content-Type", "application/json")
			dec, err := newDecoder(r)
			if err != nil {
				t.Fatalf("expected an event decoder, got %v", err)
			}
			calls := 0
			lookup := func(typ, eventOrName string) state.EventDefinition {
				calls++
				if typ != test.typ || eventOrName != "Custom" {
					t.Fatalf("expected lookup (%q, Custom), got (%q, %q)", test.typ, typ, eventOrName)
				}
				traits := types.Object([]types.Property{{Name: "joinedAt", Type: types.DateTime()}})
				properties := types.Object([]types.Property{{Name: "purchasedAt", Type: types.DateTime()}})
				return state.EventDefinition{Schemas: state.EventSchemas{Traits: traits, Properties: properties}}
			}
			for event, err := range dec.Events(decoderTestConnectionID, lookup, false) {
				if err != nil {
					t.Fatalf("expected normalized event, got %v", err)
				}
				for name, want := range map[string]string{
					"traits":     `{"joinedAt":"2026-11-04T11:00:00Z"}`,
					"properties": `{"purchasedAt":"2026-11-04T13:00:00Z"}`,
				} {
					if got := string(event[name].(json.Value)); got != want {
						t.Fatalf("expected %s=%s, got %s", name, want, got)
					}
				}
			}
			if calls != 1 {
				t.Fatalf("expected one lookup, got %d", calls)
			}

		})
	}
}

// TestDecoderEventSchemaInvariant rejects broken internal event fields.
func TestDecoderEventSchemaInvariant(t *testing.T) {
	for _, test := range []struct {
		name       string
		body       string
		badTraits  bool
		properties bool
		invalid    bool
	}{
		{"missing properties", `{"type":"identify","userId":"u"}`, false, true, true},
		{"wrong traits type", `{"type":"track","event":"Custom","userId":"u"}`, true, false, true},
		{"no properties schema", `{"type":"identify","userId":"u"}`, false, false, false},
	} {
		t.Run(test.name, func(t *testing.T) {

			r := httptest.NewRequest(http.MethodPost, "/events", strings.NewReader(test.body))
			r.Header.Set("Content-Type", "application/json")
			dec, err := newDecoder(r)
			if err != nil {
				t.Fatalf("expected an event decoder, got %v", err)
			}
			if test.badTraits {
				dec.context = map[string]any{"traits": 1}
			}
			lookup := func(typ, eventOrName string) state.EventDefinition {
				schema := types.Object([]types.Property{{Name: "name", Type: types.String(), ReadOptional: true}})
				if test.properties {
					return state.EventDefinition{Schemas: state.EventSchemas{Properties: schema}}
				}
				return state.EventDefinition{Schemas: state.EventSchemas{Traits: schema}}
			}
			count := 0
			for event, err := range dec.Events(decoderTestConnectionID, lookup, false) {
				count++
				if err != nil {
					if !test.invalid {
						t.Fatalf("expected a valid event, got %v", err)
					}
					if _, ok := errors.AsType[ValidationError](err); ok {
						t.Fatalf("expected an internal error, got %T", err)
					}
					if _, ok := errors.AsType[*errors.BadRequestError](err); ok {
						t.Fatalf("expected an internal error, got %T", err)
					}
					if !strings.HasPrefix(err.Error(), "core/internal/collector: ") {
						t.Fatalf("expected a package-prefixed error, got %v", err)
					}
					if event != nil {
						t.Fatalf("expected no event, got %#v", event)
					}
					continue
				}
				if test.invalid {
					t.Fatal("expected an internal error, got nil")
				}
				if _, ok := event["properties"]; ok {
					t.Fatalf("expected absent properties, got %#v", event["properties"])
				}
			}
			if count != 1 {
				t.Fatalf("expected one result, got %d", count)
			}

		})
	}
}

func lookupTestEventDefinition(typ, eventOrName string) state.EventDefinition {
	var workspace state.Workspace
	return workspace.LookupEventDefinition(typ, eventOrName)
}
