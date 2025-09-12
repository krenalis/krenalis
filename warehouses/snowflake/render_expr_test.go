//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"strings"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

func Test_renderExpr(t *testing.T) {
	tests := []struct {
		expr    meergo.Expr
		query   string
		invalid bool
	}{
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIs, "qwerty"),
			query: `"ID" = 'qwerty'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "age", Type: types.Float(32)}, meergo.OpIsNot, 18.0),
			query: `"AGE" <> 18`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "a", Type: types.Text()}, meergo.OpIs, meergo.Column{Name: "b", Type: types.Text()}),
			query: `"A" = "B"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "values", Type: types.JSON()}, meergo.OpIs, json.Value(`{"foo":2,"boo":true}`)),
			query: `"VALUES" = PARSE_JSON('{"foo":2,"boo":true}')`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(32)}, meergo.OpIsLessThan, 6.5),
			query: `"WEIGHT" < 6.5`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(32)}, meergo.OpIsLessThanOrEqualTo, 6.5),
			query: `"WEIGHT" <= 6.5`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "weight", Type: types.Float(32)}, meergo.OpIsGreaterThan, 6.5),
			query: `"WEIGHT" > 6.5`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Decimal(5, 0)}, meergo.OpIsGreaterThanOrEqualTo, decimal.MustInt(3289)),
			query: `"COUNT" >= 3289`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253621, time.UTC)),
			query: `"TIMESTAMP" < '1900-01-02 23:32:11.940253621'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsOnOrBefore, time.Date(1900, 1, 2, 23, 32, 11, 940253621, time.UTC)),
			query: `"TIMESTAMP" <= '1900-01-02 23:32:11.940253621'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.Date()}, meergo.OpIsAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"TIMESTAMP" > '1900-01-02'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.Date()}, meergo.OpIsOnOrAfter, time.Date(1900, 1, 2, 0, 0, 0, 0, time.UTC)),
			query: `"TIMESTAMP" >= '1900-01-02'`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Float(64)}, meergo.OpIsBetween, 5.0, 10.0),
			query: `"ID" BETWEEN 5 AND 10`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpContains, "foo"),
			query: `CONTAINS("NAME", 'foo')`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpDoesNotContain, "boo"),
			query: `NOT CONTAINS("NAME", 'boo')`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Float(32)}, meergo.OpIsOneOf, 3.5, 9.2, 5.0),
			query: `"ID" IN (3.5,9.2,5)`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "a", Type: types.Float(64)}, meergo.OpIsOneOf, 5.3, 12.6, 9.0),
			query: `"A" IN (5.3,12.6,9)`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpStartsWith, "foo"),
			query: `STARTSWITH("NAME", 'foo')`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "name", Type: types.Text()}, meergo.OpEndsWith, "foo"),
			query: `ENDSWITH("NAME", 'foo')`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "active", Type: types.Boolean()}, meergo.OpIsTrue),
			query: `"ACTIVE"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "active", Type: types.Boolean()}, meergo.OpIsFalse),
			query: `NOT "ACTIVE"`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIsNull),
			query: `"ID" IS NULL`,
		},
		{
			expr:  meergo.NewBaseExpr(meergo.Column{Name: "id", Type: types.Text()}, meergo.OpIsNotNull),
			query: `"ID" IS NOT NULL`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpAnd,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"TIMESTAMP" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpAnd,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"TIMESTAMP" > '1700-01-02 23:32:11' AND "TIMESTAMP" < '1900-01-02 23:32:11'`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpOr,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsGreaterThan, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					meergo.NewBaseExpr(meergo.Column{Name: "timestamp", Type: types.DateTime()}, meergo.OpIsLessThan, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"TIMESTAMP" > '1700-01-02 23:32:11' OR "TIMESTAMP" < '1900-01-02 23:32:11'`,
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
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Decimal(5, 0)}, meergo.OpIs, decimal.MustInt(100)),
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Decimal(5, 0)}, meergo.OpIs, decimal.MustInt(200)),
						meergo.NewBaseExpr(meergo.Column{Name: "count", Type: types.Decimal(5, 0)}, meergo.OpIs, decimal.MustInt(300)),
					}),
				}),
			query: `("ID" = 'abc_42' OR "ID" = 'abc_50' OR "ID" = 'abc_60') AND ("COUNT" = 100 OR "COUNT" = 200 OR "COUNT" = 300)`,
		},
		{
			expr: meergo.NewMultiExpr(
				meergo.OpOr,
				[]meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "type", Type: types.Text(), Nullable: true}, meergo.OpIsNotEmpty),
					// meergo.NewBaseExpr(meergo.Column{Name: "properties", Type: types.JSON()}, meergo.OpIsEmpty), // See issue https://github.com/meergo/meergo/issues/1804.
					meergo.NewBaseExpr(meergo.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, meergo.OpIsEmpty),
					meergo.NewBaseExpr(meergo.Column{Name: "properties", Type: types.Map(types.Text())}, meergo.OpIsNotEmpty),
				}),
			query: `("TYPE" IS NOT NULL AND "TYPE" <> '') OR ("SCORES" IS NULL OR "SCORES" = ARRAY_CONSTRUCT()) OR "PROPERTIES" <> OBJECT_CONSTRUCT()`,
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
