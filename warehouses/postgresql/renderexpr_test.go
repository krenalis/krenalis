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

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestArrayContains checks queries and counts for arrays containing or missing
// a given value, including null and empty arrays.
func TestArrayContains(t *testing.T) {
	warehouse, db := newTestPostgreSQLWarehouse(t)
	instant := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	clock := time.Date(1970, 1, 1, 10, 30, 0, 123456000, time.UTC)
	offsetInstant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	integer39 := decimal.MustParse("123456789012345678901234567890123456790")
	for i, test := range []struct {
		name  string
		typ   types.Type
		value any
		other any
	}{
		{"string", types.String(), "f'o\\o", "bar"},
		{"int32", types.Int(32), -2147483648, 2147483647},
		{"int64", types.Int(64), 9223372036854775807, -9223372036854775808},
		{"uint64", types.Int(64).Unsigned(), uint(18446744073709551615), uint(1)},
		{"decimal", types.Decimal(18, 4), decimal.MustParse("1.25"), decimal.MustParse("1")},
		{"decimal-small", types.Decimal(18, 8), decimal.MustParse("0.00000001"), decimal.MustParse("0.00000002")},
		{"decimal38", types.Decimal(38, 0), decimal.MustParse("12345678901234567890123456789012345678"), decimal.MustParse("1")},
		{"decimal39", types.Decimal(76, 0), decimal.MustParse("123456789012345678901234567890123456789"), integer39},
		{"decimal39-swapped", types.Decimal(76, 0), integer39, decimal.MustParse("123456789012345678901234567890123456789")},
		{"decimal-negative-zero", types.Decimal(76, 0), decimal.MustParse("-0"), decimal.MustParse("1")},
		{"float32", types.Float(32), float64(float32(1.2345678)), 1.5},
		{"float64", types.Float(64), 0.1, 1.5},
		{"datetime", types.DateTime(), instant, instant.Add(time.Hour)},
		// The same local date and time with different offsets must not match.
		{"datetime-positive", types.DateTime(), offsetInstant.In(time.FixedZone("", 7200)), offsetInstant.Add(2 * time.Hour)},
		{"datetime-negative", types.DateTime(), offsetInstant.In(time.FixedZone("", -18000)), offsetInstant.Add(-5 * time.Hour)},
		{"date", types.Date(), instant, instant.AddDate(0, 0, 1)},
		{"time", types.Time(), clock, clock.Add(time.Hour)},
		{"year", types.Int(16).AsYear(), 2026, 2025},
		{"uuid", types.UUID(), "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440001"},
		{"ip", types.IP(), "192.0.2.1", "192.0.2.2"},
	} {
		t.Run(test.name, func(t *testing.T) {

			idColumn := warehouses.Column{Name: "id", Type: types.Int(32)}
			column := warehouses.Column{Name: "value", Type: types.Array(test.typ), Nullable: true}
			table := warehouses.Table{
				Name:    fmt.Sprintf("array_contains_%d", i),
				Columns: []warehouses.Column{idColumn, column},
				Keys:    []string{"id"},
			}
			mustExecSQL(t, db, "CREATE TABLE "+quoteIdent(table.Name)+" ("+quoteIdent(idColumn.Name)+
				" INTEGER PRIMARY KEY, "+quoteIdent(column.Name)+" "+typeToPostgresType(column.Type)+")")
			// Store datetime elements in UTC in the timestamp without time zone column.
			stored := test.value
			if test.typ.IsDateTime() {
				stored = test.value.(time.Time).UTC()
			}
			err := warehouse.Merge(t.Context(), table, [][]any{
				{0, nil},
				{1, []any{}},
				{2, []any{stored}},
				{3, []any{test.other}},
				{4, []any{test.other, stored, stored}},
			}, nil)
			if err != nil {
				t.Fatalf("expected array fixture merge, got %v", err)
			}

			for _, op := range []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain} {
				name := "contains"
				if op == warehouses.OpDoesNotContain {
					name = "does_not_contain"
				}
				t.Run(name, func(t *testing.T) {

					expr := warehouses.NewBaseExpr(column, op, test.value)
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
						err = rows.Scan(row...)
						if err != nil {
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

					counts, err := warehouse.Counts(t.Context(), table.Name, []warehouses.Expr{expr})
					if err != nil {
						t.Fatalf("expected array counts, got %v", err)
					}
					if !slices.Equal(counts, []int{len(want)}) {
						t.Errorf("expected counts [%d], got %v", len(want), counts)
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

// Test_renderExpr converts various expressions to SQL and compares with the
// expected query strings.
func Test_renderExpr(t *testing.T) {
	datetimes := warehouses.Column{Name: "values", Type: types.Array(types.DateTime())}
	instant := time.Date(2026, 10, 20, 23, 30, 45, 123456000, time.UTC)
	positiveOffset := instant.In(time.FixedZone("", 2*60*60))
	negativeOffset := instant.In(time.FixedZone("", -5*60*60))
	tests := []struct {
		expr    warehouses.Expr
		query   string
		invalid bool
	}{
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "ip_addr", Type: types.IP()}, warehouses.OpIs, "127.0.0.1"),
			query: `"ip_addr" = '127.0.0.1'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "age", Type: types.Int(32)}, warehouses.OpIsNot, 18),
			query: `"age" <> 18`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.String()}, warehouses.OpIs, warehouses.Column{Name: "b", Type: types.String()}),
			query: `"a" = "b"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OpIsLessThan, 6.5),
			query: `"weight" < 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OpIsLessThanOrEqualTo, 6.5),
			query: `"weight" <= 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(64)}, warehouses.OpIsGreaterThan, 6.5092373641509225),
			query: `"weight" > 6.5092373641509225`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OpIsGreaterThanOrEqualTo, decimal.MustInt(3289)),
			query: `"count" >= 3289`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" < '1900-01-02 23:32:11.940253'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsOnOrBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" <= '1900-01-02 23:32:11.940253'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.Date()}, warehouses.OpIsAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"timestamp" > '1900-01-02'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.Date()}, warehouses.OpIsOnOrAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"timestamp" >= '1900-01-02'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Int(32)}, warehouses.OpIsBetween, 5, 10),
			query: `"id" BETWEEN 5 AND 10`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpContains, "foo"),
			query: `POSITION('foo' IN "name") > 0`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpDoesNotContain, "boo"),
			query: `POSITION('boo' IN "name") = 0`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Int(32))}, warehouses.OpContains, 5),
			query: `5 = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Int(32))}, warehouses.OpDoesNotContain, 7),
			query: `NOT (7 = ANY("values"))`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.String())}, warehouses.OpContains, "foo"),
			query: `'foo' = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.String())}, warehouses.OpDoesNotContain, "boo"),
			query: `NOT ('boo' = ANY("values"))`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(76, 0))}, warehouses.OpContains, decimal.MustParse("123456789012345678901234567890123456789")),
			query: `123456789012345678901234567890123456789 = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(18, 4))}, warehouses.OpContains, decimal.MustParse("1.25")),
			query: `1.25 = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.Array(types.Decimal(76, 0))}, warehouses.OpContains, decimal.MustParse("1e2")),
			query: `1e2 = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(datetimes, warehouses.OpContains, positiveOffset),
			query: `'2026-10-20 23:30:45.123456' = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(datetimes, warehouses.OpContains, negativeOffset),
			query: `'2026-10-20 23:30:45.123456' = ANY("values")`,
		},
		{
			expr:  warehouses.NewBaseExpr(datetimes, warehouses.OpDoesNotContain, positiveOffset),
			query: `NOT ('2026-10-20 23:30:45.123456' = ANY("values"))`,
		},
		{
			expr:  warehouses.NewBaseExpr(datetimes, warehouses.OpDoesNotContain, negativeOffset),
			query: `NOT ('2026-10-20 23:30:45.123456' = ANY("values"))`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Int(32)}, warehouses.OpIsOneOf, 3, 9, 5),
			query: `"id" IN (3,9,5)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.Float(64)}, warehouses.OpIsOneOf, 5.3, 12.6, 9.0),
			query: `"a" IN (5.3,12.6,9)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpStartsWith, "foo"),
			query: `LEFT("name", LENGTH('foo')) = 'foo'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.String()}, warehouses.OpEndsWith, "foo"),
			query: `RIGHT("name", LENGTH('foo')) = 'foo'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "active", Type: types.Boolean()}, warehouses.OpIsTrue),
			query: `"active"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "active", Type: types.Boolean()}, warehouses.OpIsFalse),
			query: `NOT "active"`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsTrue),
			query: `"properties" IS NOT DISTINCT FROM 'true'::jsonb`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsFalse),
			query: `"properties" IS NOT DISTINCT FROM 'false'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"enabled"},
				Operator: warehouses.OpIsTrue,
			},
			query: `("properties" -> 'enabled') IS NOT DISTINCT FROM 'true'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"archived"},
				Operator: warehouses.OpIsFalse,
			},
			query: `("properties" -> 'archived') IS NOT DISTINCT FROM 'false'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a", "b"},
				Operator: warehouses.OpIsTrue,
			},
			query: `("properties" -> 'a' -> 'b') IS NOT DISTINCT FROM 'true'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a", "b"},
				Operator: warehouses.OpIsFalse,
			},
			query: `("properties" -> 'a' -> 'b') IS NOT DISTINCT FROM 'false'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a.b"},
				Operator: warehouses.OpIsTrue,
			},
			query: `("properties" -> 'a.b') IS NOT DISTINCT FROM 'true'::jsonb`,
		},
		{
			expr: &warehouses.BaseExpr{
				Column:   warehouses.Column{Name: "properties", Type: types.JSON()},
				Keys:     []string{"a'b", "c'd"},
				Operator: warehouses.OpIsFalse,
			},
			query: `("properties" -> 'a''b' -> 'c''d') IS NOT DISTINCT FROM 'false'::jsonb`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNull),
			query: `"id" IS NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIsNotNull),
			query: `"id" IS NOT NULL`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"timestamp" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' AND "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpOr,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' OR "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpAnd,
				[]warehouses.Expr{
					warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_42"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_50"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpIs, "abc_60"),
					}),
					warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Int(32)}, warehouses.OpIs, 100),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Int(32)}, warehouses.OpIs, 200),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Int(32)}, warehouses.OpIs, 300),
					}),
				}),
			query: `("id" = 'abc_42' OR "id" = 'abc_50' OR "id" = 'abc_60') AND ("count" = 100 OR "count" = 200 OR "count" = 300)`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.OpOr,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.String(), Nullable: true}, warehouses.OpIsNotEmpty),
					// warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsEmpty), // See issue https://github.com/krenalis/krenalis/issues/1804.
					warehouses.NewBaseExpr(warehouses.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.Map(types.String())}, warehouses.OpIsNotEmpty),
				}),
			query: `("type" IS NOT NULL AND "type" <> '') OR array_length("scores", 1) IS NULL OR "properties" <> '{}'::jsonb`,
		},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var b strings.Builder
			err := renderExpr(&b, test.expr)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected query %q, got error: %s", test.query, err)
				}
				return
			}
			got := b.String()
			if test.invalid {
				t.Fatalf("expected invalid, got query %q", got)
			}
			if test.query != got {
				t.Fatalf("\nexpected query:  %s\ngot:             %s", test.query, got)
			}
		})
	}
}

// Test_renderExpr_errors ensures that invalid expressions trigger an error
// during SQL rendering.
func Test_renderExpr_errors(t *testing.T) {
	invalidColumn := warehouses.NewBaseExpr(warehouses.Column{Name: "bad name", Type: types.String()}, warehouses.OpIs, "v")
	var b strings.Builder
	if err := renderExpr(&b, invalidColumn); err == nil {
		t.Fatal("expected error for invalid column name")
	}

	badOp := warehouses.NewMultiExpr(warehouses.LogicalOperator(99), []warehouses.Expr{invalidColumn})
	b.Reset()
	if err := renderExpr(&b, badOp); err == nil {
		t.Fatal("expected error for invalid operator")
	}
}
