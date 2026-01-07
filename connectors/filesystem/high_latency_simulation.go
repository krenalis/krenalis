// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

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
