// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// Test_unmarshalWhere verifies decoding of persisted Where values.
func Test_unmarshalWhere(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Int(32)},
		{Name: "b", Type: types.String()},
		{Name: "c", Type: types.Int(8)},
		{Name: "d", Type: types.Float(64)},
		{Name: "e", Type: types.String()},
		{Name: "f", Type: types.Boolean()},
		{Name: "g", Type: types.Int(32).Unsigned()},
		{Name: "h", Type: types.JSON()},
		{Name: "i", Type: types.JSON()},
		{Name: "j", Type: types.DateTime()},
		{Name: "k", Type: types.Date()},
		{Name: "l", Type: types.Time()},
		{Name: "m", Type: types.Decimal(5, 3)},
		{Name: "n", Type: types.UUID()},
		{Name: "o", Type: types.IP()},
		{Name: "p", Type: types.Year()},
		{Name: "q", Type: types.Array(types.String())},
	})

	vDecimalInt := decimal.MustParse("34")
	vDecimalFloat := decimal.MustParse("85.027")
	vDateTime := time.Date(2024, 9, 12, 11, 3, 6, 801586201, time.UTC)
	vDate := time.Date(2024, 9, 12, 0, 0, 0, 0, time.UTC)
	vTime := time.Date(1970, 1, 1, 11, 3, 6, 801586201, time.UTC)

	tests := []struct {
		name      string
		condition string
		expected  WhereCondition
	}{
		{
			name:      "signed integer",
			condition: `{"property":["a"],"operator":"Is","values":[5]}`,
			expected:  WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
		},
		{
			name:      "string",
			condition: `{"property":["b"],"operator":"IsNot","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"b"}, Operator: OpIsNot, Values: []any{"foo"}},
		},
		{
			name:      "signed integer range",
			condition: `{"property":["c"],"operator":"IsBetween","values":[10,20]}`,
			expected:  WhereCondition{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{10, 20}},
		},
		{
			name:      "float",
			condition: `{"property":["d"],"operator":"IsLessThan","values":[34.98]}`,
			expected:  WhereCondition{Property: []string{"d"}, Operator: OpIsLessThan, Values: []any{34.98}},
		},
		{
			name:      "numeric string",
			condition: `{"property":["e"],"operator":"IsGreaterThan","values":["34"]}`,
			expected:  WhereCondition{Property: []string{"e"}, Operator: OpIsGreaterThan, Values: []any{"34"}},
		},
		{
			name:      "boolean unary",
			condition: `{"property":["f"],"operator":"IsTrue"}`,
			expected:  WhereCondition{Property: []string{"f"}, Operator: OpIsTrue},
		},
		{
			name:      "unsigned integer",
			condition: `{"property":["g"],"operator":"IsNotOneOf","values":[1,2,3]}`,
			expected:  WhereCondition{Property: []string{"g"}, Operator: OpIsNotOneOf, Values: []any{uint(1), uint(2), uint(3)}},
		},
		{
			name:      "JSON property",
			condition: `{"property":["h"],"operator":"Is","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"h"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
		},
		{
			name:      "JSON path",
			condition: `{"property":["h","x"],"operator":"Is","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"h", "x"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
		},
		{
			name:      "JSON number",
			condition: `{"property":["i"],"operator":"IsBetween","values":["34"]}`,
			expected:  WhereCondition{Property: []string{"i"}, Operator: OpIsBetween, Values: []any{JSONConditionValue{String: "34", Number: &vDecimalInt}}},
		},
		{
			name:      "datetime",
			condition: `{"property":["j"],"operator":"IsAfter","values":["2024-09-12T11:03:06.801586201Z"]}`,
			expected:  WhereCondition{Property: []string{"j"}, Operator: OpIsAfter, Values: []any{vDateTime}},
		},
		{
			name:      "date",
			condition: `{"property":["k"],"operator":"IsBefore","values":["2024-09-12"]}`,
			expected:  WhereCondition{Property: []string{"k"}, Operator: OpIsBefore, Values: []any{vDate}},
		},
		{
			name:      "time",
			condition: `{"property":["l"],"operator":"IsOnOrBefore","values":["11:03:06.801586201"]}`,
			expected:  WhereCondition{Property: []string{"l"}, Operator: OpIsOnOrBefore, Values: []any{vTime}},
		},
		{
			name:      "decimal",
			condition: `{"property":["m"],"operator":"Is","values":[85.027]}`,
			expected:  WhereCondition{Property: []string{"m"}, Operator: OpIs, Values: []any{vDecimalFloat}},
		},
		{
			name:      "UUID",
			condition: `{"property":["n"],"operator":"IsNot","values":["38d065ab-ca46-4812-a83c-a9712e09c153"]}`,
			expected:  WhereCondition{Property: []string{"n"}, Operator: OpIsNot, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
		},
		{
			name:      "IP",
			condition: `{"property":["o"],"operator":"Is","values":["192.168.1.1"]}`,
			expected:  WhereCondition{Property: []string{"o"}, Operator: OpIs, Values: []any{"192.168.1.1"}},
		},
		{
			name:      "year",
			condition: `{"property":["p"],"operator":"Is","values":[2024]}`,
			expected:  WhereCondition{Property: []string{"p"}, Operator: OpIs, Values: []any{2024}},
		},
		{
			name:      "array element",
			condition: `{"property":["q"],"operator":"Contains","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"q"}, Operator: OpContains, Values: []any{"foo"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := unmarshalWhere([]byte(`{"operator":"And","rules":[`+test.condition+`]}`), schema)
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			expected := &Where{Operator: OpAnd, Rules: []WhereRule{
				&test.expected}}
			if !expected.Equal(got) {
				t.Fatalf("\nexpected %#v\ngot      %#v", expected, got)
			}
		})
	}

	groups := []struct {
		name     string
		where    string
		expected Where
	}{
		{
			name: "and group",
			where: `{"operator":"And","rules":[` +
				`{"property":["a"],"operator":"Is","values":[720347]},` +
				`{"property":["b"],"operator":"IsLessThan","values":["foo boo"]},` +
				`{"property":["c"],"operator":"IsBetween","values":[66,88]}` + `]}`,
			expected: Where{
				Operator: OpAnd,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{720347}},
					&WhereCondition{Property: []string{"b"}, Operator: OpIsLessThan, Values: []any{"foo boo"}},
					&WhereCondition{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{66, 88}},
				},
			},
		},
		{
			name: "or group",
			where: `{"operator":"Or","rules":[` +
				`{"property":["d"],"operator":"Is","values":[90481.26681]},` +
				`{"property":["e"],"operator":"StartsWith","values":["Boo"]},` +
				`{"property":["f"],"operator":"IsFalse"}` + `]}`,
			expected: Where{
				Operator: OpOr,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"d"}, Operator: OpIs, Values: []any{90481.26681}},
					&WhereCondition{Property: []string{"e"}, Operator: OpStartsWith, Values: []any{"Boo"}},
					&WhereCondition{Property: []string{"f"}, Operator: OpIsFalse},
				},
			},
		},
		{
			name: "nested group",
			where: `{"operator":"And","rules":[` +
				`{"property":["a"],"operator":"Is","values":[5]},` +
				`{"operator":"Or","rules":[` +
				`{"property":["b"],"operator":"Contains","values":["foo"]},` +
				`{"property":["f"],"operator":"IsTrue"}` +
				`]}` +
				`]}`,
			expected: Where{
				Operator: OpAnd,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
					&Where{
						Operator: OpOr,
						Rules: []WhereRule{
							&WhereCondition{Property: []string{"b"}, Operator: OpContains, Values: []any{"foo"}},
							&WhereCondition{Property: []string{"f"}, Operator: OpIsTrue},
						},
					},
				},
			},
		},
	}

	for _, test := range groups {
		t.Run(test.name, func(t *testing.T) {
			got, err := unmarshalWhere([]byte(test.where), schema)
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			if !test.expected.Equal(got) {
				t.Fatalf("\nexpected %#v\ngot      %#v", &test.expected, got)
			}
		})
	}

	invalid := []struct {
		name          string
		where         string
		errorContains string
	}{
		{name: "invalid JSON", where: `{`, errorContains: "unexpected EOF"},
		{name: "root is not an object", where: `[]`, errorContains: "where group must be an object"},
		{name: "missing operator", where: `{"rules":[]}`, errorContains: "where group must contain operator and rules"},
		{name: "missing rules", where: `{"operator":"And"}`, errorContains: "where group must contain operator and rules"},
		{
			name:          "rules is not an array",
			where:         `{"operator":"And","rules":{}}`,
			errorContains: "where rules must be an array",
		},
		{
			name:          "invalid logical operator",
			where:         `{"operator":"Invalid","rules":[]}`,
			errorContains: "invalid logical operator",
		},
		{
			name:          "rule is neither group nor condition",
			where:         `{"operator":"And","rules":[{}]}`,
			errorContains: "where rule must be either a group or a condition",
		},
		{
			name:          "rule is both group and condition",
			where:         `{"operator":"And","rules":[{"property":["a"],"operator":"Is","values":[5],"rules":[]}]}`,
			errorContains: "where rule must be either a group or a condition",
		},
		{
			name:          "unknown property",
			where:         `{"operator":"And","rules":[{"property":["unknown"],"operator":"Is","values":[5]}]}`,
			errorContains: `property path "unknown" does not exist`,
		},
		{
			name:          "invalid condition operator",
			where:         `{"operator":"And","rules":[{"property":["a"],"operator":"Invalid","values":[5]}]}`,
			errorContains: "invalid operator",
		},
		{
			name:          "wrong string value type",
			where:         `{"operator":"And","rules":[{"property":["b"],"operator":"Is","values":[5]}]}`,
			errorContains: "where condition value must be a string",
		},
		{
			name:          "wrong integer value type",
			where:         `{"operator":"And","rules":[{"property":["a"],"operator":"Is","values":["foo"]}]}`,
			errorContains: "invalid syntax",
		},
		{
			name:          "wrong array element type",
			where:         `{"operator":"And","rules":[{"property":["q"],"operator":"Contains","values":[5]}]}`,
			errorContains: "where condition value must be a string",
		},
		{
			name:          "invalid datetime",
			where:         `{"operator":"And","rules":[{"property":["j"],"operator":"IsAfter","values":["invalid"]}]}`,
			errorContains: "cannot parse",
		},
		{
			name:          "invalid date",
			where:         `{"operator":"And","rules":[{"property":["k"],"operator":"IsBefore","values":["invalid"]}]}`,
			errorContains: "cannot parse",
		},
		{
			name:          "invalid time",
			where:         `{"operator":"And","rules":[{"property":["l"],"operator":"IsBefore","values":["invalid"]}]}`,
			errorContains: "cannot parse",
		},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			_, err := unmarshalWhere([]byte(test.where), schema)
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), test.errorContains) {
				t.Fatalf("expected error containing %q, got %q", test.errorContains, err)
			}
		})
	}

	t.Run("legacy group", func(t *testing.T) {
		_, err := unmarshalWhere([]byte(`{"logical":"And","conditions":[]}`), schema)
		if err == nil {
			t.Fatal("expected legacy where representation to be rejected")
		}
		if got, expected := err.Error(), "where group must contain operator and rules"; got != expected {
			t.Fatalf("expected error %q, got %q", expected, got)
		}
	})

}

