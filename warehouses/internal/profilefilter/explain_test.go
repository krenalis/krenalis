// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"errors"
	"testing"

	"github.com/krenalis/krenalis/warehouses"
)

func TestReadExplanation(t *testing.T) {
	const profile = "00000000-0000-0000-0000-000000000001"
	for _, test := range []struct {
		name   string
		rows   *outcomeRows
		counts []bool
		fail   bool
	}{
		{"unknown", &outcomeRows{rows: [][]any{{profile, 0, nil, 1, 2, 0}}}, []bool{true}, false},
		{"group", &outcomeRows{rows: [][]any{{profile, 0, false, nil, nil, 0}}}, []bool{false}, false},
		{"missing profile", &outcomeRows{}, []bool{false}, false},
		{"wrong profile", &outcomeRows{rows: [][]any{{"another-profile", 0, true, nil, nil, 0}}}, []bool{false}, true},
		{"wrong index", &outcomeRows{rows: [][]any{{profile, 1, true, nil, nil, 0}}}, []bool{false}, true},
		{"wrong index type", &outcomeRows{rows: [][]any{{profile, "0", true, nil, nil, 0}}}, []bool{false}, true},
		{"wrong truth", &outcomeRows{rows: [][]any{{profile, 0, "true", nil, nil, 0}}}, []bool{false}, true},
		{"source conflict", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 1}}}, []bool{false}, true},
		{"bad diagnostic", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, -1}}}, []bool{false}, true},
		{"bad diagnostic type", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, "0"}}}, []bool{false}, true},
		{"negative count", &outcomeRows{rows: [][]any{{profile, 0, true, -1, 1, 0}}}, []bool{true}, true},
		{"inverted count", &outcomeRows{rows: [][]any{{profile, 0, true, 2, 1, 0}}}, []bool{true}, true},
		{"missing count", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 0}}}, []bool{true}, true},
		{"unexpected count", &outcomeRows{rows: [][]any{{profile, 0, true, 0, 1, 0}}}, []bool{false}, true},
		{"incomplete", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 0}}}, []bool{false, false}, true},
		{"too many", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 0}, {profile, 1, true, nil, nil, 0}}}, []bool{false}, true},
		{"scan failure", &outcomeRows{rows: [][]any{{}}, scanError: errors.New("scan")}, []bool{false}, true},
		{"iteration failure", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 0}}, errorAtEnd: errors.New("iterate")}, []bool{false}, true},
		{"close failure", &outcomeRows{rows: [][]any{{profile, 0, true, nil, nil, 0}}, closeError: errors.New("close")}, []bool{false}, true},
		{"empty close failure", &outcomeRows{closeError: errors.New("close")}, []bool{false}, true},
		{"zero nodes", &outcomeRows{}, nil, true},
		{"too many nodes", &outcomeRows{}, make([]bool, 102), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := ReadExplanation(t.Context(), test.rows, test.counts, profile)
			if err != nil {
				if !test.fail || result != nil || !test.rows.closed {
					t.Fatalf("expected failure=%v with closed cursor and no result, got %+v, %v", test.fail, result, err)
				}
				return
			}
			if test.fail || !test.rows.closed || len(result) != len(test.rows.rows) {
				t.Fatalf("expected complete result with failure=%v and closed cursor, got %+v", test.fail, result)
			}
			if test.name == "unknown" && (result[0].Count == nil || result[0].Count.Certain != 1 ||
				result[0].Count.Possible != 2 || result[0].Truth != warehouses.FilterUnknown) {
				t.Fatalf("expected UNKNOWN with bounds [1,2], got %+v", result)
			}
		})
	}
}

func TestReadExplanationCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rows := &outcomeRows{rows: [][]any{{"00000000-0000-0000-0000-000000000001", 0, true, nil, nil, 0}}}
	result, err := ReadExplanation(ctx, rows, []bool{false}, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		if !errors.Is(err, context.Canceled) || result != nil || !rows.closed {
			t.Fatalf("expected cancellation with closed cursor and no result, got %+v, %v", result, err)
		}
		return
	}
	t.Fatal("expected cancellation, got success")
}
