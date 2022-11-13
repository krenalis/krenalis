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

// ConnectorType represents a connector type.
type ConnectorType int

const (
	AppType ConnectorType = iota + 1
	DatabaseType
	FileType
	MobileType
	ServerType
	StorageType
	WebsiteType
)

// String returns the string representation of typ.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) String() string {
	switch typ {
	case AppType:
		return "App"
	case DatabaseType:
		return "Database"
	case FileType:
		return "File"
	case MobileType:
		return "Mobile"
	case ServerType:
		return "Server"
	case StorageType:
		return "Storage"
	case WebsiteType:
		return "Website"
	}
	panic("invalid connector type")
}

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid ConnectorType value.
func (typ ConnectorType) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

var (
	connectorsMu sync.RWMutex
	connectors   = struct {
		apps      map[string]connector.AppConnectionFunc
		databases map[string]connector.DatabaseConnectionFunc
		files     map[string]connector.FileConnectionFunc
		mobiles   map[string]connector.MobileConnectionFunc
		servers   map[string]connector.ServerConnectionFunc
		storages  map[string]connector.StorageConnectionFunc
		websites  map[string]connector.WebsiteConnectionFunc
	}{
		apps:      make(map[string]connector.AppConnectionFunc),
		databases: make(map[string]connector.DatabaseConnectionFunc),
		files:     make(map[string]connector.FileConnectionFunc),
		mobiles:   make(map[string]connector.MobileConnectionFunc),
		servers:   make(map[string]connector.ServerConnectionFunc),
		storages:  make(map[string]connector.StorageConnectionFunc),
		websites:  make(map[string]connector.WebsiteConnectionFunc),
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

// RegisterMobileConnector makes a mobile connector available by the provided
// name. If RegisterDatabaseConnector is called twice with the same name or if
// fn is nil, it panics.
func RegisterMobileConnector(name string, fn connector.MobileConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterMobileConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.mobiles[name]; dup {
		panic("apis: RegisterMobileConnector called twice for connector " + name)
	}
	connectors.mobiles[name] = fn
}

// RegisterServerConnector makes a server connector available by the provided
// name. If RegisterServerConnector is called twice with the same name or if fn
// is nil, it panics.
func RegisterServerConnector(name string, fn connector.ServerConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterServerConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.servers[name]; dup {
		panic("apis: RegisterServerConnector called twice for connector " + name)
	}
	connectors.servers[name] = fn
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
	if _, dup := connectors.storages[name]; dup {
		panic("apis: RegisterStorageConnector called twice for connector " + name)
	}
	connectors.storages[name] = fn
}

// RegisterWebsiteConnector makes a website connector available by the provided
// name. If RegisterWebsiteConnector is called twice with the same name or if
// fn is nil, it panics.
func RegisterWebsiteConnector(name string, fn connector.WebsiteConnectionFunc) {
	if fn == nil {
		panic("apis: RegisterWebsiteConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.websites[name]; dup {
		panic("apis: RegisterWebsiteConnector called twice for connector " + name)
	}
	connectors.websites[name] = fn
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

// newMobileConnection returns a new mobile connection for the mobile connector
// with the given name.
func newMobileConnection(ctx context.Context, name string, conf *connector.MobileConfig) (connector.MobileConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.mobiles[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown mobile connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newServerConnection returns a new server connection for the server connector
// with the given name.
func newServerConnection(ctx context.Context, name string, conf *connector.ServerConfig) (connector.ServerConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.servers[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown server connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newStorageConnection returns a new storage connection for the storage
// connector with the given name.
func newStorageConnection(ctx context.Context, name string, conf *connector.StorageConfig) (connector.StorageConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.storages[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown storage connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// newWebsiteConnection returns a new website connection for the website
// connector with the given name.
func newWebsiteConnection(ctx context.Context, name string, conf *connector.WebsiteConfig) (connector.WebsiteConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.websites[name]
	if !ok {
		return nil, fmt.Errorf("apis: unknown website connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
