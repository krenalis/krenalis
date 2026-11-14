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

// TestQueryWithNonConformingStrings verifies that string filter operands
// remain safe when the database defaults to non-standard string escaping.
func TestQueryWithNonConformingStrings(t *testing.T) {

	warehouse, db := newTestPostgreSQLWarehouse(t)
	mustExecSQL(t, db, `CREATE TABLE "query_string_escaping" ("id" integer, "value" text)`)
	mustExecSQL(t, db, `INSERT INTO "query_string_escaping" VALUES (1, 'safe')`)
	text := `\' OR TRUE --`
	_, err := db.Exec(t.Context(), `INSERT INTO "query_string_escaping" VALUES (2, $1)`, text)
	if err != nil {
		t.Fatalf("expected string fixture insertion, got %v", err)
	}
	mustExecSQL(t, db, "ALTER DATABASE "+quoteIdent(testDatabase)+" SET standard_conforming_strings = off")
	db.Reset()

	idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
	column := warehouses.Column{Name: "value", Type: types.String()}
	expr := warehouses.NewBaseExpr(column, warehouses.OpIs, text)
	rows, total, err := warehouse.Query(t.Context(), warehouses.RowQuery{
		Table: "query_string_escaping", Columns: []warehouses.Column{idColumn},
		Where: expr, OrderBy: []warehouses.Column{idColumn},
	}, true)
	if err != nil {
		t.Fatalf("expected query execution, got %v", err)
	}
	defer rows.Close()
	var ids []int
	values := make([]any, 1)
	for rows.Next() {
		err = rows.Scan(values...)
		if err != nil {
			t.Fatalf("expected row scan, got %v", err)
		}
		id, ok := values[0].(int)
		if !ok {
			t.Fatalf("expected integer row ID, got %T", values[0])
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("expected complete query results, got %v", err)
	}
	if !slices.Equal(ids, []int{2}) || total != 1 {
		t.Fatalf("expected matching IDs [2] and total 1, got %v and %d", ids, total)
	}

}
