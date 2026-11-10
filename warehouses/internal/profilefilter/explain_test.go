// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"errors"
	"strings"
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

// TestReadLastPreviousExplanation checks pair metadata and cursor failures.
func TestReadLastPreviousExplanation(t *testing.T) {

	const profile = "00000000-0000-0000-0000-000000000001"
	for _, test := range []struct {
		name                       string
		candidates, absent, unique any
		invalid                    bool
	}{
		{"unique", 2, 0, true, false}, {"tie", 2, 0, false, false},
		{"ambiguous", 4, 0, false, false}, {"possible absence", 2, 1, false, false},
		{"certain absence", 0, 1, false, false}, {"negative", -1, 1, false, true},
		{"one identity", 1, 1, false, true}, {"empty present", 0, 0, false, true},
		{"unique empty", 0, 1, true, true}, {"unique optional", 2, 1, true, true},
		{"unique too many", 3, 0, true, true}, {"missing unique", 2, 0, nil, true},
		{"wrong unique", 2, 0, "true", true}, {"wrong count", "2", 0, true, true},
		{"missing count", nil, 0, true, true}, {"wrong absence", 2, true, false, true},
		{"bad absence", 2, 2, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := &outcomeRows{rows: [][]any{{profile, 0, nil, test.candidates, test.absent, 0, test.unique}}}
			statement := Statement{CountNodes: []bool{false}, PairNodes: []bool{true}}
			result, err := statement.ReadExplanation(t.Context(), rows, profile)
			if err != nil {
				if !test.invalid || result != nil || !rows.closed {
					t.Fatalf("expected failure=%v without partial result, got %+v, %v", test.invalid, result, err)
				}
				return
			}
			if test.invalid || !rows.closed || len(result) != 1 || result[0].Pair == nil ||
				result[0].Pair.Candidates != int64(test.candidates.(int)) ||
				result[0].Pair.AbsencePossible != (test.absent == 1) || result[0].Pair.Unique != test.unique {
				t.Fatalf("expected complete ordered-pair metadata, got %+v", result)
			}
		})
	}

	for _, rows := range []*outcomeRows{
		{rows: [][]any{{}}, scanError: errors.New("scan")},
		{rows: [][]any{{profile, 0, nil, 2, 1, 0, false}}, errorAtEnd: errors.New("iterate")},
		{rows: [][]any{{profile, 0, nil, 2, 1, 0, false}}, closeError: errors.New("close")},
		{rows: [][]any{{profile, 0, nil, 2, 1, 1, false}}},
	} {
		statement := Statement{CountNodes: []bool{false}, PairNodes: []bool{true}}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected failure without partial result and closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected technical failure, got success")
	}

	// Mixed trees keep their metadata distinct, including the Sum text columns.
	rows := &outcomeRows{rows: [][]any{
		{profile, 0, nil, nil, nil, 0, nil, nil, nil},
		{profile, 1, true, 2, 0, 0, nil, nil, true},
		{profile, 2, true, 2, 3, 0, "10", "20", nil},
	}}
	statement := Statement{CountNodes: []bool{false, false, false}, SumNodes: []bool{false, false, true},
		PairNodes: []bool{false, true, false}}
	result, err := statement.ReadExplanation(t.Context(), rows, profile)
	if err != nil {
		t.Fatalf("expected mixed protocol, got %v", err)
	}
	if !rows.closed || result[0].Pair != nil || result[1].Pair == nil || result[2].Sum == nil ||
		result[2].Pair != nil {
		t.Fatalf("expected separate pair and Sum metadata, got %+v", result)
	}
	for _, statement := range []Statement{
		{CountNodes: []bool{true}, PairNodes: []bool{true}},
		{CountNodes: []bool{false}, SumNodes: []bool{true}, PairNodes: []bool{true}},
		{CountNodes: []bool{false}, ExperienceNodes: []bool{true}, PairNodes: []bool{true}},
		{CountNodes: []bool{false}, PairNodes: []bool{true, false}},
	} {
		rows := &outcomeRows{}
		_, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err == nil || !rows.closed {
			t.Fatalf("expected conflicting metadata rejected and cursor closed, got %v", err)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rows = &outcomeRows{rows: [][]any{{profile, 0, true, 2, 0, 0, true}}}
	statement = Statement{CountNodes: []bool{false}, PairNodes: []bool{true}}
	result, err = statement.ReadExplanation(ctx, rows, profile)
	if err != nil {
		if !errors.Is(err, context.Canceled) || result != nil || !rows.closed {
			t.Fatalf("expected cancellation without partial result and closed cursor, got %+v, %v", result, err)
		}
		return
	}
	t.Fatal("expected cancellation, got success")

}

// TestReadSumExplanation validates numeric bounds before exposing any result.
func TestReadSumExplanation(t *testing.T) {
	const profile = "00000000-0000-0000-0000-000000000001"
	for _, test := range []struct {
		name         string
		lower, upper any
		fail         bool
	}{
		{"finite", "-20.0001", "35.0000", false},
		{"unbounded", nil, nil, false},
		{"one unbounded", nil, "30", false},
		{"exact large", "9007199254740993.0001", "9007199254740993.0002", false},
		{"inverted", "20.0001", "20", true},
		{"wrong type", 20, "30", true},
		{"invalid", "not-a-number", "30", true},
		{"nonfinite", "Infinity", nil, true},
		{"precision", "1e76", nil, true},
		{"scale", "1e-38", nil, true},
		{"oversize", strings.Repeat("1", 129), nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := &outcomeRows{rows: [][]any{{profile, 0, nil, 1, 3, 0, test.lower, test.upper}}}
			statement := Statement{CountNodes: []bool{false}, SumNodes: []bool{true}}
			result, err := statement.ReadExplanation(t.Context(), rows, profile)
			if err != nil {
				if !test.fail || result != nil || !rows.closed {
					t.Fatalf("expected failure=%v with closed cursor and no result, got %+v, %v", test.fail, result, err)
				}
				return
			}
			if test.fail || !rows.closed || len(result) != 1 || result[0].Sum == nil || result[0].Count != nil {
				t.Fatalf("expected valid Sum bounds with failure=%v, got %+v", test.fail, result)
			}
			if result[0].Sum.Certain != 1 || result[0].Sum.Possible != 3 {
				t.Fatalf("expected qualified count [1,3], got %+v", result[0].Sum)
			}
		})
	}
	for _, pair := range [][2]any{{nil, nil}, {-1, 1}, {2, 1}, {0, nil}, {"0", 1}} {
		rows := &outcomeRows{rows: [][]any{{profile, 0, nil, pair[0], pair[1], 0, "0", "30"}}}
		statement := Statement{CountNodes: []bool{false}, SumNodes: []bool{true}}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected invalid count failure with closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected invalid Sum qualification counts, got success")
	}
	for _, rows := range []*outcomeRows{
		{rows: [][]any{{profile, 0, nil, 0, 1, 0, "0", "30"}}, errorAtEnd: errors.New("iterate")},
		{rows: [][]any{{profile, 0, nil, 0, 1, 0, "0", "30"}}, closeError: errors.New("close")},
		{rows: [][]any{{}}, scanError: errors.New("scan")},
	} {
		statement := Statement{CountNodes: []bool{false}, SumNodes: []bool{true}}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected failure without partial result and closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected technical failure, got success")
	}
}
