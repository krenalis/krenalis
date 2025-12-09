// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/meergo/meergo/core/internal/datastore"
	"github.com/meergo/meergo/core/internal/events"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	meergoMetrics "github.com/meergo/meergo/tools/metrics"
)

var maxQueuedIdentities = 1000
var maxQueuedEventIdentityTime = 200 * time.Millisecond

// identityWriter represents an identity writer for a pipeline.
type identityWriter struct {
	pipeline    int // pipeline identifier
	writer      *datastore.EventIdentityWriter
	metrics     *metrics.Collector
	mu          sync.Mutex                // for transformer, identities, and timer
	transformer *transformers.Transformer // protected by mu
	identities  []events.Event            // protected by mu
	timer       *time.Timer               // protected by mu
}

// newIdentityWriter returns a new identityWriter for the provided pipeline.
//
// It must be called on a frozen state.
func newIdentityWriter(ds *datastore.Datastore, pipeline *state.Pipeline, provider transformers.FunctionProvider, metrics *metrics.Collector) *identityWriter {
	iw := &identityWriter{
		pipeline: pipeline.ID,
		metrics:  metrics,
	}
	ws := pipeline.Connection().Workspace()
	store := ds.Store(ws.ID)
	iw.writer = store.NewEventIdentityWriter(pipeline.ID, iw.ack)
	if t := pipeline.Transformation; t.Mapping != nil || t.Function != nil {
		iw.transformer, _ = transformers.New(pipeline, provider, nil)
	}
	return iw
}

// Close closes iw.
//
// It must be called on a frozen state.
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

	// If the pipeline lacks a transformation, write the identity directly to the store.
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
func (iw *identityWriter) ack(pipeline int, ids []string, err error) {
	if err != nil {
		iw.metrics.FinalizeFailed(iw.pipeline, len(ids), err.Error())
		return
	}
	iw.metrics.FinalizePassed(pipeline, len(ids))
}

func (iw *identityWriter) transformAndWrite(events []events.Event) {

	meergoMetrics.Increment("Collector.IdentityWriter.transformAndWrite.calls", 1)
	meergoMetrics.Increment("Collector.IdentityWriter.transformAndWrite.passed_identities", len(events))

	records := make([]transformers.Record, len(events))
	for i, identity := range events {
		records[i].Attributes = identity
	}

	iw.mu.Lock()
	transformer := iw.transformer
	iw.mu.Unlock()

	ctx := context.Background()
	err := transformer.Transform(ctx, records)
	if err != nil {
		if err2, ok := err.(transformers.FunctionExecError); ok {
			iw.metrics.TransformationFailed(iw.pipeline, len(records), err2.Error())
		} else {
			iw.metrics.TransformationFailed(iw.pipeline, len(records), "an internal error occurred")
			slog.Error("core/events/collector: unexpected error occurred transforming event", "err", err)
		}
		return
	}
	for i, record := range records {
		if err = record.Err; err != nil {
			switch err.(type) {
			case transformers.RecordTransformationError:
				iw.metrics.TransformationFailed(iw.pipeline, 1, err.Error())
			case transformers.RecordValidationError:
				iw.metrics.TransformationPassed(iw.pipeline, 1)
				iw.metrics.OutputValidationFailed(iw.pipeline, 1, err.Error())
			}
			continue
		}
		iw.metrics.TransformationPassed(iw.pipeline, 1)
		iw.metrics.OutputValidationPassed(iw.pipeline, 1)
		event := events[i]
		id, _ := event["userId"].(string)
		// Write the identity on the data warehouse.
		err = iw.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    event["anonymousId"].(string),
			Attributes:     record.Attributes,
			LastChangeTime: event["timestamp"].(time.Time),
		}, event["messageId"].(string))
		_ = err // TODO(marco): handle the error
	}

}

// writeDirect writes the identity without performing any transformation.
func (iw *identityWriter) writeDirect(event events.Event) error {
	id, _ := event["userId"].(string)
	return iw.writer.Write(datastore.Identity{
		ID:             id,
		AnonymousID:    event["anonymousId"].(string),
		Attributes:     map[string]any{},
		LastChangeTime: event["timestamp"].(time.Time),
	}, event["messageId"].(string))
}
