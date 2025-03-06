//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package filesystem

import (
	"io"
	"math/rand/v2"
	"time"
)

// highLatencyReadCloser is an io.ReadCloser that simulates random high I/O
// latency. This can be useful for testing.
type highLatencyReadCloser struct {
	f io.ReadCloser
}

func (h *highLatencyReadCloser) Read(p []byte) (n int, err error) {
	simulateHighIOLatency()
	return h.f.Read(p)
}

func (h *highLatencyReadCloser) Close() (err error) {
	simulateHighIOLatency()
	return h.f.Close()
}

// simulateHighIOLatency returns after a random amount of time, to simulate an
// high I/O random latency.
func simulateHighIOLatency() {
	latency := rand.Float64() + 0.3 // seconds.
	time.Sleep(time.Duration(latency * 1e9))
}
