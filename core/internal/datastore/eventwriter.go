// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/backoff"
	"github.com/meergo/meergo/tools/metrics"
)

// eventFlusherTuning configures an EventWriter.
type eventFlusherTuning struct {

	// QueueSize is the size of the internal channel used to queue incoming events.
	// A larger buffer can absorb short slowdowns in flush() without blocking writers.
	QueueSize int

	// BatchSize is the preferred number of events per flush.
	// When the buffer reaches this size, EventWriter flushes as soon as possible.
	BatchSize int

	// MaxBatchSize is the maximum number of buffered events before forcing a flush.
	// When the buffer reaches this size, EventWriter flushes immediately.
	MaxBatchSize int

	// MinFlushInterval is the smallest delay used by the adaptive scheduler.
	// It avoids flushing too often when events arrive slowly but continuously.
	MinFlushInterval time.Duration

	// MaxFlushLatency is the longest time an event is allowed to wait in the buffer.
	// Even if the batch is not full, EventWriter will flush within this time.
	MaxFlushLatency time.Duration

	// IdleFlushDelay triggers a flush when no new events arrive for this duration,
	// as long as there are buffered events to write.
	IdleFlushDelay time.Duration

	// RateAlpha is the EWMA smoothing factor for the event rate estimate.
	// Valid range is [0, 1]. Higher means more weight to recent observations.
	RateAlpha float64
}

type eventRow struct {
	pipeline int
	row      []any
	ack      streams.Ack
}

type EventWriter struct {
	events  chan<- eventRow
	flusher *eventFlusher
	closed  atomic.Bool
}

// newEventWriter constructs and starts a EventWriter.
// The returned EventWriter is ready to use.
func newEventWriter(store *Store) *EventWriter {
	tuning := eventFlusherTuning{
		QueueSize:        8192,
		BatchSize:        5000,
		MaxBatchSize:     25000,
		MinFlushInterval: 250 * time.Millisecond,
		MaxFlushLatency:  10 * time.Second,
		IdleFlushDelay:   750 * time.Millisecond,
		RateAlpha:        0.4,
	}

	events := make(chan eventRow, tuning.QueueSize)
	flusher := newEventFlusher(store, events, tuning)
	flusher.Start()

	return &EventWriter{events: events, flusher: flusher}
}

