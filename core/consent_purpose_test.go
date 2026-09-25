// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestAddRequiredConsentProperties checks that each consent purpose path adds
// the expected profile property, if any.
func TestAddRequiredConsentProperties(t *testing.T) {

	email := types.Property{Name: "email", Type: types.String()}
	marketing := types.Property{
		Name: "marketing", Type: types.Boolean(), ReadOptional: true, Description: "Marketing consent",
	}
	consents := types.Property{
		Name: "consents", Type: types.JSON(), ReadOptional: true, Description: "Platform consents",
	}
	profileSchema := types.Object([]types.Property{
		email,
		marketing,
		consents,
		{Name: "text", Type: types.String()},
		{Name: "nested", ReadOptional: true, Type: types.Object([]types.Property{consents})},
	})
	tests := []struct {
		name          string
		property      string
		jsonKey       string
		wantAddedPath string
		wantSchema    types.Type
	}{
		{
			name: "boolean property", property: "marketing", wantAddedPath: "marketing",
			wantSchema: types.Object([]types.Property{email, marketing}),
		},
		{
			name: "JSON property with dotted key", property: "consents", jsonKey: "a.b", wantAddedPath: "consents",
			wantSchema: types.Object([]types.Property{email, consents}),
		},
		{
			name: "nested JSON property", property: "nested.consents", jsonKey: " purpose code ",
			wantAddedPath: "nested.consents",
			wantSchema: types.Object([]types.Property{
				email,
				{Name: "nested", ReadOptional: true, Type: types.Object([]types.Property{consents})},
			}),
		},
		{name: "dot access into JSON", property: "consents.marketing"},
		{name: "JSON property without key", property: "consents"},
		{name: "key access into boolean", property: "marketing", jsonKey: "key"},
		{name: "string property", property: "text"},
		{name: "missing property", property: "missing"},
		{name: "empty path"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			inputSchema := types.Object([]types.Property{email})
			purpose := state.NewConsentPurpose(state.ConsentPurpose{
				ID:                     "purpose",
				ProfileConsentLocation: &state.ProfileConsentLocation{Property: test.property, JSONKey: test.jsonKey},
			})
			schema, added, err := addRequiredConsentProperties(inputSchema, profileSchema, []*state.ConsentPurpose{purpose})
			if err != nil {
				t.Fatal(err)
			}

			wantSchema := inputSchema
			if test.wantAddedPath != "" {
				wantSchema = test.wantSchema
			}
			if !types.Equal(schema, wantSchema) {
				t.Fatalf("schema = %#v\nwant %#v", schema, wantSchema)
			}

			if test.wantAddedPath == "" {
				if len(added) != 0 {
					t.Fatalf("expected no consent property, got %v", added)
				}
				return
			}
			if len(added) != 1 || !added[test.wantAddedPath] {
				t.Fatalf("expected only %q to be added, got %v", test.wantAddedPath, added)
			}

		})
	}

}

