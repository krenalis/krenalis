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
				{Name: "first_name", Type: types.Text()},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
			}),
		},
		{
			name: "Nullable Object",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				}), Nullable: true},
			}),
			err: "user schema properties cannot be nullable",
		},
		{
			name: "Array with Object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
				{Name: "data", Type: types.Array(types.Object([]types.Property{
					{Name: "a", Type: types.Int(32)},
					{Name: "b", Type: types.Text()},
				}))},
			}),
			err: "user schema properties cannot have type 'Array(Object)'",
		},
		{
			name: "Property with a placeholder",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text()},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32), Placeholder: "1234"},
				})},
			}),
			err: "user schema properties cannot have a placeholder",
		},
		{
			name: "Meta properties",
			schema: types.Object([]types.Property{
				{Name: "__id__", Type: types.Text()},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				})},
			}),
			err: "user schema cannot have meta properties",
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
