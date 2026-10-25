// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestStructuralJSONRendering preserves structural equality for every JSON
// kind.
func TestStructuralJSONRendering(t *testing.T) {
	for _, test := range []struct {
		raw     string
		literal string
	}{
		{`"100"`, `'"100"'::jsonb`},
		{`100`, `'100'::jsonb`},
		{`100.0`, `'100.0'::jsonb`},
		{`1e76`, `'1e76'::jsonb`},
		{`true`, `'true'::jsonb`},
		{`false`, `'false'::jsonb`},
		{`null`, `'null'::jsonb`},
		{`[1,"a"]`, `'[1,"a"]'::jsonb`},
		{`{"x":2}`, `'{"x":2}'::jsonb`},
	} {
		for _, negative := range []bool{false, true} {
			t.Run(test.raw+"/"+map[bool]string{false: "positive", true: "negative"}[negative], func(t *testing.T) {
				operator := warehouses.OpIs
				want := `COALESCE("value" = ` + test.literal + `, FALSE)`
				if negative {
					operator, want = warehouses.OpIsNot, "NOT "+want
				}
				expr := warehouses.NewBaseExpr(warehouses.Column{Name: "value", Type: types.JSON()}, operator, json.Value(test.raw))
				var b strings.Builder
				err := renderExpr(&b, expr)
				if err != nil {
					t.Fatalf("expected structural expression, got %v", err)
				}
				if got := b.String(); got != want {
					t.Fatalf("expected %s, got %s", want, got)
				}
			})
		}
	}
}

