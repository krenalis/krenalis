// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/datastore/diffschemas"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
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
				{Name: "amount", Type: types.Decimal(10, 2), ReadOptional: true, Semantic: types.Money()},
				{
					Name: "percentages",
					Type: types.Array(types.Decimal(18, 4).WithDecimalRange(
						decimal.MustParse("-1"), decimal.MustParse("2.5"),
					)),
					ReadOptional: true,
					Semantic:     types.Percentage(),
				},
				{
					Name: "measurements", Type: types.Map(types.Decimal(10, 2)), ReadOptional: true,
					Semantic: types.Measurement().WithUnitOfMeasure(types.Kilogram),
				},
				{Name: "duration", Type: types.Int(64), ReadOptional: true, Semantic: types.Duration(types.Second)},
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
		{
			name: "Formatted datetime semantic",
			schema: types.Object([]types.Property{
				{
					Name: "updated_at", Type: types.String(), ReadOptional: true,
					Semantic: types.FormattedDateTime("2006-01-02 15:04:05"),
				},
			}),
			err: "profile schema properties cannot have datetime semantic",
		},
		{
			name: "Money semantic on int",
			schema: types.Object([]types.Property{
				{Name: "amount", Type: types.Int(64), ReadOptional: true, Semantic: types.Money()},
			}),
			err: "profile schema properties with money semantic must have decimal values",
		},
		{
			name: "Percentage semantic with wrong decimal precision",
			schema: types.Object([]types.Property{
				{
					Name: "percentage", Type: types.Decimal(17, 4), ReadOptional: true,
					Semantic: types.Percentage(),
				},
			}),
			err: "profile schema properties with percentage semantic must have decimal(18,4) values",
		},
		{
			name: "Percentage semantic with wrong decimal scale",
			schema: types.Object([]types.Property{
				{
					Name: "percentage", Type: types.Decimal(18, 3), ReadOptional: true,
					Semantic: types.Percentage(),
				},
			}),
			err: "profile schema properties with percentage semantic must have decimal(18,4) values",
		},
		{
			name: "Measurement semantic on map of int",
			schema: types.Object([]types.Property{
				{
					Name: "measurements", Type: types.Map(types.Int(64)), ReadOptional: true,
					Semantic: types.Measurement(),
				},
			}),
			err: "profile schema properties with measurement semantic must have decimal values",
		},
		{
			name: "Duration semantic on decimal",
			schema: types.Object([]types.Property{
				{
					Name: "duration", Type: types.Decimal(10, 2), ReadOptional: true,
					Semantic: types.Duration(types.Second),
				},
			}),
			err: "profile schema properties with duration semantic must have signed int(64) values",
		},
		{
			name: "Duration semantic on int(32)",
			schema: types.Object([]types.Property{
				{Name: "duration", Type: types.Int(32), ReadOptional: true, Semantic: types.Duration(types.Second)},
			}),
			err: "profile schema properties with duration semantic must have signed int(64) values",
		},
		{
			name: "Duration semantic on unsigned int(64)",
			schema: types.Object([]types.Property{
				{
					Name: "duration", Type: types.Int(64).Unsigned(), ReadOptional: true,
					Semantic: types.Duration(types.Second),
				},
			}),
			err: "profile schema properties with duration semantic must have signed int(64) values",
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

// Test_profileSchemaChangeRequiresWarehouseDDL tests which profile schema
// changes require DDL on the data warehouse.
func Test_profileSchemaChangeRequiresWarehouseDDL(t *testing.T) {

	property := func(name string) types.Property {
		return types.Property{Name: name, Type: types.String(), ReadOptional: true}
	}

	tests := []struct {
		name      string
		oldSchema types.Type
		newSchema types.Type
		rePaths   map[string]any
		expected  bool
	}{
		{
			name:      "Identical schemas",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("a"), property("b")}),
		},
		{
			name: "Description changed",
			oldSchema: types.Object([]types.Property{
				property("a"),
			}),
			newSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), ReadOptional: true, Description: "New description"},
			}),
		},
		{
			name: "Nested map semantic changed",
			oldSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Map(types.Decimal(10, 2)), ReadOptional: true},
				}), ReadOptional: true},
			}),
			newSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{
						Name: "a", Type: types.Map(types.Decimal(10, 2)), ReadOptional: true,
						Semantic: types.Measurement().WithUnitOfMeasure(types.Kilogram),
					},
				}), ReadOptional: true},
			}),
		},
		{
			name:      "Property added",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("a"), property("b")}),
			expected:  true,
		},
		{
			name:      "Property removed",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("a")}),
			expected:  true,
		},
		{
			name:      "Property renamed",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("b")}),
			rePaths:   map[string]any{"b": "a"},
			expected:  true,
		},
		{
			name:      "Top-level properties reordered",
			oldSchema: types.Object([]types.Property{property("a"), property("b")}),
			newSchema: types.Object([]types.Property{property("b"), property("a")}),
			expected:  true,
		},
		{
			name: "Nested properties reordered",
			oldSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{property("a"), property("b")}), ReadOptional: true},
			}),
			newSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{property("b"), property("a")}), ReadOptional: true},
			}),
			expected: true,
		},
		{
			name:      "Explicit property recreation",
			oldSchema: types.Object([]types.Property{property("a")}),
			newSchema: types.Object([]types.Property{property("a")}),
			rePaths:   map[string]any{"a": nil},
			expected:  true,
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			// Keep the fixtures within the profile schema domain. Invalid
			// properties, including nullable ones, belong to validation tests.
			if err := checkAllowedPropertyProfileSchema(test.oldSchema); err != nil {
				t.Fatalf("old schema contains a property not allowed in a profile schema: %s", err)
			}
			if err := checkAllowedPropertyProfileSchema(test.newSchema); err != nil {
				t.Fatalf("new schema contains a property not allowed in a profile schema: %s", err)
			}

			operations, err := diffschemas.Diff(test.oldSchema, test.newSchema, test.rePaths, "")
			if err != nil {
				t.Fatal(err)
			}
			actual := profileSchemaChangeRequiresWarehouseDDL(test.oldSchema, test.newSchema, operations)
			if actual != test.expected {
				t.Fatalf("expected %t, got %t", test.expected, actual)
			}

		})

	}

}

