//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

const settingsEnvKey = "MEERGO_TEST_PATH_POSTGRESQL"

// Test_Upsert_Query creates a table, inserts a row using the Upsert method with
// all supported PostgreSQL types, and retrieves the data using the Query
// method. It verifies that the returned columns and values match the expected
// results.
//
// Before running the test, set the environment variable
// MEERGO_TEST_PATH_POSTGRESQL with the path to the database credentials in JSON
// format.
func Test_Upsert_Query(t *testing.T) {

	cols := []struct {
		DriverType  string
		DriverValue any
		MeergoType  types.Type
		MeergoValue any
	}{
		{"SERIAL", int64(1), types.Int(32), 1},
		{"smallint", int64(1), types.Int(16), 1},
		{"integer", int64(1), types.Int(32), 1},
		{"bigint", int64(1), types.Int(64), 1},
		{"numeric(10,3)", "1.123", types.Decimal(10, 3), decimal.RequireFromString("1.123")},
		{"real", float64(float32(1.123)), types.Float(32), float64(float32(1.123))},
		{"double precision", 1.123, types.Float(64), 1.123},
		{"character varying", "foo", types.Text(), "foo"},
		{"character varying(3)", "foo", types.Text().WithCharLen(3), "foo"},
		{"text", "FOO", types.Text(), "FOO"},
		{"timestamp without time zone", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"timestamp with time zone", time.Date(2023, 1, 1, 1, 2, 3, 0, time.Local), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"date", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"time", "02:03:00", types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{"boolean", true, types.Boolean(), true},
		{"inet", "127.0.0.1", types.Inet(), "127.0.0.1"},
		{"uuid", "4d92d698-687d-4447-b34f-6b29d74a9730", types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{"jsonb", []byte(`{"foo": "boo"}`), types.JSON(), json.RawMessage(`{"foo":"boo"}`)},
	}

	table := meergo.Table{
		Name:    "test_meergo_query",
		Columns: make([]types.Property, len(cols)),
		Key:     "c0",
	}
	for i, c := range cols {
		table.Columns[i] = types.Property{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.MeergoType,
			Nullable: true,
		}
	}

	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	// Open connector.
	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}
	var config = meergo.DatabaseConfig{Settings: settings}
	connector, err := New(&config)
	if err != nil {
		t.Fatalf("cannot open the warehouse from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer connector.Close()
	err = connector.openDB()
	if err != nil {
		t.Fatalf("cannot open the database: %s", err)
	}

	// Create the table and add a row.
	create := bytes.NewBufferString("CREATE TABLE " + table.Name + " (\n\t")
	for i, c := range table.Columns {
		if i > 0 {
			create.WriteString(",\n\t")
		}
		create.WriteString(c.Name)
		create.WriteByte(' ')
		create.WriteString(cols[i].DriverType)
		if i == 0 {
			create.WriteString(" PRIMARY KEY")
		}
	}
	create.WriteString("\n)")
	_, err = connector.db.ExecContext(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err := connector.db.ExecContext(context.Background(), "DROP TABLE "+table.Name)
		if err != nil {
			t.Logf("cannot drop %s table: %s", table.Name, err)
		}
	}()
	row := map[string]any{}
	for i, c := range table.Columns {
		row[c.Name] = cols[i].MeergoValue
	}
	err = connector.Upsert(context.Background(), table, []map[string]any{row})
	if err != nil {
		t.Logf("cannot upsert: %s", err)
	}

	// Execute the query.
	query := "SELECT "
	for i, c := range table.Columns {
		if i > 0 {
			query += ", "
		}
		query += c.Name
	}
	query += " FROM " + table.Name
	rows, columns, err := connector.Query(context.Background(), query)
	if err != nil {
		t.Fatalf("query execution is failed: %s", err)
	}
	if !reflect.DeepEqual(table.Columns, columns) {
		t.Fatalf("expected columns %#v, got %#v", table.Columns, columns)
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
			if !reflect.DeepEqual(cols[i].DriverValue, v) {
				t.Fatalf("expected value %#v (type %T) for column %q, got %#v (type %T)", cols[i].DriverValue, cols[i].DriverValue, cols[i].DriverType, v, v)
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("cannot scan row: %s", err)
	}

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
