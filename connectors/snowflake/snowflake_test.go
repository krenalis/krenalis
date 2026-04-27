// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/test/snowflaketester"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

const settingsEnvKey = "KRENALIS_TEST_PATH_SNOWFLAKE"

func Test_Columns(t *testing.T) {

	// Create a test database on Snowflake.
	testDB, err := snowflaketester.CreateTestDatabase()
	if err != nil {
		panic(err)
	}
	defer func() {
		err := testDB.Teardown()
		if err != nil {
			t.Logf("cannot teardown test Snowflake database: %s", err)
		}
	}()

	// Open the Snowflake connector.
	env := connectors.DatabaseEnv{Settings: newTestSettingsStore(testDB.Settings().JSON())}
	connector, err := New(&env)
	if err != nil {
		t.Fatalf("cannot open the database from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer func() {
		if err := connector.Close(); err != nil {
			t.Fatalf("unexpected error closing the database: %s", err)
		}
	}()
	if err = connector.openDB(context.Background()); err != nil {
		t.Fatalf("cannot open the database: %s", err)
	}

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
	_, err = connector.db.ExecContext(context.Background(), create)
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = connector.db.ExecContext(context.Background(), `DROP TABLE "`+tableName+`"`)
		if err != nil {
			t.Logf("cannot drop %s table: %s", tableName, err)
		}
	}()

	expected := []connectors.Column{
		{Name: "a", Type: types.Boolean(), Writable: true},
		{Name: "b", Type: types.Float(64), Nullable: true, Writable: true},
		{Name: "c", Type: types.Decimal(10, 3), Writable: true},
		{Name: "d", Type: types.DateTime(), Nullable: true, Writable: true},
		{Name: "e", Type: types.Date(), Writable: true},
		{Name: "f", Type: types.Time(), Nullable: true, Writable: true},
		{Name: "g", Type: types.JSON(), Nullable: true, Writable: true},
		{Name: "h", Type: types.String().WithMaxBytes(16_777_216).WithMaxLength(16_777_216), Writable: true},
		{Name: "i", Type: types.String().WithMaxLength(50), Nullable: true, Writable: true},
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
// Set the environment variable KRENALIS_TEST_PATH_SNOWFLAKE with the path to the
// database credentials in JSON format for running the test.
func Test_Merge_Query(t *testing.T) {

	cols := []struct {
		DriverType    string
		DriverValue   any
		KrenalisType  types.Type
		KrenalisValue any
	}{
		{"BOOLEAN", true, types.Boolean(), true},
		{"FLOAT", 703.219, types.Float(64), 703.219},
		{"NUMBER(4,2)", "12.67", types.Decimal(4, 2), decimal.MustParse("12.67")},
		{"TIMESTAMP_NTZ", time.Date(2024, 11, 7, 17, 29, 46, 320176551, time.UTC), types.DateTime(), time.Date(2024, 11, 7, 17, 29, 46, 320176551, time.UTC)},
		{"DATE", time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2024, 11, 7, 0, 0, 0, 0, time.UTC)},
		{"TIME", time.Date(1, 1, 1, 17, 29, 46, 320176551, time.UTC), types.Time(), time.Date(1970, 1, 1, 17, 29, 46, 320176551, time.UTC)},
		{"VARIANT", "{\n  \"foo\": \"boo\"\n}", types.JSON(), json.Value(`{"foo":"boo"}`)},
		{"VARCHAR", "foo", types.String().WithMaxBytes(16_777_216).WithMaxLength(16_777_216), "foo"},
		{"ARRAY", "[\n  {\n    \"foo\": \"boo\"\n  },\n  [\n    1,\n    2,\n    3\n  ]\n]", types.Array(types.JSON()), []any{json.Value(`{"foo":"boo"}`), json.Value(`[1, 2, 3]`)}},
	}

	table := connectors.Table{
		Name:    "test_krenalis_query",
		Columns: make([]connectors.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = connectors.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.KrenalisType,
			Nullable: true,
		}
	}

	// Create a test database on Snowflake.
	testDB, err := snowflaketester.CreateTestDatabase()
	if err != nil {
		panic(err)
	}
	defer func() {
		err := testDB.Teardown()
		if err != nil {
			t.Logf("cannot teardown test Snowflake database: %s", err)
		}
	}()

	// Open the Snowflake connector.
	env := connectors.DatabaseEnv{Settings: newTestSettingsStore(testDB.Settings().JSON())}
	connector, err := New(&env)
	if err != nil {
		t.Fatalf("cannot open the database from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer func() {
		if err := connector.Close(); err != nil {
			t.Fatalf("unexpected error closing the database: %s", err)
		}
	}()
	if err = connector.openDB(context.Background()); err != nil {
		t.Fatalf("cannot open the database: %s", err)
	}

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
	_, err = connector.db.ExecContext(context.Background(), create.String())
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
		row[i] = c.KrenalisValue
	}
	err = connector.Merge(context.Background(), table, [][]any{row})
	if err != nil {
		t.Fatalf("cannot merge: %s", err)
	}

	// Execute the query.
	var query strings.Builder
	query.WriteString(`SELECT "`)
	for i, c := range table.Columns {
		if i > 0 {
			query.WriteString(`", "`)
		}
		query.WriteString(c.Name)
	}
	query.WriteString(`" FROM "` + table.Name + `"`)
	rows, columns, err := connector.Query(context.Background(), query.String())
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

// func newSnowflakeFromENV(t *testing.T) *Snowflake {
// 	settingsFile, ok := os.LookupEnv(settingsEnvKey)
// 	if !ok {
// 		t.Skipf("the %s environment variable is not present", settingsEnvKey)
// 	}
// 	// Open connector.
// 	settings, err := os.ReadFile(settingsFile)
// 	if err != nil {
// 		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
// 	}
// 	env := connectors.DatabaseEnv{Settings: newTestSettingsStore(settings)}
// 	connector, err := New(&env)
// 	if err != nil {
// 		t.Fatalf("cannot open the database from settings in the %s environment variable: %s", settingsEnvKey, err)
// 	}
// 	if err = connector.openDB(context.Background()); err != nil {
// 		t.Fatalf("cannot open the database: %s", err)
// 	}
// 	return connector
// }

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

type testSettingsStore struct {
	settings json.Value
}

func newTestSettingsStore(settings json.Value) *testSettingsStore {
	return &testSettingsStore{settings: settings}
}

func (s *testSettingsStore) Load(ctx context.Context, dst any) error {
	return json.Unmarshal(s.settings, dst)
}

func (s *testSettingsStore) Store(ctx context.Context, src any) error {
	settings, err := json.Marshal(src)
	if err != nil {
		return err
	}
	s.settings = settings
	return nil
}
