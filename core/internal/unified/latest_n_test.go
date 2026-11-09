// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package unified

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
)

// TestLatestN validates direct construction independently of the public codec.
func TestLatestN(t *testing.T) {
	at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	scope := Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-time.Hour), End: at}
	predicate := &OccurrencePredicate{Record: &state.WhereCondition{Property: []string{"event"}, Operator: state.OpExists}}
	for _, size := range []int{-1, 0, 1, 10, 11} {
		root := &Rule{LatestN: &LatestN{Size: size, Scope: scope, Predicate: predicate}}
		err := Validate(root, at, Page{Limit: 1})
		invalid := size < 1 || size > 10
		if err != nil {
			if !invalid {
				t.Fatalf("expected valid size %d, got %v", size, err)
			}
			continue
		}
		if invalid {
			t.Fatalf("expected invalid size %d, got success", size)
		}
	}
	for _, predicate := range []*OccurrencePredicate{nil, {OccurredIn: &scope},
		{Operator: state.OpOr, Children: []OccurrencePredicate{*predicate, {OccurredIn: &scope}}}} {
		err := Validate(&Rule{LatestN: &LatestN{Size: 2, Scope: scope, Predicate: predicate}}, at, Page{Limit: 1})
		if err == nil {
			t.Fatal("expected invalid block predicate, got success")
		}
	}
}
