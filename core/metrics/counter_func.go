// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"fmt"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// CounterFunc is a Prometheus counter metric that calls a function to retrieve
// its current value when collected.
//
// Example:
//
//	counter := RegisterCounterFunc(
//	    "my_counter",
//	    "Counter from function",
//	    func() float64 {
//	        return someValue()
//	    })
//
// The [CounterFunc.Collect] and [CounterFunc.Describe] methods are called by
// the Prometheus collector and are safe for concurrent use.
type CounterFunc struct {
	desc        *prometheus.Desc
	labelValues []string
	vec         *CounterFuncVec
	get         func() float64
}

// RegisterCounterFunc registers a [CounterFunc] metric with the given name,
// help description, and value retrieval function. Metric name must start with
// [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces. It panics if a metric
// with the same name is already registered.
func RegisterCounterFunc(name, help string, get func() float64) *CounterFunc {
	c := &CounterFunc{
		desc: prometheus.NewDesc(name, help, nil, nil),
		get:  get,
	}
	prometheus.MustRegister(c)
	return c
}

// Collect sends the current metric value, obtained by calling the get function,
// to the provided Prometheus metrics channel.
// It is safe for concurrent use by multiple goroutines.
func (c *CounterFunc) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, c.get(), c.labelValues...)
}

// Describe sends the metric descriptor to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *CounterFunc) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// Unregister unregisters the metric.
func (c *CounterFunc) Unregister() {
	if c.vec != nil {
		key := strings.Join(c.labelValues, "\xff")
		c.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(c)
}

// CounterFuncVec is a collection of [CounterFunc] metrics partitioned by label
// values. Each metric's value is retrieved by calling its registered function.
//
// Example:
//
//	vec := RegisterCounterFuncVec(
//	    "requests_total",
//	    "Total requests by method",
//	    []string{"method"})
//	counter := vec.Register(func() float64 {
//	    return getRequestCount("GET")
//	}, "GET")
//
// The [CounterFuncVec.Register] method is safe for concurrent use by
// multiple goroutines. The [CounterFuncVec.Collect] and
// [CounterFuncVec.Describe] methods are called by the Prometheus collector.
type CounterFuncVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map
}

// RegisterCounterFuncVec registers a [CounterFuncVec] metric vector with the
// given name, help string, and label names. Metric and label names must start
// with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterCounterFuncVec(name, help string, labels []string) *CounterFuncVec {
	v := &CounterFuncVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all [CounterFunc] metrics in the vector to Prometheus by
// calling their [CounterFunc.Collect] methods.
// It is safe for concurrent use by multiple goroutines.
func (vec *CounterFuncVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*CounterFunc).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this metric vector and all its registered
// [CounterFunc] metrics to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (vec *CounterFuncVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*CounterFunc).Describe(ch)
		return true
	})
}

// Register registers a [CounterFunc] metric for the given label values and
// returns it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already registered.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *CounterFuncVec) Register(get func() float64, labelValues ...string) *CounterFunc {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	if counter, ok := vec.metrics.Load(key); ok {
		return counter.(*CounterFunc)
	}
	c := &CounterFunc{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		vec:         vec,
		get:         get,
	}
	if _, loaded := vec.metrics.LoadOrStore(key, c); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return c
}

// Unregister unregisters the vector.
func (vec *CounterFuncVec) Unregister() {
	prometheus.Unregister(vec)
}
