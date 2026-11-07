// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"errors"
	"testing"
)

// TestReadExperienceExplanation checks metadata and complete-or-error reads.
func TestReadExperienceExplanation(t *testing.T) {

	const profile = "00000000-0000-0000-0000-000000000001"
	for _, test := range []struct {
		name               string
		candidates, absent any
		fail               bool
	}{
		{"certain absence", 0, 1, false},
		{"possible absence", 1, 1, false},
		{"unique", 1, 0, false},
		{"ambiguous", 2, 0, false},
		{"large", int(9223372036854775807), 0, false},
		{"negative", -1, 0, true},
		{"missing", nil, 1, true},
		{"wrong type", "1", 1, true},
		{"wrong absence type", 1, true, true},
		{"negative absence", 1, -1, true},
		{"invalid absence", 1, 2, true},
		{"empty nonempty selection", 0, 0, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			rows := &outcomeRows{rows: [][]any{{profile, 0, nil, test.candidates, test.absent, 0}}}
			statement := Statement{CountNodes: []bool{false}, ExperienceNodes: []bool{true}}
			result, err := statement.ReadExplanation(t.Context(), rows, profile)
			if err != nil {
				if !test.fail || result != nil || !rows.closed {
					t.Fatalf("expected failure=%v without partial result and closed cursor, got %+v, %v", test.fail, result, err)
				}
				return
			}
			if test.fail || !rows.closed || len(result) != 1 || result[0].Experience == nil || result[0].Count != nil {
				t.Fatalf("expected selection metadata with failure=%v and closed cursor, got %+v", test.fail, result)
			}
			selection := result[0].Experience
			if selection.Candidates != int64(test.candidates.(int)) || selection.AbsencePossible != (test.absent == 1) {
				t.Fatalf("expected selection %v/%v, got %+v", test.candidates, test.absent, selection)
			}
		})
	}

	for _, rows := range []*outcomeRows{
		{rows: [][]any{{}}, scanError: errors.New("scan")},
		{rows: [][]any{{profile, 0, true, 1, 0, 0}}, errorAtEnd: errors.New("iterate")},
		{rows: [][]any{{profile, 0, true, 1, 0, 0}}, closeError: errors.New("close")},
	} {
		statement := Statement{CountNodes: []bool{false}, ExperienceNodes: []bool{true}}
		result, err := statement.ReadExplanation(t.Context(), rows, profile)
		if err != nil {
			if result != nil || !rows.closed {
				t.Fatalf("expected no partial result and closed cursor, got %+v", result)
			}
			continue
		}
		t.Fatal("expected technical failure, got success")
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rows := &outcomeRows{rows: [][]any{{profile, 0, true, 1, 0, 0}}}
	statement := Statement{CountNodes: []bool{false}, ExperienceNodes: []bool{true}}
	result, err := statement.ReadExplanation(ctx, rows, profile)
	if err != nil {
		if !errors.Is(err, context.Canceled) || result != nil || !rows.closed {
			t.Fatalf("expected cancellation without partial result and closed cursor, got %+v, %v", result, err)
		}
		return
	}
	t.Fatal("expected cancellation, got success")

}
