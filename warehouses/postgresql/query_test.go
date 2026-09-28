// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"slices"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Test_appendJoins renders join clauses and checks for errors on invalid join
// conditions.
func Test_appendJoins(t *testing.T) {
	join := warehouses.Join{
		Type:  warehouses.InnerJoin,
		Table: "t2",
		Condition: warehouses.NewBaseExpr(
			warehouses.Column{Name: "id", Type: types.Int(32)},
			warehouses.OpIs,
			warehouses.Column{Name: "fk", Type: types.Int(32)},
		),
	}
	var b strings.Builder
	if err := appendJoins(&b, []warehouses.Join{join}); err != nil {
		t.Fatal(err)
	}
	expected := " JOIN \"t2\" ON \"id\" = \"fk\""
	if b.String() != expected {
		t.Fatalf("expected %q, got %q", expected, b.String())
	}

	bad := warehouses.Join{
		Type:      warehouses.InnerJoin,
		Table:     "t3",
		Condition: warehouses.NewBaseExpr(warehouses.Column{Name: "bad name", Type: types.Int(32)}, warehouses.OpIs, 1),
	}
	b.Reset()
	if err := appendJoins(&b, []warehouses.Join{bad}); err == nil {
		t.Fatal("expected error for bad join condition")
	}
}

// TestCounts checks unconditional and conditional row counts.
func TestCounts(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	mustExecSQL(t, pool, `CREATE TABLE "count_conditions" ("flag" BOOLEAN, "other_flag" BOOLEAN)`)
	mustExecSQL(t, pool, `INSERT INTO "count_conditions" VALUES
		(TRUE, TRUE), (TRUE, FALSE), (FALSE, TRUE), (NULL, TRUE)`)

	flag := warehouses.NewBaseExpr(warehouses.Column{Name: "flag", Type: types.Boolean()}, warehouses.OpIsTrue)
	falseFlag := warehouses.NewBaseExpr(warehouses.Column{Name: "flag", Type: types.Boolean()}, warehouses.OpIsFalse)
	otherFlag := warehouses.NewBaseExpr(warehouses.Column{Name: "other_flag", Type: types.Boolean()}, warehouses.OpIsTrue)
	conditions := []warehouses.Expr{
		nil,
		flag,
		falseFlag,
		otherFlag,
		warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{flag, otherFlag}),
	}

	totalCounts, err := warehouse.Counts(t.Context(), "count_conditions", nil)
	if err != nil {
		t.Fatalf("expected counts, got %v", err)
	}
	if !slices.Equal(totalCounts, []int{4}) {
		t.Fatalf("expected counts [4], got %v", totalCounts)
	}

	_, err = warehouse.Counts(t.Context(), "count_conditions", []warehouses.Expr{})
	if err == nil {
		t.Fatalf("expected an error, got %v", err)
	}

	counts, err := warehouse.Counts(t.Context(), "count_conditions", conditions)
	if err != nil {
		t.Fatalf("expected counts, got %v", err)
	}
	want := []int{4, 2, 1, 3, 1}
	if !slices.Equal(counts, want) {
		t.Fatalf("expected counts %v, got %v", want, counts)
	}

}

// TestQueryOrdering verifies that an empty OrderBy is omitted and descending order is applied to every ordered column.
func TestQueryOrdering(t *testing.T) {

	warehouse, pool := newTestPostgreSQLWarehouse(t)
	mustExecSQL(t, pool, `CREATE TABLE "query_ordering" ("a" INTEGER NOT NULL, "b" INTEGER NOT NULL)`)
	mustExecSQL(t, pool, `INSERT INTO "query_ordering" VALUES (2, 1), (2, 2), (1, 3)`)

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
