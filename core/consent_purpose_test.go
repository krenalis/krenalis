// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

// TestConsentPurposeLocationsJSON checks optional and invalid consent locations
// in API request JSON.
func TestConsentPurposeLocationsJSON(t *testing.T) {

	tests := []struct {
		name                string
		locations           string
		wantDecodeError     bool
		wantValidationError bool
	}{
		{name: "locations omitted"},
		{name: "null event locations", locations: `,"eventConsentLocations":null`},
		{name: "null profile location", locations: `,"profileConsentLocation":null`},
		{name: "empty event locations", locations: `,"eventConsentLocations":[]`},
		{name: "case-sensitive purpose codes", locations: `,"eventConsentLocations":[{"purposeCode":"marketing"},{"purposeCode":"Marketing"}]`},
		{name: "null event location", locations: `,"eventConsentLocations":[null]`, wantValidationError: true},
		{name: "empty event location", locations: `,"eventConsentLocations":[{}]`, wantValidationError: true},
		{name: "null purpose code", locations: `,"eventConsentLocations":[{"purposeCode":null}]`, wantValidationError: true},
		{name: "empty purpose code", locations: `,"eventConsentLocations":[{"purposeCode":""}]`, wantValidationError: true},
		{name: "string event location", locations: `,"eventConsentLocations":["marketing"]`, wantDecodeError: true},
		{name: "object event locations", locations: `,"eventConsentLocations":{}`, wantDecodeError: true},
		{name: "boolean profile location", locations: `,"profileConsentLocation":{"property":"marketing"}`},
		{name: "null profile JSON key", locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":null}`},
		{name: "dotted profile JSON key", locations: `,"profileConsentLocation":{"property":"consents","jsonKey":"a.b"}`},
		{name: "arbitrary profile property", locations: `,"profileConsentLocation":{"property":"missing","jsonKey":"key"}`},
		{name: "empty profile location", locations: `,"profileConsentLocation":{}`, wantValidationError: true},
		{name: "string profile location", locations: `,"profileConsentLocation":""`, wantDecodeError: true},
		{name: "array profile location", locations: `,"profileConsentLocation":[]`, wantDecodeError: true},
		{name: "null profile property", locations: `,"profileConsentLocation":{"property":null}`, wantValidationError: true},
		{name: "empty profile property", locations: `,"profileConsentLocation":{"property":""}`, wantValidationError: true},
		{name: "empty profile JSON key", locations: `,"profileConsentLocation":{"property":"marketing","jsonKey":""}`},
		{name: "numeric profile JSON key", locations: `,"profileConsentLocation":{"property":"consents","jsonKey":1}`, wantDecodeError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.wantDecodeError && test.wantValidationError {
				t.Fatal("expected only one error stage, got both decoding and validation")
			}

			var purpose ConsentPurposeToSet
			err := json.Unmarshal([]byte(`{"name":"Marketing"`+test.locations+`}`), &purpose)
			if err != nil {
				if !test.wantDecodeError {
					t.Fatalf("expected no JSON decoding error, got %v", err)
				}
				return
			}
			if test.wantDecodeError {
				t.Fatal("expected a JSON decoding error, got nil")
			}

			err = validateConsentPurposeToSet(purpose)
			if err != nil {
				if !test.wantValidationError {
					t.Fatalf("expected no validation error, got %v", err)
				}
				return
			}
			if test.wantValidationError {
				t.Fatal("expected a validation error, got nil")
			}

		})
	}

}

// TestValidateConsentPurposeToSet checks consent location constraints.
func TestValidateConsentPurposeToSet(t *testing.T) {

	keyWithLiteralEscape := ` say "yes".\u0061 `
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
		{name: "purpose codes are case-sensitive", eventCodes: []string{"marketing", "Marketing"}},
		{name: "duplicated purpose code", eventCodes: []string{"marketing", "marketing"}, wantError: true},
		{
			name: "non-adjacent duplicated purpose code", eventCodes: []string{"marketing", "analytics", "marketing"},
			wantError: true,
		},
		{name: "empty purpose code", eventCodes: []string{""}, wantError: true},
		{name: "empty purpose code among valid codes", eventCodes: []string{"a", "", "b"}, wantError: true},
		{name: "whitespace-only purpose code", eventCodes: []string{" \t\n"}},
		{name: "control and format characters in purpose code", eventCodes: []string{invisibleKey}},
		{name: "NUL in purpose code", eventCodes: []string{nulKey}, wantError: true},
		{name: "purpose code at code point limit", eventCodes: []string{keyAtLimit}},
		{name: "purpose code over code point limit", eventCodes: []string{keyOverLimit}, wantError: true},
		{name: "boolean profile location", profileLocation: &ProfileConsentLocation{Property: "consents.marketing"}},
		{
			name:            "JSON profile location",
			profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: keyWithLiteralEscape},
		},
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
			name:            "JSON key at code point limit",
			profileLocation: &ProfileConsentLocation{Property: "consents", JSONKey: keyAtLimit},
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
					t.Fatalf("expected no validation error, got %v", err)
				}
				return
			}
			if test.wantError {
				t.Fatal("expected a validation error, got nil")
			}

		})
	}

}
