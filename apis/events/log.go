//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package events

import (
	"bytes"
	"context"
	"encoding/json"
	_log "log"
	"time"

	"chichi/apis/postgres"

	"github.com/segmentio/ksuid"
)

type eventsLog struct {
	ctx context.Context
	db  *postgres.DB
}

func newEventsLog(ctx context.Context, db *postgres.DB) *eventsLog {
	return &eventsLog{ctx: ctx, db: db}
}

func (log *eventsLog) Append(events []*collectedEvent) <-chan error {
	ack := make(chan error, 1)
	go func() {
		var b bytes.Buffer
		enc := json.NewEncoder(&b)
		enc.SetEscapeHTML(false)
		for _, event := range events {
			_ = enc.Encode(event)
			source := b.Bytes()
			_, err := log.db.Exec(log.ctx, "INSERT INTO event_collected (id, source) VALUES ($1, $2)", event.id, source)
			if err != nil {
				ack <- err
				return
			}
			b.Reset()
		}
		ack <- nil
	}()
	return ack
}

// Delivered sets the event, with identifier id, as delivered to the endpoint
// of the given action.
func (log *eventsLog) Delivered(id ksuid.KSUID, action int) {
	now := time.Now().UTC()
	go func() {
		_, err := log.db.Exec(log.ctx, "INSERT INTO event_processed (id, action, timestamp, state)"+
			" VALUES ($1, $2, $3, 'Delivered')", id, action, now)
		if err != nil {
			_log.Printf("cannot set event as delivered (event = %q, action = %d)", id.String(), action)
		}
	}()
}

// TransformationFailed sets the event, with identifier id, as failed due to an
// error during the transformation of the given action.
func (log *eventsLog) TransformationFailed(id ksuid.KSUID, action int, err error) {
	now := time.Now().UTC()
	go func() {
		var errString string
		if err != nil {
			errString = err.Error()
		}
		_log.Printf("[warning] transformation failed when processing event %s with action %d: %q", id, action, errString)
		_, err := log.db.Exec(log.ctx, "INSERT INTO event_processed (id, action, timestamp, state, error)"+
			" VALUES ($1, $2, $3, 'TransformationFailed', $4)", id, action, now, errString)
		if err != nil {
			_log.Printf("cannot log transformation failed (event = %q, action = %d)", id.String(), action)
		}
	}()
}
