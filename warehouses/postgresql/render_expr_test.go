// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/warehouses"
)

// Test_renderExpr converts various expressions to SQL and compares with the
// expected query strings.
func Test_renderExpr(t *testing.T) {
	tests := []struct {
		expr    warehouses.Expr
		query   string
		invalid bool
	}{
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIs, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "ip_addr", Type: types.Inet()}, warehouses.OpIs, "127.0.0.1"),
			query: `"ip_addr" = '127.0.0.1'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "age", Type: types.Int(32)}, warehouses.OpIsNot, 18),
			query: `"age" <> 18`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.Text()}, warehouses.OpIs, warehouses.Column{Name: "b", Type: types.Text()}),
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.Text()}, warehouses.OpContains, "foo"),
			query: `POSITION('foo' IN "name") > 0`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.Text()}, warehouses.OpDoesNotContain, "boo"),
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Int(32)}, warehouses.OpIsOneOf, 3, 9, 5),
			query: `"id" IN (3,9,5)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "a", Type: types.Float(64)}, warehouses.OpIsOneOf, 5.3, 12.6, 9.0),
			query: `"a" IN (5.3,12.6,9)`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.Text()}, warehouses.OpStartsWith, "foo"),
			query: `LEFT("name", LENGTH('foo')) = 'foo'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "name", Type: types.Text()}, warehouses.OpEndsWith, "foo"),
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIsNull),
			query: `"id" IS NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIsNotNull),
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
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIs, "abc_42"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIs, "abc_50"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OpIs, "abc_60"),
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
					warehouses.NewBaseExpr(warehouses.Column{Name: "type", Type: types.Text(), Nullable: true}, warehouses.OpIsNotEmpty),
					// warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsEmpty), // See issue https://github.com/meergo/meergo/issues/1804.
					warehouses.NewBaseExpr(warehouses.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.Map(types.Text())}, warehouses.OpIsNotEmpty),
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
	invalidColumn := warehouses.NewBaseExpr(warehouses.Column{Name: "bad name", Type: types.Text()}, warehouses.OpIs, "v")
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