// Test_Where_Equal verifies equality of Where values, including nested
// groups.
func Test_Where_Equal(t *testing.T) {

	tests := []struct {
		name   string
		w1, w2 *Where
		equal  bool
	}{
		{
			name:  "both nil",
			w1:    nil,
			w2:    nil,
			equal: true,
		},
		{
			name: "other nil",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull}},
			},
			w2:    nil,
			equal: false,
		},
		{
			name: "receiver nil",
			w1:   nil,
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull}},
			},
			equal: false,
		},
		{
			name: "equal unary conditions",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull}},
			},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull}},
			},
			equal: true,
		},
		{
			name: "equal scalar values",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"b"}, Operator: OpIs, Values: []any{5}}},
			},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"b"}, Operator: OpIs, Values: []any{5}}},
			},
			equal: true,
		},
		{
			name: "equal time values",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}},
			},
			equal: true,
		},
		{
			name: "equal decimal values",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}},
			},
			equal: true,
		},
		{
			name: "different logical operator",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsFalse}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsFalse}},
			},
			equal: false,
		},
		{
			name: "different property",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsFalse}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"b"}, Operator: OpIsFalse}},
			},
			equal: false,
		},
		{
			name: "different condition operator",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsFalse}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsTrue}},
			},
			equal: false,
		},
		{
			name: "different value count",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"b"}, Operator: OpIs, Values: []any{"foo"}}},
			},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"b"}, Operator: OpIs, Values: []any{"foo", "boo"}}},
			},
			equal: false,
		},
		{
			name: "different rule count",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull}},
			},
			w2:    &Where{Operator: OpAnd},
			equal: false,
		},
		{
			name: "different rule types",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&Where{Operator: OpOr},
			}},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&WhereCondition{Property: []string{"a"}, Operator: OpIsNull},
			}},
			equal: false,
		},
		{
			name:  "nil rule",
			w1:    &Where{Operator: OpAnd, Rules: []WhereRule{nil}},
			w2:    &Where{Operator: OpAnd, Rules: []WhereRule{nil}},
			equal: false,
		},
		{
			name: "different time value",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 15, 45, 12, 0, time.UTC)}}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}},
			},
			equal: false,
		},
		{
			name: "different decimal value",
			w1: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("39.05")}}},
			},
			w2: &Where{Operator: OpOr, Rules: []WhereRule{
				&WhereCondition{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}},
			},
			equal: false,
		},
		{
			name: "equal nested groups",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&Where{Operator: OpOr, Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{1}},
				}},
			}},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&Where{Operator: OpOr, Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{1}},
				}},
			}},
			equal: true,
		},
		{
			name: "different nested condition",
			w1: &Where{Operator: OpAnd, Rules: []WhereRule{
				&Where{Operator: OpOr, Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{1}},
				}},
			}},
			w2: &Where{Operator: OpAnd, Rules: []WhereRule{
				&Where{Operator: OpOr, Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{2}},
				}},
			}},
			equal: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			equal := test.w1.Equal(test.w2)
			if test.equal != equal {
				t.Fatalf("expected equal %t, got %t", test.equal, equal)
			}
		})
	}

}

