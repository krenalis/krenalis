// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
)

// TestPhoneAndCountryColumnTypes checks the warehouse column types used for
// phone and country values.
func TestPhoneAndCountryColumnTypes(t *testing.T) {
	for _, test := range []struct {
		name string
		typ  types.Type
		want string
	}{
		{"phone", types.String().AsPhone(), "VARCHAR(16)"},
		{"phone array", types.Array(types.String().AsPhone()), "ARRAY"},
		{"country alpha2", types.String().AsCountry(types.ISO3166Alpha2), "VARCHAR(2)"},
		{"country alpha3", types.String().AsCountry(types.ISO3166Alpha3), "VARCHAR(3)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := typeToSnowflakeType(test.typ); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

// TestScannerPhone checks that phone values are canonical across supported
// container shapes.
func TestScannerPhone(t *testing.T) {

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"canonical E.164", "+390236618300", true},
		{"possible but not valid", "+12001230101", true},
		{"too short", "+12530000", false},
		{"formatted", "+39 02-36618 300", false},
		{"double plus", "++390236618300", false},
		{"padding", "+390236618300 ", false},
		{"unicode digits", "+３９０２３６６１８３００", false},
		{"national", "0236618300", false},
		{"empty", "", false},
	}
	for _, test := range tests {
		for _, shape := range []string{"string", "array", "map", "array of maps"} {
			t.Run(test.name+"/"+shape, func(t *testing.T) {

				typ := types.String().AsPhone()
				var raw any = test.input
				var want any = test.input
				switch shape {
				case "array":
					typ = types.Array(typ)
					want = []any{test.input}
				case "map":
					typ = types.Map(typ)
					want = map[string]any{"home": test.input}
				case "array of maps":
					typ = types.Array(types.Map(typ))
					want = []any{map[string]any{"home": test.input}}
				}
				if shape != "string" {
					data, err := json.Marshal(want)
					if err != nil {
						t.Fatalf("expected value to marshal, got %v", err)
					}
					raw = string(data)
				}
				s := &scanner{}
				got, err := s.normalize("phone", typ, raw)
				if err != nil {
					if test.valid {
						t.Fatalf("expected no error, got %v", err)
					}
					if test.input != "" && strings.Contains(err.Error(), test.input) {
						t.Fatalf("expected an error without the phone value, got %q", err)
					}
					return
				}
				if !test.valid {
					t.Fatalf("expected an error, got %#v", got)
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("expected unchanged value %#v, got %#v", want, got)
				}

			})
		}
	}

}

// TestScannerPhoneEscapedJSON accepts equivalent JSON encodings, not
// alternative phone representations.
func TestScannerPhoneEscapedJSON(t *testing.T) {
	s := &scanner{}
	raw := `{"home":"\u002b390236618300"}`
	got, err := s.normalize("phone", types.Map(types.String().AsPhone()), raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]any{"home": "+390236618300"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
