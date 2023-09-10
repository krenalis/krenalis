//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"testing"
	"time"

	wh "chichi/apis/datastore/warehouses"
	"chichi/connector/types"

	"github.com/open2b/nuts/decimal"
)

func Test_renderExpr(t *testing.T) {
	var (
		id        = wh.ExprColumn{Name: "id", Type: types.PtText}
		count     = wh.ExprColumn{Name: "count", Type: types.PtDecimal}
		weight    = wh.ExprColumn{Name: "weight", Type: types.PtFloat}
		timestamp = wh.ExprColumn{Name: "timestamp", Type: types.PtDateTime}
		values    = wh.ExprColumn{Name: "values", Type: types.PtJSON}
	)
	cases := []struct {
		expr    wh.Expr
		query   string
		invalid bool
	}{
		{
			expr:  wh.NewBaseExpr(id, wh.OperatorEqual, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  wh.NewBaseExpr(values, wh.OperatorEqual, map[string]any{"foo": 2, "boo": true}),
			query: `"values" = PARSE_JSON('{"boo":true,"foo":2}')`,
		},
		{
			expr:  wh.NewBaseExpr(weight, wh.OperatorGreaterEqual, 6.5),
			query: `"weight" >= 6.5`,
		},
		{
			expr:  wh.NewBaseExpr(id, wh.OperatorIsNull, nil),
			query: `"id" IS NULL`,
		},
		{
			expr:  wh.NewBaseExpr(id, wh.OperatorIsNotNull, nil),
			query: `"id" IS NOT NULL`,
		},
		{
			expr:  wh.NewBaseExpr(count, wh.OperatorGreaterEqual, decimal.Int(3289)),
			query: `"count" >= 3289`,
		},
		{
			expr:  wh.NewBaseExpr(timestamp, wh.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" < '1900-01-02 23:32:11.940253'`,
		},
		{
			expr: wh.NewMultiExpr(
				wh.LogicalOperatorAnd,
				[]wh.Expr{
					wh.NewBaseExpr(timestamp, wh.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"timestamp" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: wh.NewMultiExpr(
				wh.LogicalOperatorAnd,
				[]wh.Expr{
					wh.NewBaseExpr(timestamp, wh.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					wh.NewBaseExpr(timestamp, wh.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' AND "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: wh.NewMultiExpr(
				wh.LogicalOperatorOr,
				[]wh.Expr{
					wh.NewBaseExpr(timestamp, wh.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					wh.NewBaseExpr(timestamp, wh.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' OR "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: wh.NewMultiExpr(
				wh.LogicalOperatorAnd,
				[]wh.Expr{
					wh.NewMultiExpr(wh.LogicalOperatorOr, []wh.Expr{
						wh.NewBaseExpr(id, wh.OperatorEqual, "abc_42"),
						wh.NewBaseExpr(id, wh.OperatorEqual, "abc_50"),
						wh.NewBaseExpr(id, wh.OperatorEqual, "abc_60"),
					}),
					wh.NewMultiExpr(wh.LogicalOperatorOr, []wh.Expr{
						wh.NewBaseExpr(count, wh.OperatorEqual, decimal.Int(100)),
						wh.NewBaseExpr(count, wh.OperatorEqual, decimal.Int(200)),
						wh.NewBaseExpr(count, wh.OperatorEqual, decimal.Int(300)),
					}),
				}),
			query: `("id" = 'abc_42' OR "id" = 'abc_50' OR "id" = 'abc_60') AND ("count" = 100 OR "count" = 200 OR "count" = 300)`,
		},
	}
	for _, cas := range cases {
		t.Run("", func(t *testing.T) {
			gotQuery, gotErr := renderExpr(cas.expr)
			if cas.invalid {
				if gotErr == nil {
					t.Fatalf("expecting invalid, got query %q", gotQuery)
				}
				return
			}
			if gotErr != nil {
				t.Fatalf("expecting query %q, got error: %s", cas.query, gotErr)
			}
			if cas.query != gotQuery {
				t.Fatalf("\nexpecting query:  %s\ngot:              %s", cas.query, gotQuery)
			}
		})
	}
}
