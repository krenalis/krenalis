// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/streams"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
)

// EventIdentityWriter writes identities into the data warehouse, in case when
// identities are imported from events. It deletes the anonymous identities when
// a non-anonymous identity with the same Anonymous ID on the same connection is
// written.
type EventIdentityWriter struct {
	store      *Store
	pipeline   int
	connection int
	columns    []warehouses.Column
	identities chan<- flusherRow[map[string]any]
	flusher    *flusher[map[string]any]

	mu        sync.Mutex
	pipelines map[int]struct{} // pipelines of the pipeline's connection. Access using 'mu'. If nil, it means that the pipeline does not exist anymore.
	aligned   bool             // indicates if the pipeline's output schema is aligned with the profile schema. access using 'mu'.
	flatter   *flatter         // access using 'mu'. nil for pipelines that import identities from events with no transformations.

	closed atomic.Bool
}

// newEventIdentityWriter returns an identity writer for writing user
// identities, relative to the pipeline, on the data warehouse, in case of
// importing identities from events.
//
// It must be called on a frozen state.
func newEventIdentityWriter(store *Store, pipelineID int) *EventIdentityWriter {

	// Initialize the EventIdentityWriter.
	w := &EventIdentityWriter{
		store:     store,
		pipeline:  pipelineID,
		pipelines: map[int]struct{}{},
	}

	pipeline, _ := store.ds.state.Pipeline(pipelineID)
	connection := pipeline.Connection()
	workspace := connection.Workspace()
	w.connection = connection.ID
	if pipeline.OutSchema.Valid() {
		err := schemas.CheckAlignment(pipeline.OutSchema, workspace.ProfileSchema, nil)
		if err == nil {
			w.aligned = true
			w.flatter = newFlatter(pipeline.OutSchema, store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		w.aligned = true
	}
	for _, p := range connection.Pipelines() {
		w.pipelines[p.ID] = struct{}{}
	}
	store.mu.Lock()
	store.eventIdentityWriters[pipeline.ID] = w
	store.mu.Unlock()

	w.columns = make([]warehouses.Column, 7)
	w.columns[0] = warehouses.Column{Name: "_pipeline", Type: types.Int(32)}
	w.columns[1] = warehouses.Column{Name: "_is_anonymous", Type: types.Boolean()}
	w.columns[2] = warehouses.Column{Name: "_identity_id", Type: types.String()}
	w.columns[3] = warehouses.Column{Name: "_connection", Type: types.Int(32)}
	w.columns[4] = warehouses.Column{Name: "_anonymous_ids", Type: types.Array(types.String()), Nullable: true}
	w.columns[5] = warehouses.Column{Name: "_updated_at", Type: types.DateTime()}
	w.columns[6] = warehouses.Column{Name: "_run", Type: types.Int(32), Nullable: true}
	w.columns = appendColumnsFromProperties(w.columns, pipeline.Transformation.OutPaths, store.profileColumnByProperty())

	// Start the flusher.
	opts := flusherOptions{
		QueueSize:        32768,
		BatchSize:        20000,
		MaxBatchSize:     100000,
		MinFlushInterval: 500 * time.Millisecond,
		MaxFlushLatency:  15 * time.Second,
		IdleFlushDelay:   2 * time.Second,
		RateAlpha:        0.3,
	}
	workspaceID := workspace.ID
	connectionID := connection.ID
	w.flusher = newFlusher(store, opts, func(ctx context.Context, identities []map[string]any) error {
		return store.warehouse().MergeIdentities(ctx, w.columns, identities)
	}, func(err error) {
		slog.Warn("cannot flush event identities to the data warehouse; retrying.", "workspace", workspaceID, "connection", connectionID, "pipeline", pipelineID, "error", err)
	})
	w.identities = w.flusher.Ch()

	return w
}

// Close closes the EventWriter. It panics if it has been already closed.
//
// When Close is called, no other calls to EventIdentityWriter's methods should
// be in progress and no other shall be made.
func (w *EventIdentityWriter) Close(ctx context.Context) error {
	if w.closed.Swap(true) {
		panic("EventIdentityWriter already closed")
	}
	// Close the flusher.
	err := w.flusher.Close(ctx)
	w.store.mu.Lock()
	delete(w.store.eventIdentityWriters, w.pipeline)
	w.store.mu.Unlock()
	return err
}

// Write writes an identity. If a valid profile schema has been provided, the
// attributes must comply with it. It returns immediately, deferring the
// validation of the attributes and the actual write operation to a later time.
//
// If the pipeline of w does not exist anymore, returns an error.
func (w *EventIdentityWriter) Write(ctx context.Context, identity Identity, ack streams.Ack) error {

	key := identityKey{pipeline: w.pipeline}
	if identity.ID == "" {
		key.isAnonymous = true
		key.identityID = identity.AnonymousID
	} else {
		key.identityID = identity.ID
	}

	w.mu.Lock()
	pipelines := w.pipelines
	aligned := w.aligned
	flatter := w.flatter
	w.mu.Unlock()

	if pipelines == nil {
		return ErrPipelineNotExist
	}
	if !aligned {
		return &schemas.Error{Msg: "pipeline output schema is no aligned with the profile schema"}
	}

	if !key.isAnonymous {
		// Delete anonymous identities with the same anonymous ID as the
		// incoming non-anonymous identity. The identities to be deleted must be
		// deleted from all pipelines in the connection, not just from the pipeline
		// from which the identity is being imported.
		for pipeline := range pipelines {
			key := identityKey{
				pipeline:    pipeline,
				isAnonymous: true,
				identityID:  identity.AnonymousID,
			}
			row := map[string]any{
				"$purge":        true,
				"_pipeline":     key.pipeline,
				"_is_anonymous": true,
				"_identity_id":  key.identityID,
				"_connection":   w.connection,
				"_updated_at":   identity.UpdatedAt,
			}
			select {
			case w.identities <- flusherRow[map[string]any]{key: key, row: row}:
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	var row map[string]any
	if flatter == nil {
		row = map[string]any{}
	} else {
		row = identity.Attributes
		flatter.flat(row)
	}
	row["_pipeline"] = key.pipeline
	row["_is_anonymous"] = key.isAnonymous
	row["_identity_id"] = key.identityID
	row["_connection"] = w.connection
	if !key.isAnonymous {
		row["_anonymous_ids"] = []any{identity.AnonymousID}
	}
	row["_updated_at"] = identity.UpdatedAt

	select {
	case w.identities <- flusherRow[map[string]any]{key: key, pipeline: w.pipeline, row: row, ack: ack}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}

}

// onCreatePipeline is called when a pipeline of the connection of iw's pipeline
// is created.
//
// The notification is propagated by the Store.onCreatePipeline method.
func (w *EventIdentityWriter) onCreatePipeline(n state.CreatePipeline) {
	w.mu.Lock()
	if w.pipelines != nil {
		w.pipelines[n.ID] = struct{}{}
	}
	w.mu.Unlock()
}

// onDeleteConnection is called the connection of the iw's pipeline is deleted.
//
// The notification is propagated by the Store.onDeleteConnection method.
func (w *EventIdentityWriter) onDeleteConnection(_ state.DeleteConnection) {
	w.mu.Lock()
	w.pipelines = nil
	w.mu.Unlock()
}

// onDeletePipeline is called when a pipeline of the connection of iw's pipeline
// is deleted.
//
// The notification is propagated by the Store.onDeletePipeline method.
func (w *EventIdentityWriter) onDeletePipeline(n state.DeletePipeline) {
	w.mu.Lock()
	if n.ID == w.pipeline {
		w.pipelines = nil
	} else {
		delete(w.pipelines, n.ID)
	}
	w.mu.Unlock()
}

// onEndAlterProfileSchema is called when the alter of the profile schema of a
// workspace ends.
//
// This notification is propagated by the Store.onEndAlterProfileSchema method.
func (w *EventIdentityWriter) onEndAlterProfileSchema(_ state.EndAlterProfileSchema) {
	pipeline, ok := w.store.ds.state.Pipeline(w.pipeline)
	if !ok {
		return
	}
	var aligned bool
	var flatter *flatter
	if pipeline.OutSchema.Valid() {
		workspace := pipeline.Connection().Workspace()
		err := schemas.CheckAlignment(pipeline.OutSchema, workspace.ProfileSchema, nil)
		if err == nil {
			aligned = true
			flatter = newFlatter(pipeline.OutSchema, w.store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		aligned = true
	}
	w.mu.Lock()
	w.aligned = aligned
	w.flatter = flatter
	w.mu.Unlock()
}

// onUpdatePipeline is called when a pipeline of the connection of iw's pipeline
// is updated.
//
// The notification is propagated by the Store.onUpdatePipeline method.
func (w *EventIdentityWriter) onUpdatePipeline(n state.UpdatePipeline) {
	var aligned bool
	var flatter *flatter
	if n.OutSchema.Valid() {
		workspace, ok := w.store.ds.state.Workspace(w.store.workspace)
		if !ok {
			return
		}
		err := schemas.CheckAlignment(n.OutSchema, workspace.ProfileSchema, nil)
		if err == nil {
			aligned = true
			flatter = newFlatter(n.OutSchema, w.store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		aligned = true
	}
	w.mu.Lock()
	w.aligned = aligned
	w.flatter = flatter
	w.mu.Unlock()
}
