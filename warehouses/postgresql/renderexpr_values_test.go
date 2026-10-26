// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestArrayValueRendering checks the operand encoding of ordinary arrays.
func TestArrayValueRendering(t *testing.T) {
	instant := time.Date(2026, 10, 20, 12, 0, 0, 0, time.FixedZone("", 7200))
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	for _, test := range []struct {
		name  string
		typ   types.Type
		value any
		want  string
	}{

		{"string", types.String(), "foo", `'foo' = ANY("value")`},
		{"int64", types.Int(64), 9223372036854775807, `9223372036854775807 = ANY("value")`},
		{"uint64", types.Int(64).Unsigned(), uint(18446744073709551615), `18446744073709551615 = ANY("value")`},
		{"decimal", types.Decimal(18, 4), decimal.MustParse("1.25"), `1.25 = ANY("value")`},
		{"float32", types.Float(32), float64(float32(1.2345678)), `1.2345677614212036 = ANY("value")`},
		{"datetime", types.DateTime(), instant, `'2026-10-20 10:00:00' = ANY("value")`},
		{"date", types.Date(), instant, `'2026-10-20' = ANY("value")`},
		{"time", types.Time(), clock, `'10:30:00.123456' = ANY("value")`},
	} {
		for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
			t.Run(fmt.Sprintf("%s/%d", test.name, op), func(t *testing.T) {
				column := warehouses.Column{Name: "value", Type: types.Array(test.typ), Nullable: true}
				var b strings.Builder
				err := renderExpr(&b, warehouses.NewBaseExpr(column, op, test.value))
				if err != nil {
					t.Fatalf("expected array expression, got %v", err)
				}
				if !strings.Contains(b.String(), test.want) {
					t.Fatalf("expected SQL containing %s, got %s", test.want, b.String())
				}
			})
		}
	}
}

// TestValueExpressions checks ordinary predicates as SQL truth values and as
// query membership, including absent values and temporal/array regressions.
func TestValueExpressions(t *testing.T) {
	warehouse, db := newTestPostgreSQLWarehouse(t)
	for i, c := range warehouseexpr.ValueCases() {
		t.Run(c.Name, func(t *testing.T) {

			idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
			column := warehouses.Column{Name: "value", Type: c.Type, Nullable: true}
			table := warehouses.Table{
				Name:    fmt.Sprintf("filter_values_%d", i),
				Columns: []warehouses.Column{idColumn, column},
				Keys:    []string{"id"},
			}
			mustExecSQL(t, db, "CREATE TABLE "+quoteIdent(table.Name)+" ("+quoteIdent(idColumn.Name)+
				" INTEGER PRIMARY KEY, "+quoteIdent(column.Name)+" "+typeToPostgresType(c.Type)+")")
			err := warehouse.Merge(t.Context(), table, [][]any{{0, nil}, {1, c.Value}}, nil)
			if err != nil {
				t.Fatalf("expected value fixture merge, got %v", err)
			}
			expr := warehouses.NewBaseExpr(column, c.Operator, c.Values...)
			var matches []int
			if c.Matches {
				matches = []int{1}
			}
			checkJSONExpression(t, warehouse, table.Name, idColumn, expr, matches)

			var b strings.Builder
			err = renderExpr(&b, expr)
			if err != nil {
				t.Fatalf("expected value expression, got %v", err)
			}
			statement := "SELECT " + quoteIdent(idColumn.Name) + ", (" + b.String() + ") FROM " +
				quoteIdent(table.Name) + " ORDER BY " + quoteIdent(idColumn.Name)
			rows, err := db.Query(t.Context(), statement)
			if err != nil {
				t.Fatalf("expected value truth-table query, got %v", err)
			}
			defer rows.Close()
			count := 0
			for rows.Next() {
				var id int
				var got sql.NullBool
				if err := rows.Scan(&id, &got); err != nil {
					t.Fatalf("expected value truth-table row, got %v", err)
				}
				want := sql.NullBool{}
				if id == 1 {
					want = sql.NullBool{Bool: c.Matches, Valid: true}
				}
				if got != want {
					t.Errorf("expected fixture %d result %v, got %v", id, want, got)
				}
				count++
			}
			err = rows.Err()
			if err != nil {
				t.Fatalf("expected complete value truth table, got %v", err)
			}
			if count != 2 {
				t.Fatalf("expected 2 value truth-table rows, got %d", count)
			}

		})
	}
}
