// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"

	"github.com/google/go-cmp/cmp"
)

// Test_Filter_JSON verifies the recursive filter JSON representation.
func Test_Filter_JSON(t *testing.T) {

	data := []byte(`{"operator":"and","rules":[` +
		`{"property":"a","operator":"is","values":["5"]},` +
		`{"operator":"or","rules":[` +
		`{"property":"b","operator":"is null"},` +
		`{"property":"c","operator":"is one of","values":["x","y"]}` +
		`]}` +
		`]}`)
	var filter Filter
	err := json.Unmarshal(data, &filter)
	if err != nil {
		t.Fatalf("unexpected error %q", err)
	}
	expected := Filter{
		Operator: OpAnd,
		Rules: []FilterRule{
			&FilterCondition{Property: "a", Operator: OpIs, Values: []string{"5"}},
			&Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsNull},
					&FilterCondition{Property: "c", Operator: OpIsOneOf, Values: []string{"x", "y"}},
				},
			},
		},
	}
	if diff := cmp.Diff(&expected, &filter); diff != "" {
		t.Fatalf("unexpected filter (-want +got):\n%s", diff)
	}
	got, err := json.Marshal(&filter)
	if err != nil {
		t.Fatalf("unexpected error %q", err)
	}
	if string(got) != string(data) {
		t.Fatalf("expected %s, got %s", data, got)
	}

}

// Test_Filter_MarshalJSONDoesNotValidateSemantics verifies that JSON encoding
// does not reject filters solely because they are semantically invalid.
func Test_Filter_MarshalJSONDoesNotValidateSemantics(t *testing.T) {

	filter := Filter{
		Operator: "invalid",
		Rules: []FilterRule{
			&Filter{Operator: "invalid"},
			&FilterCondition{Operator: "invalid", Values: []string{}},
		},
	}
	got, err := json.Marshal(filter)
	if err != nil {
		t.Fatalf("unexpected marshal error %q", err)
	}
	expected := `{"operator":"invalid","rules":[{"operator":"invalid","rules":[]},` +
		`{"property":"","operator":"invalid","values":[]}]}`
	if string(got) != expected {
		t.Fatalf("expected %s, got %s", expected, got)
	}

}

// Test_Filter_MarshalJSONRejectsInvalidStrings verifies that JSON encoding
// returns an error for invalid UTF-8 in filter strings.
func Test_Filter_MarshalJSONRejectsInvalidStrings(t *testing.T) {

	invalid := string([]byte{0xff})
	tests := []struct {
		name   string
		filter Filter
	}{
		{
			name:   "logical operator",
			filter: Filter{Operator: FilterLogical(invalid)},
		},
		{
			name: "property",
			filter: Filter{Rules: []FilterRule{
				&FilterCondition{Property: invalid},
			}},
		},
		{
			name: "condition operator",
			filter: Filter{Rules: []FilterRule{
				&FilterCondition{Operator: FilterOperator(invalid)},
			}},
		},
		{
			name: "condition value",
			filter: Filter{Rules: []FilterRule{
				&FilterCondition{Values: []string{invalid}},
			}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := json.Marshal(&test.filter)
			if err != nil {
				return
			}
			t.Fatal("expected error, got no error")
		})
	}

}

// Test_Filter_MarshalJSONRejectsNilReceivers verifies that MarshalJSON returns
// an error when called directly on a nil receiver.
func Test_Filter_MarshalJSONRejectsNilReceivers(t *testing.T) {

	var filter *Filter
	_, err := filter.MarshalJSON()
	if err != nil {
		if err.Error() != "filter cannot be nil" {
			t.Fatalf("expected filter error %q, got %q", "filter cannot be nil", err)
		}
	}
	if err == nil {
		t.Fatal("expected filter error, got no error")
	}

	var condition *FilterCondition
	_, err = condition.MarshalJSON()
	if err != nil {
		if err.Error() != "filter condition cannot be nil" {
			t.Fatalf("expected condition error %q, got %q", "filter condition cannot be nil", err)
		}
	}
	if err == nil {
		t.Fatal("expected condition error, got no error")
	}

}

// Test_Filter_MarshalJSONRejectsNilRules verifies that JSON encoding returns
// an error for nil rules.
func Test_Filter_MarshalJSONRejectsNilRules(t *testing.T) {

	tests := []struct {
		name     string
		rule     FilterRule
		expected string
	}{
		{"interface", nil, "filter rule cannot be nil"},
		{"group", (*Filter)(nil), "filter group cannot be nil"},
		{"condition", (*FilterCondition)(nil), "filter condition cannot be nil"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter := Filter{Operator: OpAnd, Rules: []FilterRule{test.rule}}
			_, err := json.Marshal(&filter)
			if err != nil {
				if !strings.Contains(err.Error(), test.expected) {
					t.Fatalf("expected error %q, got %q", test.expected, err)
				}
				return
			}
			t.Fatal("expected error, got no error")
		})
	}

}

// Test_Filter_MarshalJSONValues verifies the JSON representation of nil,
// empty, and non-empty condition values.
func Test_Filter_MarshalJSONValues(t *testing.T) {

	tests := []struct {
		name      string
		condition *FilterCondition
		expected  string
	}{
		{
			name:      "nil",
			condition: &FilterCondition{Property: "a", Operator: OpIsNull},
			expected:  `{"property":"a","operator":"is null"}`,
		},
		{
			name:      "empty",
			condition: &FilterCondition{Property: "a", Operator: OpIsNull, Values: []string{}},
			expected:  `{"property":"a","operator":"is null","values":[]}`,
		},
		{
			name:      "non-empty",
			condition: &FilterCondition{Property: "a", Operator: OpIs, Values: []string{"5"}},
			expected:  `{"property":"a","operator":"is","values":["5"]}`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(test.condition)
			if err != nil {
				t.Fatalf("unexpected condition marshal error %q", err)
			}
			if string(got) != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, got)
			}

			filter := &Filter{Operator: OpAnd, Rules: []FilterRule{test.condition}}
			got, err = json.Marshal(filter)
			if err != nil {
				t.Fatalf("unexpected filter marshal error %q", err)
			}
			expectedFilter := `{"operator":"and","rules":[` + test.expected + `]}`
			if string(got) != expectedFilter {
				t.Fatalf("expected %s, got %s", expectedFilter, got)
			}

			var encoded bytes.Buffer
			err = json.Encode(&encoded, filter)
			if err != nil {
				t.Fatalf("unexpected encode error %q", err)
			}
			if encoded.String() != expectedFilter {
				t.Fatalf("expected %s, got %s", expectedFilter, encoded.String())
			}
		})
	}

}

