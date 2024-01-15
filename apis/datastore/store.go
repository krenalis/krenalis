//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"context"
	"errors"
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
	events    []map[string]any
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
func (store *Store) AddEvents(events []map[string]any) {
	store.mustBeOpen()
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
}

// DestinationUser returns the external ID of the destination user of the
// action that matches with the corresponding property. If it cannot be
// found, then the empty string and false are returned.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) DestinationUser(ctx context.Context, action int, property string) (string, bool, error) {
	store.mustBeOpen()
	return store.warehouse.DestinationUser(ctx, action, property)
}

// Events returns an iterator over the results of the query on the 'events'
// table of the data warehouse, ordered from the most recent to the oldest.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error. If the schema specified in the query is not conform to the schema of
// the 'events' table, it returns a *SchemaError error.
func (store *Store) Events(ctx context.Context, query EventsQuery) (Records, error) {
	store.mustBeOpen()
	records, _, err := store.warehouse.Records(ctx, warehouses.RecordsQuery{
		Table:      "events",
		Schema:     query.Schema,
		Properties: query.Properties,
		ID:         types.Property{Name: "gid", Type: types.Int(32)},
		Where:      query.Where,
		OrderBy:    types.Property{Name: "timestamp", Type: types.DateTime()},
		OrderDesc:  true,
		First:      query.First,
		Limit:      query.Limit,
	})
	return records, err
}

// EventsQuery represents a query for the Events method.
type EventsQuery struct {

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	Where expr.Expr

	// Schema contains the types of the properties in Properties and Where.
	Schema types.Type

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

type IdentitiesWriter = warehouses.IdentitiesWriter

// IdentitiesWriter returns an IdentitiesWriter for writing user identities with
// the given schema, relative to the action, on the data warehouse.
// fromEvent indicates if the user identities are imported from an event or not.
// ack is the ack function (see the documentation of IdentitiesWriter for more
// details about it).
// If the schema specified is not conform to the schema of the table
// 'users_identities' in the data warehouse, calls to the method 'Write' of the
// returned 'IdentitiesWriter' return a *SchemaError error.
func (store *Store) IdentitiesWriter(ctx context.Context, schema types.Type, action int, fromEvent bool, ack warehouses.IdentitiesAckFunc) IdentitiesWriter {
	store.mustBeOpen()
	return store.warehouse.IdentitiesWriter(ctx, schema, action, fromEvent, ack)
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) InitWarehouse(ctx context.Context) error {
	store.mustBeOpen()
	return store.warehouse.Init(ctx)
}

// SetDestinationUser sets the destination user relative to the action, with
// the given external user ID and external property.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	store.mustBeOpen()
	return store.warehouse.SetDestinationUser(ctx, action, externalUserID, externalProperty)
}

// Schemas returns the schemas of users, groups, and events for the relative
// tables. If a table doesn't exist, it won't be included in returned schemas.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
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
			schemas[table.Name] = table.Schema
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

	// Retrieve the workspace actions and simply return if there are none.
	wsActions := store.ds.state.Actions()
	if len(wsActions) == 0 {
		return nil
	}
	var actions []int
	for _, action := range wsActions {
		actions = append(actions, action.ID)
	}

	// TODO(Gianluca): should the users / users_identities schema be handled by
	// Chichi, or internally by the data warehouse? See the issue
	// https://github.com/open2b/chichi/issues/392.
	schemas, err := store.Schemas(ctx)
	if err != nil {
		return err
	}

	// Determine the identifiers properties.
	usersIdentities, ok := schemas["users_identities"]
	if !ok {
		return errors.New("missing 'users_identities' schema")
	}
	count := len(ws.Identifiers) + len(ws.AnonymousIdentifiers.Priority)
	identifiers := make([]types.Property, count)
	for i, ident := range append(ws.Identifiers, ws.AnonymousIdentifiers.Priority...) {
		identifiers[i], ok = usersIdentities.Property(ident)
		if !ok {
			return fmt.Errorf("identifier %q not found within 'users_identities' schema", ident)
		}
	}

	// Take the 'users' schema.
	usersSchema, ok := schemas["users"]
	if !ok {
		return errors.New("missing 'users' schema")
	}

	return store.warehouse.ResolveSyncUsers(ctx, actions, identifiers, usersSchema)
}

type Records = warehouses.Records

// Users returns an iterator over the results of the query on the 'users' table
// of the data warehouse and an estimated count of the users that would be
// returned if First and Limit were not provided in the query.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error. If the schema specified in the query is not conform to the schema of
// the 'users' table, it returns a *SchemaError error.
func (store *Store) Users(ctx context.Context, query UsersQuery) (Records, int, error) {
	store.mustBeOpen()
	records, count, err := store.warehouse.Records(ctx, warehouses.RecordsQuery{
		Table:      "users",
		Schema:     query.Schema,
		Properties: query.Properties,
		ID:         types.Property{Name: "Id", Type: types.Int(32)},
		Where:      query.Where,
		OrderBy:    query.OrderBy,
		OrderDesc:  query.OrderDesc,
		First:      query.First,
		Limit:      query.Limit,
	})
	return records, count, err
}

// UsersQuery represents a query for the Users method.
type UsersQuery struct {

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	Where expr.Expr

	// OrderBy, when provided, is the property for which the returned records
	// are ordered.
	OrderBy types.Property

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// Schema contains the types of the properties in Properties and Where.
	Schema types.Type

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
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
