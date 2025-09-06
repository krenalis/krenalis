//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package httpclient

import (
	"math"
	"time"
)

const tauLatency = 30.0 // EMA time constant (in seconds)

// latencyAvg computes an Exponential Moving Average (EMA) of observed latency
// values. The EMA is updated each time Observe is called, using a time constant
// tauLatency.
type latencyAvg struct {
	latency    time.Duration // current EMA value
	lastUpdate time.Time     // last update timestamp
}

// Observe updates the EMA with a new observed latency d.
// The EMA is recalculated based on the time elapsed since the last update.
func (l *latencyAvg) Observe(d time.Duration) {
	now := time.Now()
	if l.lastUpdate.IsZero() {
		l.latency = d
		l.lastUpdate = now
		return
	}
	dt := max(0, now.Sub(l.lastUpdate).Seconds())
	alpha := 1 - math.Exp(-dt/tauLatency)
	v := float64(d.Nanoseconds())
	prev := float64(l.latency.Nanoseconds())
	latency := alpha*v + (1-alpha)*prev
	l.latency = time.Duration(max(0, latency))
	l.lastUpdate = now
}
