// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"math"
	"testing"
	"time"
)

// Test that Set clamps errorRate in [0,1] and update never exceeds bounds.
func TestErrorRate_BoundsAndClamp(t *testing.T) {

	var er errorRate

	// Test Set clamps to [0, 1]
	er.Set(-2)
	if er.rate != 0 {
		t.Errorf("Set(-2): got %f, want 0", er.rate)
	}
	er.Set(1.5)
	if er.rate != 1 {
		t.Errorf("Set(1.5): got %f, want 1", er.rate)
	}

	// Force a negative rate and test update clamps it to [0, 1]
	er.rate = -0.5
	er.lastUpdate = time.Now().Add(-time.Second) // simulate time passing
	er.update(1)
	if er.rate < 0 {
		t.Errorf("update on negative rate: got %f, want >=0", er.rate)
	}
	if er.rate > 1 {
		t.Errorf("update on negative rate: got %f, want <=1", er.rate)
	}

	// Force a rate > 1 and test update clamps it to [0, 1]
	er.rate = 1.5
	er.lastUpdate = time.Now().Add(-time.Second) // simulate time passing
	er.update(0)
	if er.rate > 1 {
		t.Errorf("update on >1 rate: got %f, want <=1", er.rate)
	}
	if er.rate < 0 {
		t.Errorf("update on >1 rate: got %f, want >=0", er.rate)
	}
}

// Test that errorRate converges to the extremes (1 or 0) after many updates.
func TestErrorRate_ConvergesToLimits(t *testing.T) {

	var er errorRate

	const tol = 0.01      // Acceptable tolerance for convergence
	const maxSteps = 1000 // Fewer steps are enough if we simulate time
	const tau = 30.0      // Must match the production value

	// Converge towards 1 (all failures)
	er.Set(0)
	for i := 0; i < maxSteps; i++ {
		// Simulate the passage of at least tau seconds between updates
		er.lastUpdate = time.Now().Add(-time.Duration(tau) * time.Second)
		er.Failure()
		if 1-er.rate < tol {
			break
		}
	}
	if 1-er.rate >= tol {
		t.Errorf("Failure did not bring rate close to 1: got %f", er.rate)
	}

	// Converge towards 0 (all successes)
	er.Set(1)
	for i := 0; i < maxSteps; i++ {
		er.lastUpdate = time.Now().Add(-time.Duration(tau) * time.Second)
		er.Success()
		if er.rate < tol {
			break
		}
	}
	if er.rate >= tol {
		t.Errorf("Success did not bring rate close to 0: got %f", er.rate)
	}
}

// Test that a long gap between updates makes EMA adapt quickly.
func TestErrorRate_AdaptiveAlpha(t *testing.T) {

	const dtFactor = 1.0 // Simulate a gap equal to tau
	const tol = 0.02     // Acceptable tolerance for floating point

	if tau <= 0 {
		t.Fatal("tau must be > 0 for this test to be valid")
	}

	var e errorRate

	// Calculate expected alpha for the test interval
	expectedAlpha := 1 - math.Exp(-dtFactor)
	// Expected rate after update
	expectedUp := expectedAlpha*1 + (1-expectedAlpha)*0
	expectedDown := expectedAlpha*0 + (1-expectedAlpha)*1

	// Case 1: Sudden jump from 0 to 1 after a long interval.
	e.Set(0)
	e.lastUpdate = time.Now().Add(-time.Duration(float64(tau)*dtFactor) * time.Second)
	e.update(1)
	t.Logf("rate after long gap with error=1: %f (expected ~%.3f)", e.rate, expectedUp)
	if math.Abs(e.rate-expectedUp) > tol {
		t.Errorf("Long dt with error=1: expected rate ≈ %.3f, got %f", expectedUp, e.rate)
	}

	// Case 2: Sudden drop from 1 to 0 after a long interval.
	e.Set(1)
	e.lastUpdate = time.Now().Add(-time.Duration(float64(tau)*dtFactor) * time.Second)
	e.update(0)
	t.Logf("rate after long gap with error=0: %f (expected ~%.3f)", e.rate, expectedDown)
	if math.Abs(e.rate-expectedDown) > tol {
		t.Errorf("Long dt with error=0: expected rate ≈ %.3f, got %f", expectedDown, e.rate)
	}
}