// Test_Where_MarshalJSON verifies the persisted JSON representation of Where
// values.
func Test_Where_MarshalJSON(t *testing.T) {

	tests := []struct {
		name     string
		where    Where
		expected []byte
	}{
		{
			name: "condition values",
			where: Where{
				Operator: OpAnd,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
					&WhereCondition{Property: []string{"b"}, Operator: OpIsNot, Values: []any{"foo"}},
					&WhereCondition{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{10, 20}},
					&WhereCondition{Property: []string{"d"}, Operator: OpIsLessThan, Values: []any{34.98}},
					&WhereCondition{Property: []string{"e"}, Operator: OpIsGreaterThan, Values: []any{decimal.MustParse("34")}},
					&WhereCondition{Property: []string{"f"}, Operator: OpIsTrue},
					&WhereCondition{Property: []string{"g"}, Operator: OpIsNotOneOf, Values: []any{1, 2, 3}},
					&WhereCondition{Property: []string{"h"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
					&WhereCondition{Property: []string{"i"}, Operator: OpIsBetween, Values: []any{JSONConditionValue{String: "34", Number: new(decimal.MustParse("34"))}}},
				},
			},
			expected: []byte(`{"operator":"And","rules":[` +
				`{"property":["a"],"operator":"Is","values":[5]},` +
				`{"property":["b"],"operator":"IsNot","values":["foo"]},` +
				`{"property":["c"],"operator":"IsBetween","values":[10,20]},` +
				`{"property":["d"],"operator":"IsLessThan","values":[34.98]},` +
				`{"property":["e"],"operator":"IsGreaterThan","values":[34]},` +
				`{"property":["f"],"operator":"IsTrue"},` +
				`{"property":["g"],"operator":"IsNotOneOf","values":[1,2,3]},` +
				`{"property":["h"],"operator":"Is","values":["foo"]},` +
				`{"property":["i"],"operator":"IsBetween","values":["34"]}` + `]}`),
		},
		{
			name: "time pointer",
			where: Where{
				Operator: OpOr,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIsAfter, Values: []any{new(time.Date(2024, 9, 12, 11, 3, 6, 820793551, time.UTC))}},
				},
			},
			expected: []byte(`{"operator":"Or","rules":[{"property":["a"],"operator":"IsAfter","values":["2024-09-12T11:03:06.820793551Z"]}` + `]}`),
		},
		{
			name: "nested group",
			where: Where{
				Operator: OpAnd,
				Rules: []WhereRule{
					&WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
					&Where{Operator: OpOr, Rules: []WhereRule{
						&WhereCondition{Property: []string{"b"}, Operator: OpIsNull},
						&WhereCondition{Property: []string{"c"}, Operator: OpIsTrue},
					}},
				},
			},
			expected: []byte(`{"operator":"And","rules":[` +
				`{"property":["a"],"operator":"Is","values":[5]},` +
				`{"operator":"Or","rules":[` +
				`{"property":["b"],"operator":"IsNull"},` +
				`{"property":["c"],"operator":"IsTrue"}` +
				`]}` +
				`]}`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := json.Marshal(&test.where)
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			if !bytes.Equal(test.expected, got) {
				t.Fatalf("\nexpected %s\ngot      %s", string(test.expected), string(got))
			}
		})
	}

}

