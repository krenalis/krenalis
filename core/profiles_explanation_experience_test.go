// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/unified"
	"github.com/krenalis/krenalis/warehouses"
)

// TestExplainExperienceSelection distinguishes candidate and absence states.
func TestExplainExperienceSelection(t *testing.T) {
	for _, test := range []struct {
		candidates int64
		absent     bool
		kind       string
		absence    string
	}{
		{0, true, "absent", "certain"},
		{1, true, "ambiguous", "possible"},
		{1, false, "unique", "impossible"},
		{2, false, "ambiguous", "impossible"},
	} {
		nodes := []warehouses.FilterNodeOutcome{{Truth: warehouses.FilterUnknown,
			Experience: &warehouses.OccurrenceSelection{Candidates: test.candidates, AbsencePossible: test.absent}}}
		if test.candidates == 0 {
			nodes[0].Truth = warehouses.FilterNoMatch
		}
		index := 0
		result, err := explainProfileRule(&warehouses.LastOccurrence{}, unified.Rule{Experience: &unified.Experience{}},
			nodes, &index)
		if err != nil {
			t.Fatalf("expected experience explanation, got %v", err)
		}
		if result.Experience == nil || result.Experience.Selection.Kind != test.kind ||
			result.Experience.Selection.Absence != test.absence || result.Experience.Selection.Candidates != test.candidates {
			t.Fatalf("expected %s/%s with %d candidates, got %+v", test.kind, test.absence, test.candidates, result)
		}
	}
}
