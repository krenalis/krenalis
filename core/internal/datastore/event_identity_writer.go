//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/metrics"
	"github.com/meergo/meergo/types"
)

// EventIdentityWriter writes user identities into the data warehouse, in case
// when identities are imported from events. It deletes the anonymous identities
// when a non-anonymous identity with the same Anonymous ID on the same
// connection is written.
type EventIdentityWriter struct {
	store      *Store
	action     int
	connection int
	ack        EventIdentityWriterAckFunc
	columns    []meergo.Column

	mu      sync.Mutex
	actions map[int]struct{} // actions of the action's connection. Access using 'mu'. If nil, it means that the action does not exist anymore.
	aligned bool             // indicates if the action's output schema is aligned with the user schema. access using 'mu'.
	flatter *flatter         // access using 'mu'. nil for actions that import identities from events with no transformations.
	index   map[identityKey]int
	rows    []map[string]any
	ackIDs  []string
	timer   *time.Timer

	close struct {
		ctx    context.Context
		cancel context.CancelFunc
		atomic.Bool
		sync.WaitGroup
	}
}

// newEventIdentityWriter returns an identity writer for writing user
// identities, relative to the action, on the data warehouse, in case of
// importing identities from events.
//
// It returns an error if an open event identity writer for the provided action
// already exists.
func newEventIdentityWriter(store *Store, actionID int, ack EventIdentityWriterAckFunc) (*EventIdentityWriter, error) {

	// Initialize the EventIdentityWriter.
	iw := &EventIdentityWriter{
		store:   store,
		action:  actionID,
		index:   map[identityKey]int{},
		ack:     ack,
		actions: map[int]struct{}{},
	}
	iw.close.ctx, iw.close.cancel = context.WithCancel(context.Background())

	// Finalize the initialization of the EventIdentityWriter in a frozen state.
	store.ds.state.Freeze()
	action, ok := store.ds.state.Action(actionID)
	if !ok {
		store.ds.state.Unfreeze()
		return nil, errors.New("action does not exist")
	}
	connection := action.Connection()
	iw.connection = connection.ID
	if action.OutSchema.Valid() {
		workspace := connection.Workspace()
		err := schemas.CheckAlignment(action.OutSchema, workspace.UserSchema, nil)
		if err == nil {
			iw.aligned = true
			iw.flatter = newFlatter(action.OutSchema, store.identityColumnByProperty())
		}
	} else {
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		iw.aligned = true
	}
	for _, a := range connection.Actions() {
		iw.actions[a.ID] = struct{}{}
	}
	var err error
	store.mu.Lock()
	if _, ok := store.eventIdentityWriters[action.ID]; ok {
		err = errors.New("event identity writer for action already exists")
	} else {
		store.eventIdentityWriters[action.ID] = iw
	}
	store.mu.Unlock()
	store.ds.state.Unfreeze()

	if err != nil {
		return nil, err
	}

	iw.columns = make([]meergo.Column, 7)
	iw.columns[0] = meergo.Column{Name: "__action__", Type: types.Int(32)}
	iw.columns[1] = meergo.Column{Name: "__is_anonymous__", Type: types.Boolean()}
	iw.columns[2] = meergo.Column{Name: "__identity_id__", Type: types.Text()}
	iw.columns[3] = meergo.Column{Name: "__connection__", Type: types.Int(32)}
	iw.columns[4] = meergo.Column{Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true}
	iw.columns[5] = meergo.Column{Name: "__last_change_time__", Type: types.DateTime()}
	iw.columns[6] = meergo.Column{Name: "__execution__", Type: types.Int(32), Nullable: true}
	iw.columns = appendColumnsFromProperties(iw.columns, action.Transformation.OutPaths, store.userColumnByProperty())

	return iw, nil
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

// Write writes a user identity. If a valid user schema has been provided, the
// properties must comply with it. It returns immediately, deferring the
// validation of the properties and the actual write operation to a later time.
//
// On error, it calls the ack function with:
//   - the ErrInspectionMode error if the data warehouse is in inspection mode.
//   - the ErrMaintenanceMode error if the data warehouse is in maintenance
//     mode.
//   - a *schemas.Error value, if the action output schema is not aligned with
//     the user schema.
//
// If the action of iw does not exist anymore, returns an error.
//
// It panics if called on a closed writer.
func (iw *EventIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.close.Load() {
		panic("call Write on a closed identity writer")
	}

	metrics.Increment("EventIdentityWriter.Write.calls", 1)

	key := identityKey{action: iw.action}
	if identity.ID == "" {
		key.isAnonymous = true
		key.identityID = identity.AnonymousID
	} else {
		key.identityID = identity.ID
	}

	iw.mu.Lock()
	actions := iw.actions
	aligned := iw.aligned
	flatter := iw.flatter
	iw.mu.Unlock()

	if actions == nil {
		return ErrActionNotExist
	}
	if !aligned {
		return &schemas.Error{Msg: "action output schema is no aligned with the user schema"}
	}

	if !key.isAnonymous {
		// Delete anonymous identities with the same anonymous ID as the
		// incoming non-anonymous identity. The identities to be deleted must be
		// deleted from all actions in the connection, not just from the action
		// from which the identity is being imported.
		for action := range actions {
			key := identityKey{
				action:      action,
				isAnonymous: true,
				identityID:  identity.AnonymousID,
			}
			row := map[string]any{
				"$purge":               true,
				"__action__":           key.action,
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
		row = identity.Properties
		flatter.flat(row)
	}
	row["__action__"] = key.action
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
			iw.ack(iw.action, ackIDs, err)
			return
		}
		defer done()
		err = iw.store.warehouse().MergeIdentities(ctx, iw.columns, rows)
		iw.ack(iw.action, ackIDs, err)
	})
}

