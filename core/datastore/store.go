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
	"log/slog"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

const flushEventsQueueTimeout = 1 * time.Second // interval to flush queued Events the data warehouse

// ErrDifferentWarehouse is an error indicating that the data warehouse being
// attempted to connect to, during the change of the warehouse settings, is a
// different data warehouse.
var ErrDifferentWarehouse = errors.New("the data warehouse is a different data warehouse")

// ErrNormalMode is returned by Store methods when they cannot execute due to
// the data warehouse being in normal mode.
var ErrNormalMode = errors.New("the data warehouse is in normal mode")

// ErrInspectionMode is returned by Store methods when they cannot execute due
// to the data warehouse being in inspection mode.
var ErrInspectionMode = errors.New("the data warehouse is in inspection mode")

// ErrMaintenanceMode is returned by Store methods when they cannot execute due
// to the data warehouse being in maintenance mode.
var ErrMaintenanceMode = errors.New("the data warehouse is in maintenance mode")

// ErrAlterSchemaInProgress is a error indicating that the an alter schema
// operation is currently in progress on the data warehouse.
var ErrAlterSchemaInProgress = meergo.ErrAlterSchemaInProgress

// ErrIdentityResolutionInProgress is a error indicating that the Identity
// Resolution is currently in progress on the data warehouse.
var ErrIdentityResolutionInProgress = meergo.ErrIdentityResolutionInProgress

// IdentityWriterAckFunc is the function called when a batch of user identities
// have been written to the data warehouse.
type IdentityWriterAckFunc func(ids []string, err error)

// destinationsUsersTable represents the _destinations_users table.
var destinationsUsersTable = meergo.WarehouseTable{
	Name: "_destinations_users",
	Columns: []meergo.Column{
		{Name: "__action__", Type: types.Int(32)},
		{Name: "__user__", Type: types.Text()},
		{Name: "__property__", Type: types.Text()},
	},
	Keys: []string{"__action__", "__user__"},
}

type Store struct {
	ds               *Datastore
	workspace        int
	warehouse        meergo.Warehouse
	columnByProperty struct {
		mu       sync.Mutex
		user     map[string]meergo.Column // including meta properties.
		identity map[string]meergo.Column // including meta properties.
	}
	closed               atomic.Bool
	mu                   sync.Mutex // for 'events' and 'eventIdentityWriters' fields
	events               [][]any
	eventIdentityWriters map[int]*EventIdentityWriter // action -> *EventIdentityWriter
	mc                   *modeCoordinator
}

// newStore returns a new Store for the workspace ws.
// It must be called when the state is frozen.
func newStore(ds *Datastore, ws *state.Workspace) (*Store, error) {
	store := &Store{
		ds:                   ds,
		workspace:            ws.ID,
		eventIdentityWriters: map[int]*EventIdentityWriter{},
	}
	store.mc = newModeCoordinator(ws.Warehouse.Mode)
	var err error
	store.warehouse, err = meergo.RegisteredWarehouse(ws.Warehouse.Name).New(&meergo.WarehouseConfig{
		Settings: ws.Warehouse.Settings,
	})
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	store.columnByProperty.user = userColumnByProperty(ws.UserSchema)
	store.columnByProperty.user["__id__"] = meergo.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.user["__last_change_time__"] = meergo.Column{Name: "__last_change_time__", Type: types.DateTime()}
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

// AddEvents adds events to the store.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
func (store *Store) AddEvents(events [][]any) error {
	// TODO(Gianluca): see the issue https://github.com/meergo/meergo/issues/989.
	store.mustBeOpen()
	_, done, err := store.mc.StartOperation(context.Background(), normalMode)
	if err != nil {
		return err
	}
	defer done()
	store.mu.Lock()
	store.events = append(store.events, events...)
	store.mu.Unlock()
	return nil
}

// AlterSchema alters the user schema.
//
// userSchema is the user schema without meta properties (this parameter is
// useful for obtaining type information and for creating views), while
// operations is the set of operations to apply in order to migrate the current
// schema to userSchema.
//
// If another alter schema operation is in progress on the data warehouse,
// returns a ErrAlterSchemaInProgress error.
//
// If an Identity Resolution is in progress, returns an
// ErrIdentityResolutionInProgress error.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If an error occurs with the data warehouse, it returns a
// *DataWarehouseError error.
func (store *Store) AlterSchema(ctx context.Context, userSchema types.Type, operations []meergo.AlterSchemaOperation) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|maintenanceMode)
	if err != nil {
		return err
	}
	defer done()
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
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) AlterSchemaQueries(ctx context.Context, userSchema types.Type, operations []meergo.AlterSchemaOperation) ([]string, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, anyMode)
	if err != nil {
		return nil, err
	}
	defer done()
	userColumns := propertiesToColumns(userSchema)
	return store.warehouse.AlterSchemaQueries(ctx, userColumns, operations)
}

