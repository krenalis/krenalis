// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/events"
	"github.com/meergo/meergo/tools/metrics"
)

type EventWriter struct {
	store     *Store
	ack       EventWriterAckFunc
	mu        sync.Mutex // for 'rows' and 'pipelines' fields
	rows      [][]any
	pipelines []int
	close     struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
	}
}

func newEventWriter(store *Store, ack EventWriterAckFunc) *EventWriter {
	ew := &EventWriter{
		store: store,
		ack:   ack,
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
func (ew *EventWriter) Close(ctx context.Context) {
	stop := context.AfterFunc(ctx, func() {
		ew.close.cancelCtx()
	})
	ew.flush()
	stop()
}

// Write writes an event to the store.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
func (ew *EventWriter) Write(event events.Event, pipeline int) error {

	row := make([]any, 66)

	// connectionId
	row[0] = event["connectionId"]

	// anonymousId
	row[1] = event["anonymousId"]

	// channel
	row[2] = event["channel"]

	// category
	row[3] = event["category"]

	// context.
	if eventContext, ok := event["context"].(map[string]any); ok {

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
			row[45] = referrer["name"]
			row[46] = referrer["version"]
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
	row[54] = event["event"]

	// groupId
	row[55] = event["groupId"]

	// messageId
	row[56] = event["messageId"]

	// name
	if eventContext, ok := event["context"].(map[string]any); ok {
		row[57] = eventContext["name"]
	}

	// properties
	row[58] = event["properties"]

	// receivedAt
	row[59] = event["receivedAt"]

	// sentAt
	row[60] = event["sentAt"]

	// timestamp
	row[61] = event["timestamp"]

	// traits
	row[62] = event["traits"]

	// type
	row[63] = event["type"]

	// previousId
	row[64] = event["previousId"]

	// userId
	row[65] = event["userId"]

	ew.mu.Lock()
	ew.rows = append(ew.rows, row)
	ew.pipelines = append(ew.pipelines, pipeline)
	ew.mu.Unlock()

	return nil
}

func (ew *EventWriter) flush() {

	metrics.Increment("EventWriter.flush.calls", 1)

	ew.mu.Lock()
	rows, pipelines := ew.rows, ew.pipelines
	ew.rows, ew.pipelines = nil, nil
	ew.mu.Unlock()

	if rows == nil {
		return
	}

	ctx, done, err := ew.store.mc.StartOperation(ew.close.ctx, normalMode)
	if err != nil {
		// Warehouse mode is not normal: discard events.
		if ew.ack != nil {
			events := make([]AckEvent, len(rows))
			for i := range rows {
				events[i].Pipeline = pipelines[i]
			}
			metrics.Increment("EventWriter.ack_sents", 1)
			ew.ack(events, err)
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
			slog.Error("core/datastore: cannot flush the event queue", "err", err)
			select {
			case <-time.After(time.Duration(rand.IntN(2000)) * time.Millisecond):
			case <-ctx.Done():
				return
			}
			continue
		}
		if ew.ack != nil {
			events := make([]AckEvent, len(rows))
			for i := range rows {
				events[i].Pipeline = pipelines[i]
			}
			metrics.Increment("EventWriter.ack_sents", 1)
			ew.ack(events, nil)
		}
		return
	}

}
