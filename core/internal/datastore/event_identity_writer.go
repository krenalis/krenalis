// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/types"
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
	ack        EventIdentityWriterAckFunc
	columns    []warehouses.Column

	mu        sync.Mutex
	pipelines map[int]struct{} // pipelines of the pipeline's connection. Access using 'mu'. If nil, it means that the pipeline does not exist anymore.
	aligned   bool             // indicates if the pipeline's output schema is aligned with the profile schema. access using 'mu'.
	flatter   *flatter         // access using 'mu'. nil for pipelines that import identities from events with no transformations.
	index     map[identityKey]int
	rows      []map[string]any
	ackIDs    []string
	timer     *time.Timer

	close struct {
		ctx    context.Context
		cancel context.CancelFunc
		atomic.Bool
		sync.WaitGroup
	}
}

// newEventIdentityWriter returns an identity writer for writing user
// identities, relative to the pipeline, on the data warehouse, in case of
// importing identities from events.
//
// It must be called on a frozen state.
func newEventIdentityWriter(store *Store, pipelineID int, ack EventIdentityWriterAckFunc) *EventIdentityWriter {

	// Initialize the EventIdentityWriter.
	iw := &EventIdentityWriter{
		store:     store,
		pipeline:  pipelineID,
		index:     map[identityKey]int{},
		ack:       ack,
		pipelines: map[int]struct{}{},
	}

	pipeline, _ := store.ds.state.Pipeline(pipelineID)
	connection := pipeline.Connection()
	iw.connection = connection.ID
	if pipeline.OutSchema.Valid() {
		workspace := connection.Workspace()
		err := schemas.CheckAlignment(pipeline.OutSchema, workspace.ProfileSchema, nil)
		if err == nil {
			iw.aligned = true
			iw.flatter = newFlatter(pipeline.OutSchema, store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		iw.aligned = true
	}
	for _, p := range connection.Pipelines() {
		iw.pipelines[p.ID] = struct{}{}
	}
	store.mu.Lock()
	store.eventIdentityWriters[pipeline.ID] = iw
	store.mu.Unlock()

	iw.columns = make([]warehouses.Column, 7)
	iw.columns[0] = warehouses.Column{Name: "__pipeline__", Type: types.Int(32)}
	iw.columns[1] = warehouses.Column{Name: "__is_anonymous__", Type: types.Boolean()}
	iw.columns[2] = warehouses.Column{Name: "__identity_id__", Type: types.Text()}
	iw.columns[3] = warehouses.Column{Name: "__connection__", Type: types.Int(32)}
	iw.columns[4] = warehouses.Column{Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true}
	iw.columns[5] = warehouses.Column{Name: "__last_change_time__", Type: types.DateTime()}
	iw.columns[6] = warehouses.Column{Name: "__execution__", Type: types.Int(32), Nullable: true}
	iw.columns = appendColumnsFromProperties(iw.columns, pipeline.Transformation.OutPaths, store.profileColumnByProperty())

	iw.close.ctx, iw.close.cancel = context.WithCancel(context.Background())

	return iw
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// If the writer is already closed, it does nothing and returns immediately.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a
// *datastore.UnavailableError error.
//
// TODO(Gianluca): if these errors returned from Close seem strange, it's
// because we still need to discuss the issue
// https://github.com/meergo/meergo/issues/1224 and understand precisely what
// model we want to implement for the operations and compatible methods.
//
// It must be called on a frozen state.
func (iw *EventIdentityWriter) Close(ctx context.Context) error {
	if iw.close.Load() {
		return nil
	}
	ctx, done, err := iw.store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	// Mark as closed and return if it was already closed in the meantime.
	if iw.close.Swap(true) {
		return nil
	}
	delete(iw.store.eventIdentityWriters, iw.pipeline)
	// Cancel the flushes if the context is cancelled.
	stop := context.AfterFunc(ctx, func() { iw.close.cancel() })
	defer stop()
	// Wait for the flushes and method calls to terminate.
	iw.close.Wait()
	// Perform a final flush.
	iw.mu.Lock()
	iw.flush()
	iw.mu.Unlock()
	iw.close.Wait()
	return nil
}

// Write writes an identity. If a valid profile schema has been provided, the
// attributes must comply with it. It returns immediately, deferring the
// validation of the attributes and the actual write operation to a later time.
//
// On error, it calls the ack function with:
//   - the ErrInspectionMode error if the data warehouse is in inspection mode.
//   - the ErrMaintenanceMode error if the data warehouse is in maintenance
//     mode.
//   - a *schemas.Error value, if the pipeline output schema is not aligned with
//     the profile schema.
//
// If the pipeline of iw does not exist anymore, returns an error.
//
// It panics if called on a closed writer.
func (iw *EventIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.close.Load() {
		panic("call Write on a closed identity writer")
	}

	metrics.Increment("EventIdentityWriter.Write.calls", 1)

	key := identityKey{pipeline: iw.pipeline}
	if identity.ID == "" {
		key.isAnonymous = true
		key.identityID = identity.AnonymousID
	} else {
		key.identityID = identity.ID
	}

	iw.mu.Lock()
	pipelines := iw.pipelines
	aligned := iw.aligned
	flatter := iw.flatter
	iw.mu.Unlock()

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
				"$purge":               true,
				"__pipeline__":         key.pipeline,
				"__is_anonymous__":     true,
				"__identity_id__":      key.identityID,
				"__connection__":       iw.connection,
				"__last_change_time__": identity.LastChangeTime,
			}
			iw.appendRow(key, row, "")
		}
	}

	var row map[string]any
	if flatter == nil {
		row = map[string]any{}
	} else {
		row = identity.Attributes
		flatter.flat(row)
	}
	row["__pipeline__"] = key.pipeline
	row["__is_anonymous__"] = key.isAnonymous
	row["__identity_id__"] = key.identityID
	row["__connection__"] = iw.connection
	if !key.isAnonymous {
		row["__anonymous_ids__"] = []any{identity.AnonymousID}
	}
	row["__last_change_time__"] = identity.LastChangeTime

	iw.appendRow(key, row, ackID)

	return nil
}

