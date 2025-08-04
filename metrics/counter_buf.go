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

// CounterBuf is a counter metric that buffers increments locally and
// consolidates them before exporting to Prometheus.
//
// Example:
//
//	counter := RegisterCounterBuf("requests_total", "Total number of requests")
//	counter.Inc()
//	counter.Consolidate()
//
// The [CounterBuf.Inc] and [CounterBuf.Consolidate] methods should
// only be called from a single goroutine. The [CounterBuf.Collect] and
// [CounterBuf.Describe] methods are called by the Prometheus collector and
// are safe for concurrent use.
type CounterBuf struct {
	desc        *prometheus.Desc
	labelValues []string
	vec         *CounterBufVec

	count uint64

	consolidated struct {
		sync.Mutex
		count uint64
	}
}

// RegisterCounterBuf registers a [CounterBuf] metric with the given name and
// help description. Metric name must start with [a-zA-Z_] and contain only
// [a-zA-Z0-9_], no spaces. It panics if a metric with the same name is already
// registered.
func RegisterCounterBuf(name, help string) *CounterBuf {
	c := &CounterBuf{
		desc: prometheus.NewDesc(name, help, nil, nil),
	}
	prometheus.MustRegister(c)
	return c
}

// Inc increments the buffered counter by 1.
func (c *CounterBuf) Inc() {
	c.count++
}

// Consolidate moves the locally buffered count into the consolidated count,
// resetting the buffer to zero.
func (c *CounterBuf) Consolidate() {
	c.consolidated.Lock()
	c.consolidated.count += c.count
	c.count = 0
	c.consolidated.Unlock()
}

// Collect sends the consolidated counter metric to the Prometheus metrics
// channel. It is safe for concurrent use by multiple goroutines.
func (c *CounterBuf) Collect(ch chan<- prometheus.Metric) {
	c.consolidated.Lock()
	count := c.consolidated.count
	c.consolidated.Unlock()
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, float64(count), c.labelValues...)
}

// Describe sends the descriptor of this counter to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *CounterBuf) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Unregister unregisters the metric.
func (c *CounterBuf) Unregister() {
	if c.vec != nil {
		key := strings.Join(c.labelValues, "\xff")
		c.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(c)
}

// CounterBufVec is a collection of CounterBuf metrics partitioned by label
// values.
//
// Example:
//
//	vec := RegisterCounterBufVec(
//	    "requests_total",
//	    "Total requests by method",
//	    []string{"method"})
//	counter := vec.Register("GET")
//	counter.Inc()
//	counter.Consolidate()
//
// The [CounterBufVec.Register] method is safe for concurrent use by multiple
// goroutines. The [CounterBufVec.Collect] and [CounterBufVec.Describe] methods
// are called by the Prometheus collector.
type CounterBufVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // map[string]*CounterBuf
}

// RegisterCounterBufVec registers a [CounterBufVec] metric vector with the
// given name, help description, and label names. Metric and label names must
// start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces. It panics if a
// metric with the same name is already registered.
func RegisterCounterBufVec(name, help string, labels []string) *CounterBufVec {
	vec := &CounterBufVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
	}
	prometheus.MustRegister(vec)
	return vec
}

// Collect sends all the collected CounterBuf metrics of the vector to
// Prometheus. It is safe for concurrent use by multiple goroutines.
func (vec *CounterBufVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*CounterBuf).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this counter vector and all its registered
// counters to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (vec *CounterBufVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*CounterBuf).Describe(ch)
		return true
	})
}

// Register registers a [CounterBuf] metric for the given label values and
// returns it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already registered.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *CounterBufVec) Register(labelValues ...string) *CounterBuf {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	c := &CounterBuf{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		vec:         vec,
	}
	if _, loaded := vec.metrics.LoadOrStore(key, c); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", key))
	}
	return c
}

// Unregister unregisters the vector.
func (vec *CounterBufVec) Unregister() {
	prometheus.Unregister(vec)
}
