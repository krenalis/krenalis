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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/events"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// ErrInspectionMode is returned by Store methods when they cannot execute due
// to the data warehouse being in inspection mode.
var ErrInspectionMode = errors.New("the data warehouse is in inspection mode")

// ErrMaintenanceMode is returned by Store methods when they cannot execute due
// to the data warehouse being in maintenance mode.
var ErrMaintenanceMode = errors.New("the data warehouse is in maintenance mode")

type Store struct {
	ds        *Datastore
	workspace int
	warehouse warehouses.Warehouse
	mode      state.WarehouseMode
	mu        sync.Mutex // for the events field
	events    []map[string]any
	closed    atomic.Bool
	runningIR chan struct{} // prevents concurrent executions of the Workspace Identity Resolution.
}

// newStore returns a new Store for the workspace ws.
func newStore(ds *Datastore, ws *state.Workspace) (*Store, error) {
	store := &Store{
		ds:        ds,
		workspace: ws.ID,
		mode:      ws.Warehouse.Mode,
		runningIR: make(chan struct{}, 1),
	}
	var err error
	store.warehouse, err = openWarehouse(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	if ws.Warehouse.Mode == state.Normal {
		go func() {
			ticker := time.NewTicker(flushEventsQueueTimeout)
			for range ticker.C {
				store.mu.Lock()
				events := store.events
				store.events = nil
				store.mu.Unlock()
				if events != nil {
					go store.flushEvents(events)
				}
			}
		}()
	}
	return store, nil
}

// AlterSchema alters the users schemas by applying the given operations.
//
// operations must contain at least one operation.
//
// If one of the specified operations is not supported by the data warehouse,
// for example if a type is not supported, this method returns a
// UnsupportedSchemaChangeErr error.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) AlterSchema(ctx context.Context, operations []warehouses.AlterSchemaOperation) error {
	if len(operations) == 0 {
		return errors.New("operations cannot be empty")
	}
	store.mustBeOpen()
	return store.warehouse.AlterSchema(ctx, operations)
}

// AlterSchemaQueries returns the queries that would be executed altering the
// "users" (and the "users_identities") schema with the given operations.
//
// operations must contain at least one operation.
//
// If one of the specified operations is not supported by the data warehouse,
// for example if a type is not supported, this method returns a
// UnsupportedSchemaChangeErr error.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) AlterSchemaQueries(ctx context.Context, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	if len(operations) == 0 {
		return nil, errors.New("operations cannot be empty")
	}
	store.mustBeOpen()
	return store.warehouse.AlterSchemaQueries(ctx, operations)
}

// AddEvents adds events to the store.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
func (store *Store) AddEvents(events []map[string]any) error {
	switch store.mode {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	store.mustBeOpen()
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
	return nil
}

// DeleteConnectionIdentities deletes the identities of a connection.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) DeleteConnectionIdentities(ctx context.Context, connection int) error {
	store.mustBeOpen()
	switch store.mode {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	return store.warehouse.DeleteConnectionIdentities(ctx, connection)
}

// DestinationUsers returns the external app identifiers of the destination
// users of the action whose external matching property value matches with the
// given property value. If it cannot be found, then an empty slice and false
// are returned.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) DestinationUsers(ctx context.Context, action int, propertyValue string) ([]string, error) {
	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return nil, ErrMaintenanceMode
	}
	return store.warehouse.DestinationUsers(ctx, action, propertyValue)
}

// DuplicatedDestinationUsers returns the external app identifiers of two users
// on the action which have the same value for the matching property, along with
// true.
//
// If there are no users on the action matching this condition, no external app
// identifiers are returned and the returned boolean is false.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) DuplicatedDestinationUsers(ctx context.Context, action int) (string, string, bool, error) {
	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return "", "", false, ErrMaintenanceMode
	}
	return store.warehouse.DuplicatedDestinationUsers(ctx, action)
}

// DuplicatedUsers returns the GIDs of two users which have the same value for
// the given property, along with true.
// If there are no users matching this condition, no GIDs are returned and the
// returned boolean is false.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) DuplicatedUsers(ctx context.Context, property string) (int, int, bool, error) {
	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return 0, 0, false, ErrMaintenanceMode
	}
	return store.warehouse.DuplicatedUsers(ctx, property)
}

// Events returns an iterator over the results of the query on the 'events'
// table of the data warehouse, ordered from the most recent to the oldest.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) Events(ctx context.Context, query EventsQuery) (Records, error) {

	// TODO(Gianluca): check alignment / do normalization here. See the issue
	// https://github.com/open2b/chichi/issues/728.

	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return nil, ErrMaintenanceMode
	}
	q := queryParams{
		Table:       "events",
		TableSchema: events.WarehouseSchemaWithGID,
		IDColumn:    "gid",
		Properties:  query.Properties,
		Where:       query.Where,
		OrderBy:     "timestamp",
		OrderDesc:   true,
		First:       query.First,
		Limit:       query.Limit,
	}
	records, _, err := store.records(ctx, q)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// EventsQuery represents a query for the Events method.
