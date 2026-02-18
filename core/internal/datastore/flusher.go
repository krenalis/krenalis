// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"math"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/backoff"
)

type (
	startOperationFunc func(ctx context.Context, modes allowedMode) (context.Context, func(), error)
	flushFunc[T any]   func(context.Context, []T) error
)

type flusherRow[T any] struct {
	pipeline int         // pipeline identifier; if 0, no metrics are recorded
	key      any         // deduplication key; if nil, deduplication is disabled
	row      T           // row to be flushed
	ack      streams.Ack // ack callback; if nil, it is not invoked
}

type flusher[T any] struct {
	rows  chan flusherRow[T]
	close struct {
		atomic.Bool                    // true if Close or Stop has been called
		stop        chan bool          // signals the loop to stop; the value indicates whether to drain
		ctx         context.Context    // context passed to the flush function; canceled by close's cancel
		cancel      context.CancelFunc // cancels an in-flight flush
		done        chan struct{}      // closed once the loop has terminated
	}
}

func newFlusher[T any](opts flusherOptions, startOperation startOperationFunc, flush flushFunc[T]) *flusher[T] {
	ctx, cancel := context.WithCancel(context.Background())
	f := &flusher[T]{
		rows: make(chan flusherRow[T], opts.QueueSize),
	}
	f.close.stop = make(chan bool, 1)
	f.close.done = make(chan struct{})
	f.close.ctx = ctx
	f.close.cancel = cancel
	go f.loop(opts, startOperation, flush)
	return f
}

// Ch returns the channel. The Close and Stop methods close this channel.
func (f *flusher[T]) Ch() chan<- flusherRow[T] {
	return f.rows
}

// Close closes the flusher and waits for all pending rows to be flushed.
// If ctx is canceled or expires, the current flush is aborted and no
// additional rows are flushed; Close returns ctx.Err().
//
// After Close is called, subsequent calls to Close or Stop do nothing.
func (f *flusher[T]) Close(ctx context.Context) error {
	return f.stop(ctx, true)
}

// Stop stops and closes the flusher, waiting only for any in-flight flush to
// complete and not flushing any additional rows. If ctx is canceled or expires,
// the current flush is aborted and Stop returns ctx.Err().
//
// After Stop is called, subsequent calls to Stop or Close do nothing.
func (f *flusher[T]) Stop(ctx context.Context) error {
	return f.stop(ctx, false)
}

// stop stops the flusher.
// If drain is true, all pending rows are flushed before returning.
func (f *flusher[T]) stop(ctx context.Context, drain bool) error {
	if f.close.Swap(true) {
		return nil
	}
	// Signals the loop to stop.
	f.close.stop <- drain
	close(f.close.stop)
	close(f.rows)
	// Waits until the loop terminates or the context is canceled.
	select {
	case <-f.close.done:
		f.close.cancel() // ensure resources associated with the context are released
		return nil
	case <-ctx.Done():
		f.close.cancel() // abort an in-flight flush, if any
		<-f.close.done
		return ctx.Err()
	}
}

