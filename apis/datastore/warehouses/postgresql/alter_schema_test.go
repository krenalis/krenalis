// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b

package postgresql

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

func Test_alterSchemaQueries(t *testing.T) {

	tests := []struct {
		name            string
		userColumns     []warehouses.Column // without "__id__" and "__last_change_time__", which are added by the test
		ops             []warehouses.AlterSchemaOperation
		expectedQueries []string // except the "DROP" and "CREATE VIEW" queries.
		expectedErr     error
	}{
		{
			name: "Add a first level Text property",
			userColumns: []warehouses.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" varchar",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" varchar",
			},
		},
		{
			name: "Add a first level Float64 property",
			userColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Add a first level Float64 (non-real) property",
			userColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Add a second level property",
			userColumns: []warehouses.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "b", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" varchar",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" varchar",
			},
		},
		{
			name: "Add a first level Array property",
			userColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Array(types.Text()), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Array(types.Text())},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" varchar[]",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" varchar[]",
			},
		},
		{
			name: "Add a first level Text property",
			userColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" varchar",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" varchar",
			},
		},
		{
			name: "Add a first level Object property",
			userColumns: []warehouses.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" integer",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" integer",
			},
		},
		{
			name: "Add two first level Text properties",
			userColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" varchar,\n\tADD COLUMN \"b\" integer",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" varchar,\n\tADD COLUMN \"b\" integer",
			},
		},
		{
			name: "Drop a first level property",
			userColumns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tDROP COLUMN \"a\"",
				"ALTER TABLE \"_user_identities\"\n\tDROP COLUMN \"a\"",
			},
		},
		{
			name: "Drop two first level properties",
			userColumns: []warehouses.Column{
				{Name: "z", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationDropColumn, Column: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
				"ALTER TABLE \"_user_identities\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
			},
		},
		{
			name: "Rename a first level property",
			userColumns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tRENAME COLUMN \"a\" TO \"b\"",
				"ALTER TABLE \"_user_identities\"\n\tRENAME COLUMN \"a\" TO \"b\"",
			},
		},
		{
			userColumns: []warehouses.Column{
				{Name: "b", Type: types.Boolean(), Nullable: true},
				{Name: "i16", Type: types.Int(16), Nullable: true},
				{Name: "i32", Type: types.Int(32), Nullable: true},
				{Name: "i64", Type: types.Int(64), Nullable: true},
				{Name: "f32", Type: types.Float(32), Nullable: true},
				{Name: "f64", Type: types.Float(64), Nullable: true},
				{Name: "dec", Type: types.Decimal(3, 1), Nullable: true},
				{Name: "dt", Type: types.DateTime(), Nullable: true},
				{Name: "d", Type: types.Date(), Nullable: true},
				{Name: "t", Type: types.Time(), Nullable: true},
				{Name: "u", Type: types.UUID(), Nullable: true},
				{Name: "j", Type: types.JSON(), Nullable: true},
				{Name: "t", Type: types.Text(), Nullable: true},
				{Name: "at", Type: types.Array(types.Text()), Nullable: true},
				{Name: "ai32", Type: types.Array(types.Int(32)), Nullable: true},
			},
			name: "Test many types",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.Boolean()},
				{Operation: warehouses.OperationAddColumn, Column: "i16", Type: types.Int(16)},
				{Operation: warehouses.OperationAddColumn, Column: "i32", Type: types.Int(32)},
				{Operation: warehouses.OperationAddColumn, Column: "i64", Type: types.Int(64)},
				{Operation: warehouses.OperationAddColumn, Column: "f32", Type: types.Float(32)},
				{Operation: warehouses.OperationAddColumn, Column: "f64", Type: types.Float(64)},
				{Operation: warehouses.OperationAddColumn, Column: "dec", Type: types.Decimal(3, 1)},
				{Operation: warehouses.OperationAddColumn, Column: "dt", Type: types.DateTime()},
				{Operation: warehouses.OperationAddColumn, Column: "d", Type: types.Date()},
				{Operation: warehouses.OperationAddColumn, Column: "t", Type: types.Time()},
				{Operation: warehouses.OperationAddColumn, Column: "u", Type: types.UUID()},
				{Operation: warehouses.OperationAddColumn, Column: "j", Type: types.JSON()},
				{Operation: warehouses.OperationAddColumn, Column: "t", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "at", Type: types.Array(types.Text())},
				{Operation: warehouses.OperationAddColumn, Column: "ai32", Type: types.Array(types.Int(32))},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"" +
					"\n\tADD COLUMN \"b\" boolean," +
					"\n\tADD COLUMN \"i16\" smallint," +
					"\n\tADD COLUMN \"i32\" integer," +
					"\n\tADD COLUMN \"i64\" bigint," +
					"\n\tADD COLUMN \"f32\" real," +
					"\n\tADD COLUMN \"f64\" double precision," +
					"\n\tADD COLUMN \"dec\" decimal(3, 1)," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone," +
					"\n\tADD COLUMN \"d\" date," +
					"\n\tADD COLUMN \"t\" time without time zone," +
					"\n\tADD COLUMN \"u\" uuid," +
					"\n\tADD COLUMN \"j\" jsonb," +
					"\n\tADD COLUMN \"t\" varchar," +
					"\n\tADD COLUMN \"at\" varchar[]," +
					"\n\tADD COLUMN \"ai32\" integer[]",
				"ALTER TABLE \"_user_identities\"" +
					"\n\tADD COLUMN \"b\" boolean," +
					"\n\tADD COLUMN \"i16\" smallint," +
					"\n\tADD COLUMN \"i32\" integer," +
					"\n\tADD COLUMN \"i64\" bigint," +
					"\n\tADD COLUMN \"f32\" real," +
					"\n\tADD COLUMN \"f64\" double precision," +
					"\n\tADD COLUMN \"dec\" decimal(3, 1)," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone," +
					"\n\tADD COLUMN \"d\" date," +
					"\n\tADD COLUMN \"t\" time without time zone," +
					"\n\tADD COLUMN \"u\" uuid," +
					"\n\tADD COLUMN \"j\" jsonb," +
					"\n\tADD COLUMN \"t\" varchar," +
					"\n\tADD COLUMN \"at\" varchar[]," +
					"\n\tADD COLUMN \"ai32\" integer[]",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userColumns := test.userColumns
			for _, c := range userColumns {
				if !c.Nullable {
					t.Fatalf("test %q is wrong: every column within 'userColumns' must be nullable, but column %q is not nullable", test.name, c.Name)
				}
			}
			userColumns = append([]warehouses.Column{
				{Name: "__id__", Type: types.Int(32)},
				{Name: "__last_change_time__", Type: types.DateTime()},
			}, userColumns...)
			got := alterSchemaQueries("_users_0", userColumns, test.ops)
			// Exclude from the test the queries that drop or create views.
			got = slices.DeleteFunc(got, func(query string) bool {
				return strings.HasPrefix(query, "DROP VIEW") ||
					strings.HasPrefix(query, "CREATE VIEW")
			})
			if !reflect.DeepEqual(got, test.expectedQueries) {
				t.Fatalf("expected queries %#v, got %#v", test.expectedQueries, got)
			}
		})
	}

}

