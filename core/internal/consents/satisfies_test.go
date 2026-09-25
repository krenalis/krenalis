// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/json"
)

var givenConsentsCases = []struct {
	name     string
	required []string
	matchAll bool
	given    map[string]any
	want     bool
}{
	{
		name:     "no required purposes",
		required: nil,
		matchAll: true,
		given:    map[string]any{},
		want:     true,
	},
	{
		name:     "AND: all required consents are true",
		required: []string{"marketing", "analytics"},
		matchAll: true,
		given: map[string]any{
			"marketing": true,
			"analytics": true,
			"other":     false,
		},
		want: true,
	},
	{
		name:     "AND: one required consent is false",
		required: []string{"marketing", "analytics"},
		matchAll: true,
		given: map[string]any{
			"marketing": true,
			"analytics": false,
		},
		want: false,
	},
	{
		name:     "AND: one required consent is missing",
		required: []string{"marketing", "analytics"},
		matchAll: true,
		given: map[string]any{
			"marketing": true,
		},
		want: false,
	},
	{
		name:     "AND: required consent is not a bool",
		required: []string{"marketing"},
		matchAll: true,
		given: map[string]any{
			"marketing": "true",
		},
		want: false,
	},
	{
		name:     "AND: no consent is given",
		required: []string{"marketing"},
		matchAll: true,
		given:    map[string]any{},
		want:     false,
	},
	{
		name:     "OR: all required consents are true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"marketing": true,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: one required consent is true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"marketing": false,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: one required consent is missing and the other is true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: every required consent is missing",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"other": true,
		},
		want: false,
	},
	{
		name:     "OR: no required consent is true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"marketing": false,
			"analytics": false,
		},
		want: false,
	},
	{
		name:     "OR: no consent is given",
		required: []string{"marketing"},
		matchAll: false,
		given:    map[string]any{},
		want:     false,
	},
}

