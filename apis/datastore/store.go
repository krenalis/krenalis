//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chichi/apis/datastore/expr"
	"chichi/apis/datastore/warehouses"
	"chichi/apis/state"
	"chichi/connector/types"
)

type Store struct {
	ds        *Datastore
	workspace int
	warehouse warehouses.Warehouse
	mu        sync.Mutex // for the events field
	events    [][]any
	closed    atomic.Bool
}

// newStore returns a new Store for the workspace ws.
func newStore(ds *Datastore, ws *state.Workspace) (*Store, error) {
	store := &Store{
		ds:        ds,
		workspace: ws.ID,
	}
	var err error
	store.warehouse, err = openWarehouse(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	go func() {
		ticker := time.NewTicker(flushEventsQueueTimeout)
		for {
			select {
			case <-ticker.C:
				store.mu.Lock()
				events := store.events
				store.events = nil
				store.mu.Unlock()
				if events != nil {
					go store.flushEvents(events)
				}
			}
		}
	}()
	return store, nil
}

// AddEvents adds events to the store.
func (store *Store) AddEvents(events [][]any) {
	store.mustBeOpen()
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
}

// DestinationUser returns the external ID of the destination user of the
// action that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	store.mustBeOpen()
	return store.warehouse.DestinationUser(ctx, action, property)
}

// Events returns the events that satisfy the where condition with only the
// given columns, ordered by order if order is not the zero Property, and in
// range [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Events(ctx context.Context, columns []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error) {
	store.mustBeOpen()
	return store.warehouse.Select(ctx, "events", columns, where, order, first, limit)
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) InitWarehouse(ctx context.Context) error {
	store.mustBeOpen()
	return store.warehouse.Init(ctx)
}

// SetDestinationUser sets the destination user relative to the action, with
// the given external user ID and external property.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error {
	store.mustBeOpen()
	return store.warehouse.SetDestinationUser(ctx, connection, externalUserID, externalProperty)
}

// SetIdentity sets the identity id (which may have an anonymous ID) imported
// from the action. fromEvents indicates if the identity has been imported from
// an event or not.
func (store *Store) SetIdentity(ctx context.Context, identity map[string]any, id string, anonID string, action int, fromEvent bool) error {
	store.mustBeOpen()
	return store.warehouse.SetIdentity(ctx, identity, id, anonID, action, fromEvent)
}

// Schemas returns the schemas of users, groups, and events for the relative
// tables. If a table doesn't exist, it won't be included in returned schemas.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Schemas(ctx context.Context) (map[string]types.Type, error) {
	store.mustBeOpen()
	tables, err := store.warehouse.Tables(ctx)
	if err != nil {
		return nil, err
	}
	schemas := make(map[string]types.Type)
	for _, table := range tables {
		switch table.Name {
		case "users", "users_identities", "groups", "groups_identities", "events":
			properties, err := ColumnsToProperties(table.Columns)
			if err != nil {
				return nil, err
			}
			schemas[table.Name] = types.Object(properties)
		}
	}
	return schemas, nil
}

// ResolveSyncUsers resolves and sync the users.
func (store *Store) ResolveSyncUsers(ctx context.Context) error {

	store.mustBeOpen()

	// Retrieve the workspace.
	ws, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil
	}

	// Retrieve the actions identifiers.
	var actionsIdentifiers []warehouses.ActionIdentifiers
	usersIdentities := ws.Schemas["users_identities"]
	for _, action := range store.ds.state.Actions() {
		var columns []types.Property
		if action.Identifiers != nil {
			count := len(action.Identifiers) + len(ws.AnonymousIdentifiers.Priority)
			columns = make([]types.Property, count)
			for i, ident := range append(action.Identifiers, ws.AnonymousIdentifiers.Priority...) {
				var err error
				columns[i], err = PropertyPathToColumn(*usersIdentities, ident)
				if err != nil {
					return err
				}
			}
		}
		actionsIdentifiers = append(actionsIdentifiers, warehouses.ActionIdentifiers{
			Action:             action.ID,
			IdentifiersColumns: columns,
		})
	}

	usersColumns := PropertiesToColumns(ws.Schemas["users"].Properties())

	return store.warehouse.ResolveSyncUsers(ctx, actionsIdentifiers, usersColumns)
}

// Users returns the users that satisfy the where condition with only the given
// properties, ordered by order if order is not the zero Property, and in range
// [first,first+limit] with first >= 0 and 0 < limit <= 1000.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) Users(ctx context.Context, properties []types.Property, where expr.Expr, order types.Property, first, limit int) ([]map[string]any, error) {
	store.mustBeOpen()
	columns := PropertiesToColumns(properties)
	rows, err := store.warehouse.Select(ctx, "users", columns, where, order, first, limit)
	if err != nil {
		return nil, err
	}
	users := make([]map[string]any, len(rows))
	for i, row := range rows {
		users[i], _ = deserializeRowAsMap(properties, row)
	}
	return users, nil
}

// UsersSlice is like Users but returns the users as a slice.
//
// If an error occurs with the data warehouse, it returns a DataWarehouseError
// error.
func (store *Store) UsersSlice(ctx context.Context, properties []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error) {
	store.mustBeOpen()
	columns := PropertiesToColumns(properties)
	rows, err := store.warehouse.Select(ctx, "users", columns, where, order, first, limit)
	if err != nil {
		return nil, err
	}
	users := make([][]any, len(rows))
	for i, row := range rows {
		users[i] = deserializeRowAsSlice(properties, row)
	}
	return users, nil
}

// close closes the store.
// It flushes the events and closes the data warehouse.
// It panics if it has already been called.
func (store *Store) close() error {
	if store.closed.Swap(true) {
		panic("apis/datastore/store already closed")
	}
	store.mu.Lock()
	if len(store.events) > 0 {
		store.flushEvents(store.events)
		store.events = nil
	}
	store.mu.Unlock()
	err := store.warehouse.Close()
	if err != nil {
		return fmt.Errorf("error occurred closing data warehouse: %s", err)
	}
	return err
}

// mustBeOpen panics if store has been closed.
func (store *Store) mustBeOpen() {
	if store.closed.Load() {
		panic("apis/datastore/store is closed")
	}
}