// BatchIdentityWriter returns an identity writer for writing user identities in
// batch, relative to the given action (which must be in execution) on the data
// warehouse. purge reports whether identities should be purged from the data
// warehouse after all identities have been written. The ack parameter is the
// acknowledgment function.
//
// If the action's output schema does not align with the user schema, it returns
// a *schemas.Error error.
//
// It panics if the ack function is nil.
func (store *Store) BatchIdentityWriter(action *state.Action, purge bool, ack IdentityWriterAckFunc) (*BatchIdentityWriter, error) {
	store.mustBeOpen()
	if ack == nil {
		panic("nil ack function")
	}
	connection := action.Connection()
	execution, ok := action.Execution()
	if !ok {
		return nil, fmt.Errorf("action is not in execution")
	}
	// Check that action's output schema is aligned with the user schema.
	workspace := connection.Workspace()
	err := schemas.CheckAlignment(action.OutSchema, workspace.UserSchema, nil)
	if err != nil {
		return nil, err
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
		columns:    map[string]meergo.Column{},
	}
	return &iw, nil
}

// CanChangeWarehouseSettings determines if it is possible to change the
// warehouse settings of the store's workspace to the given settings.
// If an attempt is made to connect a data warehouse which has already been
// connected to another workspace, the method returns the error
// ErrDifferentWarehouse.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) CanChangeWarehouseSettings(ctx context.Context, toSettings []byte) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, anyMode)
	if err != nil {
		return err
	}
	defer done()
	ws, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil
	}
	// Count the users on the current warehouse.
	query := meergo.RowQuery{
		Columns: []meergo.Column{{Name: "__id__", Type: types.UUID()}},
		Table:   "users",
	}
	_, count1, err := store.warehouse.Query(ctx, query, true)
	if err != nil {
		return err
	}
	// Count the users on the warehouse that will be connected.
	dw, err := meergo.RegisteredWarehouse(ws.Warehouse.Name).New(&meergo.WarehouseConfig{
		Settings: toSettings,
	})
	if err != nil {
		return err
	}
	_, count2, err := dw.Query(ctx, query, true)
	if err != nil {
		return err
	}
	// If the number of users is different, it means (except for the "unlucky"
	// cases where Identity Resolution is in progress) that an attempt is being
	// made to connect to another data warehouse.
	if count1 != count2 {
		return ErrDifferentWarehouse
	}
	return nil
}

// DeleteDestinationUsers deletes the destination users of the provided action.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) DeleteDestinationUsers(ctx context.Context, action int) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	where := meergo.NewBaseExpr(
		meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIs, action)
	return store.warehouse.Delete(ctx, "_user_identities", where)
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
		columns: map[string]meergo.Column{},
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
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()
	query.table = "events"
	records, _, err := store.query(ctx, query, eventColumnByProperty, false)
	return records, err
}

