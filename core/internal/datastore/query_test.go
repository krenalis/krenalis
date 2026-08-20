// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"reflect"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// TestConvertWhereSimple tests convertWhere with a single condition.
func TestConvertWhereSimple(t *testing.T) {
	column := warehouses.Column{Name: "a", Type: types.Int(32)}
	columns := map[string]warehouses.Column{
		"a": column,
	}
	where := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
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
		Operator: state.OpOr,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIsGreaterThan, Values: []any{5}},
			&state.WhereCondition{Property: []string{"b", "c"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{10}},
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

// TestConvertWhereMultipleValues tests convertWhere with a condition containing
// multiple values.
func TestConvertWhereMultipleValues(t *testing.T) {
	column := warehouses.Column{Name: "a", Type: types.Int(32)}
	where := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{
				Property: []string{"a"},
				Operator: state.OpIsBetween,
				Values:   []any{5, 10},
			},
		},
	}
	got, err := convertWhere(where, map[string]warehouses.Column{"a": column})
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(column, warehouses.OpIsBetween, 5, 10),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereNested tests convertWhere with a nested logical group.
func TestConvertWhereNested(t *testing.T) {
	colA := warehouses.Column{Name: "a", Type: types.Int(32)}
	colB := warehouses.Column{Name: "b", Type: types.Int(32)}
	colC := warehouses.Column{Name: "c", Type: types.Int(32)}
	columns := map[string]warehouses.Column{"a": colA, "b": colB, "c": colC}
	where := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
			&state.Where{
				Operator: state.OpOr,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"b"}, Operator: state.OpIsGreaterThan, Values: []any{2}},
					&state.WhereCondition{Property: []string{"c"}, Operator: state.OpIsLessThan, Values: []any{3}},
				},
			},
		},
	}
	got, err := convertWhere(where, columns)
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(colA, warehouses.OpIs, 1),
		warehouses.NewMultiExpr(warehouses.OpOr, []warehouses.Expr{
			warehouses.NewBaseExpr(colB, warehouses.OpIsGreaterThan, 2),
			warehouses.NewBaseExpr(colC, warehouses.OpIsLessThan, 3),
		}),
	})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("expected %#v, got %#v", want, got)
	}
}

// TestConvertWhereExistsOperators tests convertWhere with "exists" and
// "does not exist" operators.
func TestConvertWhereExistsOperators(t *testing.T) {
	colA := warehouses.Column{Name: "a", Type: types.Int(32)}
	colBC := warehouses.Column{Name: "b_c", Type: types.Int(32)}
	colBD := warehouses.Column{Name: "b_d", Type: types.String()}
	colBEF := warehouses.Column{Name: "b_e_f", Type: types.String()}
	colEF := warehouses.Column{Name: "e_f", Type: types.Boolean()}
	colEG := warehouses.Column{Name: "e_g", Type: types.String()}
	colH := warehouses.Column{Name: "h", Type: types.String()}
	columns := map[string]warehouses.Column{
		"a":     colA,
		"b.c":   colBC,
		"b.d":   colBD,
		"b.e.f": colBEF,
		"b2.c":  {Name: "b2_c", Type: types.String()},
		"e.f":   colEF,
		"e.g":   colEG,
		"h":     colH,
	}
	where := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpExists},
			&state.WhereCondition{Property: []string{"h"}, Operator: state.OpDoesNotExist},
			&state.WhereCondition{Property: []string{"b"}, Operator: state.OpDoesNotExist},
			&state.WhereCondition{Property: []string{"e"}, Operator: state.OpExists},
		},
	}
	got, err := convertWhere(where, columns)
	if err != nil {
		t.Fatalf("convertWhere returned error: %v", err)
	}
	want := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
		warehouses.NewBaseExpr(colA, warehouses.OpIsNotNull),
		warehouses.NewBaseExpr(colH, warehouses.OpIsNull),
		warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
			warehouses.NewBaseExpr(colBC, warehouses.OpIsNull),
			warehouses.NewBaseExpr(colBD, warehouses.OpIsNull),
			warehouses.NewBaseExpr(colBEF, warehouses.OpIsNull),
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

// TestConvertWhereRejectsNilValues tests convertWhere with nil expressions and
// rules.
func TestConvertWhereRejectsNilValues(t *testing.T) {
	tests := []struct {
		name     string
		where    *state.Where
		expected error
	}{
		{
			name:     "nil expression",
			expected: errors.New("where expression cannot be nil"),
		},
		{
			name: "nil rule",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules:    []state.WhereRule{nil},
			},
			expected: errors.New("unsupported where rule"),
		},
		{
			name: "nil group",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules:    []state.WhereRule{(*state.Where)(nil)},
			},
			expected: errors.New("where group cannot be nil"),
		},
		{
			name: "nil condition",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules:    []state.WhereRule{(*state.WhereCondition)(nil)},
			},
			expected: errors.New("where condition cannot be nil"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			_, err := convertWhere(test.where, nil)
			if !reflect.DeepEqual(test.expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.expected, test.expected, err, err)
			}

		})
	}
}

// TestConvertWhereRejectsInvalidOperators tests convertWhere with logical and
// condition operators outside their valid ranges.
func TestConvertWhereRejectsInvalidOperators(t *testing.T) {
	tests := []struct {
		name     string
		where    *state.Where
		expected error
	}{
		{
			name: "logical operator below range",
			where: &state.Where{
				Operator: state.WhereLogical(-1),
			},
			expected: errors.New("invalid logical operator -1 in where expression"),
		},
		{
			name: "logical operator above range",
			where: &state.Where{
				Operator: state.WhereLogical(state.OpOr + 1),
			},
			expected: errors.New("invalid logical operator 2 in where expression"),
		},
		{
			name: "condition operator below range",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: state.WhereOperator(-1)},
				},
			},
			expected: errors.New(`invalid operator -1 for property "a" in where expression`),
		},
		{
			name: "condition operator above range",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.WhereCondition{
						Property: []string{"a"},
						Operator: state.WhereOperator(state.OpDoesNotExist + 1),
					},
				},
			},
			expected: errors.New(`invalid operator 26 for property "a" in where expression`),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			_, err := convertWhere(test.where, nil)
			if !reflect.DeepEqual(test.expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.expected, test.expected, err, err)
			}

		})
	}
}

