// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package prometheus

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Histogram is a Prometheus histogram metric that maintains its state
// internally, protecting all updates with a single mutex so that the sample
// count, sum and bucket counters are modified atomically.
//
// Example:
//
//	h := RegisterHistogram(
//	    "request_duration_seconds",
//	    "Histogram of request durations",
//	    []float64{0.1, 0.2, 0.5, 1, 2})
//	h.Observe(0.27)
//
// All methods are safe for concurrent use by multiple goroutines.
type Histogram struct {
	desc        *prometheus.Desc
	labelValues []string
	les         []float64
	vec         *HistogramVec

	mu      sync.Mutex
	count   uint64
	sum     float64
	buckets []uint64 // one counter per configured bucket
}

// RegisterHistogram registers a [Histogram] metric with the given name, help
// string, and bucket upper bounds (the “le” values). Metric name must start
// with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterHistogram(name, help string, les []float64) *Histogram {
	h := &Histogram{
		desc:    prometheus.NewDesc(name, help, nil, nil),
		les:     les,
		buckets: make([]uint64, len(les)),
	}
	prometheus.MustRegister(h)
	return h
}

// Collect sends the current histogram state to the Prometheus metrics channel.
// Implements the prometheus.Collector interface.
func (h *Histogram) Collect(ch chan<- prometheus.Metric) {

	bucketCounts := make(map[float64]uint64, len(h.les))

	h.mu.Lock()
	count := h.count
	sum := h.sum
	var cumulative uint64
	for i, le := range h.les {
		cumulative += h.buckets[i]
		bucketCounts[le] = cumulative
	}
	h.mu.Unlock()

	ch <- prometheus.MustNewConstHistogram(h.desc, count, sum, bucketCounts, h.labelValues...)
}

// Describe sends the metric descriptor to the provided channel.
func (h *Histogram) Describe(ch chan<- *prometheus.Desc) {
	ch <- h.desc
}

// Observe records a single observation value into the histogram data.
// It increments the count and sum, and increments only the first bucket whose
// boundary is greater than or equal to the value.
func (h *Histogram) Observe(v float64) {
	h.mu.Lock()
	h.count++
	h.sum += v
	for i, le := range h.les {
		if v <= le {
			h.buckets[i]++
			break
		}
	}
	h.mu.Unlock()
}

// Unregister unregisters the metric.
func (h *Histogram) Unregister() {
	if h.vec != nil {
		key := strings.Join(h.labelValues, "\xff")
		h.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(h)
}

// HistogramVec is a collection of [Histogram] metrics partitioned by label
// values.
//
// Example:
//
//	vec := RegisterHistogramVec(
//	    "request_duration_seconds",
//	    "Request duration by method",
//	    []string{"method"},
//	    []float64{0.1, 0.2, 0.5, 1, 2})
//	hist := vec.Register("GET")
//	hist.Observe(0.33)
//
// The [HistogramVec.Register], [HistogramVec.Collect] and
// [HistogramVec.Describe] methods are safe for concurrent use by multiple
// goroutines.
type HistogramVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	les     []float64
	metrics sync.Map // map[string]*Histogram
}

// RegisterHistogramVec registers a [HistogramVec] metric vector with the given
// name, help string, label names and bucket boundaries. Metric and label names
// must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterHistogramVec(name, help string, labels []string, les []float64) *HistogramVec {
	v := &HistogramVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
		les:    les,
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all registered [Histogram] metrics in the vector to Prometheus.
// Implements prometheus.Collector.
func (vec *HistogramVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*Histogram).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this histogram vector and all its registered
// histograms to the provided channel.
func (vec *HistogramVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*Histogram).Describe(ch)
		return true
	})
}

// Register registers a [Histogram] metric for the given label values and
// returns it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already exists.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *HistogramVec) Register(labelValues ...string) *Histogram {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	metric := &Histogram{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		les:         vec.les,
		vec:         vec,
		buckets:     make([]uint64, len(vec.les)),
	}
	if _, loaded := vec.metrics.LoadOrStore(key, metric); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return metric
}

// Unregister unregisters the vector.
func (vec *HistogramVec) Unregister() {
	prometheus.Unregister(vec)
}
