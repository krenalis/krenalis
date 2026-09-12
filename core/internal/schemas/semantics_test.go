// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package schemas

import (
	"reflect"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateSemantics checks scalar and nested values without modifying them.
func TestValidateSemantics(t *testing.T) {

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

			for _, shape := range []string{"string", "array", "map", "nested", "object"} {

				t.Run(shape, func(t *testing.T) {

					typ := test.semantic
					var value any = test.value
					path := "value"
					switch shape {
					case "array":
						typ = types.Array(typ)
						value = []any{test.value}
						path += "[0]"
					case "map":
						typ = types.Map(typ)
						value = map[string]any{"home": test.value}
						path += "[\"home\"]"
					case "nested":
						typ = types.Array(types.Map(types.Array(typ)))
						value = []any{map[string]any{"home": []any{test.value}}}
						path += "[0][\"home\"][0]"
					case "object":
						typ = types.Object([]types.Property{{Name: "inner", Type: typ}})
						value = map[string]any{"inner": test.value}
						path += ".inner"
					}
					property := types.Property{Name: "value", Type: typ}
					err := ValidateSemantics(property, value)
					if err != nil {
						if test.valid {
							t.Fatal(err)
						}
						validationError, ok := errors.AsType[*SemanticValidationError](err)
						if !ok {
							t.Fatalf("expected SemanticValidationError, got %T", err)
						}
						if validationError.Path != path {
							t.Fatalf("got path %q, want %q", validationError.Path, path)
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

	schema := types.Object([]types.Property{
		{Name: "country", Type: country},
		{Name: "phone", Type: types.String().AsPhone()},
		{Name: "plain", Type: types.String()},
	})
	value := map[string]any{"country": nil, "plain": "ZZ"}
	expected := map[string]any{"country": nil, "plain": "ZZ"}
	err := ValidateSemantics(types.Property{Type: schema}, value)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(value, expected) {
		t.Fatalf("value was changed: %#v", value)
	}

}

// TestValidateSemanticsErrors checks invalid arguments and bounded error messages.
func TestValidateSemanticsErrors(t *testing.T) {

	tests := []struct {
		property types.Property
		value    any
	}{
		{types.Property{}, "IT"},
		{types.Property{Type: types.String().AsCountry(types.ISO3166Alpha2)}, true},
		{types.Property{Type: types.String().AsPhone()}, true},
		{types.Property{Type: types.String().AsPhone()}, "\xff"},
		{types.Property{Type: types.Array(types.String().AsPhone())}, "x"},
		{types.Property{Type: types.Map(types.String().AsPhone())}, "x"},
		{types.Property{Type: types.Object([]types.Property{{Name: "country", Type: types.String()}})}, "x"},
		{
			types.Property{Name: "countries", Type: types.Map(types.String().AsCountry(types.ISO3166Alpha2))},
			map[string]any{strings.Repeat("x", 10000): "ZZ"},
		},
	}

	for _, test := range tests {

		err := ValidateSemantics(test.property, test.value)
		if err != nil {
			if _, ok := errors.AsType[*SemanticValidationError](err); !ok {
				t.Fatalf("expected SemanticValidationError, got %T", err)
			}
			if len(err.Error()) > 256 {
				t.Fatalf("error message is too long: %d bytes", len(err.Error()))
			}
			continue
		}
		t.Fatal("invalid argument was accepted")

	}

}
