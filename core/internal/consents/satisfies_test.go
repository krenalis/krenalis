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
	operator state.ConsentPurposesOperator
	given    map[string]any
	want     bool
}{
	{
		name:     "no required purposes",
		required: nil,
		operator: state.PurposesAnd,
		given:    map[string]any{},
		want:     true,
	},
	{
		name:     "all: all required consents are true",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesAnd,
		given: map[string]any{
			"marketing": true,
			"analytics": true,
			"other":     false,
		},
		want: true,
	},
	{
		name:     "all: one required consent is false",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesAnd,
		given: map[string]any{
			"marketing": true,
			"analytics": false,
		},
		want: false,
	},
	{
		name:     "all: one required consent is missing",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesAnd,
		given: map[string]any{
			"marketing": true,
		},
		want: false,
	},
	{
		name:     "all: required consent is not a bool",
		required: []string{"marketing"},
		operator: state.PurposesAnd,
		given: map[string]any{
			"marketing": "true",
		},
		want: false,
	},
	{
		name:     "all: no consent is given",
		required: []string{"marketing"},
		operator: state.PurposesAnd,
		given:    map[string]any{},
		want:     false,
	},
	{
		name:     "any: all required consents are true",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesOr,
		given: map[string]any{
			"marketing": true,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "any: one required consent is true",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesOr,
		given: map[string]any{
			"marketing": false,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "any: one required consent is missing and the other is true",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesOr,
		given: map[string]any{
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "any: every required consent is missing",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesOr,
		given: map[string]any{
			"other": true,
		},
		want: false,
	},
	{
		name:     "any: no required consent is true",
		required: []string{"marketing", "analytics"},
		operator: state.PurposesOr,
		given: map[string]any{
			"marketing": false,
			"analytics": false,
		},
		want: false,
	},
	{
		name:     "any: no consent is given",
		required: []string{"marketing"},
		operator: state.PurposesOr,
		given:    map[string]any{},
		want:     false,
	},
}

// BenchmarkSatisfiesEvent measures event consent checks.
func BenchmarkSatisfiesEvent(b *testing.B) {
	purposes := []*state.ConsentPurpose{
		purposeWithEventKeys("marketing", "mkt"),
		purposeWithEventKeys("analytics"),
	}
	event := map[string]any{
		"context": map[string]any{
			"consents": map[string]any{"mkt": true, "analytics": json.Value("true")},
		},
	}
	tests := []struct {
		name     string
		purposes []*state.ConsentPurpose
		operator state.ConsentPurposesOperator
	}{
		{name: "all", purposes: purposes, operator: state.PurposesAnd},
		{name: "any", purposes: purposes, operator: state.PurposesOr},
		{
			name: "fallback-location",
			purposes: []*state.ConsentPurpose{
				purposeWithEventKeys("marketing", "missing", "missing2", "mkt"),
				purposeWithEventKeys("analytics"),
			},
			operator: state.PurposesAnd,
		},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var got bool
			b.ReportAllocs()
			for b.Loop() {
				got = SatisfiesEvent(test.operator, test.purposes, event)
			}
			if !got {
				b.Fatal("expected consent, got false")
			}
		})
	}
}

// BenchmarkSatisfiesProfile measures profile consent checks.
func BenchmarkSatisfiesProfile(b *testing.B) {
	purposes := []*state.ConsentPurpose{
		purposeWithLocations("marketing", "", "privacy.marketing"),
		purposeWithLocations("analytics", "", "consents", "analytics"),
	}
	profile := map[string]any{
		"privacy":  map[string]any{"marketing": true},
		"consents": json.Value(`{"analytics":true}`),
	}
	tests := []struct {
		name     string
		operator state.ConsentPurposesOperator
	}{
		{name: "all", operator: state.PurposesAnd},
		{name: "any", operator: state.PurposesOr},
	}
	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			var got bool
			b.ReportAllocs()
			for b.Loop() {
				got = SatisfiesProfile(test.operator, purposes, profile)
			}
			if !got {
				b.Fatal("expected consent, got false")
			}
		})
	}
}

