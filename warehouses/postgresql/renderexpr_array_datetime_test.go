// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestArrayDateTimeContains compares offset operands with stored UTC instants.
func TestArrayDateTimeContains(t *testing.T) {
	warehouse, db := newTestPostgreSQLWarehouse(t)
	instant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	for _, test := range []struct {
		name   string
		offset int
	}{
		{"utc", 0},
		{"positive", 2 * 60 * 60},
		{"negative", -5 * 60 * 60},
	} {
		t.Run(test.name, func(t *testing.T) {

			idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
			column := warehouses.Column{Name: "value", Type: types.Array(types.DateTime()), Nullable: true}
			table := warehouses.Table{
				Name:    "array_datetime_" + test.name,
				Columns: []warehouses.Column{idColumn, column},
				Keys:    []string{"id"},
			}
			mustExecSQL(t, db, "CREATE TABLE "+quoteIdent(table.Name)+" ("+quoteIdent(idColumn.Name)+
				" INTEGER PRIMARY KEY, "+quoteIdent(column.Name)+" "+typeToPostgresType(column.Type)+")")
			err := warehouse.Merge(t.Context(), table, [][]any{
				{0, nil},
				{1, []any{}},
				{2, []any{instant}},
				{3, []any{instant.Add(time.Hour)}},
				{4, []any{instant.Add(time.Hour), instant, instant}},
			}, nil)
			if err != nil {
				t.Fatalf("expected array fixture merge, got %v", err)
			}

			for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
				t.Run(fmt.Sprint(op), func(t *testing.T) {

					value := instant.In(time.FixedZone("", test.offset))
					expr := warehouses.NewBaseExpr(column, op, value)
					want := []int{2, 4}
					if op == warehouses.OpDoesNotContain {
						want = []int{1, 3}
					}
					query := warehouses.RowQuery{
						Table: table.Name, Columns: []warehouses.Column{idColumn},
						Where: expr, OrderBy: []warehouses.Column{idColumn},
					}
					rows, total, err := warehouse.Query(t.Context(), query, true)
					if err != nil {
						t.Fatalf("expected array query, got %v", err)
					}
					defer rows.Close()
					var got []int
					row := make([]any, 1)
					for rows.Next() {
						if err := rows.Scan(row...); err != nil {
							t.Fatalf("expected array query row, got %v", err)
						}
						got = append(got, row[0].(int))
					}
					err = rows.Err()
					if err != nil {
						t.Fatalf("expected complete array query, got %v", err)
					}
					if !slices.Equal(got, want) || total != len(want) {
						t.Errorf("expected ids %v and total %d, got %v and %d", want, len(want), got, total)
					}

					var b strings.Builder
					err = renderExpr(&b, expr)
					if err != nil {
						t.Fatalf("expected array expression, got %v", err)
					}
					statement := "SELECT " + quoteIdent(idColumn.Name) + ", (" + b.String() + ") FROM " +
						quoteIdent(table.Name) + " ORDER BY " + quoteIdent(idColumn.Name)
					truthRows, err := db.Query(t.Context(), statement)
					if err != nil {
						t.Fatalf("expected array truth query, got %v", err)
					}
					defer truthRows.Close()
					count := 0
					for truthRows.Next() {
						var id int
						var truth sql.NullBool
						if err := truthRows.Scan(&id, &truth); err != nil {
							t.Fatalf("expected array truth row, got %v", err)
						}
						wantTruth := sql.NullBool{Bool: slices.Contains(want, id), Valid: id != 0}
						if truth != wantTruth {
							t.Errorf("expected id %d truth %v, got %v", id, wantTruth, truth)
						}
						count++
					}
					err = truthRows.Err()
					if err != nil {
						t.Fatalf("expected complete array truth query, got %v", err)
					}
					if count != 5 {
						t.Errorf("expected 5 array truth rows, got %d", count)
					}

				})
			}

		})
	}
}

// TestArrayDateTimeContainsRendering normalizes datetime operands and preserves
// the wall-clock values of date and time operands.
func TestArrayDateTimeContainsRendering(t *testing.T) {
	instant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	clock := time.Date(2026, 10, 20, 0, 30, 0, 123456000, time.FixedZone("", 7200))
	for _, test := range []struct {
		name  string
		typ   types.Type
		value time.Time
		want  string
	}{
		{"utc", types.DateTime(), instant, `'2026-10-20 23:30:45.123456' = ANY("value")`},
		{"positive", types.DateTime(), instant.In(time.FixedZone("", 7200)), `'2026-10-20 23:30:45.123456' = ANY("value")`},
		{"negative", types.DateTime(), instant.In(time.FixedZone("", -18000)), `'2026-10-20 23:30:45.123456' = ANY("value")`},
		{"date", types.Date(), clock, `'2026-10-20' = ANY("value")`},
		{"time", types.Time(), clock, `'00:30:00.123456' = ANY("value")`},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
			t.Run(fmt.Sprintf("%s/%d", test.name, op), func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.Array(test.typ), Nullable: true}
				var b strings.Builder
				err := renderExpr(&b, warehouses.NewBaseExpr(column, op, test.value))
				if err != nil {
					t.Fatalf("expected array expression, got %v", err)
				}
				want := test.want
				if op == warehouses.OpDoesNotContain {
					want = "NOT (" + want + ")"
				}
				if b.String() != want {
					t.Errorf("expected SQL %s, got %s", want, b.String())
				}
			})
		}
	}
}
