//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package sender

import (
	"github.com/meergo/meergo/metrics"
)

// Queue available.
var queueAvailableMetric = metrics.RegisterGaugeFuncVec(
	"meergo_sender_queue_available",
	"Number of available events in the event queue",
	[]string{"connector", "connection"})

// Queue wait.
var queueWaitMetric = metrics.RegisterHistogramBufVec(
	"meergo_sender_queue_wait",
	"Time spent waiting in the event queue (in seconds)",
	[]string{"connector", "connection"},
	[]float64{
		0.005, // 5ms
		0.01,  // 10ms
		0.025, // 25ms
		0.05,  // 50ms
		0.075, // 75ms
		0.1,   // 100ms
		0.15,  // 150ms
		0.2,   // 200ms ← target
		0.3,
		0.5,
		0.75,
		1.0,
		2.0,
	})
