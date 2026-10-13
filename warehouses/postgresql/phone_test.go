// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestPhoneAndCountryColumnTypes checks the warehouse column types used for
// phone and country values.
func TestPhoneAndCountryColumnTypes(t *testing.T) {
	for _, test := range []struct {
		name string
		typ  types.Type
		want string
	}{
		{"phone", types.String().AsPhone(), "character varying(16)"},
		{"phone array", types.Array(types.String().AsPhone()), "character varying(16)[]"},
		{"country alpha2", types.String().AsCountry(types.ISO3166Alpha2), "character(2)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := typeToPostgresType(test.typ); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

// TestScannerPhone checks that phone values are canonical across
// supported container shapes.
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
				data, err := json.Marshal(test.input)
				if err != nil {
					t.Fatalf("expected value to marshal, got %v", err)
				}
				switch shape {
				case "array":
					typ = types.Array(typ)
					want = []any{test.input}
					var err error
					raw, err = pgtype.NewMap().Encode(pgtype.VarcharArrayOID, pgtype.BinaryFormatCode, []string{test.input}, nil)
					if err != nil {
						t.Fatalf("expected no error, got %v", err)
					}
				case "map":
					typ = types.Map(typ)
					want = map[string]any{"home": test.input}
					raw = []byte(`{"home":` + string(data) + `}`)
				case "array of maps":
					typ = types.Array(types.Map(typ))
					want = []any{map[string]any{"home": test.input}}
					var err error
					values := []map[string]any{{"home": test.input}}
					raw, err = pgtype.NewMap().Encode(pgtype.JSONBArrayOID, pgtype.BinaryFormatCode, values, nil)
					if err != nil {
						t.Fatalf("expected no error, got %v", err)
					}
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
	raw := []byte(`{"home":"\u002b390236618300"}`)
	got, err := s.normalize("phone", types.Map(types.String().AsPhone()), raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := map[string]any{"home": "+390236618300"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}
