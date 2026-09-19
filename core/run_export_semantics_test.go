// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestConvertToExternalSemantics checks country validation and phone
// preservation when the internal and external types have the same semantic and
// options.
func TestConvertToExternalSemantics(t *testing.T) {

	countryAlpha2 := types.String().AsCountry(types.ISO3166Alpha2)
	phone := types.String().AsPhone()

	tests := []struct {
		name      string
		typ       types.Type
		value     string
		wantValid bool
	}{
		{"current country", countryAlpha2, "IT", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ITA", true},
		{"long alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ITAL", false},
		{"former country", countryAlpha2, "AN", true},
		{"unknown country", countryAlpha2, "ZZ", false},
		{"reserved country", countryAlpha2, "UK", false},
		{"lowercase country", countryAlpha2, "it", false},
		{"empty country", countryAlpha2, "", false},
		{"short country", countryAlpha2, "I", false},
		{"long country", countryAlpha2, "ITA", false},
		{"non-ASCII country", countryAlpha2, "é", false},
		{"canonical phone", phone, "+390236618300", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := convertToExternal(test.value, test.typ, test.typ, "source", "target")
			if err != nil {
				if test.wantValid {
					t.Fatalf("expected no error, got %v", err)
				}
				expected := errMatchingPropertyConversion("source", "target")
				if err.Error() != expected.Error() {
					t.Fatalf("expected error %q, got %q", expected, err)
				}
				return
			}
			if !test.wantValid {
				expected := errMatchingPropertyConversion("source", "target")
				t.Fatalf("expected error %q, got nil", expected)
			}
			if got != test.value {
				t.Fatalf("expected %#v, got %#v", test.value, got)
			}
		})
	}

}