func TestSatisfiesEvent(t *testing.T) {

	for _, c := range givenConsentsCases {
		t.Run(c.name, func(t *testing.T) {
			event := map[string]any{
				"context": map[string]any{
					"consents": c.given,
				},
			}
			got := SatisfiesEvent(c.operator, requiredPurposes(c.required), event)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

	cases := []struct {
		name     string
		required []string
		operator state.ConsentPurposesOperator
		event    map[string]any
		want     bool
	}{
		{
			name:     "all: missing context",
			required: []string{"marketing"},
			operator: state.PurposesAnd,
			event:    map[string]any{},
			want:     false,
		},
		{
			name:     "all: missing consents",
			required: []string{"marketing"},
			operator: state.PurposesAnd,
			event: map[string]any{
				"context": map[string]any{},
			},
			want: false,
		},
		{
			name:     "any: missing context",
			required: []string{"marketing"},
			operator: state.PurposesOr,
			event:    map[string]any{},
			want:     false,
		},
		{
			name:     "no required purposes and missing context",
			required: nil,
			operator: state.PurposesAnd,
			event:    map[string]any{},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesEvent(c.operator, requiredPurposes(c.required), c.event)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

}

func TestSatisfiesProfile(t *testing.T) {

	for _, c := range givenConsentsCases {
		t.Run(c.name, func(t *testing.T) {
			profile := map[string]any{"consents": c.given}
			got := SatisfiesProfile(c.operator, requiredPurposes(c.required), profile)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

	cases := []struct {
		name     string
		required []string
		operator state.ConsentPurposesOperator
		profile  map[string]any
		want     bool
	}{
		{
			name:     "all: missing consents",
			required: []string{"marketing"},
			operator: state.PurposesAnd,
			profile:  map[string]any{},
			want:     false,
		},
		{
			name:     "all: consents is not an object",
			required: []string{"marketing"},
			operator: state.PurposesAnd,
			profile: map[string]any{
				"consents": true,
			},
			want: false,
		},
		{
			name:     "any: missing consents",
			required: []string{"marketing"},
			operator: state.PurposesOr,
			profile:  map[string]any{},
			want:     false,
		},
		{
			name:     "no required purposes and missing consents",
			required: nil,
			operator: state.PurposesAnd,
			profile:  map[string]any{},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesProfile(c.operator, requiredPurposes(c.required), c.profile)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
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
			name:  "literal dot in JSON key",
			key:   "a.b",
			value: json.Value(`{"a.b":true,"a":{"b":false}}`),
			want:  true,
		},
		{
			name:  "spaces in JSON key",
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
			name:     "property path does not traverse JSON",
			property: "consents.marketing",
			value:    json.Value(`{"marketing":true}`),
			want:     false,
		},
		{
			name:     "property path does not traverse into JSON",
			property: "consents.nested",
			key:      "marketing",
			value:    json.Value(`{"nested":{"marketing":true}}`),
			want:     false,
		},
		{
			name:  "bare JSON boolean is not accepted",
			value: json.Value("true"),
			want:  false,
		},
		{
			name:  "empty key means no JSON key",
			value: json.Value(`{"":true}`),
			want:  false,
		},
		{
			name:  "missing JSON key",
			key:   "marketing",
			value: json.Value(`{}`),
			want:  false,
		},
		{
			name:  "false JSON value",
			key:   "marketing",
			value: json.Value(`{"marketing":false}`),
			want:  false,
		},
		{
			name:  "null JSON value",
			key:   "marketing",
			value: json.Value(`{"marketing":null}`),
			want:  false,
		},
		{
			name:  "string JSON value",
			key:   "marketing",
			value: json.Value(`{"marketing":"true"}`),
			want:  false,
		},
		{
			name:  "number JSON value",
			key:   "marketing",
			value: json.Value(`{"marketing":1}`),
			want:  false,
		},
		{
			name:  "JSON array",
			key:   "0",
			value: json.Value(`[true]`),
			want:  false,
		},
		{
			name:  "map instead of JSON value",
			key:   "marketing",
			value: map[string]any{"marketing": true},
			want:  false,
		},
		{
			name:  "literal backslash escape in key",
			key:   `\u0061`,
			value: json.Value(`{"\\u0061":true,"a":false}`),
			want:  true,
		},
		{
			name:  "control characters in key",
			key:   "a\n",
			value: json.Value(`{"a\n":true}`),
			want:  true,
		},
		{
			name:  "bool property does not support JSON key",
			key:   "marketing",
			value: true,
			want:  false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			property := test.property
			if property == "" {
				property = "consents"
			}
			purpose := purposeWithLocations("marketing", "", property, test.key)
			got := SatisfiesProfile(state.PurposesAnd, []*state.ConsentPurpose{purpose}, map[string]any{"consents": test.value})
			if got != test.want {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}

}

// TestSatisfiesProfileLocations checks consent values at profile locations.
func TestSatisfiesProfileLocations(t *testing.T) {
	cases := []struct {
		name       string
		purposes   []*state.ConsentPurpose
		operator   state.ConsentPurposesOperator
		attributes map[string]any
		want       bool
	}{
		{
			name:     "nested profile property grants consent",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": true},
			},
			want: true,
		},
		{
			name:     "profile property is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": "true"},
			},
			want: false,
		},
		{
			name:     "profile property is an object",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": true},
			},
			want: false,
		},
		{
			name:     "JSON key grants consent",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy", "marketing")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": json.Value(`{"marketing":true}`),
			},
			want: true,
		},
		{
			name:     "bare JSON boolean is not accepted",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": json.Value("true"),
			},
			want: false,
		},
		{
			name:     "nested bare JSON boolean is not accepted",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": json.Value("true")},
			},
			want: false,
		},
		{
			name:     "JSON key value is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy", "marketing")},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy": json.Value(`{"marketing":"true"}`),
			},
			want: false,
		},
		{
			name:       "profile property does not exist",
			purposes:   []*state.ConsentPurpose{purposeWithLocations("marketing", "", "privacy.marketing")},
			operator:   state.PurposesAnd,
			attributes: map[string]any{},
			want:       false,
		},
		{
			name: "all: each purpose is granted at its configured profile location",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "privacy.marketing"),
				purposeWithLocations("analytics", "", "analyticsConsent"),
			},
			operator: state.PurposesAnd,
			attributes: map[string]any{
				"privacy":          map[string]any{"marketing": true},
				"analyticsConsent": true,
			},
			want: true,
		},
		{
			name: "any: a later configured profile location grants consent",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "privacy.marketing"),
				purposeWithLocations("analytics", "", "analyticsConsent"),
			},
			operator: state.PurposesOr,
			attributes: map[string]any{
				"privacy":          map[string]any{"marketing": false},
				"analyticsConsent": true,
			},
			want: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesProfile(c.operator, c.purposes, c.attributes)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func TestSatisfiesEventWithoutConfiguredLocation(t *testing.T) {

	cases := []struct {
		name     string
		purposes []*state.ConsentPurpose
		operator state.ConsentPurposesOperator
		event    map[string]any
		want     bool
	}{
		{
			name:     "all: consent value exists without configured event location",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "consents.marketing")},
			operator: state.PurposesAnd,
			event:    map[string]any{"context": map[string]any{"consents": map[string]any{"marketing": true}}},
			want:     false,
		},
		{
			name:     "any: consent value exists without configured event location",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "", "consents.marketing")},
			operator: state.PurposesOr,
			event:    map[string]any{"context": map[string]any{"consents": map[string]any{"marketing": true}}},
			want:     false,
		},
		{
			name: "any: another purpose has a configured event location",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "", "consents.marketing"),
				purposeWithLocations("analytics", "analytics", "consents.analytics"),
			},
			operator: state.PurposesOr,
			event:    map[string]any{"context": map[string]any{"consents": map[string]any{"analytics": true}}},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesEvent(c.operator, c.purposes, c.event)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

}

