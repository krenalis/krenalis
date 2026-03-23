// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/krenalis/krenalis/core/internal/streams"
)

type EventWriter struct {
	workspace int
	events    chan<- flusherRow[[]any]
	flusher   *flusher[[]any]
	closed    atomic.Bool
}

// newEventWriter constructs and starts a EventWriter.
// The returned EventWriter is ready to use.
func newEventWriter(store *Store) *EventWriter {
	w := &EventWriter{workspace: store.workspace}
	opts := flusherOptions{
		QueueSize:        8192,
		BatchSize:        5000,
		MaxBatchSize:     25000,
		MinFlushInterval: 250 * time.Millisecond,
		MaxFlushLatency:  10 * time.Second,
		IdleFlushDelay:   750 * time.Millisecond,
		RateAlpha:        0.4,
		MetricsFinalizer: store.ds.metrics.FinalizePassed,
		LogError:         w.logError,
	}
	w.flusher = newFlusher(opts, store.mc.StartOperation, func(ctx context.Context, events [][]any) error {
		return store.warehouse().Merge(ctx, eventsMergeTable, events, nil)
	})
	w.events = w.flusher.Ch()
	return w
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
	case w.events <- flusherRow[[]any]{pipeline: pipeline, row: row, ack: event.Ack}:
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
	// Close the flusher.
	return w.flusher.Close(ctx)
}

// logError logs an error that occurred while flushing the events.
func (w *EventWriter) logError(err error) {
	slog.Warn("cannot flush events to the data warehouse; retrying.", "workspace", w.workspace, "error", err)
}
