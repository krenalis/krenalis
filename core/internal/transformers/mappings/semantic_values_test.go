// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestMappingSemantics checks semantic validation during conversion and preserves valid equal-type values.
func TestMappingSemantics(t *testing.T) {

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
					outSchema := types.Object([]types.Property{{Name: "target", Type: typ}})
					target, _ := outSchema.Properties().ByName("target")
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
								inputType = target.Type
							case "json":
								inputType, inputValue = types.JSON(), json.Value(data)
							case "constant":
								expr = "'" + test.value + "'"
							}
							inSchema := types.Object([]types.Property{{Name: "source", Type: inputType}})
							mapping, err := New(map[string]string{"target": expr}, inSchema, outSchema, false)
							if err != nil {
								if test.valid {
									t.Fatal(err)
								}
								return
							}
							got, err := mapping.Transform(map[string]any{"source": inputValue}, Create)
							if err != nil {
								if test.valid {
									t.Fatal(err)
								}
								if _, ok := errors.AsType[ValidationError](err); !ok {
									t.Fatalf("expected ValidationError, got %T", err)
								}
								return
							}
							if !test.valid {
								t.Fatal("invalid value was accepted")
							}
							if !reflect.DeepEqual(got["target"], value) {
								t.Fatalf("value changed: got %#v, want %#v", got["target"], value)
							}

						})

					}

				})

			}

		})

	}

}