// TestAddRequiredConsentPropertiesMultiplePurposes checks that missing
// properties are added once, existing pipeline properties are preserved, and
// incompatible purpose paths are skipped.
func TestAddRequiredConsentPropertiesMultiplePurposes(t *testing.T) {

	email := types.Property{Name: "email", Type: types.String()}
	pipelineMarketing := types.Property{
		Name: "marketing", Type: types.Boolean(), Description: "Pipeline marketing consent",
	}
	profileMarketing := types.Property{
		Name: "marketing", Type: types.Boolean(), ReadOptional: true, Description: "Profile marketing consent",
	}
	analytics := types.Property{Name: "analytics", Type: types.Boolean(), ReadOptional: true}
	consents := types.Property{Name: "consents", Type: types.JSON(), ReadOptional: true}
	inputSchema := types.Object([]types.Property{email, pipelineMarketing})
	profileSchema := types.Object([]types.Property{
		profileMarketing,
		analytics,
		consents,
		{Name: "text", Type: types.String()},
	})
	purposes := []*state.ConsentPurpose{
		state.NewConsentPurpose(state.ConsentPurpose{ID: "no-profile"}),
		state.NewConsentPurpose(state.ConsentPurpose{
			ID: "marketing", ProfileConsentLocation: &state.ProfileConsentLocation{Property: "marketing"},
		}),
		state.NewConsentPurpose(state.ConsentPurpose{
			ID: "text", ProfileConsentLocation: &state.ProfileConsentLocation{Property: "text"},
		}),
		state.NewConsentPurpose(state.ConsentPurpose{
			ID: "analytics", ProfileConsentLocation: &state.ProfileConsentLocation{Property: "analytics"},
		}),
		state.NewConsentPurpose(state.ConsentPurpose{
			ID:                     "platform-one",
			ProfileConsentLocation: &state.ProfileConsentLocation{Property: "consents", JSONKey: "measurement"},
		}),
		state.NewConsentPurpose(state.ConsentPurpose{
			ID:                     "platform-two",
			ProfileConsentLocation: &state.ProfileConsentLocation{Property: "consents", JSONKey: "personalization"},
		}),
	}
	schema, added, err := addRequiredConsentProperties(inputSchema, profileSchema, purposes)
	if err != nil {
		t.Fatal(err)
	}

	wantSchema := types.Object([]types.Property{email, pipelineMarketing, analytics, consents})
	if !types.Equal(schema, wantSchema) {
		t.Fatalf("schema = %#v\nwant %#v", schema, wantSchema)
	}
	if len(added) != 2 || !added["analytics"] || !added["consents"] {
		t.Fatalf("added = %v, want only analytics and consents", added)
	}

}

// TestAddRequiredConsentPropertiesNonObjectIntermediateProperty checks that an
// error is returned when an intermediate schema property is not an object.
func TestAddRequiredConsentPropertiesNonObjectIntermediateProperty(t *testing.T) {

	profileSchema := types.Object([]types.Property{
		{
			Name: "nested",
			Type: types.Object([]types.Property{
				{Name: "consents", Type: types.JSON()},
			}),
		},
	})
	inputSchema := types.Object([]types.Property{{Name: "nested", Type: types.String()}})
	purpose := state.NewConsentPurpose(state.ConsentPurpose{
		ID:                     "purpose",
		ProfileConsentLocation: &state.ProfileConsentLocation{Property: "nested.consents", JSONKey: "marketing"},
	})

	_, _, err := addRequiredConsentProperties(inputSchema, profileSchema, []*state.ConsentPurpose{purpose})
	if err != nil {
		return
	}

	t.Fatal("expected an error for a non-object intermediate property")

}

// TestConsentPurposeLocationsJSON checks optional fields and invalid locations
// in API requests.
func TestConsentPurposeLocationsJSON(t *testing.T) {

	tests := []struct {
		locations string
		wantError bool
	}{
		{``, false},
		{`,"eventConsentLocations":null,"profileConsentLocation":null`, false},
		{`,"eventConsentLocations":[]`, false},
		{`,"eventConsentLocations":[{"purposeCode":"marketing"},{"purposeCode":"Marketing"}]`, false},
		{`,"eventConsentLocations":[null]`, true},
		{`,"eventConsentLocations":[{}]`, true},
		{`,"eventConsentLocations":[{"purposeCode":null}]`, true},
		{`,"eventConsentLocations":[{"purposeCode":""}]`, true},
		{`,"eventConsentLocations":["marketing"]`, true},
		{`,"eventConsentLocations":{}`, true},
		{`,"profileConsentLocation":{"property":"marketing"}`, false},
		{`,"profileConsentLocation":{"property":"marketing","jsonKey":null}`, false},
		{`,"profileConsentLocation":{"property":"consents","jsonKey":"a.b"}`, false},
		{`,"profileConsentLocation":{"property":"missing","jsonKey":"key"}`, false},
		{`,"profileConsentLocation":{}`, true},
		{`,"profileConsentLocation":""`, true},
		{`,"profileConsentLocation":[]`, true},
		{`,"profileConsentLocation":{"property":null}`, true},
		{`,"profileConsentLocation":{"property":""}`, true},
		{`,"profileConsentLocation":{"property":"marketing","jsonKey":""}`, false},
		{`,"profileConsentLocation":{"property":"consents","jsonKey":1}`, true},
	}

	for _, test := range tests {
		t.Run(test.locations, func(t *testing.T) {

			var purpose ConsentPurposeToSet
			err := json.Unmarshal([]byte(`{"name":"Marketing"`+test.locations+`}`), &purpose)
			if err != nil {
				if !test.wantError {
					t.Fatalf("unexpected JSON decoding error: %s", err)
				}
				return
			}

			err = validateConsentPurposeToSet(purpose)
			if err != nil {
				if !test.wantError {
					t.Fatalf("expected valid consent locations, got %s", err)
				}
				return
			}
			if test.wantError {
				t.Fatal("expected invalid consent locations, got no error")
			}

		})
	}

}

