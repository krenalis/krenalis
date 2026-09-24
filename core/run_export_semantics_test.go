// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestConvertToExternalSemantics checks that matching country and phone types
// pass their values through unchanged.
func TestConvertToExternalSemantics(t *testing.T) {

	countryAlpha2 := types.String().AsCountry(types.ISO3166Alpha2)
	countryAlpha3 := types.String().AsCountry(types.ISO3166Alpha3)
	phone := types.String().AsPhone()

	tests := []struct {
		name   string
		in, ex types.Type
		value  string
	}{
		{"current country", countryAlpha2, countryAlpha2, "IT"},
		{"former country", countryAlpha2, countryAlpha2, "AN"},
		{"alpha-3 country", countryAlpha3, countryAlpha3, "ITA"},
		{"matching country semantics skip validation", countryAlpha2, countryAlpha2, "ZZ"},
		{"matching phone semantics preserve value", phone, phone, "+390236618300"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := convertToExternal(test.value, test.in, test.ex, "source", "target")
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != test.value {
				t.Fatalf("expected %#v, got %#v", test.value, got)
			}
		})
	}

}
