// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
)

// TestExperienceNth checks selectors constructed without public JSON parsing.
func TestExperienceNth(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		first    bool
		position int
		invalid  bool
	}{
		{false, 0, false}, {true, 0, false}, {false, 1, false}, {false, 10, false},
		{false, -1, true}, {false, 11, true}, {true, 1, true},
	} {
		experience := &Experience{First: test.first, Nth: test.position,
			Scope: Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-time.Hour), End: at},
			Verify: &OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"event"},
				Operator: state.OpExists}}}
		root := &Rule{Experience: experience}
		err := Validate(root, at, Page{Limit: 1})
		if err != nil {
			if !test.invalid {
				t.Fatalf("expected valid position %d, got %v", test.position, err)
			}
			continue
		}
		if test.invalid {
			t.Fatalf("expected invalid first=%v position=%d, got success", test.first, test.position)
		}
	}
}
