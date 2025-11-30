// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package state

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/types"
)

func Test_unmarshalWhere(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Int(32)},
		{Name: "b", Type: types.String()},
		{Name: "c", Type: types.Int(8)},
		{Name: "d", Type: types.Float(64)},
		{Name: "e", Type: types.String()},
		{Name: "f", Type: types.Boolean()},
		{Name: "g", Type: types.Uint(32)},
		{Name: "h", Type: types.JSON()},
		{Name: "i", Type: types.JSON()},
		{Name: "j", Type: types.DateTime()},
		{Name: "k", Type: types.Date()},
		{Name: "l", Type: types.Time()},
		{Name: "m", Type: types.Decimal(5, 3)},
		{Name: "n", Type: types.UUID()},
		{Name: "o", Type: types.Inet()},
	})

	vDecimalInt := decimal.MustParse("34")
	vDecimalFloat := decimal.MustParse("85.027")
	vDateTime := time.Date(2024, 9, 12, 11, 3, 6, 801586201, time.UTC)
	vDate := time.Date(2024, 9, 12, 0, 0, 0, 0, time.UTC)
	vTime := time.Date(1970, 1, 1, 11, 3, 6, 801586201, time.UTC)

	tests := []struct {
		condition string
		expected  WhereCondition
	}{
		{
			condition: `{"property":["a"],"operator":"Is","values":[5]}`,
			expected:  WhereCondition{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
		},
		{
			condition: `{"property":["b"],"operator":"IsNot","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"b"}, Operator: OpIsNot, Values: []any{"foo"}},
		},
		{
			condition: `{"property":["c"],"operator":"IsBetween","values":[10,20]}`,
			expected:  WhereCondition{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{10, 20}},
		},
		{
			condition: `{"property":["d"],"operator":"IsLessThan","values":[34.98]}`,
			expected:  WhereCondition{Property: []string{"d"}, Operator: OpIsLessThan, Values: []any{34.98}},
		},
		{
			condition: `{"property":["e"],"operator":"IsGreaterThan","values":["34"]}`,
			expected:  WhereCondition{Property: []string{"e"}, Operator: OpIsGreaterThan, Values: []any{"34"}},
		},
		{
			condition: `{"property":["f"],"operator":"IsTrue"}`,
			expected:  WhereCondition{Property: []string{"f"}, Operator: OpIsTrue},
		},
		{
			condition: `{"property":["g"],"operator":"IsNotOneOf","values":[1,2,3]}`,
			expected:  WhereCondition{Property: []string{"g"}, Operator: OpIsNotOneOf, Values: []any{uint(1), uint(2), uint(3)}},
		},
		{
			condition: `{"property":["h"],"operator":"Is","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"h"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
		},
		{
			condition: `{"property":["h","x"],"operator":"Is","values":["foo"]}`,
			expected:  WhereCondition{Property: []string{"h", "x"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
		},
		{
			condition: `{"property":["i"],"operator":"IsBetween","values":["34"]}`,
			expected:  WhereCondition{Property: []string{"i"}, Operator: OpIsBetween, Values: []any{JSONConditionValue{String: "34", Number: &vDecimalInt}}},
		},
		{
			condition: `{"property":["j"],"operator":"IsAfter","values":["2024-09-12T11:03:06.801586201Z"]}`,
			expected:  WhereCondition{Property: []string{"j"}, Operator: OpIsAfter, Values: []any{vDateTime}},
		},
		{
			condition: `{"property":["k"],"operator":"IsBefore","values":["2024-09-12"]}`,
			expected:  WhereCondition{Property: []string{"k"}, Operator: OpIsBefore, Values: []any{vDate}},
		},
		{
			condition: `{"property":["l"],"operator":"IsOnOrBefore","values":["11:03:06.801586201"]}`,
			expected:  WhereCondition{Property: []string{"l"}, Operator: OpIsOnOrBefore, Values: []any{vTime}},
		},
		{
			condition: `{"property":["m"],"operator":"Is","values":[85.027]}`,
			expected:  WhereCondition{Property: []string{"m"}, Operator: OpIs, Values: []any{vDecimalFloat}},
		},
		{
			condition: `{"property":["n"],"operator":"IsNot","values":["38d065ab-ca46-4812-a83c-a9712e09c153"]}`,
			expected:  WhereCondition{Property: []string{"n"}, Operator: OpIsNot, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
		},
		{
			condition: `{"property":["o"],"operator":"Is","values":["192.168.1.1"]}`,
			expected:  WhereCondition{Property: []string{"o"}, Operator: OpIs, Values: []any{"192.168.1.1"}},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := unmarshalWhere([]byte(`{"logical":"And","conditions":[`+test.condition+`]}`), schema)
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			expected := &Where{Logical: OpAnd, Conditions: []WhereCondition{test.expected}}
			if !reflect.DeepEqual(expected, got) {
				t.Fatalf("\nexpected %#v\ngot      %#v", expected, got)
			}
		})
	}

	tests2 := []struct {
		where    string
		expected Where
	}{
		{
			where: `{"logical":"And","conditions":[` +
				`{"property":["a"],"operator":"Is","values":[720347]},` +
				`{"property":["b"],"operator":"IsLessThan","values":["foo boo"]},` +
				`{"property":["c"],"operator":"IsBetween","values":[66,88]}` + `]}`,
			expected: Where{
				Logical: OpAnd,
				Conditions: []WhereCondition{
					{Property: []string{"a"}, Operator: OpIs, Values: []any{720347}},
					{Property: []string{"b"}, Operator: OpIsLessThan, Values: []any{"foo boo"}},
					{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{66, 88}},
				},
			},
		},
		{
			where: `{"logical":"Or","conditions":[` +
				`{"property":["d"],"operator":"Is","values":[90481.26681]},` +
				`{"property":["e"],"operator":"StartsWith","values":["Boo"]},` +
				`{"property":["f"],"operator":"IsFalse"}` + `]}`,
			expected: Where{
				Logical: OpOr,
				Conditions: []WhereCondition{
					{Property: []string{"d"}, Operator: OpIs, Values: []any{90481.26681}},
					{Property: []string{"e"}, Operator: OpStartsWith, Values: []any{"Boo"}},
					{Property: []string{"f"}, Operator: OpIsFalse},
				},
			},
		},
	}

	for _, test := range tests2 {
		t.Run("", func(t *testing.T) {
			got, err := unmarshalWhere([]byte(test.where), schema)
			if err != nil {
				t.Fatalf("unexpected error %q", err)
			}
			if got == nil {
				t.Fatalf("unexpected nil")
			}
			if !reflect.DeepEqual(&test.expected, got) {
				t.Fatalf("\nexpected %#v\ngot      %#v", &test.expected, got)
			}
		})
	}

}

func Test_Where_Equal(t *testing.T) {

	tests := []struct {
		w1, w2 *Where
		equal  bool
	}{
		{
			w1:    nil,
			w2:    nil,
			equal: true,
		},
		{
			w1:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsNull}}},
			w2:    nil,
			equal: false,
		},
		{
			w1:    nil,
			w2:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsNull}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsNull}}},
			w2:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsNull}}},
			equal: true,
		},
		{
			w1:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"b"}, Operator: OpIs, Values: []any{5}}}},
			w2:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"b"}, Operator: OpIs, Values: []any{5}}}},
			equal: true,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}}},
			equal: true,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}}},
			equal: true,
		},
		{
			w1:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsFalse}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsFalse}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsFalse}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"b"}, Operator: OpIsFalse}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsFalse}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"a"}, Operator: OpIsTrue}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"b"}, Operator: OpIs, Values: []any{"foo"}}}},
			w2:    &Where{Logical: OpAnd, Conditions: []WhereCondition{{Property: []string{"b"}, Operator: OpIs, Values: []any{"foo", "boo"}}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 15, 45, 12, 0, time.UTC)}}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"x", "y"}, Operator: OpIsBetween, Values: []any{time.Date(2025, 01, 01, 12, 30, 0, 0, time.UTC), time.Date(2025, 12, 31, 16, 45, 12, 0, time.UTC)}}}},
			equal: false,
		},
		{
			w1:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("39.05")}}}},
			w2:    &Where{Logical: OpOr, Conditions: []WhereCondition{{Property: []string{"c"}, Operator: OpIs, Values: []any{decimal.MustParse("12.89")}}}},
			equal: false,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			equal := test.w1.Equal(test.w2)
			if test.equal != equal {
				t.Fatalf("expected equal %t, got %t", test.equal, equal)
			}
		})
	}
}

