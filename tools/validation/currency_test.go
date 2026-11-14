// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
