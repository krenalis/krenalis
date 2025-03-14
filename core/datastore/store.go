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
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/telemetry"
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

// ErrAlterInProgress is a error indicating that an operation that alter the
// columns of the user tables is currently in progress on the data warehouse.
var ErrAlterInProgress = meergo.ErrWarehouseAlterInProgress

// ErrIdentityResolutionInProgress is a error indicating that the Identity
// Resolution is currently in progress on the data warehouse.
var ErrIdentityResolutionInProgress = meergo.ErrWarehouseIdentityResolutionInProgress

// AckEvent represents an ack event.
type AckEvent struct {
	Action int
	ID     string
}

// EventWriterAckFunc is the function called when events have been written to
// the data warehouse.
type EventWriterAckFunc func(events []AckEvent, err error)

// EventIdentityWriterAckFunc is the function called when a batch of user
// identities from events have been written to the data warehouse.
type EventIdentityWriterAckFunc func(action int, ids []string, err error)

// IdentityWriterAckFunc is the function called when a batch of user identities
// have been written to the data warehouse.
type IdentityWriterAckFunc func(ids []string, err error)

// destinationsUsersTable represents the _destinations_users table.
var destinationsUsersTable = meergo.Table{
	Name: "_destinations_users",
	Columns: []meergo.Column{
		{Name: "__action__", Type: types.Int(32)},
		{Name: "__external_id__", Type: types.Text()},
		{Name: "__out_matching_value__", Type: types.Text()},
	},
	Keys: []string{"__action__", "__external_id__"},
}

type Store struct {
	ds               *Datastore
	wh               atomic.Value // warehouse
	workspace        int
	columnByProperty struct {
		mu       sync.Mutex
		user     map[string]meergo.Column // including meta properties.
		identity map[string]meergo.Column // including meta properties.
	}
	closed               atomic.Bool
	mu                   sync.Mutex                   // for the 'eventIdentityWriters' field
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
	wh, err := getWarehouseInstance(ws.Warehouse.Type, ws.Warehouse.Settings)
	if err != nil {
		return nil, fmt.Errorf("cannot open data warehouse: %s", err)
	}
	store.wh.Store(wh)
	store.columnByProperty.user = userColumnByProperty(ws.UserSchema)
	store.columnByProperty.user["__id__"] = meergo.Column{Name: "__id__", Type: types.UUID()}
	store.columnByProperty.user["__last_change_time__"] = meergo.Column{Name: "__last_change_time__", Type: types.DateTime()}
	store.columnByProperty.identity = identityColumnByProperty(store.columnByProperty.user)
	return store, nil
}

// AlterUserSchema alters the user schema.
//
// schema is the user schema without meta properties (this parameter is useful
// for obtaining type information and for creating views), while operations is
// the set of operations to apply in order to migrate the current schema to the
// given schema.
//
// TODO(Gianluca): in this method, there is an inconsistency related to the
// parameters, that is: the schema is passed as properties, while the operations
// are columns, so there is a mix of different levels of abstraction. This is
// discussed in the issue https://github.com/meergo/meergo/issues/862.
//
// If another alter schema operation is in progress on the data warehouse,
// returns a ErrAlterInProgress error.
//
// If an Identity Resolution is in progress, returns an
// ErrIdentityResolutionInProgress error.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If an error occurs with the data warehouse, it returns a
// *DataWarehouseError error.
func (store *Store) AlterUserSchema(ctx context.Context, schema types.Type, operations []meergo.AlterOperation) error {
	store.mustBeOpen()
	// TODO(Gianluca): the context here is discarded, rather than passed to the
	// actual IR execution. See issue
	// https://github.com/meergo/meergo/issues/1224.
	_, done, err := store.mc.StartOperation(ctx, normalMode|maintenanceMode)
	if err != nil {
		return err
	}
	defer done()
	columns := util.PropertiesToColumns(schema)
	// TODO(Gianluca): The Background context is used here, since the store does
	// not provide any. See issue https://github.com/meergo/meergo/issues/1224.
	return store.warehouse().AlterUserColumns(context.Background(), columns, operations)
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
	return store.warehouse().Delete(ctx, "_destinations_users", where)
}

