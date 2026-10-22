// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package prometheus provides custom Prometheus metric types with support for
// buffered updates and function-based value retrieval.
//
// It includes counters, gauges, and histograms—each available in standalone and
// vector forms.
//
// Available types:
//
//   - [Counter] and [CounterVec] implement a counter whose value is stored atomically.
//   - [CounterFunc] and [CounterFuncVec] implement a counter that retrieves its value by calling a function at collection time. Useful when the counter is managed externally.
//   - [CounterBuf] and [CounterBufVec] implement a counter that buffers increments locally before consolidating. Useful for high-frequency updates with reduced lock contention.
//   - [GaugeFunc] and [GaugeFuncVec] implement a gauge that retrieves its value via a function. Suitable for externally tracked values that can fluctuate up or down.
//   - [GaugeBuf] and [GaugeBufVec] implement a gauge that buffers increments locally before consolidating. Useful for high-frequency updates with reduced lock contention.
//   - [Histogram] and [HistogramVec] implement a histogram whose value is protected by a mutex.
//   - [HistogramBuf] and [HistogramBufVec] implement a histogram that buffers observations locally before consolidating, reducing contention during frequent updates.
package prometheus
