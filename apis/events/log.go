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
	"errors"
	"log/slog"
	"sync"
	"time"

	"chichi/apis/postgres"
	"chichi/apis/transformers"

	"github.com/segmentio/ksuid"
)

type eventsLog struct {
	db    *postgres.DB
	close struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		sync.WaitGroup
	}
}

func newEventsLog(db *postgres.DB) *eventsLog {
	log := &eventsLog{db: db}
	log.close.ctx, log.close.cancelCtx = context.WithCancel(context.Background())
	return log
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
			log.close.Add(1)
			_, err := log.db.Exec(log.close.ctx, "INSERT INTO event_collected (id, source) VALUES ($1, $2)", event.id, source)
			log.close.Done()
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

// Close closes the events log.
func (log *eventsLog) Close() {
	log.close.cancelCtx()
	log.close.Wait()
}

// Delivered sets the event, with identifier id, as delivered for the given
// action.
func (log *eventsLog) Delivered(id ksuid.KSUID, action int) {
	now := time.Now().UTC()
	log.close.Add(1)
	go func() {
		defer log.close.Done()
		_, err := log.db.Exec(log.close.ctx, "INSERT INTO event_processed (id, action, timestamp, state)"+
			" VALUES ($1, $2, $3, 'Delivered')", id, action, now)
		if err != nil {
			slog.Error("cannot set event as delivered", "event", id.String(), "action", action)
		}
	}()
}

// TransformationFailed sets the event, with identifier id, as failed due to an
// error during the transformation of the given action.
func (log *eventsLog) TransformationFailed(id ksuid.KSUID, action int, terr error) {
	if terr == nil {
		panic("terr is nil")
	}
	now := time.Now().UTC()
	if err, ok := terr.(transformers.FunctionExecutionError); ok {
		terr = errors.New("an internal error occurred during the transformation")
		slog.Error("transformation failed when processing event", "event", id, "action", action, "err", err)
	}
	log.close.Add(1)
	go func() {
		defer log.close.Done()
		_, err := log.db.Exec(log.close.ctx, "INSERT INTO event_processed (id, action, timestamp, state, error)"+
			" VALUES ($1, $2, $3, 'TransformationFailed', $4)", id, action, now, terr.Error())
		if err != nil {
			slog.Error("cannot log failed transformation", "event", id.String(), "action", action, "err", err, "terr", terr)
		}
	}()
}
