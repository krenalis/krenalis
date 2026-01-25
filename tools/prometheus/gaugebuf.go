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

// GaugeBuf is a custom gauge metric that buffers updates locally and
// consolidates them before exporting to Prometheus.
//
// Example:
//
//		gauge := RegisterGaugeBuf(
//	     "memory_usage_bytes",
//	     "Current memory usage in bytes")
//		gauge.Set(1024)
//		gauge.Consolidate()
//
// The [GaugeBuf.Set] and [GaugeBuf.Consolidate] methods should only be called
// from a single goroutine. The [GaugeBuf.Collect] and [GaugeBuf.Describe]
// methods are called by the Prometheus collector and are safe for concurrent
// use.
type GaugeBuf struct {
	desc        *prometheus.Desc
	labelValues []string
	vec         *GaugeBufVec

	value float64

	consolidated struct {
		sync.Mutex
		value float64
	}
}

// RegisterGaugeBuf registers a [GaugeBuf] metric with the given name and help
// description. Metric name must start with [a-zA-Z_] and contain only
// [a-zA-Z0-9_], no spaces. It panics if a metric with the same name is already
// registered.
func RegisterGaugeBuf(name, help string) *GaugeBuf {
	g := &GaugeBuf{
		desc: prometheus.NewDesc(name, help, nil, nil),
	}
	prometheus.MustRegister(g)
	return g
}

// Consolidate moves the locally buffered gauge value into the consolidated
// value, resetting the buffer to zero.
func (g *GaugeBuf) Consolidate() {
	g.consolidated.Lock()
	g.consolidated.value = g.value
	g.value = 0
	g.consolidated.Unlock()
}

// Collect sends the consolidated gauge metric to the Prometheus metrics
// channel. It is safe for concurrent use by multiple goroutines.
func (g *GaugeBuf) Collect(ch chan<- prometheus.Metric) {
	g.consolidated.Lock()
	value := g.consolidated.value
	g.consolidated.Unlock()
	ch <- prometheus.MustNewConstMetric(g.desc, prometheus.GaugeValue, value, g.labelValues...)
}

// Describe sends the descriptor of this gauge to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (g *GaugeBuf) Describe(ch chan<- *prometheus.Desc) {
	ch <- g.desc
}

// Set updates the buffered gauge to the given value.
func (g *GaugeBuf) Set(v float64) {
	g.value = v
}

// Unregister unregisters the metric.
func (g *GaugeBuf) Unregister() {
	if g.vec != nil {
		key := strings.Join(g.labelValues, "\xff")
		g.vec.metrics.Delete(key)
		return
	}
	prometheus.Unregister(g)
}

// GaugeBufVec is a collection of [GaugeBuf] metrics partitioned by label
// values.
//
// Example:
//
//	vec := RegisterGaugeBufVec(
//	    "memory_usage_bytes",
//	    "Memory usage by service",
//	    []string{"service"})
//	gauge := vec.Register("auth")
//	gauge.Set(2048)
//	gauge.Consolidate()
//
// The [GaugeBufVec.Register] method is safe for concurrent use by multiple
// goroutines. The [GaugeBufVec.Collect] and [GaugeBufVec.Describe] methods are
// called by the Prometheus collector.
type GaugeBufVec struct {
	name    string
	help    string
	labels  []string
	desc    *prometheus.Desc
	metrics sync.Map // map[string]*GaugeBuf
}

// RegisterGaugeBufVec registers a [GaugeBufVec] metric vector with the given
// name, help description, and label names. Metric and label names must start
// with [a-zA-Z_] and contain only [a-zA-Z0-9_], no spaces. It panics if a
// metric with the same name is already registered.
func RegisterGaugeBufVec(name, help string, labels ...string) *GaugeBufVec {
	v := &GaugeBufVec{
		name:   name,
		help:   help,
		labels: labels,
		desc:   prometheus.NewDesc(name, help, labels, nil),
	}
	prometheus.MustRegister(v)
	return v
}

// Collect sends all the collected [GaugeBuf] metrics of the vector to
// Prometheus. It is safe for concurrent use by multiple goroutines.
func (vec *GaugeBufVec) Collect(ch chan<- prometheus.Metric) {
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeBuf).Collect(ch)
		return true
	})
}

// Describe sends the descriptor of this gauge vector and all its registered
// gauges to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (vec *GaugeBufVec) Describe(ch chan<- *prometheus.Desc) {
	ch <- vec.desc
	vec.metrics.Range(func(_, metric any) bool {
		metric.(*GaugeBuf).Describe(ch)
		return true
	})
}

// Register registers a [GaugeBuf] metric for the given label values and returns
// it.
// Label values must match the label names in number.
// It is safe for concurrent use by multiple goroutines.
func (vec *GaugeBufVec) Register(labelValues ...string) *GaugeBuf {
	if len(labelValues) != len(vec.labels) {
		panic(fmt.Sprintf("metrics: expected %d label values, got %d", len(vec.labels), len(labelValues)))
	}
	key := strings.Join(labelValues, "\xff")
	g := &GaugeBuf{
		desc:        prometheus.NewDesc(vec.name, vec.help, vec.labels, nil),
		labelValues: labelValues,
		vec:         vec,
	}
	if _, loaded := vec.metrics.LoadOrStore(key, g); loaded {
		panic(fmt.Sprintf("metrics: label values %v already registered", labelValues))
	}
	return g
}

// Unregister unregisters the vector.
func (vec *GaugeBufVec) Unregister() {
	prometheus.Unregister(vec)
}
