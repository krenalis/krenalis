//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// Test_renderExpr converts various expressions to SQL and compares with the
// expected query strings.
func Test_renderExpr(t *testing.T) {
	tests := []struct {
		expr    meergo.Expr
		query   string
		invalid bool
	}{
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIs, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "ip_addr", Type: types.Inet()}, meergo.OpIs, "127.0.0.1"),
			query: `"ip_addr" = '127.0.0.1'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "age", Type: types.Int(32)}, meergo.OpIsNot, 18),
			query: `"age" <> 18`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "a", Type: types.Text()}, meergo.OpIs, meergo.Column{Name: "b", Type: types.Text()}),
			query: `"a" = "b"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(32)}, meergo.OpIsLessThan, 6.5),
			query: `"weight" < 6.5`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(32)}, meergo.OpIsLessThanOrEqualTo, 6.5),
			query: `"weight" <= 6.5`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(64)}, meergo.OpIsGreaterThan, 6.5092373641509225),
			query: `"weight" > 6.5092373641509225`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Decimal(5, 0)}, meergo.OpIsGreaterThanOrEqualTo, decimal.MustInt(3289)),
			query: `"count" >= 3289`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" < '1900-01-02 23:32:11.940253'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsOnOrBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" <= '1900-01-02 23:32:11.940253'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.Date()}, meergo.OpIsAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"timestamp" > '1900-01-02'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.Date()}, meergo.OpIsOnOrAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"timestamp" >= '1900-01-02'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Int(32)}, meergo.OpIsBetween, 5, 10),
			query: `"id" BETWEEN 5 AND 10`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpContains, "foo"),
			query: `POSITION('foo' IN "name") > 0`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpDoesNotContain, "boo"),
			query: `POSITION('boo' IN "name") = 0`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "values", Type: types.Array(types.Int(32))}, meergo.OpContains, 5),
			query: `5 = ANY("values")`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "values", Type: types.Array(types.Int(32))}, meergo.OpDoesNotContain, 7),
			query: `NOT (7 = ANY("values"))`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Int(32)}, meergo.OpIsOneOf, 3, 9, 5),
			query: `"id" IN (3,9,5)`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "a", Type: types.Float(64)}, meergo.OpIsOneOf, 5.3, 12.6, 9.0),
			query: `"a" IN (5.3,12.6,9)`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpStartsWith, "foo"),
			query: `LEFT("name", LENGTH('foo')) = 'foo'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpEndsWith, "foo"),
			query: `RIGHT("name", LENGTH('foo')) = 'foo'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "active", Type: types.Boolean()}, meergo.OpIsTrue),
			query: `"active"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "active", Type: types.Boolean()}, meergo.OpIsFalse),
			query: `NOT "active"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIsNull),
			query: `"id" IS NULL`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIsNotNull),
			query: `"id" IS NOT NULL`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpAnd,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"timestamp" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpAnd,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' AND "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpOr,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' OR "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpAnd,
				[]meergo.Expr{
					meergo.NewMultiExpr(meergo.OpOr, []meergo.Expr{
						meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIs, "abc_42"),
						meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIs, "abc_50"),
						meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIs, "abc_60"),
					}),
					meergo.NewMultiExpr(meergo.OpOr, []meergo.Expr{
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Int(32)}, meergo.OpIs, 100),
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Int(32)}, meergo.OpIs, 200),
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Int(32)}, meergo.OpIs, 300),
					}),
				}),
			query: `("id" = 'abc_42' OR "id" = 'abc_50' OR "id" = 'abc_60') AND ("count" = 100 OR "count" = 200 OR "count" = 300)`,
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
	invalidColumn := meergo.NewBaseExpr(meergo.Column{Name: "bad name", Type: types.Text()}, meergo.OpIs, "v")
	var b strings.Builder
	if err := renderExpr(&b, invalidColumn); err == nil {
		t.Fatal("expected error for invalid column name")
	}

	badOp := meergo.NewMultiExpr(meergo.LogicalOperator(99), []meergo.Expr{invalidColumn})
	b.Reset()
	if err := renderExpr(&b, badOp); err == nil {
		t.Fatal("expected error for invalid operator")
	}
}
