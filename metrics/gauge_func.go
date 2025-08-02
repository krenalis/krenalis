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

// GaugeFunc is a Prometheus gauge metric that calls a function to retrieve its
// current value when collected.
//
// Example:
//
//	g := NewGaugeFunc(
//	    "current_temperature",
//	    "Current temperature",
//	    func() float64 {
//	        return readTemperatureSensor()
//	    })
//
// The [GaugeFunc.Collect] and [GaugeFunc.Describe] methods are called by the
// Prometheus collector and are safe for concurrent use.
type GaugeFunc struct {
	desc   *prometheus.Desc
	labels []string
	get    func() float64
}

// NewGaugeFunc creates and registers a new [GaugeFunc] metric with the given
// name, help string, and value retrieval function. Metric and label names must
// start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func NewGaugeFunc(name, help string, get func() float64) *GaugeFunc {
	g := &GaugeFunc{
		desc: prometheus.NewDesc(name, help, nil, nil),
		get:  get,
	}
	prometheus.MustRegister(g)
	return g
}

// Collect sends the current gauge value, obtained by calling the get function,
// to the provided Prometheus metrics channel.
// It is safe for concurrent use by multiple goroutines.
func (g *GaugeFunc) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, g.get(), g.labels...)
}

// Describe sends the metric descriptor to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (g *GaugeFunc) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.desc
}

// GaugeFuncVec is a collection of [GaugeFunc] metrics partitioned by label
// values. Each metric's value is retrieved by calling its registered function.
//
// Example:
//
//		vec := NewGaugeFuncVec(
//	     "temperature_celsius",
//	     "Temperature by location",
//	    []string{"location"})
//		gauge := vec.LoadOrStore([]string{"server_room"}, func() float64 {
//		    return readTemp("server_room")
//		})
//
// The [GaugeFuncVec.LoadOrStore] method is safe for concurrent use by multiple
// goroutines. The [GaugeFuncVec.Collect] and [GaugeFuncVec.Describe] methods
// are called by the Prometheus collector.
type GaugeFuncVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // sync.Map allows non-blocking writes during Collect.
}

// NewGaugeFuncVec creates and registers a new [GaugeFuncVec] metric vector
// with the given name, help string, and label names. Metric and label names
// must start with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func NewGaugeFuncVec(name, help string, labels []string) *GaugeFuncVec {
	v := &GaugeFuncVec{
		name:   name,
		help:   help,
		desc:   prometheus.NewDesc(name, help, labels, nil),
		labels: labels,
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all [GaugeFunc] metrics in the vector to Prometheus by calling
// their [GaugeFunc.Collect] methods.
// It is safe for concurrent use by multiple goroutines.
func (v *GaugeFuncVec) Collect(ch chan<- prometheus.Metric) {
	v.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeFunc).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this metric vector and all its stored
// [GaugeFunc] metrics to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (v *GaugeFuncVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- v.desc
	v.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeFunc).Describe(ch)
		return true
	})
}

// LoadOrStore returns the [GaugeFunc] metric for the given label values,
// creating and storing it if it does not already exist. The provided get
// function is used to retrieve the metric value on collection.
//
// Label values must match the vector's label names in number.
// This method is safe for concurrent use by multiple goroutines.
func (v *GaugeFuncVec) LoadOrStore(labels []string, get func() float64) *GaugeFunc {
	if len(labels) != len(v.labels) {
		panic(fmt.Sprintf("metrics: expected %d labels, got %d", len(v.labels), len(labels)))
	}
	key := strings.Join(labels, "\xff")
	if gauge, ok := v.metrics.Load(key); ok {
		return gauge.(*GaugeFunc)
	}
	constLabels := make(map[string]string, len(labels))
	for i, name := range v.labels {
		constLabels[name] = labels[i]
	}
	g := &GaugeFunc{
		desc:   prometheus.NewDesc(v.name, v.help, labels, constLabels),
		labels: labels,
		get:    get,
	}
	gauge, _ := v.metrics.LoadOrStore(key, g)
	return gauge.(*GaugeFunc)
}
