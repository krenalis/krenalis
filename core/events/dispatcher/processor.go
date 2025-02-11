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

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/postgres"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
)

const pipeSize = 100

// ValidationError is the interface implemented by validation errors.
type ValidationError interface {
	error
	PropertyPath() string
}

// processor processes events received from source streams and sent them to
// their data warehouses.
type processor struct {
	db     *postgres.DB
	state  *state.State
	events struct {
		in  chan *dispatchingEvent
		out chan *dispatchingEvent
	}
	connectors          *connectors.Connectors
	transformerProvider transformers.Provider
	operationStore      events.OperationStore
	metrics             *metrics.Collector
	close               struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
	}
}

// newProcessor returns a new processor.
func newProcessor(db *postgres.DB, st *state.State, opStore events.OperationStore, connectors *connectors.Connectors, provider transformers.Provider, metrics *metrics.Collector) (*processor, error) {

	processor := processor{
		db:                  db,
		state:               st,
		connectors:          connectors,
		transformerProvider: provider,
		operationStore:      opStore,
		metrics:             metrics,
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

			// Transform the event.
			action := event.action
			var properties map[string]any
			if t := action.Transformation; t.Mapping != nil || t.Function != nil {
				transformer, _ := transformers.New(action, processor.transformerProvider, nil)
				records := []transformers.Record{{Properties: event.properties}}
				_ = transformer.Transform(ctx, records)
				if err := records[0].Err; err != nil {
					if _, ok := err.(ValidationError); ok {
						processor.metrics.TransformationPassed(action.ID, 1)
						processor.metrics.OutputValidationFailed(action.ID, 1, err.Error())
						continue
					}
					processor.metrics.TransformationFailed(action.ID, 1, err.Error())
					continue
				}
				processor.metrics.TransformationPassed(action.ID, 1)
				processor.metrics.OutputValidationPassed(action.ID, 1)
				properties = records[0].Properties
			}

			// Make the event request.
			app := processor.connectors.App(event.action.Connection())
			var err error
			event.request, err = app.EventRequest(ctx, events.NewConnectorEvent(event.properties), event.action.EventType, event.action.OutSchema, properties, false)
			if err != nil {
				processor.metrics.FinalizeFailed(action.ID, 1, err.Error())
				continue
			}

			// Persist the event request.
			request, err := dumpEventRequest(event.request)
			if err != nil {
				processor.metrics.FinalizeFailed(action.ID, 1, err.Error())
				continue
			}
			ctx := context.Background()
			_, err = processor.db.Exec(ctx, "INSERT INTO event_dispatching (action, event, request) VALUES ($1, $2, $3)", event.action.ID, event.id, request)
			if err != nil {
				if postgres.IsDuplicateKeyValue(err) {
					// The event is already present in the database. This happens if it was previously inserted but
					// failed to signal that the event has been processed for this action.
					// There's no need to route it to the dispatcher since the restoration procedure handles it.
					processor.operationStore.Done(events.DoneEvent{Action: action.ID, ID: event.properties["id"].(string)})
					continue
				}
				processor.metrics.FinalizeFailed(action.ID, 1, err.Error())
				continue
			}

			processor.operationStore.Done(events.DoneEvent{Action: action.ID, ID: event.properties["id"].(string)})

			processor.events.out <- event

		case <-ctx.Done():
			return
		}
	}

}

func dumpEventRequest(req *meergo.EventRequest) ([]byte, error) {
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
