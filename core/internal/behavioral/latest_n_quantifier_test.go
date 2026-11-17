// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package behavioral

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
)

// TestLatestNQuantifier checks canonical operands and exact bounded execution.
func TestLatestNQuantifier(t *testing.T) {
	for _, test := range []struct {
		name         string
		quantifier   LatestNQuantifier
		size         int
		lower, upper int64
		invalid      bool
	}{
		{"default All", LatestNQuantifier{}, 3, 3, 3, false},
		{"Some", LatestNQuantifier{Kind: LatestSome}, 3, 1, 3, false},
		{"zero", LatestNQuantifier{Kind: LatestCardinality, Value: "0"}, 3, 0, 0, false},
		{"exact N", LatestNQuantifier{Kind: LatestCardinality, Value: "3"}, 3, 3, 3, false},
		{"at least N", LatestNQuantifier{Kind: LatestCardinality, Operator: GreaterOrEqual, Value: "3"}, 3, 3, 3, false},
		{"at most huge", LatestNQuantifier{Kind: LatestCardinality, Operator: LessOrEqual, Value: "1000000000000000000000000000000"}, 3, 0, 3, false},
		{"between redundant upper", LatestNQuantifier{Kind: LatestCardinality, Operator: Between, Value: "2", Upper: "1000000000000000000000000000000"}, 3, 2, 3, false},
		{"between point", LatestNQuantifier{Kind: LatestCardinality, Operator: Between, Value: "3", Upper: "3"}, 3, 3, 3, false},
		{"All operands", LatestNQuantifier{Value: "0"}, 3, 0, 0, true},
		{"Some operands", LatestNQuantifier{Kind: LatestSome, Upper: "0"}, 3, 0, 0, true},
		{"unknown kind", LatestNQuantifier{Kind: 99}, 3, 0, 0, true},
		{"missing value", LatestNQuantifier{Kind: LatestCardinality}, 3, 0, 0, true},
		{"negative", LatestNQuantifier{Kind: LatestCardinality, Value: "-1"}, 3, 0, 0, true},
		{"negative zero not canonical", LatestNQuantifier{Kind: LatestCardinality, Value: "-0"}, 3, 0, 0, true},
		{"fraction", LatestNQuantifier{Kind: LatestCardinality, Value: "1.0"}, 3, 0, 0, true},
		{"exponent", LatestNQuantifier{Kind: LatestCardinality, Value: "1E2"}, 3, 0, 0, true},
		{"leading zero", LatestNQuantifier{Kind: LatestCardinality, Value: "01"}, 3, 0, 0, true},
		{"extra upper", LatestNQuantifier{Kind: LatestCardinality, Value: "1", Upper: "2"}, 3, 0, 0, true},
		{"inverted huge", LatestNQuantifier{Kind: LatestCardinality, Operator: Between, Value: "1000000000000000000000000000000", Upper: "4"}, 3, 0, 0, true},
		{"incompatible huge", LatestNQuantifier{Kind: LatestCardinality, Value: "1000000000000000000000000000000"}, 3, 0, 0, true},
		{"incompatible N+1", LatestNQuantifier{Kind: LatestCardinality, Value: "4"}, 3, 0, 0, true},
		{"inverted", LatestNQuantifier{Kind: LatestCardinality, Operator: Between, Value: "2", Upper: "1"}, 3, 0, 0, true},
		{"strict greater", LatestNQuantifier{Kind: LatestCardinality, Operator: Greater, Value: "1"}, 3, 0, 0, true},
		{"different", LatestNQuantifier{Kind: LatestCardinality, Operator: Different, Value: "1"}, 3, 0, 0, true},
		{"invalid size", LatestNQuantifier{}, 0, 0, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			at := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
			root := &Rule{LatestN: &LatestN{Size: test.size, Quantifier: test.quantifier,
				Scope: Scope{Event: "Order Completed", Grain: "registered-message", Start: at.Add(-time.Hour), End: at},
				Predicate: &OccurrencePredicate{Record: &state.WhereCondition{
					Property: []string{"event"}, Operator: state.OpExists}}}}
			err := Validate(root, at, Page{Limit: 1})
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid direct semantic quantifier, got %v", err)
				}
			}
			if err == nil && test.invalid {
				t.Fatal("expected invalid direct semantic quantifier, got success")
			}
			accepted, err := test.quantifier.Accepted(test.size)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected accepted range [%d,%d], got %v", test.lower, test.upper, err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid block quantifier, got success")
			}
			if test.quantifier.Kind == LatestAll {
				if accepted != nil {
					t.Fatalf("expected nil legacy All range, got %+v", accepted)
				}
				return
			}
			if accepted == nil || accepted.Upper == nil || accepted.Lower != test.lower || *accepted.Upper != test.upper {
				t.Fatalf("expected accepted range [%d,%d], got %+v", test.lower, test.upper, accepted)
			}
		})
	}
}
