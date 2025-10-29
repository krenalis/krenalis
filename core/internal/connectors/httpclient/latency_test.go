// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"math"
	"testing"
	"time"
)

// Test basic behavior of Observe: first observation sets the latency directly
// and negative values after initialization are clamped to zero. A future
// lastUpdate time should be treated as no time elapsed.
func TestLatencyAvg_BoundsAndClamp(t *testing.T) {
	var l latencyAvg

	// First observe sets the latency exactly.
	first := 100 * time.Millisecond
	l.Observe(first)
	if l.latency != first {
		t.Fatalf("first observe: got %v want %v", l.latency, first)
	}

	// Force negative previous latency and observe a negative value.
	l.latency = -50 * time.Millisecond
	l.lastUpdate = time.Now().Add(-time.Second)
	l.Observe(-25 * time.Millisecond)
	if l.latency < 0 {
		t.Errorf("observe with negative values produced %v", l.latency)
	}

	// lastUpdate in the future should yield dt=0 and keep the latency.
	before := l.latency
	l.lastUpdate = time.Now().Add(time.Second)
	l.Observe(200 * time.Millisecond)
	if l.latency != before {
		t.Errorf("future lastUpdate changed latency: got %v want %v", l.latency, before)
	}
}

// Test that repeated observations converge toward the observed latency when
// enough time elapses between updates.
func TestLatencyAvg_Converges(t *testing.T) {
	var l latencyAvg
	const target = 150 * time.Millisecond
	const tol = 5 * time.Millisecond
	const maxSteps = 1000

	for i := 0; i < maxSteps; i++ {
		l.lastUpdate = time.Now().Add(-time.Duration(tauLatency) * time.Second)
		l.Observe(target)
		if absDuration(l.latency-target) < tol {
			break
		}
	}
	if absDuration(l.latency-target) >= tol {
		t.Errorf("latency did not converge: got %v want %v", l.latency, target)
	}
}

// Test that a long gap between observations makes the EMA adapt quickly.
func TestLatencyAvg_AdaptiveAlpha(t *testing.T) {
	var l latencyAvg
	const dtFactor = 1.0
	const tol = 2 * time.Millisecond

	expectedAlpha := 1 - math.Exp(-dtFactor)
	d := 200 * time.Millisecond

	// Upward jump after long interval.
	l.Observe(0)
	l.lastUpdate = time.Now().Add(-time.Duration(float64(tauLatency)*dtFactor) * time.Second)
	l.Observe(d)
	expectedUp := time.Duration(expectedAlpha * float64(d))
	if diff := absDuration(l.latency - expectedUp); diff > tol {
		t.Errorf("long gap up: got %v want ~%v", l.latency, expectedUp)
	}

	// Downward jump after long interval.
	l.latency = d
	l.lastUpdate = time.Now().Add(-time.Duration(float64(tauLatency)*dtFactor) * time.Second)
	l.Observe(0)
	expectedDown := time.Duration((1 - expectedAlpha) * float64(d))
	if diff := absDuration(l.latency - expectedDown); diff > tol {
		t.Errorf("long gap down: got %v want ~%v", l.latency, expectedDown)
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
