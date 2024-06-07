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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
)

// ErrInspectionMode is returned by Store methods when they cannot execute due
// to the data warehouse being in inspection mode.
var ErrInspectionMode = errors.New("the data warehouse is in inspection mode")

// ErrMaintenanceMode is returned by Store methods when they cannot execute due
// to the data warehouse being in maintenance mode.
var ErrMaintenanceMode = errors.New("the data warehouse is in maintenance mode")

// Query represents a query on a table of a data warehouse.
type Query struct {

	// table is the table to query.
	table string

	// id is the path of a property whose value is returned in the 'Record.ID'
	// field. The property must have type Int(32) and cannot be nullable. It is
	// meaningful only if the method executing the query returns a Records
	// iterator.
	id string

	// count retrieves the total number of rows that match the filter,
	// irrespective of the first and limit parameters. It is meaningful only if
	// the method has a count return parameter.
	count bool

	// Properties are the paths of the properties to return. It cannot be empty
	// and cannot contain overlapped paths.
	Properties []string

	// Filter, when not nil, filters the records to return.
	Filter *state.Filter

	// OrderBy, when non-empty, is the path of property for which the returned
	// rows are ordered.
	OrderBy string

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many rows should be returned and must be >= 0. If 0,
	// it means that there is no limit.
	Limit int
}

type Store struct {
	ds               *Datastore
	workspace        int
	warehouse        warehouses.Warehouse
	columnByProperty struct {
		mu       sync.Mutex
		user     map[string]warehouses.Column // including meta properties.
		identity map[string]warehouses.Column // including meta properties.
	}
	mu        sync.Mutex // for mode and events fields
	mode      state.WarehouseMode
	events    [][]any
	closed    atomic.Bool
	runningIR chan struct{} // prevents concurrent executions of the Identity Resolution.
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
	store.columnByProperty.user = columnByProperty(ws.UserSchema)
	store.columnByProperty.user["__id__"] = warehouses.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.identity = identityColumnByProperty(store.columnByProperty.user)
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

// AlterSchema alters the user schema.
//
// userSchema is the user schema without meta properties (this parameter is
// useful for obtaining type information and for creating views), while
// operations is the set of operations to apply in order to migrate the current
// schema to userSchema.
//
// If one of the specified operations is not supported by the data warehouse,
// for example if a type is not supported, this method returns a
// UnsupportedSchemaChangeErr error.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If an error occurs with the data warehouse, it returns a
// *DataWarehouseError error.
func (store *Store) AlterSchema(ctx context.Context, userSchema types.Type, operations []warehouses.AlterSchemaOperation) error {
	store.mustBeOpen()
	if store.Mode() == state.Inspection {
		return ErrInspectionMode
	}
	userColumns := propertiesToColumns(types.Properties(userSchema))
	return store.warehouse.AlterSchema(ctx, userColumns, operations)
}

// AlterSchemaQueries returns the queries of a schema altering operation.
//
// userSchema is the user schema without meta properties (this parameter is
// useful for obtaining type information and for creating views), while
// operations is the set of operations to apply in order to migrate the current
// schema to userSchema.
//
// If one of the specified operations is not supported by the data warehouse,
// for example if a type is not supported, this method returns a
// UnsupportedSchemaChangeErr error.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) AlterSchemaQueries(ctx context.Context, userSchema types.Type, operations []warehouses.AlterSchemaOperation) ([]string, error) {
	store.mustBeOpen()
	userColumns := propertiesToColumns(types.Properties(userSchema))
	return store.warehouse.AlterSchemaQueries(ctx, userColumns, operations)
}