func Test_Where_MarshalJSON(t *testing.T) {

	jn := decimal.MustParse("34")
	jt := time.Date(2024, 9, 12, 11, 3, 6, 820793551, time.UTC)

	tests := []struct {
		where    Where
		expected []byte
	}{
		{
			where: Where{
				Logical: OpAnd,
				Conditions: []WhereCondition{
					{Property: []string{"a"}, Operator: OpIs, Values: []any{5}},
					{Property: []string{"b"}, Operator: OpIsNot, Values: []any{"foo"}},
					{Property: []string{"c"}, Operator: OpIsBetween, Values: []any{10, 20}},
					{Property: []string{"d"}, Operator: OpIsLessThan, Values: []any{34.98}},
					{Property: []string{"e"}, Operator: OpIsGreaterThan, Values: []any{jn}},
					{Property: []string{"f"}, Operator: OpIsTrue},
					{Property: []string{"g"}, Operator: OpIsNotOneOf, Values: []any{1, 2, 3}},
					{Property: []string{"h"}, Operator: OpIs, Values: []any{JSONConditionValue{String: "foo"}}},
					{Property: []string{"i"}, Operator: OpIsBetween, Values: []any{JSONConditionValue{String: "34", Number: &jn}}},
				},
			},
			expected: []byte(`{"logical":"And","conditions":[` +
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
			where: Where{
				Logical: OpOr,
				Conditions: []WhereCondition{
					{Property: []string{"a"}, Operator: OpIsAfter, Values: []any{jt}},
				},
			},
			expected: []byte(`{"logical":"Or","conditions":[{"property":["a"],"operator":"IsAfter","values":["2024-09-12T11:03:06.820793551Z"]}` + `]}`),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got, err := test.where.MarshalJSON()
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

func Test_JSONConditionValue_Marshal(t *testing.T) {

	jn := decimal.MustInt(34)
	jf := decimal.MustParse("893.051")

	tests := []struct {
		v        JSONConditionValue
		expected []byte
	}{
		{v: JSONConditionValue{String: ""}, expected: []byte(`""`)},
		{v: JSONConditionValue{String: "foo"}, expected: []byte(`"foo"`)},
		{v: JSONConditionValue{String: "34", Number: &jn}, expected: []byte(`"34"`)},
		{v: JSONConditionValue{String: "893.051", Number: &jf}, expected: []byte(`"893.051"`)},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
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