// Test_Filter_UnmarshalJSONDoesNotValidateSemantics verifies that JSON
// decoding does not reject filters solely because they are semantically
// invalid.
func Test_Filter_UnmarshalJSONDoesNotValidateSemantics(t *testing.T) {

	data := []byte(`{"operator":"invalid","rules":[` +
		`{"operator":"invalid","rules":[]},` +
		`{"property":"","operator":"invalid","values":[]}]}`)
	var filter Filter
	err := json.Unmarshal(data, &filter)
	if err != nil {
		t.Fatalf("unexpected unmarshal error %q", err)
	}
	expected := Filter{
		Operator: "invalid",
		Rules: []FilterRule{
			&Filter{Operator: "invalid", Rules: []FilterRule{}},
			&FilterCondition{Operator: "invalid", Values: []string{}},
		},
	}
	if diff := cmp.Diff(&expected, &filter); diff != "" {
		t.Fatalf("unexpected filter (-want +got):\n%s", diff)
	}

}

// Test_Filter_UnmarshalJSONRejectsInvalidStructure verifies structural
// validation while decoding filters.
func Test_Filter_UnmarshalJSONRejectsInvalidStructure(t *testing.T) {

	tests := []string{
		`null`,
		`[]`,
		`{"operator":"and"}`,
		`{"rules":[]}`,
		`{"operator":1,"rules":[]}`,
		`{"operator":"and","operator":"or","rules":[]}`,
		`{"operator":"and","rules":[],"rules":[]}`,
		`{"operator":"and","rules":"invalid"}`,
		`{"operator":"and","rules":null}`,
		`{"operator":"and","rules":[],"values":[]}`,
		`{"operator":"and","rules":[true]}`,
		`{"operator":"and","rules":[{"rules":[]}]}`,
		`{"operator":"and","rules":[{"operator":"or","rules":[],"property":"a"}]}`,
		`{"operator":"and","rules":[{}]}`,
		`{"operator":"and","rules":[{"property":"a","values":[]}]}`,
		`{"operator":"and","rules":[{"property":1,"operator":"is"}]}`,
		`{"operator":"and","rules":[{"property":"a","operator":1}]}`,
		`{"operator":"and","rules":[{"property":"a","operator":"is","values":true}]}`,
		`{"operator":"and","rules":[{"property":"a","operator":"is","values":[1]}]}`,
		`{"operator":"and","rules":[{"property":"a","operator":"is","values":[],"values":[]}]}`,
		`{"operator":"and","rules":[{"property":"a","operator":"is","values":["1"],"extra":true}]}`,
		`{"operator":"and","rules":[],"logical":"and"}`,
	}
	for _, data := range tests {
		t.Run(data, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(data), &filter)
			if err == nil {
				t.Fatal("expected error, got no error")
			}
		})
	}
}

// Test_Filter_UnmarshalJSONRejectsRootConditions verifies that decoding a
// condition as a Filter reports that the top-level value must be a group.
func Test_Filter_UnmarshalJSONRejectsRootConditions(t *testing.T) {

	tests := []string{
		`{"property":"a","operator":"is","values":["1"]}`,
		`{"values":true,"property":"a","operator":"is"}`,
	}
	for _, data := range tests {
		t.Run(data, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(data), &filter)
			if err != nil {
				if !strings.Contains(err.Error(), "filter must be a group") {
					t.Fatalf("expected group error, got %q", err)
				}
				return
			}
			t.Fatal("expected error, got no error")
		})
	}

}

// Test_Filter_UnmarshalJSONLimits verifies filter complexity limits while
// decoding JSON.
func Test_Filter_UnmarshalJSONLimits(t *testing.T) {

	filterWithDepth := func(depth int) *Filter {
		var rule FilterRule = &FilterCondition{Property: "a", Operator: OpIs, Values: []string{"1"}}
		for range depth {
			rule = &Filter{Operator: OpAnd, Rules: []FilterRule{rule}}
		}
		return rule.(*Filter)
	}
	filterWithRuleCount := func(ruleCount int) *Filter {
		rules := make([]FilterRule, ruleCount)
		for i := range rules {
			rules[i] = &FilterCondition{Property: "a", Operator: OpIs, Values: []string{"1"}}
		}
		return &Filter{Operator: OpAnd, Rules: rules}
	}
	filterWithNestedRuleCount := func(ruleCount int) *Filter {
		nestedRuleCount := ruleCount / 2
		filter := filterWithRuleCount(ruleCount - nestedRuleCount - 1)
		filter.Rules = append(filter.Rules, filterWithRuleCount(nestedRuleCount))
		return filter
	}

	tests := []struct {
		name     string
		filter   *Filter
		expected error
	}{
		{"maximum depth", filterWithDepth(maxFilterDepth), nil},
		{"excessive depth", filterWithDepth(maxFilterDepth + 1), fmt.Errorf("filter exceeds maximum depth of %d", maxFilterDepth)},
		{"maximum rule count", filterWithRuleCount(maxFilterRuleCount), nil},
		{"excessive rule count", filterWithRuleCount(maxFilterRuleCount + 1), fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)},
		{"maximum nested rule count", filterWithNestedRuleCount(maxFilterRuleCount), nil},
		{"excessive nested rule count", filterWithNestedRuleCount(maxFilterRuleCount + 1), fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.filter)
			if err != nil {
				t.Fatalf("unexpected marshal error %q", err)
			}
			var filter Filter
			err = json.Unmarshal(data, &filter)
			if cause := errors.Unwrap(err); cause != nil {
				err = cause
			}
			if !reflect.DeepEqual(test.expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.expected, test.expected, err, err)
			}
		})
	}

}