// onCreateAction is called when an action of the connection of iw's action is
// created.
//
// The notification is propagated by the Store.onCreateAction method.
func (iw *EventIdentityWriter) onCreateAction(n state.CreateAction) {
	iw.mu.Lock()
	if iw.actions != nil {
		iw.actions[n.ID] = struct{}{}
	}
	iw.mu.Unlock()
}

// onDeleteAction is called when an action of the connection of iw's action is
// deleted.
//
// The notification is propagated by the Store.onDeleteAction method.
func (iw *EventIdentityWriter) onDeleteAction(n state.DeleteAction) {
	iw.mu.Lock()
	if n.ID == iw.action {
		iw.actions = nil
	} else {
		delete(iw.actions, n.ID)
	}
	iw.mu.Unlock()
}

// onDeleteConnection is called the connection of the iw's action is deleted.
//
// The notification is propagated by the Store.onDeleteConnection method.
func (iw *EventIdentityWriter) onDeleteConnection(_ state.DeleteConnection) {
	iw.mu.Lock()
	iw.actions = nil
	iw.mu.Unlock()
}

// onEndAlterUserSchema is called when the alter of the user schema of a
// workspace ends.
//
// This notification is propagated by the Store.onEndAlterUserSchema method.
func (iw *EventIdentityWriter) onEndAlterUserSchema(_ state.EndAlterUserSchema) {
	action, ok := iw.store.ds.state.Action(iw.action)
	if !ok {
		return
	}
	var aligned bool
	var flatter *flatter
	if action.OutSchema.Valid() {
		workspace := action.Connection().Workspace()
		err := schemas.CheckAlignment(action.OutSchema, workspace.UserSchema, nil)
		if err == nil {
			aligned = true
			flatter = newFlatter(action.OutSchema, iw.store.identityColumnByProperty())
		}
	} else {
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		aligned = true
	}
	iw.mu.Lock()
	iw.aligned = aligned
	iw.flatter = flatter
	iw.mu.Unlock()
}

// onUpdateAction is called when an action of the connection of iw's action is
// updated.
//
// The notification is propagated by the Store.onUpdateAction method.
func (iw *EventIdentityWriter) onUpdateAction(n state.UpdateAction) {
	var aligned bool
	var flatter *flatter
	if n.OutSchema.Valid() {
		workspace, ok := iw.store.ds.state.Workspace(iw.store.workspace)
		if !ok {
			return
		}
		err := schemas.CheckAlignment(n.OutSchema, workspace.UserSchema, nil)
		if err == nil {
			aligned = true
			flatter = newFlatter(n.OutSchema, iw.store.identityColumnByProperty())
		}
	} else {
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		aligned = true
	}
	iw.mu.Lock()
	iw.aligned = aligned
	iw.flatter = flatter
	iw.mu.Unlock()
}
