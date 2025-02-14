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
	"time"

	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	meergoMetrics "github.com/meergo/meergo/metrics"
)

var maxQueuedIdentities = 1000
var maxQueuedEventIdentityTime = 200 * time.Millisecond

func (c *Collector) writeIdentity(action *state.Action, identity events.Event) error {

	meergoMetrics.Increment("Collector.writeIdentity.calls", 1)

	a, ok := c.actions.Load(action.ID)
	if !ok {
		return nil
	}
	sa := a.(*actionIdentityWriter)

	// If the action lacks a transformation, write the identity directly to the store.
	if sa.transformer == nil {
		id, ok := identity["userId"].(string)
		// Since there are no properties, do not store anonymous identities.
		if !ok {
			meergoMetrics.Increment("Collector.writeIdentity.user_discarded_as_anonymous_and_without_transformation", 1)
			c.identityAck(action.ID, []string{identity["id"].(string)}, nil)
			return nil
		}
		err := sa.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    identity["anonymousId"].(string),
			Properties:     map[string]any{},
			LastChangeTime: identity["timestamp"].(time.Time),
		}, identity["id"].(string))
		if err == datastore.ErrActionNotExist {
			c.actions.Delete(action.ID)
			return nil
		}
		return err
	}

	var identities []events.Event

	sa.mu.Lock()
	if sa.identities == nil {
		// Set the timer.
		sa.timer = time.AfterFunc(maxQueuedEventIdentityTime, func() {
			sa.mu.Lock()
			if sa.identities == nil {
				sa.mu.Unlock()
				return
			}
			identities := sa.identities
			sa.identities = nil
			sa.timer = nil
			sa.mu.Unlock()
			c.transformAndWriteIdentities(sa, identities)
		})
	}
	sa.identities = append(sa.identities, identity)
	if len(sa.identities) == maxQueuedIdentities {
		identities = sa.identities
		sa.identities = nil
	}
	sa.mu.Unlock()

	// Transform the identities.
	if identities != nil {
		go c.transformAndWriteIdentities(sa, identities)
	}

	return nil
}

func (c *Collector) transformAndWriteIdentities(action *actionIdentityWriter, identities []events.Event) {

	meergoMetrics.Increment("Collector.transformAndWriteIdentities.calls", 1)
	meergoMetrics.Increment("Collector.transformAndWriteIdentities.passed_identities", len(identities))

	records := make([]transformers.Record, len(identities))
	for i, identity := range identities {
		records[i].Properties = identity
	}

	ctx := context.Background()
	err := action.transformer.Transform(ctx, records)
	if err != nil {
		slog.Error("core/events/collector: unexpected error occurred transforming event", "err", err)
		return
	}
	for i, record := range records {
		if err = record.Err; err != nil {
			if _, ok := record.Err.(ValidationError); ok {
				c.metrics.TransformationPassed(action.id, 1)
				c.metrics.OutputValidationFailed(action.id, 1, record.Err.Error())
				continue
			}
			c.metrics.TransformationFailed(action.id, 1, record.Err.Error())
			continue
		}
		c.metrics.TransformationPassed(action.id, 1)
		c.metrics.OutputValidationPassed(action.id, 1)
		identity := identities[i]
		id, ok := identity["userId"].(string)
		// Discard anonymous events with no properties.
		if !ok && len(record.Properties) == 0 {
			meergoMetrics.Increment("Collector.transformAndWriteIdentities.discarded_as_anonymous_and_without_properties", 1)
			continue
		}
		// Write the identity on the data warehouse.
		err = action.writer.Write(datastore.Identity{
			ID:             id,
			AnonymousID:    identity["anonymousId"].(string),
			Properties:     record.Properties,
			LastChangeTime: identity["timestamp"].(time.Time),
		}, identity["id"].(string))
		_ = err // TODO(marco): handle the error
	}

}