// Test_Filter_UnmarshalJSONValues verifies that absent and null condition
// values decode as nil while an empty array remains non-nil.
func Test_Filter_UnmarshalJSONValues(t *testing.T) {

	tests := []struct {
		name     string
		data     string
		expected []string
	}{
		{
			name: "absent",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is null"}]}`,
		},
		{
			name: "null",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is null","values":null}]}`,
		},
		{
			name: "absent with valued operator",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is"}]}`,
		},
		{
			name: "null with valued operator",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is","values":null}]}`,
		},
		{
			name: "empty",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is null","values":[]}]}`,
			expected: []string{},
		},
		{
			name: "non-empty",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is","values":["5"]}]}`,
			expected: []string{"5"},
		},
		{
			name: "null element",
			data: `{"operator":"and","rules":[` +
				`{"property":"a","operator":"is","values":[null]}]}`,
			expected: []string{""},
		},
		{
			name:     "members in a different order",
			data:     `{"rules":[{"values":["5"],"operator":"is","property":"a"}],"operator":"and"}`,
			expected: []string{"5"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var filter Filter
			err := json.Unmarshal([]byte(test.data), &filter)
			if err != nil {
				t.Fatalf("unexpected unmarshal error %q", err)
			}
			if len(filter.Rules) != 1 {
				t.Fatalf("expected one rule, got %d", len(filter.Rules))
			}
			condition, ok := filter.Rules[0].(*FilterCondition)
			if !ok {
				t.Fatalf("expected a filter condition, got %T", filter.Rules[0])
			}
			if diff := cmp.Diff(test.expected, condition.Values); diff != "" {
				t.Fatalf("unexpected values (-want +got):\n%s", diff)
			}
		})
	}

}

