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
	"log"
	"sync"
	"sync/atomic"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/datastore/warehouses/clickhouse"
	"chichi/apis/datastore/warehouses/postgresql"
	"chichi/apis/datastore/warehouses/snowflake"
	"chichi/apis/state"

	"github.com/redis/go-redis/v9"
)

type (
	SettingsError      = warehouses.SettingsError
	DataWarehouseError = warehouses.DataWarehouseError
)

// InvalidSettings is the error returned when the Redis database or data
// warehouse settings are not valid.
type InvalidSettings struct {
	Err error
}

func (err InvalidSettings) Error() string {
	return err.Err.Error()
}

// ConnectionFailed is the error returned when a connection to a Redis database
// or to a data warehouse cannot be established.
type ConnectionFailed struct {
	Err error
}

func (err ConnectionFailed) Error() string {
	return err.Err.Error()
}

type RedisConfig struct {
	Network  string
	Addr     string
	Username string
	Password string
	DB       int
}

type Datastore struct {
	state  *state.State
	redis  *redis.Client
	mu     sync.Mutex // for the store field
	store  map[int]*Store
	closed atomic.Bool
}

// New returns a *Datastore instance.
func New(st *state.State, redisConfig RedisConfig) *Datastore {
	ds := &Datastore{
		state: st,
		store: map[int]*Store{},
	}
	ds.redis = redis.NewClient(&redis.Options{
		Network:  redisConfig.Network,
		Addr:     redisConfig.Addr,
		Username: redisConfig.Username,
		Password: redisConfig.Password,
		DB:       redisConfig.DB,
	})
	ds.state.AddListener(ds.onSetWarehouse)
	for _, account := range st.Accounts() {
		for _, ws := range account.Workspaces() {
			if ws.Warehouse == nil {
				continue
			}
			go func(ws *state.Workspace) {
				store, err := newStore(ds, ws)
				if err != nil {
					log.Printf("[error] %s", err)
					return
				}
				ds.mu.Lock()
				ds.store[ws.ID] = store
				ds.mu.Unlock()
			}(ws)
		}
	}
	return ds
}

// Close closes the datastore.
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
			log.Printf("[warning] cannot close store: %s", err)
		}
	}
	ds.mu.Unlock()
}

// PingWarehouse validates data warehouse settings, tries to establish a
// connection to the data warehouse with these settings, and returns the
// settings in a canonical form.
//
// It returns a SettingsError error if the settings are not valid, and a
// DataWarehouseError error if an error occurs with the data warehouse.
func (ds *Datastore) PingWarehouse(ctx context.Context, typ state.WarehouseType, settings []byte) ([]byte, error) {
	ds.mustBeOpen()
	dw, err := openWarehouse(typ, settings)
	if err != nil {
		return nil, err
	}
	err = dw.Ping(ctx)
	if err != nil {
		_ = dw.Close()
		return nil, err
	}
	err = dw.Close()
	if err != nil {
		return nil, err
	}
	// Return the settings in a canonical form.
	return dw.Settings(), nil
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

func (ds *Datastore) onSetWarehouse(n state.SetWarehouse) {
	ws, _ := ds.state.Workspace(n.Workspace)
	go ds.setStore(ws)
	return
}

func (ds *Datastore) setStore(ws *state.Workspace) {
	var err error
	var nextStore *Store
	if ws.Warehouse != nil {
		nextStore, err = newStore(ds, ws)
		if err != nil {
			log.Printf("[error] cannot create a new store for workspace %d: %s", ws.ID, err)
		}
	}
	ds.mu.Lock()
	prevStore := ds.store[ws.ID]
	ds.store[ws.ID] = nextStore
	ds.mu.Unlock()
	if prevStore != nil {
		err = prevStore.close()
		if err != nil {
			log.Printf("[error] error occurred closing a store for the workspace %d: %s", ws.ID, err)
		}
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
