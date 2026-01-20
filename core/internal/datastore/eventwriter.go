// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/metrics"
)

type EventWriter struct {
	store     *Store
	mu        sync.Mutex // for 'rows' and 'pipelines' fields
	rows      [][]any
	pipelines []int
	acks      []streams.Ack
	close     struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
	}
}

func newEventWriter(store *Store) *EventWriter {
	ew := &EventWriter{
		store: store,
	}
	ew.close.ctx, ew.close.cancelCtx = context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(flushEventsQueueTimeout)
		done := ew.close.ctx.Done()
		for {
			select {
			case <-ticker.C:
				ew.flush()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return ew
}

// Close closes the event writer.
func (ew *EventWriter) Close() {
	ew.close.cancelCtx()
}

// Write writes an event to the store.
func (ew *EventWriter) Write(event streams.Event, pipeline int) {

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
	if eventContext, ok := event.Attributes["context"].(map[string]any); ok {
		row[57] = eventContext["name"]
	}

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

	ew.mu.Lock()
	ew.rows = append(ew.rows, row)
	ew.pipelines = append(ew.pipelines, pipeline)
	ew.acks = append(ew.acks, event.Ack)
	ew.mu.Unlock()

}

func (ew *EventWriter) flush() {

	metrics.Increment("EventWriter.flush.calls", 1)

	ew.mu.Lock()
	rows, pipelines, acks := ew.rows, ew.pipelines, ew.acks
	ew.rows, ew.pipelines, ew.acks = nil, nil, nil
	ew.mu.Unlock()

	if rows == nil {
		return
	}

	ctx, done, err := ew.store.mc.StartOperation(ew.close.ctx, normalMode)
	if err != nil {
		// Warehouse mode is not normal: discard events.
		for _, ack := range acks {
			ack()
		}
		for i := range rows {
			ew.store.ds.metrics.FinalizeFailed(pipelines[i], 1, err.Error())
		}
		return
	}
	defer done()

	for {
		metrics.Increment("EventWriter.flush.for_loop_iterations", 1)
		err = ew.store.warehouse().Merge(ctx, eventsMergeTable, rows, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("core/datastore: cannot flush the event queue", "error", err)
			select {
			case <-time.After(time.Duration(rand.IntN(2000)) * time.Millisecond):
			case <-ctx.Done():
				return
			}
			continue
		}
		for _, ack := range acks {
			ack()
		}
		for i := range rows {
			ew.store.ds.metrics.FinalizePassed(pipelines[i], 1)
		}
		return
	}

}
