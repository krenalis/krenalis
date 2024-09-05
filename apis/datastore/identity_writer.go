//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/schemas"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"
)

// Identity is an identity
type Identity struct {
	ID             string                 // Identifier of the identity; it is empty for anonymous identities.
	AnonymousID    string                 // AnonymousID of identities received via events.
	Properties     map[string]interface{} // Properties of the user schema.
	LastChangeTime time.Time              // Last change time in UTC.
}

// identityKey represents a key in the _user_identities table.
type identityKey struct {
	action      int
	isAnonymous bool
	identityID  string
}

// BatchIdentityWriter writes user identities into the data warehouse in the
// case when identities are imported in batch.
type BatchIdentityWriter struct {
	store      *Store
	action     int
	connection int
	execution  int
	ack        IdentityWriterAckFunc
	flatter    *flatter
	columns    map[string]warehouses.Column
	rows       []map[string]any
	index      map[identityKey]int
	ackIDs     []string
	purge      bool
	closed     bool
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// If purge needs to be done, it purges all identities of the action for which
// neither the Write method nor the Keep method has been called.
//
// In case an error occurs with the data warehouse, a DataWarehouseError error
// is returned.
//
// If the writer is already closed, it does nothing and returns immediately.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
//
// TODO(Gianluca): if these errors returned from Close seem strange, it's
// because we still need to discuss the issue
// https://github.com/meergo/meergo/issues/1002 and understand precisely what
// model we want to implement for the operations and compatible methods.
func (iw *BatchIdentityWriter) Close(ctx context.Context) error {
	if iw.closed {
		return nil
	}
	ctx, done, err := iw.store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	iw.closed = true
	if iw.rows != nil {
		columns := identitiesMergeColumns(iw.columns)
		err := iw.store.warehouse.MergeIdentities(ctx, columns, iw.rows)
		if err != nil {
			return err
		}
		iw.ack(iw.ackIDs, nil)
	}
	if iw.purge {
		where := warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
			warehouses.NewBaseExpr(warehouses.Column{Name: "__action__", Type: types.Int(32)}, warehouses.OpEqual, iw.action),
			warehouses.NewBaseExpr(warehouses.Column{Name: "__execution__", Type: types.Int(32)}, warehouses.OpNotEqual, iw.execution),
		})
		err := iw.store.warehouse.Delete(ctx, "_user_identities", where)
		if err != nil {
			return err
		}
	}
	return nil
}

// Keep keeps the identity with the identifier id. Use Keep instead of Write
// when there is no need to modify the identity, but to ensure it is not purged
// in case of reload.
func (iw *BatchIdentityWriter) Keep(id string) {
	if iw.closed {
		panic("call Keep on a closed identity writer")
	}
	if !iw.purge {
		return
	}
	key := identityKey{action: iw.action, identityID: id}
	row := map[string]any{
		"$purge":           false,
		"__action__":       key.action,
		"__is_anonymous__": false,
		"__identity_id__":  key.identityID,
		"__connection__":   iw.connection,
		"__execution__":    iw.execution,
	}
	iw.addRow(key, row)
}

// Write writes a user identity. If a valid user schema has been provided, the
// properties must comply with it. It returns immediately, deferring the
// validation of the properties and the actual write operation to a later time.
//
// If an error occurs during validation of the properties, it calls the ack
// function with the value of ackID and the error.
//
// When a batch of identities has been written to the data warehouse, it calls
// the ack function with the ackID of the written identities and a nil error.
//
// It panics if called on a closed writer.
func (iw *BatchIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.closed {
		panic("call Write on a closed identity writer")
	}
	key := identityKey{action: iw.action, identityID: identity.ID}
	row := identity.Properties
	iw.flatter.flat(row, iw.columns)
	row["__action__"] = key.action
	row["__is_anonymous__"] = false
	row["__identity_id__"] = key.identityID
	row["__connection__"] = iw.connection
	row["__last_change_time__"] = identity.LastChangeTime
	row["__execution__"] = iw.execution
	iw.addRow(key, row)
	iw.ackIDs = append(iw.ackIDs, ackID)
	return nil
}

// addRow adds a row to the rows, replacing an existing row with the same key.
func (iw *BatchIdentityWriter) addRow(key identityKey, row map[string]any) {
	if i, ok := iw.index[key]; ok {
		iw.rows[i] = row
		return
	}
	iw.index[key] = len(iw.rows)
	iw.rows = append(iw.rows, row)
}

