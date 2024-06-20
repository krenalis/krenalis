//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"context"
	"slices"
	"strings"
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
	ack        IdentityWriterAckFunc
	flatter    *flatter
	columns    map[string]warehouses.Column
	rows       []map[string]any
	ackIDs     []string
	closed     bool
}

// newBatchIdentityWriter returns a new identity writer to write identities for
// the provided action in the case when identities are imported in batch.
func newBatchIdentityWriter(store *Store, action *state.Action, ack IdentityWriterAckFunc) *BatchIdentityWriter {
	connection := action.Connection()
	iw := BatchIdentityWriter{
		store:      store,
		action:     action.ID,
		connection: connection.ID,
		flatter:    newFlatter(action.OutSchema, store.identityColumnByProperty()),
		ack:        ack,
		columns:    map[string]warehouses.Column{},
	}
	return &iw
}

// Close closes the Writer, ensuring the completion of all pending or ongoing
// write operations. In the event of a canceled context, it interrupts ongoing
// writes, discards pending ones, and returns.
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
	iw.rows = append(iw.rows, row)
	iw.ackIDs = append(iw.ackIDs, ackID)
	return nil
}

// EventIdentityWriter writes user identities into the data warehouse, in case
// when identities are imported from events.. It deletes the anonymous
// identities when a non-anonymous identity with the same Anonymous ID on the
// same connection is written.
type EventIdentityWriter struct {
	store             *Store
	action            int
	connection        int
	connectionActions []int // IDs of the actions of the connection.
	ack               IdentityWriterAckFunc
	flatter           *flatter
	columns           map[string]warehouses.Column
	rows              []map[string]any
	ackIDs            []string
	closed            bool
}

// newEventIdentityWriter returns a new identity writer to write identities for
// the provided action, in case when identities are imported from events.
func newEventIdentityWriter(store *Store, action *state.Action, ack IdentityWriterAckFunc) *EventIdentityWriter {
	connection := action.Connection()
	var connectionActions []int
	for _, action := range connection.Actions() {
		connectionActions = append(connectionActions, action.ID)
	}
	iw := EventIdentityWriter{
		store:             store,
		action:            action.ID,
		connection:        connection.ID,
		connectionActions: connectionActions,
		ack:               ack,
		columns:           map[string]warehouses.Column{},
	}
	// An action's OutSchema may be invalid if the action on events has no
	// mapping, so it imports identities without properties. In that case, the
	// flatter should not be initialized.
	if schema := action.OutSchema; schema.Valid() {
		schema := action.OutSchema
		iw.flatter = newFlatter(schema, store.identityColumnByProperty())
	}
	return &iw
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
// When a batch of event identities has been written to the data warehouse, it
// calls the ack function with the ackID of the written identities and a nil
// error.
//
// It panics if called on a closed writer.
func (iw *EventIdentityWriter) Write(identity Identity, ackID string) error {
	if iw.closed {
		panic("call Write on a closed identity writer")
	}
	isAnonymous := identity.ID == ""
	if !isAnonymous {
		// Delete anonymous identities with the same anonymous ID as the
		// incoming non-anonymous identity. The identities to be deleted must be
		// deleted from all actions in the connection, not just from the action
		// from which the identity is being imported.
		for _, action := range iw.connectionActions {
			iw.rows = append(iw.rows, map[string]any{
				"$deleted":             true,
				"__action__":           action,
				"__is_anonymous__":     true,
				"__identity_id__":      identity.AnonymousID,
				"__connection__":       iw.connection,
				"__last_change_time__": identity.LastChangeTime,
			})
		}
	}
	var row map[string]any
	if iw.flatter == nil {
		row = map[string]any{}
	} else {
		row = identity.Properties
		iw.flatter.flat(row, iw.columns)
	}
	row["__action__"] = iw.action
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
	columns := make([]warehouses.Column, 6+len(iwColumns))
	columns[0] = warehouses.Column{Name: "__action__", Type: types.Int(32)}
	columns[1] = warehouses.Column{Name: "__is_anonymous__", Type: types.Text()}
	columns[2] = warehouses.Column{Name: "__identity_id__", Type: types.Text()}
	columns[3] = warehouses.Column{Name: "__connection__", Type: types.Int(32)}
	columns[4] = warehouses.Column{Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true}
	columns[5] = warehouses.Column{Name: "__last_change_time__", Type: types.DateTime()}
	columnsNames := maps.Keys(iwColumns)
	slices.Sort(columnsNames)
	for i, name := range columnsNames {
		columns[i+6] = iwColumns[name]
	}
	return columns
}
