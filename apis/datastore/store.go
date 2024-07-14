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

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

// ErrInspectionMode is returned by Store methods when they cannot execute due
// to the data warehouse being in inspection mode.
var ErrInspectionMode = errors.New("the data warehouse is in inspection mode")

// ErrMaintenanceMode is returned by Store methods when they cannot execute due
// to the data warehouse being in maintenance mode.
var ErrMaintenanceMode = errors.New("the data warehouse is in maintenance mode")

// IdentityWriterAckFunc is the function called when a batch of user identities
// have been written to the data warehouse.
type IdentityWriterAckFunc func(ids []string, err error)

type Store struct {
	ds               *Datastore
	workspace        int
	warehouse        warehouses.Warehouse
	columnByProperty struct {
		mu       sync.Mutex
		user     map[string]warehouses.Column // including meta properties.
		identity map[string]warehouses.Column // including meta properties.
	}
	closed               atomic.Bool
	runningIR            chan struct{} // prevents concurrent executions of the Identity Resolution.
	mu                   sync.Mutex    // for 'mode', 'events' and 'eventIdentityWriters' fields
	mode                 state.WarehouseMode
	events               [][]any
	eventIdentityWriters map[int]*EventIdentityWriter // action -> *EventIdentityWriter
}

// newStore returns a new Store for the workspace ws.
// It must be called when the state is frozen.
func newStore(ds *Datastore, ws *state.Workspace) (*Store, error) {
	store := &Store{
		ds:                   ds,
		workspace:            ws.ID,
		mode:                 ws.Warehouse.Mode,
		runningIR:            make(chan struct{}, 1),
		eventIdentityWriters: map[int]*EventIdentityWriter{},
	}
	var err error
	store.warehouse, err = openWarehouse(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	store.columnByProperty.user = userColumnByProperty(ws.UserSchema)
	store.columnByProperty.user["__id__"] = warehouses.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.user["__last_change_time__"] = warehouses.Column{Name: "__last_change_time__", Type: types.DateTime()}
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
	userColumns := propertiesToColumns(userSchema)
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
	userColumns := propertiesToColumns(userSchema)
	return store.warehouse.AlterSchemaQueries(ctx, userColumns, operations)
}

// AddEvents adds events to the store.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
func (store *Store) AddEvents(events [][]any) error {
	store.mustBeOpen()
	switch store.Mode() {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
	return nil
}

// BatchIdentityWriter returns an identity writer for writing user identities in
// batch, relative to the given action (which must be in execution) on the data
// warehouse. purge reports whether identities should be purged from the data
// warehouse after all identities have been written. The ack parameter is the
// acknowledgment function.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
//
// It panics if the ack function is nil.
func (store *Store) BatchIdentityWriter(action *state.Action, purge bool, ack IdentityWriterAckFunc) (*BatchIdentityWriter, error) {
	store.mustBeOpen()
	if ack == nil {
		panic("nil ack function")
	}
	switch store.Mode() {
	case state.Inspection:
		return nil, ErrInspectionMode
	case state.Maintenance:
		return nil, ErrMaintenanceMode
	}
	connection := action.Connection()
	execution, ok := action.Execution()
	if !ok {
		return nil, fmt.Errorf("action is not in execution")
	}
	iw := BatchIdentityWriter{
		store:      store,
		action:     action.ID,
		connection: connection.ID,
		execution:  execution.ID,
		flatter:    newFlatter(action.OutSchema, store.identityColumnByProperty()),
		index:      map[identityKey]int{},
		ack:        ack,
		purge:      purge,
		columns:    map[string]warehouses.Column{},
	}
	return &iw, nil
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

// EventIdentityWriter returns an identity writer for writing user identities,
// relative to the action, on the data warehouse, in case of importing
// identities from events. ack is the ack function (see the documentation of
// IdentityWriter for more details about it).
//
// Creating more than one EventIdentityWriter per action at the same time is not
// supported.
func (store *Store) EventIdentityWriter(actionID int, ack IdentityWriterAckFunc) (*EventIdentityWriter, error) {
	store.mustBeOpen()

	if ack == nil {
		panic("nil ack function")
	}

	// Initialize the EventIdentityWriter.
	iw := &EventIdentityWriter{
		store:   store,
		action:  actionID,
		index:   map[identityKey]int{},
		ack:     ack,
		columns: map[string]warehouses.Column{},
		actions: map[int]struct{}{},
	}

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
		// The action's out schema is invalid when importing identities from
		// events without any transformation in the action.
		identityColumns := store.identityColumnByProperty()
		iw.flatter = newFlatter(action.OutSchema, identityColumns)
	}
	for _, a := range connection.Actions() {
		iw.actions[a.ID] = struct{}{}
	}
	store.mu.Lock()
	store.eventIdentityWriters[action.ID] = iw
	store.mu.Unlock()
	store.ds.state.Unfreeze()

	return iw, nil
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
	records, _, err := store.query(ctx, query, eventColumnByProperty, false)
	return records, err
}

// InitWarehouse initializes the data warehouse creating the events and the
// destinations_users tables.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If an error occurs with the data warehouse, it returns a
// *DataWarehouseError error.
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

// PurgeIdentities purges the identities of the provided actions from the data
// warehouse.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) PurgeIdentities(ctx context.Context, actions []int) error {
	store.mustBeOpen()
	switch store.Mode() {
	case state.Inspection:
		return ErrInspectionMode
	case state.Maintenance:
		return ErrMaintenanceMode
	}
	return store.warehouse.PurgeIdentities(ctx, actions, 0)
}

// RunIdentityResolution runs the Identity Resolution.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a
// *DataWarehouseError error.
func (store *Store) RunIdentityResolution(ctx context.Context) error {
	store.mustBeOpen()

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
	// TODO(Gianluca): also take a look at https://github.com/meergo/meergo/issues/354.
	store.runningIR <- struct{}{}
	defer func() {
		<-store.runningIR
	}()

	// Retrieve the workspace.
	ws, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil
	}

	// Determine the identifiers columns.
	identifiers := make([]warehouses.Column, len(ws.Identifiers))
	for i, ident := range ws.Identifiers {
		identifier, err := types.PropertyByPath(ws.UserSchema, ident)
		if err != nil {
			return err
		}
		if !CanBeIdentifier(identifier.Type) {
			return fmt.Errorf("identifier %q has a not allowed type %v", identifier.Name, identifier.Type)
		}
		identifiers[i] = warehouses.Column{
			Name:     strings.ReplaceAll(ident, ".", "_"),
			Type:     identifier.Type,
			Nullable: identifier.Nullable,
		}
	}

	// Determine the user columns.
	userColumns := propertiesToColumns(ws.UserSchema)

	// Determine the primary sources for every user column.
	userPrimarySources := make(map[string]int, len(ws.UserPrimarySources))
	for p, s := range ws.UserPrimarySources {
		c := strings.ReplaceAll(p, ".", "_")
		userPrimarySources[c] = s
	}

	return store.warehouse.RunIdentityResolution(ctx, identifiers, userColumns, userPrimarySources)
}

