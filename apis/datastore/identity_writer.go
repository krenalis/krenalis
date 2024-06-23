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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"golang.org/x/exp/maps"
)

// Identity is an identity
type Identity struct {
	ID             string                 // Identifier of the identity; it is empty for anonymous identities.
	AnonymousID    string                 // AnonymousID of identities received via events.
	Properties     map[string]interface{} // Properties of the user schema.
	LastChangeTime time.Time              // Last change time in UTC.
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
	ackIDs     []string
	purge      bool
	closed     bool
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// In case of reimports, it purges all identities of the action for which
// neither the Write method nor the Keep method has been called.
//
// In case an error occurs with the data warehouse, a DataWarehouseError error
// is returned.
//
// If the writer is already closed, it does nothing and returns immediately.
func (iw *BatchIdentityWriter) Close(ctx context.Context) error {
	if iw.closed {
		return nil
	}
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
		err := iw.store.warehouse.PurgeIdentities(ctx, iw.action, iw.execution)
		if err != nil {
			return err
		}
	}
	return nil
}

// Keep keeps the identity with the identifier id. Use Keep instead of Write
// when there is no need to modify the identity, but to ensure it is not purged
// in case of reimports.
func (iw *BatchIdentityWriter) Keep(id string) {
	if iw.closed {
		panic("call Keep on a closed identity writer")
	}
	if !iw.purge {
		return
	}
	row := map[string]any{
		"$purge":           false,
		"__action__":       iw.action,
		"__is_anonymous__": false,
		"__connection__":   iw.connection,
		"__identity_id__":  id,
		"__execution__":    iw.execution,
	}
	iw.rows = append(iw.rows, row)
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
	row := identity.Properties
	iw.flatter.flat(row, iw.columns)
	row["__action__"] = iw.action
	row["__is_anonymous__"] = false
	row["__connection__"] = iw.connection
	row["__identity_id__"] = identity.ID
	row["__last_change_time__"] = identity.LastChangeTime
	row["__execution__"] = iw.execution
	iw.rows = append(iw.rows, row)
	iw.ackIDs = append(iw.ackIDs, ackID)
	return nil
}

// EventIdentityWriter writes user identities into the data warehouse, in case
// when identities are imported from events. It deletes the anonymous identities
// when a non-anonymous identity with the same Anonymous ID on the same
// connection is written.
type EventIdentityWriter struct {
	store      *Store
	connection int
	ack        IdentityWriterAckFunc
	columns    map[string]warehouses.Column
	rows       []map[string]any
	ackIDs     []string
	closed     bool
	mu         sync.Mutex       // for the 'action', 'actions' and 'flatter' fields.
	action     int              // access using 'mu'. If 0, it means that the action does not exist anymore.
	actions    map[int]struct{} // actions of the action's connection. Access using 'mu'.
	flatter    *flatter         // access using 'mu'. Nil for actions that import identities from events with no transformations.
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
//
// In case an error occurs with the data warehouse, a DataWarehouseError error
// is returned.
//
// If the writer is already closed, it does nothing and returns immediately.
func (iw *EventIdentityWriter) Close(ctx context.Context) error {
	if iw.closed {
		return nil
	}
	iw.closed = true
	if iw.rows == nil {
		return nil
	}
	columns := identitiesMergeColumns(iw.columns)
	err := iw.store.warehouse.MergeIdentities(ctx, columns, iw.rows)
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
// If an error occurs during validation of the properties, it calls the ack
// function with the value of ackID and the error.
//
// If the action of iw does not exist anymore, returns an error.
//
// When a batch of event identities has been written to the data warehouse, it
// calls the ack function with the ackID of the written identities and a nil
// error.
//
// It panics if called on a closed writer.
func (iw *EventIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.closed {
		panic("call Write on a closed identity writer")
	}

	// Read the action from iw and check if it has been deleted.
	iw.mu.Lock()
	action := iw.action
	iw.mu.Unlock()
	if action == 0 {
		return errors.New("action does not exist anymore")
	}

	isAnonymous := identity.ID == ""
	if !isAnonymous {
		// Delete anonymous identities with the same anonymous ID as the
		// incoming non-anonymous identity. The identities to be deleted must be
		// deleted from all actions in the connection, not just from the action
		// from which the identity is being imported.
		iw.mu.Lock()
		actions := iw.actions
		iw.mu.Unlock()
		for action := range actions {
			iw.rows = append(iw.rows, map[string]any{
				"$purge":               true,
				"__action__":           action,
				"__is_anonymous__":     true,
				"__identity_id__":      identity.AnonymousID,
				"__connection__":       iw.connection,
				"__last_change_time__": identity.LastChangeTime,
			})
		}
	}
	iw.mu.Lock()
	flatter := iw.flatter
	iw.mu.Unlock()
	var row map[string]any
	if flatter == nil {
		row = map[string]any{}
	} else {
		row = identity.Properties
		flatter.flat(row, iw.columns)
	}
	row["__action__"] = action
	row["__is_anonymous__"] = isAnonymous
	row["__connection__"] = iw.connection
	if !isAnonymous {
		row["__anonymous_ids__"] = []string{identity.AnonymousID}
	}
	if isAnonymous {
		row["__identity_id__"] = identity.AnonymousID
	} else {
		row["__identity_id__"] = identity.ID
	}
	row["__last_change_time__"] = identity.LastChangeTime
	iw.rows = append(iw.rows, row)
	iw.ackIDs = append(iw.ackIDs, ackID)
	return nil
}

// onAddAction is called when an action of the connection of iw's action is
// added.
//
// The notification is propagated by the Store.onAddAction method.
func (iw *EventIdentityWriter) onAddAction(n state.AddAction) {
	iw.mu.Lock()
	iw.actions[n.ID] = struct{}{}
	iw.mu.Unlock()
}

// onDeleteAction is called when an action of the connection of iw's action is
// deleted.
//
// The notification is propagated by the Store.onDeleteAction method.
func (iw *EventIdentityWriter) onDeleteAction(n state.DeleteAction) {
	iw.mu.Lock()
	delete(iw.actions, n.ID)
	if n.ID == iw.action {
		iw.action = 0
	}
	iw.mu.Unlock()

}

// onSetAction is called when an action of the connection of iw's action is set.
//
// The notification is propagated by the Store.onSetAction method.
func (iw *EventIdentityWriter) onSetAction(n state.SetAction) {
	identityColumns := iw.store.identityColumnByProperty()
	var flatter *flatter
	if n.OutSchema.Valid() {
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		flatter = newFlatter(n.OutSchema, identityColumns)
	}
	iw.mu.Lock()
	iw.flatter = flatter
	iw.mu.Unlock()
}

// onSetWorkspaceUserSchema is called when the user schema of the workspace of
// the iw's connection is set.
//
// The notification is propagated by the Store.onSetWorkspaceUserSchema method.
func (iw *EventIdentityWriter) onSetWorkspaceUserSchema(_ state.SetWorkspaceUserSchema) {
	iw.mu.Lock()
	actionID := iw.action
	iw.mu.Unlock()
	if actionID == 0 {
		return
	}
	action, _ := iw.store.ds.state.Action(actionID)
	identityColumns := iw.store.identityColumnByProperty()
	var flatter *flatter
	if action.OutSchema.Valid() {
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		flatter = newFlatter(action.OutSchema, identityColumns)
	}
	iw.mu.Lock()
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
	columnsNames := maps.Keys(iwColumns)
	slices.Sort(columnsNames)
	for i, name := range columnsNames {
		columns[i+7] = iwColumns[name]
	}
	return columns
}
