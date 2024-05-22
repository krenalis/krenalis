// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b

package postgresql

import (
	"reflect"
	"testing"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"
)

func Test_alterSchemaQueries(t *testing.T) {

	tests := []struct {
		name            string
		usersColumns    []warehouses.Column // without "__id__", which is added by the test
		ops             []warehouses.AlterSchemaOperation
		expectedQueries []string // except the "DROP" and "CREATE VIEW" queries.
		expectedErr     string
	}{
		{
			name: "Add a first level Text property",
			usersColumns: []warehouses.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"a\" varchar",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"a\" varchar",
			},
		},
		{
			name: "Add a first level Float64 property",
			usersColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Add a first level Float64 (non-real) property",
			usersColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"f\" double precision",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"f\" double precision",
			},
		},
		{
			name: "Float64 real properties are not supported",
			usersColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64).AsReal(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the column \"f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Unsupported type at first-level property",
			usersColumns: []warehouses.Column{
				{Name: "f", Type: types.Float(64).AsReal(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the column \"f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Unsupported type at second-level property",
			usersColumns: []warehouses.Column{
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_f", Type: types.Float(64).AsReal(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the column \"x_f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Enum are not supported",
			usersColumns: []warehouses.Column{
				{Name: "a", Type: types.Text().WithValues("Happy", "Angry", "Sad"), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text().WithValues("Happy", "Angry", "Sad")},
			},
			expectedErr: "unsupported alter schema operation: the type of the column \"a\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Add a second level property",
			usersColumns: []warehouses.Column{
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
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" varchar",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" varchar",
			},
		},
		{
			name: "Add a first level Array property",
			usersColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Array(types.Text()), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Array(types.Text())},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"a\" varchar[]",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"a\" varchar[]",
			},
		},
		{
			name: "Add a first level Text property",
			usersColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"a\" varchar",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"a\" varchar",
			},
		},
		{
			name: "Add a first level Object property",
			usersColumns: []warehouses.Column{
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "x_a", Type: types.Text(), Nullable: true},
				{Name: "x_b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" integer",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"x_a\" varchar,\n\tADD COLUMN \"x_b\" integer",
			},
		},
		{
			name: "Add two first level Text properties",
			usersColumns: []warehouses.Column{
				{Name: "z", Type: types.Text(), Nullable: true},
				{Name: "a", Type: types.Text(), Nullable: true},
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Text()},
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tADD COLUMN \"a\" varchar,\n\tADD COLUMN \"b\" integer",
				"ALTER TABLE \"_users_identities\"\n\tADD COLUMN \"a\" varchar,\n\tADD COLUMN \"b\" integer",
			},
		},
		{
			name: "Drop a first level property",
			usersColumns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tDROP COLUMN \"a\"",
				"ALTER TABLE \"_users_identities\"\n\tDROP COLUMN \"a\"",
			},
		},
		{
			name: "Drop two first level properties",
			usersColumns: []warehouses.Column{
				{Name: "z", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationDropColumn, Column: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
				"ALTER TABLE \"_users_identities\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
			},
		},
		{
			name: "Rename a first level property",
			usersColumns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"_users\"\n\tRENAME COLUMN \"a\" TO \"b\"",
				"ALTER TABLE \"_users_identities\"\n\tRENAME COLUMN \"a\" TO \"b\"",
			},
		},
		{
			usersColumns: []warehouses.Column{
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
				"ALTER TABLE \"_users\"" +
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
				"ALTER TABLE \"_users_identities\"" +
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
			usersColumns := test.usersColumns
			for _, c := range usersColumns {
				if !c.Nullable {
					t.Fatalf("test %q is wrong: every column within 'usersColumns' must be nullable, but column %q is not nullable", test.name, c.Name)
				}
			}
			usersColumns = append([]warehouses.Column{{Name: "__id__", Type: types.Int(32)}}, usersColumns...)
			gotQueries, gotErr := alterSchemaQueries(usersColumns, test.ops)
			if len(gotQueries) > 2 {
				gotQueries = gotQueries[2 : len(gotQueries)-2]
			}
			var gotErrStr string
			if gotErr != nil {
				gotErrStr = gotErr.Error()
			}
			if gotErrStr != test.expectedErr {
				t.Fatalf("expected error %q, got %q", test.expectedErr, gotErrStr)
			}
			if !reflect.DeepEqual(gotQueries, test.expectedQueries) {
				t.Fatalf("expected queries %#v, got %#v", test.expectedQueries, gotQueries)
			}
		})
	}

}
