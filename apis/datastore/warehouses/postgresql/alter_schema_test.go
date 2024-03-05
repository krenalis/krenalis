// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b

package postgresql

import (
	"reflect"
	"testing"

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"
)

func Test_alterSchemaQueries(t *testing.T) {

	tests := []struct {
		name            string
		ops             []warehouses.AlterSchemaOperation
		expectedQueries []string
		expectedErr     string
	}{
		{
			name: "Add a first level not-nullable Text property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Text()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"a\" varchar NOT NULL DEFAULT ''",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"a\" varchar NOT NULL DEFAULT ''",
			},
		},
		{
			name: "Add a first level not-nullable Float64 property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"f\" double precision NOT NULL DEFAULT 0",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"f\" double precision NOT NULL DEFAULT 0",
			},
		},
		{
			name: "Add a first level not-nullable Float64 (non-real) property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"f\" double precision NOT NULL DEFAULT 0",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"f\" double precision NOT NULL DEFAULT 0",
			},
		},
		{
			name: "Float64 real properties are not supported",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the property \"f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Unsupported type at first-level property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the property \"f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Unsupported type at second-level property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "x.f", Type: types.Float(64).AsReal()},
			},
			expectedErr: "unsupported alter schema operation: the type of the property \"x.f\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Enum are not supported",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Text().WithValues("Happy", "Angry", "Sad")},
			},
			expectedErr: "unsupported alter schema operation: the type of the property \"a\" is not supported by the PostgreSQL driver",
		},
		{
			name: "Add a second level nullable property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text(), Nullable: false},
					{Name: "b", Type: types.Text(), Nullable: true},
				})},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"x_a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"x_b\" varchar",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"x_a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"x_b\" varchar",
			},
		},
		{
			name: "Add a first level not-nullable Array property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Array(types.Text())},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"a\" varchar[] NOT NULL DEFAULT '{}'",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"a\" varchar[] NOT NULL DEFAULT '{}'",
			},
		},
		{
			name: "Add a first level nullable Text property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Text(), Nullable: true},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"a\" varchar",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"a\" varchar",
			},
		},
		{
			name: "Add a first level Object property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "x", Type: types.Object([]types.Property{
					{Name: "a", Type: types.Text()},
					{Name: "b", Type: types.Int(32)},
				})},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"x_a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"x_b\" integer NOT NULL DEFAULT 0",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"x_a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"x_b\" integer NOT NULL DEFAULT 0",
			},
		},
		{
			name: "Add two first level Text properties",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "a", Type: types.Text()},
				{Operation: warehouses.OperationAddProperty, Path: "b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tADD COLUMN \"a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"b\" integer NOT NULL DEFAULT 0",
				"ALTER TABLE \"users_identities\"\n\tADD COLUMN \"a\" varchar NOT NULL DEFAULT '',\n\tADD COLUMN \"b\" integer NOT NULL DEFAULT 0",
			},
		},
		{
			name: "Drop a first level property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "a"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tDROP COLUMN \"a\"",
				"ALTER TABLE \"users_identities\"\n\tDROP COLUMN \"a\"",
			},
		},
		{
			name: "Drop two first level properties",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationDropProperty, Path: "a"},
				{Operation: warehouses.OperationDropProperty, Path: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
				"ALTER TABLE \"users_identities\"\n\tDROP COLUMN \"a\",\n\tDROP COLUMN \"b\"",
			},
		},
		{
			name: "Rename a first level property",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationRenameProperty, Path: "a", Name: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"\n\tRENAME COLUMN \"a\" TO \"b\"",
				"ALTER TABLE \"users_identities\"\n\tRENAME COLUMN \"a\" TO \"b\"",
			},
		},
		{
			name: "Test many types",
			ops: []warehouses.AlterSchemaOperation{
				{Operation: warehouses.OperationAddProperty, Path: "b", Type: types.Boolean()},
				{Operation: warehouses.OperationAddProperty, Path: "i16", Type: types.Int(16)},
				{Operation: warehouses.OperationAddProperty, Path: "i32", Type: types.Int(32)},
				{Operation: warehouses.OperationAddProperty, Path: "i64", Type: types.Int(64)},
				{Operation: warehouses.OperationAddProperty, Path: "f32", Type: types.Float(32)},
				{Operation: warehouses.OperationAddProperty, Path: "f64", Type: types.Float(64)},
				{Operation: warehouses.OperationAddProperty, Path: "dec", Type: types.Decimal(3, 1)},
				{Operation: warehouses.OperationAddProperty, Path: "dt", Type: types.DateTime()},
				{Operation: warehouses.OperationAddProperty, Path: "d", Type: types.Date()},
				{Operation: warehouses.OperationAddProperty, Path: "t", Type: types.Time()},
				{Operation: warehouses.OperationAddProperty, Path: "u", Type: types.UUID()},
				{Operation: warehouses.OperationAddProperty, Path: "j", Type: types.JSON()},
				{Operation: warehouses.OperationAddProperty, Path: "t", Type: types.Text()},
				{Operation: warehouses.OperationAddProperty, Path: "at", Type: types.Array(types.Text())},
				{Operation: warehouses.OperationAddProperty, Path: "ai32", Type: types.Array(types.Int(32))},
			},
			expectedQueries: []string{
				"ALTER TABLE \"users\"" +
					"\n\tADD COLUMN \"b\" boolean NOT NULL DEFAULT false," +
					"\n\tADD COLUMN \"i16\" smallint NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"i32\" integer NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"i64\" bigint NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"f32\" real NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"f64\" double precision NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"dec\" decimal(3, 1) NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone NOT NULL DEFAULT '0001-01-01 00:00:00'," +
					"\n\tADD COLUMN \"d\" date NOT NULL DEFAULT '0001-01-01'," +
					"\n\tADD COLUMN \"t\" time without time zone NOT NULL DEFAULT '00:00:00'," +
					"\n\tADD COLUMN \"u\" uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'," +
					"\n\tADD COLUMN \"j\" jsonb NOT NULL DEFAULT null," +
					"\n\tADD COLUMN \"t\" varchar NOT NULL DEFAULT ''," +
					"\n\tADD COLUMN \"at\" varchar[] NOT NULL DEFAULT '{}'," +
					"\n\tADD COLUMN \"ai32\" integer[] NOT NULL DEFAULT '{}'",
				"ALTER TABLE \"users_identities\"" +
					"\n\tADD COLUMN \"b\" boolean NOT NULL DEFAULT false," +
					"\n\tADD COLUMN \"i16\" smallint NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"i32\" integer NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"i64\" bigint NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"f32\" real NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"f64\" double precision NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"dec\" decimal(3, 1) NOT NULL DEFAULT 0," +
					"\n\tADD COLUMN \"dt\" timestamp without time zone NOT NULL DEFAULT '0001-01-01 00:00:00'," +
					"\n\tADD COLUMN \"d\" date NOT NULL DEFAULT '0001-01-01'," +
					"\n\tADD COLUMN \"t\" time without time zone NOT NULL DEFAULT '00:00:00'," +
					"\n\tADD COLUMN \"u\" uuid NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000'," +
					"\n\tADD COLUMN \"j\" jsonb NOT NULL DEFAULT null," +
					"\n\tADD COLUMN \"t\" varchar NOT NULL DEFAULT ''," +
					"\n\tADD COLUMN \"at\" varchar[] NOT NULL DEFAULT '{}'," +
					"\n\tADD COLUMN \"ai32\" integer[] NOT NULL DEFAULT '{}'",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotQueries, gotErr := alterSchemaQueries(test.ops)
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
