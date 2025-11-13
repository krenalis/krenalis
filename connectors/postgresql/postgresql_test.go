// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/test/testimages"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDatabase = "meergo"
	testUser     = "meergo"
	testPassword = "meergo"
	testSchema   = "public"
)

// Test_Merge_Query tests the Merge and Query methods on supported types. It
// creates a table, inserts a row, and retrieves the data, verifying that the
// returned columns and values match the expected results.
func Test_Merge_Query(t *testing.T) {

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
		{"numeric(10,3)", "1.123", types.Decimal(10, 3), decimal.MustParse("1.123")},
		{"real", float64(float32(1.123)), types.Float(32), float64(float32(1.123))},
		{"double precision", 1.123, types.Float(64), 1.123},
		{"char(3)", "foo", types.Text().WithCharLen(3), "foo"},
		{"character varying", "foo", types.Text(), "foo"},
		{"character varying(3)", "foo", types.Text().WithCharLen(3), "foo"},
		{"text", "FOO", types.Text(), "FOO"},
		//{"bytea", []byte("FOO"), types.Text(), "FOO"},
		{"timestamp without time zone", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"timestamp with time zone", time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{"date", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{"time", "02:03:00.000000", types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		//{"time with time zone", time.Date(1970, 1, 1, 2, 3, 0, 0, time.Local).Format("15:04:05Z07"), types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{"boolean", true, types.Boolean(), true},
		{"inet", "127.0.0.1/32", types.Inet(), "127.0.0.1/32"},
		{"uuid", "4d92d698-687d-4447-b34f-6b29d74a9730", types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{"json", `{"foo":"boo"}`, types.JSON(), json.Value(`{"foo":"boo"}`)},
		{"jsonb", `{"foo": "boo"}`, types.JSON(), json.Value(`{"foo":"boo"}`)},
	}

	table := connectors.Table{
		Name:    "test_meergo_query",
		Columns: make([]connectors.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = connectors.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.MeergoType,
			Nullable: true,
		}
	}

	// Run the PostgreSQL container.
	ctx := context.Background()
	postgresContainer, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(testDatabase),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Error(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(innerSettings{
		Host:     host,
		Port:     port.Int(),
		Username: testUser,
		Password: testPassword,
		Database: testDatabase,
		Schema:   testSchema,
	})
	if err != nil {
		t.Fatal(err)
	}

	env := connectors.DatabaseEnv{Settings: settings}
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
		if i == 0 {
			create.WriteString(" PRIMARY KEY")
		}
	}
	create.WriteString("\n)")
	_, err = connector.pool.Exec(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = connector.pool.Exec(context.Background(), "DROP TABLE "+table.Name)
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
			if t, ok := v.(time.Time); ok {
				v = t.UTC()
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

func Test_quoteJSON(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{`5`, `'5'`},
		{`""`, `'""'`},
		{`"abc"`, `'"abc"'`},
		{`"hello\u0020world"`, `'"hello\u0020world"'`},
		{`"'hello'\u0020'world'"`, `'"''hello''\u0020''world''"'`},
		{`"\u0000"`, `'""'`},
		{`"hello\u0000world"`, `'"helloworld"'`},
		{`"'hello'\u0000'world'"`, `'"''hello''''world''"'`},
		{`"hello\\u0000world"`, `'"hello\\u0000world"'`},
		{`"hello\\\u0000world"`, `'"hello\\world"'`},
		{`"hello\\\\u0000world"`, `'"hello\\\\u0000world"'`},
		{`"hello\\\\\u0000world"`, `'"hello\\\\world"'`},
		{`"hello\n\u0000world"`, `'"hello\nworld"'`},
		{`"\u0000world"`, `'"world"'`},
		{`"hello\u0000"`, `'"hello"'`},
		{`"hello\u0000world' \u0000' hello \\u0000 world"`, `'"helloworld'' '' hello \\u0000 world"'`},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var got strings.Builder
			quoteJSON(&got, []byte(test.s))
			if test.expected != got.String() {
				t.Fatalf("expected %q, got %q", test.expected, got.String())
			}
		})
	}
}