// Events returns the events according to the provided query. The returned
// events conform to the event schema. query.Properties must contain at least
// one property from the event schema, excluding the originalTimestamp property.
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
	var addedType bool
	if !slices.Contains(query.Properties, "type") {
		addedType = slices.ContainsFunc(query.Properties, func(name string) bool {
			switch name {
			case "category", "event", "groupId", "name", "properties":
				return true
			}
			return false
		})
		if addedType {
			properties := make([]string, len(query.Properties)+1)
			copy(properties, query.Properties)
			properties[len(properties)-1] = "type"
			query.Properties = properties
		}
	}
	records, _, err := store.query(ctx, query, eventColumnByProperty, false)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if user, ok := record["user"]; ok && user == nil {
			delete(record, "user")
		}
		if userId, ok := record["userId"]; ok && userId == "" {
			record["userId"] = nil
		}
		if ctx, ok := record["context"].(map[string]any); ok {
			for name, value := range ctx {
				if value, ok := value.(map[string]any); ok {
					zero := true
					for _, v := range value {
						if !isZero(v) {
							zero = false
							break
						}
					}
					if zero {
						delete(ctx, name)
					} else if name == "session" && value["start"] == false {
						delete(value, "start")
					}
					continue
				}
				if isZero(value) {
					delete(ctx, name)
				}
			}
		}
		if typ, ok := record["type"]; ok {
			switch typ.(string) {
			case "alias":
				delete(record, "category")
				delete(record, "event")
				delete(record, "groupId")
				delete(record, "name")
				delete(record, "properties")
			case "identify":
				delete(record, "category")
				delete(record, "event")
				delete(record, "groupId")
				delete(record, "name")
				delete(record, "properties")
			case "group":
				delete(record, "category")
				delete(record, "event")
				delete(record, "name")
				delete(record, "properties")
			case "page":
				delete(record, "event")
				delete(record, "groupId")
			case "screen":
				delete(record, "category")
				delete(record, "event")
				delete(record, "groupId")
			case "track":
				delete(record, "category")
				delete(record, "groupId")
				delete(record, "name")
			}
			if addedType {
				delete(record, "type")
			}
		}
	}
	return records, nil
}

// LatestIdentityResolution returns information about the latest Identity
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
func (store *Store) LatestIdentityResolution(ctx context.Context) (startTime, endTime *time.Time, err error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, normalMode|inspectionMode)
	if err != nil {
		return nil, nil, err
	}
	defer done()
	return store.warehouse().LatestIdentityResolution(ctx)
}

// DestinationUser represents a user to be merged.
type DestinationUser struct {
	ExternalID       string // The unique identifier assigned to the user by the app.
	OutMatchingValue string // The value for the out matching property in the app.
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
			values[j+1] = user.ExternalID
			values[j+2] = user.OutMatchingValue
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
	return store.warehouse().Merge(ctx, destinationsUsersTable, rows, deleted)
}

// Mode returns the data warehouse mode.
func (store *Store) Mode() state.WarehouseMode {
	return store.mc.Mode()
}

// NewBatchIdentityWriter returns an identity writer for writing user identities
// in batch, relative to the given action (which must be in execution) on the
// data warehouse. purge reports whether identities should be purged from the
// data warehouse after all identities have been written. The ack parameter is
// the acknowledgment function.
//
// If the action's output schema does not align with the user schema, it returns
// a *schemas.Error error.
//
// It panics if the ack function is nil.
func (store *Store) NewBatchIdentityWriter(action *state.Action, purge bool, ack IdentityWriterAckFunc) (*BatchIdentityWriter, error) {
	store.mustBeOpen()
	return newBatchIdentityWriter(store, action, purge, ack)
}

// NewEventIdentityWriter returns an identity writer for writing user
// identities, relative to the action, on the data warehouse, in case of
// importing identities from events.
//
// It returns an error if an open event identity writer for the provided action
// already exists.
func (store *Store) NewEventIdentityWriter(actionID int, ack EventIdentityWriterAckFunc) (*EventIdentityWriter, error) {
	store.mustBeOpen()
	return newEventIdentityWriter(store, actionID, ack)
}

// NewEventWriter returns a new writer to write events.
func (store *Store) NewEventWriter(ack EventWriterAckFunc) *EventWriter {
	store.mustBeOpen()
	return newEventWriter(store, ack)
}

