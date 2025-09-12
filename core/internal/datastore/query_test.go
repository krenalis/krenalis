//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package datastore

import (
	"reflect"
	"testing"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/types"
)

// TestConvertWhereSimple tests convertWhere with a single condition.
func TestConvertWhereSimple(t *testing.T) {
	column := meergo.Column{Name: "a", Type: types.Int(32)}
	columns := map[string]meergo.Column{
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
	want := meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
		meergo.NewBaseExpr(column, meergo.OpIs, 1),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereMultiple tests convertWhere with multiple conditions.
func TestConvertWhereMultiple(t *testing.T) {
	colA := meergo.Column{Name: "a", Type: types.Int(32)}
	colBC := meergo.Column{Name: "b_c", Type: types.Int(32)}
	columns := map[string]meergo.Column{
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
	want := meergo.NewMultiExpr(meergo.OpOr, []meergo.Expr{
		meergo.NewBaseExpr(colA, meergo.OpIsGreaterThan, 5),
		meergo.NewBaseExpr(colBC, meergo.OpIsLessThanOrEqualTo, 10),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereExistsOperators tests convertWhere with exists operators.
func TestConvertWhereExistsOperators(t *testing.T) {
	colA := meergo.Column{Name: "a", Type: types.Int(32)}
	colBC := meergo.Column{Name: "b_c", Type: types.Int(32)}
	colBD := meergo.Column{Name: "b_d", Type: types.Text()}
	colEF := meergo.Column{Name: "e_f", Type: types.Boolean()}
	colEG := meergo.Column{Name: "e_g", Type: types.Text()}
	columns := map[string]meergo.Column{
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
	want := meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
		meergo.NewBaseExpr(colA, meergo.OpIsNotNull),
		meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
			meergo.NewBaseExpr(colBC, meergo.OpIsNull),
			meergo.NewBaseExpr(colBD, meergo.OpIsNull),
		}),
		meergo.NewMultiExpr(meergo.OpOr, []meergo.Expr{
			meergo.NewBaseExpr(colEF, meergo.OpIsNotNull),
			meergo.NewBaseExpr(colEG, meergo.OpIsNotNull),
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
	_, err := convertWhere(where, map[string]meergo.Column{})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
}