// AddEvents adds events to the store.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
func (store *Store) AddEvents(events [][]any) error {
	switch store.Mode() {
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
	switch store.Mode() {
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
	if store.Mode() == state.Maintenance {
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
	if store.Mode() == state.Maintenance {
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
func (store *Store) DuplicatedUsers(ctx context.Context, property string) (uuid.UUID, uuid.UUID, bool, error) {
	store.mustBeOpen()
	if store.Mode() == state.Maintenance {
		return uuid.UUID{}, uuid.UUID{}, false, ErrMaintenanceMode
	}
	column := strings.ReplaceAll(property, ".", "_")
	return store.warehouse.DuplicatedUsers(ctx, column)
}

// Events returns the events according to the provided query.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) Events(ctx context.Context, query Query) ([]map[string]any, error) {
	store.mustBeOpen()
	if store.Mode() == state.Maintenance {
		return nil, ErrMaintenanceMode
	}
	query.table = "events"
	records, _, err := store.query(ctx, query, eventColumnByProperty)
	return records, err
}

// IdentityWriterAckFunc is the function called when a batch of user identities
// have been written to the data warehouse.
type IdentityWriterAckFunc func(ids []string, err error)

// IdentityWriter returns an identity writer for writing user identities with
// the given schema, relative to the connection, on the data warehouse. ack is
// the ack function (see the documentation of IdentityWriter for more details
// about it).
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
//
// It panics if the ack function is nil.
//
// TODO(marco): ack is currently not implemented.
func (store *Store) IdentityWriter(schema types.Type, connection int, ack IdentityWriterAckFunc) (*IdentityWriter, error) {
	if ack == nil {
		panic("nil ack function")
	}
	store.mustBeOpen()
	switch store.Mode() {
	case state.Inspection:
		return nil, ErrInspectionMode
	case state.Maintenance:
		return nil, ErrMaintenanceMode
	}
	return newIdentityWriter(store, connection, schema, ack), nil
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) InitWarehouse(ctx context.Context) error {
	store.mustBeOpen()
	if store.Mode() == state.Inspection {
		return ErrInspectionMode
	}
	return store.warehouse.Init(ctx)
}

// Mode returns the data warehouse mode.
func (store *Store) Mode() state.WarehouseMode {
	store.mu.Lock()
	mode := store.mode
	store.mu.Unlock()
	return mode
}

// RunIdentityResolution runs the Identity Resolution.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error.
func (store *Store) RunIdentityResolution(ctx context.Context) error {

	switch store.Mode() {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}

	// Prevent concurrent executions of the Identity Resolution. This is a
	// workaround for the PostgreSQL error:
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

	// Determine the identifiers columns.
	identifiers := make([]warehouses.Column, len(ws.Identifiers))
	for i, ident := range ws.Identifiers {
		path := strings.Split(ident, ".")
		identifier, err := ws.UserSchema.PropertyByPath(path)
		if err != nil {
			return err
		}
		if !CanBeIdentifier(identifier.Type) {
			return fmt.Errorf("identifier %q has a not allowed type %v", identifier.Name, identifier.Type)
		}
		identifiers[i] = warehouses.Column{
			Name:     strings.Join(path, "_"),
			Type:     identifier.Type,
			Nullable: identifier.Nullable,
		}
	}

	// Determine the user columns.
	userColumns := propertiesToColumns(types.Properties(ws.UserSchema))

	// Determine the primary sources for every user column.
	userPrimarySources := make(map[string]int, len(ws.UserPrimarySources))
	for p, s := range ws.UserPrimarySources {
		c := strings.ReplaceAll(p, ".", "_")
		userPrimarySources[c] = s
	}

	return store.warehouse.RunIdentityResolution(ctx, connections, identifiers, userColumns, userPrimarySources)
}

// SetDestinationUser sets the destination user for an action.
//
// If the data warehouse is in inspection mode, it returns the
// ErrInspectionMode error. If it is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) SetDestinationUser(ctx context.Context, action int, externalUserID, externalProperty string) error {
	store.mustBeOpen()
	switch store.Mode() {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	return store.warehouse.SetDestinationUser(ctx, action, externalUserID, externalProperty)
}

// Users returns the users according to the provided query.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) Users(ctx context.Context, query Query) ([]map[string]any, int, error) {
	store.mustBeOpen()
	if store.Mode() == state.Maintenance {
		return nil, 0, ErrMaintenanceMode
	}
	query.table = "users"
	query.count = true
	return store.query(ctx, query, store.userColumnByProperty())
}

// UserRecords returns an iterator over the users, according to the provided
// query. schema is the expected schema of the provided properties to retrive.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If the schema, which must be valid, does not
// conform to the user schema, it returns a *SchemaError error. If an error
// occurs with the data warehouse, it returns a *DataWarehouseError error.
func (store *Store) UserRecords(ctx context.Context, query Query, schema types.Type) (*Records, error) {
	store.mustBeOpen()
	if store.Mode() == state.Maintenance {
		return nil, ErrMaintenanceMode
	}
	query.table = "users"
	query.id = "__id__"
	return store.records(ctx, query, schema, store.userColumnByProperty())
}

// UserIdentities returns the user identities according to the provided query.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) UserIdentities(ctx context.Context, query Query) ([]map[string]any, int, error) {
	store.mustBeOpen()
	if store.Mode() == state.Maintenance {
		return nil, 0, ErrMaintenanceMode
	}
	query.table = "user_identities"
	query.count = true
	return store.query(ctx, query, store.identityColumnByProperty())
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

// identityColumnByProperty returns the map from properties to columns for the
// identity schema.
func (store *Store) identityColumnByProperty() map[string]warehouses.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.identity
	store.columnByProperty.mu.Unlock()
	return columns
}

// query executes the provided query on the data warehouse and returns an
// iterator over the results and an estimated count of the rows that would be
// returned if First and Limit of query were not provided.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) query(ctx context.Context, query Query, columnByProperty map[string]warehouses.Column) ([]map[string]any, int, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty)

	var where warehouses.Expr
	if query.Filter != nil {
		var err error
		where, err = exprFromFilter(query.Filter, columnByProperty)
		if err != nil {
			return nil, 0, err
		}
	}

	var orderBy warehouses.Column
	var orderDesc bool
	if query.OrderBy != "" {
		var ok bool
		orderBy, ok = columnByProperty[query.OrderBy]
		if !ok {
			return nil, 0, fmt.Errorf("property path %s does not exist", query.OrderBy)
		}
		orderDesc = query.OrderDesc
	}

	rows, count, err := store.warehouse.Query(ctx, warehouses.RowQuery{
		Columns:   columns,
		Table:     query.table,
		Where:     where,
		OrderBy:   orderBy,
		OrderDesc: orderDesc,
		First:     query.First,
		Limit:     query.Limit,
	})
	if err != nil {
		return nil, 0, err
	}

	records := make([]map[string]any, 0)

	defer rows.Close()
	row := make([]any, len(columns))
	values := newScanValues(columns, row, store.warehouse.Normalize)
	for rows.Next() {
		if err := rows.Scan(values...); err != nil {
			return nil, 0, err
		}
		records = append(records, unflat(row))
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}

	// Since the count is an estimate, being counted separately from the actual
	// number of record returned, ensure to not return a value lower than the
	// actually returned number of users.
	count = max(len(records), count)

	return records, count, nil
}

// userColumnByProperty returns the map from properties to columns for the user
// schema.
func (store *Store) userColumnByProperty() map[string]warehouses.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.user
	store.columnByProperty.mu.Unlock()
	return columns
}