// TestValidateConsentPurposeToSet checks consent location constraints.
func TestValidateConsentPurposeToSet(t *testing.T) {

	key := ` say "yes".\u0061 `
	invisibleKey := " \t\n\x01\u200b"
	nulKey := "a\x00"
	keyAtLimit := strings.Repeat("😀", 1024)
	keyOverLimit := strings.Repeat("😀", 1025)
	tests := []struct {
		name            string
		eventCodes      []string
		profileLocation *ProfileConsentLocation
		wantError       bool
	}{
		{name: "no locations"},
		{name: "maximum number of event locations", eventCodes: []string{"a", "b", "c", "d", "e"}},
		{name: "too many event locations", eventCodes: []string{"a", "b", "c", "d", "e", "f"}, wantError: true},
		{name: "hash-prefixed purpose code", eventCodes: []string{"#CFK567"}},
		{name: "dotted purpose code", eventCodes: []string{"vendor.marketing"}},
		{name: "duplicated purpose code", eventCodes: []string{"marketing", "marketing"}, wantError: true},
		{name: "empty purpose code", eventCodes: []string{""}, wantError: true},
		{name: "empty purpose code among valid codes", eventCodes: []string{"a", "", "b"}, wantError: true},
		{name: "whitespace-only purpose code", eventCodes: []string{" \t\n"}},
		{name: "control and format characters in purpose code", eventCodes: []string{invisibleKey}},
		{name: "NUL in purpose code", eventCodes: []string{nulKey}, wantError: true},
		{name: "purpose code at code point limit", eventCodes: []string{keyAtLimit}},
		{name: "purpose code over code point limit", eventCodes: []string{keyOverLimit}, wantError: true},
		{name: "Boolean profile location", profileLocation: &ProfileConsentLocation{Property: "consents.marketing"}},
		{name: "JSON profile location", profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: key}},
		{name: "empty profile property", profileLocation: &ProfileConsentLocation{}, wantError: true},
		{
			name: "bracketed profile property", wantError: true,
			profileLocation: &ProfileConsentLocation{Property: `consents["marketing"]`},
		},
		{
			name: "invalid dotted profile property", wantError: true,
			profileLocation: &ProfileConsentLocation{Property: "consents..marketing"},
		},
		{
			name:            "profile property at code point limit",
			profileLocation: &ProfileConsentLocation{Property: strings.Repeat("a", 1024)},
		},
		{
			name: "profile property over code point limit", wantError: true,
			profileLocation: &ProfileConsentLocation{Property: strings.Repeat("a", 1025)},
		},
		{
			name:            "empty JSON key means no key",
			profileLocation: &ProfileConsentLocation{Property: "marketing", JSONKey: ""},
		},
		{
			name:            "control and format characters in JSON key",
			profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: invisibleKey},
		},
		{
			name: "NUL in JSON key", wantError: true,
			profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: nulKey},
		},
		{
			name:            "independent property and JSON key limits",
			profileLocation: &ProfileConsentLocation{Property: strings.Repeat("a", 1024), JSONKey: keyAtLimit},
		},
		{
			name: "JSON key over code point limit", wantError: true,
			profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: keyOverLimit},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			purpose := ConsentPurposeToSet{Name: "Marketing", ProfileConsentLocation: test.profileLocation}
			for _, code := range test.eventCodes {
				purpose.EventConsentLocations = append(purpose.EventConsentLocations, EventConsentLocation{PurposeCode: code})
			}

			err := validateConsentPurposeToSet(purpose)
			if err != nil {
				if !test.wantError {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}
			if test.wantError {
				t.Fatal("expected an error, got nil")
			}

		})
	}

}