// PreviewUserSchemaUpdate previews a user schema update returning the queries
// that would be executed updating the user schema of the store.
//
// schema is the user schema without meta properties (this parameter is useful
// for obtaining type information and for creating views), while operations is
// the set of operations to apply in order to migrate the current schema to the
// given schema.
//
// TODO(Gianluca): in this method, there is an inconsistency related to the
// parameters, that is: the schema is passed as properties, while the operations
// are columns, so there is a mix of different levels of abstraction. This is
// discussed in the issue https://github.com/meergo/meergo/issues/862.
//
// If an error occurs with the data warehouse, it returns a *DataWarehouseError
// error.
func (store *Store) PreviewUserSchemaUpdate(ctx context.Context, schema types.Type, operations []meergo.AlterOperation) ([]string, error) {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, anyMode)
	if err != nil {
		return nil, err
	}
	defer done()
	userColumns := util.PropertiesToColumns(schema)
	return store.warehouse().AlterUserColumnsQueries(ctx, userColumns, operations)
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
	err = store.warehouse().Delete(ctx, "_user_identities", where)
	if err != nil {
		return err
	}
	return store.warehouse().Delete(ctx, "_destinations_users", where)
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
// The given user schema will be used to repair the user tables.
//
// This method should only be called on warehouses that have already been
// initialized, with the aim of correcting any extraordinary issues (such as
// accidental table deletions) in an attempt to make Meergo functional again.
//
// If an error occurs with the data warehouse during the repair, it returns a
// *DataWarehouseError error.
func (store *Store) Repair(ctx context.Context, userSchema types.Type) error {
	store.mustBeOpen()
	ctx, done, err := store.mc.StartOperation(ctx, anyMode)
	if err != nil {
		return err
	}
	defer done()
	userColumns := util.PropertiesToColumns(userSchema)
	return store.warehouse().Repair(ctx, userColumns)
}

// StartIdentityResolution starts an Identity Resolution on the store's
// workspace.
//
// If the data warehouse is in inspection mode, it returns the ErrInspectionMode
// error. If it is in maintenance mode, it returns the ErrMaintenanceMode error.
func (store *Store) StartIdentityResolution(ctx context.Context) error {
	store.mustBeOpen()

	ctx, span := telemetry.TraceSpan(ctx, "Store.StartIdentityResolution")
	defer span.End()

	// TODO(Gianluca): the context here is discarded, rather than passed to the
	// actual IR execution. See issue
	// https://github.com/meergo/meergo/issues/1224.
	_, done, err := store.mc.StartOperation(ctx, normalMode)
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
	userColumns := util.PropertiesToColumns(ws.UserSchema)

	// Determine the primary sources for every user column.
	userPrimarySources := make(map[string]int, len(ws.UserPrimarySources))
	for p, s := range ws.UserPrimarySources {
		c := strings.ReplaceAll(p, ".", "_")
		userPrimarySources[c] = s
	}

	// Resolve the identities in a separate goroutine.
	go func() {
		// TODO(Gianluca): The Background context is used here, since the store
		// does not provide any. See issue
		// https://github.com/meergo/meergo/issues/1224.
		err := store.warehouse().ResolveIdentities(context.Background(), identifiers, userColumns, userPrimarySources)
		if err != nil {
			switch err {
			case meergo.ErrWarehouseAlterInProgress, meergo.ErrWarehouseIdentityResolutionInProgress:
			default:
				slog.Error("core/datastore: the execution of the Identity Resolution failed", "workspace", store.workspace, "err", err)
			}
		}
	}()

	return nil
}

