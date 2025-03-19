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
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"
)

var maxQueuedIdentities = 1000
var maxQueuedEventIdentityTime = 200 * time.Millisecond

// identityWriter represents an identity writer for an action.
type identityWriter struct {
	action      int //action identifier
	writer      *datastore.EventIdentityWriter
	mu          sync.Mutex // for transformer, records, and timer.
	transformer *transformers.Transformer
	identities  []events.Event
	timer       *time.Timer
}

// newIdentityWriter returns a new identityWriter for the provided action.
func newIdentityWriter(ds *datastore.Datastore, action *state.Action, provider transformers.FunctionProvider, ack datastore.EventIdentityWriterAckFunc) *identityWriter {
	iw := &identityWriter{action: action.ID}
	ws := action.Connection().Workspace()
	store := ds.Store(ws.ID)
	iw.writer, _ = store.NewEventIdentityWriter(action.ID, ack)
	if t := action.Transformation; t.Mapping != nil || t.Function != nil {
		iw.transformer, _ = transformers.New(action, provider, nil)
	}
	return iw
}

// Close closes iw.
func (iw *identityWriter) Close(ctx context.Context) error {
	if iw.timer != nil {
		iw.timer.Stop()
		iw.timer = nil
	}
	return iw.writer.Close(ctx)
}

// SetTransformer sets the transformer.
// If the transformer is nil, no transformation will be performed.
func (iw *identityWriter) SetTransformer(transformer *transformers.Transformer) {
	iw.mu.Lock()
	iw.transformer = transformer
	iw.mu.Unlock()
}

func (c *Collector) writeIdentity(action *state.Action, identity events.Event) error {

	meergoMetrics.Increment("Collector.writeIdentity.calls", 1)

	w, ok := c.identityWriters.Load(action.ID)
	if !ok {
		return nil
	}
	iw := w.(*identityWriter)

	// If the action lacks a transformation, write the identity directly to the store.
	if iw.transformer == nil {
		id, ok := identity["userId"].(string)
		// Since there are no properties, do not store anonymous identities.
		if !ok {
			meergoMetrics.Increment("Collector.writeIdentity.user_discarded_as_anonymous_and_without_transformation", 1)
			c.identityAck(action.ID, []string{identity["id"].(string)}, nil)
			return nil
		}
		err := iw.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    identity["anonymousId"].(string),
			Properties:     map[string]any{},
			LastChangeTime: identity["timestamp"].(time.Time),
		}, identity["id"].(string))
		if err == datastore.ErrActionNotExist {
			c.identityWriters.Delete(action.ID)
			return nil
		}
		return err
	}

	var identities []events.Event

	iw.mu.Lock()
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
			c.transformAndWriteIdentities(iw, identities)
		})
	}
	iw.identities = append(iw.identities, identity)
	if len(iw.identities) == maxQueuedIdentities {
		identities = iw.identities
		iw.identities = nil
	}
	iw.mu.Unlock()

	// Transform the identities.
	if identities != nil {
		go c.transformAndWriteIdentities(iw, identities)
	}

	return nil
}

func (c *Collector) transformAndWriteIdentities(iw *identityWriter, identities []events.Event) {

	meergoMetrics.Increment("Collector.transformAndWriteIdentities.calls", 1)
	meergoMetrics.Increment("Collector.transformAndWriteIdentities.passed_identities", len(identities))

	records := make([]transformers.Record, len(identities))
	for i, identity := range identities {
		records[i].Properties = identity
	}

	ctx := context.Background()
	err := iw.transformer.Transform(ctx, records)
	if err != nil {
		slog.Error("core/events/collector: unexpected error occurred transforming event", "err", err)
		return
	}
	for i, record := range records {
		if err = record.Err; err != nil {
			if _, ok := record.Err.(ValidationError); ok {
				c.metrics.TransformationPassed(iw.action, 1)
				c.metrics.OutputValidationFailed(iw.action, 1, record.Err.Error())
				continue
			}
			c.metrics.TransformationFailed(iw.action, 1, record.Err.Error())
			continue
		}
		c.metrics.TransformationPassed(iw.action, 1)
		c.metrics.OutputValidationPassed(iw.action, 1)
		identity := identities[i]
		id, ok := identity["userId"].(string)
		// Discard anonymous events with no properties.
		if !ok && len(record.Properties) == 0 {
			meergoMetrics.Increment("Collector.transformAndWriteIdentities.discarded_as_anonymous_and_without_properties", 1)
			continue
		}
		// Write the identity on the data warehouse.
		err = iw.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    identity["anonymousId"].(string),
			Properties:     record.Properties,
			LastChangeTime: identity["timestamp"].(time.Time),
		}, identity["id"].(string))
		_ = err // TODO(marco): handle the error
	}

}
