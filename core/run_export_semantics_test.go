// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestConvertToExternalSemantics checks the matching property's semantic after conversion.
func TestConvertToExternalSemantics(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic types.Type
		value    string
		valid    bool
	}{
		{"current country", country, "IT", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ITA", true},
		{"long alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ITAL", false},
		{"former country", country, "AN", true},
		{"unknown country", country, "ZZ", false},
		{"reserved country", country, "UK", false},
		{"lowercase country", country, "it", false},
		{"empty country", country, "", false},
		{"short country", country, "I", false},
		{"long country", country, "ITA", false},
		{"non-ASCII country", country, "é", false},
		{"phone at limit", types.String().AsPhone(), "+123456789012345", true},
		{"long phone", types.String().AsPhone(), "+1234567890123456", false},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", true},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", false},
		{"empty phone", types.String().AsPhone(), "", true},
		{"invalid UTF-8 phone", types.String().AsPhone(), "\xff", false},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", true},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			schema := types.Object([]types.Property{{Name: "value", Type: test.semantic}})
			p, _ := schema.Properties().ByName("value")
			got, err := convertToExternal(test.value, types.String(), p.Type, "source", "target")
			if err != nil {
				if test.valid {
					t.Fatal(err)
				}
				if err.Error() != errMatchingPropertyConversion("source", "target").Error() {
					t.Fatalf("unexpected error: %s", err)
				}
				return
			}
			if !test.valid {
				t.Fatal("invalid matching value was accepted")
			}
			if got != test.value {
				t.Fatalf("value changed: got %#v, want %#v", got, test.value)
			}

		})

	}

	p := types.Property{Name: "country", Type: country}
	_, err := convertToExternal(12, types.Int(32), p.Type, "source", "target")
	if err != nil {
		return
	}
	t.Fatal("numeric value was accepted as a country code")

}
