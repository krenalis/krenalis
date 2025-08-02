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

// BufferedHistogram is a custom histogram metric that buffers observations
// locally and consolidates them before exporting to Prometheus.
//
// Example:
//
//	h := NewBufferedHistogram(
//	    "request_duration_seconds",
//	    "Histogram of request durations",
//	    []float64{0.1, 0.2, 0.5, 1, 2})
//	h.Observe(0.3)
//	h.Consolidate()
//
// The [BufferedHistogram.Observe] and [BufferedHistogram.Consolidate] methods
// should only be called from a single goroutine. The
// [BufferedHistogram.Collect] and [BufferedHistogram.Describe] methods are
// called by the Prometheus collector.
type BufferedHistogram struct {
	desc   *prometheus.Desc
	labels []string
	les    []float64

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

// NewBufferedHistogram creates a new [BufferedHistogram] with specified name,
// help string, and bucket upper bounds (less or equal thresholds). Metric name
// must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func NewBufferedHistogram(name, help string, les []float64) *BufferedHistogram {
	h := &BufferedHistogram{
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
func (h *BufferedHistogram) Collect(ch chan<- prometheus.Metric) {

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

	ch <- prometheus.MustNewConstHistogram(h.desc, count, sum, buckets, h.labels...)
}

// Consolidate consolidates buffered observations into the locked consolidated
// counters, resetting the buffers for next use.
func (h *BufferedHistogram) Consolidate() {
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
func (h *BufferedHistogram) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.desc
}

// Observe records a single observation value into the buffered histogram data.
// It increments the count and sum, and increments only the first bucket
// whose boundary is greater than or equal to the value.
func (h *BufferedHistogram) Observe(v float64) {
	h.count++
	h.sum += v
	for i, le := range h.les {
		if v <= le {
			h.buckets[i]++
			break
		}
	}
}

// BufferedHistogramVec is a collection of [BufferedHistogram] metrics
// partitioned by label values.
//
// Example:
//
//	vec := NewBufferedHistogramVec(
//	    "request_duration_seconds",
//	    "Request duration by method and code",
//	    []string{"method", "code"},
//	    []float64{0.1, 0.5, 1, 2})
//	hist := vec.LoadOrStore("GET", "200")
//	hist.Observe(0.3)
//	hist.Consolidate()
//
// The [BufferedHistogramVec.LoadOrStore] method is safe for concurrent use by
// multiple goroutines. The [BufferedHistogramVec.Collect] and
// [BufferedHistogramVec.Describe] methods are called by the Prometheus
// collector.
type BufferedHistogramVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	les     []float64
	metrics sync.Map // sync.Map allows non-blocking writes during Collect.
}

// NewBufferedHistogramVec creates and registers a new [BufferedHistogramVec]
// with given name, help, label names, and bucket boundaries. Metric and label
// names must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func NewBufferedHistogramVec(name, help string, labels []string, les []float64) *BufferedHistogramVec {
	v := &BufferedHistogramVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
		les:    les,
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all the collected metrics of the vector to Prometheus.
// Implements the prometheus.Collector interface.
// It is safe for concurrent use by multiple goroutines.
func (v *BufferedHistogramVec) Collect(ch chan<- prometheus.Metric) {
	v.metrics.Range(func(_, metric any) bool {
		metric.(*BufferedHistogram).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this histogram vector and all its stored
// metrics to the provided channel. It implements prometheus.Collector.
// It is safe for concurrent use by multiple goroutines.
func (v *BufferedHistogramVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
	v.metrics.Range(func(_, metric any) bool {
		metric.(*BufferedHistogram).Describe(ch)
		return true
	})
}

// LoadOrStore returns the [BufferedHistogram] for the given label values,
// creating and storing it if it does not already exist.
// Label values can be any valid UTF-8 encoded Unicode string.
// It is safe for concurrent use by multiple goroutines.
func (v *BufferedHistogramVec) LoadOrStore(labels []string) *BufferedHistogram {
	if len(labels) != len(v.labels) {
		panic(fmt.Sprintf("metrics: expected %d labels, got %d", len(v.labels), len(labels)))
	}
	key := strings.Join(labels, "\xff")
	if hist, ok := v.metrics.Load(key); ok {
		return hist.(*BufferedHistogram)
	}
	h := &BufferedHistogram{
		desc:    prometheus.NewDesc(v.name, v.help, v.labels, nil),
		labels:  labels,
		les:     v.les,
		buckets: make([]uint32, len(v.les)),
	}
	h.consolidated.buckets = make([]uint64, len(v.les))
	hist, _ := v.metrics.LoadOrStore(key, h)
	return hist.(*BufferedHistogram)
}