func TestSatisfiesProfileWithoutConfiguredLocation(t *testing.T) {

	cases := []struct {
		name     string
		purposes []*state.ConsentPurpose
		operator state.ConsentPurposesOperator
		profile  map[string]any
		want     bool
	}{
		{
			name:     "all: consent value exists without configured profile location",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "marketing", "")},
			operator: state.PurposesAnd,
			profile:  map[string]any{"consents": map[string]any{"marketing": true}},
			want:     false,
		},
		{
			name:     "any: consent value exists without configured profile location",
			purposes: []*state.ConsentPurpose{purposeWithLocations("marketing", "marketing", "")},
			operator: state.PurposesOr,
			profile:  map[string]any{"consents": map[string]any{"marketing": true}},
			want:     false,
		},
		{
			name: "any: another purpose has a configured profile location",
			purposes: []*state.ConsentPurpose{
				purposeWithLocations("marketing", "marketing", ""),
				purposeWithLocations("analytics", "analytics", "consents.analytics"),
			},
			operator: state.PurposesOr,
			profile:  map[string]any{"consents": map[string]any{"analytics": true}},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SatisfiesProfile(c.operator, c.purposes, c.profile)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}

}

// TestSatisfiesEventLocations checks event location precedence and all/any
// consent matching.
func TestSatisfiesEventLocations(t *testing.T) {
	cases := []struct {
		name     string
		purposes []*state.ConsentPurpose
		operator state.ConsentPurposesOperator
		given    map[string]any
		want     bool
	}{
		{
			name:     "consent is granted at the first location",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			operator: state.PurposesAnd,
			given:    map[string]any{"marketing": true},
			want:     true,
		},
		{
			name:     "consent is granted at a fallback location",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			operator: state.PurposesAnd,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
		{
			name:     "location with special characters is supported",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			operator: state.PurposesAnd,
			given:    map[string]any{"#CFK567": true},
			want:     true,
		},
		{
			name:     "location containing a period is read as one key",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "vendor.marketing")},
			operator: state.PurposesAnd,
			given:    map[string]any{"vendor.marketing": true},
			want:     true,
		},
		{
			name:     "first present location denies consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			operator: state.PurposesAnd,
			given:    map[string]any{"marketing": false, "mkt": true},
			want:     false,
		},
		{
			name:     "first present location is null",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			operator: state.PurposesAnd,
			given:    map[string]any{"marketing": nil, "mkt": true},
			want:     false,
		},
		{
			name:     "first present location is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			operator: state.PurposesAnd,
			given:    map[string]any{"marketing": "true", "mkt": true},
			want:     false,
		},
		{
			name:     "first present location grants consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
			operator: state.PurposesAnd,
			given:    map[string]any{"marketing": true, "mkt": false},
			want:     true,
		},
		{
			name:     "fallback location denies consent",
			purposes: []*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt", "#CFK567")},
			operator: state.PurposesAnd,
			given:    map[string]any{"mkt": false, "other": true},
			want:     false,
		},
		{
			name: "all: each purpose is granted at one of its locations",
			purposes: []*state.ConsentPurpose{
				purposeWithEventKeys("marketing", "mkt"),
				purposeWithEventKeys("analytics", "#CFK567"),
			},
			operator: state.PurposesAnd,
			given:    map[string]any{"mkt": true, "#CFK567": true},
			want:     true,
		},
		{
			name: "any: one purpose is granted at a fallback location",
			purposes: []*state.ConsentPurpose{
				purposeWithEventKeys("marketing", "mkt"),
				purposeWithEventKeys("analytics", "#CFK567"),
			},
			operator: state.PurposesOr,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			event := map[string]any{"context": map[string]any{"consents": c.given}}
			got := SatisfiesEvent(c.operator, c.purposes, event)
			if got != c.want {
				t.Fatalf("expected %v, got %v", c.want, got)
			}
		})
	}
}

