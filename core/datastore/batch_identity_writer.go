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
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/types"
)

var ErrActionNotExist = errors.New("action does not exist")

var maxQueuedIdentityRows = 1000
var maxQueuedIdentityTime = 500 * time.Millisecond

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
	columns    []meergo.Column
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
		err := iw.store.warehouse().MergeIdentities(ctx, iw.columns, iw.rows)
		if err != nil {
			return err
		}
		iw.ack(iw.ackIDs, nil)
	}
	if iw.purge {
		where := meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
			meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIs, iw.action),
			meergo.NewBaseExpr(meergo.Column{Name: "__execution__", Type: types.Int(32)}, meergo.OpIsNot, iw.execution),
		})
		err := iw.store.warehouse().Delete(ctx, "_user_identities", where)
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
	iw.flatter.flat(row)
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

// identitiesMergeColumns returns the columns to be used during the identities
// merge operation, both when importing in batch.
func identitiesMergeColumns(iwColumns map[string]meergo.Column) []meergo.Column {
	columns := make([]meergo.Column, 7+len(iwColumns))
	columns[0] = meergo.Column{Name: "__action__", Type: types.Int(32)}
	columns[1] = meergo.Column{Name: "__is_anonymous__", Type: types.Boolean()}
	columns[2] = meergo.Column{Name: "__identity_id__", Type: types.Text()}
	columns[3] = meergo.Column{Name: "__connection__", Type: types.Int(32)}
	columns[4] = meergo.Column{Name: "__anonymous_ids__", Type: types.Array(types.Text()), Nullable: true}
	columns[5] = meergo.Column{Name: "__last_change_time__", Type: types.DateTime()}
	columns[6] = meergo.Column{Name: "__execution__", Type: types.Int(32), Nullable: true}
	i := 7
	for _, column := range iwColumns {
		columns[i] = column
		i++
	}
	slices.SortFunc(columns[7:], func(a, b meergo.Column) int {
		return cmp.Compare(a.Name, b.Name)
	})
	return columns
}
