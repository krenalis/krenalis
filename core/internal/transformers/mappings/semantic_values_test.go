// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestMappingSemanticValidation checks country and phone semantic validation
// across mapping paths and ensures valid values are preserved.
func TestMappingSemanticValidation(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)

	tests := []struct {
		name  string
		typ   types.Type
		value string
		valid bool
	}{
		{"current country", country, "US", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "USA", true},
		{"long alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "USAA", false},
		{"former alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ANT", true},
		{"unknown alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ZZZ", false},
		{"reserved alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "EUR", false},
		{"lowercase alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "usa", false},
		{"former country", country, "AN", true},
		{"unknown country", country, "ZZ", false},
		{"reserved country", country, "UK", false},
		{"lowercase country", country, "us", false},
		{"empty country", country, "", false},
		{"short country", country, "I", false},
		{"long country", country, "USA", false},
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

					typ := test.typ
					var value any = test.value
					data := strconv.Quote(test.value)
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
					outSchema := types.Object([]types.Property{{Name: "target", Type: typ, CreateRequired: true}})
					for _, mode := range []string{"convert", "equal types", "json", "constant"} {

						if mode == "constant" && shape != "string" {
							continue
						}
						if mode == "equal types" && !test.valid {
							continue
						}

						t.Run(mode, func(t *testing.T) {

							inputType, inputValue := typ, value
							expr := "source"
							switch mode {
							case "convert":
								inputType = types.String()
								switch shape {
								case "array":
									inputType = types.Array(inputType)
								case "map":
									inputType = types.Map(inputType)
								case "nested":
									inputType = types.Array(types.Map(types.Array(inputType)))
								case "object":
									inputType = types.Object([]types.Property{{Name: "inner", Type: inputType}})
								}
							case "equal types":
								inputType = typ
							case "json":
								inputType, inputValue = types.JSON(), json.Value(data)
							case "constant":
								expr = strconv.Quote(test.value)
							}
							inSchema := types.Object([]types.Property{{Name: "source", Type: inputType}})
							mapping, err := New(map[string]string{"target": expr}, inSchema, outSchema, false)
							if err != nil {
								if test.valid {
									t.Fatalf("expected mapping creation to succeed, got %v", err)
								}
								if mode != "constant" {
									t.Fatalf("expected mapping creation to succeed for %s mode, got %v", mode, err)
								}
								if !strings.Contains(err.Error(), " is not convertible to the ") {
									t.Fatalf("expected constant conversion error, got %v", err)
								}
								return
							}
							got, err := mapping.Transform(map[string]any{"source": inputValue}, Create)
							if err != nil {
								if test.valid {
									t.Fatalf("expected transformation to succeed, got %v", err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("expected ValidationError, got %T: %v", err, err)
								}
								return
							}
							if !test.valid {
								t.Fatalf("expected invalid value to be rejected, got %#v", got["target"])
							}
							if !reflect.DeepEqual(got["target"], value) {
								t.Fatalf("expected value %#v, got %#v", value, got["target"])
							}

						})

					}

				})

			}

		})

	}

}
