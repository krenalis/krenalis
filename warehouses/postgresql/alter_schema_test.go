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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// Test_alterUserSchemaQueries checks that alterUserSchema generates the
// expected set of SQL statements for a variety of operations.
func Test_alterUserSchemaQueries(t *testing.T) {

	tests := []struct {
		name            string
		columns         []meergo.Column // without "__id__" and "__last_change_time__", which are added by the test
		ops             []meergo.AlterOperation
		expectedQueries []string // except the "DROP" and "CREATE VIEW" queries.
		expectedErr     error
	}{
		{
			name: "Add a first level text property",
			columns: []meergo.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" character varying",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" character varying",
			},
		},
		{
			name: "Add a first level Float64 property",
			columns: []meergo.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Add a first level Float64 (non-real) property",
			columns: []meergo.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Add a second level property",
			columns: []meergo.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "b", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "x_a", Type: types.Text()},
				{Operation: meergo.OperationAddColumn, Column: "x_b", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"x_a\" character varying,\n\tADD COLUMN \"x_b\" character varying",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"x_a\" character varying,\n\tADD COLUMN \"x_b\" character varying",
			},
		},
		{
			name: "Add a first level array property",
			columns: []meergo.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Array(types.Text()), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "a", Type: types.Array(types.Text())},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" character varying[]",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" character varying[]",
			},
		},
		{
			name: "Add a first level text property",
			columns: []meergo.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" character varying",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" character varying",
			},
		},
		{
			name: "Add a first level object property",
			columns: []meergo.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_b", Type: types.Int(32), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "x_a", Type: types.Text()},
				{Operation: meergo.OperationAddColumn, Column: "x_b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"x_a\" character varying,\n\tADD COLUMN \"x_b\" integer",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"x_a\" character varying,\n\tADD COLUMN \"x_b\" integer",
			},
		},
		{
			name: "Add two first level text properties",
			columns: []meergo.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "a", Type: types.Text()},
				{Operation: meergo.OperationAddColumn, Column: "b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tADD COLUMN \"a\" character varying,\n\tADD COLUMN \"b\" integer",
				"ALTER TABLE \"_user_identities\"\n\tADD COLUMN \"a\" character varying,\n\tADD COLUMN \"b\" integer",
			},
		},
		{
			name: "Drop a first level property",
			columns: []meergo.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationDropColumn, Column: "a"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tDROP COLUMN \"a\"",
				"ALTER TABLE \"_user_identities\"\n\tDROP COLUMN \"a\"",
			},
		},
		{
			name: "Drop two first level properties",
			columns: []meergo.Column{
				{Name: "z", Type: types.Int(32), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationDropColumn, Column: "a"},
				{Operation: meergo.OperationDropColumn, Column: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
				"ALTER TABLE \"_user_identities\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
			},
		},
		{
			name: "Rename a first level property",
			columns: []meergo.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationRenameColumn, Column: "a", NewColumn: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"\n\tRENAME COLUMN \"a\" TO \"b\"",
				"ALTER TABLE \"_user_identities\"\n\tRENAME COLUMN \"a\" TO \"b\"",
			},
		},
		{
			columns: []meergo.Column{
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
			ops: []meergo.AlterOperation{
				{Operation: meergo.OperationAddColumn, Column: "b", Type: types.Boolean()},
				{Operation: meergo.OperationAddColumn, Column: "i16", Type: types.Int(16)},
				{Operation: meergo.OperationAddColumn, Column: "i32", Type: types.Int(32)},
				{Operation: meergo.OperationAddColumn, Column: "i64", Type: types.Int(64)},
				{Operation: meergo.OperationAddColumn, Column: "f32", Type: types.Float(32)},
				{Operation: meergo.OperationAddColumn, Column: "f64", Type: types.Float(64)},
				{Operation: meergo.OperationAddColumn, Column: "dec", Type: types.Decimal(3, 1)},
				{Operation: meergo.OperationAddColumn, Column: "dt", Type: types.DateTime()},
				{Operation: meergo.OperationAddColumn, Column: "d", Type: types.Date()},
				{Operation: meergo.OperationAddColumn, Column: "t", Type: types.Time()},
				{Operation: meergo.OperationAddColumn, Column: "u", Type: types.UUID()},
				{Operation: meergo.OperationAddColumn, Column: "j", Type: types.JSON()},
				{Operation: meergo.OperationAddColumn, Column: "t", Type: types.Text()},
				{Operation: meergo.OperationAddColumn, Column: "at", Type: types.Array(types.Text())},
				{Operation: meergo.OperationAddColumn, Column: "ai32", Type: types.Array(types.Int(32))},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users_0\"" +
					"\n\tADD COLUMN \"b\" boolean," +
					"\n\tADD COLUMN \"i16\" smallint," +
					"\n\tADD COLUMN \"i32\" integer," +
					"\n\tADD COLUMN \"i64\" bigint," +
					"\n\tADD COLUMN \"f32\" real," +
					"\n\tADD COLUMN \"f64\" double precision," +
					"\n\tADD COLUMN \"dec\" numeric(3, 1)," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone," +
					"\n\tADD COLUMN \"d\" date," +
					"\n\tADD COLUMN \"t\" time without time zone," +
					"\n\tADD COLUMN \"u\" uuid," +
					"\n\tADD COLUMN \"j\" jsonb," +
					"\n\tADD COLUMN \"t\" character varying," +
					"\n\tADD COLUMN \"at\" character varying[]," +
					"\n\tADD COLUMN \"ai32\" integer[]",
				"ALTER TABLE \"_user_identities\"" +
					"\n\tADD COLUMN \"b\" boolean," +
					"\n\tADD COLUMN \"i16\" smallint," +
					"\n\tADD COLUMN \"i32\" integer," +
					"\n\tADD COLUMN \"i64\" bigint," +
					"\n\tADD COLUMN \"f32\" real," +
					"\n\tADD COLUMN \"f64\" double precision," +
					"\n\tADD COLUMN \"dec\" numeric(3, 1)," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone," +
					"\n\tADD COLUMN \"d\" date," +
					"\n\tADD COLUMN \"t\" time without time zone," +
					"\n\tADD COLUMN \"u\" uuid," +
					"\n\tADD COLUMN \"j\" jsonb," +
					"\n\tADD COLUMN \"t\" character varying," +
					"\n\tADD COLUMN \"at\" character varying[]," +
					"\n\tADD COLUMN \"ai32\" integer[]",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			columns := test.columns
			for _, c := range columns {
				if !c.Nullable {
					t.Fatalf("test %q is wrong: every column within 'columns' must be nullable, but column %q is not nullable", test.name, c.Name)
				}
			}
			columns = append([]meergo.Column{
				{Name: "__id__", Type: types.Int(32)},
				{Name: "__last_change_time__", Type: types.DateTime()},
			}, columns...)
			got := alterUserSchemaQueries("_users_0", columns, test.ops)
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

// Test_typeToPostgresType verifies the mapping between Meergo types and the
// PostgreSQL type strings returned by typeToPostgresType.
func Test_typeToPostgresType(t *testing.T) {

	tests := []struct {
		typ      types.Type
		expected string
	}{
		// boolean.
		{types.Boolean(), "boolean"},

		// int.
		{types.Int(8), "smallint"},
		{types.Int(16), "smallint"},
		{types.Int(16).WithIntRange(0, 10), "smallint"},
		{types.Int(24), "integer"},
		{types.Int(32), "integer"},
		{types.Int(64), "bigint"},
		{types.Int(64).WithIntRange(0, 10), "bigint"},

		// uint.
		{types.Uint(8), "smallint"},
		{types.Uint(16), "integer"},
		{types.Uint(16).WithUintRange(0, 10), "integer"},
		{types.Uint(24), "integer"},
		{types.Uint(32), "bigint"},
		{types.Uint(64), "numeric(20, 0)"},
		{types.Uint(64).WithUintRange(1, 200), "numeric(20, 0)"},

		// float.
		{types.Float(32), "real"},
		{types.Float(32).AsReal(), "real"},
		{types.Float(32).WithFloatRange(0, 100), "real"},
		{types.Float(64), "double precision"},
		{types.Float(64).AsReal(), "double precision"},
		{types.Float(64).WithFloatRange(0, 100), "double precision"},

		// decimal.
		{types.Decimal(10, 3), "numeric(10, 3)"},
		{types.Decimal(10, 3).WithDecimalRange(decimal.MustInt(0), decimal.MustInt(1000)), "numeric(10, 3)"},

		// datetime.
		{types.DateTime(), "timestamp without time zone"},

		// date.
		{types.Date(), "date"},

		// time.
		{types.Time(), "time without time zone"},

		// year.
		{types.Year(), "smallint"},

		// uuid.
		{types.UUID(), "uuid"},

		// json.
		{types.JSON(), "jsonb"},

		// inet.
		{types.Inet(), "inet"},

		// text.
		{types.Text(), "character varying"},
		{types.Text().WithByteLen(256), "character varying(256)"},
		{types.Text().WithCharLen(300), "character varying(300)"},
		{types.Text().WithByteLen(10).WithCharLen(10), "character varying(10)"},
		{types.Text().WithByteLen(5).WithCharLen(10), "character varying(5)"},
		{types.Text().WithByteLen(500).WithCharLen(10), "character varying(10)"},

		// array.
		{types.Array(types.Text()), "character varying[]"},
		{types.Array(types.Time()), "time without time zone[]"},
		{types.Array(types.Uint(32)), "bigint[]"},

		// map.
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
