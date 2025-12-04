// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package diffschemas

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

func TestDiff(t *testing.T) {

	tests := []struct {
		name        string
		fromSchema  types.Type
		toSchema    types.Type
		rePaths     map[string]any
		expectedOps []warehouses.AlterOperation
		expectedErr string
	}{
		{
			name: "First level property drop and rename",
			fromSchema: types.Object([]types.Property{
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{
				"lastName": "firstName",
			},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "lastName"},
				{Operation: warehouses.OperationRenameColumn, Column: "firstName", NewColumn: "lastName"},
			},
		},
		{
			name: "Second level property drop and rename",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "firstName", Type: types.String(), Nullable: true},
					{Name: "lastName", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "lastName", Type: types.String(), Nullable: true},
				})},
			}),
			rePaths: map[string]any{
				"x.lastName": "x.firstName",
			},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "x_lastName"},
				{Operation: warehouses.OperationRenameColumn, Column: "x_firstName", NewColumn: "x_lastName"},
			},
		},
		{
			name: "First level property drop and rename, but its type has changed",
			fromSchema: types.Object([]types.Property{
				{Name: "firstName", Type: types.String(), Nullable: true},
				{Name: "lastName", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "lastName", Type: types.String().WithMaxLength(10), Nullable: true},
			}),
			rePaths: map[string]any{
				"lastName": "firstName",
			},
			expectedErr: `error on property "lastName": type changes are not supported`,
		},
		{
			name: "No changes",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Changes in descriptions are not influent",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true, Description: "old description"},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true, Description: "new description"},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "One property added at first level (type string)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.String()},
			},
		},
		{
			name: "One property added at first level (type int(32), Nullable)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.Int(32), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.Int(32)},
			},
		},
		{
			name: "One property added at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_c", Type: types.String()},
			},
		},
		{
			name: "One property dropped at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "x_c"},
			},
		},
		{
			name: "One property renamed at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			rePaths: map[string]any{"x.c": "x.b"},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "x_b", NewColumn: "x_c"},
			},
		},
		{
			name: "One property dropped and one created at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_c", Type: types.String()},
				{Operation: warehouses.OperationDropColumn, Column: "x_b"},
			},
		},
		{
			name: "Two properties added at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
				{Name: "d", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "c", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "d", Type: types.String()},
			},
		},
		{
			name: "One property dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "c"},
			},
		},
		{
			name: "Two properties dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "b"},
				{Operation: warehouses.OperationDropColumn, Column: "c"},
			},
		},
		{
			name: "First level property type mismatch",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(64), Nullable: true},
			}),
			expectedErr: `error on property "a": type changes are not supported`,
		},
		{
			name: "One property added, one dropped at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.String()},
				{Operation: warehouses.OperationDropColumn, Column: "a"},
			},
		},
		{
			name: "One property renamed at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"b": "a"},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "b"},
			},
		},
		{
			name: "One property renamed at first level, and also its name is changed",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Int(32), Nullable: true},
			}),
			rePaths:     map[string]any{"b": "a"},
			expectedErr: `error on property "a" (renamed to "b"): type changes are not supported`,
		},
		{
			name: "Two properties renamed at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "c", Type: types.String(), Nullable: true},
				{Name: "d", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"c": "a", "d": "b"},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "c"},
				{Operation: warehouses.OperationRenameColumn, Column: "b", NewColumn: "d"},
			},
		},
		{
			name: "One property removed and then added again (with another type) at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(32), Nullable: true},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Int(32)},
			},
		},
		{
			name: "One property removed and then added again (with the same type) at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
			},
		},
		{
			name: "One property removed and then added again (with another type) at second level",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), Nullable: true},
				})},
			}),
			rePaths: map[string]any{"x.a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "x_a"},
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Int(32)},
			},
		},
		{
			name: "Property order is changed at first level (total of two properties), no renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Property order is changed at first level (total of three properties), no renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Property order is changed at first level, with renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
				{Name: "d", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b2", Type: types.String(), Nullable: true},
				{Name: "d", Type: types.String(), Nullable: true},
				{Name: "c", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"b2": "b"},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "b", NewColumn: "b2"},
			},
		},
		{
			name: "Property order is changed at second level, with renamings",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b2", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			rePaths: map[string]any{"x.b2": "x.b"},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "x_b", NewColumn: "x_b2"},
			},
		},
		{
			name: "Comprehensive test 1",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
				})},
				{Name: "z", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), Nullable: true},
				})},
				{Name: "z2", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{
				"x.a": nil,
				"z2":  "z",
			},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "z", NewColumn: "z2"},
				{Operation: warehouses.OperationDropColumn, Column: "x_a"},
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Int(32)},
			},
		},
		{
			name: "Comprehensive test 2",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
				})},
				{Name: "z", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(32), Nullable: true},
				})},
				{Name: "z2", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{
				"x.a": nil,
				"z2":  "z",
			},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "z", NewColumn: "z2"},
				{Operation: warehouses.OperationDropColumn, Column: "y_a"},
				{Operation: warehouses.OperationDropColumn, Column: "x_a"},
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Int(32)},
			},
		},
		{
			name: "Dropping of object properties",
			fromSchema: types.Object([]types.Property{
				{Name: "v", Type: types.String(), Nullable: true},
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "v", Type: types.String(), Nullable: true},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "x_a"},
				{Operation: warehouses.OperationDropColumn, Column: "x_b"},
				{Operation: warehouses.OperationDropColumn, Column: "y_c"},
				{Operation: warehouses.OperationDropColumn, Column: "y_d"},
			},
		},
		{
			name: "One non-nullable object added at first level",
			fromSchema: types.Object([]types.Property{
				{Name: "c", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "c", Type: types.String(), Nullable: true},
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.String()},
			},
		},
		{
			name: "https://github.com/meergo/meergo/issues/693 (1)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
				{Name: "e", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "b", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
				{Name: "e", Type: types.String(), Nullable: true},
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
			},
		},
		{
			name: "https://github.com/meergo/meergo/issues/693 (2)",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "e", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "e", Type: types.String(), Nullable: true},
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{"a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
			},
		},
		{
			name: "Renaming of an object property",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
					{Name: "e", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String(), Nullable: true},
						{Name: "g", Type: types.Object([]types.Property{
							{Name: "h", Type: types.String(), Nullable: true},
							{Name: "i", Type: types.String(), Nullable: true},
						})},
					})},
				})},
				{Name: "j", Type: types.String(), Nullable: true},
				{Name: "k", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
					{Name: "e_new_name", Type: types.Object([]types.Property{
						{Name: "f", Type: types.String(), Nullable: true},
						{Name: "g", Type: types.Object([]types.Property{
							{Name: "h", Type: types.String(), Nullable: true},
							{Name: "i", Type: types.String(), Nullable: true},
						})},
					})},
				})},
				{Name: "j", Type: types.String(), Nullable: true},
				{Name: "k", Type: types.String(), Nullable: true},
			}),
			rePaths: map[string]any{
				"b.e_new_name": "b.e",
			},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "b_e_f", NewColumn: "b_e_new_name_f"},
				{Operation: warehouses.OperationRenameColumn, Column: "b_e_g_h", NewColumn: "b_e_new_name_g_h"},
				{Operation: warehouses.OperationRenameColumn, Column: "b_e_g_i", NewColumn: "b_e_new_name_g_i"},
			},
		},
		{
			name: "Changing order of properties with type object",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String(), Nullable: true},
					{Name: "d", Type: types.String(), Nullable: true},
				})},
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Changing order of properties within objects",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "a", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Property renamed and added again with the same name, but different type",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(64), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "a2", Type: types.Int(64), Nullable: true},
			}),
			rePaths: map[string]any{"a2": "a", "a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "a2"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
			},
		},
		{
			name: "Property renamed and added again with the same name and same type (int(64))",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(64), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.Int(64), Nullable: true},
				{Name: "a2", Type: types.Int(64), Nullable: true},
			}),
			rePaths: map[string]any{"a2": "a", "a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "a2"},
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Int(64)},
			},
		},
		{
			name: "Property renamed and added again with the same name and same type (object)",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(64), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(64), Nullable: true},
				})},
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(64), Nullable: true},
				})}}),
			rePaths: map[string]any{"x2": "x", "x": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "x_a", NewColumn: "x2_a"},
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Int(64)},
			},
		},
		{
			name: "Property renamed and added again with the same name, but different type. Within an object property",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Int(64), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "a2", Type: types.Int(64), Nullable: true},
				})},
			}),
			rePaths: map[string]any{"x.a2": "x.a", "x.a": nil},
			expectedOps: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "x_a", NewColumn: "x_a2"},
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.String()},
			},
		},
		{
			name: "Rename an object property and create a new property with the same name (but different type)",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "a", Type: types.String(), Nullable: true},
				})},
			}),
			expectedOps: []warehouses.AlterOperation{},
		},
		{
			name: "Rename an object property while adding a new property to it",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
					{Name: "b", Type: types.String()},
				})},
			}),
			rePaths:     map[string]any{"x2": "x"},
			expectedErr: "it is not possible to rename an object property (\"x\", renamed to \"x2\") and simultaneously make changes to its descendant properties",
		},
		{
			name: "Rename an object property while deleting a property of it",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
					{Name: "b", Type: types.String()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
			}),
			rePaths:     map[string]any{"x2": "x"},
			expectedErr: "it is not possible to rename an object property (\"x\", renamed to \"x2\") and simultaneously make changes to its descendant properties",
		},
		{
			name: "Rename an object property while renaming a new property of it",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x2", Type: types.Object([]types.Property{
					{Name: "a2", Type: types.String()},
				})},
			}),
			rePaths:     map[string]any{"x2": "x", "x2.a2": "x.a"},
			expectedErr: "it is not possible to rename an object property (\"x\", renamed to \"x2\") and simultaneously make changes to its descendant properties",
		},
		{
			name: "One property added at first level (type string), but rePaths are invalid",
			fromSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
			}),
			rePaths:     map[string]any{"b": nil},
			expectedErr: "rePaths cannot contain {..., \"b\": null, ...}, as there are no properties named \"b\" in the old schema",
		},
		{
			name: "One property dropped at second level, but the rePaths are invalid",
			fromSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
					{Name: "c", Type: types.String(), Nullable: true},
				})},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String(), Nullable: true},
					{Name: "b", Type: types.String(), Nullable: true},
				})},
			}),
			rePaths:     map[string]any{"x.c": nil},
			expectedErr: "rePaths cannot contain \"x.c\", as this property no longer exists in the new schema",
		},
		{
			name: "Property renamed and added again with the same name and same type, but the rePaths are invalid",
			fromSchema: types.Object([]types.Property{
				{Name: "bar", Type: types.Int(64), Nullable: true},
			}),
			toSchema: types.Object([]types.Property{
				{Name: "foo", Type: types.Int(64), Nullable: true},
				{Name: "bar", Type: types.Int(64), Nullable: true},
			}),
			rePaths: map[string]any{
				"foo": "bar",
				// "a":  nil, -> commented on purpose: the test must verify that an error is returned when it is missing.
			},
			expectedErr: "property \"bar\" has been renamed and still appears in the new schema, so it means that it must be declared in rePaths (as a renamed property, or as a new property)",
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
				{Name: "a", Type: types.String()},
			}),
			expected: []string{"a"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "a", Type: types.String()},
				{Name: "b", Type: types.String()},
			}),
			expected: []string{"a", "b"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
					{Name: "b", Type: types.String()},
				})},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
				})},
			}),
			expected: []string{"x.a", "x.b", "y.c", "y.d"},
		},
		{
			obj: types.Object([]types.Property{
				{Name: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.String()},
					{Name: "b", Type: types.String()},
				})},
				{Name: "x2", Type: types.Int(32)},
				{Name: "y", Type: types.Object([]types.Property{
					{Name: "c", Type: types.String()},
					{Name: "d", Type: types.String()},
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
	fi, err := os.CreateTemp("", "diffschemas_test_*.json")
	if err != nil {
		panic(err)
	}
	defer fi.Close()
	var b json.Buffer
	err = b.EncodeIndent(content, "", "    ")
	if err != nil {
		panic(err)
	}
	_, err = b.WriteTo(fi)
	if err != nil {
		panic(err)
	}
	return fi.Name()
}