func (f *flusher[T]) loop(opts flusherOptions, startOperation startOperationFunc, innerFlush flushFunc[T]) {

	var (
		dedup = make(map[any]int, opts.BatchSize)
		rows  = make([]T, 0, opts.BatchSize)
		acks  = make([]streams.Ack, 0, opts.BatchSize)
	)

	var metrics map[int]int
	if opts.MetricsFinalizer != nil {
		metrics = make(map[int]int) // pipeline to count
	}

	// firstBuffered is the time when the first row was buffered.
	var firstBuffered time.Time

	var lastArrival time.Time

	var rate float64 // rows/sec (EWMA)

	// idleTimer keeps latency low at low traffic and makes tests faster.
	//
	// Reset: on each incoming row
	// Stop:  when the rows are flushed
	idleTimer := time.NewTimer(time.Hour) // <- opts.IdleFlushDelay (750ms)

	// adaptiveTimer schedules a flush while rows keep arriving (so idleTimer may not fire).
	// It estimates how fast rows arrive (EWMA) and sets the timer to roughly when we should reach
	// opts.BatchSize rows in the buffer (kept within [opts.MinFlushInterval, opts.MaxFlushLatency]
	// and not later than maxTimer).
	//
	// Reset: on each incoming row (after updating the rate estimate)
	// Stop:  when the rows are flushed
	adaptiveTimer := time.NewTimer(time.Hour)

	// maxTimer guarantees the oldest buffered row waits at most opts.MaxFlushLatency.
	//
	// Reset: once the first row is buffered
	// Stop:  when the rows are flushed
	maxTimer := time.NewTimer(time.Hour) // <- opts.MaxFlushLatency (5s)

	stopTimers := func() {
		idleTimer.Stop()
		adaptiveTimer.Stop()
		maxTimer.Stop()
	}

	defer func() {
		stopTimers()
		close(f.close.done)
	}()

	// Since rows is empty, stop all timers.
	stopTimers()

	flush := func() {

		stopTimers()
		firstBuffered = time.Time{}

		if len(rows) == 0 {
			return
		}

		var flushCtx context.Context

		bo := backoff.New(1000)
		bo.SetCap(10 * time.Second)

		for bo.Next(f.close.ctx) {
			ctx, done, err := startOperation(f.close.ctx, normalMode)
			if err != nil {
				continue
			}
			flushCtx = ctx
			defer done()
			break
		}
		if err := f.close.ctx.Err(); err != nil {
			return
		}

		// Flush buffered rows. If the flush is interrupted (Close canceled the context),
		// return and let the main loop exit without starting another flush.
		var latestErrorMsg string
		bo = backoff.New(1000)
		bo.SetCap(10 * time.Second)
		for bo.Next(flushCtx) {
			err := innerFlush(flushCtx, rows)
			if err != nil {
				if opts.LogError != nil {
					if msg := err.Error(); msg != latestErrorMsg {
						opts.LogError(err)
						latestErrorMsg = msg
					}
				}
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
			opts.MetricsFinalizer(pipeline, count)
		}

		clear(dedup)
		if cap(rows) > opts.MaxBatchSize {
			rows = make([]T, 0, opts.BatchSize)
		} else {
			rows = rows[:0]
		}
		if cap(acks) > opts.MaxBatchSize {
			acks = make([]streams.Ack, 0, opts.BatchSize)
		} else {
			acks = acks[:0]
		}
		clear(metrics)
	}

	var stop bool  // indicates that processing must stop
	var drain bool // if stopping, specifies whether to drain remaining data

	for {

		select {

		case row, ok := <-f.rows:
			// Return and optionally drain when there are no more rows.
			if !ok {
				if drain || <-f.close.stop {
					flush()
				}
				return
			}
			// Return immediately if processing must stop without draining.
			if stop && !drain {
				return
			}

			var now time.Time
			if !stop {
				now = time.Now()
			}

			if row.ack != nil {
				acks = append(acks, row.ack)
			}
			if metrics != nil && row.pipeline != 0 {
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
			// Append the row.
			if !duplicated {
				rows = append(rows, row.row)
				if len(rows) == opts.MaxBatchSize {
					flush()
				}
			}

			// Skip scheduling and rate updates if the flusher is stopped.
			if stop {
				continue
			}

			// EWMA rate estimate from inter-arrival time.
			if !lastArrival.IsZero() {
				dt := now.Sub(lastArrival).Seconds()
				if dt > 0 {
					inst := 1.0 / dt
					if rate == 0 {
						rate = inst
					} else {
						alpha := opts.RateAlpha
						rate = alpha*inst + (1-alpha)*rate
					}
				}
			}

			// Continue if there are no pending rows.
			if len(rows) == 0 {
				continue
			}

			if !duplicated && len(rows) == 1 {
				firstBuffered = now
				maxTimer.Reset(opts.MaxFlushLatency)
			}

			idleTimer.Reset(opts.IdleFlushDelay)
			lastArrival = now

			// Adaptive schedule: expected time to reach BatchSize.
			if rate > 0 {
				remaining := float64(opts.BatchSize - len(rows))
				if remaining < 0 {
					remaining = 0
				}
				sec := remaining / rate
				if math.IsNaN(sec) || math.IsInf(sec, 0) || sec < 0 {
					sec = float64(opts.MaxFlushLatency / time.Second)
				}
				d := min(max(time.Duration(sec*float64(time.Second)), opts.MinFlushInterval), opts.MaxFlushLatency)
				// Respect max-latency deadline.
				if !firstBuffered.IsZero() {
					deadline := firstBuffered.Add(opts.MaxFlushLatency)
					until := time.Until(deadline)
					if until <= 0 {
						// Deadline already exceeded: flush immediately instead of Reset(0).
						flush()
						continue
					}
					if until < d {
						d = until
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

		case d, ok := <-f.close.stop:
			if ok {
				stop = true
				drain = d
			}
		}

	}
}

// flusherOptions contains the options used to configure the flusher.
type flusherOptions struct {

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

	// MetricsFinalizer receives the number of flushed rows per pipeline
	// after a successful flush. It is called once per pipeline with its count.
	MetricsFinalizer func(pipeline int, count int)

	// LogError is invoked when a flush attempt fails. Duplicate consecutive
	// error messages are suppressed until the message changes.
	LogError func(err error)
}