// LastIdentityResolution returns information about the last Identity
// Resolution.
//
// In particular:
//
//   - if the Identity Resolution has been started and completed, returns its
//     start time and end time;
//   - if it is in progress, returns its start time and nil for the end time;
//   - if no Identity Resolution has ever been executed, returns nil and nil.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) LastIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, nil, err
	}
	defer done()
	return store.warehouse.LastIdentityResolution(ctx)
}

// DestinationUser represents a destination user to merge.
type DestinationUser struct {
	User     string
	Property string
}

// MergeDestinationUsers merges the destination users for an action. users
// contains the users to update or create. idsToDelete contains the identifiers
// of the users to delete.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) MergeDestinationUsers(ctx context.Context, action int, users []DestinationUser, idsToDelete []string) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	var rows [][]any
	if users != nil {
		rows = make([][]any, len(users))
		values := make([]any, 3*len(users))
		for i, user := range users {
			j := i * 3
			values[j+0] = action
			values[j+1] = user.User
			values[j+2] = user.Property
			rows[i] = values[j : j+3]
		}
	}
	var deleted []any
	if idsToDelete != nil {
		deleted = make([]any, len(idsToDelete)*2)
		for i, id := range idsToDelete {
			j := i * 2
			deleted[j] = action
			deleted[j+1] = id
		}
	}
	return store.warehouse.Merge(ctx, destinationsUsersTable, rows, deleted)
}

// Mode returns the data warehouse mode.
func (store *Store) Mode() state.WarehouseMode {
	return store.mc.Mode()
}

// PurgeActions purges the provided actions from the data warehouse, deleting
// their associated identities and destination users.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) PurgeActions(ctx context.Context, actions []int) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	values := make([]any, len(actions))
	for i, action := range actions {
		values[i] = action
	}
	where := meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIsOneOf, values...)
	err = store.warehouse.Delete(ctx, "_user_identities", where)
	if err != nil {
		return err
	}
	return store.warehouse.Delete(ctx, "_destinations_users", where)
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
//
// This method should only be called on warehouses that have already been
// initialized, with the aim of correcting any extraordinary issues (such as
// accidental table deletions) in an attempt to make Meergo functional again.
//
// If an error occurs with the data warehouse during the repair, it returns a
// *DataWarehouseError error.
func (store *Store) Repair(ctx context.Context) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, anyMode)
	if err != nil {
		return err
	}
	defer done()
	return store.warehouse.Repair(ctx)
}

// ResolveIdentities resolves the identities of the store's workspace.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
//
// If an Identity Resolution is already in execution, returns an
// IdentityResolutionInProgress error.
//
// If an alter schema operation is in progress on the data warehouse, returns a
// AlterSchemaInProgress error.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) ResolveIdentities(ctx context.Context) error {
	store.mustBeOpen()

	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()

	// Retrieve the workspace.
	ws, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil
	}

	// Determine the identifiers columns.
	identifiers := make([]meergo.Column, len(ws.Identifiers))
	for i, ident := range ws.Identifiers {
		identifier, err := types.PropertyByPath(ws.UserSchema, ident)
		if err != nil {
			return errors.New("unexpected error: identifier does not exist in user schema")
		}
		identifiers[i] = meergo.Column{
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

	return store.warehouse.ResolveIdentities(ctx, identifiers, userColumns, userPrimarySources)
}

// UserIdentities returns the user identities according to the provided query.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) UserIdentities(ctx context.Context, query Query) ([]map[string]any, int, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, 0, err
	}
	defer done()
	query.table = "_user_identities"
	query.count = true
	return store.query(ctx, query, store.identityColumnByProperty(), true)
}

