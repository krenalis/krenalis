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

// CounterFunc is a Prometheus counter metric that calls a function to retrieve
// its current value when collected.
//
// Example:
//
//	counter := NewCounterFunc(
//	    "my_counter",
//	    "Counter from function",
//	    func() float64 {
//	        return someValue()
//	    })
//
// The [CounterFunc.Collect] and [CounterFunc.Describe] methods are called by
// the Prometheus collector and are safe for concurrent use.
type CounterFunc struct {
	desc   *prometheus.Desc
	labels []string
	get    func() float64
}

// NewCounterFunc creates and registers a new [CounterFunc] metric with the
// given name, help description, and value retrieval function. Metric name must
// start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces. It panics if a
// metric with the same name is already registered.
func NewCounterFunc(name, help string, get func() float64) *CounterFunc {
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
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.CounterValue, c.get(), c.labels...)
}

// Describe sends the metric descriptor to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *CounterFunc) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

// CounterFuncVec is a collection of [CounterFunc] metrics partitioned by label
// values. Each metric's value is retrieved by calling its registered function.
//
// Example:
//
//	vec := NewCounterFuncVec(
//	    "requests_total",
//	    "Total requests by method",
//	    []string{"method"})
//	counter := vec.LoadOrStore([]string{"GET"}, func() float64 {
//	    return getRequestCount("GET")
//	})
//
// The [CounterFuncVec.LoadOrStore] method is safe for concurrent use by
// multiple goroutines. The [CounterFuncVec.Collect] and
// [CounterFuncVec.Describe] methods are called by the Prometheus collector.
type CounterFuncVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map
}

// NewCounterFuncVec creates and registers a new [CounterFuncVec] metric vector
// with the given name, help string, and label names. Metric and label names
// must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func NewCounterFuncVec(name, help string, labels []string) *CounterFuncVec {
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
func (v *CounterFuncVec) Collect(ch chan<- prometheus.Metric) {
	v.metrics.Range(func(_, metric any) bool {
		metric.(*CounterFunc).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this metric vector and all its stored
// [CounterFunc] metrics to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (v *CounterFuncVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
	v.metrics.Range(func(_, metric any) bool {
		metric.(*CounterFunc).Describe(ch)
		return true
	})
}

// LoadOrStore returns the [CounterFunc] metric for the given label values,
// creating and storing it if it does not already exist. The provided get
// function is used to retrieve the metric value on collection.
//
// Label values must match the vector's label names in number.
// This method is safe for concurrent use by multiple goroutines.
func (v *CounterFuncVec) LoadOrStore(labels []string, get func() float64) *CounterFunc {
	if len(labels) != len(v.labels) {
		panic(fmt.Sprintf("metrics: expected %d labels, got %d", len(v.labels), len(labels)))
	}
	key := strings.Join(labels, "%ff")
	if counter, ok := v.metrics.Load(key); ok {
		return counter.(*CounterFunc)
	}
	constLabels := make(map[string]string, len(labels))
	for i, name := range v.labels {
		constLabels[name] = labels[i]
	}
	c := &CounterFunc{
		desc:   prometheus.NewDesc(v.name, v.help, v.labels, constLabels),
		labels: labels,
		get:    get,
	}
	counter, _ := v.metrics.LoadOrStore(key, c)
	return counter.(*CounterFunc)
}