func TestSatisfiesProfileIgnoresEventLocations(t *testing.T) {
	if got := SatisfiesProfile(
		state.PurposesAnd,
		[]*state.ConsentPurpose{purposeWithEventKeys("marketing", "mkt")},
		map[string]any{"consents": map[string]any{"mkt": true}},
	); got {
		t.Fatalf("expected false, got %v", got)
	}
}

// purposeWithEventKeys returns a consent purpose with event locations for the
// given keys under context.consents.
func purposeWithEventKeys(eventKey string, otherKeys ...string) *state.ConsentPurpose {
	eventLocations := make([]state.EventConsentLocation, len(otherKeys)+1)
	eventLocations[0] = state.EventConsentLocation{PurposeCode: eventKey}
	for i, code := range otherKeys {
		eventLocations[i+1] = state.EventConsentLocation{PurposeCode: code}
	}
	return &state.ConsentPurpose{
		ID:                    eventKey,
		Name:                  eventKey,
		EventConsentLocations: eventLocations,
	}
}

// requiredPurposes returns consent purposes for the given IDs with their
// default event and profile locations.
func requiredPurposes(ids []string) []*state.ConsentPurpose {
	purposes := make([]*state.ConsentPurpose, len(ids))
	for i, id := range ids {
		purposes[i] = purposeWithLocations(id, id, "consents."+id)
	}
	return purposes
}

// purposeWithLocations returns a purpose with the given event consent key,
// profile property, and optional JSON key.
func purposeWithLocations(id, eventKey, profileProperty string, jsonKey ...string) *state.ConsentPurpose {
	purpose := state.ConsentPurpose{ID: id, Name: id}
	if eventKey != "" {
		purpose.EventConsentLocations = []state.EventConsentLocation{{PurposeCode: eventKey}}
	}
	if profileProperty != "" {
		purpose.ProfileConsentLocation = &state.ProfileConsentLocation{Property: profileProperty}
		if len(jsonKey) > 0 {
			purpose.ProfileConsentLocation.JSONKey = jsonKey[0]
		}
	}
	return &purpose
}
