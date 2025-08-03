//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package metrics

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// HistogramBuf is a histogram metric that buffers observations locally and
// consolidates them before exporting to Prometheus.
//
// Example:
//
//	h := RegisterHistogramBuf(
//	    "request_duration_seconds",
//	    "Histogram of request durations",
//	    []float64{0.1, 0.2, 0.5, 1, 2})
//	h.Observe(0.3)
//	h.Consolidate()
//
// The [HistogramBuf.Observe] and [HistogramBuf.Consolidate] methods should only
// be called from a single goroutine. The [HistogramBuf.Collect] and
// [HistogramBuf.Describe] methods are called by the Prometheus collector.
type HistogramBuf struct {
	desc        *prometheus.Desc
	labelValues []string
	les         []float64
	vec         *HistogramBufVec

	count   uint32
	sum     float64
	buckets []uint32

	consolidated struct {
		sync.Mutex
		count   uint64
		sum     float64
		buckets []uint64
	}
}

// RegisterHistogramBuf registers a new [HistogramBuf] with specified name, help
// string, and bucket upper bounds (less or equal thresholds). Metric name must
// start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterHistogramBuf(name, help string, les []float64) *HistogramBuf {
	h := &HistogramBuf{
		desc:    prometheus.NewDesc(name, help, nil, nil),
		les:     les,
		buckets: make([]uint32, len(les)),
	}
	h.consolidated.buckets = make([]uint64, len(les))
	prometheus.MustRegister(h)
	return h
}

// Collect sends the consolidated histogram metric to Prometheus, resetting
// the consolidated counters is avoided to maintain cumulative metric behavior.
// It is safe for concurrent use by multiple goroutines.
func (h *HistogramBuf) Collect(ch chan<- prometheus.Metric) {

	h.consolidated.Lock()
	count := h.consolidated.count
	sum := h.consolidated.sum
	buckets := make(map[float64]uint64, len(h.les))
	var cumulative uint64
	for i, v := range h.consolidated.buckets {
		cumulative += v
		buckets[h.les[i]] = cumulative
	}
	h.consolidated.Unlock()

	ch <- prometheus.MustNewConstHistogram(h.desc, count, sum, buckets, h.labelValues...)
}

// Consolidate consolidates buffered observations into the locked consolidated
// counters, resetting the buffers for next use.
// It must not be celled after the metric has been removed.
func (h *HistogramBuf) Consolidate() {
	h.consolidated.Lock()
	h.consolidated.count += uint64(h.count)
	h.count = 0
	h.consolidated.sum += h.sum
	h.sum = 0
	for i, v := range h.buckets {
		h.consolidated.buckets[i] += uint64(v)
		h.buckets[i] = 0
	}
	h.consolidated.Unlock()
}

// Describe sends the descriptor of this histogram to the provided channel.
// Implements the prometheus.Collector interface.
// It is safe for concurrent use by multiple goroutines.
func (h *HistogramBuf) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.desc
}

// Observe records a single observation value into the buffered histogram data.
// It increments the count and sum, and increments only the first bucket
// whose boundary is greater than or equal to the value.
// It must not be celled after the metric has been removed.
func (h *HistogramBuf) Observe(v float64) {
	h.count++
	h.sum += v
	for i, le := range h.les {
		if v <= le {
			h.buckets[i]++
			break
		}
	}
}

// Unregister unregisters the metric.
func (h *HistogramBuf) Unregister() {
	if h.vec != nil {
		key := strings.Join(h.labelValues, "\xff")
		h.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(h)
}

// HistogramBufVec is a collection of [HistogramBuf] metrics partitioned by
// label values.
//
// Example:
//
//	vec := RegisterHistogramBufVec(
//	    "request_duration_seconds",
//	    "Request duration by method and code",
//	    []string{"method", "code"},
//	    []float64{0.1, 0.5, 1, 2})
//	hist := vec.Register("GET", "200")
//	hist.Observe(0.3)
//	hist.Consolidate()
//
// The [HistogramBufVec.CreateMetric] method is safe for concurrent use by
// multiple goroutines. The [HistogramBufVec.Collect] and
// [HistogramBufVec.Describe] methods are called by the Prometheus
// collector.
type HistogramBufVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	les     []float64
	metrics sync.Map // sync.Map allows non-blocking writes during Collect.
}

// RegisterHistogramBufVec registers a [HistogramBufVec] with given name, help,
// label names, and bucket boundaries. Metric and label names must start with
// [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterHistogramBufVec(name, help string, labels []string, les []float64) *HistogramBufVec {
	vec := &HistogramBufVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
		les:    les,
	}
	prometheus.MustRegister(vec)
	return vec
}

// Collect sends all the collected metrics of the vector to Prometheus.
// Implements the prometheus.Collector interface.
// It is safe for concurrent use by multiple goroutines.
func (vec *HistogramBufVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*HistogramBuf).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this histogram vector and all its stored
// metrics to the provided channel. It implements prometheus.Collector.
// It is safe for concurrent use by multiple goroutines.
func (vec *HistogramBufVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*HistogramBuf).Describe(ch)
		return true
	})
}

// Register registers a [HistogramBuf] for the given label values and returns
// it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already registered.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *HistogramBufVec) Register(labelValues ...string) *HistogramBuf {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	h := &HistogramBuf{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		les:         vec.les,
		vec:         vec,
		buckets:     make([]uint32, len(vec.les)),
	}
	h.consolidated.buckets = make([]uint64, len(vec.les))
	if _, loaded := vec.metrics.LoadOrStore(key, h); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return h
}

// Unregister unregisters the vector.
func (vec *HistogramBufVec) Unregister() {
	prometheus.Unregister(vec)
}
