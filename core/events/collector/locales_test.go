//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package collector

import (
	"testing"
)

func Test_countryCode(t *testing.T) {

	tests := []struct {
		code string
		ok   bool
	}{
		{"", false},
		{"US", true},
		{"-US", false},
		{"Us", false},
		{"us", false},
		{"GB", true},
		{"FR", true},
		{"IT", true},
		{"CN", true},
		{"LK", true},
		{"CI", false},
	}

	for _, test := range tests {
		got, ok := countryCode(test.code)
		if ok != test.ok {
			t.Errorf("countryCode(%q): expected %t, got %t", test.code, test.ok, ok)
		}
		if test.ok && test.code != got {
			t.Errorf("countryCode(%q): expected %q, got %q", test.code, test.code, got)
		}
	}
}

func Test_localeCode(t *testing.T) {

	tests := []struct {
		code     string
		expected string
		ok       bool
	}{
		{"", "", false},
		{"en", "en-US", true},
		{"en-US", "en-US", true},
		{"en-GB", "en-GB", true},
		{"fr", "fr-FR", true},
		{"it", "it-IT", true},
		{"it-", "", false},
		{"FRfr-", "", false},
		{"zh", "zh-CN", true},
		{"zh-CN", "zh-CN", true},
		{"si-LK", "si-LK", true},
	}

	for _, test := range tests {
		got, ok := localeCode(test.code)
		if ok != test.ok {
			t.Errorf("localeCode(%q): expected %t, got %t", test.code, test.ok, ok)
		}
		if test.ok && test.expected != got {
			t.Errorf("localeCode(%q): expected %q, got %q", test.code, test.expected, got)
		}
	}
}
