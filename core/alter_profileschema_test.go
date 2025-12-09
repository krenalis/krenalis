// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/meergo/meergo/tools/types"
)

func Test_checkAllowedTypesProfileSchema(t *testing.T) {

	tests := []struct {
		name   string
		schema types.Type
		err    string
	}{
		{
			name: "No errors",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
		},
		{
			name: "Nullable object",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true, Nullable: true},
			}),
			err: "profile schema properties cannot be nullable",
		},
		{
			name: "Array with object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), ReadOptional: true},
					{Name: "b", Type: types.String(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: `profile schema properties cannot have type array(object)`,
		},
		{
			name: "Property with a prefilled value",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
				{Name: "billing_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), Prefilled: "1234"},
				}), ReadOptional: true},
			}),
			err: "profile schema properties cannot have a prefilled value",
		},
		{
			name: "Meta properties",
			schema: types.Object([]types.Property{
				{Name: "_id", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String(), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "profile schema cannot have meta properties",
		},
		{
			name: "Array with unique elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithUnique(), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify unique elements",
		},
		{
			name: "Arrays which specify a minimum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMinElements(1), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify minimum elements count",
		},
		{
			name: "Arrays which specify a maximum number of elements",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Array(types.Int(32)).WithMaxElements(types.MaxElements - 1), ReadOptional: true},
			}),
			err: "profile schema properties with type array cannot specify maximum elements count",
		},
		{
			name: "Map with object item",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "data", Type: types.Map(types.Object([]types.Property{
					{Name: "a", Type: types.String(), ReadOptional: true},
				})), ReadOptional: true},
			}),
			err: "profile schema properties cannot have type map(object)",
		},
		{
			name: "String with values",
			schema: types.Object([]types.Property{
				{Name: "first_name", Type: types.String(), ReadOptional: true},
				{Name: "shipping_address", Type: types.Object([]types.Property{
					{Name: "street1", Type: types.String().WithValues("a", "b", "c"), ReadOptional: true},
					{Name: "street2", Type: types.String(), ReadOptional: true},
					{Name: "number", Type: types.Int(32), ReadOptional: true},
				}), ReadOptional: true},
			}),
			err: "profile schema properties with type string cannot specify values",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotErr := checkAllowedPropertyProfileSchema(test.schema)
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

func Test_validatePrimarySources(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "first_name", Type: types.String(), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
		{Name: "phone_numbers", Type: types.Array(types.String()), ReadOptional: true},
	})

	tests := []struct {
		primarySources map[string]int
		expectedErr    string
	}{
		{
			primarySources: nil,
		},
		{
			primarySources: map[string]int{},
		},
		{
			primarySources: map[string]int{"first_name": 12345},
		},
		{
			primarySources: map[string]int{"first_name": 0},
			expectedErr:    "primary source identifier 0 is not valid",
		},
		{
			primarySources: map[string]int{"first_name": 2147483648},
			expectedErr:    "primary source identifier 2147483648 is not valid",
		},
		{
			primarySources: map[string]int{"address.street": 12345},
		},
		{
			primarySources: map[string]int{"first_name": 12345, "not_a_prop": 6789},
			expectedErr:    "property path \"not_a_prop\" does not exist",
		},
		{
			primarySources: map[string]int{"address": 12345},
			expectedErr:    "primary sources cannot be specified for object properties",
		}, {
			primarySources: map[string]int{"phone_numbers": 12345},
			expectedErr:    "primary sources cannot be specified for array(string) properties",
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotErr := validatePrimarySources(schema, test.primarySources)
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedErr {
				t.Fatalf("expected error %q, got %q instead", test.expectedErr, gotErrStr)
			}
		})
	}

}