func TestSatisfiesEvent(t *testing.T) {
	for _, c := range givenConsentsCases {
		t.Run(c.name, func(t *testing.T) {
			event := map[string]any{
				"context": map[string]any{
					"consents": c.given,
				},
			}
			got := SatisfiesEvent(requiredPurposes(c.required), c.matchAll, event)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}

	cases := []struct {
		name     string
		required []string
		matchAll bool
		event    map[string]any
		want     bool
	}{
		{
			name:     "AND: missing context",
			required: []string{"marketing"},
			matchAll: true,
			event:    map[string]any{},
			want:     false,
		},
		{
			name:     "AND: missing consents",
			required: []string{"marketing"},
			matchAll: true,
			event: map[string]any{
				"context": map[string]any{},
			},
			want: false,
		},
		{
			name:     "OR: missing context",
			required: []string{"marketing"},
			matchAll: false,
			event:    map[string]any{},
			want:     false,
		},
		{
			name:     "no required purposes and missing context",
			required: nil,
			matchAll: true,
			event:    map[string]any{},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesEvent(requiredPurposes(c.required), c.matchAll, c.event)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestSatisfiesProfile(t *testing.T) {
	for _, c := range givenConsentsCases {
		t.Run(c.name, func(t *testing.T) {
			profile := map[string]any{"consents": c.given}
			got := SatisfiesProfile(requiredPurposes(c.required), c.matchAll, profile)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}

	cases := []struct {
		name     string
		required []string
		matchAll bool
		profile  map[string]any
		want     bool
	}{
		{
			name:     "AND: missing consents",
			required: []string{"marketing"},
			matchAll: true,
			profile:  map[string]any{},
			want:     false,
		},
		{
			name:     "AND: consents is not an object",
			required: []string{"marketing"},
			matchAll: true,
			profile: map[string]any{
				"consents": true,
			},
			want: false,
		},
		{
			name:     "OR: missing consents",
			required: []string{"marketing"},
			matchAll: false,
			profile:  map[string]any{},
			want:     false,
		},
		{
			name:     "no required purposes and missing consents",
			required: nil,
			matchAll: true,
			profile:  map[string]any{},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesProfile(requiredPurposes(c.required), c.matchAll, c.profile)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// TestSatisfiesProfileJSONKeys checks explicit JSON keys without implicit JSON
// traversal or coercion.
func TestSatisfiesProfileJSONKeys(t *testing.T) {

	tests := []struct {
		name     string
		property string
		key      string
		value    any
		want     bool
	}{
		{
			name:  "literal dot",
			key:   "a.b",
			value: json.Value(`{"a.b":true,"a":{"b":false}}`),
			want:  true,
		},
		{
			name:  "spaces",
			key:   " purpose code ",
			value: json.Value(`{" purpose code ":true}`),
			want:  true,
		},
		{
			name:  "quotes in key",
			key:   `say "yes"`,
			value: json.Value(`{"say \"yes\"":true}`),
			want:  true,
		},
		{
			name:     "dotted path",
			property: "consents.marketing",
			value:    json.Value(`{"marketing":true}`),
			want:     false,
		},
		{
			name:     "JSON prefix",
			property: "consents.nested",
			key:      "marketing",
			value:    json.Value(`{"nested":{"marketing":true}}`),
			want:     false,
		},
		{
			name:  "bare JSON Boolean",
			value: json.Value("true"),
			want:  false,
		},
		{
			name:  "empty key means no JSON key",
			value: json.Value(`{"":true}`),
			want:  false,
		},
		{
			name:  "missing",
			key:   "marketing",
			value: json.Value(`{}`),
			want:  false,
		},
		{
			name:  "false",
			key:   "marketing",
			value: json.Value(`{"marketing":false}`),
			want:  false,
		},
		{
			name:  "null",
			key:   "marketing",
			value: json.Value(`{"marketing":null}`),
			want:  false,
		},
		{
			name:  "string",
			key:   "marketing",
			value: json.Value(`{"marketing":"true"}`),
			want:  false,
		},
		{
			name:  "number",
			key:   "marketing",
			value: json.Value(`{"marketing":1}`),
			want:  false,
		},
		{
			name:  "array",
			key:   "0",
			value: json.Value(`[true]`),
			want:  false,
		},
		{
			name:  "schema object instead of JSON",
			key:   "marketing",
			value: map[string]any{"marketing": true},
			want:  false,
		},
		{
			name: "literal backslash escape", key: `\u0061`,
			value: json.Value(`{"\\u0061":true,"a":false}`), want: true,
		},
		{
			name: "control characters in key", key: "a\n",
			value: json.Value(`{"a\n":true}`), want: true,
		},
		{name: "Boolean property with a JSON key", key: "marketing", value: true, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			property := test.property
			if property == "" {
				property = "consents"
			}
			purpose := purposeWithLocations("marketing", "", property, test.key)
			got := SatisfiesProfile([]*state.ConsentPurpose{purpose}, true, map[string]any{"consents": test.value})
			if got != test.want {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}

}

// TestSatisfiesWithConfiguredProfilePaths checks consent values at configured profile locations.
func TestSatisfiesWithConfiguredProfilePaths(t *testing.T) {
	cases := []struct {
		name       string
		purposes   []*state.ConsentPurpose
		matchAll   bool
		attributes map[string]any
		want       bool
	}{
		{
			name:     "nested profile path",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": true},
			},
			want: true,
		},
		{
			name:     "the profile path holds a value that is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": "true"},
			},
			want: false,
		},
		{
			name:     "the profile path holds an object",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": true},
			},
			want: false,
		},
		{
			name:     "profile path inside a JSON property",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy", "marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": json.Value(`{"marketing":true}`),
			},
			want: true,
		},
		{
			name:     "the profile path holds a JSON property that is a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": json.Value("true"),
			},
			want: false,
		},
		{
			name:     "the profile path holds a JSON property nested in an object that is a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": json.Value("true")},
			},
			want: false,
		},
		{
			name:     "the profile path inside a JSON property holds a value that is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy", "marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": json.Value(`{"marketing":"true"}`),
			},
			want: false,
		},
		{
			name:       "the profile path does not exist",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			matchAll:   true,
			attributes: map[string]any{},
			want:       false,
		},
		{
			name: "AND: every purpose is read from its own profile path",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "privacy.marketing"),
				purposeWithLocations("analytics", "", "analyticsConsent"),
			},
			matchAll: true,
			attributes: map[string]any{
				"privacy":          map[string]any{"marketing": true},
				"analyticsConsent": true,
			},
			want: true,
		},
		{
			name: "OR: only the purpose read from the nested profile path is granted",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "privacy.marketing"),
				purposeWithLocations("analytics", "", "analyticsConsent"),
			},
			matchAll: false,
			attributes: map[string]any{
				"privacy":          map[string]any{"marketing": true},
				"analyticsConsent": false,
			},
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesProfile(c.purposes, c.matchAll, c.attributes)
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

func TestSatisfiesWithUnknownPath(t *testing.T) {

	cases := []struct {
		name       string
		purposes   []*state.ConsentPurpose
		matchAll   bool
		attributes map[string]any
		event      bool
		want       bool
	}{
		{
			name:       "AND: unknown event path",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "", "consents.marketing")},
			matchAll:   true,
			attributes: map[string]any{"context": map[string]any{"consents": map[string]any{"marketing": true}}},
			event:      true,
			want:       false,
		},
		{
			name:       "OR: unknown event path",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "", "consents.marketing")},
			attributes: map[string]any{"context": map[string]any{"consents": map[string]any{"marketing": true}}},
			event:      true,
			want:       false,
		},
		{
			name: "OR: unknown event path and another purpose is granted",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "consents.marketing"),
				purposeWithLocations("analytics", "analytics", "consents.analytics"),
			},
			attributes: map[string]any{"context": map[string]any{"consents": map[string]any{"analytics": true}}},
			event:      true,
			want:       true,
		},
		{
			name:       "AND: unknown profile path",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "marketing", "")},
			matchAll:   true,
			attributes: map[string]any{"consents": map[string]any{"marketing": true}},
			want:       false,
		},
		{
			name:       "OR: unknown profile path",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "marketing", "")},
			attributes: map[string]any{"consents": map[string]any{"marketing": true}},
			want:       false,
		},
		{
			name: "OR: unknown profile path and another purpose is granted",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "marketing", ""),
				purposeWithLocations("analytics", "analytics", "consents.analytics"),
			},
			attributes: map[string]any{"consents": map[string]any{"analytics": true}},
			want:       true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got bool
			if c.event {
				got = SatisfiesEvent(c.purposes, c.matchAll, c.attributes)
			} else {
				got = SatisfiesProfile(c.purposes, c.matchAll, c.attributes)
			}
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}

}