func Test_typeToPostgresType(t *testing.T) {

	tests := []struct {
		typ      types.Type
		expected string
	}{
		// Boolean.
		{types.Boolean(), "boolean"},

		// Int.
		{types.Int(8), "smallint"},
		{types.Int(16), "smallint"},
		{types.Int(16).WithIntRange(0, 10), "smallint"},
		{types.Int(24), "integer"},
		{types.Int(32), "integer"},
		{types.Int(64), "bigint"},
		{types.Int(64).WithIntRange(0, 10), "bigint"},

		// Uint.
		{types.Uint(8), "smallint"},
		{types.Uint(16), "integer"},
		{types.Uint(16).WithUintRange(0, 10), "integer"},
		{types.Uint(24), "integer"},
		{types.Uint(32), "bigint"},
		{types.Uint(64), "decimal(20, 0)"},
		{types.Uint(64).WithUintRange(1, 200), "decimal(20, 0)"},

		// Float.
		{types.Float(32), "real"},
		{types.Float(32).AsReal(), "real"},
		{types.Float(32).WithFloatRange(0, 100), "real"},
		{types.Float(64), "double precision"},
		{types.Float(64).AsReal(), "double precision"},
		{types.Float(64).WithFloatRange(0, 100), "double precision"},

		// Decimal.
		{types.Decimal(10, 3), "decimal(10, 3)"},
		{types.Decimal(10, 3).WithDecimalRange(decimal.MustInt(0), decimal.MustInt(1000)), "decimal(10, 3)"},

		// DateTime.
		{types.DateTime(), "timestamp without time zone"},

		// Date.
		{types.Date(), "date"},

		// Time.
		{types.Time(), "time without time zone"},

		// Year.
		{types.Year(), "smallint"},

		// UUID.
		{types.UUID(), "uuid"},

		// JSON.
		{types.JSON(), "jsonb"},

		// Inet.
		{types.Inet(), "inet"},

		// Text.
		{types.Text(), "varchar"},
		{types.Text().WithByteLen(256), "varchar(256)"},
		{types.Text().WithCharLen(300), "varchar(300)"},
		{types.Text().WithByteLen(10).WithCharLen(10), "varchar(10)"},
		{types.Text().WithByteLen(5).WithCharLen(10), "varchar(5)"},
		{types.Text().WithByteLen(500).WithCharLen(10), "varchar(10)"},

		// Array.
		{types.Array(types.Text()), "varchar[]"},
		{types.Array(types.Time()), "time without time zone[]"},
		{types.Array(types.Uint(32)), "bigint[]"},

		// Map.
		{types.Map(types.Text()), "jsonb"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.typ), func(t *testing.T) {
			gotType := typeToPostgresType(test.typ)
			if test.expected != gotType {
				t.Fatalf("expected %q to be returned, got %q instead", test.expected, gotType)
			}

		})
	}

}
