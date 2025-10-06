//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package validation

import "testing"

func TestIsValidCurrencyCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{name: "fast path", code: "USD", want: true},
		{name: "slow path", code: "CHF", want: true},
		{name: "unknown code", code: "ZZZ", want: false},
		{name: "empty string", code: "", want: false},
		{name: "too short", code: "EU", want: false},
		{name: "too long", code: "USDT", want: false},
		{name: "lowercase", code: "eur", want: false},
	}
	for _, test := range tests {
		if got := IsValidCurrencyCode(test.code); got != test.want {
			t.Errorf("expected %t for code %q (%s), got %t", test.want, test.code, test.name, got)
		}
	}
}
