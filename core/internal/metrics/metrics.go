// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package metrics collects and stores pipeline and usage metrics in the
// database.
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/errors"
)

const flushInterval = time.Second

var (
	MinTime = TimeSlotToTime(0)           // 1970-01-01 00:00:00
	MaxTime = TimeSlotToTime(maxTimeslot) // 2262-04-11 23:47:00
)

// ErrMetricResultTooLarge indicates that a calculated metric value cannot be
// represented by the result type.
var ErrMetricResultTooLarge = errors.New("calculated metric result is too large")

// Metrics collects, stores, and queries pipeline and usage metrics.
type Metrics struct {
	db    *db.DB
	state *state.State

	Pipelines Pipelines
	Usage     Usage

	close struct {
		ctx    context.Context
		cancel context.CancelFunc
		stop   chan struct{}
		sync.WaitGroup
	}
}

// New returns a new Metrics and starts its storage workers.
func New(db *db.DB, state *state.State) *Metrics {
	m := &Metrics{
		db:    db,
		state: state,
	}
	m.Pipelines.metrics = m
	m.Pipelines.pending = map[string]*pipelineMetrics{}
	m.Pipelines.tick = 1
	m.Pipelines.stored.L = &m.Pipelines.RWMutex
	m.Usage.metrics = m
	m.Usage.events = map[eventCountKey]int{}
	m.close.ctx, m.close.cancel = context.WithCancel(context.Background())
	m.close.stop = make(chan struct{})
	m.close.Go(m.Pipelines.start)
	m.close.Go(m.Usage.start)
	return m
}

// Close closes the collection and ensures that any remaining metrics are
// stored. If the provided context expires before the operation completes, the
// ongoing store operation is interrupted and the method returns without
// guaranteeing that all metrics have been saved.
func (m *Metrics) Close(ctx context.Context) {
	stopCancel := context.AfterFunc(ctx, m.close.cancel)
	close(m.close.stop)
	m.close.Wait()
	stopCancel()
	m.close.cancel()
}

// TimeSlotFromTime returns the time slot for t that must be in UTC.
func TimeSlotFromTime(t time.Time) int32 {
	return int32(t.Unix() / int64(timeslotDuration.Seconds()))
}

// TimeSlotToTime converts a time slot back to a time.Time in UTC.
// It panics if ts is not in range [0,maxTimeslot].
func TimeSlotToTime(ts int32) time.Time {
	if ts < 0 || ts > maxTimeslot {
		panic("timeslot is out of range")
	}
	epoch := time.Unix(0, 0).UTC()
	return epoch.Add(time.Duration(ts) * timeslotDuration)
}