// Test_convertFilterToWhere verifies conversion from filters to where
// expressions.
func Test_convertFilterToWhere(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Int(32)},
		{Name: "c", Type: types.Int(8).Unsigned()},
		{Name: "d", Type: types.Float(32)},
		{Name: "e", Type: types.Float(64)},
		{Name: "f", Type: types.Decimal(10, 3)},
		{Name: "g", Type: types.DateTime()},
		{Name: "h", Type: types.Date()},
		{Name: "i", Type: types.Time()},
		{Name: "j", Type: types.Year()},
		{Name: "k", Type: types.UUID()},
		{Name: "l", Type: types.JSON()},
		{Name: "m", Type: types.IP()},
		{Name: "n", Type: types.String(), Nullable: true},
		{Name: "o", Type: types.Array(types.String()), Nullable: true},
		{Name: "p", Type: types.Object([]types.Property{{Name: "x", Type: types.String()}}), Nullable: true},
		{Name: "q", Type: types.Map(types.String()), Nullable: true},
	})

	d := decimal.MustParse("56.19")

	tests := []struct {
		name     string
		filter   Filter
		expected state.Where
	}{
		{
			name: "all property types",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsTrue},
					&FilterCondition{Property: "b", Operator: OpIs, Values: []string{"-105"}},
					&FilterCondition{Property: "c", Operator: OpIsNot, Values: []string{"197"}},
					&FilterCondition{Property: "d", Operator: OpIsGreaterThan, Values: []string{"3.802"}},
					&FilterCondition{Property: "e", Operator: OpIsGreaterThan, Values: []string{"60526.8020184552091647"}},
					&FilterCondition{Property: "e", Operator: OpIsLessThan, Values: []string{"1.7976931348623157e+308"}},
					&FilterCondition{Property: "f", Operator: OpIsLessThanOrEqualTo, Values: []string{"12.956"}},
					&FilterCondition{Property: "g", Operator: OpIsAfter, Values: []string{"2024-09-19T14:36:57.264699103"}},
					&FilterCondition{Property: "h", Operator: OpIsBetween, Values: []string{"2024-09-19", "2025-09-19"}},
					&FilterCondition{Property: "i", Operator: OpIsBefore, Values: []string{"14:36:57.264699103"}},
					&FilterCondition{Property: "j", Operator: OpIsOnOrAfter, Values: []string{"2024"}},
					&FilterCondition{Property: "k", Operator: OpIs, Values: []string{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					&FilterCondition{Property: "l", Operator: OpIs, Values: []string{"foo"}},
					&FilterCondition{Property: "l", Operator: OpIsOneOf, Values: []string{"foo", "56.19", "boo"}},
					&FilterCondition{Property: "l.x", Operator: OpExists},
					&FilterCondition{Property: "m", Operator: OpIsNot, Values: []string{"192.168.1.1"}},
					&FilterCondition{Property: "n", Operator: OpIs, Values: []string{"boo"}},
					&FilterCondition{Property: "o", Operator: OpIsNull},
					&FilterCondition{Property: "o", Operator: OpContains, Values: []string{"boo"}},
					&FilterCondition{Property: "p", Operator: OpIsEmpty},
					&FilterCondition{Property: "p.x", Operator: OpIs, Values: []string{"foo"}},
					&FilterCondition{Property: "l.x.y", Operator: OpExists},
					&FilterCondition{Property: "q", Operator: OpIsNull},
				},
			},
			expected: state.Where{
				Operator: state.OpOr,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIsTrue},
					&state.WhereCondition{Property: []string{"b"}, Operator: state.OpIs, Values: []any{-105}},
					&state.WhereCondition{Property: []string{"c"}, Operator: state.OpIsNot, Values: []any{uint(197)}},
					&state.WhereCondition{Property: []string{"d"}, Operator: state.OpIsGreaterThan, Values: []any{float64(float32(3.802))}},
					&state.WhereCondition{Property: []string{"e"}, Operator: state.OpIsGreaterThan, Values: []any{60526.80201845521}},
					&state.WhereCondition{Property: []string{"e"}, Operator: state.OpIsLessThan, Values: []any{1.7976931348623157e+308}},
					&state.WhereCondition{Property: []string{"f"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{decimal.MustParse("12.956")}},
					&state.WhereCondition{Property: []string{"g"}, Operator: state.OpIsAfter, Values: []any{time.Date(2024, 9, 19, 14, 36, 57, 264699103, time.UTC)}},
					&state.WhereCondition{Property: []string{"h"}, Operator: state.OpIsBetween, Values: []any{time.Date(2024, 9, 19, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 19, 0, 0, 0, 0, time.UTC)}},
					&state.WhereCondition{Property: []string{"i"}, Operator: state.OpIsBefore, Values: []any{time.Date(1970, 1, 1, 14, 36, 57, 264699103, time.UTC)}},
					&state.WhereCondition{Property: []string{"j"}, Operator: state.OpIsOnOrAfter, Values: []any{2024}},
					&state.WhereCondition{Property: []string{"k"}, Operator: state.OpIs, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					&state.WhereCondition{Property: []string{"l"}, Operator: state.OpIs, Values: []any{state.JSONConditionValue{String: "foo"}}},
					&state.WhereCondition{Property: []string{"l"}, Operator: state.OpIsOneOf, Values: []any{
						state.JSONConditionValue{String: "foo"},
						state.JSONConditionValue{String: "56.19", Number: &d},
						state.JSONConditionValue{String: "boo"},
					}},
					&state.WhereCondition{Property: []string{"l", "x"}, Operator: state.OpExists},
					&state.WhereCondition{Property: []string{"m"}, Operator: state.OpIsNot, Values: []any{"192.168.1.1"}},
					&state.WhereCondition{Property: []string{"n"}, Operator: state.OpIs, Values: []any{"boo"}},
					&state.WhereCondition{Property: []string{"o"}, Operator: state.OpIsNull},
					&state.WhereCondition{Property: []string{"o"}, Operator: state.OpContains, Values: []any{"boo"}},
					&state.WhereCondition{Property: []string{"p"}, Operator: state.OpIsEmpty},
					&state.WhereCondition{Property: []string{"p", "x"}, Operator: state.OpIs, Values: []any{"foo"}},
					&state.WhereCondition{Property: []string{"l", "x", "y"}, Operator: state.OpExists},
					&state.WhereCondition{Property: []string{"q"}, Operator: state.OpIsNull},
				},
			},
		},
		{
			name: "and",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsTrue},
				},
			},
			expected: state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIsTrue},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := convertFilterToWhere(&test.filter, schema)
			if diff := cmp.Diff(&test.expected, got); diff != "" {
				t.Fatalf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

// Test_convertWhereToFilter verifies conversion from where expressions to
// filters.
func Test_convertWhereToFilter(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Int(32)},
		{Name: "c", Type: types.Int(8).Unsigned()},
		{Name: "d", Type: types.Float(32)},
		{Name: "e", Type: types.Float(64)},
		{Name: "f", Type: types.Decimal(10, 3)},
		{Name: "g", Type: types.DateTime()},
		{Name: "h", Type: types.Date()},
		{Name: "i", Type: types.Time()},
		{Name: "j", Type: types.Year()},
		{Name: "k", Type: types.UUID()},
		{Name: "l", Type: types.JSON()},
		{Name: "m", Type: types.IP()},
		{Name: "n", Type: types.String(), Nullable: true},
		{Name: "o", Type: types.Array(types.String()), Nullable: true},
		{Name: "p", Type: types.Object([]types.Property{{Name: "x", Type: types.String()}}), Nullable: true},
		{Name: "q", Type: types.Map(types.String()), Nullable: true},
	})

	d := decimal.MustParse("56.19")

	tests := []struct {
		name     string
		where    state.Where
		expected Filter
	}{
		{
			name: "all property types",
			where: state.Where{
				Operator: state.OpOr,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIsTrue},
					&state.WhereCondition{Property: []string{"b"}, Operator: state.OpIs, Values: []any{-105}},
					&state.WhereCondition{Property: []string{"c"}, Operator: state.OpIsNot, Values: []any{uint(197)}},
					&state.WhereCondition{Property: []string{"d"}, Operator: state.OpIsGreaterThan, Values: []any{float64(float32(3.802))}},
					&state.WhereCondition{Property: []string{"e"}, Operator: state.OpIsGreaterThan, Values: []any{60526.80201845521}},
					&state.WhereCondition{Property: []string{"e"}, Operator: state.OpIsLessThan, Values: []any{1.7976931348623157e+308}},
					&state.WhereCondition{Property: []string{"f"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{decimal.MustParse("12.956")}},
					&state.WhereCondition{Property: []string{"g"}, Operator: state.OpIsAfter, Values: []any{time.Date(2024, 9, 19, 14, 36, 57, 264699103, time.UTC)}},
					&state.WhereCondition{Property: []string{"h"}, Operator: state.OpIsBetween, Values: []any{time.Date(2024, 9, 19, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 19, 0, 0, 0, 0, time.UTC)}},
					&state.WhereCondition{Property: []string{"i"}, Operator: state.OpIsBefore, Values: []any{time.Date(1970, 1, 1, 14, 36, 57, 264699103, time.UTC)}},
					&state.WhereCondition{Property: []string{"j"}, Operator: state.OpIsOnOrAfter, Values: []any{2024}},
					&state.WhereCondition{Property: []string{"k"}, Operator: state.OpIs, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					&state.WhereCondition{Property: []string{"l"}, Operator: state.OpIs, Values: []any{state.JSONConditionValue{String: "foo"}}},
					&state.WhereCondition{Property: []string{"l"}, Operator: state.OpIsOneOf, Values: []any{
						state.JSONConditionValue{String: "foo"},
						state.JSONConditionValue{String: "56.19", Number: &d},
						state.JSONConditionValue{String: "boo"},
					}},
					&state.WhereCondition{Property: []string{"l", "x"}, Operator: state.OpExists},
					&state.WhereCondition{Property: []string{"m"}, Operator: state.OpIsNot, Values: []any{"192.168.1.1"}},
					&state.WhereCondition{Property: []string{"n"}, Operator: state.OpIs, Values: []any{"boo"}},
					&state.WhereCondition{Property: []string{"o"}, Operator: state.OpIsNull},
					&state.WhereCondition{Property: []string{"o"}, Operator: state.OpContains, Values: []any{"boo"}},
					&state.WhereCondition{Property: []string{"p"}, Operator: state.OpIsEmpty},
					&state.WhereCondition{Property: []string{"p", "x"}, Operator: state.OpIs, Values: []any{"foo"}},
					&state.WhereCondition{Property: []string{"l", "x", "y"}, Operator: state.OpExists},
					&state.WhereCondition{Property: []string{"q"}, Operator: state.OpIsNull},
				},
			},
			expected: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsTrue},
					&FilterCondition{Property: "b", Operator: OpIs, Values: []string{"-105"}},
					&FilterCondition{Property: "c", Operator: OpIsNot, Values: []string{"197"}},
					&FilterCondition{Property: "d", Operator: OpIsGreaterThan, Values: []string{"3.802"}},
					&FilterCondition{Property: "e", Operator: OpIsGreaterThan, Values: []string{"60526.80201845521"}},
					&FilterCondition{Property: "e", Operator: OpIsLessThan, Values: []string{"1.7976931348623157e+308"}},
					&FilterCondition{Property: "f", Operator: OpIsLessThanOrEqualTo, Values: []string{"12.956"}},
					&FilterCondition{Property: "g", Operator: OpIsAfter, Values: []string{"2024-09-19T14:36:57.264699103"}},
					&FilterCondition{Property: "h", Operator: OpIsBetween, Values: []string{"2024-09-19", "2025-09-19"}},
					&FilterCondition{Property: "i", Operator: OpIsBefore, Values: []string{"14:36:57.264699103"}},
					&FilterCondition{Property: "j", Operator: OpIsOnOrAfter, Values: []string{"2024"}},
					&FilterCondition{Property: "k", Operator: OpIs, Values: []string{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					&FilterCondition{Property: "l", Operator: OpIs, Values: []string{"foo"}},
					&FilterCondition{Property: "l", Operator: OpIsOneOf, Values: []string{"foo", "56.19", "boo"}},
					&FilterCondition{Property: "l.x", Operator: OpExists},
					&FilterCondition{Property: "m", Operator: OpIsNot, Values: []string{"192.168.1.1"}},
					&FilterCondition{Property: "n", Operator: OpIs, Values: []string{"boo"}},
					&FilterCondition{Property: "o", Operator: OpIsNull},
					&FilterCondition{Property: "o", Operator: OpContains, Values: []string{"boo"}},
					&FilterCondition{Property: "p", Operator: OpIsEmpty},
					&FilterCondition{Property: "p.x", Operator: OpIs, Values: []string{"foo"}},
					&FilterCondition{Property: "l.x.y", Operator: OpExists},
					&FilterCondition{Property: "q", Operator: OpIsNull},
				},
			},
		},
		{
			name: "and",
			where: state.Where{
				Operator: state.OpAnd,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIsTrue},
				},
			},
			expected: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsTrue},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := convertWhereToFilter(&test.where, schema)
			if diff := cmp.Diff(&test.expected, got); diff != "" {
				t.Fatalf("unexpected value (-want +got):\n%s", diff)
			}
		})
	}
}

