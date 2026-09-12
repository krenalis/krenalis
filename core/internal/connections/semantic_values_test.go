// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestNormalizeSemantics checks semantic validation during normalization, including nested containers.
func TestNormalizeSemantics(t *testing.T) {

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

			for _, shape := range []string{"string", "bytes", "array", "map", "nested"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.semantic
					var value any = test.value
					switch shape {
					case "bytes":
						value = []byte(test.value)
					case "array":
						typ = types.Array(typ)
						value = []any{test.value}
					case "map":
						typ = types.Map(typ)
						value = map[string]any{"home": test.value}
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						value = []any{map[string]any{"home": []any{test.value}}}
					}
					schema := types.Object([]types.Property{{Name: "value", Type: typ}})
					p, _ := schema.Properties().ByName("value")
					got, err := normalize(p.Name, p.Type, value, p.Nullable, nil)
					if err != nil {
						if test.valid {
							t.Fatal(err)
						}
						if _, ok := errors.AsType[InputValidationError](err); !ok {
							t.Fatalf("expected InputValidationError, got %T", err)
						}
						return
					}
					if !test.valid {
						t.Fatal("invalid value was accepted")
					}
					if shape == "bytes" {
						value = test.value
					}
					if !reflect.DeepEqual(got, value) {
						t.Fatalf("value changed: got %#v, want %#v", got, value)
					}

				})

			}

			// An object's properties must supply their own semantics.
			schema := types.Object([]types.Property{{Name: "value", Type: test.semantic}})
			p := types.Property{Name: "object", Type: schema}
			_, err := normalize(p.Name, p.Type, map[string]any{"value": test.value}, p.Nullable, nil)
			if err != nil {
				if test.valid {
					t.Fatal(err)
				}
				return
			}
			if !test.valid {
				t.Fatal("invalid object property was accepted")
			}

		})

	}

}
