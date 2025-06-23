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

	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
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

var emptyJSONObject = json.Value("{}")

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

	row := make([]any, 65)

	// id
	row[0] = event["id"]

	// connection
	row[1] = event["connection"]

	// anonymousId
	row[2] = event["anonymousId"]

	// channel
	if channel, ok := event["channel"]; ok {
		row[3] = channel
	} else {
		row[3] = ""
	}

	// category
	if category, ok := event["category"]; ok {
		row[4] = category
	} else {
		row[4] = ""
	}

	eventContext := event["context"].(map[string]any)

	// app
	if app, ok := eventContext["app"].(map[string]any); ok {
		row[5] = app["name"]
		row[6] = app["version"]
		row[7] = app["build"]
		row[8] = app["namespace"]
	} else {
		row[5], row[6], row[7], row[8] = "", "", "", ""
	}

	// browser
	if browser, ok := eventContext["browser"].(map[string]any); ok {
		row[9] = browser["name"]
		row[10] = browser["other"]
		row[11] = browser["version"]
	} else {
		row[9], row[10], row[11] = "", "", ""
	}

	// campaign
	if campaign, ok := eventContext["campaign"].(map[string]any); ok {
		row[12] = campaign["name"]
		row[13] = campaign["source"]
		row[14] = campaign["medium"]
		row[15] = campaign["term"]
		row[16] = campaign["content"]
	} else {
		row[12], row[13], row[14], row[15], row[16] = "", "", "", "", ""
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
	} else {
		row[17], row[18], row[19], row[20], row[21], row[22], row[23], row[24] = "", "", false, "", "", "", "", ""
	}

	// ip
	if ip, ok := eventContext["ip"]; ok {
		row[25] = ip
	} else {
		row[25] = "0.0.0.0"
	}

	// library
	if library, ok := eventContext["library"].(map[string]any); ok {
		row[26] = library["name"]
		row[27] = library["version"]
	} else {
		row[26], row[27] = "", ""
	}

	// locale
	if locale, ok := eventContext["locale"]; ok {
		row[28] = locale
	} else {
		row[28] = ""
	}

	// location
	if location, ok := eventContext["location"].(map[string]any); ok {
		row[29] = location["city"]
		row[30] = location["country"]
		row[31] = location["latitude"]
		row[32] = location["longitude"]
		row[33] = location["speed"]
	} else {
		row[29], row[30], row[31], row[32], row[33] = "", "", 0.0, 0.0, 0.0
	}

	// network
	if network, ok := eventContext["network"].(map[string]any); ok {
		row[34] = network["bluetooth"]
		row[35] = network["carrier"]
		row[36] = network["cellular"]
		row[37] = network["wifi"]
	} else {
		row[34], row[35], row[36], row[37] = false, "", false, false
	}

	// os
	if os, ok := eventContext["os"].(map[string]any); ok {
		row[38] = os["name"]
		row[39] = os["version"]
	} else {
		row[38], row[39] = "", ""
	}

	// page
	if page, ok := eventContext["page"].(map[string]any); ok {
		row[40] = page["path"]
		row[41] = page["referrer"]
		row[42] = page["search"]
		row[43] = page["title"]
		row[44] = page["url"]
	} else {
		row[40], row[41], row[42], row[43], row[44] = "", "", "", "", ""
	}

	// referrer
	if referrer, ok := eventContext["referrer"].(map[string]any); ok {
		row[45] = referrer["name"]
		row[46] = referrer["version"]
	} else {
		row[45], row[46] = "", ""
	}

	// screen
	if screen, ok := eventContext["screen"].(map[string]any); ok {
		row[47] = screen["width"]
		row[48] = screen["height"]
		row[49] = screen["density"]
	} else {
		row[47], row[48], row[49] = int16(0), int16(0), decimal.Decimal{}
	}

	// session
	if session, ok := eventContext["session"].(map[string]any); ok {
		row[50] = session["id"]
		row[51] = session["start"]
	} else {
		row[50], row[51] = 0, false
	}

	// timezone
	if timezone, ok := eventContext["timezone"]; ok {
		row[52] = timezone
	} else {
		row[52] = ""
	}

	// userAgent
	if userAgent, ok := eventContext["userAgent"]; ok {
		row[53] = userAgent
	} else {
		row[53] = ""
	}

	// event
	if event, ok := event["event"]; ok {
		row[54] = event
	} else {
		row[54] = ""
	}

	// groupId
	if groupId, ok := event["groupId"]; ok {
		row[55] = groupId
	} else {
		row[55] = ""
	}

	// messageId
	row[56] = event["messageId"]

	// name
	if name, ok := eventContext["name"]; ok {
		row[57] = name
	} else {
		row[57] = ""
	}

	// properties
	if properties, ok := event["properties"]; ok {
		row[58] = properties
	} else {
		row[58] = emptyJSONObject
	}

	// receivedAt
	row[59] = event["receivedAt"]

	// sentAt
	row[60] = event["sentAt"]

	// timestamp
	row[61] = event["timestamp"]

	// traits
	if traits, ok := event["traits"]; ok {
		row[62] = traits
	} else {
		row[62] = emptyJSONObject
	}

	// type
	row[63] = event["type"]

	// userId
	if userId := event["userId"]; userId != nil {
		row[64] = userId
	} else {
		row[64] = ""
	}

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

	for {
		metrics.Increment("EventWriter.flush.for_loop_iterations", 1)
		err := ew.store.warehouse().Merge(ew.close.ctx, eventsMergeTable, rows, nil)
		if err != nil {
			done := ew.close.ctx.Done()
			if _, ok := <-done; ok {
				return
			}
			slog.Error("core/datastore: cannot flush the event queue", "err", err)
			select {
			case <-time.After(time.Duration(rand.IntN(2000)) * time.Millisecond):
			case <-done:
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
