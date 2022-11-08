//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"context"
	"fmt"
	"sync"

	"chichi/connector"
)

var (
	connectorsMu sync.RWMutex
	connectors   = struct {
		apps      map[string]connector.AppConnectionFunc
		databases map[string]connector.DatabaseConnectionFunc
		storage   map[string]connector.StorageConnectionFunc
		files     map[string]connector.FileConnectionFunc
	}{
		apps:      make(map[string]connector.AppConnectionFunc),
		databases: make(map[string]connector.DatabaseConnectionFunc),
		storage:   make(map[string]connector.StorageConnectionFunc),
		files:     make(map[string]connector.FileConnectionFunc),
	}
)

// RegisterAppConnector makes an app connector available by the provided name.
// If RegisterAppConnector is called twice with the same name or if fn is nil,
// it panics.
func RegisterAppConnector(name string, fn connector.AppConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterAppConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.apps[name]; dup {
		panic("apis: RegisterAppConnector called twice for connector " + name)
	}
	connectors.apps[name] = fn
}

// RegisterDatabaseConnector makes a database connector available by the
// provided name. If RegisterDatabaseConnector is called twice with the same
// name or if fn is nil, it panics.
func RegisterDatabaseConnector(name string, fn connector.DatabaseConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterDatabaseConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.databases[name]; dup {
		panic("apis: RegisterDatabaseConnector called twice for connector " + name)
	}
	connectors.databases[name] = fn
}

// RegisterFileConnector makes a file connector available by the provided name.
// If RegisterFileConnector is called twice with the same name or if fn is nil,
// it panics.
func RegisterFileConnector(name string, fn connector.FileConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterFileConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.files[name]; dup {
		panic("apis: RegisterFileConnector called twice for connector " + name)
	}
	connectors.files[name] = fn
}

// RegisterStorageConnector makes a storage connector available by the provided
// name. If RegisterStorageConnector is called twice with the same name or if
// fn is nil, it panics.
func RegisterStorageConnector(name string, fn connector.StorageConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterStorageConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.storage[name]; dup {
		panic("apis: RegisterStorageConnector called twice for connector " + name)
	}
	connectors.storage[name] = fn
}

// newAppConnection returns a new app connection for the app connector with the
// given name.
func newAppConnection(ctx context.Context, name string, conf *connector.AppConfig) (connector.AppConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.apps[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown app connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newDatabaseConnection returns a new database connection for the database
// connector with the given name.
func newDatabaseConnection(ctx context.Context, name string, conf *connector.DatabaseConfig) (connector.DatabaseConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.databases[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown database connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newFileConnection returns a new file connection for the file connector with
// the given name.
func newFileConnection(ctx context.Context, name string, conf *connector.FileConfig) (connector.FileConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.files[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown file connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newStorageConnection returns a new storage connection for the storage
// connector with the given name.
func newStorageConnection(ctx context.Context, name string, conf *connector.StorageConfig) (connector.StorageConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.storage[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown storage connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
