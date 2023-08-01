//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package datastore

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/datastore/warehouses/clickhouse"
	"chichi/apis/datastore/warehouses/postgresql"
	"chichi/apis/state"

	"github.com/redis/go-redis/v9"
)

type Datastore struct {
	state *state.State
	mu    sync.Mutex
	store map[int]*Store
}

// New returns a *Datastore instance.
func New(st *state.State) *Datastore {
	ds := &Datastore{
		state: st,
		store: map[int]*Store{},
	}
	ds.state.AddListener(ds.onSetRedis)
	ds.state.AddListener(ds.onSetWarehouse)
	for _, account := range st.Accounts() {
		for _, ws := range account.Workspaces() {
			if ws.Redis == nil || ws.Warehouse == nil {
				continue
			}
			go func(ws *state.Workspace) {
				store, err := newStore(ws)
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

func (ds *Datastore) Store(workspace int) *Store {
	ds.mu.Lock()
	store := ds.store[workspace]
	ds.mu.Unlock()
	return store
}

func (ds *Datastore) onSetRedis(n state.SetRedis) {
	ws, _ := ds.state.Workspace(n.Workspace)
	go ds.setStore(ws)
	return
}

func (ds *Datastore) onSetWarehouse(n state.SetWarehouse) {
	ws, _ := ds.state.Workspace(n.Workspace)
	go ds.setStore(ws)
	return
}

func (ds *Datastore) setStore(ws *state.Workspace) {
	var err error
	var nextStore *Store
	if ws.Redis != nil && ws.Warehouse != nil {
		nextStore, err = newStore(ws)
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
// It returns an error if typ or settings are not valid.
func openWarehouse(typ state.WarehouseType, settings []byte) (warehouses.Warehouse, error) {
	switch typ {
	case state.BigQuery, state.Redshift, state.Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not yet supported", typ)
	case state.PostgreSQL:
		return postgresql.Open(settings)
	case state.ClickHouse:
		return clickhouse.Open(settings)
	}
	return nil, fmt.Errorf("warehouse type %d is not valid", typ)
}

// openRedis opens a Redis database with the given settings.
// It returns an error if settings are not valid.
func openRedis(settings []byte) (*redis.Client, error) {
	type RedisConfig struct {
		Network  string
		Addr     string
		Username string
		Password string
		DB       int
	}
	var s RedisConfig
	err := json.Unmarshal(settings, &s)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal settings: %s", err)
	}
	// Instantiate a new client for Redis.
	client := redis.NewClient(&redis.Options{
		Network:  s.Network,
		Addr:     s.Addr,
		Username: s.Username,
		Password: s.Password,
		DB:       s.DB,
	})
	return client, nil
}
