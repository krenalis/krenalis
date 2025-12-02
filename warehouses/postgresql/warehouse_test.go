// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/test/testimages"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"

	"github.com/google/go-cmp/cmp"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDatabase = "meergo"
	testUser     = "meergo"
	testPassword = "meergo"
)

func Test_Merge(t *testing.T) {

	cols := []struct {
		MeergoType  types.Type
		MeergoValue any
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
		Name:    "test_meergo_merge",
		Columns: make([]warehouses.Column, len(cols)),
		Keys:    []string{"c0"},
	}
	for i, c := range cols {
		table.Columns[i] = warehouses.Column{
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
	testHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(map[string]any{
		"Host":     testHost,
		"Port":     testPort.Int(),
		"Username": testUser,
		"Password": testPassword,
		"Database": testDatabase,
		"Schema":   "public",
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
		create.WriteString(typeToPostgresType(cols[i].MeergoType))
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
		row1[i] = cols[i].MeergoValue
	}
	row2 := make([]any, len(table.Columns))
	for i := range table.Columns {
		if i == 0 {
			row2[i] = cols[0].MeergoValue.(string) + "2"
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
		switch c.MeergoType.Kind() {
		case types.JSONKind:
			v, ok := got.(json.Value)
			if !ok {
				t.Fatalf("type %s: expected a json.Value value, got %#v (type %T)", c.MeergoType, got, got)
			}
			v, err = json.Compact(v)
			if err != nil {
				t.Fatalf("type %s: cannot compact JSON value %#v", c.MeergoType, got)
			}
			got = v
		case types.ArrayKind:
			if c.MeergoType.Elem().Kind() == types.JSONKind {
				elements, ok := got.([]any)
				if !ok {
					t.Fatalf("type %s: expected a []any value, got %#v (type %T)", c.MeergoType, got, got)
				}
				for i, element := range elements {
					v, ok := element.(json.Value)
					if !ok {
						t.Fatalf("type %s: expected a json.Value element, got %#v (type %T)", c.MeergoType, element, element)
					}
					v, err = json.Compact(v)
					if err != nil {
						t.Fatalf("type %s: cannot compact JSON value %#v", c.MeergoType, got)
					}
					elements[i] = v
				}
			}
		case types.MapKind:
			if c.MeergoType.Elem().Kind() == types.JSONKind {
				elements, ok := got.(map[string]any)
				if !ok {
					t.Fatalf("type %s: expected a map[string]any value, got %#v (type %T)", c.MeergoType, got, got)
				}
				for key, value := range elements {
					v, ok := value.(json.Value)
					if !ok {
						t.Fatalf("type %s: expected a json.Value element, got %#v (type %T)", c.MeergoType, value, value)
					}
					v, err = json.Compact(v)
					if err != nil {
						t.Fatalf("type %s: cannot compact JSON value %#v", c.MeergoType, got)
					}
					elements[key] = v
				}
			}
		}
		if !cmp.Equal(c.MeergoValue, got) {
			t.Fatalf("type %s: expected %#v (type %T), got %#v (type %T)", c.MeergoType, c.MeergoValue, c.MeergoValue, got, got)
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
			t.Fatalf("type %s: expected nil, got %#v (type %T)", c.MeergoType, got, got)
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
