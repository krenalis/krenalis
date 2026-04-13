// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package clickhouse

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/clickhouse"
)

// Test_Merge_Query tests the Merge and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
func Test_Merge_Query(t *testing.T) {

	cols := []struct {
		DriverType    string
		DriverValue   any
		KrenalisType  types.Type
		KrenalisValue any
	}{
		{"Bool", true, types.Boolean(), true},
		{"Int8", int8(-23), types.Int(8), -23},
		{"Int16", int16(791), types.Int(16), 791},
		{"Int32", int32(51046253), types.Int(32), 51046253},
		{"Int64", int64(-530103712643), types.Int(64), -530103712643},
		{"UInt8", uint8(215), types.Int(8).Unsigned(), 215},
		{"UInt16", uint16(1057), types.Int(16).Unsigned(), 1057},
		{"UInt32", uint32(2029183764), types.Int(32).Unsigned(), 2029183764},
		{"UInt64", uint64(530103712643), types.Int(64).Unsigned(), 530103712643},
		{"Float32", float32(1.3561), types.Float(32), float64(float32(1.3561))},
		{"Float64", 390.491835234, types.Float(64), 390.491835234},
		{"Decimal(10,3)", decimal.MustParse("1.123"), types.Decimal(10, 3), decimal.MustParse("1.123")},
		{"DateTime('UTC')", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"DateTime64(9, 'UTC')", time.Date(2023, 1, 1, 1, 2, 3, 239016638, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 239016638, time.UTC)},
		{"Date", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"Date32", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"UUID", "4d92d698-687d-4447-b34f-6b29d74a9730", types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{"IPv4", net.ParseIP("127.0.0.1").To4(), types.IP(), "127.0.0.1"},
		{"IPv6", net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), types.IP(), "2001:0db8:85a3:0000:0000:8a2e:0370:7334"},
		{"String", "foo", types.String(), "foo"},
		{"LowCardinality(String)", "foo", types.String(), "foo"},
		{"FixedString(3)", "boo", types.String().WithMaxBytes(3), "boo"},
		{"Enum8('hello' = 1, 'world' = 2)", "hello", types.String().WithValues("hello", "world"), "hello"},
		{"Enum16('hello' = 1, 'world' = 2, 'clickhouse' = 3)", "clickhouse", types.String().WithValues("hello", "world", "clickhouse"), "clickhouse"},
		{"Enum16('hello' = 1, 'world' = 2, 'clickhouse' = 3)", "clickhouse", types.String().WithValues("hello", "world", "clickhouse"), "clickhouse"},
		{"Array(String)", []string{"boo", "foo"}, types.Array(types.String()), []any{"boo", "foo"}},
		{"Map(String,Int32)", map[string]int32{"a": 1, "b": 2}, types.Map(types.Int(32)), map[string]any{"a": 1, "b": 2}},
		{"Nullable(Float32)", float32(1.2), types.Float(32), float64(float32(1.2))},
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
			Nullable: strings.HasPrefix(c.DriverType, "Nullable("),
		}
	}

	// Run the Clickhouse container.
	const (
		username = "test_krenalis"
		password = "test_krenalis"
		database = "test_krenalis"
	)
	ctx := context.Background()
	clickHouseContainer, err := clickhouse.Run(ctx,
		testimages.ClickHouse,
		clickhouse.WithUsername(username),
		clickhouse.WithPassword(password),
		clickhouse.WithDatabase(database),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(clickHouseContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	testHost, err := clickHouseContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := clickHouseContainer.MappedPort(ctx, "9000/tcp")
	if err != nil {
		t.Fatal(err)
	}

	// Open connector.
	settings, err := json.Marshal(innerSettings{
		Host:     testHost,
		Port:     testPort.Int(),
		Username: username,
		Password: password,
		Database: database,
	})
	if err != nil {
		t.Fatal(err)
	}
	env := connectors.DatabaseEnv{Settings: newTestSettingsStore(settings)}
	connector, err := New(&env)
	if err != nil {
		t.Fatal(err)
	}
	defer connector.Close()
	if err = connector.openDB(context.Background()); err != nil {
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
	create.WriteString("\n)\nENGINE = MergeTree\nORDER BY (")
	for i, key := range table.Keys {
		if i > 0 {
			create.WriteByte(',')
		}
		create.WriteString(key)
	}
	create.WriteByte(')')
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
	row := make([]any, len(cols))
	for i, c := range cols {
		row[i] = c.KrenalisValue
	}
	err = connector.Merge(context.Background(), table, [][]any{row})
	if err != nil {
		t.Fatalf("cannot upsert: %s", err)
	}

	// Execute the query.
	var query strings.Builder
	query.WriteString("SELECT ")
	for i, c := range table.Columns {
		if i > 0 {
			query.WriteString(", ")
		}
		query.WriteString(c.Name)
	}
	query.WriteString(" FROM " + table.Name)
	rows, columns, err := connector.Query(context.Background(), query.String())
	if err != nil {
		t.Fatalf("query execution is failed: %s", err)
	}
	defer rows.Close()
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
			switch vt := v.(type) {
			case time.Time:
				switch table.Columns[i].Type.Kind() {
				case types.DateTimeKind:
					v = vt.UTC()
				case types.DateKind:
					v = time.Date(vt.Year(), vt.Month(), vt.Day(), 0, 0, 0, 0, time.UTC)
				}
			case fmt.Stringer:
				// Normalize the decimal type in the same way as the normalization does.
				// This avoids the explicit dependency on "github.com/shopspring/decimal" for the krenalis module.
				if typ := cols[i].KrenalisType; typ.Kind() == types.DecimalKind {
					v, err = decimal.Parse(vt.String(), typ.Precision(), typ.Scale())
					if err != nil {
						t.Fatalf("column %q: an error occurred parsing %v (%T) as decimal: %s", table.Columns[i].Name, v, v, err)
					}
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
