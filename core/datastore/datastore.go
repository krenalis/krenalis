//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

// DataWarehouseNotExist is returned by the Datastore.NormalizeWarehouseSettings
// method when the provided data warehouse does not exist.
var DataWarehouseNotExist = errors.New("data warehouse does not exist")

type (
	DataWarehouseError                 = meergo.DataWarehouseError
	DataWarehouseNotInitializableError = meergo.NotInitializableError
	SettingsError                      = meergo.SettingsError
)

// InvalidSettings is the error returned when the data warehouse settings are
// not valid.
type InvalidSettings struct {
	Err error
}

func (err InvalidSettings) Error() string {
	return err.Err.Error()
}

// ConnectionFailed is the error returned when a connection to a data warehouse
// cannot be established.
type ConnectionFailed struct {
	Err error
}

func (err ConnectionFailed) Error() string {
	return err.Err.Error()
}

type Datastore struct {
	state  *state.State
	mu     sync.Mutex // for the store field
	store  map[int]*Store
	closed atomic.Bool
}

// New returns a *Datastore instance.
func New(st *state.State) *Datastore {
	ds := &Datastore{
		state: st,
		store: map[int]*Store{},
	}
	st.Freeze()
	ds.state.AddListener(ds.onAddAction)
	ds.state.AddListener(ds.onAddWorkspace)
	ds.state.AddListener(ds.onDeleteAction)
	ds.state.AddListener(ds.onDeleteConnection)
	ds.state.AddListener(ds.onSetAction)
	ds.state.AddListener(ds.onSetWarehouse)
	ds.state.AddListener(ds.onSetWarehouseMode)
	ds.state.AddListener(ds.onSetWorkspaceUserSchema)
	for _, ws := range st.Workspaces() {
		store, err := newStore(ds, ws)
		if err != nil {
			slog.Error("cannot create a store", "err", err)
			continue
		}
		ds.store[ws.ID] = store
	}
	st.Unfreeze()
	return ds
}

// CanInitialize indicates whether the warehouse with the provided name and
// settings can be initialized.
//
// It returns:
//
//   - A *DataWarehouseNotInitializableError if the data warehouse is not
//     initializable;
//   - a *SettingsError error if the settings are not valid;
//   - a *DataWarehouseError if an error occurred with the data warehouse.
func (ds *Datastore) CanInitialize(ctx context.Context, name string, settings []byte) error {
	ds.mustBeOpen()
	dw, err := meergo.RegisteredWarehouse(name).New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		return err
	}
	defer dw.Close()
	err = dw.CanInitialize(ctx)
	if err != nil {
		return err
	}
	return dw.Close()
}

// Close closes the datastore. When Close is called, no other calls to
// datastore's methods should be in progress and no other shall be made.
// It panics if it has already been called.
func (ds *Datastore) Close() {
	if ds.closed.Swap(true) {
		panic("core/datastore already closed")
	}
	var err error
	ds.mu.Lock()
	for _, store := range ds.store {
		err = store.close()
		if err != nil {
			slog.Warn("cannot close store", "err", err)
		}
	}
	ds.mu.Unlock()
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
//
// It returns a SettingsError error if the settings are not valid, and a
// *DataWarehouseError error if an error occurs with the data warehouse.
func (ds *Datastore) Initialize(ctx context.Context, name string, settings []byte) error {
	ds.mustBeOpen()
	dw, err := meergo.RegisteredWarehouse(name).New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		return err
	}
	defer dw.Close()
	err = dw.Initialize(ctx)
	if err != nil {
		return err
	}
	return dw.Close()
}

