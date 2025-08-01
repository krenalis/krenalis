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

// BufferedCounter is a custom counter metric that buffers increments locally
// and consolidates them before exporting to Prometheus.
//
// Example:
//
//	counter := NewBufferedCounter("requests_total", "Total number of requests")
//	counter.Inc()
//	counter.Consolidate()
//
// The [BufferedCounter.Inc] and [BufferedCounter.Consolidate] methods should
// only be called from a single goroutine. The [BufferedCounter.Collect] and
// [BufferedCounter.Describe] methods are called by the Prometheus collector and
// are safe for concurrent use.
type BufferedCounter struct {
	desc   *prometheus.Desc
	labels []string

	count uint64

	consolidated struct {
		sync.Mutex
		count uint64
	}
}

// NewBufferedCounter creates and registers a new [BufferedCounter] metric with
// the given name and help description. Metric name must start with [a-zA-Z_]
// and contain only [a-zA-Z0-9_], no spaces. It panics if a metric with the same
// name is already registered.
func NewBufferedCounter(name, help string) *BufferedCounter {
	c := &BufferedCounter{
		desc: prometheus.NewDesc(name, help, nil, nil),
	}
	prometheus.MustRegister(c)
	return c
}

// Inc increments the buffered counter by 1.
func (c *BufferedCounter) Inc() {
	c.count++
}

// Consolidate moves the locally buffered count into the consolidated count,
// resetting the buffer to zero.
func (c *BufferedCounter) Consolidate() {
	c.consolidated.Lock()
	c.consolidated.count += c.count
	c.count = 0
	c.consolidated.Unlock()
}

// Collect sends the consolidated counter metric to the Prometheus metrics
// channel. It is safe for concurrent use by multiple goroutines.
func (c *BufferedCounter) Collect(ch chan<- prometheus.Metric) {
	c.consolidated.Lock()
	count := c.consolidated.count
	c.consolidated.Unlock()
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, float64(count), c.labels...)
}

// Describe sends the descriptor of this counter to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *BufferedCounter) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// BufferedCounterVec is a collection of BufferedCounter metrics partitioned by
// label values.
//
// Example:
//
//	vec := NewBufferedCounterVec(
//	    "requests_total",
//	    "Total requests by method",
//	    []string{"method"})
//	counter := vec.LoadOrStore([]string{"GET"})
//	counter.Inc()
//	counter.Consolidate()
//
// The [BufferedCounterVec.LoadOrStore] method is safe for concurrent use by
// multiple goroutines. The [BufferedCounterVec.Collect] and
// [BufferedCounterVec.Describe] methods are called by the Prometheus collector.
type BufferedCounterVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // map[string]*BufferedCounter
}

// NewBufferedCounterVec creates and registers a new [BufferedCounterVec] metric
// vector with the given name, help description, and label names. Metric and
// label names must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no
// spaces. It panics if a metric with the same name is already registered.
func NewBufferedCounterVec(name, help string, labels []string) *BufferedCounterVec {
	v := &BufferedCounterVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all the collected BufferedCounter metrics of the vector to
// Prometheus. It is safe for concurrent use by multiple goroutines.
func (v *BufferedCounterVec) Collect(ch chan<- prometheus.Metric) {
	v.metrics.Range(func(_, metric any) bool {
		metric.(*BufferedCounter).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this counter vector and all its stored
// counters to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (v *BufferedCounterVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
	v.metrics.Range(func(_, metric any) bool {
		metric.(*BufferedCounter).Describe(ch)
		return true
	})
}

// LoadOrStore returns the [BufferedCounter] for the given label values,
// creating and storing it if it does not already exist. Label values must match
// the label names in number.
// It is safe for concurrent use by multiple goroutines.
func (v *BufferedCounterVec) LoadOrStore(labelValues []string) *BufferedCounter {
	if len(labelValues) != len(v.labels) {
		panic(fmt.Sprintf("metrics: expected %d labels, got %d", len(v.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "%ff")
	if c, ok := v.metrics.Load(key); ok {
		return c.(*BufferedCounter)
	}

	constLabels := make(map[string]string, len(labelValues))
	for i, name := range v.labels {
		constLabels[name] = labelValues[i]
	}

	c := &BufferedCounter{
		desc:   prometheus.NewDesc(v.name, v.help, v.labels, constLabels),
		labels: labelValues,
	}
	counter, _ := v.metrics.LoadOrStore(key, c)
	return counter.(*BufferedCounter)
}