// Test_convertFilterWhereNested verifies conversion between nested filters and
// where expressions.
func Test_convertFilterWhereNested(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Int(32)},
		{Name: "b", Type: types.String()},
		{Name: "c", Type: types.Boolean()},
	})
	filter := &Filter{
		Operator: OpAnd,
		Rules: []FilterRule{
			&FilterCondition{Property: "a", Operator: OpIs, Values: []string{"5"}},
			&Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpContains, Values: []string{"foo"}},
					&FilterCondition{Property: "c", Operator: OpIsTrue},
				},
			},
		},
	}
	where := convertFilterToWhere(filter, schema)
	expected := &state.Where{
		Operator: state.OpAnd,
		Rules: []state.WhereRule{
			&state.WhereCondition{Property: []string{"a"}, Operator: state.OpIs, Values: []any{5}},
			&state.Where{
				Operator: state.OpOr,
				Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"b"}, Operator: state.OpContains, Values: []any{"foo"}},
					&state.WhereCondition{Property: []string{"c"}, Operator: state.OpIsTrue},
				},
			},
		},
	}
	if !expected.Equal(where) {
		t.Fatalf("expected %#v, got %#v", expected, where)
	}
	if diff := cmp.Diff(filter, convertWhereToFilter(where, schema)); diff != "" {
		t.Fatalf("unexpected filter after round trip (-want +got):\n%s", diff)
	}
}

func Test_parseDecimal(t *testing.T) {
	tests := []struct {
		n        string
		expected string
		valid    bool
	}{
		{"123.456", "123.456", true},
		{"-123.456", "-123.456", true},
		{"1e10", "10000000000", true},
		{"-1.23e-4", "-0.000123", true},
		{"123", "123", true},
		{"1.23", "1.23", true},
		{"-1.23", "-1.23", true},
		{"invalid", "", false},
		{"123abc", "", false},
		{"123.45.67", "", false},
		{"", "", false},
		{"123e", "", false},
		{"1.23e10.5", "", false},
		{"NaN", "", false},
		{"+Inf", "", false},
		{"-Inf", "", false},
		{"0", "0", true},
		{".123", "0.123", true},
	}

	for _, test := range tests {
		t.Run(test.n, func(t *testing.T) {
			got, valid := parseDecimal(test.n)
			if !valid {
				if test.valid {
					t.Fatalf("expected valid, got invalid")
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid, got valid")
			}
			if !decimal.MustParse(test.expected).Equal(got) {
				t.Fatalf("expected %s, got %s", test.expected, got)
			}
		})
	}
}

// Test_parseFloat verifies parsing floating-point filter values.
func Test_parseFloat(t *testing.T) {
	tests := []struct {
		n        string
		bitSize  int
		expected float64
		valid    bool
	}{
		{"123.456", 64, 123.456, true},
		{"-123.456", 64, -123.456, true},
		{"1e10", 64, 1e10, true},
		{"-1.23e-4", 64, -1.23e-4, true},
		{"67.0597e+183", 64, 67.0597e+183, true},
		{"123", 32, float64(float32(123)), true},
		{"1.23", 64, 1.23, true},
		{"1.23", 32, float64(float32(1.23)), true},
		{"01.23", 64, 1.23, true},
		{"-01.23", 64, -1.23, true},
		{"00", 64, 0, true},
		{"invalid", 64, 0, false},
		{"123abc", 64, 0, false},
		{"123.45.67", 64, 0, false},
		{"", 64, 0, false},
		{"123e", 64, 0, false},
		{"1.23e10.5", 64, 0, false},
		{"1.23e", 64, 0, false},
		{"1.23e-", 64, 0, false},
		{"NaN", 64, 0, false},
		{"+Inf", 64, 0, false},
		{"-Inf", 64, 0, false},
		{"0", 64, 0, true},
		{"0.", 64, 0, false},
		{".123", 64, 0, false},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%q/%d", test.n, test.bitSize), func(t *testing.T) {
			got, valid := parseFloat(test.n, test.bitSize)
			if !valid {
				if test.valid {
					t.Fatalf("expected valid, got invalid")
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid, got valid")
			}
			if test.expected != got {
				t.Fatalf("expected %g, got %g", test.expected, got)
			}
		})
	}
}

