// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package profilefilter

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/krenalis/krenalis/warehouses"
)

type outcomeRows struct {
	rows                              [][]any
	position                          int
	scanError, errorAtEnd, closeError error
	closed                            bool
}

// Close records cleanup and returns the configured failure.
func (r *outcomeRows) Close() error {
	r.closed = true
	return r.closeError
}

// Err returns the configured iteration failure.
func (r *outcomeRows) Err() error { return r.errorAtEnd }

// Next advances through the bounded fixture rows.
func (r *outcomeRows) Next() bool {
	if r.position >= len(r.rows) {
		return false
	}
	r.position++
	return true
}

// Scan copies a normalized fixture or returns the configured failure.
func (r *outcomeRows) Scan(dest ...any) error {
	if r.scanError != nil {
		return r.scanError
	}
	copy(dest, r.rows[r.position-1])
	return nil
}

// TestRead rejects incomplete results and always closes the cursor.
func TestRead(t *testing.T) {
	row := []any{"00000000-0000-0000-0000-000000000001", nil, true, 0, 0, 1, 0}
	tests := []struct {
		name string
		rows *outcomeRows
		page warehouses.ProfileOutcomeRequest
		fail bool
	}{
		{"unknown", &outcomeRows{rows: [][]any{row}}, warehouses.ProfileOutcomeRequest{Limit: 1}, false},
		{"source conflict", &outcomeRows{rows: [][]any{{nil, nil, false, 0, 0, 0, 1}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"invalid diagnostic", &outcomeRows{rows: [][]any{{nil, nil, false, 0, 0, 0, "bad"}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"metadata", &outcomeRows{rows: [][]any{{nil, nil, false, 0, 0, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, false},
		{"scan", &outcomeRows{rows: [][]any{row}, scanError: errors.New("scan failed")},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"late error", &outcomeRows{rows: [][]any{row}, errorAtEnd: errors.New("late read failed")},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"close", &outcomeRows{rows: [][]any{row}, closeError: errors.New("close failed")},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"empty stream", &outcomeRows{}, warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"overflow", &outcomeRows{rows: [][]any{{nil, nil, false, math.MaxInt, 1, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"negative", &outcomeRows{rows: [][]any{{nil, nil, false, -1, 0, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"duplicate identity", &outcomeRows{rows: [][]any{row, row}}, warehouses.ProfileOutcomeRequest{Limit: 2}, true},
		{"too many rows", &outcomeRows{rows: [][]any{row, row}}, warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"wrong truth", &outcomeRows{rows: [][]any{{row[0], "true", true, 1, 0, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"wrong marker", &outcomeRows{rows: [][]any{{row[0], true, nil, 1, 0, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"wrong id", &outcomeRows{rows: [][]any{{"bad", true, true, 1, 0, 0, 0}}},
			warehouses.ProfileOutcomeRequest{Limit: 1}, true},
		{"matching only", &outcomeRows{rows: [][]any{row}}, warehouses.ProfileOutcomeRequest{Limit: 1,
			MatchingOnly: true}, true},
		{"metadata mixed", &outcomeRows{rows: [][]any{{nil, nil, false, 1, 0, 0, 0}, row}},
			warehouses.ProfileOutcomeRequest{Limit: 2}, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := Read(t.Context(), test.rows, test.page)
			if err != nil {
				if !test.fail {
					t.Fatalf("expected complete result, got %v", err)
				}
				if result != nil {
					t.Fatalf("expected nil failed result, got %+v", result)
				}
			}
			if test.fail && err == nil {
				t.Fatal("expected operation failure, got success")
			}
			if !test.rows.closed {
				t.Fatal("expected closed rows, got open cursor")
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	rows := &outcomeRows{rows: [][]any{row}}
	result, err := Read(ctx, rows, warehouses.ProfileOutcomeRequest{Limit: 1})
	if err == nil || result != nil || !rows.closed {
		t.Fatalf("expected canceled read with no result and closed rows, got %+v, %v, %v", result, err, rows.closed)
	}
}