// SetDestinationUser sets the destination user for an action.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
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
	return store.query(ctx, query, store.userColumnByProperty(), true)
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
	query.table = "_user_identities"
	query.count = true
	return store.query(ctx, query, store.identityColumnByProperty(), true)
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
	return store.records(ctx, query, schema, "__id__", store.userColumnByProperty(), true)
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

// identityColumnByProperty returns the map from properties to columns for the
// identity schema.
func (store *Store) identityColumnByProperty() map[string]warehouses.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.identity
	store.columnByProperty.mu.Unlock()
	return columns
}

// mustBeOpen panics if store has been closed.
func (store *Store) mustBeOpen() {
	if store.closed.Load() {
		panic("apis/datastore/store is closed")
	}
}

// onAddAction is called when an action of the store's workspace is added.
//
// The notification is propagated by the Store.onAddAction method.
func (store *Store) onAddAction(n state.AddAction) {
	store.mu.Lock()
	for _, iw := range store.eventIdentityWriters {
		if iw.connection == n.Connection {
			iw.onAddAction(n)
		}
	}
	store.mu.Unlock()
}

// onDeleteAction is called when an action of the store's workspace is deleted.
//
// The notification is propagated by the Store.onDeleteAction method.
func (store *Store) onDeleteAction(n state.DeleteAction) {
	connection := n.Action().Connection().ID
	store.mu.Lock()
	for _, iw := range store.eventIdentityWriters {
		if iw.connection == connection {
			iw.onDeleteAction(n)
		}
	}
	store.mu.Unlock()
}

// onDeleteConnection is called when a connection of the store's workspace is
// deleted.
//
// The notification is propagated by the Store.onDeleteConnection method.
func (store *Store) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	store.mu.Lock()
	for _, iw := range store.eventIdentityWriters {
		if iw.connection == connection.ID {
			iw.onDeleteConnection(n)
		}
	}
	store.mu.Unlock()
}

// onSetAction is called when an action of the store's workspace is set.
//
// The notification is propagated by the Store.onSetAction method.
func (store *Store) onSetAction(n state.SetAction) func() {
	store.mu.Lock()
	iw, ok := store.eventIdentityWriters[n.ID]
	store.mu.Unlock()
	if !ok {
		return nil
	}
	return func() {
		iw.onSetAction(n)
	}
}

// onSetWorkspaceUserSchema is called when the user schema of the store's
// workspace is set.
//
// The notification is propagated by the Store.onSetWorkspaceUserSchema method.
func (store *Store) onSetWorkspaceUserSchema(n state.SetWorkspaceUserSchema) {

	// Update the user and the identity columns.
	store.columnByProperty.mu.Lock()
	store.columnByProperty.user = userColumnByProperty(n.UserSchema)
	store.columnByProperty.user["__id__"] = warehouses.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.user["__last_change_time__"] = warehouses.Column{Name: "__last_change_time__", Type: types.DateTime()}
	store.columnByProperty.identity = identityColumnByProperty(store.columnByProperty.user)
	store.columnByProperty.mu.Unlock()

	// Propagate the notification to the EventIdentityWriters.
	store.mu.Lock()
	for _, iw := range store.eventIdentityWriters {
		iw.onSetWorkspaceUserSchema(n)
	}
	store.mu.Unlock()

}

// query executes the provided query on the data warehouse and returns an
// iterator over the results and an estimated count of the rows that would be
// returned if First and Limit of query were not provided.
//
// columnByProperty is the mapping from the path of a property to the relative
// column, and omitNil indicates whether properties with a nil value should be
// omitted from each record.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) query(ctx context.Context, query Query, columnByProperty map[string]warehouses.Column, omitNil bool) ([]map[string]any, int, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty, omitNil)

	var where warehouses.Expr
	if query.Where != nil {
		var err error
		where, err = exprFromWhere(query.Where, columnByProperty)
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
