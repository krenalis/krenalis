//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package metrics provides custom Prometheus metric types with support for
// buffered updates and function-based value retrieval.
//
// It includes counters, gauges, and histograms—both standalone and vector forms.
//
// Available types:
//
//   - [CounterBufVec] is a vector of [CounterBuf], partitioned by label values.
//   - [CounterBuf] is a counter that buffers increments locally before consolidating. Useful for high-frequency updates with reduced lock contention.
//   - [CounterFuncVec] is a vector of CounterFunc, indexed by label values.
//   - [CounterFunc] is a counter that retrieves its value by calling a function at collection time. Use when the counter is managed externally.
//   - [GaugeFuncVec] is a vector of [GaugeFunc], partitioned by labels.
//   - [GaugeFunc] is a gauge that retrieves its value via a function. Suitable for externally tracked values that can go up or down.
//   - [HistogramBufVec] is a vector of [HistogramBuf], grouped by label values.
//   - [HistogramBuf] is a histogram that buffers observations locally and consolidates them later. Reduces contention during frequent updates.
package metrics

import (
	"expvar"
	"strings"
	"time"
)

// Enabled enables the exposure of the Meergo metrics via the HTTP endpoint
// /debug/vars.
const Enabled = false

// Increment increments the metric at path of the given delta.
//
// If the metric with the given path does not exist, this method creates it.
//
// path is the path of the metric, where the various components of the path are
// separated by the '.' character. Although it can contain any character, it is
// preferable that each component contains only alphanumeric characters and
// underscores, since these components will be exposed as keys of a JSON object
// and may processed by various JSON tools (such as jq).
//
// Some examples of paths:
//
//	warehouses.PostgreSQL.Merge.calls
//	dispatched.processed_events
func Increment(path string, delta int) {
	if !Enabled {
		return
	}
	getOrCreate(path, intVar).(*expvar.Int).Add(int64(delta))
}

// Set sets the value of the metric at the given path.
//
// If the metric with the given path does not exist, this method creates it.
//
// Also sets the value of a dummy metric "<path>_last_updated" that contains the
// timestamp of the last update of the metric at path. This is useful because,
// unlike for example an incremental metric, the metrics set with Set are valid
// in reference to a specific timestamp.
//
// path is the path of the metric, where the various components of the path are
// separated by the '.' character. Although it can contain any character, it is
// preferable that each component contains only alphanumeric characters and
// underscores, since these components will be exposed as keys of a JSON object
// and may processed by various JSON tools (such as jq).
//
// Some examples of paths:
//
//	warehouses.PostgreSQL.Merge.calls
//	dispatched.processed_events
func Set[T int | string](path string, value T) {
	if !Enabled {
		return
	}
	switch any(value).(type) {
	case int:
		getOrCreate(path, intVar).(*expvar.Int).Set(int64(any(value).(int)))
	case string:
		getOrCreate(path, stringVar).(*expvar.String).Set(any(value).(string))
	default:
		panic("unexpected")
	}
	timestamp := time.Now().UTC().Format(time.DateTime) + " (UTC)"
	getOrCreate(path+"_last_updated", stringVar).(*expvar.String).Set(timestamp)
}

type expvarType int

const (
	intVar expvarType = iota + 1
	stringVar
)

func init() {
	if !Enabled {
		return
	}
	expvar.NewMap("metrics")
}

func getOrCreate(path string, typ expvarType) expvar.Var {
	parts := strings.Split(path, ".")
	obj := expvar.Get("metrics")
	for i, name := range parts {
		last := i == len(parts)-1
		if last {
			if evar := obj.(*expvar.Map).Get(name); evar != nil {
				return evar
			}
			switch typ {
			case intVar:
				v := new(expvar.Int)
				obj.(*expvar.Map).Set(name, v)
				return v
			case stringVar:
				v := new(expvar.String)
				obj.(*expvar.Map).Set(name, v)
				return v
			}
		} else {
			if mapp := obj.(*expvar.Map).Get(name); mapp != nil {
				obj = mapp
				continue
			}
			mapp := new(expvar.Map)
			obj.(*expvar.Map).Set(name, mapp)
			obj = mapp
			continue
		}
	}
	panic("unexpected")
}
