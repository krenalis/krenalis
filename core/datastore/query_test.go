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
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

// TestExprFromWhereSimple tests exprFromWhere with a single condition.
func TestExprFromWhereSimple(t *testing.T) {
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
	got, err := exprFromWhere(where, columns)
	if err != nil {
		t.Fatalf("exprFromWhere returned error: %v", err)
	}
	want := meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
		meergo.NewBaseExpr(column, meergo.OpIs, 1),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestExprFromWhereMultiple tests exprFromWhere with multiple conditions.
func TestExprFromWhereMultiple(t *testing.T) {
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
	got, err := exprFromWhere(where, columns)
	if err != nil {
		t.Fatalf("exprFromWhere returned error: %v", err)
	}
	want := meergo.NewMultiExpr(meergo.OpOr, []meergo.Expr{
		meergo.NewBaseExpr(colA, meergo.OpIsGreaterThan, 5),
		meergo.NewBaseExpr(colBC, meergo.OpIsLessThanOrEqualTo, 10),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestExprFromWhereExistsOperators tests exprFromWhere with exists operators.
func TestExprFromWhereExistsOperators(t *testing.T) {
	colA := meergo.Column{Name: "a", Type: types.Int(32)}
	colB := meergo.Column{Name: "b", Type: types.Int(32)}
	columns := map[string]meergo.Column{
		"a": colA,
		"b": colB,
	}
	where := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpExists},
			{Property: []string{"b"}, Operator: state.OpDoesNotExist},
		},
	}
	got, err := exprFromWhere(where, columns)
	if err != nil {
		t.Fatalf("exprFromWhere returned error: %v", err)
	}
	want := meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
		meergo.NewBaseExpr(colA, meergo.OpIsNotNull),
		meergo.NewBaseExpr(colB, meergo.OpIsNull),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestExprFromWhereUnknownProperty tests exprFromWhere with an unknown property path.
func TestExprFromWhereUnknownProperty(t *testing.T) {
	where := &state.Where{
		Logical: state.OpAnd,
		Conditions: []state.WhereCondition{
			{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
		},
	}
	_, err := exprFromWhere(where, map[string]meergo.Column{})
	if err == nil {
		t.Fatalf("expected error, got no error")
	}
}
