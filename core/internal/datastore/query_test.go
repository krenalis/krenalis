// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// TestConvertWhereSimple tests convertWhere with a single condition.
func TestConvertWhereSimple(t *testing.T) {
	column := warehouses.Column{Name: "a", Type: types.Int(32)}
	columns := map[string]warehouses.Column{
		"a": column,
	}
	where := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
		},
	}
	got, err := convertWhere(where, columns)
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(column, warehouses.OpIs, 1),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereMultiple tests convertWhere with multiple conditions.
func TestConvertWhereMultiple(t *testing.T) {
	colA := warehouses.Column{Name: "a", Type: types.Int(32)}
	colBC := warehouses.Column{Name: "b_c", Type: types.Int(32)}
	columns := map[string]warehouses.Column{
		"a":   colA,
		"b.c": colBC,
	}
	where := &state.Where{
		Logical: state.OpOr,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpIsGreaterThan, Values: []any{5}},
			{Property: []string{"b", "c"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{10}},
		},
	}
	got, err := convertWhere(where, columns)
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
		warehouses.NewBaseExpr(colA, warehouses.OpIsGreaterThan, 5),
		warehouses.NewBaseExpr(colBC, warehouses.OpIsLessThanOrEqualTo, 10),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereExistsOperators tests convertWhere with exists operators.
func TestConvertWhereExistsOperators(t *testing.T) {
	colA := warehouses.Column{Name: "a", Type: types.Int(32)}
	colBC := warehouses.Column{Name: "b_c", Type: types.Int(32)}
	colBD := warehouses.Column{Name: "b_d", Type: types.Text()}
	colEF := warehouses.Column{Name: "e_f", Type: types.Boolean()}
	colEG := warehouses.Column{Name: "e_g", Type: types.Text()}
	columns := map[string]warehouses.Column{
		"a":   colA,
		"b.c": colBC,
		"b.d": colBD,
		"e.f": colEF,
		"e.g": colEG,
	}
	where := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpExists},
			{Property: []string{"b"}, Operator: state.OpDoesNotExist},
			{Property: []string{"e"}, Operator: state.OpExists},
		},
	}
	got, err := convertWhere(where, columns)
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(colA, warehouses.OpIsNotNull),
		warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
			warehouses.NewBaseExpr(colBC, warehouses.OpIsNull),
			warehouses.NewBaseExpr(colBD, warehouses.OpIsNull),
		}),
		warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
			warehouses.NewBaseExpr(colEF, warehouses.OpIsNotNull),
			warehouses.NewBaseExpr(colEG, warehouses.OpIsNotNull),
		}),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereUnknownProperty tests convertWhere with an unknown property
// path.
func TestConvertWhereUnknownProperty(t *testing.T) {
	where := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
		},
	}
	_, err := convertWhere(where, map[string]warehouses.Column{})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
}
