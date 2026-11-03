// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// BenchmarkMappingTransform measures time and allocations per transformed
// record. The Mapping, schemas, and input records are constructed outside the
// timer. The same source supports main and the source-preservation PR's API.
func BenchmarkMappingTransform(b *testing.B) {

	integers := make([]any, 1024)
	for i := range integers {
		integers[i] = i
	}
	phones := make([]any, 64)
	for i := range phones {
		phones[i] = "+390236618300"
	}
	object := types.Object([]types.Property{
		{Name: "name", Type: types.String(), CreateRequired: true},
		{Name: "value", Type: types.Int(32)},
	})
	optionalObject := types.Object([]types.Property{
		{Name: "name", Type: types.String(), CreateRequired: true, ReadOptional: true},
		{Name: "value", Type: types.Int(32)},
	})
	objects := make([]any, 128)
	for i := range objects {
		objects[i] = map[string]any{"name": "value", "value": i}
	}
	jsonValues := make([]any, 64)
	for i := range jsonValues {
		jsonValues[i] = json.Value("null")
	}
	tests := []struct {
		name   string
		source types.Type
		target types.Type
		value  any
		expr   string
	}{
		{"scalar", types.String(), types.String(), "value", "source"},
		{"integers", types.Array(types.Int(32)), types.Array(types.Int(32)), integers, "source"},
		{"phones", types.Array(types.String()), types.Array(types.String().AsPhone()), phones, "source"},
		{"objects", types.Array(object), types.Array(object), objects, "source"},
		{"objects_schema", types.Array(types.AsRole(object, types.Source)), types.Array(object), objects, "source"},
		{"objects_coalesce", types.Array(object), types.Array(object), objects, "coalesce(source, source)"},
		{"objects_if", types.Array(object), types.Array(object), objects, "if(true, source, source)"},
		{"objects_optional", types.Array(optionalObject), types.Array(optionalObject), objects, "source"},
		{"json_values", types.Array(types.JSON()), types.Array(types.JSON()), jsonValues, "source"},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {

			inSchema := types.Object([]types.Property{{Name: "source", Type: test.source}})
			outSchema := types.Object([]types.Property{{Name: "target", Type: test.target, CreateRequired: true}})
			expressions := map[string]string{"target": test.expr}
			var mapping *Mapping
			var err error
			switch constructor := any(New).(type) {
			case func(map[string]string, types.Type, types.Type) (*Mapping, error):
				mapping, err = constructor(expressions, inSchema, outSchema)
			case func(map[string]string, types.Type, types.Type, bool) (*Mapping, error):
				mapping, err = constructor(expressions, inSchema, outSchema, false)
			default:
				b.Fatalf("expected a supported Mapping constructor, got %T", constructor)
			}
			if err != nil {
				b.Fatalf("expected mapping creation to succeed, got %v", err)
			}
			attributes := map[string]any{"source": test.value}
			got, err := mapping.Transform(attributes, Create)
			if err != nil {
				b.Fatalf("expected transformation to succeed, got %v", err)
			}
			if !reflect.DeepEqual(got["target"], test.value) {
				b.Fatalf("expected unchanged value, got %#v", got)
			}

			b.ReportAllocs()
			for b.Loop() {
				_, err := mapping.Transform(attributes, Create)
				if err != nil {
					b.Fatalf("expected transformation to succeed, got %v", err)
				}
			}

		})
	}

}
