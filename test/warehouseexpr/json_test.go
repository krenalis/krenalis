// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouseexpr

import (
	"testing"

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
			if value, ok := value.(json.Value); !ok || !json.Valid(value) {
				t.Errorf("expected JSON operand for %s at index %d, got %T", c.Name, i, value)
			}
		}

	}

}
