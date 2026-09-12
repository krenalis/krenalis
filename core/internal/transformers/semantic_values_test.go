// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
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
		{"phone at limit", types.String().AsPhone(), "+123456789012345", true},
		{"long phone", types.String().AsPhone(), "+1234567890123456", false},
		{"multibyte phone at limit", types.String().AsPhone(), "éééééééé", true},
		{"multibyte phone over limit", types.String().AsPhone(), "ééééééééé", false},
		{"empty phone", types.String().AsPhone(), "", true},
		{"phone without format restriction", types.String().AsPhone(), "a (b)", true},
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
