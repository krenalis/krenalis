// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"github.com/meergo/meergo/core/internal/db"

	"github.com/prometheus/client_golang/prometheus"
)

// dbPoolMetrics holds Prometheus metrics tracking database connection pool
// statistics.
type dbPoolMetrics struct {
	db *db.DB

	acquiredConns *prometheus.Desc
	maxConns      *prometheus.Desc
	acquireDur    *prometheus.Desc
	acquireCount  *prometheus.Desc
	newConnsCount *prometheus.Desc
}

// registerDBPoolMetrics registers metrics for monitoring the connection pool of
// the given database.
func registerDBPoolMetrics(db *db.DB) *dbPoolMetrics {
	metrics := &dbPoolMetrics{
		db: db,
		acquiredConns: prometheus.NewDesc(
			"meergo_db_acquired_conns",
			"Current number of connections in use",
			nil, nil,
		),
		maxConns: prometheus.NewDesc(
			"meergo_db_max_conns",
			"Configured maximum number of simultaneous connections",
			nil, nil,
		),
		acquireDur: prometheus.NewDesc(
			"meergo_db_acquire_duration_seconds_total",
			"Cumulative seconds spent acquiring connections",
			nil, nil,
		),
		acquireCount: prometheus.NewDesc(
			"meergo_db_acquire_count_total",
			"Total number of successful connection acquisitions",
			nil, nil,
		),
		newConnsCount: prometheus.NewDesc(
			"meergo_db_new_conns_count_total",
			"Total number of newly opened connections (pool churn indicator)",
			nil, nil,
		),
	}
	prometheus.MustRegister(metrics)
	return metrics
}

// Collect sends the current metric values to the provided Prometheus metrics
// channel. It is safe for concurrent use by multiple goroutines.
func (c *dbPoolMetrics) Collect(ch chan<- prometheus.Metric) {
	stats := c.db.PoolStats()
	// Gauges
	ch <- prometheus.MustNewConstMetric(
		c.acquiredConns, prometheus.GaugeValue, float64(stats.AcquiredConns()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.maxConns, prometheus.GaugeValue, float64(stats.MaxConns()),
	)
	// Counters
	ch <- prometheus.MustNewConstMetric(
		c.acquireDur, prometheus.CounterValue, stats.AcquireDuration().Seconds(),
	)
	ch <- prometheus.MustNewConstMetric(
		c.acquireCount, prometheus.CounterValue, float64(stats.AcquireCount()),
	)
	ch <- prometheus.MustNewConstMetric(
		c.newConnsCount, prometheus.CounterValue, float64(stats.NewConnsCount()),
	)
}

// Describe sends the metric descriptors to the provided channel.
// It is safe for concurrent use by multiple goroutines.
func (c *dbPoolMetrics) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

// Unregister unregisters the metrics.
func (c *dbPoolMetrics) Unregister() {
	prometheus.Unregister(c)
}