// EventIdentityWriter writes user identities into the data warehouse, in case
// when identities are imported from events. It deletes the anonymous identities
// when a non-anonymous identity with the same Anonymous ID on the same
// connection is written.
type EventIdentityWriter struct {
	store      *Store
	action     int
	connection int
	ack        IdentityWriterAckFunc
	columns    map[string]warehouses.Column
	rows       []map[string]any
	index      map[identityKey]int
	ackIDs     []string
	closed     bool

	mu      sync.Mutex       // for the 'actions' and 'flatter' fields.
	actions map[int]struct{} // actions of the action's connection. Access using 'mu'. If nil, it means that the action does not exist anymore.
	aligned bool             // indicates if the action's output schema is aligned with the user schema. access using 'mu'.
	flatter *flatter         // access using 'mu'. nil for actions that import identities from events with no transformations.
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// In case an error occurs with the data warehouse, a DataWarehouseError error
// is returned.
//
// If the writer is already closed, it does nothing and returns immediately.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
//
// TODO(Gianluca): if these errors returned from Close seem strange, it's
// because we still need to discuss the issue
// https://github.com/meergo/meergo/issues/1002 and understand precisely what
// model we want to implement for the operations and compatible methods.
func (iw *EventIdentityWriter) Close(ctx context.Context) error {
	if iw.closed {
		return nil
	}
	ctx, done, err := iw.store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	iw.closed = true
	if iw.rows == nil {
		return nil
	}
	columns := identitiesMergeColumns(iw.columns)
	err = iw.store.warehouse.MergeIdentities(ctx, columns, iw.rows)
	if err != nil {
		return err
	}
	iw.ack(iw.ackIDs, nil)
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
// When a batch of event identities has been written to the data warehouse, it
// calls the ack function with the ackID of the written identities and a nil
// error.
//
// If the action of iw does not exist anymore, returns an error.
//
// It panics if called on a closed writer.
func (iw *EventIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.closed {
		panic("call Write on a closed identity writer")
	}

	switch iw.store.Mode() {
	case state.Inspection:
		iw.ack([]string{ackID}, ErrInspectionMode)
		return nil
	case state.Maintenance:
		iw.ack([]string{ackID}, ErrMaintenanceMode)
		return nil
	}

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
		return errors.New("action does not exist anymore")
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
			iw.addRow(key, row)
		}
	}

	var row map[string]any
	if flatter == nil {
		row = map[string]any{}
	} else {
		row = identity.Properties
		flatter.flat(row, iw.columns)
	}
	row["__action__"] = key.action
	row["__is_anonymous__"] = key.isAnonymous
	row["__identity_id__"] = key.identityID
	row["__connection__"] = iw.connection
	if !key.isAnonymous {
		row["__anonymous_ids__"] = []string{identity.AnonymousID}
	}
	row["__last_change_time__"] = identity.LastChangeTime

	iw.addRow(key, row)
	iw.ackIDs = append(iw.ackIDs, ackID)

	return nil
}

// addRow adds a row to the rows, replacing an existing row with the same key.
func (iw *EventIdentityWriter) addRow(key identityKey, row map[string]any) {
	if i, ok := iw.index[key]; ok {
		iw.rows[i] = row
		return
	}
	iw.index[key] = len(iw.rows)
	iw.rows = append(iw.rows, row)
}

// onAddAction is called when an action of the connection of iw's action is
// added.
//
// The notification is propagated by the Store.onAddAction method.
func (iw *EventIdentityWriter) onAddAction(n state.AddAction) {
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

// onSetAction is called when an action of the connection of iw's action is set.
//
// The notification is propagated by the Store.onSetAction method.
func (iw *EventIdentityWriter) onSetAction(n state.SetAction) {
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

// onSetWorkspaceUserSchema is called when the user schema of the workspace of
// the iw's connection is set.
//
// The notification is propagated by the Store.onSetWorkspaceUserSchema method.
func (iw *EventIdentityWriter) onSetWorkspaceUserSchema(_ state.SetWorkspaceUserSchema) {
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

// flatter allows flattening a map[string]any containing user schema properties
// into a map[string]any representing user table columns.
type flatter struct {
	name       string
	column     warehouses.Column
	properties []*flatter
}

// newFlattener returns a new flattener that flattens properties according to
// the given schema and mapping from properties to the respective columns.
func newFlatter(schema types.Type, columnByProperty map[string]warehouses.Column) *flatter {
	flatters := map[string]*flatter{
		"": {properties: []*flatter{}},
	}
	for path, property := range types.Walk(schema) {
		base := ""
		if i := strings.LastIndex(path, "."); i > 0 {
			base = path[:i]
		}
		parent := flatters[base]
		node := &flatter{name: property.Name}
		if property.Type.Kind() == types.ObjectKind {
			node.properties = []*flatter{}
			flatters[path] = node
		} else {
			node.column = columnByProperty[path]
		}
		parent.properties = append(parent.properties, node)

	}
	return flatters[""]
}

// flat flats proprieties and updates the columns argument with the columns in
// properties.
func (f *flatter) flat(properties map[string]any, columns map[string]warehouses.Column) {
	f.flatRec(true, properties, properties, columns)
}

func (f *flatter) flatRec(isRoot bool, root, properties map[string]any, columns map[string]warehouses.Column) {
	for _, ff := range f.properties {
		v, ok := properties[ff.name]
		if !ok {
			continue
		}
		if ff.properties == nil {
			if !isRoot {
				root[ff.column.Name] = v
				delete(root, ff.name)
			}
			if _, ok := columns[ff.column.Name]; !ok {
				columns[ff.column.Name] = ff.column
			}
		} else {
			ff.flatRec(false, root, v.(map[string]any), columns)
		}
	}
}

// identitiesMergeColumns returns the columns to be used during the identities
// merge operation, both when importing in batch and from events.
func identitiesMergeColumns(iwColumns map[string]warehouses.Column) []warehouses.Column {
	columns := make([]warehouses.Column, 7+len(iwColumns))
	columns[0] = warehouses.Column{Name: "__action__", Type: types.Int(32)}
	columns[1] = warehouses.Column{Name: "__is_anonymous__", Type: types.Text()}
	columns[2] = warehouses.Column{Name: "__identity_id__", Type: types.Text()}
	columns[3] = warehouses.Column{Name: "__connection__", Type: types.Int(32)}
	columns[4] = warehouses.Column{Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true}
	columns[5] = warehouses.Column{Name: "__last_change_time__", Type: types.DateTime()}
	columns[6] = warehouses.Column{Name: "__execution__", Type: types.Int(32)}
	i := 7
	for name := range iwColumns {
		columns[i] = iwColumns[name]
		i++
	}
	slices.SortFunc(columns[7:], func(a, b warehouses.Column) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return columns
}