func Test_validatePrimarySources(t *testing.T) {

	const validPrimarySourceID = "7B3mN9qK2xA4"

	schema := types.Object([]types.Property{
		{Name: "first_name", Type: types.String(), ReadOptional: true},
		{Name: "address", Type: types.Object([]types.Property{
			{Name: "street", Type: types.String(), ReadOptional: true},
		}), ReadOptional: true},
		{Name: "phone_numbers", Type: types.Array(types.String()), ReadOptional: true},
	})

	tests := []struct {
		primarySources map[string]string
		expectedErr    string
	}{
		{
			primarySources: nil,
		},
		{
			primarySources: map[string]string{},
		},
		{
			primarySources: map[string]string{"first_name": validPrimarySourceID},
		},
		{
			primarySources: map[string]string{"first_name": ""},
			expectedErr:    `primary source identifier "" is not valid`,
		},
		{
			primarySources: map[string]string{"first_name": "7B3mN9qK2xA"},
			expectedErr:    `primary source identifier "7B3mN9qK2xA" is not valid`,
		},
		{
			primarySources: map[string]string{"first_name": "7B3mN9qK2x0"},
			expectedErr:    `primary source identifier "7B3mN9qK2x0" is not valid`,
		},
		{
			primarySources: map[string]string{"address.street": validPrimarySourceID},
		},
		{
			primarySources: map[string]string{"first_name": validPrimarySourceID, "not_a_prop": "9zQ4Tn7B3mS6"},
			expectedErr:    "property path \"not_a_prop\" does not exist",
		},
		{
			primarySources: map[string]string{"address": validPrimarySourceID},
			expectedErr:    "primary sources cannot be specified for object properties",
		}, {
			primarySources: map[string]string{"phone_numbers": validPrimarySourceID},
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
