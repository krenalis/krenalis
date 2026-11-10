// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package warehouses

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/types"
)

// TestValidateLastPrevious checks the closed physical node before rendering.
func TestValidateLastPrevious(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	threshold := decimal.MustParse("-30")
	node := LastPreviousDifference{Source: OccurrenceSource{Table: "events",
		Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "id", Type: types.String()},
		Timestamp: Column{Name: "timestamp", Type: types.DateTime()}}, Start: end.AddDate(0, -1, 0), End: end,
		Value: DecimalReading{Column: Column{Name: "properties", Type: types.JSON()}, Keys: []string{"total"},
			Type: types.Decimal(18, 4)}, Threshold: &threshold}
	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles",
		Key: Column{Name: "id", Type: types.UUID()}}, Limit: 1}
	for _, test := range []struct {
		name    string
		change  func(*LastPreviousDifference)
		invalid bool
	}{
		{"bound", func(p *LastPreviousDifference) {}, false},
		{"none", func(p *LastPreviousDifference) { p.Threshold = nil; p.ReadableDifferences = "none" }, false},
		{"all", func(p *LastPreviousDifference) { p.Threshold = nil; p.ReadableDifferences = "all" }, false},
		{"missing", func(p *LastPreviousDifference) { p.Threshold = nil }, true},
		{"both", func(p *LastPreviousDifference) { p.ReadableDifferences = "all" }, true},
		{"bad domain", func(p *LastPreviousDifference) { p.Threshold = nil; p.ReadableDifferences = "other" }, true},
		{"direction on domain", func(p *LastPreviousDifference) { p.Threshold = nil; p.ReadableDifferences = "all"; p.AtLeast = true }, true},
		{"scale", func(p *LastPreviousDifference) { v := decimal.MustParse("1.00001"); p.Threshold = &v }, true},
		{"upper outside", func(p *LastPreviousDifference) { v := decimal.MustParse("199999999999999.9999"); p.Threshold = &v }, true},
		{"lower outside", func(p *LastPreviousDifference) { v := decimal.MustParse("-199999999999999.9999"); p.Threshold = &v }, true},
		{"upper endpoint", func(p *LastPreviousDifference) { v := decimal.MustParse("199999999999999.9998"); p.Threshold = &v }, false},
		{"lower endpoint", func(p *LastPreviousDifference) { v := decimal.MustParse("-199999999999999.9998"); p.Threshold = &v }, false},
		{"identity", func(p *LastPreviousDifference) { p.Source.Identity = Column{} }, true},
		{"period", func(p *LastPreviousDifference) { p.Start = p.End }, true},
		{"timestamp precision", func(p *LastPreviousDifference) { p.End = p.End.Add(time.Nanosecond) }, true},
		{"unbounded start", func(p *LastPreviousDifference) { p.UnboundedStart = true }, true},
		{"unbounded", func(p *LastPreviousDifference) { p.UnboundedStart = true; p.Start = time.Time{} }, false},
		{"reading", func(p *LastPreviousDifference) { p.Value.Type = types.Decimal(19, 4) }, true},
		{"qualification", func(p *LastPreviousDifference) { p.Qualification = &RecordQualification{} }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			modified := node
			test.change(&modified)
			err := ValidateProfileFilter(Filter{Root: &modified}, request)
			if err != nil {
				if !test.invalid {
					t.Fatalf("expected valid prepared difference, got %v", err)
				}
				return
			}
			if test.invalid {
				t.Fatal("expected invalid prepared difference, got success")
			}
		})
	}
	for _, size := range []int{100, 101} {
		root := &FilterGroup{Operator: OpAnd, Children: make([]FilterCondition, size)}
		for i := range root.Children {
			root.Children[i] = &node
		}
		err := ValidateProfileFilter(Filter{Root: root}, request)
		if err != nil {
			if size == 100 {
				t.Fatalf("expected valid root budget, got %v", err)
			}
			continue
		}
		if size == 101 {
			t.Fatal("expected excess root budget rejection, got success")
		}
	}
}
