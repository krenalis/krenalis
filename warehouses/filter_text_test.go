// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

// TestExactColumnText validates native literal comparisons at the boundary.
func TestExactColumnText(t *testing.T) {
	for _, test := range []struct {
		name    string
		column  Column
		value   string
		invalid bool
	}{
		{"literal", Column{Name: "event", Type: types.String()}, "Purchase'", false},
		{"empty", Column{Name: "event", Type: types.String(), Nullable: true}, "", false},
		{"unicode", Column{Name: "event", Type: types.String()}, "é", false},
		{"identifier", Column{Name: "bad.name", Type: types.String()}, "Purchase", true},
		{"type", Column{Name: "event", Type: types.JSON()}, "Purchase", true},
		{"UTF-8", Column{Name: "event", Type: types.String()}, "\xff", true},
		{"NUL", Column{Name: "event", Type: types.String()}, "a\x00b", true},
		{"size", Column{Name: "event", Type: types.String()}, strings.Repeat("x", 4097), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			node := &IndependentOccurrenceCount{Source: OccurrenceSource{Table: "events",
				Profile:   Column{Name: "kpid", Type: types.UUID()},
				Identity:  Column{Name: "message_id", Type: types.String()},
				Timestamp: Column{Name: "timestamp", Type: types.DateTime()}}, Start: end.Add(-time.Hour), End: end,
				Qualification: &ExactColumnText{Column: test.column, Value: test.value}}
			err := ValidateProfileFilter(Filter{Root: node}, ProfileOutcomeRequest{Population: ProfileSource{
				Table: "profiles", Key: Column{Name: "_kpid", Type: types.UUID()}}, Limit: 1})
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid exact text qualification, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid exact text qualification, got success")
			}
		})
	}
}
