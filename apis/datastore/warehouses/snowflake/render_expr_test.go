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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

func Test_renderExpr(t *testing.T) {
	cases := []struct {
		expr    warehouses.Expr
		query   string
		invalid bool
	}{
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorEqual, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.JSON()}, warehouses.OperatorEqual, map[string]any{"foo": 2, "boo": true}),
			query: `"values" = PARSE_JSON('{"boo":true,"foo":2}')`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "weight", Type: types.Float(32)}, warehouses.OperatorGreaterEqual, 6.5),
			query: `"weight" >= 6.5`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorIsNull, nil),
			query: `"id" IS NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorIsNotNull, nil),
			query: `"id" IS NOT NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OperatorGreaterEqual, decimal.NewFromInt(3289)),
			query: `"count" >= 3289`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" < '1900-01-02 23:32:11.940253'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.LogicalOperatorAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"timestamp" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.LogicalOperatorAnd,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' AND "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.LogicalOperatorOr,
				[]warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					warehouses.NewBaseExpr(warehouses.Column{Name: "timestamp", Type: types.DateTime()}, warehouses.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' OR "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: warehouses.NewMultiExpr(
				warehouses.LogicalOperatorAnd,
				[]warehouses.Expr{
					warehouses.NewMultiExpr(warehouses.LogicalOperatorOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorEqual, "abc_42"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorEqual, "abc_50"),
						warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.Text()}, warehouses.OperatorEqual, "abc_60"),
					}),
					warehouses.NewMultiExpr(warehouses.LogicalOperatorOr, []warehouses.Expr{
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OperatorEqual, decimal.NewFromInt(100)),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OperatorEqual, decimal.NewFromInt(200)),
						warehouses.NewBaseExpr(warehouses.Column{Name: "count", Type: types.Decimal(5, 0)}, warehouses.OperatorEqual, decimal.NewFromInt(300)),
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
