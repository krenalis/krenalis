// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package types_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestDecodeSemantics checks semantic constraints, including byte limits and nested containers.
func TestDecodeSemantics(t *testing.T) {

	country := types.String().AsCountry(types.ISO3166Alpha2)
	tests := []struct {
		name     string
		semantic types.Type
		value    string
		valid    bool
	}{
		{"current country", country, "IT", true},
		{"alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ITA", true},
		{"former alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ANT", true},
		{"unknown alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ZZZ", false},
		{"reserved alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "EUR", false},
		{"empty alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "", false},
		{"short alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "IT", false},
		{"lowercase alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "ita", false},
		{"mixed case alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "Ita", false},
		{"numeric alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "380", false},
		{"non-ASCII alpha-3 country", types.String().AsCountry(types.ISO3166Alpha3), "éA", false},
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
					input := strings.NewReader("{\"value\":" + data + "}")
					_, err := types.Decode[map[string]any](input, schema)
					if err != nil {
						if test.valid {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[*types.SchemaValidationError](err); !ok {
							t.Fatalf("expected SchemaValidationError, got %T", err)
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

}

// TestSemanticConstraintsRoundTrip checks that semantics do not introduce string constraints.
func TestSemanticConstraintsRoundTrip(t *testing.T) {

	typesToTest := []types.Type{
		types.String().AsCountry(types.ISO3166Alpha2),
		types.String().AsCountry(types.ISO3166Alpha3),
		types.String().AsPhone(),
	}
	for _, typ := range typesToTest {
		data, err := typ.MarshalJSON()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "maxBytes") || strings.Contains(string(data), "maxLength") {
			t.Fatalf("unexpected string constraints were serialized: %s", data)
		}
		var got types.Type
		err = got.UnmarshalJSON(data)
		if err != nil {
			t.Fatal(err)
		}
		if n, ok := got.MaxBytes(); ok || n != 0 {
			t.Fatalf("expected no maxBytes constraint, got %d and %t", n, ok)
		}
		if n, ok := got.MaxLength(); ok || n != 0 {
			t.Fatalf("expected no maxLength constraint, got %d and %t", n, ok)
		}
	}

}
