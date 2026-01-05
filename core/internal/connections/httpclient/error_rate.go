// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"math"
	"time"
)

const tau = 30.0 // Seconds, tuning parameter for EMA memory window

// errorRate implements an exponential moving average (EMA) for tracking failure
// rate, adapted to account for irregular update intervals.
type errorRate struct {
	rate       float64
	lastUpdate time.Time
}

// Success updates the EMA after a successful request.
func (e *errorRate) Success() { e.update(0) }

// Failure updates the EMA after a failed request.
func (e *errorRate) Failure() { e.update(1) }

// Set sets the rate with the provided value.
func (e *errorRate) Set(rate float64) {
	e.rate = max(0, min(rate, 1))
	e.lastUpdate = time.Now()
}

// update recalculates the EMA with the given value (0 for success, 1 for
// failure). The smoothing factor alpha is adjusted based on the time since the
// last update.
func (e *errorRate) update(v float64) {
	now := time.Now()
	dt := max(0, now.Sub(e.lastUpdate).Seconds())
	alpha := 1 - math.Exp(-dt/tau)
	rate := alpha*v + (1-alpha)*e.rate
	e.rate = max(0, min(rate, 1))
	e.lastUpdate = now
}
