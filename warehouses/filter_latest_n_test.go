// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateLatestN checks the boundary without semantic preparation.
func TestValidateLatestN(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	node := LatestOccurrences{Size: 2, Source: OccurrenceSource{Table: "events",
		Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "id", Type: types.String()},
		Timestamp: Column{Name: "timestamp", Type: types.DateTime()}}, Start: end.AddDate(0, -1, 0), End: end,
		Predicate: &RecordQualification{Expr: NewBaseExpr(Column{Name: "v", Type: types.String()}, OpIs, "yes")}}
	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles",
		Key: Column{Name: "id", Type: types.UUID()}}, Limit: 1}
	for _, test := range []struct {
		size                                        int
		missingPredicate, missingIdentity, reversed bool
	}{
		{1, false, false, false}, {10, false, false, false},
		{0, false, false, false}, {11, false, false, false}, {-1, false, false, false},
		{2, true, false, false}, {2, false, true, false}, {2, false, false, true},
	} {
		modified := node
		modified.Size = test.size
		if test.missingPredicate {
			modified.Predicate = nil
		}
		if test.missingIdentity {
			modified.Source.Identity = Column{}
		}
		if test.reversed {
			modified.Start, modified.End = modified.End, modified.Start
		}
		invalid := test.size < 1 || test.size > 10 || test.missingPredicate || test.missingIdentity || test.reversed
		err := ValidateProfileFilter(Filter{Root: &modified}, request)
		if err != nil {
			if !invalid {
				t.Fatalf("expected valid size %d, got %v", test.size, err)
			}
			continue
		}
		if invalid {
			t.Fatalf("expected invalid physical block %+v, got success", test)
		}
	}
	for _, size := range []int{100, 101} {
		root := &FilterGroup{Operator: OpAnd, Children: make([]FilterCondition, size)}
		for i := range root.Children {
			root.Children[i] = &node
		}
		err := ValidateProfileFilter(Filter{Root: root}, request)
		if (err != nil) != (size == 101) {
			t.Fatalf("expected node budget valid=%v, got %v", size == 100, err)
		}
	}
}
