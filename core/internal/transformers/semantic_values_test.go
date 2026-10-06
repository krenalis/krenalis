// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestUnmarshalSemanticValidation checks semantic validation and decoding
// of country and phone values across scalar and composite types in
// JavaScript and Python.
func TestUnmarshalSemanticValidation(t *testing.T) {

	alpha2Country := types.String().AsCountry(types.ISO3166Alpha2)
	alpha3Country := types.String().AsCountry(types.ISO3166Alpha3)
	phone := types.String().AsPhone()
	tests := []struct {
		name  string
		typ   types.Type
		value string
		valid bool
	}{
		{"current alpha-2 country", alpha2Country, "US", true},
		{"current alpha-3 country", alpha3Country, "USA", true},
		{"long alpha-3 country", alpha3Country, "USAA", false},
		{"former alpha-3 country", alpha3Country, "ANT", true},
		{"unknown alpha-3 country", alpha3Country, "ZZZ", false},
		{"reserved alpha-3 country", alpha3Country, "EUR", false},
		{"lowercase alpha-3 country", alpha3Country, "usa", false},
		{"former alpha-2 country", alpha2Country, "AN", true},
		{"unknown alpha-2 country", alpha2Country, "ZZ", false},
		{"reserved alpha-2 country", alpha2Country, "UK", false},
		{"lowercase alpha-2 country", alpha2Country, "us", false},
		{"empty alpha-2 country", alpha2Country, "", false},
		{"short alpha-2 country", alpha2Country, "I", false},
		{"long alpha-2 country", alpha2Country, "USA", false},
		{"non-ASCII alpha-2 country", alpha2Country, "é", false},
		{"canonical phone", phone, "+390236618300", true},
		{"structurally possible phone", phone, "+12001230101", true},
		{"local-only phone", phone, "+12530000", false},
		{"double plus phone", phone, "++390236618300", false},
		{"long phone", phone, "+1234567890123456", false},
		{"multibyte phone at limit", phone, "éééééééé", false},
		{"multibyte phone over limit", phone, "ééééééééé", false},
		{"empty phone", phone, "", false},
		{"non-phone text", phone, "a (b)", false},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			valueJSON, err := json.Marshal(test.value)
			if err != nil {
				t.Fatalf("expected value to be JSON-encodable, got %v", err)
			}

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.typ
					var value any = test.value
					data := string(valueJSON)
					switch shape {
					case "array":
						typ = types.Array(typ)
						value = []any{test.value}
						data = "[" + data + "]"
					case "map":
						typ = types.Map(typ)
						value = map[string]any{"home": test.value}
						data = "{\"home\":" + data + "}"
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						value = []any{map[string]any{"home": []any{test.value}}}
						data = "[{\"home\":[" + data + "]}]"
					case "object":
						typ = types.Object([]types.Property{{Name: "inner", Type: typ}})
						value = map[string]any{"inner": test.value}
						data = "{\"inner\":" + data + "}"
					}
					schema := types.Object([]types.Property{{Name: "value", Type: typ}})
					for _, language := range []state.Language{state.JavaScript, state.Python} {

						t.Run(language.String(), func(t *testing.T) {

							records := make([]Record, 1)
							input := strings.NewReader("{\"records\":[{\"value\":{\"value\":" + data + "}}]}")
							err := Unmarshal(input, records, schema, language, false)
							if err != nil {
								t.Fatalf("expected Unmarshal to succeed, got %v", err)
							}
							if err := records[0].Err; err != nil {
								if test.valid {
									t.Fatalf("expected no record error, got %v", err)
								}
								if _, ok := errors.AsType[RecordValidationError](err); !ok {
									t.Fatalf("expected RecordValidationError, got %T: %v", err, err)
								}
								return
							}
							if !test.valid {
								t.Fatalf("expected invalid value to be rejected, got attributes %#v", records[0].Attributes)
							}
							if !reflect.DeepEqual(records[0].Attributes["value"], value) {
								t.Fatalf("expected value %#v, got %#v", value, records[0].Attributes["value"])
							}

						})

					}

				})

			}

		})

	}

}

// TestUnmarshalPhone checks canonical output, nested string leaves and
// post-normalization uniqueness.
func TestUnmarshalPhone(t *testing.T) {

	phone := types.String().AsPhone()
	tests := []struct {
		typ  types.Type
		data string
		want any
	}{
		{types.Array(types.Map(phone)), `[{"home":"+39 02-36618 300"}]`, []any{map[string]any{"home": "+390236618300"}}},
		{
			types.Array(phone).WithMaxElements(2), `["+39 02-36618 300","+12001230101"]`,
			[]any{"+390236618300", "+12001230101"},
		},
		{types.Array(phone).WithUnique(), `["+390236618300","+39 02-36618 300"]`, nil},
		{types.Array(phone), `["+390236618300","+39 02-36618 300"]`, []any{"+390236618300", "+390236618300"}},
	}
	for _, test := range tests {
		schema := types.Object([]types.Property{{Name: "phone", Type: test.typ}})
		for _, language := range []state.Language{state.JavaScript, state.Python} {
			records := make([]Record, 1)
			input := strings.NewReader(`{"records":[{"value":{"phone":` + test.data + `}}]}`)
			err := Unmarshal(input, records, schema, language, false)
			if err != nil {
				t.Fatalf("expected no error for %s and %s, got %v", language, test.data, err)
			}
			if err := records[0].Err; err != nil {
				if test.want != nil {
					t.Fatalf("expected no record error, got %v", err)
				}
				continue
			}
			if test.want == nil || !reflect.DeepEqual(records[0].Attributes["phone"], test.want) {
				t.Fatalf("expected phone %#v, got %#v", test.want, records[0].Attributes["phone"])
			}
		}
	}

}