// Test_WhereCondition_Equal verifies equality of WhereCondition values,
// including non-comparable and JSON values.
func Test_WhereCondition_Equal(t *testing.T) {

	number34 := decimal.MustInt(34)
	number34Copy := decimal.MustInt(34)
	number35 := decimal.MustInt(35)
	tests := []struct {
		name        string
		left, right *WhereCondition
		equal       bool
	}{
		{name: "both nil", left: nil, right: nil, equal: true},
		{name: "receiver nil", left: nil, right: &WhereCondition{}, equal: false},
		{name: "other nil", left: &WhereCondition{}, right: nil, equal: false},
		{
			name:  "equal slices",
			left:  &WhereCondition{Values: []any{[]int{1, 2}}},
			right: &WhereCondition{Values: []any{[]int{1, 2}}},
			equal: true,
		},
		{
			name:  "different slices",
			left:  &WhereCondition{Values: []any{[]int{1, 2}}},
			right: &WhereCondition{Values: []any{[]int{1, 3}}},
			equal: false,
		},
		{
			name:  "equal maps",
			left:  &WhereCondition{Values: []any{map[string]int{"a": 1}}},
			right: &WhereCondition{Values: []any{map[string]int{"a": 1}}},
			equal: true,
		},
		{
			name:  "different maps",
			left:  &WhereCondition{Values: []any{map[string]int{"a": 1}}},
			right: &WhereCondition{Values: []any{map[string]int{"a": 2}}},
			equal: false,
		},
		{
			name: "equal JSON values",
			left: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number34},
			}},
			right: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number34Copy},
			}},
			equal: true,
		},
		{
			name: "different JSON strings",
			left: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number34},
			}},
			right: &WhereCondition{Values: []any{
				JSONConditionValue{String: "35", Number: &number34},
			}},
			equal: false,
		},
		{
			name: "JSON number presence differs",
			left: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34"},
			}},
			right: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number34},
			}},
			equal: false,
		},
		{
			name: "different JSON numbers",
			left: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number34},
			}},
			right: &WhereCondition{Values: []any{
				JSONConditionValue{String: "34", Number: &number35},
			}},
			equal: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.left.Equal(test.right); got != test.equal {
				t.Fatalf("expected equal %t, got %t", test.equal, got)
			}
		})
	}

}