// TestTypedFilterOrdinarySQLRegression preserves ordinary literals and
// operators.
func TestTypedFilterOrdinarySQLRegression(t *testing.T) {
	for _, test := range []struct {
		name  string
		typ   types.Type
		op    warehouses.Operator
		value any
		want  string
	}{
		{"signed maximum", types.Int(64), warehouses.OpIs, int(9223372036854775807), `"value" = 9223372036854775807`},
		{"unsigned maximum", types.Int(64).Unsigned(), warehouses.OpIs, uint(18446744073709551615), `"value" = 18446744073709551615`},
		{"negative ordinary", types.Int(8), warehouses.OpIsNot, 100, `"value" <> 100`},
		{"float outside decimal range", types.Float(64), warehouses.OpIs, 1e100, `"value" = 1E+100`},
		{"datetime offset", types.DateTime(), warehouses.OpIsAfter, time.Date(2026, 10, 24, 12, 0, 0, 123456789, time.FixedZone("", 7200)), `"value" > '2026-10-24 12:00:00.123456'`},
	} {
		t.Run(test.name, func(t *testing.T) {
			expr := warehouses.NewBaseExpr(warehouses.Column{Name: "value", Type: test.typ}, test.op, test.value)
			var b strings.Builder
			err := renderExpr(&b, expr)
			if err != nil {
				t.Fatalf("expected ordinary expression, got %v", err)
			}
			if got := b.String(); got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

// TestTypedJSONOperandErrors rejects invalid raw operands and comparison modes.
func TestTypedJSONOperandErrors(t *testing.T) {
	column := warehouses.Column{Name: "value", Type: types.JSON()}
	for _, test := range []struct {
		name   string
		op     warehouses.Operator
		values []any
	}{
		{"nil raw", warehouses.OpIs, []any{json.Value(nil)}},
		{"empty raw", warehouses.OpIs, []any{json.Value{}}},
		{"invalid raw", warehouses.OpIs, []any{json.Value("1e")}},
		{"multiple values", warehouses.OpIs, []any{json.Value("1 2")}},
		{"wrong Go type", warehouses.OpIs, []any{"100"}},
		{"object", warehouses.OpIsOneOf, []any{json.Value("{}")}},
		{"array", warehouses.OpIsOneOf, []any{json.Value("[]")}},
		{"Boolean equality", warehouses.OpIs, []any{json.Value("false")}},
		{"Null contains", warehouses.OpContains, []any{json.Value("null")}},
		{"Number prefix", warehouses.OpStartsWith, []any{json.Value("100")}},
		{"mixed bounds", warehouses.OpIsBetween, []any{json.Value("100"), json.Value(`"100"`)}},
		{"excessive precision", warehouses.OpIs, []any{json.Value("1e76")}},
		{"excessive scale", warehouses.OpIs, []any{json.Value("1e-38")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var b strings.Builder
			err := renderExpr(&b, newJSONFilterExpr(column, test.op, test.values...))
			if err != nil {
				return
			}
			t.Fatalf("expected operand error, got %s", b.String())
		})
	}
	for _, mode := range []warehouses.JSONComparisonMode{warehouses.JSONStructuralComparison, 2} {
		t.Run("mode/"+map[warehouses.JSONComparisonMode]string{0: "structural ordering", 2: "unknown"}[mode], func(t *testing.T) {
			expr := warehouses.NewBaseExpr(column, warehouses.OpIsGreaterThan, json.Value("100"))
			expr.JSONComparison = mode
			var b strings.Builder
			err := renderExpr(&b, expr)
			if err != nil {
				return
			}
			t.Fatalf("expected comparison mode error, got %s", b.String())
		})
	}
}

// TestTypedJSONRendering checks SQL against explicit typed predicate snapshots.
func TestTypedJSONRendering(t *testing.T) {
	for _, test := range []struct {
		name   string
		op     warehouses.Operator
		values []any
		want   string
	}{
		{"String numeric text", warehouses.OpIs, []any{json.Value(`"100"`)}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'string' THEN ("value" #>> '{}') COLLATE "C" = '100' ELSE FALSE END, FALSE)`},
		{"Number decimal notation", warehouses.OpIs, []any{json.Value("100.0")}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'number' THEN ("value")::numeric = 100 ELSE FALSE END, FALSE)`},
		{"Number exponent notation", warehouses.OpIs, []any{json.Value("1e2")}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'number' THEN ("value")::numeric = 100 ELSE FALSE END, FALSE)`},
		{"negative Number", warehouses.OpIsNot, []any{json.Value("100")}, `NOT COALESCE(CASE WHEN jsonb_typeof("value") = 'number' THEN ("value")::numeric = 100 ELSE FALSE END, FALSE)`},
		{"mixed membership", warehouses.OpIsOneOf, []any{json.Value("100"), json.Value(`"100"`), json.Value("false"), json.Value("null")}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'string' THEN ("value" #>> '{}') COLLATE "C" IN ('100') WHEN jsonb_typeof("value") = 'number' THEN ("value")::numeric IN (100) WHEN jsonb_typeof("value") = 'boolean' THEN "value" IN ('false'::jsonb) WHEN jsonb_typeof("value") = 'null' THEN TRUE ELSE FALSE END, FALSE)`},
		{"negative mixed membership", warehouses.OpIsNotOneOf, []any{json.Value("100"), json.Value(`"100"`), json.Value("false"), json.Value("null")}, `NOT COALESCE(CASE WHEN jsonb_typeof("value") = 'string' THEN ("value" #>> '{}') COLLATE "C" IN ('100') WHEN jsonb_typeof("value") = 'number' THEN ("value")::numeric IN (100) WHEN jsonb_typeof("value") = 'boolean' THEN "value" IN ('false'::jsonb) WHEN jsonb_typeof("value") = 'null' THEN TRUE ELSE FALSE END, FALSE)`},
		{"Null member", warehouses.OpIsOneOf, []any{json.Value("null")}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'null' THEN TRUE ELSE FALSE END, FALSE)`},
		{"negative Null member", warehouses.OpIsNotOneOf, []any{json.Value("null")}, `NOT COALESCE(CASE WHEN jsonb_typeof("value") = 'null' THEN TRUE ELSE FALSE END, FALSE)`},
		{"false member", warehouses.OpIsOneOf, []any{json.Value("false")}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'boolean' THEN "value" IN ('false'::jsonb) ELSE FALSE END, FALSE)`},
		{"empty String", warehouses.OpIs, []any{json.Value(`""`)}, `COALESCE(CASE WHEN jsonb_typeof("value") = 'string' THEN ("value" #>> '{}') COLLATE "C" = '' ELSE FALSE END, FALSE)`},
	} {
		t.Run(test.name, func(t *testing.T) {
			column := warehouses.Column{Name: "value", Type: types.JSON()}
			expr := newJSONFilterExpr(column, test.op, test.values...)
			var b strings.Builder
			err := renderExpr(&b, expr)
			if err != nil {
				t.Fatalf("expected typed expression, got %v", err)
			}
			if got := b.String(); got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}
