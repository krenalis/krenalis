//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"
)

var maxQueuedIdentities = 1000
var maxQueuedEventIdentityTime = 200 * time.Millisecond

// identityWriter represents an identity writer for an action.
type identityWriter struct {
	action         int //action identifier
	writer         *datastore.EventIdentityWriter
	metrics        *metrics.Collector
	operationStore events.OperationStore
	mu             sync.Mutex                // for transformer, identities, and timer
	transformer    *transformers.Transformer // protected by mu
	identities     []events.Event            // protected by mu
	timer          *time.Timer               // protected by mu
}

// newIdentityWriter returns a new identityWriter for the provided action.
func newIdentityWriter(ds *datastore.Datastore, action *state.Action, provider transformers.FunctionProvider, opStore events.OperationStore, metrics *metrics.Collector) *identityWriter {
	iw := &identityWriter{
		action:         action.ID,
		metrics:        metrics,
		operationStore: opStore,
	}
	ws := action.Connection().Workspace()
	store := ds.Store(ws.ID)
	iw.writer, _ = store.NewEventIdentityWriter(action.ID, iw.ack)
	if t := action.Transformation; t.Mapping != nil || t.Function != nil {
		iw.transformer, _ = transformers.New(action, provider, nil)
	}
	return iw
}

// Close closes iw.
func (iw *identityWriter) Close(ctx context.Context) error {
	iw.mu.Lock()
	if iw.timer != nil {
		iw.timer.Stop()
		iw.timer = nil
	}
	iw.mu.Unlock()
	return iw.writer.Close(ctx)
}

// SetTransformer sets the transformer.
// If the transformer is nil, no transformation will be performed.
func (iw *identityWriter) SetTransformer(transformer *transformers.Transformer) {
	iw.mu.Lock()
	iw.transformer = transformer
	iw.mu.Unlock()
}

// Write writes the identity of the provided event into the data warehouse.
func (iw *identityWriter) Write(event events.Event) error {

	meergoMetrics.Increment("Collector.IdentityWriter.Write.calls", 1)

	iw.mu.Lock()

	// If the action lacks a transformation, write the identity directly to the store.
	if iw.transformer == nil {
		iw.mu.Unlock()
		return iw.writeDirect(event)
	}

	var batch []events.Event

	if iw.identities == nil {
		// Set the timer.
		iw.timer = time.AfterFunc(maxQueuedEventIdentityTime, func() {
			iw.mu.Lock()
			if iw.identities == nil {
				iw.mu.Unlock()
				return
			}
			identities := iw.identities
			iw.identities = nil
			iw.timer = nil
			iw.mu.Unlock()
			iw.transformAndWrite(identities)
		})
	}
	iw.identities = append(iw.identities, event)
	if len(iw.identities) == maxQueuedIdentities {
		batch = iw.identities
		iw.identities = nil
	}
	iw.mu.Unlock()

	// Transform the identities.
	if batch != nil {
		go iw.transformAndWrite(batch)
	}

	return nil
}

// ack acknowledges when identities are written to the data warehouse.
func (iw *identityWriter) ack(action int, ids []string, err error) {
	doneEvents := make([]events.DoneEvent, len(ids))
	for i, id := range ids {
		doneEvents[i].Action = action
		doneEvents[i].ID = id
	}
	iw.operationStore.Done(doneEvents...)
	if err != nil {
		iw.metrics.FinalizeFailed(iw.action, len(ids), err.Error())
		return
	}
	iw.metrics.FinalizePassed(action, len(ids))
}

func (iw *identityWriter) transformAndWrite(events []events.Event) {

	meergoMetrics.Increment("Collector.IdentityWriter.transformAndWrite.calls", 1)
	meergoMetrics.Increment("Collector.IdentityWriter.transformAndWrite.passed_identities", len(events))

	records := make([]transformers.Record, len(events))
	for i, identity := range events {
		records[i].Properties = identity
	}

	iw.mu.Lock()
	transformer := iw.transformer
	iw.mu.Unlock()

	ctx := context.Background()
	err := transformer.Transform(ctx, records)
	if err != nil {
		if err2, ok := err.(transformers.FunctionExecError); ok {
			iw.metrics.TransformationFailed(iw.action, len(records), err2.Error())
		} else {
			iw.metrics.TransformationFailed(iw.action, len(records), "an internal error occurred")
			slog.Error("core/events/collector: unexpected error occurred transforming event", "err", err)
		}
		return
	}
	for i, record := range records {
		if err = record.Err; err != nil {
			switch err.(type) {
			case transformers.RecordTransformationError:
				iw.metrics.TransformationFailed(iw.action, 1, err.Error())
			case transformers.RecordValidationError:
				iw.metrics.TransformationPassed(iw.action, 1)
				iw.metrics.OutputValidationFailed(iw.action, 1, err.Error())
			}
			continue
		}
		iw.metrics.TransformationPassed(iw.action, 1)
		iw.metrics.OutputValidationPassed(iw.action, 1)
		event := events[i]
		id, ok := event["userId"].(string)
		// Discard anonymous events with no properties.
		if !ok && len(record.Properties) == 0 {
			meergoMetrics.Increment("Collector.IdentityWriter.transformAndWrite.discarded_as_anonymous_and_without_properties", 1)
			continue
		}
		// Write the identity on the data warehouse.
		err = iw.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    event["anonymousId"].(string),
			Properties:     record.Properties,
			LastChangeTime: event["timestamp"].(time.Time),
		}, event["id"].(string))
		_ = err // TODO(marco): handle the error
	}

}

// writeDirect writes the identity without performing any transformation.
func (iw *identityWriter) writeDirect(event events.Event) error {
	id, ok := event["userId"].(string)
	// Since there are no properties, do not store anonymous identities.
	if !ok {
		meergoMetrics.Increment("Collector.IdentityWriter.write.user_discarded_as_anonymous_and_without_transformation", 1)
		iw.ack(iw.action, []string{event["id"].(string)}, nil)
		return nil
	}
	err := iw.writer.Write(datastore.Identity{
		ID:             id,
		AnonymousID:    event["anonymousId"].(string),
		Properties:     map[string]any{},
		LastChangeTime: event["timestamp"].(time.Time),
	}, event["id"].(string))
	return err
}
