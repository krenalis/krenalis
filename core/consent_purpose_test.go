// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
)

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
