// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestDecimalExpressionLimits checks decimal operands in direct expressions.
func TestDecimalExpressionLimits(t *testing.T) {
	for _, test := range []struct {
		text  string
		valid bool
	}{
		{"0", true}, {"1.2300", true}, {"1e75", true}, {"1e-37", true},
		{strings.Repeat("9", 76), true},
		{"-" + strings.Repeat("9", 39) + "." + strings.Repeat("9", 37), true},
		{"1e76", false}, {"1e-38", false}, {strings.Repeat("9", 77), false},
		{"1e999999", false}, {"1e-999999", false},
	} {
		for _, typ := range []types.Type{types.Decimal(3, 2), types.Array(types.Decimal(3, 2))} {
			number := decimal.MustParse(test.text)
			column := warehouses.Column{Name: "value", Type: typ}
			operators := []warehouses.Operator{
				warehouses.OpIs, warehouses.OpIsNot, warehouses.OpIsLessThan, warehouses.OpIsLessThanOrEqualTo,
				warehouses.OpIsGreaterThan, warehouses.OpIsGreaterThanOrEqualTo,
				warehouses.OpIsBetween, warehouses.OpIsNotBetween, warehouses.OpIsOneOf, warehouses.OpIsNotOneOf,
			}
			if typ.Kind() == types.ArrayKind {
				operators = []warehouses.Operator{warehouses.OpContains, warehouses.OpDoesNotContain}
			}
			for _, op := range operators {
				t.Run(fmt.Sprintf("%s/%s/%d", typ, test.text, op), func(t *testing.T) {
					values := []any{number}
					if op == warehouses.OpIsBetween || op == warehouses.OpIsNotBetween ||
						op == warehouses.OpIsOneOf || op == warehouses.OpIsNotOneOf {
						values = append([]any{decimal.Decimal{}}, values...)
					}
					expr := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
						warehouses.NewBaseExpr(column, op, values...),
					})
					var b strings.Builder
					err := renderExpr(&b, expr)
					if err != nil {
						if test.valid || err.Error() != "decimal operand exceeds precision 76 or scale 37" {
							t.Fatalf("expected range error only for invalid operands, got %v", err)
						}
						return
					}
					if !test.valid {
						t.Fatalf("expected decimal range error, got SQL %s", b.String())
					}
				})
			}
		}
	}
}

// Test_renderExpr converts various expressions to SQL and compares with the
// expected query strings.
func Test_renderExpr(t *testing.T) {
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "values", Type: types.JSON()}, warehouses.OpIs, json.Value(`{"foo":2,"boo":true}`)),
			query: `COALESCE("values" = '{"foo":2,"boo":true}'::jsonb, FALSE)`,
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
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpExists),
			query: `"id" IS NOT NULL`,
		},
		{
			expr:  warehouses.NewBaseExpr(warehouses.Column{Name: "id", Type: types.String()}, warehouses.OpDoesNotExist),
			query: `"id" IS NULL`,
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
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.JSON()}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "scores", Type: types.Array(types.Int(32)), Nullable: true}, warehouses.OpIsEmpty),
					warehouses.NewBaseExpr(warehouses.Column{Name: "properties", Type: types.Map(types.String())}, warehouses.OpIsNotEmpty),
				}),
			query: `("type" IS NOT NULL AND "type" <> '') OR COALESCE(("properties" IS NULL OR "properties" IN ('null'::jsonb, '""'::jsonb, '[]'::jsonb, '{}'::jsonb)), FALSE) OR array_length("scores", 1) IS NULL OR "properties" <> '{}'::jsonb`,
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
