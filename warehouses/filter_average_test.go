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

// TestValidateAverageFilter bounds malformed prepared inputs before rendering.
func TestValidateAverageFilter(t *testing.T) {
	end := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	request := ProfileOutcomeRequest{Population: ProfileSource{Table: "profiles",
		Key: Column{Name: "_kpid", Type: types.UUID()}}, Limit: 1}
	for _, test := range []struct {
		name   string
		change func(*DeterminedOccurrenceAverage)
	}{
		{"minimum omitted", func(a *DeterminedOccurrenceAverage) { a.Minimum = 0 }},
		{"negative minimum", func(a *DeterminedOccurrenceAverage) { a.Minimum = -1 }},
		{"finer threshold", func(a *DeterminedOccurrenceAverage) { a.Threshold = decimal.MustParse("0.00001") }},
		{"large threshold", func(a *DeterminedOccurrenceAverage) { a.Threshold = decimal.MustParse("1e14") }},
		{"constant with threshold", func(a *DeterminedOccurrenceAverage) { value := true; a.DeterminedResult = &value }},
		{"unsafe table", func(a *DeterminedOccurrenceAverage) { a.Source.Table = "a.b" }},
		{"empty period", func(a *DeterminedOccurrenceAverage) { a.Start = a.End }},
		{"wrong reading type", func(a *DeterminedOccurrenceAverage) { a.Value.Type = types.Float(64) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			node := &DeterminedOccurrenceAverage{Source: OccurrenceSource{Table: "events",
				Profile: Column{Name: "kpid", Type: types.UUID()}, Identity: Column{Name: "id", Type: types.String()},
				Timestamp: Column{Name: "timestamp", Type: types.DateTime()}},
				Start: end.AddDate(0, -1, 0), End: end, Minimum: 2, Threshold: decimal.MustParse("100"),
				Value: DecimalReading{Column: Column{Name: "properties", Type: types.JSON()},
					Keys: []string{"total"}, Type: types.Decimal(18, 4)}}
			err := ValidateProfileFilter(Filter{Root: node}, request)
			if err != nil {
				t.Fatalf("expected valid prepared fixture, got %v", err)
			}
			test.change(node)
			err = ValidateProfileFilter(Filter{Root: node}, request)
			if err == nil {
				t.Fatal("expected malformed Average rejected, got success")
			}
		})
	}
}
