// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"strings"
	"testing"

	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestCompiledQueryDialect preserves all 26 original Record operators.
func TestCompiledQueryDialect(t *testing.T) {
	dialect := (&Snowflake{}).CompiledQueryDialect()
	for _, malformed := range []warehouses.Expr{
		warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: types.String()}, warehouses.OpIs),
		(*warehouses.BaseExpr)(nil),
	} {
		_, err := dialect.Record(malformed)
		if err == nil {
			t.Fatal("expected malformed prepared expression error, got success")
		}
	}
	covered := map[warehouses.Operator]bool{}
	check := func(t *testing.T, expr warehouses.Expr) {
		t.Helper()
		var original strings.Builder
		err := renderExpr(&original, expr)
		if err != nil {
			t.Fatalf("expected original renderer success, got %v", err)
		}
		actual, err := dialect.Record(expr)
		if err != nil {
			t.Fatalf("expected compiled Record renderer success, got %v", err)
		}
		if original.String() != actual {
			t.Fatalf("expected original Record expression %s, got %s", original.String(), actual)
		}
	}
	for _, c := range warehouseexpr.ValueCases() {
		t.Run(c.Name, func(t *testing.T) {
			check(t, warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: c.Type, Nullable: true}, c.Operator,
				c.Values...))
		})
		covered[c.Operator] = true
	}
	_, cases := warehouseexpr.JSONCases()
	for _, c := range cases {
		t.Run("json/"+c.Name, func(t *testing.T) {
			expr := warehouses.NewBaseExpr(warehouses.Column{Name: "slot", Type: types.JSON(), Nullable: true},
				c.Operator, c.Values...)
			expr.Keys = []string{"literal.key", "quote'key", "back\\slash"}
			expr.JSONComparison = warehouses.JSONTypedComparison
			check(t, expr)
		})
		covered[c.Operator] = true
	}
	if len(covered) != 26 {
		t.Fatalf("expected all 26 existing operators, got %d", len(covered))
	}
	for _, typ := range []types.Type{types.String(), types.DateTime()} {
		parameter, err := dialect.Parameter(2, typ)
		if err != nil || parameter == "" {
			t.Fatalf("expected physical typed parameter, got %q, %v", parameter, err)
		}
	}
	_, err := dialect.Parameter(0, types.String())
	if err == nil {
		t.Fatal("expected invalid parameter index error, got success")
	}
	_, err = dialect.Parameter(1, types.Int(64))
	if err == nil {
		t.Fatal("expected unsupported argument type error, got success")
	}
	_, err = dialect.KnownText(warehouses.Column{Name: "slot", Type: types.String()}, nil)
	if err == nil {
		t.Fatal("expected invalid JSON binding error, got success")
	}
	_, err = (&Snowflake{}).QueryCompiled(t.Context(), warehouses.CompiledQuery{})
	if err == nil {
		t.Fatal("expected envelope error without database access, got success")
	}
}
