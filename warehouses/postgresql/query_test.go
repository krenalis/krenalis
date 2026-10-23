// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
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

// Test_renderCountQuery verifies counts with optional joins and filters.
func Test_renderCountQuery(t *testing.T) {

	id := warehouses.Column{Name: "id", Type: types.Int(32)}
	joins := []warehouses.Join{{
		Type:      warehouses.InnerJoin,
		Table:     "identities",
		Condition: warehouses.NewBaseExpr(id, warehouses.OpIs, warehouses.Column{Name: "fk", Type: types.Int(32)}),
	}}
	where := warehouses.NewBaseExpr(id, warehouses.OpIs, 1)

	for _, tc := range []struct {
		name  string
		joins []warehouses.Join
		where warehouses.Expr
		want  string
	}{
		{name: "all rows", want: `SELECT COUNT(*) FROM "profiles"`},
		{name: "filtered rows", where: where, want: `SELECT COUNT(*) FROM "profiles" WHERE "id" = 1`},
		{
			name:  "joined rows",
			joins: joins,
			want:  `SELECT COUNT(*) FROM "profiles" JOIN "identities" ON "id" = "fk"`,
		},
		{
			name:  "filtered joined rows",
			joins: joins,
			where: where,
			want:  `SELECT COUNT(*) FROM "profiles" JOIN "identities" ON "id" = "fk" WHERE "id" = 1`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			statement, err := renderCountQuery("profiles", tc.joins, tc.where)
			if err != nil {
				t.Fatal(err)
			}
			if statement != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, statement)
			}
		})
	}

}
