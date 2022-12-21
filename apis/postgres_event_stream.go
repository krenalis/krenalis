//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"log"
	"time"

	"chichi/apis/postgres"
	"chichi/connector"
	"chichi/connector/ui"
)

// Make sure it implements the EventStreamConnection interface.
var _ connector.EventStreamConnection = &postgresEventStream{}

// newPostgresEventStream returns a new postgresEventStream value implementing
// an event stream on db.
func newPostgresEventStream(ctx context.Context, db *postgres.DB) *postgresEventStream {
	return &postgresEventStream{ctx, db, make(chan struct{})}
}

// postgresEventStream is an event stream implemented on a PostgreSQL database.
type postgresEventStream struct {
	ctx  context.Context
	db   *postgres.DB
	sent chan struct{}
}

// Close closes the stream. Must be called if at least one Send or Receive call
// has been made. It cannot be called concurrently with Send and Receive.
func (s *postgresEventStream) Close() error {
	return s.db.Close()
}

// Receive receives an event from the stream. Callers call the ack function to
// notify that the event has been received. The connector resends the event if
// not acknowledged.
//
// Caller do not modify the event data, even temporarily, and event is not
// retained after the ack function has been called.
func (s *postgresEventStream) Receive() (event []byte, ack func(), err error) {
	for {
		tx, err := s.db.Begin(s.ctx)
		if err != nil {
			return nil, nil, err
		}
		err = tx.QueryRow("DELETE FROM event_stream_queue WHERE timestamp =\n" +
			"\t(SELECT timestamp FROM event_stream_queue ORDER BY timestamp FOR UPDATE SKIP LOCKED LIMIT 1)\n" +
			"RETURNING event").Scan(&event)
		if err != nil {
			if err == sql.ErrNoRows {
				_ = tx.Commit()
				time.Sleep(1) // TODO(marco): implement with distributed notifications
				continue
			}
			_ = tx.Rollback()
			return nil, nil, err
		}
		ack = func() {
			err = tx.Commit()
			if err != nil {
				log.Printf("[warning] cannot delete event from event queue: %s", err)
			}
		}
		return event, ack, nil
	}
}

// Send sends an event to the stream. If ack is not nil, connector calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but event is not retained after the ack
// function has been called.
func (s *postgresEventStream) Send(event []byte, options connector.SendOptions, ack func(err error)) error {
	now := time.Now().UTC()
	go func() {
		_, err := s.db.Exec("INSERT INTO event_stream_queue (timestamp, event) VALUES ($1, $2)", now, event)
		if ack != nil {
			ack(err)
		}
	}()
	return nil
}

// ServeUI always returns an ui.ErrEventNotExist error. It exists only to
// implement the connector.EventStreamConnection interface.
func (s *postgresEventStream) ServeUI(string, []byte) (*ui.Form, *ui.Alert, error) {
	return nil, nil, ui.ErrEventNotExist
}
