// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"testing"
)

type stubCollector struct{ Collector }

func newStubCollector() *stubCollector {
	c := &stubCollector{Collector{metrics: map[int]*metrics{}}}
	c.stored.L = &c.mu
	return c
}

// Test_CollectorInvalidStep verifies that using an invalid step causes a panic.
func Test_CollectorInvalidStep(t *testing.T) {
	c := newStubCollector()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic with invalid step")
		}
	}()
	c.Passed(Step(numSteps), 1, 1)
}

// Test_CollectorPassedFailed ensures that Passed and Failed record metrics
// correctly.
func Test_CollectorPassedFailed(t *testing.T) {
	c := newStubCollector()
	c.Passed(ReceiveStep, 1, 3)
	c.Failed(FilterStep, 1, 2, "boom")

	m, ok := c.metrics[1]
	if !ok {
		t.Fatalf("metrics for pipeline 1 not found")
	}
	if got := m.passed[ReceiveStep]; got != 3 {
		t.Fatalf("expected 3 passed, got %d", got)
	}
	if got := m.failed[FilterStep]; got != 2 {
		t.Fatalf("expected 2 failed, got %d", got)
	}
	if len(m.errors) != 1 || m.errors[0].step != FilterStep || m.errors[0].count != 2 || m.errors[0].message != "boom" {
		t.Fatalf("unexpected errors: %#v", m.errors)
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
			t.Fatalf("expected panic for invalid step")
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
					t.Errorf("TimeSlotToTime(%d) did not panic", ts)
				}
			}()
			TimeSlotToTime(ts)
		}()
	}
}