// Write persists an event to the store.
// It returns an error only if the context is canceled.
func (w *EventWriter) Write(ctx context.Context, event streams.Event, pipeline int) error {

	row := make([]any, 66)

	// connectionId
	row[0] = event.Attributes["connectionId"]

	// anonymousId
	row[1] = event.Attributes["anonymousId"]

	// channel
	row[2] = event.Attributes["channel"]

	// category
	row[3] = event.Attributes["category"]

	// context.
	if eventContext, ok := event.Attributes["context"].(map[string]any); ok {

		// app
		if app, ok := eventContext["app"].(map[string]any); ok {
			row[4] = app["name"]
			row[5] = app["version"]
			row[6] = app["build"]
			row[7] = app["namespace"]
		}

		// browser
		if browser, ok := eventContext["browser"].(map[string]any); ok {
			row[8] = browser["name"]
			row[9] = browser["other"]
			row[10] = browser["version"]
		}

		// campaign
		if campaign, ok := eventContext["campaign"].(map[string]any); ok {
			row[11] = campaign["name"]
			row[12] = campaign["source"]
			row[13] = campaign["medium"]
			row[14] = campaign["term"]
			row[15] = campaign["content"]
		}

		// device
		if device, ok := eventContext["device"].(map[string]any); ok {
			row[16] = device["id"]
			row[17] = device["advertisingId"]
			row[18] = device["adTrackingEnabled"]
			row[19] = device["manufacturer"]
			row[20] = device["model"]
			row[21] = device["name"]
			row[22] = device["type"]
			row[23] = device["token"]
		}

		// ip
		row[24] = eventContext["ip"]

		// library
		if library, ok := eventContext["library"].(map[string]any); ok {
			row[25] = library["name"]
			row[26] = library["version"]
		}
		// locale
		row[27] = eventContext["locale"]

		// location
		if location, ok := eventContext["location"].(map[string]any); ok {
			row[28] = location["city"]
			row[29] = location["country"]
			row[30] = location["latitude"]
			row[31] = location["longitude"]
			row[32] = location["speed"]
		}

		// network
		if network, ok := eventContext["network"].(map[string]any); ok {
			row[33] = network["bluetooth"]
			row[34] = network["carrier"]
			row[35] = network["cellular"]
			row[36] = network["wifi"]
		}

		// os
		if os, ok := eventContext["os"].(map[string]any); ok {
			row[37] = os["name"]
			row[38] = os["other"]
			row[39] = os["version"]
		}

		// page
		if page, ok := eventContext["page"].(map[string]any); ok {
			row[40] = page["path"]
			row[41] = page["referrer"]
			row[42] = page["search"]
			row[43] = page["title"]
			row[44] = page["url"]
		}

		// referrer
		if referrer, ok := eventContext["referrer"].(map[string]any); ok {
			row[45] = referrer["id"]
			row[46] = referrer["type"]
		}

		// screen
		if screen, ok := eventContext["screen"].(map[string]any); ok {
			row[47] = screen["width"]
			row[48] = screen["height"]
			row[49] = screen["density"]
		}

		// session
		if session, ok := eventContext["session"].(map[string]any); ok {
			row[50] = session["id"]
			row[51] = session["start"]
		}

		// timezone
		row[52] = eventContext["timezone"]

		// userAgent
		row[53] = eventContext["userAgent"]
	}

	// event
	row[54] = event.Attributes["event"]

	// groupId
	row[55] = event.Attributes["groupId"]

	// messageId
	row[56] = event.Attributes["messageId"]

	// name
	row[57] = event.Attributes["name"]

	// properties
	row[58] = event.Attributes["properties"]

	// receivedAt
	row[59] = event.Attributes["receivedAt"]

	// sentAt
	row[60] = event.Attributes["sentAt"]

	// timestamp
	row[61] = event.Attributes["timestamp"]

	// traits
	row[62] = event.Attributes["traits"]

	// type
	row[63] = event.Attributes["type"]

	// previousId
	row[64] = event.Attributes["previousId"]

	// userId
	row[65] = event.Attributes["userId"]

	select {
	case w.events <- eventRow{pipeline: pipeline, row: row, ack: event.Ack}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

}

// Close closes the EventWriter. It panics if it has been already closed.
//
// When Close is called, no other calls to EventWriter's methods should be in
// progress and no other shall be made.
func (w *EventWriter) Close(ctx context.Context) error {
	if w.closed.Swap(true) {
		panic("EventWriter already closed")
	}
	return w.flusher.Close(ctx)
}

type eventFlusher struct {
	store  *Store
	tuning eventFlusherTuning

	events <-chan eventRow

	// stop and done are used to stop the loop and wait a flush termination.
	stop chan struct{}
	done chan struct{}

	// flushCtx is canceled when any Close's ctx is canceled.
	// This allows an in-flight flush to be interrupted by Close's ctx.
	flushCtx    context.Context
	cancelFlush context.CancelFunc
}

func newEventFlusher(store *Store, events <-chan eventRow, tuning eventFlusherTuning) *eventFlusher {
	ctx, cancel := context.WithCancel(context.Background())
	return &eventFlusher{
		store:       store,
		events:      events,
		tuning:      tuning,
		stop:        make(chan struct{}),
		done:        make(chan struct{}),
		flushCtx:    ctx,
		cancelFlush: cancel,
	}
}

// Start returns immediately and begins the loop in a goroutine.
func (f *eventFlusher) Start() {
	go f.loop()
}

// Close interrupts scheduling of any new flush and terminates the loop.
// If a flush is in progress, Close waits for it to finish unless ctx is
// canceled, in which case the in-flight flush context is canceled and Close
// returns ctx.Err().
func (f *eventFlusher) Close(ctx context.Context) error {
	close(f.stop)
	select {
	case <-f.done:
		f.cancelFlush() // ensure resources associated with the flush context are released
		return nil
	case <-ctx.Done():
		f.cancelFlush() // abort an in-flight flush, is any
		<-f.done
		return ctx.Err()
	}
}

func (f *eventFlusher) loop() {

	defer close(f.done)

	tuning := f.tuning

	// firstBuffered is the time when the first event was buffered.
	var firstBuffered time.Time

	var lastArrival time.Time

	var rate float64 // events/sec (EWMA)

	// idleTimer keeps latency low at low traffic and makes tests faster.
	//
	// Reset: on each incoming event
	// Stop:  when the events are flushed
	idleTimer := time.NewTimer(time.Hour) // <- tuning.IdleFlushDelay (750ms)

	// adaptiveTimer schedules a flush while events keep arriving (so idleTimer may not fire).
	// It estimates how fast events arrive (EWMA) and sets the timer to roughly when we should reach
	// tuning.BatchSize events in the buffer (kept within [tuning.MinFlushInterval, tuning.MaxFlushLatency]
	// and not later than maxTimer).
	//
	// Reset: on each incoming event (after updating the rate estimate)
	// Stop:  when the events are flushed
	adaptiveTimer := time.NewTimer(time.Hour)

	// maxTimer guarantees the oldest buffered event waits at most tuning.MaxFlushLatency.
	//
	// Reset: once the first event is buffered
	// Stop:  when the events are flushed
	maxTimer := time.NewTimer(time.Hour) // <- tuning.MaxFlushLatency (5s)

	stopTimers := func() {
		idleTimer.Stop()
		adaptiveTimer.Stop()
		maxTimer.Stop()
	}

	// Since events is empty, stop all timers.
	stopTimers()

	events := make([]eventRow, 0, tuning.BatchSize)

	flush := func() {

		stopTimers()

		if len(events) == 0 {
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

		// Flush buffered events. If the flush is interrupted (Close canceled the context),
		// return and let the main loop exit without starting another flush.
		err := f.flush(flushCtx, events)
		if err != nil {
			return
		}

		if cap(events) > tuning.MaxBatchSize {
			events = make([]eventRow, 0, tuning.BatchSize)
		} else {
			events = events[:0]
		}
		firstBuffered = time.Time{}
	}

	for {

		select {

		case event := <-f.events:

			// EWMA rate estimate from inter-arrival time.
			now := time.Now()
			if !lastArrival.IsZero() {
				dt := now.Sub(lastArrival).Seconds()
				if dt > 0 {
					inst := 1.0 / dt
					if rate == 0 {
						rate = inst
					} else {
						alpha := tuning.RateAlpha
						rate = alpha*inst + (1-alpha)*rate
					}
				}
			}

			events = append(events, event)

			if len(events) == tuning.MaxBatchSize {
				flush()
				continue
			}

			idleTimer.Reset(tuning.IdleFlushDelay)
			if len(events) == 1 {
				firstBuffered = now
				maxTimer.Reset(tuning.MaxFlushLatency)
			}
			lastArrival = now

			// Adaptive schedule: expected time to reach BatchSize.
			if rate > 0 {
				remaining := float64(tuning.BatchSize - len(events))
				if remaining < 0 {
					remaining = 0
				}
				sec := remaining / rate
				if math.IsNaN(sec) || math.IsInf(sec, 0) || sec < 0 {
					sec = float64(tuning.MaxFlushLatency / time.Second)
				}
				d := time.Duration(sec * float64(time.Second))
				if d < tuning.MinFlushInterval {
					d = tuning.MinFlushInterval
				}
				if d > tuning.MaxFlushLatency {
					d = tuning.MaxFlushLatency
				}
				// Respect max-latency deadline.
				if !firstBuffered.IsZero() {
					deadline := firstBuffered.Add(tuning.MaxFlushLatency)
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

		if len(events) == 0 {
			stopTimers()
		}

	}
}

// flush writes the given events.
// It returns ctx.Err() on context cancellation.
func (f *eventFlusher) flush(ctx context.Context, events []eventRow) error {

	metrics.Increment("EventWriter.flush.calls", 1)

	rows := make([][]any, len(events))
	for i, event := range events {
		rows[i] = event.row
	}

	for {
		metrics.Increment("EventWriter.flush.for_loop_iterations", 1)
		err := f.store.warehouse().Merge(ctx, eventsMergeTable, rows, nil)
		if err != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
			slog.Error("core/datastore: cannot flush the event queue", "error", err)
			select {
			case <-time.After(time.Duration(rand.IntN(2000)) * time.Millisecond):
			case <-ctx.Done():
				return ctx.Err()
			}
			continue
		}
		for _, event := range events {
			event.ack()
			f.store.ds.metrics.FinalizePassed(event.pipeline, 1)
		}
		return nil
	}

}
