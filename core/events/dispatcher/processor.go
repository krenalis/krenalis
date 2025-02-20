//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package dispatcher

import (
	"context"

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"
)

const pipeSize = 50_000

// ValidationError is the interface implemented by validation errors.
type ValidationError interface {
	error
	PropertyPath() string
}

// processor processes events received from source streams and sent them to
// their data warehouses.
type processor struct {
	db     *db.DB
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
func newProcessor(db *db.DB, st *state.State, opStore events.OperationStore, connectors *connectors.Connectors, provider transformers.Provider, metrics *metrics.Collector) (*processor, error) {

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
				err := transformer.Transform(ctx, records)
				if err != nil {
					processor.metrics.TransformationFailed(action.ID, len(records), err.Error())
					continue
				}
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
			meergoMetrics.Increment("processor.worker.event_request_created", 1)

			processor.operationStore.Done(events.DoneEvent{Action: action.ID, ID: event.properties["id"].(string)})

			processor.events.out <- event

		case <-ctx.Done():
			return
		}
	}

}
