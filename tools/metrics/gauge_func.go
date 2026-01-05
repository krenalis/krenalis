// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
//	g := RegisterGaugeFunc(
//	    "current_temperature",
//	    "Current temperature",
//	    func() float64 {
//	        return readTemperatureSensor()
//	    })
//
// The [GaugeFunc.Collect] and [GaugeFunc.Describe] methods are called by the
// Prometheus collector and are safe for concurrent use.
type GaugeFunc struct {
	desc        *prometheus.Desc
	labelValues []string
	vec         *GaugeFuncVec
	get         func() float64
}

// RegisterGaugeFunc registers a [GaugeFunc] metric with the given name, help
// string, and value retrieval function. Metric and label names must start with
// [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterGaugeFunc(name, help string, get func() float64) *GaugeFunc {
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
	ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, g.get(), g.labelValues...)
}

// Describe sends the metric descriptor to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (g *GaugeFunc) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.desc
}

// Unregister unregisters the metric.
func (g *GaugeFunc) Unregister() {
	if g.vec != nil {
		key := strings.Join(g.labelValues, "\xff")
		g.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(g)
}

// GaugeFuncVec is a collection of [GaugeFunc] metrics partitioned by label
// values. Each metric's value is retrieved by calling its registered function.
//
// Example:
//
//	vec := RegisterGaugeFuncVec(
//	    "temperature_celsius",
//	    "Temperature by location",
//	    []string{"location"})
//	gauge := vec.Register(func() float64 {
//	    return readTemp("server_room")
//	}, "server_room")
//
// The [GaugeFuncVec.Register] method is safe for concurrent use by multiple
// goroutines. The [GaugeFuncVec.Collect] and [GaugeFuncVec.Describe] methods
// are called by the Prometheus collector.
type GaugeFuncVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // sync.Map allows non-blocking writes during Collect.
}

// RegisterGaugeFuncVec registers a [GaugeFuncVec] metric vector with the given
// name, help string, and label names. Metric and label names must start with
// [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces.
// It panics if a metric with the same name is already registered.
func RegisterGaugeFuncVec(name, help string, labels []string) *GaugeFuncVec {
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
func (vec *GaugeFuncVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeFunc).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this metric vector and all its registered
// [GaugeFunc] metrics to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (vec *GaugeFuncVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeFunc).Describe(ch)
		return true
	})
}

// Register registers a [GaugeFunc] metric for the given label values and
// returns it.
//
// It panics if the number of label values does not match the vector's label
// names, or if a metric with the same label values already registered.
//
// This method is safe for concurrent use by multiple goroutines.
func (vec *GaugeFuncVec) Register(get func() float64, labelValues ...string) *GaugeFunc {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	g := &GaugeFunc{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		vec:         vec,
		get:         get,
	}
	if _, loaded := vec.metrics.LoadOrStore(key, g); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return g
}

// Unregister unregisters the vector.
func (vec *GaugeFuncVec) Unregister() {
	prometheus.Unregister(vec)
}
