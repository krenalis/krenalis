// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package consents

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
)

var givenConsentsCases = []struct {
	name     string
	required []string
	matchAll bool
	given    map[string]any
	want     bool
}{
	{
		name:     "no required codes",
		required: nil,
		matchAll: true,
		given:    map[string]any{},
		want:     true,
	},
	{
		name:     "AND: all required codes are true",
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
		name:     "AND: one required code is false",
		required: []string{"marketing", "analytics"},
		matchAll: true,
		given: map[string]any{
			"marketing": true,
			"analytics": false,
		},
		want: false,
	},
	{
		name:     "AND: one required code is missing",
		required: []string{"marketing", "analytics"},
		matchAll: true,
		given: map[string]any{
			"marketing": true,
		},
		want: false,
	},
	{
		name:     "AND: required code is not a bool",
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
		name:     "OR: all required codes are true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"marketing": true,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: one required code is true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"marketing": false,
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: one required code is missing and the other is true",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"analytics": true,
		},
		want: true,
	},
	{
		name:     "OR: every required code is missing",
		required: []string{"marketing", "analytics"},
		matchAll: false,
		given: map[string]any{
			"other": true,
		},
		want: false,
	},
	{
		name:     "OR: no required code is true",
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
			name:     "no required codes and missing context",
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
			name:     "no required codes and missing consents",
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

func TestSatisfiesWithConfiguredPaths(t *testing.T) {
	cases := []struct {
		name       string
		purposes   []*state.ConsentPurpose
		matchAll   bool
		attributes map[string]any
		event      bool
		want       bool
	}{
		{
			name:     "event path out of the context",
			purposes: []*state.ConsentPurpose{purposeWithPaths("marketing", "consents.marketing", "")},
			matchAll: true,
			attributes: map[string]any{
				"consents": map[string]any{"marketing": true},
			},
			event: true,
			want:  true,
		},
		{
			name:     "event path at the root of the event",
			purposes: []*state.ConsentPurpose{purposeWithPaths("marketing", "marketingConsent", "")},
			matchAll: true,
			attributes: map[string]any{
				"marketingConsent": true,
			},
			event: true,
			want:  true,
		},
		{
			name:     "the event path takes precedence over the code",
			purposes: []*state.ConsentPurpose{purposeWithPaths("marketing", "marketingConsent", "")},
			matchAll: true,
			attributes: map[string]any{
				"context":          map[string]any{"consents": map[string]any{"marketing": true}},
				"marketingConsent": false,
			},
			event: true,
			want:  false,
		},
		{
			name:     "nested profile path",
			purposes: []*state.ConsentPurpose{purposeWithPaths("marketing", "", "privacy.marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": true},
			},
			want: true,
		},
		{
			name:     "the profile path holds a value that is not a bool",
			purposes: []*state.ConsentPurpose{purposeWithPaths("marketing", "", "privacy.marketing")},
			matchAll: true,
			attributes: map[string]any{
				"privacy": map[string]any{"marketing": "true"},
			},
			want: false,
		},
		{
			name:       "the profile path does not exist",
			purposes:   []*state.ConsentPurpose{purposeWithPaths("marketing", "", "privacy.marketing")},
			matchAll:   true,
			attributes: map[string]any{},
			want:       false,
		},
		{
			name: "AND: every purpose is read from its own profile path",
			purposes: []*state.ConsentPurpose{
				purposeWithPaths("marketing", "", "privacy.marketing"),
				purposeWithPaths("analytics", "", "analyticsConsent"),
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
				purposeWithPaths("marketing", "", "privacy.marketing"),
				purposeWithPaths("analytics", "", "analyticsConsent"),
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

func TestSatisfiesWithAliases(t *testing.T) {
	cases := []struct {
		name     string
		purposes []*state.ConsentPurpose
		matchAll bool
		given    map[string]any
		profile  bool
		want     bool
	}{
		{
			name:     "the consent is given with the code",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"marketing": true},
			want:     true,
		},
		{
			name:     "the consent is given with an alias",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
		{
			name:     "the consent is given with an alias that is not a property name",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"#CFK567": true},
			want:     true,
		},
		{
			name:     "the code denies the consent and an alias grants it",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": false, "mkt": true},
			want:     true,
		},
		{
			name:     "an alias denies the consent and the code grants it",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt")},
			matchAll: true,
			given:    map[string]any{"marketing": true, "mkt": false},
			want:     true,
		},
		{
			name:     "neither the code nor the aliases grant the consent",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt", "#CFK567")},
			matchAll: true,
			given:    map[string]any{"mkt": false, "other": true},
			want:     false,
		},
		{
			name: "AND: each purpose is granted with one of its aliases",
			purposes: []*state.ConsentPurpose{
				purposeWithAliases("marketing", "mkt"),
				purposeWithAliases("analytics", "#CFK567"),
			},
			matchAll: true,
			given:    map[string]any{"mkt": true, "#CFK567": true},
			want:     true,
		},
		{
			name: "OR: only the purpose granted with an alias satisfies the consents",
			purposes: []*state.ConsentPurpose{
				purposeWithAliases("marketing", "mkt"),
				purposeWithAliases("analytics", "#CFK567"),
			},
			matchAll: false,
			given:    map[string]any{"mkt": true},
			want:     true,
		},
		{
			name:     "the aliases are ignored on a profile",
			purposes: []*state.ConsentPurpose{purposeWithAliases("marketing", "mkt")},
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

func TestSatisfiesWithAliasesAndConfiguredEventPath(t *testing.T) {
	// The aliases are not read when the event path is configured, because that
	// path alone holds the consent.
	purpose := state.NewConsentPurpose(state.ConsentPurpose{
		ID:        "marketing",
		Code:      "marketing",
		Name:      "marketing",
		Aliases:   []string{"mkt"},
		EventPath: "properties.marketingConsent",
	})
	purposes := []*state.ConsentPurpose{purpose}
	event := map[string]any{
		"context":    map[string]any{"consents": map[string]any{"marketing": true, "mkt": true}},
		"properties": map[string]any{"marketingConsent": false},
	}
	if SatisfiesEvent(purposes, true, event) {
		t.Fatal("got true, want false")
	}
	event["properties"] = map[string]any{"marketingConsent": true}
	if !SatisfiesEvent(purposes, true, event) {
		t.Fatal("got false, want true")
	}
}

// purposeWithAliases returns the consent purpose with the given code and
// aliases, with the default property paths.
func purposeWithAliases(code string, aliases ...string) *state.ConsentPurpose {
	return state.NewConsentPurpose(state.ConsentPurpose{
		ID:      code,
		Code:    code,
		Name:    code,
		Aliases: aliases,
	})
}

// requiredPurposes returns the consent purposes with the given codes, each one
// with the default property paths.
func requiredPurposes(codes []string) []*state.ConsentPurpose {
	purposes := make([]*state.ConsentPurpose, len(codes))
	for i, code := range codes {
		purposes[i] = purposeWithPaths(code, "", "")
	}
	return purposes
}

// purposeWithPaths returns the consent purpose with the given code, event path
// and profile path.
func purposeWithPaths(code, eventPath, profilePath string) *state.ConsentPurpose {
	return state.NewConsentPurpose(state.ConsentPurpose{
		ID:          code,
		Code:        code,
		Name:        code,
		EventPath:   eventPath,
		ProfilePath: profilePath,
	})
}