// TestSatisfiesWithMultipleEventPaths checks event path precedence and AND/OR
// combinations of consent purposes.
func TestSatisfiesWithMultipleEventPaths(t *testing.T) {
	cases := []struct {
		name     string
		purposes []*state.ConsentPurpose
		matchAll bool
		given    map[string]any
		profile  bool
		want     bool
	}{
		{
			name:     "the consent is given with the first path",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"marketing": true},
			want:     true,
		},
		{
			name:     "the consent is given with another path",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
		{
			name:     "the consent is given with a special event name",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"#CFK567": true},
			want:     true,
		},
		{
			name:     "an event name containing a period is read as one key",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "vendor.marketing")},
			matchAll: true,
			given:    map[string]any{"vendor.marketing": true},
			want:     true,
		},
		{
			name:     "the first path denies the consent and another grants it",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": false, "mkt": true},
			want:     false,
		},
		{
			name:     "the first present path is null and another grants consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": nil, "mkt": true},
			want:     false,
		},
		{
			name:     "the first present path is not a bool and another grants consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": "true", "mkt": true},
			want:     false,
		},
		{
			name:     "another path denies the consent and the first grants it",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": true, "mkt": false},
			want:     true,
		},
		{
			name:     "none of the paths grant the consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"mkt": false, "other": true},
			want:     false,
		},
		{
			name: "AND: each purpose is granted with one of its paths",
			purposes: []*state.ConsentPurpose{
				purposeWithEventKeys("marketing", "mkt"),
				purposeWithEventKeys("analytics", "#CFK567"),
			},
			matchAll: true,
			given:    map[string]any{"mkt": true, "#CFK567": true},
			want:     true,
		},
		{
			name: "OR: only the purpose granted with another path satisfies the consents",
			purposes: []*state.ConsentPurpose{
				purposeWithEventKeys("marketing", "mkt"),
				purposeWithEventKeys("analytics", "#CFK567"),
			},
			matchAll: false,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
		{
			name:     "additional event paths are ignored on a profile",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"mkt": true},
			profile:  true,
			want:     false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got bool
			if c.profile {
				got = SatisfiesProfile(c.purposes, c.matchAll, map[string]any{"consents": c.given})
			} else {
				event := map[string]any{"context": map[string]any{"consents": c.given}}
				got = SatisfiesEvent(c.purposes, c.matchAll, event)
			}
			if got != c.want {
				t.Fatalf("got %v, want %v", got, c.want)
			}
		})
	}
}

// purposeWithEventKeys returns a consent purpose whose event paths lead to the
// given keys under context.consents.
func purposeWithEventKeys(key string, otherKeys ...string) *state.ConsentPurpose {
	eventLocations := make([]state.EventConsentLocation, len(otherKeys)+1)
	eventLocations[0] = state.EventConsentLocation{PurposeCode: key}
	for i, code := range otherKeys {
		eventLocations[i+1] = state.EventConsentLocation{PurposeCode: code}
	}
	return state.NewConsentPurpose(state.ConsentPurpose{
		ID:                     key,
		Name:                   key,
		EventConsentLocations:  eventLocations,
		ProfileConsentLocation: &state.ProfileConsentLocation{Property: "consents." + key},
	})
}

// requiredPurposes returns the consent purposes with the given identifiers, each one
// with the default property paths.
func requiredPurposes(ids []string) []*state.ConsentPurpose {
	purposes := make([]*state.ConsentPurpose, len(ids))
	for i, id := range ids {
		purposes[i] = purposeWithLocations(id, id, "consents."+id)
	}
	return purposes
}

// purposeWithLocations returns a purpose with the given event consent name and
// profile property and optional JSON key.
func purposeWithLocations(id, eventName, profileProperty string, jsonKeys ...string) *state.ConsentPurpose {
	purpose := state.ConsentPurpose{ID: id, Name: id}
	if eventName != "" {
		purpose.EventConsentLocations = []state.EventConsentLocation{{PurposeCode: eventName}}
	}
	if profileProperty != "" {
		purpose.ProfileConsentLocation = &state.ProfileConsentLocation{Property: profileProperty}
		if len(jsonKeys) > 0 {
			purpose.ProfileConsentLocation.JSONKey = jsonKeys[0]
		}
	}
	return state.NewConsentPurpose(purpose)
}