// TestConvertWhereRejectsMissingRules tests convertWhere with empty top-level
// and nested groups.
func TestConvertWhereRejectsMissingRules(t *testing.T) {
	tests := []struct {
		name  string
		where *state.Where
	}{
		{
			name:  "top-level group",
			where: &state.Where{Operator: state.OpAnd},
		},
		{
			name: "nested group",
			where: &state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.Where{Operator: state.OpOr},
				},
			},
		},
	}
	expected := errors.New("where rules are missing")
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := convertWhere(test.where, nil)
			if !reflect.DeepEqual(expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", expected, expected, err, err)
			}
		})
	}
}

// TestConvertWhereRejectsUnsupportedObjectOperator tests convertWhere with an
// operator that cannot be translated for an object property.
func TestConvertWhereRejectsUnsupportedObjectOperator(t *testing.T) {
	where := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIs, Values: []any{1}},
		},
	}
	columns := map[string]warehouses.Column{
		"a.b": {Name: "a_b", Type: types.Int(32)},
	}
	_, err := convertWhere(where, columns)
	expected := errors.New(`operator "Is" cannot be used with object property "a" in where expression`)
	if !reflect.DeepEqual(expected, err) {
		t.Fatalf("expected error %q (type %T), got error %q (type %T)", expected, expected, err, err)
	}
}

// TestConvertWhereUnknownProperty tests convertWhere with an unknown property
// path.
func TestConvertWhereUnknownProperty(t *testing.T) {
	tests := []struct {
		name     string
		operator state.WhereOperator
	}{
		{"exists", state.OpExists},
		{"is", state.OpIs},
	}
	expected := errors.New(`property "a" does not map to any warehouse columns`)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			where := &state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: test.operator},
				},
			}
			_, err := convertWhere(where, map[string]warehouses.Column{})
			if !reflect.DeepEqual(expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", expected, expected, err, err)
			}

		})
	}
}

// TestConvertWhereOperatorValues verifies that state.WhereLogical values and
// state.WhereOperator values through state.OpIsNotNull match their warehouse
// counterparts.
func TestConvertWhereOperatorValues(t *testing.T) {

	logicalOperators := []struct {
		name      string
		state     state.WhereLogical
		warehouse warehouses.LogicalOperator
	}{
		{"and", state.OpAnd, warehouses.OpAnd},
		{"or", state.OpOr, warehouses.OpOr},
	}

	for _, operator := range logicalOperators {
		t.Run("logical/"+operator.name, func(t *testing.T) {

			if got := warehouses.LogicalOperator(operator.state); got != operator.warehouse {
				t.Fatalf("expected warehouse operator %d, got %d", operator.warehouse, got)
			}

		})
	}

	conditionOperators := []struct {
		name      string
		state     state.WhereOperator
		warehouse warehouses.Operator
	}{
		{"is", state.OpIs, warehouses.OpIs},
		{"is not", state.OpIsNot, warehouses.OpIsNot},
		{"is less than", state.OpIsLessThan, warehouses.OpIsLessThan},
		{"is less than or equal to", state.OpIsLessThanOrEqualTo, warehouses.OpIsLessThanOrEqualTo},
		{"is greater than", state.OpIsGreaterThan, warehouses.OpIsGreaterThan},
		{"is greater than or equal to", state.OpIsGreaterThanOrEqualTo, warehouses.OpIsGreaterThanOrEqualTo},
		{"is between", state.OpIsBetween, warehouses.OpIsBetween},
		{"is not between", state.OpIsNotBetween, warehouses.OpIsNotBetween},
		{"contains", state.OpContains, warehouses.OpContains},
		{"does not contain", state.OpDoesNotContain, warehouses.OpDoesNotContain},
		{"is one of", state.OpIsOneOf, warehouses.OpIsOneOf},
		{"is not one of", state.OpIsNotOneOf, warehouses.OpIsNotOneOf},
		{"starts with", state.OpStartsWith, warehouses.OpStartsWith},
		{"ends with", state.OpEndsWith, warehouses.OpEndsWith},
		{"is before", state.OpIsBefore, warehouses.OpIsBefore},
		{"is on or before", state.OpIsOnOrBefore, warehouses.OpIsOnOrBefore},
		{"is after", state.OpIsAfter, warehouses.OpIsAfter},
		{"is on or after", state.OpIsOnOrAfter, warehouses.OpIsOnOrAfter},
		{"is true", state.OpIsTrue, warehouses.OpIsTrue},
		{"is false", state.OpIsFalse, warehouses.OpIsFalse},
		{"is empty", state.OpIsEmpty, warehouses.OpIsEmpty},
		{"is not empty", state.OpIsNotEmpty, warehouses.OpIsNotEmpty},
		{"is null", state.OpIsNull, warehouses.OpIsNull},
		{"is not null", state.OpIsNotNull, warehouses.OpIsNotNull},
	}

	for _, operator := range conditionOperators {
		t.Run("condition/"+operator.name, func(t *testing.T) {

			if got := warehouses.Operator(operator.state); got != operator.warehouse {
				t.Fatalf("expected warehouse operator %d, got %d", operator.warehouse, got)
			}

		})
	}

}