// Test_WhereLogical_JSON verifies the external representation of every
// logical operator and the handling of invalid values.
func Test_WhereLogical_JSON(t *testing.T) {

	tests := []struct {
		op   WhereLogical
		name string
	}{
		{op: OpAnd, name: "And"},
		{op: OpOr, name: "Or"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expected := []byte(`"` + test.name + `"`)
			got, err := test.op.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("expected %s, got %s", expected, got)
			}
			if got := test.op.String(); got != test.name {
				t.Fatalf("expected %s, got %s", test.name, got)
			}
			var decoded WhereLogical
			if err := json.Unmarshal(got, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded != test.op {
				t.Fatalf("expected %d, got %d", test.op, decoded)
			}
		})
	}

	invalid := []struct {
		name string
		op   WhereLogical
	}{
		{name: "negative", op: WhereLogical(-1)},
		{name: "above maximum", op: OpOr + 1},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.op.MarshalJSON()
			if err == nil {
				t.Fatal("expected MarshalJSON to return an error")
			}
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("expected String to panic")
				}
				if fmt.Sprint(recovered) != err.Error() {
					t.Fatalf("expected panic %q, got %q", err, recovered)
				}
			}()
			_ = test.op.String()
		})
	}

}

// Test_WhereOperator_JSON verifies the external representation of every
// condition operator and the handling of invalid values.
func Test_WhereOperator_JSON(t *testing.T) {

	tests := []struct {
		op   WhereOperator
		name string
	}{
		{op: OpIs, name: "Is"},
		{op: OpIsNot, name: "IsNot"},
		{op: OpIsLessThan, name: "IsLessThan"},
		{op: OpIsLessThanOrEqualTo, name: "IsLessThanOrEqualTo"},
		{op: OpIsGreaterThan, name: "IsGreaterThan"},
		{op: OpIsGreaterThanOrEqualTo, name: "IsGreaterThanOrEqualTo"},
		{op: OpIsBetween, name: "IsBetween"},
		{op: OpIsNotBetween, name: "IsNotBetween"},
		{op: OpContains, name: "Contains"},
		{op: OpDoesNotContain, name: "DoesNotContain"},
		{op: OpIsOneOf, name: "IsOneOf"},
		{op: OpIsNotOneOf, name: "IsNotOneOf"},
		{op: OpStartsWith, name: "StartsWith"},
		{op: OpEndsWith, name: "EndsWith"},
		{op: OpIsBefore, name: "IsBefore"},
		{op: OpIsOnOrBefore, name: "IsOnOrBefore"},
		{op: OpIsAfter, name: "IsAfter"},
		{op: OpIsOnOrAfter, name: "IsOnOrAfter"},
		{op: OpIsTrue, name: "IsTrue"},
		{op: OpIsFalse, name: "IsFalse"},
		{op: OpIsEmpty, name: "IsEmpty"},
		{op: OpIsNotEmpty, name: "IsNotEmpty"},
		{op: OpIsNull, name: "IsNull"},
		{op: OpIsNotNull, name: "IsNotNull"},
		{op: OpExists, name: "Exists"},
		{op: OpDoesNotExist, name: "DoesNotExist"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expected := []byte(`"` + test.name + `"`)
			got, err := test.op.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, expected) {
				t.Fatalf("expected %s, got %s", expected, got)
			}
			if got := test.op.String(); got != test.name {
				t.Fatalf("expected %s, got %s", test.name, got)
			}
			var decoded WhereOperator
			if err := json.Unmarshal(got, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded != test.op {
				t.Fatalf("expected %d, got %d", test.op, decoded)
			}
		})
	}

	t.Run("legacy name", func(t *testing.T) {
		var op WhereOperator
		if err := json.Unmarshal([]byte(`"OpIsNotBetween"`), &op); err == nil {
			t.Fatal("expected legacy operator name to be rejected")
		}
	})

	invalid := []struct {
		name string
		op   WhereOperator
	}{
		{name: "negative", op: WhereOperator(-1)},
		{name: "above maximum", op: OpDoesNotExist + 1},
	}
	for _, test := range invalid {
		t.Run(test.name, func(t *testing.T) {
			_, err := test.op.MarshalJSON()
			if err == nil {
				t.Fatal("expected MarshalJSON to return an error")
			}
			defer func() {
				recovered := recover()
				if recovered == nil {
					t.Fatal("expected String to panic")
				}
				if fmt.Sprint(recovered) != err.Error() {
					t.Fatalf("expected panic %q, got %q", err, recovered)
				}
			}()
			_ = test.op.String()
		})
	}

}

// Test_JSONConditionValue_MarshalJSON verifies that JSON condition values are
// persisted using their raw strings.
func Test_JSONConditionValue_MarshalJSON(t *testing.T) {

	tests := []struct {
		name     string
		v        JSONConditionValue
		expected []byte
	}{
		{name: "empty", v: JSONConditionValue{String: ""}, expected: []byte(`""`)},
		{name: "string", v: JSONConditionValue{String: "foo"}, expected: []byte(`"foo"`)},
		{name: "integer", v: JSONConditionValue{String: "34", Number: new(decimal.MustInt(34))}, expected: []byte(`"34"`)},
		{
			name:     "decimal",
			v:        JSONConditionValue{String: "893.051", Number: new(decimal.MustParse("893.051"))},
			expected: []byte(`"893.051"`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := test.v.MarshalJSON()
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			if !bytes.Equal(test.expected, got) {
				t.Fatalf("expected %s\ngot      %s", string(test.expected), string(got))
			}
		})
	}

}