// appendRow appends a row to the rows or replaces an existing row with the same key.
func (iw *EventIdentityWriter) appendRow(key identityKey, row map[string]any, ackID string) {
	iw.mu.Lock()
	if ackID != "" {
		iw.ackIDs = append(iw.ackIDs, ackID)
	}
	// If a row with the same key already exists, update that row rather than adding a duplicate.
	if i, ok := iw.index[key]; ok {
		iw.rows[i] = row
		iw.mu.Unlock()
		return
	}
	if len(iw.rows) == 0 {
		iw.timer = time.AfterFunc(maxQueuedIdentityTime, func() {
			iw.mu.Lock()
			iw.flush()
			iw.mu.Unlock()
		})
	}
	iw.index[key] = len(iw.rows)
	iw.rows = append(iw.rows, row)
	if len(iw.rows) == maxQueuedIdentityRows {
		iw.flush()
	}
	iw.mu.Unlock()
}

// flush flushes the rows, if any, into the data warehouse.
// It must be called while holding the iw.mu mutex.
func (iw *EventIdentityWriter) flush() {
	metrics.Increment("EventIdentityWriter.flush.calls", 1)
	if iw.rows == nil {
		return
	}
	rows := iw.rows
	iw.rows = nil
	ackIDs := iw.ackIDs
	iw.ackIDs = nil
	clear(iw.index)
	iw.timer = nil
	iw.close.Go(func() {
		ctx, done, err := iw.store.mc.StartOperation(iw.close.ctx, normalMode)
		if err != nil {
			// Warehouse mode is not normal: discard identities.
			iw.ack(iw.pipeline, ackIDs, err)
			return
		}
		defer done()
		err = iw.store.warehouse().MergeIdentities(ctx, iw.columns, rows)
		iw.ack(iw.pipeline, ackIDs, err)
	})
}

// onCreatePipeline is called when a pipeline of the connection of iw's pipeline
// is created.
//
// The notification is propagated by the Store.onCreatePipeline method.
func (iw *EventIdentityWriter) onCreatePipeline(n state.CreatePipeline) {
	iw.mu.Lock()
	if iw.pipelines != nil {
		iw.pipelines[n.ID] = struct{}{}
	}
	iw.mu.Unlock()
}

// onDeleteConnection is called the connection of the iw's pipeline is deleted.
//
// The notification is propagated by the Store.onDeleteConnection method.
func (iw *EventIdentityWriter) onDeleteConnection(_ state.DeleteConnection) {
	iw.mu.Lock()
	iw.pipelines = nil
	iw.mu.Unlock()
}

// onDeletePipeline is called when a pipeline of the connection of iw's pipeline
// is deleted.
//
// The notification is propagated by the Store.onDeletePipeline method.
func (iw *EventIdentityWriter) onDeletePipeline(n state.DeletePipeline) {
	iw.mu.Lock()
	if n.ID == iw.pipeline {
		iw.pipelines = nil
	} else {
		delete(iw.pipelines, n.ID)
	}
	iw.mu.Unlock()
}

// onEndAlterProfileSchema is called when the alter of the profile schema of a
// workspace ends.
//
// This notification is propagated by the Store.onEndAlterProfileSchema method.
func (iw *EventIdentityWriter) onEndAlterProfileSchema(_ state.EndAlterProfileSchema) {
	pipeline, ok := iw.store.ds.state.Pipeline(iw.pipeline)
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
			flatter = newFlatter(pipeline.OutSchema, iw.store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		aligned = true
	}
	iw.mu.Lock()
	iw.aligned = aligned
	iw.flatter = flatter
	iw.mu.Unlock()
}

// onUpdatePipeline is called when a pipeline of the connection of iw's pipeline
// is updated.
//
// The notification is propagated by the Store.onUpdatePipeline method.
func (iw *EventIdentityWriter) onUpdatePipeline(n state.UpdatePipeline) {
	var aligned bool
	var flatter *flatter
	if n.OutSchema.Valid() {
		workspace, ok := iw.store.ds.state.Workspace(iw.store.workspace)
		if !ok {
			return
		}
		err := schemas.CheckAlignment(n.OutSchema, workspace.ProfileSchema, nil)
		if err == nil {
			aligned = true
			flatter = newFlatter(n.OutSchema, iw.store.identityColumnByProperty())
		}
	} else {
		// The pipeline's out schema is invalid when importing identities from
		// events without any transformation in the pipeline.
		aligned = true
	}
	iw.mu.Lock()
	iw.aligned = aligned
	iw.flatter = flatter
	iw.mu.Unlock()
}
