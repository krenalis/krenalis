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
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/datastore/warehouses/clickhouse"
	"github.com/open2b/chichi/apis/datastore/warehouses/postgresql"
	"github.com/open2b/chichi/apis/datastore/warehouses/snowflake"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

type (
	DataWarehouseError        = warehouses.DataWarehouseError
	SettingsError             = warehouses.SettingsError
	UnsupportedAlterSchemaErr = warehouses.UnsupportedAlterSchemaErr
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

	// Add listeners.
	ds.state.AddListener(ds.onAddAction)
	ds.state.AddListener(ds.onDeleteAction)
	ds.state.AddListener(ds.onDeleteConnection)
	ds.state.AddListener(ds.onSetAction)
	ds.state.AddListener(ds.onSetWarehouse)
	ds.state.AddListener(ds.onSetWarehouseMode)
	ds.state.AddListener(ds.onSetWorkspaceUserSchema)
	for _, organization := range st.Organizations() {
		for _, ws := range organization.Workspaces() {
			if ws.Warehouse == nil {
				continue
			}
			store, err := newStore(ds, ws)
			if err != nil {
				slog.Error("cannot create a store", "err", err)
				continue
			}
			ds.store[ws.ID] = store
		}
	}
	return ds
}

// Close closes the datastore. When Close is called, no other calls to
// datastore's methods should be in progress and no other shall be made.
// It panics if it has already been called.
func (ds *Datastore) Close() {
	if ds.closed.Swap(true) {
		panic("apis/datastore already closed")
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

// NormalizeWarehouseSettings returns data warehouse settings in a canonical
// form.
//
// It returns a SettingsError error if the settings are not valid.
func (ds *Datastore) NormalizeWarehouseSettings(typ state.WarehouseType, settings []byte) ([]byte, error) {
	ds.mustBeOpen()
	dw, err := openWarehouse(typ, settings)
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

// PingWarehouse tries to establish a connection to the data warehouse with
// the given settings.
//
// It returns a SettingsError error if the settings are not valid, and a
// *DataWarehouseError error if an error occurs with the data warehouse.
func (ds *Datastore) PingWarehouse(ctx context.Context, typ state.WarehouseType, settings []byte) error {
	ds.mustBeOpen()
	dw, err := openWarehouse(typ, settings)
	if err != nil {
		return err
	}
	defer dw.Close()
	err = dw.Ping(ctx)
	if err != nil {
		return err
	}
	return dw.Close()
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
		panic("apis/datastore is closed")
	}
}

// onAddAction is called when an action is added.
func (ds *Datastore) onAddAction(n state.AddAction) func() {
	connection, _ := ds.state.Connection(n.Connection)
	ws := connection.Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return nil
	}
	return func() {
		store.onAddAction(n)
	}
}

// onDeleteAction is called when an action is deleted.
func (ds *Datastore) onDeleteAction(n state.DeleteAction) func() {
	ws := n.Action().Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return nil
	}
	return func() {
		store.onDeleteAction(n)
	}
}

// onDeleteConnection is called when a connection is deleted.
func (ds *Datastore) onDeleteConnection(n state.DeleteConnection) func() {
	ws := n.Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return nil
	}
	return func() {
		store.onDeleteConnection(n)
	}
}

// onSetAction is called when an action is set.
func (ds *Datastore) onSetAction(n state.SetAction) func() {
	action, _ := ds.state.Action(n.ID)
	ws := action.Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return nil
	}
	return store.onSetAction(n)
}

// onSetWorkspaceUserSchema is called when a workspace's user schema is set.
func (ds *Datastore) onSetWorkspaceUserSchema(n state.SetWorkspaceUserSchema) func() {
	ds.mu.Lock()
	store, ok := ds.store[n.Workspace]
	ds.mu.Unlock()
	if !ok {
		return nil
	}
	return func() {
		store.onSetWorkspaceUserSchema(n)
	}
}

func (ds *Datastore) onSetWarehouse(n state.SetWarehouse) func() {
	return func() {
		// Change the data warehouse mode of the current store.
		if n.Warehouse != nil {
			ds.mu.Lock()
			store := ds.store[n.Workspace]
			ds.mu.Unlock()
			if store != nil {
				store.mu.Lock()
				store.mode = n.Warehouse.Mode
				store.mu.Unlock()
			}
		}
		// Replace the current store with a new store.
		var err error
		var nextStore *Store
		ws, _ := ds.state.Workspace(n.Workspace)
		if ws.Warehouse != nil {
			nextStore, err = newStore(ds, ws)
			if err != nil {
				go func(workspace int, err error) {
					slog.Error("cannot create a new store", "workspace", workspace, "err", err)
				}(ws.ID, err)
			}
		}
		ds.mu.Lock()
		prevStore := ds.store[ws.ID]
		ds.store[ws.ID] = nextStore
		ds.mu.Unlock()
		// Close the previous store.
		if prevStore != nil {
			go func(workspace int) {
				err := prevStore.close()
				if err != nil {
					slog.Error("cannot close store", "workspace", workspace, "err", err)
				}
			}(ws.ID)
		}
	}
}

func (ds *Datastore) onSetWarehouseMode(n state.SetWarehouseMode) func() {
	return func() {
		// Change the data warehouse mode.
		ds.mu.Lock()
		store := ds.store[n.Workspace]
		ds.mu.Unlock()
		store.mu.Lock()
		store.mode = n.Mode
		store.mu.Unlock()
	}
}

// openWarehouse opens a data warehouse with the given type and settings.
// It returns a SettingsError error if the settings are not syntactically
// valid.
func openWarehouse(typ state.WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case state.BigQuery, state.Redshift:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case state.ClickHouse:
		return clickhouse.Open(settings)
	case state.PostgreSQL:
		return postgresql.Open(settings)
	case state.Snowflake:
		return snowflake.Open(settings)
	}
	return nil, fmt.Errorf("warehouse type %d is not valid", typ)
}

// CheckConflictingProperties checks if schema contains conflicting properties,
// and returns an error in that case.
// A property conflicts with another if their representation as columns on the
// data warehouse has the same name.
func CheckConflictingProperties(schema types.Type) error {
	columns := propertiesToColumns(schema)
	names := make(map[string]struct{})
	for _, c := range columns {
		if _, ok := names[c.Name]; ok {
			return fmt.Errorf("two or more properties cannot have the same representation as column %q", c.Name)
		}
		names[c.Name] = struct{}{}
	}
	return nil
}
