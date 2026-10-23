// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"testing"

	"github.com/krenalis/krenalis/core/testconnector"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// TestColumnsWithSpecialCharacters verifies that schema and table names with
// special characters are handled literally when retrieving column metadata.
func TestColumnsWithSpecialCharacters(t *testing.T) {

	container, err := postgres.Run(t.Context(), testimages.PostgreSQL,
		postgres.WithDatabase(testDatabase), postgres.WithUsername(testUser), postgres.WithPassword(testPassword),
		postgres.BasicWaitStrategies())
	if err != nil {
		t.Fatalf("expected PostgreSQL container, got %v", err)
	}
	t.Cleanup(func() {
		err := testcontainers.TerminateContainer(container)
		if err != nil {
			t.Errorf("expected container cleanup, got %v", err)
		}
	})
	host, err := container.Host(t.Context())
	if err != nil {
		t.Fatalf("expected PostgreSQL host, got %v", err)
	}
	port, err := container.MappedPort(t.Context(), "5432/tcp")
	if err != nil {
		t.Fatalf("expected PostgreSQL port, got %v", err)
	}
	connector, err := testconnector.NewDatabase[*PostgreSQL]("postgresql", innerSettings{
		Host: host, Port: int(port.Num()), Database: testDatabase, Username: testUser, Password: testPassword,
		Schema: testSchema,
	})
	if err != nil {
		t.Fatalf("expected PostgreSQL connector, got %v", err)
	}
	t.Cleanup(func() {
		err := connector.Close()
		if err != nil {
			t.Errorf("expected connector cleanup, got %v", err)
		}
	})
	err = connector.openDB(t.Context())
	if err != nil {
		t.Fatalf("expected connector pool, got %v", err)
	}

	for _, test := range []struct {
		name   string
		schema string
		table  string
		column string
	}{
		{"plain identifiers", testSchema, "plain_table", "plain_value"},
		{"quoted table", testSchema, "quoted'table", "quoted_table_value"},
		{"quoted schema", "quoted'schema", "plain_table", "quoted_schema_value"},
		{"quoted schema and table", "quoted'schema", "quoted'table", "both_quoted_value"},
		{"SQL injection in schema", "schema' OR TRUE --", "plain_table", "injected_schema_value"},
		{"SQL injection in table", testSchema, "table')) OR TRUE --", "injected_table_value"},
		{"double quotes in identifiers", `quoted"schema`, `quoted"table`, "double_quoted_value"},
		{"case-sensitive identifiers", "MixedSchema", "MixedTable", "mixed_case_value"},
	} {
		t.Run(test.name, func(t *testing.T) {

			_, err := connector.pool.Exec(t.Context(), "CREATE SCHEMA IF NOT EXISTS "+quoteIdent(test.schema))
			if err != nil {
				t.Fatalf("expected fixture schema, got %v", err)
			}
			name := quoteIdent(test.schema) + "." + quoteIdent(test.table)
			_, err = connector.pool.Exec(t.Context(), "CREATE TABLE "+name+" ("+quoteIdent(test.column)+" varchar(7)[])")
			if err != nil {
				t.Fatalf("expected fixture table, got %v", err)
			}

			columns, err := connector.columns(t.Context(), test.schema, test.table)
			if err != nil {
				t.Fatalf("expected column metadata, got %v", err)
			}
			if len(columns) != 1 {
				t.Fatalf("expected one column, got %d: %+v", len(columns), columns)
			}
			column := columns[0]
			if column.Name != test.column || column.Issue != "" ||
				!types.Equal(column.Type, types.Array(types.String().WithMaxLength(7))) {
				t.Fatalf("expected column %q of type array(string(7)) without issues, got %+v", test.column, column)
			}

			missingTable := test.table + "_missing"
			columns, err = connector.columns(t.Context(), test.schema, missingTable)
			if err != nil {
				want := fmt.Sprintf("table %q does not exist", missingTable)
				if err.Error() != want {
					t.Fatalf("expected error %q, got %v", want, err)
				}
				return
			}

			t.Fatalf("expected missing table error, got columns %+v", columns)

		})
	}

}

func TestTypes(t *testing.T) {

	tests := []struct {
		s          string
		udtName    string
		t          types.Type
		charLength *string
		precision  *string
		radix      *string
		scale      *string
	}{
		{`smallint`, "", types.Int(16), nil, nil, nil, nil},
		{`integer`, "", types.Int(32), nil, nil, nil, nil},
		{`bigint`, "", types.Int(64), nil, nil, nil, nil},
		{`numeric`, "", types.Decimal(10, 3), nil, new("10"), new("10"), new("3")},
		{`real`, "", types.Float(32), nil, nil, nil, nil},
		{`double precision`, "", types.Float(64), nil, nil, nil, nil},
		{`character varying`, "", types.String().WithMaxLength(20), new("20"), nil, nil, nil},
		{`character`, "", types.String().WithMaxLength(8), new("8"), nil, nil, nil},
		{`text`, "", types.String(), nil, nil, nil, nil},
		{`timestamp without time zone`, "", types.DateTime(), nil, nil, nil, nil},
		{`timestamp with time zone`, "", types.DateTime(), nil, nil, nil, nil},
		{`date`, "", types.Date(), nil, nil, nil, nil},
		{`time without time zone`, "", types.Time(), nil, nil, nil, nil},
		{`boolean`, "", types.Boolean(), nil, nil, nil, nil},
		{`uuid`, "", types.UUID(), nil, nil, nil, nil},
		{`json`, "", types.JSON(), nil, nil, nil, nil},
		{`jsonb`, "", types.JSON(), nil, nil, nil, nil},
		{`ARRAY`, `_int4`, types.Array(types.Int(32)), nil, nil, nil, nil},
		{`ARRAY`, `_varchar`, types.Array(types.String()), nil, nil, nil, nil},
		{`ARRAY`, `_bool`, types.Array(types.Boolean()), nil, nil, nil, nil},
	}

	for _, test := range tests {
		row := pgTypeInfo{
			table:      "test_table",
			column:     "test_column",
			dataType:   test.s,
			udtName:    test.udtName,
			charLength: test.charLength,
			precision:  test.precision,
			radix:      test.radix,
			scale:      test.scale,
		}
		got, _, err := columnType(row, nil, nil)
		if err != nil {
			t.Error(err)
		}
		if got.Valid() != test.t.Valid() {
			if test.t.Valid() {
				t.Errorf("%s: expected a valid type, got an invalid type", test.s)
			}
			t.Errorf("%s: expected an invalid type, got a valid type: %#v", test.s, got)
		}
		if !types.Equal(got, test.t) {
			t.Errorf("%s: unexpected type: %#v", test.s, got)
		}
	}
}

func TestUnsupportedTypes(t *testing.T) {

	tests := []string{
		// Unsupported types.
		`money`, `bytea`, `point`, `line`, `lseg`, `box`, `path`, `polygon`, `circle`,

		// Invalid types.
		``, ` `, "\x00", "\xFF", `a`,
	}

	for i, test := range tests {
		t.Run(test, func(t *testing.T) {
			row := pgTypeInfo{
				dataType: test,
				column:   fmt.Sprintf("c%d", i),
			}
			got, issue, err := columnType(row, nil, nil)
			if err != nil {
				t.Fatalf("expected no error, got error '%v'", err)
			}
			expected := fmt.Sprintf("Column %q has an unsupported type %q.", row.column, row.dataType)
			if expected != issue {
				t.Fatalf("expected issue %q, got issue %q", expected, issue)
			}
			if got.Valid() {
				t.Fatalf("expected an invalid type, got %s", got)
			}
		})
	}
}
