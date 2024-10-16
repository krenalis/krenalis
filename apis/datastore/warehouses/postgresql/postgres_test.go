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
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/go-cmp/cmp"
)

const settingsEnvKey = "MEERGO_TEST_PATH_POSTGRESQL"

func Test_Merge(t *testing.T) {

	cols := []struct {
		MeergoType  types.Type
		MeergoValue any
		DriverType  string
	}{
		{types.Boolean(), true, "boolean"},
		{types.Int(8), 103, "smallint"},
		{types.Int(16), 8030, "smallint"},
		{types.Int(24), -3582672, "integer"},
		{types.Int(32), 1023947264, "integer"},
		{types.Int(64), -603826591193, "bigint"},
		{types.Uint(8), uint(249), "smallint"},
		{types.Uint(16), uint(22941), "integer"},
		{types.Uint(24), uint(1300928), "integer"},
		{types.Uint(32), uint(3281905844), "bigint"},
		{types.Uint(64), uint(1003883597101), "numeric(20,0)"},
		{types.Float(32), float64(float32(1.123)), "real"},
		{types.Float(64), 1.123, "double precision"},
		{types.Decimal(10, 3), decimal.MustParse("1.123"), "numeric(10,3)"},
		{types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), "timestamp without time zone"},
		{types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), "date"},
		{types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC), "time without time zone"},
		{types.Year(), 2014, "smallint"},
		{types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730", "uuid"},
		{types.JSON(), json.Value(`{"foo":"boo"}`), "jsonb"},
		{types.Inet(), "127.0.0.1", "inet"},
		{types.Text(), "foo", "character varying"},
		{types.Array(types.Boolean()), []any{true, false}, "bool[]"},
		{types.Array(types.Int(8)), []any{5, -2, 12}, "smallint[]"},
		{types.Array(types.Int(16)), []any{32057, -9381, 1623}, "smallint[]"},
		{types.Array(types.Int(24)), []any{6318609, -93810, 16423}, "integer[]"},
		{types.Array(types.Int(32)), []any{7936605, -179804772, 23}, "integer[]"},
		{types.Array(types.Int(64)), []any{-193874627541, 819, 3481674621874}, "bigint[]"},
		{types.Array(types.Uint(8)), []any{uint(223), uint(66), uint(130)}, "smallint[]"},
		{types.Array(types.Uint(16)), []any{uint(65535), uint(840), uint(12)}, "integer[]"},
		{types.Array(types.Uint(24)), []any{uint(16570147), uint(193810), uint(942754)}, "integer[]"},
		{types.Array(types.Uint(32)), []any{uint(4164303781), uint(8400), uint(13)}, "bigint[]"},
		{types.Array(types.Uint(64)), []any{uint(18446744073709551615), uint(8400), uint(13)}, "numeric(20,0)[]"},
		{types.Array(types.Float(32)), []any{float64(float32(2.64)), float64(float32(1.212))}, "real[]"},
		{types.Array(types.Float(64)), []any{806.159, -54.01}, "double precision[]"},
		{types.Array(types.DateTime()), []any{time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), time.Date(2024, 10, 3, 15, 38, 36, 920638000, time.UTC)}, "timestamp without time zone[]"},
		{types.Array(types.Date()), []any{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC)}, "date[]"},
		{types.Array(types.Time()), []any{time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC), time.Date(1970, 1, 1, 15, 40, 07, 184741000, time.UTC)}, "time without time zone[]"},
		{types.Array(types.Year()), []any{1, 1970, 2020, 9999}, "smallint[]"},
		{types.Array(types.UUID()), []any{"4d92d698-687d-4447-b34f-6b29d74a9730", "4d92d698-687d-4447-b34f-6b29d74a9730"}, "uuid[]"},
		{types.Array(types.JSON()), []any{json.Value(`{"foo":"boo"}`), json.Value(`null`)}, "jsonb[]"},
		{types.Array(types.Inet()), []any{"127.0.0.1", "2001:db8:85a3::8a2e:370:7334"}, "inet[]"},
		{types.Array(types.Text()), []any{"foo", "boo"}, "character varying[]"},
		{types.Map(types.Int(32)), map[string]any{"boo": 15, "foo": 33}, "jsonb"},
		{types.Map(types.JSON()), map[string]any{"boo": json.Value(`5`), "foo": json.Value(`{"a":3,"b":5}`)}, "jsonb"},
		{types.Map(types.Text()), map[string]any{"boo": "hello", "foo": "world"}, "jsonb"},
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

	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	// Open the data warehouse.
	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}
	wh, err := Open(settings)
	if err != nil {
		t.Fatalf("cannot open the warehouse from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer wh.Close()

	pool, err := wh.connectionPool(context.Background())
	if err != nil {
		t.Fatalf("cannot open the warehouse: %s", err)
	}

	// Create the table.
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
	_, err = pool.Exec(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = pool.Exec(context.Background(), "DROP TABLE "+table.Name)
		if err != nil {
			t.Logf("cannot drop %s table: %s", table.Name, err)
		}
	}()
	row := make([]any, len(table.Columns))
	for i := range table.Columns {
		row[i] = cols[i].MeergoValue
	}
	err = wh.Merge(context.Background(), table, [][]any{row}, nil)
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
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	values := newScanValues(table.Columns, row, wh.Normalize)
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			t.Fatalf("cannot scan row: %s", err)
			return
		}
		for i, got := range row {
			c := cols[i]
			if !cmp.Equal(c.MeergoValue, got) {
				t.Fatalf("type %s: expected %#v (type %T), got %#v (type %T)", c.MeergoType, c.MeergoValue, c.MeergoValue, got, got)
			}
		}
	}
	if err = rows.Err(); err != nil {
		t.Fatalf("cannot scan rows: %s", err)
	}
}

func Test_rowEncoder(t *testing.T) {
	tests := []struct {
		columns  []warehouses.Column
		rows     [][]any
		expected [][]any
		ok       bool
	}{
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Boolean()}},
			rows:     [][]any{{true}, {false}},
			expected: [][]any{{true}, {false}},
			ok:       false,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Text()}},
			rows:     [][]any{{"boo"}, {"\x00foo"}},
			expected: [][]any{{"boo"}, {"foo"}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.JSON()}},
			rows:     [][]any{{json.Value(`"boo"`)}, {json.Value(`"\u0000foo"`)}},
			expected: [][]any{{json.Value(`"boo"`)}, {json.Value(`"foo"`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "c", Type: types.Array(types.Text())}},
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
			expected: [][]any{{json.Value(`{"boo":"{\"a\":5}"}`)}, {json.Value(`{"'boo'":"{\"b\":\"\\u0000foo\\\\u0000\"}"}`)}},
			ok:       true,
		},
		{
			columns:  []warehouses.Column{{Name: "a", Type: types.Text()}, {Name: "b", Type: types.Float(32)}, {Name: "c", Type: types.Map(types.Text())}},
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

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	columns   []warehouses.Column
	row       []any
	normalize warehouses.NormalizeFunc
	index     int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []warehouses.Column, row []any, normalize warehouses.NormalizeFunc) []any {
	values := make([]any, len(columns))
	value := &scanValue{
		columns:   columns,
		row:       row,
		normalize: normalize,
	}
	for i := range columns {
		values[i] = value
	}
	return values
}

func (sv *scanValue) Scan(src any) error {
	c := sv.columns[sv.index]
	value, err := sv.normalize(c.Name, c.Type, src, c.Nullable)
	if err != nil {
		return err
	}
	sv.row[sv.index] = value
	sv.index = (sv.index + 1) % len(sv.columns)
	return nil
}
