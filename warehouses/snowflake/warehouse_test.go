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
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"

	"github.com/google/go-cmp/cmp"
)

const settingsEnvKey = "MEERGO_TEST_PATH_SNOWFLAKE"

func Test_Merge(t *testing.T) {

	cols := []struct {
		MeergoType  types.Type
		MeergoValue any
	}{
		{types.Text(), "foo"},
		{types.Boolean(), true},
		{types.Int(8), 103},
		{types.Int(16), 8030},
		{types.Int(24), -3582672},
		{types.Int(32), 1023947264},
		{types.Int(64), -603826591193},
		{types.Uint(8), uint(249)},
		{types.Uint(16), uint(22941)},
		{types.Uint(24), uint(1300928)},
		{types.Uint(32), uint(3281905844)},
		{types.Uint(64), uint(1003883597101)},
		{types.Float(32), float64(float32(1.123))},
		{types.Float(64), 1.123},
		{types.Decimal(10, 3), decimal.MustParse("1.123")},
		{types.DateTime(), time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC)},
		{types.Date(), time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)},
		{types.Time(), time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC)},
		{types.Year(), 2014},
		{types.UUID(), "4d92d698-687d-4447-b34f-6b29d74a9730"},
		{types.JSON(), json.Value(`{"foo":"boo"}`)},
		{types.Inet(), "127.0.0.1"},
		{types.Array(types.Boolean()), []any{true, false}},
		{types.Array(types.Int(8)), []any{5, -2, 12}},
		{types.Array(types.Int(16)), []any{32057, -9381, 1623}},
		{types.Array(types.Int(24)), []any{6318609, -93810, 16423}},
		{types.Array(types.Int(32)), []any{7936605, -179804772, 23}},
		{types.Array(types.Int(64)), []any{-193874627541, 819, 3481674621874}},
		{types.Array(types.Uint(8)), []any{uint(223), uint(66), uint(130)}},
		{types.Array(types.Uint(16)), []any{uint(65535), uint(840), uint(12)}},
		{types.Array(types.Uint(24)), []any{uint(16570147), uint(193810), uint(942754)}},
		{types.Array(types.Uint(32)), []any{uint(4164303781), uint(8400), uint(13)}},
		{types.Array(types.Uint(64)), []any{uint(18446744073709551615), uint(8400), uint(13)}},
		{types.Array(types.Float(32)), []any{float64(float32(2.64)), float64(float32(1.212))}},
		{types.Array(types.Float(64)), []any{806.159, -54.01}},
		{types.Array(types.DateTime()), []any{time.Date(2023, 1, 1, 1, 2, 3, 0, time.UTC), time.Date(2024, 10, 3, 15, 38, 36, 920638000, time.UTC)}},
		{types.Array(types.Date()), []any{time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 10, 3, 0, 0, 0, 0, time.UTC)}},
		{types.Array(types.Time()), []any{time.Date(1970, 1, 1, 2, 3, 0, 0, time.UTC), time.Date(1970, 1, 1, 15, 40, 07, 184741000, time.UTC)}},
		{types.Array(types.Year()), []any{1, 1970, 2020, 9999}},
		{types.Array(types.UUID()), []any{"4d92d698-687d-4447-b34f-6b29d74a9730", "4d92d698-687d-4447-b34f-6b29d74a9730"}},
		{types.Array(types.JSON()), []any{json.Value(`{"foo":"boo"}`), json.Value(`null`)}},
		{types.Array(types.Inet()), []any{"127.0.0.1", "2001:db8:85a3::8a2e:370:7334"}},
		{types.Array(types.Text()), []any{"foo", "boo"}},
		{types.Map(types.Int(32)), map[string]any{"boo": 15, "foo": 33}},
		{types.Map(types.JSON()), map[string]any{"boo": json.Value(`5`), "foo": json.Value(`{"a":3,"b":5}`)}},
		{types.Map(types.Text()), map[string]any{"boo": "hello", "foo": "world"}},
	}

	table := meergo.Table{
		Name:    "test_meergo_merge",
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

	settingsFile, ok := os.LookupEnv(settingsEnvKey)
	if !ok {
		t.Skipf("the %s environment variable is not present", settingsEnvKey)
	}

	// Open the data warehouse.
	settings, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("cannot open the path %q specified in the %s environment variable: %s", settingsFile, settingsEnvKey, err)
	}
	wh, err := meergo.RegisteredWarehouseDriver("Snowflake").New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		t.Fatalf("cannot open the warehouse from settings in the %s environment variable: %s", settingsEnvKey, err)
	}
	defer wh.Close()

	db := wh.(*Snowflake).openDB()

	// Create the table.
	create := bytes.NewBufferString("CREATE TABLE " + quoteIdent(table.Name) + " (\n\t")
	for i, c := range table.Columns {
		if i > 0 {
			create.WriteString(",\n\t")
		}
		create.WriteString(quoteIdent(c.Name))
		create.WriteByte(' ')
		create.WriteString(typeToSnowflakeType(cols[i].MeergoType))
	}
	create.WriteString("\n)")
	_, err = db.ExecContext(context.Background(), create.String())
	if err != nil {
		t.Fatalf("cannot create table: %s", err)
	}
	defer func() {
		_, err = db.ExecContext(context.Background(), `DROP TABLE `+quoteIdent(table.Name))
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
		row2[i] = nil
	}
	err = wh.Merge(context.Background(), table, [][]any{row1, row2}, nil)
	if err != nil {
		t.Fatalf("cannot merge: %s", err)
	}

	// Execute the query.
	query := meergo.RowQuery{
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
