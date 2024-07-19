//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"testing"

	"github.com/meergo/meergo/types"
)

func Test_checkAllowedTypesUserSchema(t *testing.T) {

	tests := []struct {
		name   string
		schema types.Type
		err    string
	}{
		{
			name: "No errors",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
		},
		{
			name: "Nullable Object",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true, Nullable: true},
			}),
			err: "user schema properties cannot be nullable",
		},
		{
			name: "Array with Object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), ReadOptional: true},
					{Name: "b", Type: types.Text(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: "user schema properties cannot have type 'Array(Object)'",
		},
		{
			name: "Property with a placeholder",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), Placeholder: "1234"},
				}), ReadOptional: true},
			}),
			err: "user schema properties cannot have a placeholder",
		},
		{
			name: "Meta properties",
			schema: types.Object([]types.Property{
				{Name: "__id__", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "user schema cannot have meta properties",
		},
		{
			name: "Array with unique elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithUnique(), ReadOptional: true},
			}),
			err: "user schema properties with type Array cannot specify unique elements",
		},
		{
			name: "Arrays which specify a minimum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMinElements(1), ReadOptional: true},
			}),
			err: "user schema properties with type Array cannot specify minimum elements count",
		},
		{
			name: "Arrays which specify a maximum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMaxElements(types.MaxElements - 1), ReadOptional: true},
			}),
			err: "user schema properties with type Array cannot specify maximum elements count",
		},
		{
			name: "Map with Object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "data", Type: types.Map(types.Object([]types.Property{
					{Name: "a", Type: types.Text(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: "user schema properties cannot have type 'Map(Object)'",
		},
		{
			name: "Text with values",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text().WithValues("a", "b", "c"), ReadOptional: true},
					{Name: "street2", Type: types.Text(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "user schema properties with type Text cannot specify values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErr := checkAllowedPropertyUserSchema(test.schema)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.err {
				t.Fatalf("expected error %q, got %q", test.err, gotErrStr)
			}
		})
	}

}
