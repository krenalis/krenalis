// Copyright 2026 Open2b. All rights reserved.
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

	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"

	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDatabase = "krenalis"
	testUser     = "krenalis"
	testPassword = "krenalis"
)

func Test_Merge(t *testing.T) {

	cols := []struct {
		KrenalisType  types.Type
		KrenalisValue any
	}{
		{types.String(), "foo"},
		{types.Boolean(), true},
		{types.Int(8), 103},
		{types.Int(16), 8030},
		{types.Int(24), -3582672},
		{types.Int(32), 1023947264},
		{types.Int(64), -603826591193},
		{types.Int(8).Unsigned(), uint(249)},
		{types.Int(16).Unsigned(), uint(22941)},
		{types.Int(24).Unsigned(), uint(1300928)},
		{types.Int(32).Unsigned(), uint(3281905844)},
		{types.Int(64).Unsigned(), uint(1003883597101)},
		{types.Float(32), float64(float32(1.123))},
		{types.Float(64), 1.123},
		{types.Decimal(10, 3), decimal.MustParse("1.123")},
		{types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{types.Year(), 2014},
		{types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{types.JSON(), json.Value(`{"foo":"boo"}`)},
		{types.IP(), "127.0.0.1"},
		{types.Array(types.Boolean()), []any{true, false}},
		{types.Array(types.Int(8)), []any{5, -2, 12}},
		{types.Array(types.Int(16)), []any{32057, -9381, 1623}},
		{types.Array(types.Int(24)), []any{6318609, -93810, 16423}},
		{types.Array(types.Int(32)), []any{7936605, -179804772, 23}},
		{types.Array(types.Int(64)), []any{-193874627541, 819, 3481674621874}},
		{types.Array(types.Int(8).Unsigned()), []any{uint(223), uint(66), uint(130)}},
		{types.Array(types.Int(16).Unsigned()), []any{uint(65535), uint(840), uint(12)}},
		{types.Array(types.Int(24).Unsigned()), []any{uint(16570147), uint(193810), uint(942754)}},
		{types.Array(types.Int(32).Unsigned()), []any{uint(4164303781), uint(8400), uint(13)}},
		{types.Array(types.Int(64).Unsigned()), []any{uint(18446744073709551615), uint(8400), uint(13)}},
		{types.Array(types.Float(32)), []any{float64(float32(2.64)), float64(float32(1.212))}},
		{types.Array(types.Float(64)), []any{806.159, -54.01}},
		{types.Array(types.DateTime()), []any{time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), time.Date(2024, 10, 3, 15, 38, 36, 920638000, time.UTC)}},
		{types.Array(types.Date()), []any{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC)}},
		{types.Array(types.Time()), []any{time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC), time.Date(1970, 1, 1, 15, 40, 07, 184741000, time.UTC)}},
		{types.Array(types.Year()), []any{1, 1970, 2020, 9999}},
		{types.Array(types.UUID()), []any{"4d92d698-687d-4447-b34f-6b29d74a9730", "4d92d698-687d-4447-b34f-6b29d74a9730"}},
		{types.Array(types.JSON()), []any{json.Value(`{"foo":"boo"}`), json.Value(`null`)}},
		{types.Array(types.IP()), []any{"127.0.0.1", "2001:db8:85a3::8a2e:370:7334"}},
		{types.Array(types.String()), []any{"foo", "boo"}},
		{types.Map(types.Int(32)), map[string]any{"boo": 15, "foo": 33}},
		{types.Map(types.JSON()), map[string]any{"boo": json.Value(`5`), "foo": json.Value(`{"a":3,"b":5}`)}},
		{types.Map(types.String()), map[string]any{"boo": "hello", "foo": "world"}},
	}

	table := warehouses.Table{
		Name:    "test_krenalis_merge",
		Columns: make([]warehouses.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = warehouses.Column{
			Name:     fmt.Sprintf("c%d", i),
			Type:     c.KrenalisType,
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
	testHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(map[string]any{
		"host":     testHost,
		"port":     testPort.Int(),
		"username": testUser,
		"password": testPassword,
		"database": testDatabase,
		"schema":   "public",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Open the data warehouse.
	wh, err := warehouses.Registered("PostgreSQL").New(&warehouses.Config{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wh.Close()

	pool, err := wh.(*PostgreSQL).connectionPool(context.Background())
	if err != nil {
		t.Fatalf("cannot open the warehouse: %s", err)
	}

	// Create the table.
	create := bytes.NewBufferString("CREATE TABLE " + quoteIdent(table.Name) + " (\n\t")
	for i, c := range table.Columns {
		if i > 0 {
			create.WriteString(",\n\t")
		}
		create.WriteString(quoteIdent(c.Name))
		create.WriteByte(' ')
		create.WriteString(typeToPostgresType(cols[i].KrenalisType))
		if i == 0 {
			create.WriteString(" PRIMARY KEY")
		}
	}
	create.WriteString("\n)")
	_, err = pool.Exec(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = pool.Exec(context.Background(), "DROP TABLE "+quoteIdent(table.Name))
		if err != nil {
			t.Logf("cannot drop %s table: %s", table.Name, err)
		}
	}()

	// Merge the values.
	row1 := make([]any, len(table.Columns))
	for i := range table.Columns {
		row1[i] = cols[i].KrenalisValue
	}
	row2 := make([]any, len(table.Columns))
	for i := range table.Columns {
		if i == 0 {
			row2[i] = cols[0].KrenalisValue.(string) + "2"
			continue
		}
		row2[i] = nil
	}
	err = wh.Merge(context.Background(), table, [][]any{row1, row2}, nil)
	if err != nil {
		t.Fatalf("cannot merge: %s", err)
	}

	// Execute the query.
	query := warehouses.RowQuery{
		Table:   table.Name,
		Columns: table.Columns,
	}
	rows, count, err := wh.Query(context.Background(), query, true)
	if err != nil {
		t.Fatalf("cannot query: %s", err)
	}
	defer rows.Close()
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}

	// Scan the rows.
	var row = make([]any, len(table.Columns))
	if !rows.Next() {
		t.Fatal("expected the first row, got no rows")
	}
	if err := rows.Scan(row...); err != nil {
		t.Fatalf("unexpected error scanning the first row: %s", err)
	}
	for i, got := range row {
		c := cols[i]
		switch c.KrenalisType.Kind() {
		case types.JSONKind:
			v, ok := got.(json.Value)
			if !ok {
				t.Fatalf("type %s: expected a json.Value value, got %#v (type %T)", c.KrenalisType, got, got)
			}
			v, err = json.Compact(v)
			if err != nil {
				t.Fatalf("type %s: cannot compact JSON value %#v", c.KrenalisType, got)
			}
			got = v
		case types.ArrayKind:
			if c.KrenalisType.Elem().Kind() == types.JSONKind {
				elements, ok := got.([]any)
				if !ok {
					t.Fatalf("type %s: expected a []any value, got %#v (type %T)", c.KrenalisType, got, got)
				}
				for i, element := range elements {
					v, ok := element.(json.Value)
					if !ok {
						t.Fatalf("type %s: expected a json.Value element, got %#v (type %T)", c.KrenalisType, element, element)
					}
					v, err = json.Compact(v)
					if err != nil {
						t.Fatalf("type %s: cannot compact JSON value %#v", c.KrenalisType, got)
					}
					elements[i] = v
				}
			}
		case types.MapKind:
			if c.KrenalisType.Elem().Kind() == types.JSONKind {
				elements, ok := got.(map[string]any)
				if !ok {
					t.Fatalf("type %s: expected a map[string]any value, got %#v (type %T)", c.KrenalisType, got, got)
				}
				for key, value := range elements {
					v, ok := value.(json.Value)
					if !ok {
						t.Fatalf("type %s: expected a json.Value element, got %#v (type %T)", c.KrenalisType, value, value)
					}
					v, err = json.Compact(v)
					if err != nil {
						t.Fatalf("type %s: cannot compact JSON value %#v", c.KrenalisType, got)
					}
					elements[key] = v
				}
			}
		}
		if !cmp.Equal(c.KrenalisValue, got) {
			t.Fatalf("type %s: expected %#v (type %T), got %#v (type %T)", c.KrenalisType, c.KrenalisValue, c.KrenalisValue, got, got)
		}
	}
	if !rows.Next() {
		t.Fatal("expected the second row, got no rows")
	}
	clear(row)
	if err := rows.Scan(row...); err != nil {
		t.Fatalf("unexpected error scanning the second row: %s", err)
	}
	for i, got := range row {
		if i == 0 {
			continue
		}
		c := cols[i]
		if got != nil {
			t.Fatalf("type %s: expected nil, got %#v (type %T)", c.KrenalisType, got, got)
		}
	}
	if rows.Next() {
		t.Fatal("expected 2 row, got 3")
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("unexpected error scanning rows: %s", err)
	}
}

// Test_rowEncoder verifies that rows are encoded according to the column
// definitions and that non-encodable rows produce errors when expected.
func Test_rowEncoder(t *testing.T) {
	tests := []struct {
		columns  []warehouses.Column
		rows     [][]any
		expected [][]any
		ok       bool
	}{
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.String()}},
			rows:     [][]any{{"boo"}, {"\x00foo"}},
			expected: [][]any{{"boo"}, {"foo"}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Boolean()}},
			rows:     [][]any{{true}, {false}},
			expected: [][]any{{true}, {false}},
			ok:       false,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.JSON()}},
			rows:     [][]any{{json.Value(`"boo"`)}, {json.Value(`"\u0000foo"`)}},
			expected: [][]any{{json.Value(`"boo"`)}, {json.Value(`"foo"`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Array(types.String())}},
			rows:     [][]any{{[]any{"boo", "foo"}}, {[]any{"\x00foo", "boo", "\x00"}}},
			expected: [][]any{{[]any{"boo", "foo"}}, {[]any{"foo", "boo", ""}}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Map(types.Int(32))}},
			rows:     [][]any{{map[string]any{"boo": 5}}, {map[string]any{"'boo\x00'": 7, "hello \x00world": 2}}},
			expected: [][]any{{json.Value(`{"boo":5}`)}, {json.Value(`{"'boo'":7,"hello world":2}`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Map(types.JSON())}},
			rows:     [][]any{{map[string]any{"boo": json.Value(`{"a":5}`)}}, {map[string]any{"'boo\x00'": json.Value(`{"b":"\u0000foo\\u0000"}`)}}},
			expected: [][]any{{json.Value(`{"boo":{"a":5}}`)}, {json.Value(`{"'boo'":{"b":"foo\\u0000"}}`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "a", Type: types.String()}, {Name: "b", Type: types.Float(32)}, {Name: "c", Type: types.Map(types.String())}},
			rows:     [][]any{{"\x00boo", 1.234, map[string]any{"boo": ""}}, {"\x00", -73.55, map[string]any{"boo": "\x00foo", "hello\x00 world": "\x00"}}},
			expected: [][]any{{"boo", 1.234, json.Value(`{"boo":""}`)}, {"", -73.55, json.Value(`{"boo":"foo","hello world":""}`)}},
			ok:       true,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			enc, ok := newRowEncoder(test.columns)
			if ok != test.ok {
				t.Fatalf("expected ok %t, got %t", test.ok, ok)
			}
			if !ok {
				return
			}
			for i, row := range test.rows {
				err := enc.encode(row)
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(test.expected[i], row) {
					t.Fatalf("unexpected row:\n\texpected: %#v\n\tgot:      %#v\n\n%s\n", test.expected[i], row, cmp.Diff(test.expected[i], row))
				}
			}
		})
	}
}

