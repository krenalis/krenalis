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
			case _ = <-ticker.C:
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

	row := make([]any, 64)

	// id
	row[0] = event["id"]

	// connection
	row[1] = event["connection"]

	// anonymousId
	row[2] = event["anonymousId"]

	// category
	if category, ok := event["category"]; ok {
		row[3] = category
	} else {
		row[3] = ""
	}

	eventContext := event["context"].(map[string]any)

	// app
	if app, ok := eventContext["app"].(map[string]any); ok {
		row[4] = app["name"]
		row[5] = app["version"]
		row[6] = app["build"]
		row[7] = app["namespace"]
	} else {
		row[4], row[5], row[6], row[7] = "", "", "", ""
	}

	// browser
	if browser, ok := eventContext["browser"].(map[string]any); ok {
		row[8] = browser["name"]
		row[9] = browser["other"]
		row[10] = browser["version"]
	} else {
		row[8], row[9], row[10] = "", "", ""
	}

	// campaign
	if campaign, ok := eventContext["campaign"].(map[string]any); ok {
		row[11] = campaign["name"]
		row[12] = campaign["source"]
		row[13] = campaign["medium"]
		row[14] = campaign["term"]
		row[15] = campaign["content"]
	} else {
		row[11], row[12], row[13], row[14], row[15] = "", "", "", "", ""
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
	} else {
		row[16], row[17], row[18], row[19], row[20], row[21], row[22], row[23] = "", "", false, "", "", "", "", ""
	}

	// ip
	if ip, ok := eventContext["ip"]; ok {
		row[24] = ip
	} else {
		row[24] = "0.0.0.0"
	}

	// library
	if library, ok := eventContext["library"].(map[string]any); ok {
		row[25] = library["name"]
		row[26] = library["version"]
	} else {
		row[25], row[26] = "", ""
	}

	// locale
	if locale, ok := eventContext["locale"]; ok {
		row[27] = locale
	} else {
		row[27] = ""
	}

	// location
	if location, ok := eventContext["location"].(map[string]any); ok {
		row[28] = location["city"]
		row[29] = location["country"]
		row[30] = location["latitude"]
		row[31] = location["longitude"]
		row[32] = location["speed"]
	} else {
		row[28], row[29], row[30], row[31], row[32] = "", "", 0.0, 0.0, 0.0
	}

	// network
	if network, ok := eventContext["network"].(map[string]any); ok {
		row[33] = network["bluetooth"]
		row[34] = network["carrier"]
		row[35] = network["cellular"]
		row[36] = network["wifi"]
	} else {
		row[33], row[34], row[35], row[36] = false, "", false, false
	}

	// os
	if os, ok := eventContext["os"].(map[string]any); ok {
		row[37] = os["name"]
		row[38] = os["version"]
	} else {
		row[37], row[38] = "", ""
	}

	// page
	if page, ok := eventContext["page"].(map[string]any); ok {
		row[39] = page["path"]
		row[40] = page["referrer"]
		row[41] = page["search"]
		row[42] = page["title"]
		row[43] = page["url"]
	} else {
		row[39], row[40], row[41], row[42], row[43] = "", "", "", "", ""
	}

	// referrer
	if referrer, ok := eventContext["referrer"].(map[string]any); ok {
		row[44] = referrer["name"]
		row[45] = referrer["version"]
	} else {
		row[44], row[45] = "", ""
	}

	// screen
	if screen, ok := eventContext["screen"].(map[string]any); ok {
		row[46] = screen["width"]
		row[47] = screen["height"]
		row[48] = screen["density"]
	} else {
		row[46], row[47], row[48] = int16(0), int16(0), decimal.Decimal{}
	}

	// session
	if session, ok := eventContext["session"].(map[string]any); ok {
		row[49] = session["id"]
		row[50] = session["start"]
	} else {
		row[49], row[50] = 0, false
	}

	// timezone
	if timezone, ok := eventContext["timezone"]; ok {
		row[51] = timezone
	} else {
		row[51] = ""
	}

	// userAgent
	if userAgent, ok := eventContext["userAgent"]; ok {
		row[52] = userAgent
	} else {
		row[52] = ""
	}

	// event
	if event, ok := event["event"]; ok {
		row[53] = event
	} else {
		row[53] = ""
	}

	// groupId
	if groupId, ok := event["groupId"]; ok {
		row[54] = groupId
	} else {
		row[54] = ""
	}

	// messageId
	row[55] = event["messageId"]

	// name
	if name, ok := eventContext["name"]; ok {
		row[56] = name
	} else {
		row[56] = ""
	}

	// properties
	if properties, ok := event["properties"]; ok {
		row[57] = properties
	} else {
		row[57] = emptyJSONObject
	}

	// receivedAt
	row[58] = event["receivedAt"]

	// sentAt
	row[59] = event["sentAt"]

	// timestamp
	row[60] = event["timestamp"]

	// traits
	if traits, ok := event["traits"]; ok {
		row[61] = traits
	} else {
		row[61] = emptyJSONObject
	}

	// type
	row[62] = event["type"]

	// userId
	if userId := event["userId"]; userId != nil {
		row[63] = userId
	} else {
		row[63] = ""
	}

	ew.mu.Lock()
	ew.rows = append(ew.rows, row)
	ew.actions = append(ew.actions, action)
	ew.mu.Unlock()

	return nil
}

func (ew *EventWriter) flush() {

	ew.mu.Lock()
	rows, actions := ew.rows, ew.actions
	ew.rows, ew.actions = nil, nil
	ew.mu.Unlock()

	if rows == nil {
		return
	}

	slog.Info("flush events", "count", len(rows))
	for {
		err := ew.store.warehouse().Merge(ew.close.ctx, eventsMergeTable, rows, nil)
		if err != nil {
			done := ew.close.ctx.Done()
			if _, ok := <-done; ok {
				return
			}
			slog.Error("cannot flush the event queue", "err", err)
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
			ew.ack(events, nil)
		}
		return
	}

}
