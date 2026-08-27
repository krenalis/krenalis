// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"math"
	"math/big"
	"strconv"
	"testing"
	"time"

	"github.com/krenalis/krenalis/tools/errors"
)

const testPipelineID = "8QaT3mN7KxP5"

// newStubPipelines returns a Pipelines suitable for pipeline metrics unit tests.
func newStubPipelines() *Pipelines {
	p := &Pipelines{pending: map[string]*pipelineMetrics{}}
	p.stored.L = &p.RWMutex
	return p
}

// TestParsePipelineMetricCount verifies that the maximum int value is converted
// exactly and ErrMetricResultTooLarge is returned for the next larger value.
func TestParsePipelineMetricCount(t *testing.T) {

	got, err := parsePipelineMetricCount(strconv.Itoa(math.MaxInt))
	if err != nil {
		t.Fatal(err)
	}
	if got != math.MaxInt {
		t.Fatalf("expected %d, got %d", math.MaxInt, got)
	}

	var tooLarge big.Int
	tooLarge.SetInt64(int64(math.MaxInt))
	tooLarge.Add(&tooLarge, big.NewInt(1))
	_, err = parsePipelineMetricCount(tooLarge.String())
	if err != nil {
		if !errors.Is(err, ErrMetricResultTooLarge) {
			t.Fatalf("expected ErrMetricResultTooLarge, got %v", err)
		}
		return
	}
	t.Fatal("expected ErrMetricResultTooLarge, got nil")
}

// TestPipelineMetricsPerDateParsesExactAggregates verifies that PostgreSQL
// numeric aggregates are converted exactly and ErrMetricResultTooLarge is
// returned when an aggregate exceeds the int range.
func TestPipelineMetricsPerDateParsesExactAggregates(t *testing.T) {

	database, organization := newAggregateMetricsTestDatabase(t)
	start := time.Date(2026, time.August, 5, 0, 0, 0, 0, time.UTC)
	_, err := database.Exec(t.Context(), `INSERT INTO pipelines_metrics
		(organization, workspace, connection, pipeline, target, timeslot,
		passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6,
		failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6)
		VALUES ($1, 'workspace111', 'pipelineconn', 'pipeline1111', 'Event', $2,
		7, 0, 0, 0, 0, 0, 0, 0, 0, 3, 0, 0, 0, 0)`, organization, TimeSlotFromTime(start))
	if err != nil {
		t.Fatal(err)
	}
	pipelines := Pipelines{metrics: &Metrics{db: database}}
	result, err := pipelines.MetricsPerDate(t.Context(), start, start.AddDate(0, 0, 1),
		PipelineSelection{Workspaces: []string{"workspace111"}, Target: TargetEvent})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Series) != 1 || len(result.Series[0].Passed) != 1 || len(result.Series[0].Failed) != 1 {
		t.Fatalf("expected one daily workspace series, got %#v", result.Series)
	}
	series := result.Series[0]
	if series.Workspace != "workspace111" || series.Passed[0][ReceiveStep] != 7 || series.Failed[0][FilterStep] != 3 {
		t.Fatalf("unexpected pipeline aggregate: %#v", series)
	}

	if _, err := database.Exec(t.Context(),
		"ALTER TABLE pipelines_metrics ALTER COLUMN passed_0 TYPE numeric"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(t.Context(), "DELETE FROM pipelines_metrics"); err != nil {
		t.Fatal(err)
	}
	for _, pipeline := range []string{"overflowp111", "overflowp222"} {
		_, err := database.Exec(t.Context(), `INSERT INTO pipelines_metrics
			(organization, workspace, connection, pipeline, target, timeslot,
			passed_0, passed_1, passed_2, passed_3, passed_4, passed_5, passed_6,
			failed_0, failed_1, failed_2, failed_3, failed_4, failed_5, failed_6)
			VALUES ($1, 'workspace111', 'pipelineconn', $2, 'Event', $3,
			$4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`, organization, pipeline,
			TimeSlotFromTime(start), math.MaxInt)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = pipelines.MetricsPerDate(t.Context(), start, start.AddDate(0, 0, 1),
		PipelineSelection{Workspaces: []string{"workspace111"}, Target: TargetEvent})
	if err != nil {
		if !errors.Is(err, ErrMetricResultTooLarge) {
			t.Fatalf("expected ErrMetricResultTooLarge, got %v", err)
		}
		return
	}
	t.Fatal("expected ErrMetricResultTooLarge, got nil")
}

// Test_PipelinesInvalidStep verifies that using an invalid step causes a panic.
func Test_PipelinesInvalidStep(t *testing.T) {
	p := newStubPipelines()
	p.pending[testPipelineID] = &pipelineMetrics{}
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic with invalid step, got no panic")
		}
	}()
	p.Passed(PipelineStep(numSteps), testPipelineID, 1)
}

// Test_PipelinesPassedFailed ensures that Passed and Failed record metrics
// correctly.
func Test_PipelinesPassedFailed(t *testing.T) {
	p := newStubPipelines()
	p.pending[testPipelineID] = &pipelineMetrics{}
	p.Passed(ReceiveStep, testPipelineID, 3)
	p.Failed(FilterStep, testPipelineID, 2, "boom")

	m, ok := p.pending[testPipelineID]
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

// Test_PipelineStepString verifies that String returns the expected label for
// each PipelineStep.
func Test_PipelineStepString(t *testing.T) {
	tests := map[PipelineStep]string{
		ReceiveStep:          "Receive",
		InputValidationStep:  "InputValidation",
		FilterStep:           "Filter",
		ConsentStep:          "Consent",
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

// Test_PipelineStepString_invalid checks that String panics for an undefined
// PipelineStep.
func Test_PipelineStepString_invalid(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid step, got no panic")
		}
	}()
	_ = PipelineStep(99).String()
}

// Test_PipelineTargetString verifies that String returns the expected label for
// each PipelineTarget.
func Test_PipelineTargetString(t *testing.T) {
	tests := map[PipelineTarget]string{
		TargetNone:  "None",
		TargetEvent: "Event",
		TargetUser:  "User",
		TargetGroup: "Group",
	}
	for target, want := range tests {
		if got := target.String(); got != want {
			t.Fatalf("%v: expected %q, got %q", target, want, got)
		}
	}
}

// Test_PipelineTargetString_invalid checks that String panics for an undefined
// PipelineTarget.
func Test_PipelineTargetString_invalid(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for invalid target, got no panic")
		}
	}()
	_ = PipelineTarget(99).String()
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
