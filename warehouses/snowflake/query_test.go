// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestQueryArrayScanLimit checks that an oversized array is rejected at the
// source and produces a scan error.
func TestQueryArrayScanLimit(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	mustExecSQL(t, db, `CREATE TABLE "QUERY_ARRAYS" ("_ANONYMOUS_IDS" ARRAY)`)
	mustExecSQL(t, db, `INSERT INTO "QUERY_ARRAYS" SELECT TRANSFORM(ARRAY_GENERATE_RANGE(0, 1000000), i -> 'a')`)

	query := warehouses.RowQuery{
		Table:   "query_arrays",
		Columns: []warehouses.Column{{Name: "_anonymous_ids", Type: types.Array(types.String())}},
		Limit:   1,
	}
	rows, _, err := warehouse.Query(t.Context(), query, false)
	if err != nil {
		t.Fatalf("expected query to succeed, got %v", err)
	}
	defer rows.Close()
	if !rows.Next() {
		t.Fatalf("expected one row, got none (err %v)", rows.Err())
	}
	var payload string
	err = rows.(*scanner).rows.Scan(&payload)
	if err != nil {
		t.Fatalf("expected array projection, got %v", err)
	}
	if len(payload) != 0 {
		t.Fatalf("expected 0 payload bytes, got %d", len(payload))
	}
	values := make([]any, 1)
	err = rows.Scan(values...)
	if err != nil {
		want := fmt.Sprintf("array for column _anonymous_ids exceeds the %d-byte scan limit", maxArrayScanBytes)
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error containing %q, got %v", want, err)
		}
		return
	}
	t.Fatal("expected scan limit error, got nil")

}

// TestQueryOrdering verifies that an empty OrderBy is omitted and descending order is applied to every ordered column.
func TestQueryOrdering(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	mustExecSQL(t, db, `CREATE TABLE "QUERY_ORDERING" ("A" INTEGER NOT NULL, "B" INTEGER NOT NULL)`)
	mustExecSQL(t, db, `INSERT INTO "QUERY_ORDERING" VALUES (2, 1), (2, 2), (1, 3)`)

	columns := []warehouses.Column{
		{Name: "a", Type: types.Int(32)},
		{Name: "b", Type: types.Int(32)},
	}

	t.Run("empty", func(t *testing.T) {

		rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
			Table:   "query_ordering",
			Columns: columns,
			OrderBy: []warehouses.Column{},
		}, false)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
		}
		err = rows.Err()
		if err != nil {
			t.Fatal(err)
		}

	})

	t.Run("descending", func(t *testing.T) {

		rows, _, err := warehouse.Query(t.Context(), warehouses.RowQuery{
			Table:     "query_ordering",
			Columns:   columns,
			OrderBy:   columns,
			OrderDesc: true,
		}, false)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()

		want := [][2]int{{2, 2}, {2, 1}, {1, 3}}
		var row int
		for rows.Next() {
			if row == len(want) {
				t.Fatal("returned too many rows")
			}
			values := make([]any, len(columns))
			err = rows.Scan(values...)
			if err != nil {
				t.Fatal(err)
			}
			if values[0] != want[row][0] || values[1] != want[row][1] {
				t.Fatalf("row %d: expected %v, got %v", row, want[row], values)
			}
			row++
		}
		err = rows.Err()
		if err != nil {
			t.Fatal(err)
		}
		if row != len(want) {
			t.Fatalf("expected %d rows, got %d", len(want), row)
		}

	})

}