// UserRecords returns an iterator over the users, according to the provided
// query and schema. The properties to return are the properties of schema, and
// the returned properties will conform to schema.
//
// query.Properties must be nil.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If the schema, which must be valid, does not
// align with the user schema, it returns a *schemas.Error error. If an error
// occurs with the data warehouse, it returns a *DataWarehouseError error.
func (store *Store) UserRecords(ctx context.Context, query Query, schema types.Type, matching *Matching) (*Records, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, err
	}
	defer done()
	if query.Properties != nil {
		return nil, errors.New("query.properties is not nil")
	}
	if !schema.Valid() {
		return nil, errors.New("schema is not valid")
	}
	workspace, ok := store.ds.state.Workspace(store.workspace)
	if !ok {
		return nil, fmt.Errorf("workspace does not exist anymore")
	}
	// Check that schema is aligned with the user schema.
	err = schemas.CheckAlignment(schema, workspace.UserSchema, nil)
	if err != nil {
		return nil, err
	}
	query.table = "users"
	query.Properties = types.PropertyNames(schema)
	return store.records(ctx, query, "__id__", store.userColumnByProperty(), true, matching)
}

// Users returns the users according to the provided query.
//
// If the data warehouse is in maintenance mode, it returns the
// ErrMaintenanceMode error. If an error occurs with the data warehouse, it
// returns a *DataWarehouseError error.
func (store *Store) Users(ctx context.Context, query Query) ([]map[string]any, int, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, 0, err
	}
	defer done()
	query.table = "users"
	query.count = true
	return store.query(ctx, query, store.userColumnByProperty(), true)
}

// close closes the store.
// It flushes the events and closes the data warehouse.
// It panics if it has already been called.
func (store *Store) close() error {
	if store.closed.Swap(true) {
		panic("core/datastore/store already closed")
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

// flushEvents flushes a batch of events to the data warehouse.
func (store *Store) flushEvents(events [][]any) {
	slog.Info("flush events", "count", len(events))
	for {
		err := store.warehouse.Merge(context.Background(), eventsMergeTable, events, nil)
		if err != nil {
			slog.Error("cannot flush the event queue", "workspace", store.workspace, "err", err)
			time.Sleep(time.Duration(rand.IntN(2000)) * time.Millisecond)
			continue
		}
		break
	}
}

// identityColumnByProperty returns the map from properties to columns for the
// identity schema.
func (store *Store) identityColumnByProperty() map[string]meergo.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.identity
	store.columnByProperty.mu.Unlock()
	return columns
}

// mustBeOpen panics if store has been closed.
func (store *Store) mustBeOpen() {
	if store.closed.Load() {
		panic("core/datastore/store is closed")
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
	store.columnByProperty.user["__id__"] = meergo.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.user["__last_change_time__"] = meergo.Column{Name: "__last_change_time__", Type: types.DateTime()}
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
func (store *Store) query(ctx context.Context, query Query, columnByProperty map[string]meergo.Column, omitNil bool) ([]map[string]any, int, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty, omitNil)

	var where meergo.Expr
	if query.Where != nil {
		var err error
		where, err = exprFromWhere(query.Where, columnByProperty)
		if err != nil {
			return nil, 0, err
		}
	}

	var orderBy meergo.Column
	var orderDesc bool
	if query.OrderBy != "" {
		var ok bool
		orderBy, ok = columnByProperty[query.OrderBy]
		if !ok {
			return nil, 0, fmt.Errorf("property path %s does not exist", query.OrderBy)
		}
		orderDesc = query.OrderDesc
	}

	rows, count, err := store.warehouse.Query(ctx, meergo.RowQuery{
		Columns:   columns,
		Table:     query.table,
		Where:     where,
		OrderBy:   orderBy,
		OrderDesc: orderDesc,
		First:     query.First,
		Limit:     query.Limit,
	}, true)
	if err != nil {
		return nil, 0, err
	}

	records := make([]map[string]any, 0)

	defer rows.Close()
	row := make([]any, len(columns))
	for rows.Next() {
		if err := rows.Scan(row...); err != nil {
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
func (store *Store) userColumnByProperty() map[string]meergo.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.user
	store.columnByProperty.mu.Unlock()
	return columns
}
