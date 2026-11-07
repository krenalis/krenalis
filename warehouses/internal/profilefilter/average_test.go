// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"errors"
	"testing"

	"github.com/krenalis/krenalis/tools/decimal"
)

// TestReadAverageExplanation preserves exact totals and population uncertainty.
func TestReadAverageExplanation(t *testing.T) {
	const profile = "00000000-0000-0000-0000-000000000001"
	for _, test := range []struct {
		name         string
		lo, hi       int
		lower, upper any
		count        int64
		sum          string
		fail         bool
	}{
		{"determined", 2, 2, "220", "220.0000", 2, "220", false},
		{"empty", 0, 0, "0", "0", 0, "0", false},
		{"unknown amounts", 2, 2, nil, nil, 2, "", false},
		{"unknown population", 1, 2, "100", "220", -1, "", false},
		{"negative", 2, 2, "-220", "-220", 2, "-220", false},
		{"exact large", 2, 2, "90071992547409.1235", "90071992547409.1235", 2, "90071992547409.1235", false},
		{"inconsistent total", 2, 2, "100", "220", 0, "", true},
		{"negative count", -1, 2, "100", "220", 0, "", true},
		{"wrong bound type", 2, 2, 100, "100", 0, "", true},
		{"bound precision", 2, 2, "1e76", "1e76", 0, "", true},
		{"bound scale", 2, 2, "1e-38", "1e-38", 0, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := &outcomeRows{rows: [][]any{{profile, 0, nil, test.lo, test.hi, 0, test.lower, test.upper}}}
			statement := Statement{CountNodes: []bool{false}, SumNodes: []bool{true}, AverageNodes: []bool{true}}
			result, err := statement.ReadExplanation(t.Context(), rows, profile)
			if err != nil {
				if !test.fail || result != nil || !rows.closed {
					t.Fatalf("expected failure=%v without result and with closed cursor, got %+v, %v", test.fail, result, err)
				}
				return
			}
			if test.fail || !rows.closed || len(result) != 1 || result[0].Average == nil || result[0].Sum != nil {
				t.Fatalf("expected Average inputs with failure=%v, got %+v", test.fail, result)
			}
			values := result[0].Average
			if values.Certain != int64(test.lo) || values.Possible != int64(test.hi) ||
				(values.Count == nil) != (test.count < 0) || (values.Sum == nil) != (test.sum == "") {
				t.Fatalf("expected exact sum %s/count %d and population [%d,%d], got %+v",
					test.sum, test.count, test.lo, test.hi, values)
			}
			if values.Count != nil && *values.Count != test.count ||
				values.Sum != nil && !values.Sum.Equal(decimal.MustParse(test.sum)) {
				t.Fatalf("expected exact sum %s/count %d, got %+v", test.sum, test.count, values)
			}
		})
	}
	for _, rows := range []*outcomeRows{
		{rows: [][]any{{profile, 0, true, 2, 2, 0, "220", "220"}}, errorAtEnd: errors.New("iterate")},
		{rows: [][]any{{profile, 0, true, 2, 2, 0, "220", "220"}}, closeError: errors.New("close")},
		{rows: [][]any{{}}, scanError: errors.New("scan")},
	} {
		statement := Statement{CountNodes: []bool{false}, SumNodes: []bool{true}, AverageNodes: []bool{true}}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected technical failure without result and with closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected technical failure, got success")
	}
	for _, statement := range []Statement{
		{CountNodes: []bool{false}, AverageNodes: []bool{true}},
		{CountNodes: []bool{true}, SumNodes: []bool{true}, AverageNodes: []bool{true}},
		{CountNodes: []bool{false}, SumNodes: []bool{true}, AverageNodes: []bool{}},
	} {
		rows := &outcomeRows{}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected invalid metadata without result and with closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected invalid Average metadata rejected, got success")
	}
}
