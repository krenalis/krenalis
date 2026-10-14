// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouseexpr

import (
	"strconv"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/warehouses"
)

// TestJSONCases validates the structure of the shared JSON test cases.
func TestJSONCases(t *testing.T) {

	values, cases := JSONCases()
	if len(values) == 0 || len(cases) == 0 {
		t.Fatalf("expected nonempty fixtures and cases, got %d and %d", len(values), len(cases))
	}
	for i, value := range values {
		if value != nil && !json.Valid(value) {
			t.Errorf("expected valid JSON or an absent property at fixture index %d, got %q", i, value)
		}
	}

	names := map[string]bool{}
	for _, c := range cases {

		if c.Name == "" || names[c.Name] {
			t.Errorf("expected unique nonempty case name, got %q", c.Name)
		}
		names[c.Name] = true
		previous := -1
		for _, i := range c.Matches {
			if i < 0 || i >= len(values) || i <= previous {
				t.Errorf("expected sorted unique indices in [0, %d) for %s, got %v", len(values), c.Name, c.Matches)
				break
			}
			previous = i
		}

		var count int
		switch c.Operator {
		case warehouses.OpIs, warehouses.OpIsNot, warehouses.OpIsLessThan, warehouses.OpIsLessThanOrEqualTo,
			warehouses.OpIsGreaterThan, warehouses.OpIsGreaterThanOrEqualTo, warehouses.OpContains,
			warehouses.OpDoesNotContain, warehouses.OpStartsWith, warehouses.OpEndsWith:
			count = 1
		case warehouses.OpIsBetween, warehouses.OpIsNotBetween:
			count = 2
		case warehouses.OpIsOneOf, warehouses.OpIsNotOneOf:
			count = len(c.Values)
			if count == 0 {
				t.Errorf("expected at least one operand for %s, got 0", c.Name)
			}
		case warehouses.OpIsTrue, warehouses.OpIsFalse, warehouses.OpIsEmpty, warehouses.OpIsNotEmpty,
			warehouses.OpIsNull, warehouses.OpIsNotNull, warehouses.OpExists, warehouses.OpDoesNotExist:
		default:
			t.Errorf("expected supported JSON operator for %s, got %d", c.Name, c.Operator)
			continue
		}
		if len(c.Values) != count {
			t.Errorf("expected %d operands for %s, got %d", count, c.Name, len(c.Values))
		}
		for i, value := range c.Values {
			if _, ok := value.(warehouses.JSONConditionValue); !ok {
				t.Errorf("expected JSON operand for %s at index %d, got %T", c.Name, i, value)
			}
		}

	}

}

// TestOperand checks that operands preserve their original text and parse
// numbers correctly.
func TestOperand(t *testing.T) {

	for _, c := range []struct {
		text   string
		number bool
	}{
		{"5", true}, {"5.0", true}, {"5e0", true}, {"+5", true}, {"0.5", true},
		{"-5", true}, {".005", true}, {"0.", true},
		{"9007199254740992", true}, {"9007199254740993", true},
		{"0", true}, {"-0.0", true}, {"1000", true}, {"1e3", true},
		{"0.1234567890123456789012345678901234567", true},
		{"0.1234567890123456789012345678901234568", true},
		{"1e999999", true}, {"-1e999999", true}, {"1e-999999", true},
		{"", false}, {"foo", false}, {"true", false}, {"null", false}, {"1e", false},
		{"1e+", false}, {"1.2.3", false}, {"\"5\"", false}, {"NaN", false},
		{"Infinity", false}, {"--5", false}, {" 5", false}, {"5 ", false},
	} {
		t.Run(strconv.Quote(c.text), func(t *testing.T) {

			value := Operand(c.text)
			if value.String != c.text {
				t.Errorf("expected original value text %q, got %q", c.text, value.String)
			}
			if got := value.Number != nil; got != c.number {
				t.Fatalf("expected numeric operand %t, got %t", c.number, got)
			}
			if !c.number {
				return
			}

			expected, err := decimal.Parse(c.text, 0, 0)
			if err != nil {
				t.Fatalf("expected valid numeric fixture, got %v", err)
			}
			if !value.Number.Equal(expected) {
				t.Errorf("expected parsed number %s, got %s", expected, *value.Number)
			}

		})
	}

	for _, c := range []struct {
		left, right string
		comparison  int
	}{
		{"5", "5.0", 0}, {"5", "5e0", 0}, {"5", "+5", 0},
		{"1000", "1e3", 0}, {"0", "-0.0", 0},
		{"9007199254740992", "9007199254740993", -1},
		{"0.1234567890123456789012345678901234567", "0.1234567890123456789012345678901234568", -1},
	} {
		t.Run("relation/"+strconv.Quote(c.left)+"/"+strconv.Quote(c.right), func(t *testing.T) {
			left, right := Operand(c.left), Operand(c.right)
			if left.Number == nil || right.Number == nil {
				t.Fatalf("expected two numeric operands, got %v and %v", left.Number, right.Number)
			}
			if got := left.Number.Cmp(*right.Number); got != c.comparison {
				t.Errorf("expected numeric comparison %d, got %d", c.comparison, got)
			}
		})
	}

}
