// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package filters

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/warehouseexpr"
	"github.com/krenalis/krenalis/tools/json"
)

// TestAppliesTypedJSONFixtures checks the independent JSON operator truth
// tables.
func TestAppliesTypedJSONFixtures(t *testing.T) {
	values, cases := warehouseexpr.JSONCases()
	for _, test := range cases {
		t.Run(test.Name, func(t *testing.T) {
			where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"value"}, Operator: state.WhereOperator(test.Operator), Values: test.Values},
			}}
			for i, value := range values {
				attributes := map[string]any{}
				if value != nil {
					attributes["value"] = value
				}
				expected := slices.Contains(test.Matches, i)
				if got := Applies(where, attributes); got != expected {
					t.Fatalf("expected fixture %d to match %t, got %t", i, expected, got)
				}
			}
		})
	}
}

// TestAppliesTypedMembershipDecoding checks exact equality after decoding once.
func TestAppliesTypedMembershipDecoding(t *testing.T) {

	large := strings.Repeat("9", 39) + "." + strings.Repeat("1", 37)

	for _, test := range []struct {
		name     string
		value    json.Value
		operands []json.Value
		matches  bool
	}{
		{"String escapes", json.Value(`"fo\u006f"`), []json.Value{json.Value(`"bar"`), json.Value(`"foo"`)}, true},
		{"String operand escapes", json.Value(`"foo"`), []json.Value{json.Value(`"bar"`), json.Value(`"fo\u006f"`)}, true},
		{"String whitespace", json.Value(" \n\t\"foo\"\r "), []json.Value{json.Value(" \"foo\" \n")}, true},
		{"String Unicode", json.Value(`"😀"`), []json.Value{json.Value(`"\ud83d\ude00"`)}, true},
		{"String Unicode normalization", json.Value(`"é"`), []json.Value{json.Value(`"e\u0301"`)}, false},
		{"String backslash", json.Value(`"a\\b\"c\n"`), []json.Value{json.Value(`"a\u005cb\u0022c\u000a"`)}, true},
		{"String buffer reuse", json.Value(`"x"`), []json.Value{json.Value(`"long\u0065r"`), json.Value(`"\u0078"`)}, true},
		{"String empty", json.Value(`""`), []json.Value{json.Value(`null`), json.Value(`""`)}, true},
		{"String is not Number", json.Value(`"100"`), []json.Value{json.Value(`100`)}, false},
		{"String is not Boolean or Null", json.Value(`"false"`), []json.Value{json.Value(`false`), json.Value(`null`)}, false},
		{"Number fraction", json.Value(`100.0`), []json.Value{json.Value(`"100"`), json.Value(`1e2`)}, true},
		{"Number exponent", json.Value(`1e2`), []json.Value{json.Value(`100.00`)}, true},
		{"Number whitespace", json.Value(" 100.00\n"), []json.Value{json.Value("\t1e2 ")}, true},
		{"Number is not String", json.Value(`100`), []json.Value{json.Value(`"100"`)}, false},
		{"Number is not Boolean", json.Value(`0`), []json.Value{json.Value(`false`)}, false},
		{"Number exact integer", json.Value(`9007199254740993`), []json.Value{json.Value(`9007199254740992`)}, false},
		{"Number unsigned limit", json.Value(`18446744073709551615`), []json.Value{json.Value(`184467440737095516150e-1`)}, true},
		{"Number precision 76", json.Value(large), []json.Value{json.Value(strings.ReplaceAll(large, ".", "") + "e-37")}, true},
		{"Number precision 76 unequal", json.Value(large), []json.Value{json.Value(large[:len(large)-1] + "2")}, false},
		{"Number scale 37", json.Value("0." + strings.Repeat("0", 36) + "1"), []json.Value{json.Value(`1e-37`)}, true},
		{"Number zero", json.Value(`-0.000e99`), []json.Value{json.Value(`0`)}, true},
		{"Number unreadable", json.Value(`1e1000000000000000000`), []json.Value{json.Value(`1`)}, false},
		{"Number unreadable operand", json.Value(`1`), []json.Value{json.Value(`1e1000000000000000000`), json.Value(`1.0`)}, true},
		{"Mixed Boolean true", json.Value(`true`), []json.Value{json.Value(`"true"`), json.Value(`1`), json.Value(`false`), json.Value(`null`), json.Value(`true`)}, true},
		{"Mixed Boolean false", json.Value(`false`), []json.Value{json.Value(`"false"`), json.Value(`0`), json.Value(`true`), json.Value(`null`)}, false},
		{"Mixed Null", json.Value(`null`), []json.Value{json.Value(`"null"`), json.Value(`0`), json.Value(`false`), json.Value(`null`)}, true},
		{"Object", json.Value(`{"x":1}`), []json.Value{json.Value(`1`), json.Value(`"x"`), json.Value(`null`)}, false},
		{"Array", json.Value(`[1]`), []json.Value{json.Value(`1`), json.Value(`null`)}, false},
	} {
		for _, negative := range []bool{false, true} {
			t.Run(test.name+"/negative="+strconv.FormatBool(negative), func(t *testing.T) {
				values := make([]any, len(test.operands))
				for i, operand := range test.operands {
					values[i] = operand
				}
				operator, expected := state.OpIsOneOf, test.matches
				if negative {
					operator, expected = state.OpIsNotOneOf, !expected
				}
				where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"value"}, Operator: operator, Values: values},
				}}
				if got := Applies(where, map[string]any{"value": test.value}); got != expected {
					t.Fatalf("expected membership %t, got %t", expected, got)
				}
			})
		}
	}

}

// TestAppliesTypedMembershipPresence distinguishes missing, null, and scalar
// kinds.
func TestAppliesTypedMembershipPresence(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   any
		present bool
		matches bool
	}{
		{"missing", nil, false, false}, {"Go nil present", nil, true, true},
		{"JSON null", json.Value("null"), true, true}, {"Boolean false", json.Value("false"), true, true},
		{"Boolean true", json.Value("true"), true, false}, {"Number zero", json.Value("0"), true, false},
		{"Number", json.Value("100"), true, true}, {"Number fraction", json.Value("100.0"), true, true},
		{"Number exponent", json.Value("1e2"), true, true}, {"String", json.Value(`"100"`), true, true},
		{"String false", json.Value(`"false"`), true, false}, {"String null", json.Value(`"null"`), true, false},
		{"object", json.Value("{}"), true, false}, {"array", json.Value("[100]"), true, false},
	} {
		for _, negative := range []bool{false, true} {
			t.Run(test.name+"/negative="+strconv.FormatBool(negative), func(t *testing.T) {
				operator := state.OpIsOneOf
				expected := test.matches
				if negative {
					operator, expected = state.OpIsNotOneOf, !expected
				}
				where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
					&state.WhereCondition{Property: []string{"value"}, Operator: operator, Values: []any{
						json.Value("100"), json.Value(`"100"`), json.Value("false"), json.Value("null"),
					}},
				}}
				attributes := map[string]any{}
				if test.present {
					attributes["value"] = test.value
				}
				if got := Applies(where, attributes); got != expected {
					t.Fatalf("expected match %t, got %t", expected, got)
				}
			})
		}
	}
}