type EventsQuery struct {

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	// TODO(Gianluca): see the issue
	// https://github.com/open2b/chichi/issues/727, where we revise the "where"
	// expressions and the filters.
	Where expr.Expr

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

type IdentitiesWriter = warehouses.IdentitiesWriter

// IdentitiesWriter returns an IdentitiesWriter for writing user identities with
// the given schema, relative to the connection, on the data warehouse.
// fromEvent indicates if the user identities are imported from an event or not.
// ack is the ack function (see the documentation of IdentitiesWriter for more
// details about it).
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
func (store *Store) IdentitiesWriter(ctx context.Context, schema types.Type, connection int, fromEvent bool, ack warehouses.IdentitiesAckFunc) (IdentitiesWriter, error) {
	store.mustBeOpen()
	switch store.mode {
	case state.Inspection:
		return nil, ErrInspectionMode
	case state.Maintenance:
		return nil, ErrMaintenanceMode
	}
	return store.warehouse.IdentitiesWriter(ctx, schema, connection, fromEvent, ack), nil
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) InitWarehouse(ctx context.Context) error {
	store.mustBeOpen()
	return store.warehouse.Init(ctx)
}

// RunWorkspaceIdentityResolution runs the Workspace Identity Resolution.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
func (store *Store) RunWorkspaceIdentityResolution(ctx context.Context) error {

	switch store.mode {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}

	// Prevent concurrent executions of the Workspace Identity Resolution. This
	// is a workaround for the PostgreSQL error:
	//
	//     duplicate key value violates unique constraint "pg_proc_proname_args_nsp_index" (SQLSTATE 23505)
	//
	// TODO(Gianluca): also take a look at https://github.com/open2b/chichi/issues/354.
	store.runningIR <- struct{}{}
	defer func() {
		<-store.runningIR
	}()

	store.mustBeOpen()

	// Retrieve the workspace.
	ws, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil
	}

	// Retrieve the IDs of the workspace connections.
	wsConnections := store.ds.state.Connections()
	connections := make([]int, len(wsConnections))
	for i, c := range wsConnections {
		connections[i] = c.ID
	}

	// Determine the identifiers properties.
	identifiers := make([]types.Property, len(ws.Identifiers))
	for i, ident := range ws.Identifiers {
		path := strings.Split(ident, ".")
		identifier, err := ws.UsersSchema.PropertyByPath(path)
		if err != nil {
			return err
		}
		if !CanBeIdentifier(identifier.Type) {
			return fmt.Errorf("identifier %q has a not allowed type %v", identifier.Name, identifier.Type)
		}
		identifiers[i] = identifier
	}

	return store.warehouse.RunWorkspaceIdentityResolution(ctx, connections, identifiers, ws.UsersSchema)
}

// SetDestinationUser sets the destination user for an action.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	store.mustBeOpen()
	switch store.mode {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	return store.warehouse.SetDestinationUser(ctx, action, externalUserID, externalProperty)
}

// Users returns an iterator over the results of the query on the 'users' table
// of the data warehouse and an estimated count of the users that would be
// returned if First and Limit were not provided in the query.
//
// usersSchema is the schema of the "users" table.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) Users(ctx context.Context, query UsersQuery, usersSchema types.Type) (Records, int, error) {

	// TODO(Gianluca): check alignment / do normalization here. See the issue
	// https://github.com/open2b/chichi/issues/728.

	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return nil, 0, ErrMaintenanceMode
	}
	q := queryParams{
		Table:       "users",
		TableSchema: usersSchema,
		IDColumn:    "__id__",
		Properties:  query.Properties,
		Where:       query.Where,
		OrderBy:     query.OrderBy,
		OrderDesc:   query.OrderDesc,
		First:       query.First,
		Limit:       query.Limit,
	}
	return store.records(ctx, q)
}

// UsersQuery represents a query for the Users method.
type UsersQuery struct {

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	// TODO(Gianluca): see the issue
	// https://github.com/open2b/chichi/issues/727, where we revise the "where"
	// expressions and the filters.
	Where expr.Expr

	// OrderBy, when provided, is the name of property for which the returned
	// records are ordered.
	OrderBy string

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

// UserIdentities returns an iterator over the results of the query on the
// 'users_identities' table of the data warehouse and an estimated count of the
// user identities that would be returned if First and Limit were not provided
// in the query.
//
// usersIdentitiesSchema is the schema of the "users_identities" table.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) UserIdentities(ctx context.Context, query UsersIdentitiesQuery, usersIdentitiesSchema types.Type) (Records, int, error) {

	// TODO(Gianluca): check alignment / do normalization here. See the issue
	// https://github.com/open2b/chichi/issues/728.

	store.mustBeOpen()
	if store.mode == state.Maintenance {
		return nil, 0, ErrMaintenanceMode
	}
	q := queryParams{
		Table:       "users_identities",
		TableSchema: usersIdentitiesSchema,
		IDColumn:    "__identity_key__",
		Properties:  query.Properties,
		Where:       query.Where,
		OrderBy:     query.OrderBy,
		OrderDesc:   query.OrderDesc,
		First:       query.First,
		Limit:       query.Limit,
	}
	return store.records(ctx, q)
}

// UsersIdentitiesQuery represents a query for the Users method.
type UsersIdentitiesQuery struct {

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	// TODO(Gianluca): see the issue
	// https://github.com/open2b/chichi/issues/727, where we revise the "where"
	// expressions and the filters.
	Where expr.Expr

	// OrderBy, when provided, is the name of property for which the returned
	// records are ordered.
	OrderBy string

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

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

// CanBeIdentifier reports whether a property with type t can be used as
// identifier in the Workspace Identity Resolution.
func CanBeIdentifier(t types.Type) bool {
	switch t.Kind() {
	case types.IntKind,
		types.UintKind,
		types.UUIDKind,
		types.InetKind,
		types.TextKind:
		return true
	case types.DecimalKind:
		return t.Scale() == 0
	default:
		return false
	}
}
