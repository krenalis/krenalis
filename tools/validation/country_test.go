// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package validation

import "testing"

// TestIsValidCountryCodeAlpha2 tests current, formerly assigned, and invalid country codes.
func TestIsValidCountryCodeAlpha2(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "fast path", code: "IT", want: true},
		{name: "slow path", code: "CH", want: true},
		{name: "formerly assigned", code: "AN", want: true},
		{name: "second formerly assigned", code: "SU", want: true},
		{name: "unaligned occurrence", code: "DA", want: false},
		{name: "reserved", code: "UK", want: false},
		{name: "user-assigned", code: "ZZ", want: false},
		{name: "empty string", code: "", want: false},
		{name: "too short", code: "I", want: false},
		{name: "too long", code: "ITA", want: false},
		{name: "lowercase", code: "it", want: false},
	}
	for _, test := range tests {
		if got := IsValidCountryCodeAlpha2(test.code); got != test.want {
			t.Errorf("expected %t for code %q (%s), got %t", test.want, test.code, test.name, got)
		}
	}
}

// TestIsValidCountryCodeAlpha3 tests current, formerly assigned, and invalid alpha-3 codes.
func TestIsValidCountryCodeAlpha3(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "fast path", code: "ITA", want: true},
		{name: "slow path", code: "CHE", want: true},
		{name: "formerly assigned", code: "ANT", want: true},
		{name: "second formerly assigned", code: "SUN", want: true},
		{name: "former Czechoslovakia", code: "CSK", want: true},
		{name: "former Serbia and Montenegro", code: "SCG", want: true},
		{name: "former Sikkim", code: "SKM", want: true},
		{name: "unaligned occurrence", code: "BWA", want: true},
		{name: "invalid boundary match", code: "WAF", want: false},
		{name: "reserved", code: "EUR", want: false},
		{name: "user-assigned", code: "ZZZ", want: false},
		{name: "empty string", code: "", want: false},
		{name: "alpha-2", code: "IT", want: false},
		{name: "too long", code: "ITAL", want: false},
		{name: "lowercase", code: "ita", want: false},
		{name: "mixed case", code: "Ita", want: false},
		{name: "leading space", code: " ITA", want: false},
		{name: "trailing space", code: "ITA ", want: false},
		{name: "numeric", code: "380", want: false},
		{name: "non-ASCII", code: "éA", want: false},
	}
	for _, test := range tests {
		if got := IsValidCountryCodeAlpha3(test.code); got != test.want {
			t.Errorf("expected %t for code %q (%s), got %t", test.want, test.code, test.name, got)
		}
	}
}