func Test_parseInt(t *testing.T) {
	tests := []struct {
		n        string
		expected int
		valid    bool
	}{
		{"0", 0, true},
		{"123", 123, true},
		{"-123", -123, true},
		{"+123", 123, true},
		{"000", 0, false}, // leading zeros are not allowed.
		{"00123", 0, false},
		{"+0", 0, true},
		{"-0", 0, true},
		{"-00123", 0, false},                          // leading zeros are not allowed.
		{"-9223372036854775809", 0, false},            // negative overflow.
		{"-9223372036854775808", math.MinInt64, true}, // minimum int64.
		{"9223372036854775807", math.MaxInt64, true},  // maximum int64.
		{"9223372036854775808", 0, false},             // positive overflow.
		{"abc", 0, false},
		{"", 0, false},
	}

	for _, test := range tests {
		t.Run(test.n, func(t *testing.T) {
			got, valid := parseInt(test.n)
			if !valid {
				if test.valid {
					t.Fatalf("expected valid, got invalid")
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid, got valid")
			}
			if test.expected != got {
				t.Fatalf("expected %d, got %d", test.expected, got)
			}
		})
	}
}

// Test_parseYear verifies parsing year filter values.
func Test_parseYear(t *testing.T) {
	tests := []struct {
		year     string
		expected int
		valid    bool
	}{
		{"2023", 2023, true},
		{"1999", 1999, true},
		{"99", 99, true},
		{"0", 0, false},               // below the minimum year.
		{"1", types.MinYear, true},    // minimum year.
		{"9999", types.MaxYear, true}, // maximum year.
		{"10000", 0, false},           // above the maximum year.
		{"0000", 0, false},
		{"12345", 0, false},
		{"12a4", 0, false},
		{"-100", 0, false},
		{"+100", 0, false},
		{"123", 123, true},
		{"", 0, false},
	}

	for _, test := range tests {
		t.Run(test.year, func(t *testing.T) {
			got, valid := parseYear(test.year)
			if !valid {
				if test.valid {
					t.Fatalf("expected valid, got invalid")
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid, got valid")
			}
			if test.expected != got {
				t.Fatalf("expected %d, got %d", test.expected, got)
			}
		})
	}
}

// Test_parseUnsigned verifies parsing unsigned integer filter values.
func Test_parseUnsigned(t *testing.T) {
	tests := []struct {
		n        string
		expected uint
		valid    bool
	}{
		{"0", 0, true},
		{"123", 123, true},
		{"000", 0, false}, // leading zeros are not allowed.
		{"00123", 0, false},
		{"4294967295", 4294967295, true}, // maximum uint32 value.
		{"18446744073709551615", 18446744073709551615, true}, // maximum uint64 value.
		{"18446744073709551616", 0, false},                   // overflow.
		{"abc", 0, false},
		{"-123", 0, false},
		{"+123", 0, false},
		{"", 0, false},
	}

	for _, test := range tests {
		t.Run(test.n, func(t *testing.T) {
			got, valid := parseUnsigned(test.n)
			if !valid {
				if test.valid {
					t.Fatalf("expected valid, got invalid")
				}
				return
			}
			if !test.valid {
				t.Fatalf("expected invalid, got valid")
			}
			if test.expected != got {
				t.Fatalf("expected %d, got %d", test.expected, got)
			}
		})
	}
}

// Test_retrieveProperty verifies resolution of filter property paths.
func Test_retrieveProperty(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.String(), Nullable: true},
		{Name: "c", Type: types.JSON()},
		{Name: "d", Type: types.Object([]types.Property{{Name: "x", Type: types.JSON()}})},
	})

	tests := []struct {
		path             string
		expectedProperty types.Property
		expectedPath     string
		err              error
	}{
		{"a", types.Property{Name: "a", Type: types.Boolean()}, "a", nil},
		{"b", types.Property{Name: "b", Type: types.String(), Nullable: true}, "b", nil},
		{"c", types.Property{Name: "c", Type: types.JSON()}, "c", nil},
		{"c.x", types.Property{Name: "c", Type: types.JSON()}, "c", nil},
		{"c.x.y", types.Property{Name: "c", Type: types.JSON()}, "c", nil},
		{"d.x", types.Property{Name: "x", Type: types.JSON()}, "d.x", nil},
		{"d.x.y", types.Property{Name: "x", Type: types.JSON()}, "d.x", nil},
		{"e", types.Property{}, "", types.PathNotExistError{Path: "e"}},
		{"e.x", types.Property{}, "", types.PathNotExistError{Path: "e"}},
		{"d.z", types.Property{}, "", types.PathNotExistError{Path: "d.z"}},
	}

	properties := schema.Properties()

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			gotProperty, gotPath, err := retrieveProperty(properties, test.path)
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no error, got error %q (type %T)", err, err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no error", test.err)
			}
			if !types.Equal(types.Object([]types.Property{test.expectedProperty}), types.Object([]types.Property{gotProperty})) {
				t.Fatalf("expected property %#v, got %#v", test.expectedProperty, gotProperty)
			}
			if test.expectedPath != gotPath {
				t.Fatalf("expected path %q, got %q", test.expectedPath, gotPath)
			}
		})
	}

}

