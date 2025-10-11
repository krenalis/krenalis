//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package core

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/types"

	"github.com/google/go-cmp/cmp"
)

func Test_convertFilterToWhere(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Int(32)},
		{Name: "c", Type: types.Uint(8)},
		{Name: "d", Type: types.Float(32)},
		{Name: "e", Type: types.Float(64)},
		{Name: "f", Type: types.Decimal(10, 3)},
		{Name: "g", Type: types.DateTime()},
		{Name: "h", Type: types.Date()},
		{Name: "i", Type: types.Time()},
		{Name: "j", Type: types.Year()},
		{Name: "k", Type: types.UUID()},
		{Name: "l", Type: types.JSON()},
		{Name: "m", Type: types.Inet()},
		{Name: "n", Type: types.Text(), Nullable: true},
		{Name: "o", Type: types.Array(types.Text()), Nullable: true},
		{Name: "p", Type: types.Object([]types.Property{{Name: "x", Type: types.Text()}}), Nullable: true},
		{Name: "q", Type: types.Map(types.Text()), Nullable: true},
	})

	d := decimal.MustParse("56.19")

	tests := []struct {
		filter   Filter
		expected state.Where
	}{
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsTrue},
					{Property: "b", Operator: OpIs, Values: []string{"-105"}},
					{Property: "c", Operator: OpIsNot, Values: []string{"197"}},
					{Property: "d", Operator: OpIsGreaterThan, Values: []string{"3.802"}},
					{Property: "e", Operator: OpIsGreaterThan, Values: []string{"60526.8020184552091647"}},
					{Property: "e", Operator: OpIsLessThan, Values: []string{"1.7976931348623157e+308"}},
					{Property: "f", Operator: OpIsLessThanOrEqualTo, Values: []string{"12.956"}},
					{Property: "g", Operator: OpIsAfter, Values: []string{"2024-09-19T14:36:57.264699103"}},
					{Property: "h", Operator: OpIsBetween, Values: []string{"2024-09-19", "2025-09-19"}},
					{Property: "i", Operator: OpIsBefore, Values: []string{"14:36:57.264699103"}},
					{Property: "j", Operator: OpIsOnOrAfter, Values: []string{"2024"}},
					{Property: "k", Operator: OpIs, Values: []string{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					{Property: "l", Operator: OpIs, Values: []string{"foo"}},
					{Property: "l", Operator: OpIsOneOf, Values: []string{"foo", "56.19", "boo"}},
					{Property: "l.x", Operator: OpExists},
					{Property: "m", Operator: OpIsNot, Values: []string{"192.168.1.1"}},
					{Property: "n", Operator: OpIs, Values: []string{"boo"}},
					{Property: "o", Operator: OpIsNull},
					{Property: "o", Operator: OpContains, Values: []string{"boo"}},
					{Property: "p", Operator: OpIsNot},
					{Property: "p.x", Operator: OpIs, Values: []string{"foo"}},
					{Property: "p.x.y", Operator: OpExists},
					{Property: "q", Operator: OpIsNull},
				},
			},
			expected: state.Where{
				Logical: state.OpOr,
				Conditions: []state.WhereCondition{
					{Property: []string{"a"}, Operator: state.OpIsTrue},
					{Property: []string{"b"}, Operator: state.OpIs, Values: []any{-105}},
					{Property: []string{"c"}, Operator: state.OpIsNot, Values: []any{uint(197)}},
					{Property: []string{"d"}, Operator: state.OpIsGreaterThan, Values: []any{float64(float32(3.802))}},
					{Property: []string{"e"}, Operator: state.OpIsGreaterThan, Values: []any{60526.80201845521}},
					{Property: []string{"e"}, Operator: state.OpIsLessThan, Values: []any{1.7976931348623157e+308}},
					{Property: []string{"f"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{decimal.MustParse("12.956")}},
					{Property: []string{"g"}, Operator: state.OpIsAfter, Values: []any{time.Date(2024, 9, 19, 14, 36, 57, 264699103, time.UTC)}},
					{Property: []string{"h"}, Operator: state.OpIsBetween, Values: []any{time.Date(2024, 9, 19, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 19, 0, 0, 0, 0, time.UTC)}},
					{Property: []string{"i"}, Operator: state.OpIsBefore, Values: []any{time.Date(1970, 1, 1, 14, 36, 57, 264699103, time.UTC)}},
					{Property: []string{"j"}, Operator: state.OpIsOnOrAfter, Values: []any{2024}},
					{Property: []string{"k"}, Operator: state.OpIs, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					{Property: []string{"l"}, Operator: state.OpIs, Values: []any{state.JSONConditionValue{String: "foo"}}},
					{Property: []string{"l"}, Operator: state.OpIsOneOf, Values: []any{
						state.JSONConditionValue{String: "foo"},
						state.JSONConditionValue{String: "56.19", Number: &d},
						state.JSONConditionValue{String: "boo"},
					}},
					{Property: []string{"l", "x"}, Operator: state.OpExists},
					{Property: []string{"m"}, Operator: state.OpIsNot, Values: []any{"192.168.1.1"}},
					{Property: []string{"n"}, Operator: state.OpIs, Values: []any{"boo"}},
					{Property: []string{"o"}, Operator: state.OpIsNull},
					{Property: []string{"o"}, Operator: state.OpContains, Values: []any{"boo"}},
					{Property: []string{"p"}, Operator: state.OpIsNot},
					{Property: []string{"p", "x"}, Operator: state.OpIs, Values: []any{"foo"}},
					{Property: []string{"p", "x", "y"}, Operator: state.OpExists},
					{Property: []string{"q"}, Operator: state.OpIsNull},
				},
			},
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsTrue},
				},
			},
			expected: state.Where{
				Logical: state.OpAnd,
				Conditions: []state.WhereCondition{
					{Property: []string{"a"}, Operator: state.OpIsTrue},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := convertFilterToWhere(&test.filter, schema)
			if !reflect.DeepEqual(&test.expected, got) {
				t.Fatalf("unexpected value:\n%s", cmp.Diff(&test.expected, got))
			}
		})
	}
}

