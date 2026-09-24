// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/tools/errors"
	"github.com/krenalis/krenalis/tools/json"
)

// TestValidateWorkspaceMetricsRange verifies UTC normalization and the
// intrinsic daily-range rules used by workspace identity metrics.
func TestValidateWorkspaceMetricsRange(t *testing.T) {

	now := time.Date(2200, time.January, 1, 12, 0, 0, 0, time.UTC)
	location := time.FixedZone("test", 2*60*60)
	start, end, err := validateWorkspaceMetricsRange(
		time.Date(2026, time.August, 1, 23, 0, 0, 0, location),
		time.Date(2026, time.August, 4, 23, 0, 0, 0, location), now)
	if err != nil {
		t.Fatal(err)
	}
	wantStart := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC)
	if start != wantStart || end != wantEnd {
		t.Fatalf("expected normalized range %s/%s, got %s/%s", wantStart, wantEnd, start, end)
	}

	tests := []struct {
		name       string
		start, end time.Time
		now        time.Time
		message    string
	}{
		{
			name: "start before minimum", start: metrics.MinTime.AddDate(0, 0, -1),
			end: metrics.MinTime.AddDate(0, 0, 1), now: now,
			message: "start date is too far in the past",
		},
		{
			name: "end after maximum", start: metrics.MaxTime.AddDate(0, 0, -2),
			end: metrics.MaxTime.AddDate(0, 0, 1), now: metrics.MaxTime,
			message: "end date is too far in the future",
		},
		{
			name: "end after next midnight", start: time.Date(2026, time.August, 8, 0, 0, 0, 0, time.UTC),
			end:     time.Date(2026, time.August, 11, 0, 0, 0, 0, time.UTC),
			now:     time.Date(2026, time.August, 9, 12, 0, 0, 0, time.UTC),
			message: "end date is too far in the future",
		},
		{
			name: "empty range", start: time.Date(2026, time.August, 9, 0, 0, 0, 0, time.UTC),
			end: time.Date(2026, time.August, 9, 23, 0, 0, 0, time.UTC), now: now,
			message: "day of the end date must be after the day of the start date",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := validateWorkspaceMetricsRange(test.start, test.end, test.now)
			if err != nil {
				if _, ok := err.(errors.ResponseWriterTo); ok {
					t.Fatalf("expected a generic validation error, got %T", err)
				}
				if !strings.Contains(err.Error(), test.message) {
					t.Fatalf("expected error containing %q, got %v", test.message, err)
				}
				return
			}
			t.Fatal("expected an error, got nil")
		})
	}

}

// TestIdentityMetricsPerDateValidatesBeforeMetrics verifies that invalid public
// arguments return before refresh or database access.
func TestIdentityMetricsPerDateValidatesBeforeMetrics(t *testing.T) {

	workspace := Workspace{core: &Core{}}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	_, err := workspace.IdentityMetricsPerDate(t.Context(), start, start, nil)
	if err != nil {
		if _, ok := errors.AsType[*errors.NotFoundError](err); !ok {
			t.Fatalf("expected an empty interval to return NotFoundError, got %T: %v", err, err)
		}
		return
	}
	t.Fatal("expected an error, got nil")
}

// TestIdentityMetricsPerDateConnectionSelectionValidation verifies that
// connection selection is validated before reading workspace state or metrics.
func TestIdentityMetricsPerDateConnectionSelectionValidation(t *testing.T) {
	workspace := Workspace{core: &Core{}}
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.AddDate(0, 0, 1)
	_, err := workspace.IdentityMetricsPerDate(t.Context(), start, end, new("invalid"))
	if err != nil {
		if err.Error() != `value "invalid" is neither a valid connection identifier nor "deleted"` {
			t.Fatalf("expected an invalid connection error, got %v", err)
		}
		return
	}
	t.Fatal("expected an invalid connection error, got nil")
}

// TestWorkspaceIdentityMetricMethodsCheckCoreOpen verifies that the public
// methods preserve the existing closed-Core panic behavior.
func TestWorkspaceIdentityMetricMethodsCheckCoreOpen(t *testing.T) {
	for _, test := range []struct {
		name string
		call func(Workspace)
	}{
		{name: "identity days", call: func(workspace Workspace) {
			_, _ = workspace.IdentityMetricsPerDate(t.Context(), time.Time{}, time.Time{}, nil)
		}},
		{name: "latest identity", call: func(workspace Workspace) {
			_, _ = workspace.LatestIdentityMetric(t.Context())
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			core := &Core{}
			core.closed.Store(true)
			deferred := false
			defer func() {
				if recovered := recover(); recovered == nil {
					t.Fatal("expected a closed-Core panic, got nil")
				}
				deferred = true
			}()
			test.call(Workspace{core: core})
			if deferred {
				t.Fatal("expected panic to stop execution, got a normal return")
			}
		})
	}
}

// TestIdentityMetricConnectionsJSON verifies that an empty latest breakdown is
// encoded as an empty array.
func TestIdentityMetricConnectionsJSON(t *testing.T) {
	empty, err := json.Marshal(IdentityMetric{Connections: []IdentityConnectionMetric{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(empty), `"connections":[]`) {
		t.Fatalf("expected an empty connections array, got %s", empty)
	}
}