// Test_validateFilter verifies validation of filter structure, operators, and
// values.
func Test_validateFilter(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.String(), Nullable: true},
		{Name: "c", Type: types.Int(8)},
		{Name: "d", Type: types.JSON()},
		{Name: "e", Type: types.DateTime()},
		{Name: "f", Type: types.Date()},
		{Name: "g", Type: types.Time()},
		{Name: "h", Type: types.UUID(), ReadOptional: true},
		{Name: "i", Type: types.Year()},
		{Name: "j", Type: types.IP()},
		{Name: "k", Type: types.Array(types.String()), Nullable: true},
		{Name: "l", Type: types.Array(types.Map(types.String()))},
		{Name: "m", Type: types.Object([]types.Property{{Name: "x", Type: types.String()}}), Nullable: true},
		{Name: "n", Type: types.Map(types.String()), Nullable: true},
		{Name: "o", Type: types.String().WithValues("foo", "boo", ""), Nullable: true},
		{Name: "p", Type: types.String().WithValues("foo"), Nullable: true},
		{Name: "q", Type: types.Array(types.Float(64)), Nullable: true},
	})
	destinationRole := state.Destination
	eventTarget := state.TargetEvent

	tests := []struct {
		name     string
		filter   Filter
		role     *state.Role
		target   *state.Target
		expected []string
		err      error
	}{
		{
			name: "invalid logical operator",
			filter: Filter{
				Operator: "foo",
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: errors.New(`invalid logical operator "foo"`),
		},
		{
			name: "missing rules",
			filter: Filter{
				Operator: OpOr,
			},
			err: errors.New("rules are missing"),
		},
		{
			name: "nested filter",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIs, Values: []string{"foo"}},
					&Filter{
						Operator: OpOr,
						Rules: []FilterRule{
							&FilterCondition{Property: "c", Operator: OpIs, Values: []string{"5"}},
							&FilterCondition{Property: "d.s", Operator: OpExists},
						},
					},
				},
			},
			expected: []string{"b", "c", "d"},
		},
		{
			name: "invalid property path",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a c", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: errors.New("property path is not valid"),
		},
		{
			name: "unknown property path",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "z", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: types.PathNotExistError{Path: "z"},
		},
		{
			name: "is on boolean",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIs, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is" cannot be used with boolean properties`),
		},
		{
			name: "is on object",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "m", Operator: OpIs, Values: []string{"value"}},
				},
			},
			err: errors.New(`operator "is" cannot be used with object properties`),
		},
		{
			name: "is less than on boolean",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsLessThan, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is less than" cannot be used with boolean properties`),
		},
		{
			name: "contains on unsupported array element",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "l", Operator: OpContains, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "contains" cannot be used with array(map) properties`),
		},
		{
			name: "does not contain on boolean",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpDoesNotContain, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "does not contain" cannot be used with boolean properties`),
		},
		{
			name: "is before on boolean",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsBefore, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is before" cannot be used with boolean properties`),
		},
		{
			name: "is true on string",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsTrue},
				},
			},
			err: errors.New(`operator "is true" cannot be used with string properties`),
		},
		{
			name: "is null on non-nullable property",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: OpIsNull},
				},
			},
			err: errors.New(`operator "is null" can only be used with nullable or json properties`),
		},
		{
			name: "exists on required property",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpExists},
				},
			},
			err: errors.New(`operator "exists" can only be used with read-optional properties or with json properties that include a JSON path`),
		},
		{
			name: "does not exist on JSON root",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "d", Operator: OpDoesNotExist},
				},
			},
			err: errors.New(`operator "does not exist" can only be used with read-optional properties or with json properties that include a JSON path`),
		},
		{
			name: "invalid condition operator",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "a", Operator: "boo", Values: []string{"5"}},
				},
			},
			err: errors.New(`operator "boo" is not valid`),
		},
		{
			name: "no values with unary operator",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsNull},
				},
			},
			expected: []string{"b"},
		},
		{
			name: "empty values with unary operator",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsNull, Values: []string{}},
				},
			},
			err: errors.New(`values cannot be used with the operator "is null"`),
		},
		{
			name: "value with unary operator",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsNull, Values: []string{"5"}},
				},
			},
			err: errors.New(`values cannot be used with the operator "is null"`),
		},
		{
			name: "one value with binary operator",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIsBetween, Values: []string{"5"}},
				},
			},
			err: errors.New(`two values must be used with the operator "is between"`),
		},
		{
			name: "missing value",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpContains},
				},
			},
			err: errors.New(`only one value can be used with the operator "contains"`),
		},
		{
			name: "NUL byte in value",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpContains, Values: []string{"foo \x00"}},
				},
			},
			err: errors.New("condition value contains the NUL byte"),
		},
		{
			name: "invalid int value",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "c", Operator: OpIs, Values: []string{"75.0"}},
				},
			},
			err: errors.New(`value of the "c" property is not a valid int`),
		},
		{
			name: "invalid UUID value",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "h", Operator: OpIs, Values: []string{"invalid"}},
				},
			},
			err: errors.New(`value of the "h" property is not a valid uuid`),
		},
		{
			name: "comparison on string with allowed values",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "o", Operator: OpIsLessThan, Values: []string{"none"}},
				},
			},
			err: errors.New(`operator "is less than" cannot be used with string type that has values`),
		},
		{
			name: "value outside allowed values",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "o", Operator: OpIsOneOf, Values: []string{"foo", "moo"}},
				},
			},
			err: errors.New(`value of the "o" property is not among the allowed values`),
		},
		{
			name: "is empty when empty string is not allowed",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "p", Operator: OpIsEmpty},
				},
			},
			err: errors.New(`operator "is empty" cannot be used on string properties that exclude the empty string from allowed values`),
		},
		{
			name: "is not empty when empty string is not allowed",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "p", Operator: OpIsNotEmpty},
				},
			},
			err: errors.New(`operator "is not empty" cannot be used on string properties that exclude the empty string from allowed values`),
		},
		{
			name: "is empty on time",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "g", Operator: OpIsEmpty},
				},
			},
			err: errors.New(`operator "is empty" can only be used with json, string, object, array, and map properties`),
		},
		{
			name: "is not empty on time",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "g", Operator: OpIsNotEmpty},
				},
			},
			err: errors.New(`operator "is not empty" can only be used with json, string, object, array, and map properties`),
		},
		{
			name: "is empty on destination user object",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "m", Operator: OpIsEmpty},
				},
			},
			role: &destinationRole,
			err:  errors.New(`operator "is empty" cannot be used on object properties for destination pipelines on users`),
		},
		{
			name: "is not empty on event object",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "m", Operator: OpIsNotEmpty},
				},
			},
			target: &eventTarget,
			err:    errors.New(`operator "is not empty" cannot be used on object properties for pipelines on events`),
		},
		{
			name: "JSON in destination user filter",
			filter: Filter{
				Operator: OpAnd,
				Rules: []FilterRule{
					&FilterCondition{Property: "d.p", Operator: OpIsNotEmpty},
				},
			},
			role: &destinationRole,
			err:  errors.New(`property "d" has type json, which is not supported in data warehouse exports`),
		},
		{
			name: "all valid operators",
			filter: Filter{
				Operator: OpOr,
				Rules: []FilterRule{
					&FilterCondition{Property: "b", Operator: OpIs, Values: []string{"5"}},
					&FilterCondition{Property: "b", Operator: OpIsNot, Values: []string{"foo"}},
					&FilterCondition{Property: "c", Operator: OpIsLessThan, Values: []string{"12"}},
					&FilterCondition{Property: "c", Operator: OpIsLessThanOrEqualTo, Values: []string{"5"}},
					&FilterCondition{Property: "b", Operator: OpIsGreaterThan, Values: []string{"boo"}},
					&FilterCondition{Property: "b", Operator: OpIsEmpty},
					&FilterCondition{Property: "c", Operator: OpIsGreaterThanOrEqualTo, Values: []string{"23"}},
					&FilterCondition{Property: "c", Operator: OpIsBetween, Values: []string{"10", "20"}},
					&FilterCondition{Property: "c", Operator: OpIsNotBetween, Values: []string{"20", "30"}},
					&FilterCondition{Property: "b", Operator: OpContains, Values: []string{"abc"}},
					&FilterCondition{Property: "b", Operator: OpDoesNotContain, Values: []string{"abc"}},
					&FilterCondition{Property: "c", Operator: OpIsOneOf, Values: []string{"5", "8", "-3"}},
					&FilterCondition{Property: "b", Operator: OpIsNotOneOf, Values: []string{"a", "b", "c"}},
					&FilterCondition{Property: "b", Operator: OpStartsWith, Values: []string{"abc"}},
					&FilterCondition{Property: "b", Operator: OpEndsWith, Values: []string{"abc"}},
					&FilterCondition{Property: "e", Operator: OpIsBefore, Values: []string{"2024-09-10T15:34:31"}},
					&FilterCondition{Property: "f", Operator: OpIsOnOrBefore, Values: []string{"2024-09-10"}},
					&FilterCondition{Property: "g", Operator: OpIsAfter, Values: []string{"15:34:31"}},
					&FilterCondition{Property: "h", Operator: OpIs, Values: []string{"dbfa8339-0d12-4c94-a3fb-569199ae5c8e"}},
					&FilterCondition{Property: "h", Operator: OpExists},
					&FilterCondition{Property: "h", Operator: OpDoesNotExist},
					&FilterCondition{Property: "i", Operator: OpIsBefore, Values: []string{"2024"}},
					&FilterCondition{Property: "j", Operator: OpIs, Values: []string{"192.168.1.1"}},
					&FilterCondition{Property: "d", Operator: OpIsNotEmpty},
					&FilterCondition{Property: "e", Operator: OpIsOnOrAfter, Values: []string{"2024-09-10T15:34:31"}},
					&FilterCondition{Property: "a", Operator: OpIsTrue},
					&FilterCondition{Property: "a", Operator: OpIsFalse},
					&FilterCondition{Property: "b", Operator: OpIsNull},
					&FilterCondition{Property: "b", Operator: OpIsNotNull},
					&FilterCondition{Property: "d", Operator: OpIsEmpty},
					&FilterCondition{Property: "d.s", Operator: OpIsEmpty},
					&FilterCondition{Property: "d.s", Operator: OpExists},
					&FilterCondition{Property: "d.s", Operator: OpDoesNotExist},
					&FilterCondition{Property: "k", Operator: OpIsNull},
					&FilterCondition{Property: "k", Operator: OpContains, Values: []string{"boo"}},
					&FilterCondition{Property: "m", Operator: OpIsNull},
					&FilterCondition{Property: "m", Operator: OpIsEmpty},
					&FilterCondition{Property: "m.x", Operator: OpContains, Values: []string{"abc"}},
					&FilterCondition{Property: "n", Operator: OpIsNotNull},
					&FilterCondition{Property: "n", Operator: OpIsNotEmpty},
					&FilterCondition{Property: "o", Operator: OpIs, Values: []string{"foo"}},
					&FilterCondition{Property: "o", Operator: OpIs, Values: []string{""}},
					&FilterCondition{Property: "o", Operator: OpIsOneOf, Values: []string{"foo", "boo"}},
					&FilterCondition{Property: "o", Operator: OpIsNotNull},
					&FilterCondition{Property: "o", Operator: OpIsEmpty},
					&FilterCondition{Property: "o", Operator: OpIsNotEmpty},
					&FilterCondition{Property: "q", Operator: OpContains, Values: []string{"1.34"}},
				},
			},
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "m", "m.x", "n", "o", "q"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role := state.Source
			if test.role != nil {
				role = *test.role
			}
			target := state.TargetUser
			if test.target != nil {
				target = *test.target
			}
			got, err := validateFilter(&test.filter, schema, role, target)
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no error, got error %q (type %T)", err, err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no error", test.err)
			}
			if !slices.Equal(test.expected, got) {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}

}