func Test_convertWhereToFilter(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Int(32)},
		{Name: "c", Type: types.Uint(8)},
		{Name: "d", Type: types.Float(32)},
		{Name: "e", Type: types.Float(64)},
		{Name: "f", Type: types.Decimal(10, 3)},
		{Name: "g", Type: types.DateTime()},
		{Name: "h", Type: types.Date()},
		{Name: "i", Type: types.Time()},
		{Name: "j", Type: types.Year()},
		{Name: "k", Type: types.UUID()},
		{Name: "l", Type: types.JSON()},
		{Name: "m", Type: types.Inet()},
		{Name: "n", Type: types.Text(), Nullable: true},
		{Name: "o", Type: types.Array(types.Text()), Nullable: true},
		{Name: "p", Type: types.Object([]types.Property{{Name: "x", Type: types.Text()}}), Nullable: true},
		{Name: "q", Type: types.Map(types.Text()), Nullable: true},
	})

	d := decimal.MustParse("56.19")

	tests := []struct {
		where    state.Where
		expected Filter
	}{
		{
			where: state.Where{
				Logical: state.OpOr,
				Conditions: []state.WhereCondition{
					{Property: []string{"a"}, Operator: state.OpIsTrue},
					{Property: []string{"b"}, Operator: state.OpIs, Values: []any{-105}},
					{Property: []string{"c"}, Operator: state.OpIsNot, Values: []any{uint(197)}},
					{Property: []string{"d"}, Operator: state.OpIsGreaterThan, Values: []any{float64(float32(3.802))}},
					{Property: []string{"e"}, Operator: state.OpIsGreaterThan, Values: []any{60526.80201845521}},
					{Property: []string{"e"}, Operator: state.OpIsLessThan, Values: []any{1.7976931348623157e+308}},
					{Property: []string{"f"}, Operator: state.OpIsLessThanOrEqualTo, Values: []any{decimal.MustParse("12.956")}},
					{Property: []string{"g"}, Operator: state.OpIsAfter, Values: []any{time.Date(2024, 9, 19, 14, 36, 57, 264699103, time.UTC)}},
					{Property: []string{"h"}, Operator: state.OpIsBetween, Values: []any{time.Date(2024, 9, 19, 0, 0, 0, 0, time.UTC), time.Date(2025, 9, 19, 0, 0, 0, 0, time.UTC)}},
					{Property: []string{"i"}, Operator: state.OpIsBefore, Values: []any{time.Date(1970, 1, 1, 14, 36, 57, 264699103, time.UTC)}},
					{Property: []string{"j"}, Operator: state.OpIsOnOrAfter, Values: []any{2024}},
					{Property: []string{"k"}, Operator: state.OpIs, Values: []any{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					{Property: []string{"l"}, Operator: state.OpIs, Values: []any{state.JSONConditionValue{String: "foo"}}},
					{Property: []string{"l"}, Operator: state.OpIsOneOf, Values: []any{
						state.JSONConditionValue{String: "foo"},
						state.JSONConditionValue{String: "56.19", Number: &d},
						state.JSONConditionValue{String: "boo"},
					}},
					{Property: []string{"l.x"}, Operator: state.OpExists},
					{Property: []string{"m"}, Operator: state.OpIsNot, Values: []any{"192.168.1.1"}},
					{Property: []string{"n"}, Operator: state.OpIs, Values: []any{"boo"}},
					{Property: []string{"o"}, Operator: state.OpIsNull},
					{Property: []string{"o"}, Operator: state.OpContains, Values: []any{"boo"}},
					{Property: []string{"p"}, Operator: state.OpIsNot},
					{Property: []string{"p.x"}, Operator: state.OpIs, Values: []any{"foo"}},
					{Property: []string{"p.x.y"}, Operator: state.OpExists},
					{Property: []string{"q"}, Operator: state.OpIsNull},
				},
			},
			expected: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsTrue},
					{Property: "b", Operator: OpIs, Values: []string{"-105"}},
					{Property: "c", Operator: OpIsNot, Values: []string{"197"}},
					{Property: "d", Operator: OpIsGreaterThan, Values: []string{"3.802"}},
					{Property: "e", Operator: OpIsGreaterThan, Values: []string{"60526.80201845521"}},
					{Property: "e", Operator: OpIsLessThan, Values: []string{"1.7976931348623157e+308"}},
					{Property: "f", Operator: OpIsLessThanOrEqualTo, Values: []string{"12.956"}},
					{Property: "g", Operator: OpIsAfter, Values: []string{"2024-09-19T14:36:57.264699103"}},
					{Property: "h", Operator: OpIsBetween, Values: []string{"2024-09-19", "2025-09-19"}},
					{Property: "i", Operator: OpIsBefore, Values: []string{"14:36:57.264699103"}},
					{Property: "j", Operator: OpIsOnOrAfter, Values: []string{"2024"}},
					{Property: "k", Operator: OpIs, Values: []string{"38d065ab-ca46-4812-a83c-a9712e09c153"}},
					{Property: "l", Operator: OpIs, Values: []string{"foo"}},
					{Property: "l", Operator: OpIsOneOf, Values: []string{"foo", "56.19", "boo"}},
					{Property: "l.x", Operator: OpExists},
					{Property: "m", Operator: OpIsNot, Values: []string{"192.168.1.1"}},
					{Property: "n", Operator: OpIs, Values: []string{"boo"}},
					{Property: "o", Operator: OpIsNull},
					{Property: "o", Operator: OpContains, Values: []string{"boo"}},
					{Property: "p", Operator: OpIsNot},
					{Property: "p.x", Operator: OpIs, Values: []string{"foo"}},
					{Property: "p.x.y", Operator: OpExists},
					{Property: "q", Operator: OpIsNull},
				},
			},
		},
		{
			where: state.Where{
				Logical: state.OpAnd,
				Conditions: []state.WhereCondition{
					{Property: []string{"a"}, Operator: state.OpIsTrue},
				},
			},
			expected: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsTrue},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			got := convertWhereToFilter(&test.where, schema)
			if !reflect.DeepEqual(&test.expected, got) {
				t.Fatalf("unexpected value:\n%s", cmp.Diff(&test.expected, got))
			}
		})
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
		{"1.23", 32, float64(float32(1.23)), true},
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
		t.Run(test.n, func(t *testing.T) {
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
				t.Fatalf("expected %f, got %f", test.expected, got)
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
		{"000", 0, false}, // invalid due to leading zeros.
		{"00123", 0, false},
		{"+0", 0, true},
		{"-0", 0, true},
		{"-00123", 0, false},                          // invalid due to leading zeros.
		{"-9223372036854775809", 0, false},            // overflow negative.
		{"-9223372036854775808", math.MinInt64, true}, // minimum int64.
		{"9223372036854775807", math.MaxInt64, true},  // maximum int64.
		{"9223372036854775808", 0, false},             // overflow positive.
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

func Test_parseYear(t *testing.T) {
	tests := []struct {
		year     string
		expected int
		valid    bool
	}{
		{"2023", 2023, true},
		{"1999", 1999, true},
		{"99", 99, true},
		{"0", 0, false},               // overflow.
		{"1", types.MinYear, true},    // minimum year.
		{"9999", types.MaxYear, true}, // maximum year.
		{"10000", 0, false},           // overflow.
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

func Test_parseUint(t *testing.T) {
	tests := []struct {
		n        string
		expected uint
		valid    bool
	}{
		{"0", 0, true},
		{"123", 123, true},
		{"000", 0, false}, // leading zeros should be invalid.
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
			got, valid := parseUint(test.n)
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

func Test_parseUUID(t *testing.T) {
	tests := []struct {
		s    string
		uuid string
		ok   bool
	}{

		// Supported UUID formats.
		{"2a9b8326-aadb-416a-adc1-71761f3ff4b9", "2a9b8326-aadb-416a-adc1-71761f3ff4b9", true},
		{"2a9b8326-aadb-416a-ADC1-71761F3FF4B9", "2a9b8326-aadb-416a-adc1-71761f3ff4b9", true},
		{"2A9B8326-AADB-416A-ADC1-71761F3FF4B9", "2a9b8326-aadb-416a-adc1-71761f3ff4b9", true},

		// Unsupported UUID formats.
		{"60af802184814c8389153f9055d57e6c", "", false},
		{"60AF802184814C8389153F9055D57E6C", "", false},
		{"{60af8021-8481-4c83-8915-3f9055d57e6c}", "", false},
		{"{60AF8021-8481-4C83-8915-3F9055D57E6C}", "", false},
		{"urn:uuid:60af8021-8481-4c83-8915-3f9055d57e6c", "", false},
		{"urn:uuid:60AF8021-8481-4C83-8915-3F9055D57E6C", "", false},

		// Strings that do not represent UUIDs.
		{"", "", false},
		{"12345", "", false},
		{"abcdef0123456789", "", false},
		{"2a9b8326-aadb-416a-adc1-71761f3ff4b92a9b8326-aadb-416a-adc1-71761f3ff4b9", "", false},
	}
	for _, test := range tests {
		t.Run(test.s, func(t *testing.T) {
			gotUUID, gotOk := types.ParseUUID(test.s)
			if gotUUID != test.uuid {
				t.Fatalf("expected UUID %s, got %s", test.uuid, gotUUID)
			}
			if gotOk != test.ok {
				t.Fatalf("expected ok = %t, got %t", test.ok, gotOk)
			}
		})
	}
}

func Test_resolveFilterProperty(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Text(), Nullable: true},
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
		{"b", types.Property{Name: "b", Type: types.Text(), Nullable: true}, "b", nil},
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
			gotProperty, gotPath, err := retrieveFilterProperty(properties, test.path)
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no error, got error %q  (type %T)", err, err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no errors", test.err)
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

func Test_validateFilter(t *testing.T) {

	schema := types.Object([]types.Property{
		{Name: "a", Type: types.Boolean()},
		{Name: "b", Type: types.Text(), Nullable: true},
		{Name: "c", Type: types.Int(8)},
		{Name: "d", Type: types.JSON()},
		{Name: "e", Type: types.DateTime()},
		{Name: "f", Type: types.Date()},
		{Name: "g", Type: types.Time()},
		{Name: "h", Type: types.UUID(), ReadOptional: true},
		{Name: "i", Type: types.Year()},
		{Name: "j", Type: types.Inet()},
		{Name: "k", Type: types.Array(types.Text()), Nullable: true},
		{Name: "l", Type: types.Array(types.Map(types.Text()))},
		{Name: "m", Type: types.Object([]types.Property{{Name: "x", Type: types.Text()}}), Nullable: true},
		{Name: "n", Type: types.Map(types.Text()), Nullable: true},
		{Name: "o", Type: types.Text().WithValues("foo", "boo", ""), Nullable: true},
		{Name: "p", Type: types.Text().WithValues("foo"), Nullable: true},
	})

	tests := []struct {
		filter   Filter
		role     state.Role
		target   state.Target
		expected []string
		err      error
	}{
		{
			filter: Filter{
				Logical: "foo",
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: errors.New(`invalid logical operator "foo"`),
		},
		{
			filter: Filter{
				Logical: OpOr,
			},
			err: errors.New("conditions are missing"),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a c", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: errors.New("property path is not valid"),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "z", Operator: OpIs, Values: []string{"5"}},
				},
			},
			err: types.PathNotExistError{Path: "z"},
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIs, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is" cannot be used with boolean properties`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsLessThan, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is less than" cannot be used with boolean properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "l", Operator: OpContains, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "contains" cannot be used with array(map) properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpDoesNotContain, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "does not contain" cannot be used with boolean properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsBefore, Values: []string{"true"}},
				},
			},
			err: errors.New(`operator "is before" cannot be used with boolean properties`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpIsTrue},
				},
			},
			err: errors.New(`operator "is true" cannot be used with text properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: OpIsNull},
				},
			},
			err: errors.New(`operator "is null" can only be used with nullable or json properties`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpExists},
				},
			},
			err: errors.New(`operator "exists" can only be used with read-optional properties or with json properties that include a JSON path`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "d", Operator: OpDoesNotExist},
				},
			},
			err: errors.New(`operator "does not exist" can only be used with read-optional properties or with json properties that include a JSON path`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "a", Operator: "boo", Values: []string{"5"}},
				},
			},
			err: errors.New(`operator "boo" is not valid`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpIsNull, Values: []string{"5"}},
				},
			},
			err: errors.New(`values cannot be used with the operator "is null"`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpIsBetween, Values: []string{"5"}},
				},
			},
			err: errors.New(`two values must be used with the operator "is between"`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpContains},
				},
			},
			err: errors.New(`only one value can be used with the operator "contains"`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpContains, Values: []string{"foo \x00"}},
				},
			},
			err: errors.New("condition value contains the NUL byte"),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "c", Operator: OpIs, Values: []string{"75.0"}},
				},
			},
			err: fmt.Errorf(`value of the "c" property is not a valid int`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "o", Operator: OpIsLessThan, Values: []string{"none"}},
				},
			},
			err: fmt.Errorf(`operator "is less than" cannot be used with text type that has values`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "o", Operator: OpIsOneOf, Values: []string{"foo", "moo"}},
				},
			},
			err: fmt.Errorf(`value of the "o" property is not among the allowed values`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "p", Operator: OpIsEmpty},
				},
			},
			err: fmt.Errorf(`operator "is empty" cannot be used on text properties that exclude the empty string from allowed values`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "p", Operator: OpIsNotEmpty},
				},
			},
			err: fmt.Errorf(`operator "is not empty" cannot be used on text properties that exclude the empty string from allowed values`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "g", Operator: OpIsEmpty},
				},
			},
			err: fmt.Errorf(`operator "is empty" can only be used with json, text, object, array, and map properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "g", Operator: OpIsEmpty},
				},
			},
			err: fmt.Errorf(`operator "is empty" can only be used with json, text, object, array, and map properties`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "m", Operator: OpIsEmpty},
				},
			},
			target: state.TargetUser,
			role:   state.Destination,
			err:    fmt.Errorf(`operator "is empty" cannot be used on object properties for destination actions on users`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "m", Operator: OpIsNotEmpty},
				},
			},
			target: state.TargetEvent,
			role:   state.Source,
			err:    fmt.Errorf(`operator "is not empty" cannot be used on object properties for actions on events`),
		},
		{
			filter: Filter{
				Logical: OpAnd,
				Conditions: []FilterCondition{
					{Property: "d.p", Operator: OpIsNotEmpty},
				},
			},
			target: state.TargetUser,
			role:   state.Destination,
			err:    fmt.Errorf(`property "d" has type json, which is not supported in data warehouse exports`),
		},
		{
			filter: Filter{
				Logical: OpOr,
				Conditions: []FilterCondition{
					{Property: "b", Operator: OpIs, Values: []string{"5"}},
					{Property: "b", Operator: OpIsNot, Values: []string{"foo"}},
					{Property: "c", Operator: OpIsLessThan, Values: []string{"12"}},
					{Property: "c", Operator: OpIsLessThanOrEqualTo, Values: []string{"5"}},
					{Property: "b", Operator: OpIsGreaterThan, Values: []string{"boo"}},
					{Property: "b", Operator: OpIsEmpty},
					{Property: "c", Operator: OpIsGreaterThanOrEqualTo, Values: []string{"23"}},
					{Property: "c", Operator: OpIsBetween, Values: []string{"10", "20"}},
					{Property: "c", Operator: OpIsNotBetween, Values: []string{"20", "30"}},
					{Property: "b", Operator: OpContains, Values: []string{"abc"}},
					{Property: "b", Operator: OpDoesNotContain, Values: []string{"abc"}},
					{Property: "c", Operator: OpIsOneOf, Values: []string{"5", "8", "-3"}},
					{Property: "b", Operator: OpIsNotOneOf, Values: []string{"a", "b", "c"}},
					{Property: "b", Operator: OpStartsWith, Values: []string{"abc"}},
					{Property: "b", Operator: OpEndsWith, Values: []string{"abc"}},
					{Property: "e", Operator: OpIsBefore, Values: []string{"2024-09-10T15:34:31"}},
					{Property: "f", Operator: OpIsOnOrBefore, Values: []string{"2024-09-10"}},
					{Property: "g", Operator: OpIsAfter, Values: []string{"15:34:31"}},
					{Property: "h", Operator: OpIs, Values: []string{"dbfa8339-0d12-4c94-a3fb-569199ae5c8e"}},
					{Property: "h", Operator: OpExists},
					{Property: "h", Operator: OpDoesNotExist},
					{Property: "i", Operator: OpIsBefore, Values: []string{"2024"}},
					{Property: "j", Operator: OpIs, Values: []string{"192.168.1.1"}},
					{Property: "d", Operator: OpIsNotEmpty},
					{Property: "e", Operator: OpIsOnOrAfter, Values: []string{"2024-09-10T15:34:31"}},
					{Property: "a", Operator: OpIsTrue},
					{Property: "a", Operator: OpIsFalse},
					{Property: "b", Operator: OpIsNull},
					{Property: "b", Operator: OpIsNotNull},
					{Property: "d", Operator: OpIsEmpty},
					{Property: "d.s", Operator: OpIsEmpty},
					{Property: "d.s", Operator: OpExists},
					{Property: "d.s", Operator: OpDoesNotExist},
					{Property: "k", Operator: OpIsNull},
					{Property: "k", Operator: OpContains, Values: []string{"boo"}},
					{Property: "m", Operator: OpIsNull},
					{Property: "m", Operator: OpIsEmpty},
					{Property: "m.x", Operator: OpContains, Values: []string{"abc"}},
					{Property: "n", Operator: OpIsNotNull},
					{Property: "n", Operator: OpIsNotEmpty},
					{Property: "o", Operator: OpIs, Values: []string{"foo"}},
					{Property: "o", Operator: OpIs, Values: []string{""}},
					{Property: "o", Operator: OpIsOneOf, Values: []string{"foo", "boo"}},
					{Property: "o", Operator: OpIsNotNull},
					{Property: "o", Operator: OpIsEmpty},
					{Property: "o", Operator: OpIsNotEmpty},
				},
			},
			expected: []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "m", "m.x", "n", "o"},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			role := test.role
			if role == state.Both {
				role = state.Source
			}
			target := state.TargetUser
			if test.target == state.TargetEvent {
				target = state.TargetEvent
			}
			got, err := validateFilter(&test.filter, schema, role, target)
			if err != nil {
				if test.err == nil {
					t.Fatalf("expected no error, got error %q  (type %T)", err, err)
				}
				if !reflect.DeepEqual(test.err, err) {
					t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
				}
				return
			}
			if test.err != nil {
				t.Fatalf("expected error %q, got no errors", test.err)
			}
			if !slices.Equal(test.expected, got) {
				t.Fatalf("expected %q, got %q", test.expected, got)
			}
		})
	}

}
