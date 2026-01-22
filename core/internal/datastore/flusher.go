// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"math"
	"time"

	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/backoff"
)

type flusherRow[T any] struct {
	key      any
	pipeline int
	row      T
	ack      streams.Ack
}

type flusher[T any] struct {
	store *Store
	conf  flusherConf

	rows <-chan flusherRow[T]

	flush func(context.Context, []T) error

	// stop and done are used to stop the loop and wait a flush termination.
	stop chan struct{}
	done chan struct{}

	// flushCtx is canceled when any Close's ctx is canceled.
	// This allows an in-flight flush to be interrupted by Close's ctx.
	flushCtx    context.Context
	cancelFlush context.CancelFunc
}

func newFlusher[T any](store *Store, rows <-chan flusherRow[T], flush func(context.Context, []T) error, conf flusherConf) *flusher[T] {
	ctx, cancel := context.WithCancel(context.Background())
	f := &flusher[T]{
		store:       store,
		rows:        rows,
		flush:       flush,
		conf:        conf,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		flushCtx:    ctx,
		cancelFlush: cancel,
	}
	go f.loop()
	return f
}

// Close interrupts scheduling of any new flush and terminates the loop.
// If a flush is in progress, Close waits for it to finish unless ctx is
// canceled, in which case the in-flight flush context is canceled and Close
// returns ctx.Err().
func (f *flusher[T]) Close(ctx context.Context) error {
	close(f.stop)
	select {
	case <-f.done:
		f.cancelFlush() // ensure resources associated with the flush context are released
		return nil
	case <-ctx.Done():
		f.cancelFlush() // abort an in-flight flush, if any
		<-f.done
		return ctx.Err()
	}
}

