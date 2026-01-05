// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
)

// ErrWarehousePlatformNotExist is returned by the
// Datastore.NormalizeWarehouseSettings method when the provided warehouse
// platform does not exist.
var ErrWarehousePlatformNotExist = errors.New("warehouse platform does not exist")

// ConnectionFailed is the error returned when a connection to a data warehouse
// cannot be established.
type ConnectionFailed struct {
	Err error
}

func (err ConnectionFailed) Error() string {
	return err.Err.Error()
}

type Datastore struct {
	state   *state.State
	mu      sync.Mutex // for the store field
	store   map[int]*Store
	metrics *metrics.Collector
	closed  atomic.Bool
}

// New returns a *Datastore instance.
func New(st *state.State, metrics *metrics.Collector) *Datastore {
	ds := &Datastore{
		state:   st,
		store:   map[int]*Store{},
		metrics: metrics,
	}
	st.Freeze()
	ds.state.AddListener(ds.onCreatePipeline)
	ds.state.AddListener(ds.onCreateWorkspace)
	ds.state.AddListener(ds.onDeleteConnection)
	ds.state.AddListener(ds.onDeletePipeline)
	ds.state.AddListener(ds.onDeleteWorkspace)
	ds.state.AddListener(ds.onEndAlterProfileSchema)
	ds.state.AddListener(ds.onUpdatePipeline)
	ds.state.AddListener(ds.onUpdateWarehouse)
	ds.state.AddListener(ds.onUpdateWarehouseMode)
	for _, ws := range st.Workspaces() {
		store, err := newStore(ds, ws)
		if err != nil {
			slog.Error("core/datastore: cannot create a store", "err", err)
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
// It returns a *warehouses.WarehouseSettingsError error if the settings are not
// valid, a *warehouses.WarehouseNotInitializableError if the data warehouse is
// not initializable, and *UnavailableError if an error occurred with the data
// warehouse.
func (ds *Datastore) CanInitialize(ctx context.Context, name string, settings json.Value) error {
	ds.mustBeOpen()
	dw, err := getWarehouseInstance(name, settings)
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

// CheckMCPSettings checks that the MCP settings are valid, that is it checks
// that datastore's warehouse access with these settings is read-only (at least
// on the Meergo tables), returning a *warehouses.WarehouseSettingsNotReadOnly
// error in case it is not, explaining the reason.
func (ds *Datastore) CheckMCPSettings(ctx context.Context, name string, settings json.Value) error {
	ds.mustBeOpen()
	dw, err := getWarehouseInstance(name, settings)
	if err != nil {
		return err
	}
	defer dw.Close()
	err = dw.CheckReadOnlyAccess(ctx)
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
			slog.Warn("core/datastore: cannot close store", "err", err)
		}
	}
	ds.mu.Unlock()
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo. The given profile schema will be used by the
// initialization to build the profile tables on the warehouse with the
// corresponding columns.
//
// It returns a SettingsError error if the settings are not valid, and a
// *datastore.UnavailableError error if an error occurs with the data warehouse.
func (ds *Datastore) Initialize(ctx context.Context, name string, settings json.Value, profileSchema types.Type) error {
	ds.mustBeOpen()
	dw, err := getWarehouseInstance(name, settings)
	if err != nil {
		return err
	}
	defer dw.Close()
	profileColumns := util.PropertiesToColumns(profileSchema.Properties())
	err = dw.Initialize(ctx, profileColumns)
	if err != nil {
		return err
	}
	return dw.Close()
}

// NormalizeWarehouseSettings returns data warehouse settings in a canonical
// form.
//
// It returns the ErrWarehousePlatformNotExist error if the given warehouse
// platform does not exist, and it returns a SettingsError error if the settings
// are not valid.
func (ds *Datastore) NormalizeWarehouseSettings(platform string, settings json.Value) (json.Value, error) {
	ds.mustBeOpen()
	if _, ok := ds.state.WarehousePlatform(platform); !ok {
		return nil, ErrWarehousePlatformNotExist
	}
	dw, err := getWarehouseInstance(platform, settings)
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

// onCreatePipeline is called when a pipeline is created.
func (ds *Datastore) onCreatePipeline(n state.CreatePipeline) {
	connection, _ := ds.state.Connection(n.Connection)
	ws := connection.Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onCreatePipeline(n)
}

// onCreateWorkspace is called when a workspace is created.
func (ds *Datastore) onCreateWorkspace(n state.CreateWorkspace) {
	ws, _ := ds.state.Workspace(n.ID)
	store, _ := newStore(ds, ws)
	ds.mu.Lock()
	ds.store[ws.ID] = store
	ds.mu.Unlock()
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

// onDeletePipeline is called when a pipeline is deleted.
func (ds *Datastore) onDeletePipeline(n state.DeletePipeline) {
	ws := n.Pipeline().Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onDeletePipeline(n)
}

// onDeleteWarkspace is called when a workspace is deleted.
func (ds *Datastore) onDeleteWorkspace(n state.DeleteWorkspace) {
	ws := n.Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	if ok { // see issue https://github.com/meergo/meergo/issues/2051
		delete(ds.store, ws.ID)
	}
	ds.mu.Unlock()
	if !ok { // see issue https://github.com/meergo/meergo/issues/2051
		return
	}
	err := store.close()
	if err != nil {
		slog.Warn("core/internal/datastore: cannot close store", "err", err)
	}
}

// onEndAlterProfileSchema is called when the alter of the profile schema ends.
func (ds *Datastore) onEndAlterProfileSchema(n state.EndAlterProfileSchema) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	store.onEndAlterProfileSchema(n)
}

// onUpdatePipeline is called when a pipeline is updated.
func (ds *Datastore) onUpdatePipeline(n state.UpdatePipeline) {
	pipeline, _ := ds.state.Pipeline(n.ID)
	ws := pipeline.Connection().Workspace()
	ds.mu.Lock()
	store, ok := ds.store[ws.ID]
	ds.mu.Unlock()
	if !ok {
		return
	}
	store.onUpdatePipeline(n)
}

// onUpdateWarehouse is called when a warehouse is updated.
func (ds *Datastore) onUpdateWarehouse(n state.UpdateWarehouse) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	// Change the data warehouse mode of the store.
	store.mc.ChangeMode(n.Mode, n.CancelIncompatibleOperations)
	// Update the warehouse if the settings have changed.
	prevWarehouse := store.warehouse()
	ws, _ := ds.state.Workspace(n.Workspace)
	nextWarehouse, _ := getWarehouseInstance(ws.Warehouse.Platform, n.Settings)
	if !bytes.Equal(prevWarehouse.Settings(), nextWarehouse.Settings()) {
		store.wh.Store(nextWarehouse)
		// Close the previous warehouse.
		go func(workspace int) {
			err := prevWarehouse.Close()
			if err != nil {
				slog.Error("core/datastore: error closing a warehouse", "workspace", workspace, "err", err)
			}
		}(ws.ID)
	}
}

// onUpdateWarehouseMode is called when the mode of a warehouse is updated.
func (ds *Datastore) onUpdateWarehouseMode(n state.UpdateWarehouseMode) {
	ds.mu.Lock()
	store := ds.store[n.Workspace]
	ds.mu.Unlock()
	store.mc.ChangeMode(n.Mode, n.CancelIncompatibleOperations)
}

// CheckConflictingProperties checks if schema contains conflicting properties
// and returns an error if that case. io specifies whether the check relates
// to "input", "output", or "profile" schema.
//
// A property conflicts with another if their representation as columns in the
// data warehouse has the same name when compared case-insensitively.
func CheckConflictingProperties(io string, schema types.Type) error {
	columns := util.PropertiesToColumns(schema.Properties())
	names := make(map[string]struct{})
	for _, c := range columns {
		name := strings.ToLower(c.Name)
		if _, ok := names[name]; ok {
			return fmt.Errorf("two %s pipeline schema properties would have the same column name %q in the data warehouse, case-insensitively", io, name)
		}
		names[name] = struct{}{}
	}
	return nil
}
