// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_alterProfileSchemaQueries checks that alterProfileSchema generates the
// expected set of SQL statements for a variety of operations.
func Test_alterProfileSchemaQueries(t *testing.T) {

	tests := []struct {
		name            string
		columns         []warehouses.Column // without "_kpid" and "_updated_at", which are added by the test
		ops             []warehouses.AlterOperation
		expectedQueries []string // except the "DROP" and "CREATE VIEW" queries.
	}{
		{
			name: "Add a first level text property",
			columns: []warehouses.Column{
				{Name: "a", Type: types.String(), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"A\" VARCHAR",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"A\" VARCHAR",
			},
		},
		{
			name: "Add a first level Float64 property",
			columns: []warehouses.Column{
				{Name: "f", Type: types.Float(64), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "f", Type: types.Float(64)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"F\" FLOAT",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"F\" FLOAT",
			},
		},
		{
			name: "Add a second level property",
			columns: []warehouses.Column{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.String(), Nullable: true},
				{Name: "x_a", Type: types.String(), Nullable: true},
				{Name: "x_b", Type: types.String(), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.String()},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"X_A\" VARCHAR,\n\t           \"X_B\" VARCHAR",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"X_A\" VARCHAR,\n\t           \"X_B\" VARCHAR",
			},
		},
		{
			name: "Add a first level array property",
			columns: []warehouses.Column{
				{Name: "z", Type: types.String(), Nullable: true},
				{Name: "a", Type: types.Array(types.String()), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.Array(types.String())},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"A\" ARRAY",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"A\" ARRAY",
			},
		},
		{
			name: "Add a first level object property",
			columns: []warehouses.Column{
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "x_a", Type: types.String(), Nullable: true},
				{Name: "x_b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "x_a", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "x_b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"X_A\" VARCHAR,\n\t           \"X_B\" INT",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"X_A\" VARCHAR,\n\t           \"X_B\" INT",
			},
		},
		{
			name: "Add two first level text properties",
			columns: []warehouses.Column{
				{Name: "z", Type: types.String(), Nullable: true},
				{Name: "a", Type: types.String(), Nullable: true},
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationAddColumn, Column: "a", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "b", Type: types.Int(32)},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tADD COLUMN \"A\" VARCHAR,\n\t           \"B\" INT",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tADD COLUMN \"A\" VARCHAR,\n\t           \"B\" INT",
			},
		},
		{
			name: "Drop a first level property",
			columns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tDROP COLUMN \"A\"",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tDROP COLUMN \"A\"",
			},
		},
		{
			name: "Drop two first level properties",
			columns: []warehouses.Column{
				{Name: "z", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationDropColumn, Column: "a"},
				{Operation: warehouses.OperationDropColumn, Column: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tDROP COLUMN \"A\",\n\tDROP COLUMN \"B\"",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tDROP COLUMN \"A\",\n\tDROP COLUMN \"B\"",
			},
		},
		{
			name: "Rename a first level property",
			columns: []warehouses.Column{
				{Name: "b", Type: types.Int(32), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
				{Operation: warehouses.OperationRenameColumn, Column: "a", NewColumn: "b"},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"\n\tRENAME COLUMN \"A\" TO \"B\"",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"\n\tRENAME COLUMN \"A\" TO \"B\"",
			},
		},
		{
			name: "Test many types",
			columns: []warehouses.Column{
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
				{Name: "s", Type: types.String(), Nullable: true},
				{Name: "at", Type: types.Array(types.String()), Nullable: true},
				{Name: "ai32", Type: types.Array(types.Int(32)), Nullable: true},
			},
			ops: []warehouses.AlterOperation{
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
				{Operation: warehouses.OperationAddColumn, Column: "s", Type: types.String()},
				{Operation: warehouses.OperationAddColumn, Column: "at", Type: types.Array(types.String())},
				{Operation: warehouses.OperationAddColumn, Column: "ai32", Type: types.Array(types.Int(32))},
			},
			expectedQueries: []string{
				"ALTER TABLE \"KRENALIS_PROFILES_0\"" +
					"\n\tADD COLUMN \"B\" BOOLEAN," +
					"\n\t           \"I16\" SMALLINT," +
					"\n\t           \"I32\" INT," +
					"\n\t           \"I64\" BIGINT," +
					"\n\t           \"F32\" FLOAT," +
					"\n\t           \"F64\" FLOAT," +
					"\n\t           \"DEC\" NUMBER(3,1)," +
					"\n\t           \"DT\" TIMESTAMP_NTZ," +
					"\n\t           \"D\" DATE," +
					"\n\t           \"T\" TIME," +
					"\n\t           \"U\" VARCHAR," +
					"\n\t           \"J\" VARIANT," +
					"\n\t           \"S\" VARCHAR," +
					"\n\t           \"AT\" ARRAY," +
					"\n\t           \"AI32\" ARRAY",
				"ALTER TABLE \"KRENALIS_IDENTITIES\"" +
					"\n\tADD COLUMN \"B\" BOOLEAN," +
					"\n\t           \"I16\" SMALLINT," +
					"\n\t           \"I32\" INT," +
					"\n\t           \"I64\" BIGINT," +
					"\n\t           \"F32\" FLOAT," +
					"\n\t           \"F64\" FLOAT," +
					"\n\t           \"DEC\" NUMBER(3,1)," +
					"\n\t           \"DT\" TIMESTAMP_NTZ," +
					"\n\t           \"D\" DATE," +
					"\n\t           \"T\" TIME," +
					"\n\t           \"U\" VARCHAR," +
					"\n\t           \"J\" VARIANT," +
					"\n\t           \"S\" VARCHAR," +
					"\n\t           \"AT\" ARRAY," +
					"\n\t           \"AI32\" ARRAY",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			columns := test.columns
			for _, c := range columns {
				if !c.Nullable {
					t.Fatalf("test %q is wrong: column %q within 'columns' must be nullable", test.name, c.Name)
				}
			}
			columns = append([]warehouses.Column{
				{Name: "_kpid", Type: types.Int(32)},
				{Name: "_updated_at", Type: types.DateTime()},
			}, columns...)
			got := alterProfileSchemaQueries("KRENALIS_PROFILES_0", columns, test.ops)
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

// Test_typeToSnowflakeType verifies the mapping between Krenalis types and the
// Snowflake type strings returned by typeToSnowflakeType.
func Test_typeToSnowflakeType(t *testing.T) {

	tests := []struct {
		typ      types.Type
		expected string
	}{
		// boolean.
		{types.Boolean(), "BOOLEAN"},

		// int.
		{types.Int(8), "TINYINT"},
		{types.Int(16), "SMALLINT"},
		{types.Int(16).WithIntRange(0, 10), "SMALLINT"},
		{types.Int(24), "NUMBER(7,0)"},
		{types.Int(32), "INT"},
		{types.Int(64), "BIGINT"},
		{types.Int(64).WithIntRange(0, 10), "BIGINT"},

		// uint.
		{types.Int(8).Unsigned(), "SMALLINT"},
		{types.Int(16).Unsigned(), "INT"},
		{types.Int(16).Unsigned().WithUnsignedRange(0, 10), "INT"},
		{types.Int(24).Unsigned(), "NUMBER(8,0)"},
		{types.Int(32).Unsigned(), "BIGINT"},
		{types.Int(64).Unsigned(), "NUMBER(20,0)"},
		{types.Int(64).Unsigned().WithUnsignedRange(1, 200), "NUMBER(20,0)"},

		// float.
		{types.Float(32), "FLOAT"},
		{types.Float(32).Real(), "FLOAT"},
		{types.Float(32).WithFloatRange(0, 100), "FLOAT"},
		{types.Float(64), "FLOAT"},
		{types.Float(64).Real(), "FLOAT"},
		{types.Float(64).WithFloatRange(0, 100), "FLOAT"},

		// decimal.
		{types.Decimal(10, 3), "NUMBER(10,3)"},
		{types.Decimal(10, 3).WithDecimalRange(decimal.MustInt(0), decimal.MustInt(1000)), "NUMBER(10,3)"},

		// datetime.
		{types.DateTime(), "TIMESTAMP_NTZ"},

		// date.
		{types.Date(), "DATE"},

		// time.
		{types.Time(), "TIME"},

		// year.
		{types.Year(), "NUMBER(4,0)"},

		// uuid.
		{types.UUID(), "VARCHAR"},

		// json.
		{types.JSON(), "VARIANT"},

		// ip.
		{types.IP(), "VARCHAR"},

		// string.
		{types.String(), "VARCHAR"},
		{types.String().WithMaxBytes(256), "VARCHAR(256)"},
		{types.String().WithMaxLength(300), "VARCHAR(300)"},
		{types.String().WithMaxBytes(10).WithMaxLength(10), "VARCHAR(10)"},
		{types.String().WithMaxBytes(5).WithMaxLength(10), "VARCHAR(5)"},
		{types.String().WithMaxBytes(500).WithMaxLength(10), "VARCHAR(10)"},
		{types.String().AsPhone(), "VARCHAR(16)"},
		{types.String().AsCountry(types.ISO3166Alpha2), "VARCHAR(2)"},

		// array.
		{types.Array(types.String()), "ARRAY"},
		{types.Array(types.String().AsPhone()), "ARRAY"},
		{types.Array(types.Time()), "ARRAY"},
		{types.Array(types.Int(32).Unsigned()), "ARRAY"},

		// map.
		{types.Map(types.String()), "VARIANT"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprint(test.typ), func(t *testing.T) {
			gotType := typeToSnowflakeType(test.typ)
			if test.expected != gotType {
				t.Fatalf("expected %q to be returned, got %q instead", test.expected, gotType)
			}
		})
	}

}
