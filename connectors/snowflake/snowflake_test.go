//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package snowflake

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

const settingsEnvKey = "MEERGO_TEST_PATH_SNOWFLAKE"

func Test_Columns(t *testing.T) {

	// Open connector.
	connector := newSnowflakeFromENV(t)
	defer func() {
		if err := connector.Close(); err != nil {
			t.Fatalf("unexpected error closing the database: %s", err)
		}
	}()

	// Create the table
	tableName := "test_columns"
	create := `CREATE TABLE "` + tableName + `" (
		"a" BOOLEAN NOT NULL,
		"b" FLOAT NULL,
		"c" NUMBER(10,3) NOT NULL,
		"d" TIMESTAMP_NTZ NULL,
		"e" DATE NOT NULL,
		"f" TIME NULL,
		"g" VARIANT,
		"h" VARCHAR NOT NULL,
		"i" VARCHAR(50),
		"j" ARRAY NOT NULL
	)`
	_, err := connector.db.ExecContext(context.Background(), create)
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = connector.db.ExecContext(context.Background(), `DROP TABLE "`+tableName+`"`)
		if err != nil {
			t.Logf("cannot drop %s table: %s", tableName, err)
		}
	}()

	expected := []meergo.Column{
		{Name: "a", Type: types.Boolean(), Writable: true},
		{Name: "b", Type: types.Float(64), Nullable: true, Writable: true},
		{Name: "c", Type: types.Decimal(10, 3), Writable: true},
		{Name: "d", Type: types.DateTime(), Nullable: true, Writable: true},
		{Name: "e", Type: types.Date(), Writable: true},
		{Name: "f", Type: types.Time(), Nullable: true, Writable: true},
		{Name: "g", Type: types.JSON(), Nullable: true, Writable: true},
		{Name: "h", Type: types.Text().WithByteLen(16_777_216).WithCharLen(16_777_216), Writable: true},
		{Name: "i", Type: types.Text().WithCharLen(50), Nullable: true, Writable: true},
		{Name: "j", Type: types.Array(types.JSON()), Writable: true},
	}

	// Test the Columns method.
	got, err := connector.Columns(context.Background(), tableName)
	if err != nil {
		t.Fatalf("columns execution is failed: %s", err)
	}
	if len(expected) != len(got) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(got))
	}
	for i, c := range expected {
		if !reflect.DeepEqual(c, got[i]) {
			t.Fatalf("unexpected column:\n\nexpected: %#v\ngot:      %#v\n", c, got[i])
		}
	}

	// Test the 'columns' return parameter of the Query method.
	query := `SELECT "a", "b", "c", "d", "e", "f", "g", "h", "i", "j" FROM "` + tableName + `"`
	rows, got, err := connector.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query execution is failed: %s", err)
	}
	_ = rows.Close()
	if len(expected) != len(got) {
		t.Fatalf("expected %d columns, got %d", len(expected), len(got))
	}
	for i, c := range expected {
		if !reflect.DeepEqual(c, got[i]) {
			t.Fatalf("unexpected column:\n\nexpected: %#v\ngot:      %#v\n", c, got[i])
		}
	}

}