func (f *flusher[T]) loop() {

	defer close(f.done)

	conf := f.conf

	var (
		dedup   = make(map[any]int, conf.BatchSize)
		rows    = make([]T, 0, conf.BatchSize)
		acks    = make([]streams.Ack, 0, conf.BatchSize)
		metrics = make(map[int]int) // pipeline to count
	)

	// firstBuffered is the time when the first row was buffered.
	var firstBuffered time.Time

	var lastArrival time.Time

	var rate float64 // rows/sec (EWMA)

	// idleTimer keeps latency low at low traffic and makes tests faster.
	//
	// Reset: on each incoming row
	// Stop:  when the rows are flushed
	idleTimer := time.NewTimer(time.Hour) // <- conf.IdleFlushDelay (750ms)

	// adaptiveTimer schedules a flush while rows keep arriving (so idleTimer may not fire).
	// It estimates how fast rows arrive (EWMA) and sets the timer to roughly when we should reach
	// conf.BatchSize rows in the buffer (kept within [conf.MinFlushInterval, conf.MaxFlushLatency]
	// and not later than maxTimer).
	//
	// Reset: on each incoming row (after updating the rate estimate)
	// Stop:  when the rows are flushed
	adaptiveTimer := time.NewTimer(time.Hour)

	// maxTimer guarantees the oldest buffered row waits at most conf.MaxFlushLatency.
	//
	// Reset: once the first row is buffered
	// Stop:  when the rows are flushed
	maxTimer := time.NewTimer(time.Hour) // <- conf.MaxFlushLatency (5s)

	stopTimers := func() {
		idleTimer.Stop()
		adaptiveTimer.Stop()
		maxTimer.Stop()
	}

	// Since rows is empty, stop all timers.
	stopTimers()

	flush := func() {

		stopTimers()

		if len(rows) == 0 {
			firstBuffered = time.Time{}
			return
		}

		var flushCtx context.Context

		bo := backoff.New(1000)
		bo.SetCap(10 * time.Second)

		for bo.Next(f.flushCtx) {
			ctx, done, err := f.store.mc.StartOperation(f.flushCtx, normalMode)
			if err != nil {
				continue
			}
			flushCtx = ctx
			defer done()
			break
		}
		if err := f.flushCtx.Err(); err != nil {
			return
		}

		// Flush buffered rows. If the flush is interrupted (Close canceled the context),
		// return and let the main loop exit without starting another flush.
		bo = backoff.New(1000)
		bo.SetCap(10 * time.Second)
		for bo.Next(flushCtx) {
			err := f.flush(flushCtx, rows)
			if err != nil {
				continue
			}
			break
		}
		if err := flushCtx.Err(); err != nil {
			return
		}
		for _, ack := range acks {
			ack()
		}
		for pipeline, count := range metrics {
			f.store.ds.metrics.FinalizePassed(pipeline, count)
		}

		clear(dedup)
		if cap(rows) > conf.MaxBatchSize {
			rows = make([]T, 0, conf.BatchSize)
		} else {
			rows = rows[:0]
		}
		if cap(acks) > conf.MaxBatchSize {
			acks = make([]streams.Ack, 0, conf.BatchSize)
		} else {
			acks = acks[:0]
		}
		clear(metrics)
		firstBuffered = time.Time{}
	}

	for {

		select {

		case row := <-f.rows:

			// EWMA rate estimate from inter-arrival time.
			now := time.Now()
			if !lastArrival.IsZero() {
				dt := now.Sub(lastArrival).Seconds()
				if dt > 0 {
					inst := 1.0 / dt
					if rate == 0 {
						rate = inst
					} else {
						alpha := conf.RateAlpha
						rate = alpha*inst + (1-alpha)*rate
					}
				}
			}

			if row.ack != nil {
				acks = append(acks, row.ack)
				metrics[row.pipeline]++
			}

			// Check if the row is duplicated.
			var duplicated bool
			if row.key != nil {
				if i, ok := dedup[row.key]; ok {
					rows[i] = row.row
					duplicated = true
				} else {
					dedup[row.key] = len(rows)
				}
			}
			if !duplicated {
				rows = append(rows, row.row)
				if len(rows) == conf.MaxBatchSize {
					flush()
					continue
				}
				if len(rows) == 1 {
					firstBuffered = now
					maxTimer.Reset(conf.MaxFlushLatency)
				}
			}

			idleTimer.Reset(conf.IdleFlushDelay)
			lastArrival = now

			// Adaptive schedule: expected time to reach BatchSize.
			if rate > 0 {
				remaining := float64(conf.BatchSize - len(rows))
				if remaining < 0 {
					remaining = 0
				}
				sec := remaining / rate
				if math.IsNaN(sec) || math.IsInf(sec, 0) || sec < 0 {
					sec = float64(conf.MaxFlushLatency / time.Second)
				}
				d := time.Duration(sec * float64(time.Second))
				if d < conf.MinFlushInterval {
					d = conf.MinFlushInterval
				}
				if d > conf.MaxFlushLatency {
					d = conf.MaxFlushLatency
				}
				// Respect max-latency deadline.
				if !firstBuffered.IsZero() {
					deadline := firstBuffered.Add(conf.MaxFlushLatency)
					until := time.Until(deadline)
					if until < d {
						d = until
						if d < 0 {
							d = 0
						}
					}
				}
				adaptiveTimer.Reset(d)
			}

		case <-idleTimer.C:
			flush()

		case <-adaptiveTimer.C:
			flush()

		case <-maxTimer.C:
			flush()

		case <-f.stop:
			stopTimers()
			return

		}

		if len(rows) == 0 {
			stopTimers()
		}

	}
}

// flusherConf configures a flusher.
type flusherConf struct {

	// QueueSize is the size of the internal channel used to queue incoming rows.
	// A larger buffer can absorb short slowdowns in flush() without blocking writers.
	QueueSize int

	// BatchSize is the preferred number of rows per flush.
	// When the buffer reaches this size, EventWriter flushes as soon as possible.
	BatchSize int

	// MaxBatchSize is the maximum number of buffered rows before forcing a flush.
	// When the buffer reaches this size, EventWriter flushes immediately.
	MaxBatchSize int

	// MinFlushInterval is the smallest delay used by the adaptive scheduler.
	// It avoids flushing too often when rows arrive slowly but continuously.
	MinFlushInterval time.Duration

	// MaxFlushLatency is the longest time a row is allowed to wait in the buffer.
	// Even if the batch is not full, EventWriter will flush within this time.
	MaxFlushLatency time.Duration

	// IdleFlushDelay triggers a flush when no new rows arrive for this duration,
	// as long as there are buffered rows to write.
	IdleFlushDelay time.Duration

	// RateAlpha is the EWMA smoothing factor for the row rate estimate.
	// Valid range is [0, 1]. Higher means more weight to recent observations.
	RateAlpha float64
}
