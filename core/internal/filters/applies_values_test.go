// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package filters

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/warehouseexpr"
)

// TestAppliesValuePredicates checks ordinary values, absent keys and Go nils.
func TestAppliesValuePredicates(t *testing.T) {
	for _, c := range warehouseexpr.ValueCases() {
		t.Run(c.Name, func(t *testing.T) {
			where := &state.Where{Operator: state.OpAnd, Rules: []state.WhereRule{
				&state.WhereCondition{Property: []string{"value"}, Operator: state.WhereOperator(c.Operator), Values: c.Values},
			}}
			for _, test := range []struct {
				name       string
				attributes map[string]any
				matches    bool
			}{
				{"present", map[string]any{"value": c.Value}, c.Matches},
				{"missing", map[string]any{}, false},
				{"nil present", map[string]any{"value": nil}, false},
			} {
				t.Run(test.name, func(t *testing.T) {
					if got := Applies(where, test.attributes); got != test.matches {
						t.Fatalf("expected membership %t, got %t", test.matches, got)
					}
				})
			}
		})
	}
}