// NormalizeWarehouseSettings returns data warehouse settings in a canonical
// form.
//
// It returns the DataWarehouseNotExist error if a data warehouse with the
// provided name does not exist, and it returns a SettingsError error if the
// settings are not valid.
func (ds *Datastore) NormalizeWarehouseSettings(name string, settings []byte) ([]byte, error) {
	ds.mustBeOpen()
	if _, ok := ds.state.Warehouse(name); !ok {
		return nil, DataWarehouseNotExist
	}
	dw, err := meergo.RegisteredWarehouse(name).New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		return nil, err
	}
	settings = dw.Settings()
	err = dw.Close()
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (ds *Datastore) Store(workspace int) *Store {
	ds.mustBeOpen()
	ds.mu.Lock()
	store := ds.store[workspace]
	ds.mu.Unlock()
	return store
}

// mustBeOpen panics if the datastore has been closed.
func (ds *Datastore) mustBeOpen() {
	if ds.closed.Load() {
		panic("core/datastore is closed")
	}
}

// onAddAction is called when an action is added.
func (ds *Datastore) onAddAction(n state.AddAction) {
	connection, _ := ds.state.Connection(n.Connection)
	ws := connection.Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onAddAction(n)
}

// onAddWorkspace is called when a workspace is added.
func (ds *Datastore) onAddWorkspace(n state.AddWorkspace) {
	ws, _ := ds.state.Workspace(n.ID)
	store, _ := newStore(ds, ws)
	ds.mu.Lock()
	ds.store[ws.ID] = store
	ds.mu.Unlock()
}

// onDeleteAction is called when an action is deleted.
func (ds *Datastore) onDeleteAction(n state.DeleteAction) {
	ws := n.Action().Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onDeleteAction(n)
}

// onDeleteConnection is called when a connection is deleted.
func (ds *Datastore) onDeleteConnection(n state.DeleteConnection) {
	ws := n.Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onDeleteConnection(n)
}

// onSetAction is called when an action is set.
func (ds *Datastore) onSetAction(n state.SetAction) {
	action, _ := ds.state.Action(n.ID)
	ws := action.Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onSetAction(n)
}

// onSetWarehouse is called when the data warehouse is changed.
func (ds *Datastore) onSetWarehouse(n state.SetWarehouse) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	// Change the data warehouse mode of the store.
	store.mc.ChangeMode(n.Mode, n.CancelIncompatibleOperations)
	// Update the warehouse if the settings have changed.
	prevWarehouse := store.warehouse()
	ws, _ := ds.state.Workspace(n.Workspace)
	nextWarehouse, _ := meergo.RegisteredWarehouse(ws.Warehouse.Name).New(&meergo.WarehouseConfig{
		Settings: n.Settings,
	})
	if !bytes.Equal(prevWarehouse.Settings(), nextWarehouse.Settings()) {
		store.wh.Store(nextWarehouse)
		// Close the previous warehouse.
		go func(workspace int) {
			err := prevWarehouse.Close()
			if err != nil {
				slog.Error("error closing a warehouse", "workspace", workspace, "err", err)
			}
		}(ws.ID)
	}
}

// onSetWarehouseMode is called when the mode of a data warehouse is changed.
func (ds *Datastore) onSetWarehouseMode(n state.SetWarehouseMode) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	store.mc.ChangeMode(n.Mode, n.CancelIncompatibleOperations)
}

// onSetWorkspaceUserSchema is called when a workspace's user schema is set.
func (ds *Datastore) onSetWorkspaceUserSchema(n state.SetWorkspaceUserSchema) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	store.onSetWorkspaceUserSchema(n)
}

// CheckConflictingProperties checks if schema contains conflicting properties
// and returns an error if that case. io specifies whether the check relates
// to "input", "output", or "users" schema.
//
// A property conflicts with another if their representation as columns in the
// data warehouse has the same name.
func CheckConflictingProperties(io string, schema types.Type) error {
	columns := propertiesToColumns(schema)
	names := make(map[string]struct{})
	for _, c := range columns {
		if _, ok := names[c.Name]; ok {
			return fmt.Errorf("two properties in the %s schema would have the same column name %q in the data warehouse", io, c.Name)
		}
		names[c.Name] = struct{}{}
	}
	return nil
}
