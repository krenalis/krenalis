// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"

	"github.com/jackc/pgx/v5/pgtype"
)

// TestPhoneColumnType checks phone storage without fixed-width padding and preserves country storage.
func TestPhoneColumnType(t *testing.T) {
	for _, test := range []struct {
		typ  types.Type
		want string
	}{
		{types.String().AsPhone(), "character varying(16)"},
		{types.Array(types.String().AsPhone()), "character varying(16)[]"},
		{types.String().AsCountry(types.ISO3166Alpha2), "character(2)"},
		{types.String().AsCountry(types.ISO3166Alpha3), "character(3)"},
	} {
		if got := typeToPostgresType(test.typ); got != test.want {
			t.Fatalf("expected %q, got %q", test.want, got)
		}
	}
}

// TestScannerPhone checks that warehouse phone values are canonical at every string leaf.
func TestScannerPhone(t *testing.T) {

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"canonical", "+390236618300", true},
		{"possible", "+12001230101", true},
		{"local only", "+12530000", false},
		{"formatted", "+39 02-36618 300", false},
		{"double plus", "++390236618300", false},
		{"padding", "+390236618300 ", false},
		{"unicode digits", "+３９０２３６６１８３００", false},
		{"national", "0236618300", false},
		{"empty", "", false},
	}
	for _, test := range tests {
		for _, shape := range []string{"string", "array", "map", "nested"} {
			t.Run(test.name+"/"+shape, func(t *testing.T) {

				typ := types.String().AsPhone()
				var raw any = test.input
				var want any = test.input
				data := strconv.Quote(test.input)
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
					raw = []byte(`{"home":` + data + `}`)
				case "nested":
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
				if !test.valid || !reflect.DeepEqual(got, want) {
					t.Fatalf("expected unchanged value %#v and valid=%t, got %#v and no error", want, test.valid, got)
				}

			})
		}
	}

}

// TestScannerPhoneEscapedJSON accepts equivalent JSON encodings, not alternative phone representations.
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
