// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateExperience keeps local source validation outside the root budget.
func TestValidateExperience(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	node := &LastOccurrence{Source: OccurrenceSource{Table: "events",
		Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "id", Type: types.String()},
		Timestamp: Column{Name: "timestamp", Type: types.DateTime()}}, Start: end.AddDate(0, -1, 0), End: end,
		Verify: &RecordQualification{Expr: &BaseExpr{Column: Column{Name: "v", Type: types.String()},
			Operator: OpIs, Values: []any{"yes"}}}}
	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles",
		Key: Column{Name: "id", Type: types.UUID()}}, Limit: 1}
	for _, size := range []int{100, 101} {
		root := &FilterGroup{Operator: OpAnd, Children: make([]FilterCondition, size)}
		for i := range root.Children {
			root.Children[i] = node
		}
		err := ValidateProfileFilter(Filter{Root: root}, request)
		if err != nil {
			if size == 100 {
				t.Fatalf("expected 100 Experience nodes, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected descendant budget rejection, got success")
		}
	}
	for _, test := range []LastOccurrence{
		{Source: node.Source, Start: node.Start, End: node.End},
		{Source: node.Source, Start: node.End, End: node.Start, Verify: node.Verify},
		{Source: node.Source, Start: node.Start, End: node.End, Verify: &RecordQualification{Expr: nil}},
	} {
		err := ValidateProfileFilter(Filter{Root: &test}, request)
		if err == nil {
			t.Fatal("expected invalid prepared selection rejected, got success")
		}
	}
}
