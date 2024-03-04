//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package diffschemas

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"testing"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"
)

func TestDiff(t *testing.T) {

	tests := []struct {
		name        string
		fromSchema  types.Type
		toSchema    types.Type
		rePaths     map[string]any
		expectedOps []warehouses.AlterSchemaOperation
		expectedErr string
	}{
		{
			name: "No changes",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{},
		},
		{
			name: "Changes in labels are not influent",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Label: "old label"},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Label: "new label"},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{},
		},
		{
			name: "Changes in descriptions are not influent",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Description: "old description"},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Description: "new description"},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{},
		},
		{
			name: "One property added at first level (type Text)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "b", Type: types.Text()},
			},
		},
		{
			name: "One property added at first level (type Int(32), Nullable)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Int(32), Nullable: true},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "b", Type: types.Int(32), Nullable: true},
			},
		},
		{
			name: "One property added at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
				})},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "x.c", Type: types.Text()},
			},
		},
		{
			name: "One property dropped at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "x.c"},
			},
		},
		{
			name: "One property renamed at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "c", Type: types.Text()},
				})},
			}),
			rePaths: map[string]any{"x.c": "x.b"},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "x.b", Name: "c"},
			},
		},
		{
			name: "One property dropped and one created at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "c", Type: types.Text()},
				})},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "x.c", Type: types.Text()},
				{Operation: warehouses.OperationDropProperty, Path: "x.b"},
			},
		},
		{
			name: "Two properties added at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
				{Name: "c", Type: types.Text()},
				{Name: "d", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "c", Type: types.Text()},
				{Operation: warehouses.OperationAddProperty, Path: "d", Type: types.Text()},
			},
		},
		{
			name: "One property dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
				{Name: "c", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "c"},
			},
		},
		{
			name: "Two properties dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
				{Name: "c", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "b"},
				{Operation: warehouses.OperationDropProperty, Path: "c"},
			},
		},
		{
			name: "First level property type mismatch",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(64)},
			}),
			expectedErr: `error on property "a": type changes are not supported`,
		},
		{
			name: "One property added, one dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "b", Type: types.Text()},
				{Operation: warehouses.OperationDropProperty, Path: "a"},
			},
		},
		{
			name: "One property renamed at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Text()},
			}),
			rePaths: map[string]any{"b": "a"},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "a", Name: "b"},
			},
		},
		{
			name: "One property renamed at first level, and also its name is changed",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Int(32)},
			}),
			rePaths:     map[string]any{"b": "a"},
			expectedErr: `error on property "a" (renamed to "b"): type changes are not supported`,
		},
		{
			name: "One property renamed at first level, and also its nullability is changed",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(32), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Int(32), Nullable: false},
			}),
			rePaths:     map[string]any{"b": "a"},
			expectedErr: `error on property "a" (renamed to "b"): nullability changes are not supported`,
		},
		{
			name: "Two properties renamed at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "c", Type: types.Text()},
				{Name: "d", Type: types.Text()},
			}),
			rePaths: map[string]any{"c": "a", "d": "b"},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "a", Name: "c"},
				{Operation: warehouses.OperationRenameProperty, Path: "b", Name: "d"},
			},
		},
		{
			name: "One property removed and then added again (with another type) at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(32)},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "a"},
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Int(32)},
			},
		},
		{
			name: "One property removed and then added again (with the same type) at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "a"},
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Text()},
			},
		},
		{
			name: "One property removed and then added again (with another type) at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32)},
				})},
			}),
			rePaths: map[string]any{"x.a": nil},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "x.a"},
				{Operation: warehouses.OperationAddProperty, Path: "x.a", Type: types.Int(32)},
			},
		},
		{
			name: "Property order is changed at first level (total of two properties), no renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Text()},
				{Name: "a", Type: types.Text()},
			}),
			expectedErr: `properties order has changed (expected property "a", got "b")`,
		},
		{
			name: "Property order is changed at first level (total of three properties), no renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
				{Name: "c", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "c", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			expectedErr: `properties order has changed (expected property "b", got "c")`,
		},
		{
			name: "Property order is changed at first level, with renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
				{Name: "c", Type: types.Text()},
				{Name: "d", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b2", Type: types.Text()},
				{Name: "d", Type: types.Text()},
				{Name: "c", Type: types.Text()},
			}),
			rePaths:     map[string]any{"b2": "b"},
			expectedErr: `properties order has changed (expected property "c", got "d")`,
		},
		{
			name: "Property order is changed at second level, with renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b2", Type: types.Text()},
					{Name: "d", Type: types.Text()},
					{Name: "c", Type: types.Text()},
				})},
			}),
			rePaths:     map[string]any{"x.b2": "x.b"},
			expectedErr: `properties order has changed (expected property "c", got "d")`,
		},
		{
			name: "Changes in first level property nullability",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Nullable: false},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Text(), Nullable: true},
			}),
			expectedErr: `error on property "a": nullability changes are not supported`,
		},
		{
			name: "Changes in second level property nullability",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text(), Nullable: false},
				})},
			}),
			expectedErr: `error on property "x.a": nullability changes are not supported`,
		},
		{
			name: "Comprehensive test 1",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "z", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32)},
				})},
				{Name: "z2", Type: types.Text()},
			}),
			rePaths: map[string]any{
				"x.a": nil,
				"z2":  "z",
			},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "z", Name: "z2"},
				{Operation: warehouses.OperationDropProperty, Path: "x.a"},
				{Operation: warehouses.OperationAddProperty, Path: "x.a", Type: types.Int(32)},
			},
		},
		{
			name: "Comprehensive test 2",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
				})},
				{Name: "z", Type: types.Text()},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32)},
				})},
				{Name: "z2", Type: types.Text()},
			}),
			rePaths: map[string]any{
				"x.a": nil,
				"z2":  "z",
			},
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "z", Name: "z2"},
				{Operation: warehouses.OperationDropProperty, Path: "y.a"},
				{Operation: warehouses.OperationDropProperty, Path: "x.a"},
				{Operation: warehouses.OperationAddProperty, Path: "x.a", Type: types.Int(32)},
			},
		},
		{
			name: "Dropping of Object properties",
			fromSchema: types.Object([]types.Property{
				{Name: "v", Type: types.Text()},
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "v", Type: types.Text()},
			}),
			expectedOps: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "x.a"},
				{Operation: warehouses.OperationDropProperty, Path: "x.b"},
				{Operation: warehouses.OperationDropProperty, Path: "y.c"},
				{Operation: warehouses.OperationDropProperty, Path: "y.d"},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOps, gotErr := Diff(test.fromSchema, test.toSchema, test.rePaths, "")
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedErr {
				t.Fatalf("expected error %q, got %q", test.expectedErr, gotErrStr)
			}
			if !reflect.DeepEqual(gotOps, test.expectedOps) {
				expectedPath := dumpToJSONFile(test.expectedOps)
				gotPath := dumpToJSONFile(gotOps)
				t.Fatalf("operations mismatch. Expected operations have been dumped to %q, got operations to %q", expectedPath, gotPath)
			}
		})
	}

}

func Test_propertyPaths(t *testing.T) {

	tests := []struct {
		obj      types.Type
		expected []string
	}{
		{
			obj: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
			}),
			expected: []string{"a"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "a", Type: types.Text()},
				{Name: "b", Type: types.Text()},
			}),
			expected: []string{"a", "b"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
				})},
			}),
			expected: []string{"x.a", "x.b", "y.c", "y.d"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Text()},
				})},
				{Name: "x2", Type: types.Int(32)},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.Text()},
					{Name: "d", Type: types.Text()},
				})},
			}),
			expected: []string{"x.a", "x.b", "x2", "y.c", "y.d"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := propertyPaths(test.obj)
			if !slices.Equal(test.expected, got) {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}

}

func dumpToJSONFile(content any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "    ")
	err := enc.Encode(content)
	if err != nil {
		panic(err)
	}
	fi, err := os.CreateTemp("", "diffschemas_test_*.json")
	if err != nil {
		panic(err)
	}
	_, err = fi.Write(buf.Bytes())
	if err != nil {
		panic(err)
	}
	return fi.Name()
}
