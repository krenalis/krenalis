// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package collector

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/krenalis/krenalis/core/internal/datastore"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/core/internal/streams"
	"github.com/krenalis/krenalis/core/internal/transformers"
	"github.com/krenalis/krenalis/tools/prometheus"
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
	events      []streams.Event           // protected by mu
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
	iw.writer = store.NewEventIdentityWriter(pipeline.ID)
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
func (iw *identityWriter) Write(event streams.Event) error {

	prometheus.Increment("Collector.IdentityWriter.Write.calls", 1)

	iw.mu.Lock()

	// If the pipeline lacks a transformation, write the identity directly to the store.
	if iw.transformer == nil {
		iw.mu.Unlock()
		return iw.writeDirect(event)
	}

	if iw.events == nil {
		// Set the timer.
		iw.timer = time.AfterFunc(maxQueuedEventIdentityTime, func() {
			iw.mu.Lock()
			if iw.events == nil {
				iw.mu.Unlock()
				return
			}
			batch := iw.events
			iw.events = nil
			iw.timer = nil
			iw.mu.Unlock()
			iw.transformAndWrite(batch)
		})
	}
	iw.events = append(iw.events, event)
	var batch []streams.Event
	if len(iw.events) == maxQueuedIdentities {
		batch = iw.events
		iw.events = nil
	}
	iw.mu.Unlock()

	// Transform the identities.
	if batch != nil {
		go iw.transformAndWrite(batch)
	}

	return nil
}

func (iw *identityWriter) transformAndWrite(events []streams.Event) {

	prometheus.Increment("Collector.IdentityWriter.transformAndWrite.calls", 1)
	prometheus.Increment("Collector.IdentityWriter.transformAndWrite.passed_identities", len(events))

	records := make([]transformers.Record, len(events))
	for i, event := range events {
		records[i].Attributes = event.Attributes
	}

	iw.mu.Lock()
	transformer := iw.transformer
	iw.mu.Unlock()

	ctx := context.Background()
	err := transformer.Transform(ctx, records)
	if err != nil {
		for _, event := range events {
			event.Ack()
		}
		if err2, ok := err.(transformers.FunctionExecError); ok {
			iw.metrics.TransformationFailed(iw.pipeline, len(records), err2.Error())
		} else {
			iw.metrics.TransformationFailed(iw.pipeline, len(records), "an internal error occurred")
			slog.Error("core/events/collector: unexpected error occurred transforming event", "error", err)
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
			events[i].Ack()
			continue
		}
		iw.metrics.TransformationPassed(iw.pipeline, 1)
		iw.metrics.OutputValidationPassed(iw.pipeline, 1)
		event := events[i]
		id, _ := event.Attributes["userId"].(string)
		// Write the identity on the data warehouse.
		err = iw.writer.Write(ctx, datastore.Identity{
			ID:          id,
			AnonymousID: event.Attributes["anonymousId"].(string),
			Attributes:  record.Attributes,
			UpdatedAt:   event.Attributes["timestamp"].(time.Time),
		}, event.Ack)
		_ = err // TODO(marco): handle the error
	}

}

// writeDirect writes the identity without performing any transformation.
func (iw *identityWriter) writeDirect(event streams.Event) error {
	id, _ := event.Attributes["userId"].(string)
	return iw.writer.Write(context.Background(), datastore.Identity{
		ID:          id,
		AnonymousID: event.Attributes["anonymousId"].(string),
		Attributes:  map[string]any{},
		UpdatedAt:   event.Attributes["timestamp"].(time.Time),
	}, event.Ack)
}
