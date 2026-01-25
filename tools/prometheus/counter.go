// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package prometheus

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// Counter is a Prometheus counter metric that stores its value internally using
// an atomic uint64.
//
// Example:
//
//	counter := RegisterCounter("requests_total", "Total number of requests")
//	counter.Inc()
//	counter.Add(3)
//
// All methods are safe for concurrent use by multiple goroutines.
type Counter struct {
	desc        *prometheus.Desc
	labelValues []string
	vec         *CounterVec

	value atomic.Uint64
}

// RegisterCounter registers a [Counter] metric with the given name and help
// description. Metric name must start with [a-zA-Z_] and contain only
// [a-zA-Z0-9_], no spaces. It panics if a metric with the same name is already
// registered.
func RegisterCounter(name, help string) *Counter {
	c := &Counter{
		desc: prometheus.NewDesc(name, help, nil, nil),
	}
	prometheus.MustRegister(c)
	return c
}

// Inc increments the counter by 1.
func (c *Counter) Inc() {
	c.Add(1)
}

// Add adds the given non-negative value to the counter.
// It panics if the value is negative.
func (c *Counter) Add(v int) {
	if v < 0 {
		panic("metrics: counter cannot decrease")
	}
	c.value.Add(uint64(v))
}

// Collect sends the current counter value to the provided Prometheus metrics
// channel. It is safe for concurrent use by multiple goroutines.
func (c *Counter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, float64(c.value.Load()), c.labelValues...)
}

// Describe sends the metric descriptor to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *Counter) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Unregister unregisters the metric.
func (c *Counter) Unregister() {
	if c.vec != nil {
		key := strings.Join(c.labelValues, "\xff")
		c.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(c)
}

// CounterVec is a collection of [Counter] metrics partitioned by label values.
//
// Example:
//
//	vec := RegisterCounterVec(
//	    "requests_total",
//	    "Total requests by method",
//	    []string{"method"})
//	counter := vec.Register("GET")
//	counter.Inc()
//
// The [CounterVec.Register] method, together with [CounterVec.Collect] and
// [CounterVec.Describe], is safe for concurrent use by multiple goroutines.
type CounterVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // map[string]*Counter
}

// RegisterCounterVec registers a [CounterVec] metric vector with the given
// name, help description, and label names. Metric and label names must start
// with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterCounterVec(name, help string, labels []string) *CounterVec {
	v := &CounterVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all registered [Counter] metrics of the vector to Prometheus.
// It is safe for concurrent use by multiple goroutines.
func (vec *CounterVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*Counter).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this counter vector and all its registered
// counters to the provided channel. It is safe for concurrent use by multiple
// goroutines.
func (vec *CounterVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*Counter).Describe(ch)
		return true
	})
}

// Register registers a [Counter] metric for the given label values and returns
// it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already exists.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *CounterVec) Register(labelValues ...string) *Counter {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	metric := &Counter{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		vec:         vec,
	}
	if _, loaded := vec.metrics.LoadOrStore(key, metric); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return metric
}

// Unregister unregisters the vector.
func (vec *CounterVec) Unregister() {
	prometheus.Unregister(vec)
}
