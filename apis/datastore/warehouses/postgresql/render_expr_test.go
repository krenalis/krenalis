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

	"chichi/apis/datastore/expr"
	"chichi/connector/types"
)

func Test_renderExpr(t *testing.T) {
	var (
		id        = expr.Column{Name: "id", Type: types.TextKind}
		count     = expr.Column{Name: "count", Type: types.IntKind}
		timestamp = expr.Column{Name: "timestamp", Type: types.DateTimeKind}
		ipAddr    = expr.Column{Name: "ip_addr", Type: types.InetKind}
		weight    = expr.Column{Name: "weight", Type: types.FloatKind}
	)
	cases := []struct {
		expr    expr.Expr
		query   string
		invalid bool
	}{
		{
			expr:  expr.NewBaseExpr(id, expr.OperatorEqual, "qwerty"),
			query: `"id" = 'qwerty'`,
		},
		{
			expr:  expr.NewBaseExpr(ipAddr, expr.OperatorEqual, "127.0.0.1"),
			query: `"ip_addr" = '127.0.0.1'`,
		},
		{
			expr:  expr.NewBaseExpr(weight, expr.OperatorGreaterEqual, 6.5),
			query: `"weight" >= 6.5`,
		},
		{
			expr:  expr.NewBaseExpr(id, expr.OperatorIsNull, nil),
			query: `"id" IS NULL`,
		},
		{
			expr:  expr.NewBaseExpr(id, expr.OperatorIsNotNull, nil),
			query: `"id" IS NOT NULL`,
		},
		{
			expr:  expr.NewBaseExpr(count, expr.OperatorGreaterEqual, 3289),
			query: `"count" >= 3289`,
		},
		{
			expr:  expr.NewBaseExpr(timestamp, expr.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 940253000, time.UTC)),
			query: `"timestamp" < '1900-01-02 23:32:11.940253'`,
		},
		{
			expr: expr.NewMultiExpr(
				expr.LogicalOperatorAnd,
				[]expr.Expr{
					expr.NewBaseExpr(timestamp, expr.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 870000000, time.UTC)),
				}),
			query: `"timestamp" < '1900-01-02 23:32:11.87'`,
		},
		{
			expr: expr.NewMultiExpr(
				expr.LogicalOperatorAnd,
				[]expr.Expr{
					expr.NewBaseExpr(timestamp, expr.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					expr.NewBaseExpr(timestamp, expr.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' AND "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: expr.NewMultiExpr(
				expr.LogicalOperatorOr,
				[]expr.Expr{
					expr.NewBaseExpr(timestamp, expr.OperatorGreater, time.Date(1700, 1, 2, 23, 32, 11, 0, time.UTC)),
					expr.NewBaseExpr(timestamp, expr.OperatorLess, time.Date(1900, 1, 2, 23, 32, 11, 0, time.UTC)),
				}),
			query: `"timestamp" > '1700-01-02 23:32:11' OR "timestamp" < '1900-01-02 23:32:11'`,
		},
		{
			expr: expr.NewMultiExpr(
				expr.LogicalOperatorAnd,
				[]expr.Expr{
					expr.NewMultiExpr(expr.LogicalOperatorOr, []expr.Expr{
						expr.NewBaseExpr(id, expr.OperatorEqual, "abc_42"),
						expr.NewBaseExpr(id, expr.OperatorEqual, "abc_50"),
						expr.NewBaseExpr(id, expr.OperatorEqual, "abc_60"),
					}),
					expr.NewMultiExpr(expr.LogicalOperatorOr, []expr.Expr{
						expr.NewBaseExpr(count, expr.OperatorEqual, 100),
						expr.NewBaseExpr(count, expr.OperatorEqual, 200),
						expr.NewBaseExpr(count, expr.OperatorEqual, 300),
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

func Test_quoteString(t *testing.T) {
	tests := []struct {
		s        string
		expected string
	}{
		{"", "''"},
		{"'", "''''"},      // one single quote
		{"\"", "'\"'"},     // one double quote
		{"''", "''''''"},   // two single quotes
		{"\"\"", "'\"\"'"}, // two double quotes
		{"\x00", "'\x00'"},
		{"hello", "'hello'"},
		{"_+\tè+^", "'_+\tè+^'"},
		{"paul's car", "'paul''s car'"},
		{"hello world", "'hello world'"},
		{"hello\x00world", "'hello\x00world'"},
		{"\x00\x00\x00\x00", "'\x00\x00\x00\x00'"},
		{"\x00hello\x00world\x00", "'\x00hello\x00world\x00'"},
	}
	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			var got strings.Builder
			quoteString(&got, test.s)
			if test.expected != got.String() {
				t.Fatalf("expected %q, got %q", test.expected, got.String())
			}
		})
	}
}
