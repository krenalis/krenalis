//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package dispatcher

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/open2b/chichi/apis/connectors"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/apis/transformers"
)

const pipeSize = 100

// processor processes events received from source streams and sent them to
// their data warehouses.
type processor struct {
	sync.Mutex // for the streams field.
	db         *postgres.DB
	state      *state.State
	events     struct {
		in  chan *dispatchingEvent
		out chan *dispatchingEvent
	}
	connectors          *connectors.Connectors
	transformerProvider transformers.Provider
	close               struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
	}
}

// newProcessor returns a new processor.
func newProcessor(db *postgres.DB, st *state.State, connectors *connectors.Connectors, provider transformers.Provider) (*processor, error) {

	processor := processor{
		db:                  db,
		state:               st,
		connectors:          connectors,
		transformerProvider: provider,
	}
	processor.events.in = make(chan *dispatchingEvent, pipeSize)
	processor.events.out = make(chan *dispatchingEvent, pipeSize)
	processor.close.ctx, processor.close.cancelCtx = context.WithCancel(context.Background())

	// Starts the workers.
	for i := 0; i < 10; i++ {
		go processor.worker()
	}

	return &processor, nil
}

// Close closes the processor.
func (processor *processor) Close() {
	processor.close.cancelCtx()
}

func (processor *processor) worker() {
	for {
		ctx := processor.close.ctx
		select {
		case event := <-processor.events.in:

			// Set the connection directly to avoid the need for acquiring a mutex
			// to retrieve it in the dispatcher.
			connection := event.action.Connection()
			event.connection = connection.ID

			// Transform the event.
			action := event.action
			var extra map[string]any
			if tr := action.Transformation; tr.Mapping != nil || tr.Function != nil {
				transformer := transformers.New(action.InSchema, action.OutSchema, tr, action.ID, processor.transformerProvider, nil)
				var err error
				extra, err = transformer.Transform(ctx, event.ToMap())
				if err != nil {
					processor.setEventWithError(event.Id, action.ID, err)
					continue
				}
			}

			// Make the event request.
			var err error
			app := processor.connectors.App(connection)
			event.request, err = app.EventRequest(ctx, event.action.EventType, event.ToConnectorEvent(), extra, event.action.OutSchema, false)
			if err != nil {
				processor.setEventWithError(event.Id, action.ID, err)
				continue
			}

			// Persist the event request.
			request, err := dumpEventRequest(event.request)
			if err != nil {
				processor.setEventWithError(event.Id, action.ID, err)
				continue
			}
			ctx := context.Background()
			_, err = processor.db.Exec(ctx, "INSERT INTO event_dispatching (action, event, request) VALUES ($1, $2, $3)", event.action.ID, event.Id[:], request)
			if err != nil {
				if postgres.IsDuplicateKeyValue(err) {
					// The event is already present in the database. This happens if it was previously inserted but
					// failed to signal that the event has been processed for this action.
					// There's no need to route it to the dispatcher since the restoration procedure handles it.
					continue
				}
				slog.Error("cannot persist event request", "error", err)
				continue
			}

			processor.events.out <- event

		case <-ctx.Done():
			return
		}
	}

}

func (processor *processor) setEventWithError(id [20]byte, action int, terr error) {
	if terr == nil {
		panic("terr is nil")
	}
	now := time.Now().UTC()
	if err, ok := terr.(transformers.FunctionExecutionError); ok {
		terr = errors.New("an internal error occurred during the transformation")
		slog.Error("transformation failed when processing event", "event", id[:], "action", action, "err", err)
	}
	_, err := processor.db.Exec(processor.close.ctx, "INSERT INTO event_deliveries (id, action, timestamp, state, error)"+
		" VALUES ($1, $2, $3, 'TransformationFailed', $4)", id, action, now, terr.Error())
	if err != nil {
		slog.Error("cannot log failed transformation", "event", hex.EncodeToString(id[:]), "action", action, "err", err, "terr", terr)
	}
}

func dumpEventRequest(req *connectors.EventRequest) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(req.URL)
	b.WriteByte('\n')
	err := req.Header.Write(&b)
	if err != nil {
		return nil, err
	}
	b.WriteByte('\n')
	b.Write(req.Body)
	return b.Bytes(), nil
}
