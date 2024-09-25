//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

const settingsEnvKey = "MEERGO_TEST_PATH_CLICKHOUSE"

// Test_Upsert_Query tests the Upsert and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
//
// Set the environment variable MEERGO_TEST_PATH_CLICKHOUSE with the path to the
// database credentials in JSON format for running the test.
func Test_Upsert_Query(t *testing.T) {

	cols := []struct {
		DriverType  string
		DriverValue any
		MeergoType  types.Type
		MeergoValue any
	}{
		{"Bool", true, types.Boolean(), true},
		{"Int8", int8(-23), types.Int(8), -23},
		{"Int16", int16(791), types.Int(16), 791},
		{"Int32", int32(51046253), types.Int(32), 51046253},
		{"Int64", int64(-530103712643), types.Int(64), -530103712643},
		{"UInt8", uint8(215), types.Uint(8), 215},
		{"UInt16", uint16(1057), types.Uint(16), 1057},
		{"UInt32", uint32(2029183764), types.Uint(32), 2029183764},
		{"UInt64", uint64(530103712643), types.Uint(64), 530103712643},
		{"Float32", float32(1.3561), types.Float(32), float64(float32(1.3561))},
		{"Float64", 390.491835234, types.Float(64), 390.491835234},
		{"Decimal(10,3)", decimal.RequireFromString("1.123"), types.Decimal(10, 3), decimal.RequireFromString("1.123")},
		{"DateTime('UTC')", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"DateTime64(9, 'UTC')", time.Date(2023, 1, 1, 1, 2, 3, 239016638, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 239016638, time.UTC)},
		{"Date", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Date32", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"UUID", "4d92d698-687d-4447-b34f-6b29d74a9730", types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{"IPv4", net.ParseIP("127.0.0.1").To4(), types.Inet(), "127.0.0.1"},
		{"IPv6", net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), types.Inet(), "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"String", "foo", types.Text(), "foo"},
		{"LowCardinality(String)", "foo", types.Text(), "foo"},
		{"FixedString(3)", "boo", types.Text().WithByteLen(3), "boo"},
		{"Enum8('hello' = 1, 'world' = 2)", "hello", types.Text().WithValues("hello", "world"), "hello"},
		{"Enum16('hello' = 1, 'world' = 2, 'clickhouse' = 3)", "clickhouse", types.Text().WithValues("hello", "world", "clickhouse"), "clickhouse"},
		{"Enum16('hello' = 1, 'world' = 2, 'clickhouse' = 3)", "clickhouse", types.Text().WithValues("hello", "world", "clickhouse"), "clickhouse"},
		{"Array(String)", []string{"boo", "foo"}, types.Array(types.Text()), []any{"boo", "foo"}},
		{"Map(String,Int32)", map[string]int32{"a": 1, "b": 2}, types.Map(types.Int(32)), map[string]any{"a": 1, "b": 2}},
		{"Nullable(Float32)", float32(1.2), types.Float(32), float64(float32(1.2))},
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
			Nullable: strings.HasPrefix(c.DriverType, "Nullable("),
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
	if err = connector.openDB(); err != nil {
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
	}
	create.WriteString("\n)\nENGINE = MergeTree\nORDER BY " + table.Key)
	err = connector.db.Exec(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		err = connector.db.Exec(context.Background(), "DROP TABLE "+table.Name)
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
		t.Fatalf("cannot upsert: %s", err)
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
			if t, ok := v.(time.Time); ok {
				switch table.Columns[i].Type.Kind() {
				case types.DateTimeKind:
					v = t.UTC()
				case types.DateKind:
					v = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
				}
			}
			if expected := cols[i].DriverValue; !reflect.DeepEqual(expected, v) {
				t.Fatalf("column %q: expected %v (%T), got %v (%T)", table.Columns[i].Name, expected, expected, v, v)
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
