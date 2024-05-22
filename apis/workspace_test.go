//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"testing"

	"github.com/open2b/chichi/types"
)

func Test_checkAllowedTypesUsersSchema(t *testing.T) {

	tests := []struct {
		name   string
		schema types.Type
		err    string
	}{
		{
			name: "No errors",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Nullable: true},
				})},
			}),
		},
		{
			name: "Nullable Object",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Nullable: true},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text()},
					{Name: "street2", Type: types.Text()},
					{Name: "number", Type: types.Int(32)},
				}), Nullable: true},
			}),
			err: "property with type Object cannot be nullable",
		},
		{
			name: "Array with Object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Nullable: true},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Nullable: true},
				})},
				{Name: "data", Type: types.Array(types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), Nullable: true},
					{Name: "b", Type: types.Text(), Nullable: true},
				})), Nullable: true},
			}),
			err: "property with type Array cannot have element with type Object",
		},
		{
			name: "Property with a placeholder",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.Text(), Nullable: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Nullable: true},
				})},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.Text(), Nullable: true},
					{Name: "street2", Type: types.Text(), Nullable: true},
					{Name: "number", Type: types.Int(32), Placeholder: "1234", Nullable: true},
				})},
			}),
			err: "property cannot specify a placeholder",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErr := checkAllowedTypesUsersSchema(test.schema)
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
