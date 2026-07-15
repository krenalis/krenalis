// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"testing"
)

type stubCollector struct{ Collector }

const testPipelineID = "8QaT3mN7KxP5"

// newStubCollector returns a collector suitable for metrics unit tests.
func newStubCollector() *stubCollector {
	c := &stubCollector{Collector{metrics: map[string]*metrics{}}}
	c.stored.L = &c.mu
	return c
}

// Test_CollectorInvalidStep verifies that using an invalid step causes a panic.
func Test_CollectorInvalidStep(t *testing.T) {
	c := newStubCollector()
	c.metrics[testPipelineID] = &metrics{}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic with invalid step, got no panic")
		}
	}()
	c.Passed(Step(numSteps), testPipelineID, 1)
}

// Test_CollectorPassedFailed ensures that Passed and Failed record metrics
// correctly.
func Test_CollectorPassedFailed(t *testing.T) {
	c := newStubCollector()
	c.metrics[testPipelineID] = &metrics{}
	c.Passed(ReceiveStep, testPipelineID, 3)
	c.Failed(FilterStep, testPipelineID, 2, "boom")

	m, ok := c.metrics[testPipelineID]
	if !ok {
		t.Fatalf("expected metrics for pipeline %s, got none", testPipelineID)
	}
	if got := m.passed[ReceiveStep]; got != 3 {
		t.Fatalf("expected 3 passed, got %d", got)
	}
	if got := m.failed[FilterStep]; got != 2 {
		t.Fatalf("expected 2 failed, got %d", got)
	}
	if len(m.errors) != 1 || m.errors[0].step != FilterStep || m.errors[0].count != 2 || m.errors[0].message != "boom" {
		t.Fatalf("expected one filter error with message %q, got %#v", "boom", m.errors)
	}
}

// TestSelectionGroup verifies that Selection.group returns the IDs for the
// selected grouping dimension.
func TestSelectionGroup(t *testing.T) {
	tests := []struct {
		name      string
		selection Selection
		wantGroup metricGroup
		wantIDs   []string
	}{
		{
			name: "workspaces",
			selection: Selection{
				Workspaces: []string{"workspace-1", "workspace-2"},
			},
			wantGroup: groupWorkspace,
			wantIDs:   []string{"workspace-1", "workspace-2"},
		},
		{
			name: "connections",
			selection: Selection{
				Connections: []string{"connection-1", "connection-2"},
			},
			wantGroup: groupConnection,
			wantIDs:   []string{"connection-1", "connection-2"},
		},
		{
			name: "empty pipelines",
			selection: Selection{
				Pipelines: []string{},
			},
			wantGroup: groupPipeline,
			wantIDs:   []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			group, got, err := test.selection.group()
			if err != nil {
				t.Fatal(err)
			}
			if group != test.wantGroup {
				t.Fatalf("expected group %q, got %q", test.wantGroup, group)
			}
			if len(got) != len(test.wantIDs) {
				t.Fatalf("expected IDs %v, got %v", test.wantIDs, got)
			}
			for i := range got {
				if got[i] != test.wantIDs[i] {
					t.Fatalf("expected IDs %v, got %v", test.wantIDs, got)
				}
			}
		})
	}
}

// TestNewMetricSeries verifies that newMetricSeries creates zero-filled
// buckets for the requested group.
func TestNewMetricSeries(t *testing.T) {
	series := newMetricSeries(groupPipeline, testPipelineID, 3)

	if series.Pipeline != testPipelineID {
		t.Fatalf("expected pipeline %q, got %q", testPipelineID, series.Pipeline)
	}
	if len(series.Passed) != 3 {
		t.Fatalf("expected 3 passed buckets, got %d", len(series.Passed))
	}
	if len(series.Failed) != 3 {
		t.Fatalf("expected 3 failed buckets, got %d", len(series.Failed))
	}
}

// TestQueryMetricsEmptySelection verifies that an empty selected group returns
// empty metrics without querying the database.
func TestQueryMetricsEmptySelection(t *testing.T) {
	c := &Collector{}
	start := MinTime
	end := start.Add(Day)

	metrics, err := c.MetricsPerDate(t.Context(), start, end, Selection{
		Pipelines: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !metrics.Start.Equal(start) || !metrics.End.Equal(end) {
		t.Fatalf("expected interval [%s,%s), got [%s,%s)", start, end, metrics.Start, metrics.End)
	}
	if len(metrics.Series) != 0 {
		t.Fatalf("expected no series, got %#v", metrics.Series)
	}
}

// Test_StepString verifies that String returns the expected label for each
// Step.
func Test_StepString(t *testing.T) {
	tests := map[Step]string{
		ReceiveStep:          "Receive",
		InputValidationStep:  "InputValidation",
		FilterStep:           "Filter",
		TransformationStep:   "Transformation",
		OutputValidationStep: "OutputValidation",
		FinalizeStep:         "Finalize",
	}
	for s, want := range tests {
		if got := s.String(); got != want {
			t.Fatalf("%v: expected %q, got %q", s, want, got)
		}
	}
}

// Test_StepString_invalid checks that String panics for an undefined Step.
func Test_StepString_invalid(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid step, got no panic")
		}
	}()
	_ = Step(99).String()
}

// Test_TimeSlot checks round-trip conversion between time slots and times.
func Test_TimeSlot(t *testing.T) {

	tests := []int32{0, 1, 5, 99, 28714101, maxTimeslot}

	for _, ts := range tests {
		got := TimeSlotFromTime(TimeSlotToTime(ts))
		if ts != got {
			t.Fatalf("expected %d, got %d", ts, got)
		}
	}

}

// Test_TimeSlotToTime_OutOfRange checks that TimeSlotToTime panics when the
// slot is outside the valid range.
func Test_TimeSlotToTime_OutOfRange(t *testing.T) {
	tests := []int32{-1, maxTimeslot + 1}
	for _, ts := range tests {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("expected TimeSlotToTime(%d) to panic, got no panic", ts)
				}
			}()
			TimeSlotToTime(ts)
		}()
	}
}
