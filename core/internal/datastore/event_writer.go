//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/events"
	"github.com/meergo/meergo/metrics"
)

type EventWriter struct {
	store   *Store
	ack     EventWriterAckFunc
	mu      sync.Mutex // for 'rows' and 'actions' fields
	rows    [][]any
	actions []int
	close   struct {
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
func (ew *EventWriter) Write(event events.Event, action int) error {

	row := make([]any, 67)

	// id
	row[0] = event["id"]

	// connection
	row[1] = event["connection"]

	// anonymousId
	row[2] = event["anonymousId"]

	// channel
	row[3] = event["channel"]

	// category
	row[4] = event["category"]

	// context.
	if eventContext, ok := event["context"].(map[string]any); ok {

		// app
		if app, ok := eventContext["app"].(map[string]any); ok {
			row[5] = app["name"]
			row[6] = app["version"]
			row[7] = app["build"]
			row[8] = app["namespace"]
		}

		// browser
		if browser, ok := eventContext["browser"].(map[string]any); ok {
			row[9] = browser["name"]
			row[10] = browser["other"]
			row[11] = browser["version"]
		}

		// campaign
		if campaign, ok := eventContext["campaign"].(map[string]any); ok {
			row[12] = campaign["name"]
			row[13] = campaign["source"]
			row[14] = campaign["medium"]
			row[15] = campaign["term"]
			row[16] = campaign["content"]
		}

		// device
		if device, ok := eventContext["device"].(map[string]any); ok {
			row[17] = device["id"]
			row[18] = device["advertisingId"]
			row[19] = device["adTrackingEnabled"]
			row[20] = device["manufacturer"]
			row[21] = device["model"]
			row[22] = device["name"]
			row[23] = device["type"]
			row[24] = device["token"]
		}

		// ip
		row[25] = eventContext["ip"]

		// library
		if library, ok := eventContext["library"].(map[string]any); ok {
			row[26] = library["name"]
			row[27] = library["version"]
		}
		// locale
		row[28] = eventContext["locale"]

		// location
		if location, ok := eventContext["location"].(map[string]any); ok {
			row[29] = location["city"]
			row[30] = location["country"]
			row[31] = location["latitude"]
			row[32] = location["longitude"]
			row[33] = location["speed"]
		}

		// network
		if network, ok := eventContext["network"].(map[string]any); ok {
			row[34] = network["bluetooth"]
			row[35] = network["carrier"]
			row[36] = network["cellular"]
			row[37] = network["wifi"]
		}

		// os
		if os, ok := eventContext["os"].(map[string]any); ok {
			row[38] = os["name"]
			row[39] = os["other"]
			row[40] = os["version"]
		}

		// page
		if page, ok := eventContext["page"].(map[string]any); ok {
			row[41] = page["path"]
			row[42] = page["referrer"]
			row[43] = page["search"]
			row[44] = page["title"]
			row[45] = page["url"]
		}

		// referrer
		if referrer, ok := eventContext["referrer"].(map[string]any); ok {
			row[46] = referrer["name"]
			row[47] = referrer["version"]
		}

		// screen
		if screen, ok := eventContext["screen"].(map[string]any); ok {
			row[48] = screen["width"]
			row[49] = screen["height"]
			row[50] = screen["density"]
		}

		// session
		if session, ok := eventContext["session"].(map[string]any); ok {
			row[51] = session["id"]
			row[52] = session["start"]
		}

		// timezone
		row[53] = eventContext["timezone"]

		// userAgent
		row[54] = eventContext["userAgent"]
	}

	// event
	row[55] = event["event"]

	// groupId
	row[56] = event["groupId"]

	// messageId
	row[57] = event["messageId"]

	// name
	if eventContext, ok := event["context"].(map[string]any); ok {
		row[58] = eventContext["name"]
	}

	// properties
	row[59] = event["properties"]

	// receivedAt
	row[60] = event["receivedAt"]

	// sentAt
	row[61] = event["sentAt"]

	// timestamp
	row[62] = event["timestamp"]

	// traits
	row[63] = event["traits"]

	// type
	row[64] = event["type"]

	// previousId
	row[65] = event["previousId"]

	// userId
	row[66] = event["userId"]

	ew.mu.Lock()
	ew.rows = append(ew.rows, row)
	ew.actions = append(ew.actions, action)
	ew.mu.Unlock()

	return nil
}

func (ew *EventWriter) flush() {

	metrics.Increment("EventWriter.flush.calls", 1)

	ew.mu.Lock()
	rows, actions := ew.rows, ew.actions
	ew.rows, ew.actions = nil, nil
	ew.mu.Unlock()

	if rows == nil {
		return
	}

	ctx, done, err := ew.store.mc.StartOperation(ew.close.ctx, normalMode)
	if err != nil {
		// Warehouse mode is not normal: discard events.
		if ew.ack != nil {
			events := make([]AckEvent, len(rows))
			for i, event := range rows {
				events[i].ID = event[0].(string)
				events[i].Action = actions[i]
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
			for i, event := range rows {
				events[i].ID = event[0].(string)
				events[i].Action = actions[i]
			}
			metrics.Increment("EventWriter.ack_sents", 1)
			ew.ack(events, nil)
		}
		return
	}

}