// Test_validateFilterLimits verifies filter complexity limits for values built
// directly in Go.
func Test_validateFilterLimits(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "a", Type: types.Int(32)}})
	filterWithDepth := func(depth int) *Filter {
		var rule FilterRule = &FilterCondition{Property: "a", Operator: OpIs, Values: []string{"1"}}
		for range depth {
			rule = &Filter{Operator: OpAnd, Rules: []FilterRule{rule}}
		}
		return rule.(*Filter)
	}
	filterWithRuleCount := func(ruleCount int) *Filter {
		rules := make([]FilterRule, ruleCount)
		for i := range rules {
			rules[i] = &FilterCondition{Property: "a", Operator: OpIs, Values: []string{"1"}}
		}
		return &Filter{Operator: OpAnd, Rules: rules}
	}

	tests := []struct {
		name     string
		filter   *Filter
		expected error
	}{
		{"maximum depth", filterWithDepth(maxFilterDepth), nil},
		{"excessive depth", filterWithDepth(maxFilterDepth + 1), fmt.Errorf("filter exceeds maximum depth of %d", maxFilterDepth)},
		{"maximum rule count", filterWithRuleCount(maxFilterRuleCount), nil},
		{"excessive rule count", filterWithRuleCount(maxFilterRuleCount + 1), fmt.Errorf("filter exceeds maximum rule count of %d", maxFilterRuleCount)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateFilter(test.filter, schema, state.Source, state.TargetUser)
			if !reflect.DeepEqual(test.expected, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.expected, test.expected, err, err)
			}
		})
	}

}

// Test_validateFilterRejectsNilRules verifies that filter validation rejects
// nil rules, including interfaces containing typed nil pointers.
func Test_validateFilterRejectsNilRules(t *testing.T) {

	schema := types.Object([]types.Property{{Name: "a", Type: types.Int(32)}})
	tests := []struct {
		name string
		rule FilterRule
		err  error
	}{
		{"nil interface", nil, errors.New("unsupported filter rule")},
		{"nil group", (*Filter)(nil), errors.New("filter group cannot be nil")},
		{"nil condition", (*FilterCondition)(nil), errors.New("filter condition cannot be nil")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			filter := &Filter{Operator: OpAnd, Rules: []FilterRule{test.rule}}
			_, err := validateFilter(filter, schema, state.Source, state.TargetUser)
			if !reflect.DeepEqual(test.err, err) {
				t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
			}
		})
	}

}