// Test_Merge_Query tests the Merge and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
//
// Set the environment variable MEERGO_TEST_PATH_SNOWFLAKE with the path to the
// database credentials in JSON format for running the test.
func Test_Merge_Query(t *testing.T) {

	cols := []struct {
		DriverType  string
		DriverValue any
		MeergoType  types.Type
		MeergoValue any
	}{
		{"BOOLEAN", true, types.Boolean(), true},
		{"FLOAT", 703.219, types.Float(64), 703.219},
		{"NUMBER(4,2)", "12.67", types.Decimal(4, 2), decimal.MustParse("12.67")},
		{"TIMESTAMP_NTZ", time.Date(2024, 11, 7, 17, 29, 46, 320176551, time.UTC), types.DateTime(), time.Date(2024, 11, 7, 17, 29, 46, 320176551, time.UTC)},
		{"DATE", time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC)},
		{"TIME", time.Date(1, 1, 1, 17, 29, 46, 320176551, time.UTC), types.Time(), time.Date(1970, 1, 1, 17, 29, 46, 320176551, time.UTC)},
		{"VARIANT", "{\n  \"foo\": \"boo\"\n}", types.JSON(), json.Value(`{"foo":"boo"}`)},
		{"VARCHAR", "foo", types.Text().WithByteLen(16_777_216).WithCharLen(16_777_216), "foo"},
		{"ARRAY", "[\n  {\n    \"foo\": \"boo\"\n  },\n  [\n    1,\n    2,\n    3\n  ]\n]", types.Array(types.JSON()), []any{json.Value(`{"foo":"boo"}`), json.Value(`[1, 2, 3]`)}},
	}

	table := meergo.Table{
		Name:    "test_meergo_query",
		Columns: make([]meergo.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = meergo.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.MeergoType,
			Nullable: true,
		}
	}

	// Open connector.
	connector := newSnowflakeFromENV(t)
	defer func() {
		if err := connector.Close(); err != nil {
			t.Fatalf("unexpected error closing the database: %s", err)
		}
	}()

	// Create the table and add a row.
	create := bytes.NewBufferString(`CREATE TABLE "` + table.Name + "\" (\n\t\"")
	for i, c := range table.Columns {
		if i > 0 {
			create.WriteString(",\n\t\"")
		}
		create.WriteString(c.Name)
		create.WriteString(`" `)
		create.WriteString(cols[i].DriverType)
	}
	create.WriteString("\n)")
	_, err := connector.db.ExecContext(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = connector.db.ExecContext(context.Background(), `DROP TABLE "`+table.Name+`"`)
		if err != nil {
			t.Logf("cannot drop %s table: %s", table.Name, err)
		}
	}()
	row := make([]any, len(cols))
	for i, c := range cols {
		row[i] = c.MeergoValue
	}
	err = connector.Merge(context.Background(), table, [][]any{row})
	if err != nil {
		t.Fatalf("cannot merge: %s", err)
	}

	// Execute the query.
	query := `SELECT "`
	for i, c := range table.Columns {
		if i > 0 {
			query += `", "`
		}
		query += c.Name
	}
	query += `" FROM "` + table.Name + `"`
	rows, columns, err := connector.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query execution is failed: %s", err)
	}
	if len(table.Columns) != len(columns) {
		t.Fatalf("expected %d columns, got %d", len(table.Columns), len(columns))
	}
	for i, c := range table.Columns {
		if !reflect.DeepEqual(c, columns[i]) {
			t.Fatalf("unexpected column:\nexpected: %#v\ngot:      %#v", c, columns[i])
		}
	}
	scanner := scanner{
		values: make([]any, len(columns)),
	}
	dest := make([]any, len(columns))
	for i := range columns {
		dest[i] = &scanner
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			_ = rows.Close()
			t.Fatalf("cannot scan row: %s", err)
		}
		for i, v := range scanner.values {
			if expected := cols[i].DriverValue; !reflect.DeepEqual(expected, v) {
				t.Fatalf("column %q: expected %v (%T), got %v (%T)", table.Columns[i].Name, expected, expected, v, v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot scan row: %s", err)
	}

}

func newSnowflakeFromENV(t *testing.T) *Snowflake {
	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}
	// Open connector.
	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}
	env := meergo.DatabaseEnv{Settings: settings}
	connector, err := New(&env)
	if err != nil {
		t.Fatalf("cannot open the database from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	if err = connector.openDB(); err != nil {
		t.Fatalf("cannot open the database: %s", err)
	}
	return connector
}

// scanner implements the sql.Scanner interface to read the database values.
type scanner struct {
	index  int
	values []any
}

func (sv *scanner) Scan(src any) error {
	sv.values[sv.index] = src
	sv.index++
	sv.index %= len(sv.values)
	return nil
}

func (sv *scanner) reset() {
	sv.index = 0
}