// Test_newRowEncoder_noColumns ensures that newRowEncoder returns nil when the
// column set does not require encoding.
func Test_newRowEncoder_noColumns(t *testing.T) {
	enc, ok := newRowEncoder([]warehouses.Column{{Name: "a", Type: types.Int(32)}})
	if enc != nil || ok {
		t.Fatalf("expected nil encoder and false, got %#v %t", enc, ok)
	}
}

// Test_rowEncoder_mapZeroBytes tests map encoding and ensures embedded zero
// bytes are stripped from string values.
func Test_rowEncoder_mapZeroBytes(t *testing.T) {
	columns := []warehouses.Column{{Name: "m", Type: types.Map(types.String())}}
	enc, ok := newRowEncoder(columns)
	if !ok {
		t.Fatal("expected encoder")
	}
	row := []any{map[string]any{"foo": "bar\x00baz"}}
	if err := enc.encode(row); err != nil {
		t.Fatal(err)
	}
	expected := json.Value(`{"foo":"barbaz"}`)
	if !reflect.DeepEqual(row[0], expected) {
		t.Fatalf("expected %v, got %#v", expected, row[0])
	}
}

// Test_stripZeroBytes removes zero bytes from strings and ensures the output is
// correct.
func Test_stripZeroBytes(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{"", ""},
		{"\x00", ""},
		{"\x00\x00", ""},
		{"a", "a"},
		{"hello world", "hello world"},
		{"\x00hello\x00\x00 world\x00\x00", "hello world"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := stripZeroBytes(test.s)
			if test.expected != got {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}
}

// Test_QueryReadOnly verifies that QueryReadOnly returns rows for valid
// read-only statements.
func Test_QueryReadOnly(t *testing.T) {
	wh, pool := newTestPostgreSQLWarehouse(t)

	mustExecSQL(t, pool, `CREATE TABLE "test_queryreadonly" ("id" integer PRIMARY KEY, "name" text)`)
	mustExecSQL(t, pool, `INSERT INTO "test_queryreadonly" ("id", "name") VALUES (1, 'a'), (2, 'b')`)

	rows, columnCount, err := wh.QueryReadOnly(context.Background(), `SELECT "id", "name" FROM "test_queryreadonly" ORDER BY "id"`)
	if err != nil {
		t.Fatalf("QueryReadOnly returned error: %s", err)
	}
	defer rows.Close()
	if columnCount != 2 {
		t.Fatalf("expected 2 columns, got %d", columnCount)
	}

	var id int32
	var name string
	if !rows.Next() {
		t.Fatal("expected first row, got none")
	}
	if err := rows.Scan(&id, &name); err != nil {
		t.Fatalf("unexpected scan error on first row: %s", err)
	}
	if id != 1 || name != "a" {
		t.Fatalf("unexpected first row: id=%d name=%q", id, name)
	}

	if !rows.Next() {
		t.Fatal("expected second row, got none")
	}
	if err := rows.Scan(&id, &name); err != nil {
		t.Fatalf("unexpected scan error on second row: %s", err)
	}
	if id != 2 || name != "b" {
		t.Fatalf("unexpected second row: id=%d name=%q", id, name)
	}

	if rows.Next() {
		t.Fatal("expected 2 rows, got more")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("unexpected rows error: %s", err)
	}
}

// Test_QueryReadOnly_closeIdempotent ensures that closing read-only rows more
// than once is harmless.
func Test_QueryReadOnly_closeIdempotent(t *testing.T) {
	wh, _ := newTestPostgreSQLWarehouse(t)

	rows, _, err := wh.QueryReadOnly(context.Background(), `SELECT 1`)
	if err != nil {
		t.Fatalf("QueryReadOnly returned error: %s", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("first Close returned error: %s", err)
	}
	if err := rows.Close(); err != nil {
		t.Fatalf("second Close returned error: %s", err)
	}
}

// Test_QueryReadOnly_readOnlyTransaction ensures that PostgreSQL still enforces
// read-only execution if a statement slips past lexical validation.
func Test_QueryReadOnly_readOnlyTransaction(t *testing.T) {
	wh, pool := newTestPostgreSQLWarehouse(t)

	mustExecSQL(t, pool, `CREATE TABLE "test_queryreadonly_writes" ("value" integer NOT NULL)`)
	mustExecSQL(t, pool, `
CREATE FUNCTION test_queryreadonly_write(integer, integer)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
	INSERT INTO "test_queryreadonly_writes" ("value") VALUES ($1 + $2);
	RETURN $1 + $2;
END;
$$`)
	mustExecSQL(t, pool, `
CREATE OPERATOR <#> (
	LEFTARG = integer,
	RIGHTARG = integer,
	FUNCTION = test_queryreadonly_write
)`)

	rows, _, err := wh.QueryReadOnly(context.Background(), `SELECT 1 <#> 2`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
		}
		err = rows.Err()
	}
	if err == nil {
		t.Fatal("expected QueryReadOnly to fail in a read-only transaction")
	}
	if !strings.Contains(err.Error(), "read-only transaction") {
		t.Fatalf("expected read-only transaction error, got %q", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `SELECT COUNT(*) FROM "test_queryreadonly_writes"`).Scan(&count); err != nil {
		t.Fatalf("cannot count writes: %s", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 writes, got %d", count)
	}
}

// newTestPostgreSQLWarehouse starts a PostgreSQL container and opens a warehouse
// against it.
func newTestPostgreSQLWarehouse(t *testing.T) (*PostgreSQL, *pgxpool.Pool) {
	t.Helper()

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
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Error(err)
		}
	})

	testHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(map[string]any{
		"host":     testHost,
		"port":     testPort.Int(),
		"username": testUser,
		"password": testPassword,
		"database": testDatabase,
		"schema":   "public",
	})
	if err != nil {
		t.Fatal(err)
	}

	warehouse, err := warehouses.Registered("PostgreSQL").New(&warehouses.Config{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := warehouse.Close(); err != nil {
			t.Error(err)
		}
	})

	wh, ok := warehouse.(*PostgreSQL)
	if !ok {
		t.Fatalf("expected *PostgreSQL, got %T", warehouse)
	}
	pool, err := wh.connectionPool(ctx)
	if err != nil {
		t.Fatalf("cannot open the warehouse: %s", err)
	}

	return wh, pool
}

func mustExecSQL(t *testing.T, pool *pgxpool.Pool, sql string) {
	t.Helper()

	if _, err := pool.Exec(context.Background(), sql); err != nil {
		t.Fatalf("cannot execute SQL %q: %s", sql, err)
	}
}