// TestWarehouseUpdate tests if it is possible to update the warehouse of the
// store. If an attempt is made to connect a data warehouse which has already
// been connected to another workspace, the method returns the error
// ErrDifferentWarehouse. If an error occurs with the data warehouse, it returns
// a *DataWarehouseError error.
func (store *Store) TestWarehouseUpdate(ctx context.Context, toSettings []byte) error {
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
	_, count1, err := store.warehouse().Query(ctx, query, true)
	if err != nil {
		return err
	}
	// Count the users on the warehouse that will be connected.
	dw, err := getWarehouseInstance(ws.Warehouse.Type, toSettings)
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

// UnsetIdentityProperties unsets values for the specified identity properties
// for the given action. properties contains the property paths and must not be
// empty. If the provided action does not exist, it does nothing.
func (store *Store) UnsetIdentityProperties(ctx context.Context, action int, properties []string) error {
	store.mustBeOpen()
	if len(properties) == 0 {
		return errors.New("core/datastore: invalid empty properties")
	}
	ctx, done, err := store.mc.StartOperation(ctx, normalMode)
	if err != nil {
		return err
	}
	defer done()
	columns := appendColumnsFromProperties(nil, properties, store.userColumnByProperty())
	return store.warehouse().UnsetIdentityColumns(ctx, action, columns)
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
	query.total = true
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
	query.Properties = []string{}
	for path := range types.WalkObjects(schema) {
		query.Properties = append(query.Properties, path)
	}
	return records(ctx, store.warehouse(), query, "__id__", store.userColumnByProperty(), true, matching)
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
	query.total = true
	return store.query(ctx, query, store.userColumnByProperty(), true)
}

// close closes the store.
// It flushes the events and closes the data warehouse.
// It panics if it has already been called.
func (store *Store) close() error {
	if store.closed.Swap(true) {
		panic("core/datastore/store already closed")
	}
	err := store.warehouse().Close()
	if err != nil {
		err = fmt.Errorf("error occurred closing data warehouse: %s", err)
	}
	return err
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

// onCreateAction is called when an action of the store's workspace is created.
//
// The notification is propagated by the Store.onCreateAction method.
func (store *Store) onCreateAction(n state.CreateAction) {
	store.mu.Lock()
	for _, iw := range store.eventIdentityWriters {
		if iw.connection == n.Connection {
			iw.onCreateAction(n)
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

// onUpdateAction is called when an action of the store's workspace is updated.
//
// The notification is propagated by the Store.onUpdateAction method.
func (store *Store) onUpdateAction(n state.UpdateAction) {
	store.mu.Lock()
	iw, ok := store.eventIdentityWriters[n.ID]
	store.mu.Unlock()
	if !ok {
		return
	}
	iw.onUpdateAction(n)
}

// onUpdateUserSchema is called when the user schema of the store's is updated.
//
// The notification is propagated by the Store.onUpdateUserSchema method.
func (store *Store) onUpdateUserSchema(n state.UpdateUserSchema) {

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
		iw.onUpdateUserSchema(n)
	}
	store.mu.Unlock()

}

// query executes the provided query on the data warehouse and returns an
// iterator over the results and an estimated total number of the rows that
// would be returned if First and Limit of query were not provided.
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

	var orderBy []meergo.Column
	var orderDesc bool
	if query.OrderBy != "" {
		c, ok := columnByProperty[query.OrderBy]
		if !ok {
			return nil, 0, fmt.Errorf("property path %s does not exist", query.OrderBy)
		}
		orderBy = []meergo.Column{c}
		orderDesc = query.OrderDesc
	}

	rows, total, err := store.warehouse().Query(ctx, meergo.RowQuery{
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

	// Since total is an estimate, being counted separately from the actual
	// total number of record returned, ensure to not return a value lower than
	// the actually returned number of users.
	total = max(len(records), total)

	return records, total, nil
}

// userColumnByProperty returns the map from properties to columns for the user
// schema.
func (store *Store) userColumnByProperty() map[string]meergo.Column {
	store.columnByProperty.mu.Lock()
	columns := store.columnByProperty.user
	store.columnByProperty.mu.Unlock()
	return columns
}

// warehouse returns the store's warehouse.
func (store *Store) warehouse() meergo.Warehouse {
	return warehouse{inner: store.wh.Load().(meergo.Warehouse)}
}

// isZero reports whether v has its zero value. It supports only the types used
// in the event schema.
func isZero(v any) bool {
	switch v := v.(type) {
	case bool:
		return !v
	case int:
		return v == 0
	case float64:
		return v == 0
	case decimal.Decimal:
		return v.Sign() == 0
	case time.Time:
		return v.IsZero()
	case string:
		return v == ""
	}
	panic(fmt.Sprintf("core/datastore: unexpected type %T", v))
}
