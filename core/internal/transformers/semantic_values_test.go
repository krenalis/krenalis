// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestUnmarshalSemantics checks function results with scalar and nested semantic values in both supported languages.
func TestUnmarshalSemantics(t *testing.T) {

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
		{"canonical phone", types.String().AsPhone(), "+390236618300", true},
		{"structurally possible phone", types.String().AsPhone(), "+12001230101", true},
		{"local-only phone", types.String().AsPhone(), "+12530000", false},
		{"double plus phone", types.String().AsPhone(), "++390236618300", false},
		{"long phone", types.String().AsPhone(), "+1234567890123456", false},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", false},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", false},
		{"empty phone", types.String().AsPhone(), "", false},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", false},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.semantic
					data := strconv.Quote(test.value)
					switch shape {
					case "array":
						typ = types.Array(typ)
						data = "[" + data + "]"
					case "map":
						typ = types.Map(typ)
						data = "{\"home\":" + data + "}"
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						data = "[{\"home\":[" + data + "]}]"
					case "object":
						typ = types.Object([]types.Property{{Name: "inner", Type: typ}})
						data = "{\"inner\":" + data + "}"
					}
					schema := types.Object([]types.Property{{Name: "value", Type: typ}})
					for _, language := range []state.Language{state.JavaScript, state.Python} {

						t.Run(language.String(), func(t *testing.T) {

							records := make([]Record, 1)
							input := strings.NewReader("{\"records\":[{\"value\":{\"value\":" + data + "}}]}")
							err := Unmarshal(input, records, schema, language, false)
							if err != nil {
								t.Fatal(err)
							}
							if err := records[0].Err; err != nil {
								if test.valid {
									t.Fatal(err)
								}
								if _, ok := errors.AsType[RecordValidationError](err); !ok {
									t.Fatalf("expected RecordValidationError, got %T", err)
								}
								return
							}
							if !test.valid {
								t.Fatal("invalid value was accepted")
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
