// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"slices"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

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

// TestCounts checks unconditional and conditional row counts.
func TestCounts(t *testing.T) {

	warehouse, db := newTestSnowflakeWarehouse(t)
	mustExecSQL(t, db, `CREATE TABLE "COUNT_CONDITIONS" ("FLAG" BOOLEAN, "DATA" VARIANT)`)
	mustExecSQL(t, db, `INSERT INTO "COUNT_CONDITIONS"
		SELECT COLUMN1, PARSE_JSON(COLUMN2) FROM VALUES
		(TRUE, '{"key":true,"a.b":true}'), (FALSE, '{"key":false,"a":{"b":true}}'), (NULL, '{}'),
		(TRUE, '{"key":"true"}'), (NULL, '{"key":true}'), (NULL, NULL)`)

	flag := warehouses.NewBaseExpr(warehouses.Column{Name: "flag", Type: types.Boolean()}, warehouses.OpIsTrue)
	data := warehouses.NewBaseExpr(warehouses.Column{Name: "data", Type: types.JSON()}, warehouses.OpIsTrue)
	data.Key = "key"
	falseData := warehouses.NewBaseExpr(warehouses.Column{Name: "data", Type: types.JSON()}, warehouses.OpIsFalse)
	falseData.Key = "key"
	dottedKey := warehouses.NewBaseExpr(warehouses.Column{Name: "data", Type: types.JSON()}, warehouses.OpIsTrue)
	dottedKey.Key = "a.b"
	conditions := []warehouses.Expr{
		nil,
		flag,
		data,
		falseData,
		dottedKey,
		warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{flag, data}),
	}

	totalCounts, err := warehouse.Counts(t.Context(), "count_conditions", nil)
	if err != nil {
		t.Fatalf("expected counts, got %v", err)
	}
	if !slices.Equal(totalCounts, []int{6}) {
		t.Fatalf("expected counts [6], got %v", totalCounts)
	}

	counts, err := warehouse.Counts(t.Context(), "count_conditions", conditions)
	if err != nil {
		t.Fatalf("expected counts, got %v", err)
	}
	want := []int{6, 2, 2, 1, 1, 1}
	if !slices.Equal(counts, want) {
		t.Fatalf("expected counts %v, got %v", want, counts)
	}

}
