//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps        map[string]AppConnectionFunc
		databases   map[string]DatabaseConnectionFunc
		eventStream map[string]EventStreamConnectionFunc
		files       map[string]FileConnectionFunc
		mobiles     map[string]MobileConnectionFunc
		servers     map[string]ServerConnectionFunc
		storages    map[string]StorageConnectionFunc
		websites    map[string]WebsiteConnectionFunc
	}{
		apps:        make(map[string]AppConnectionFunc),
		databases:   make(map[string]DatabaseConnectionFunc),
		eventStream: make(map[string]EventStreamConnectionFunc),
		files:       make(map[string]FileConnectionFunc),
		mobiles:     make(map[string]MobileConnectionFunc),
		servers:     make(map[string]ServerConnectionFunc),
		storages:    make(map[string]StorageConnectionFunc),
		websites:    make(map[string]WebsiteConnectionFunc),
	}
)

// RegisterApp makes an app connector available by the provided name. If
// RegisterApp is called twice with the same name or if fn is nil, it panics.
func RegisterApp(name string, fn AppConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterApp function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.apps[name]; dup {
		panic("connector: RegisterApp called twice for connector " + name)
	}
	registry.apps[name] = fn
}

// RegisterDatabase makes a database connector available by the provided name.
// If RegisterDatabase is called twice with the same name or if fn is nil, it
// panics.
func RegisterDatabase(name string, fn DatabaseConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterDatabase function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.databases[name]; dup {
		panic("connector: RegisterDatabase called twice for connector " + name)
	}
	registry.databases[name] = fn
}

// RegisterEventStream makes an event stream connector available by the
// provided name. If RegisterEventStream is called twice with the same name or
// if fn is nil, it panics.
func RegisterEventStream(name string, fn EventStreamConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterEventStream function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[name]; dup {
		panic("connector: RegisterEventStream called twice for connector " + name)
	}
	registry.eventStream[name] = fn
}

// RegisterFile makes a file connector available by the provided name. If
// RegisterFile is called twice with the same name or if fn is nil, it panics.
func RegisterFile(name string, fn FileConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterFile function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[name]; dup {
		panic("connector: RegisterFile called twice for connector " + name)
	}
	registry.files[name] = fn
}

// RegisterMobile makes a mobile connector available by the provided name. If
// RegisterDatabase is called twice with the same name or if fn is nil, it
// panics.
func RegisterMobile(name string, fn MobileConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterMobile function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.mobiles[name]; dup {
		panic("connector: RegisterMobile called twice for connector " + name)
	}
	registry.mobiles[name] = fn
}

// RegisterServer makes a server connector available by the provided name. If
// RegisterServer is called twice with the same name or if fn is nil, it
// panics.
func RegisterServer(name string, fn ServerConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterServer function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.servers[name]; dup {
		panic("connector: RegisterServer called twice for connector " + name)
	}
	registry.servers[name] = fn
}

// RegisterStorage makes a storage connector available by the provided name. If
// RegisterStorage is called twice with the same name or if fn is nil, it
// panics.
func RegisterStorage(name string, fn StorageConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterStorage function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.storages[name]; dup {
		panic("connector: RegisterStorage called twice for connector " + name)
	}
	registry.storages[name] = fn
}

// RegisterWebsite makes a website connector available by the provided name. If
// RegisterWebsite is called twice with the same name or if fn is nil, it
// panics.
func RegisterWebsite(name string, fn WebsiteConnectionFunc) {
	if fn == nil {
		panic("connector: RegisterWebsite function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.websites[name]; dup {
		panic("connector: RegisterWebsite called twice for connector " + name)
	}
	registry.websites[name] = fn
}

// NewAppConnection returns a new app connection for the app connector with the
// given name.
func NewAppConnection(ctx context.Context, name string, conf *AppConfig) (AppConnection, error) {
	registryMu.Lock()
	f, ok := registry.apps[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown app connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewDatabaseConnection returns a new database connection for the database
// connector with the given name.
func NewDatabaseConnection(ctx context.Context, name string, conf *DatabaseConfig) (DatabaseConnection, error) {
	registryMu.Lock()
	f, ok := registry.databases[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown database connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewEventStreamConnection returns a new event stream connection for the event
// stream connector with the given name.
func NewEventStreamConnection(ctx context.Context, name string, conf *EventStreamConfig) (EventStreamConnection, error) {
	registryMu.Lock()
	f, ok := registry.eventStream[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown event stream connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewFileConnection returns a new file connection for the file connector with
// the given name.
func NewFileConnection(ctx context.Context, name string, conf *FileConfig) (FileConnection, error) {
	registryMu.Lock()
	f, ok := registry.files[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown file connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewMobileConnection returns a new mobile connection for the mobile connector
// with the given name.
func NewMobileConnection(ctx context.Context, name string, conf *MobileConfig) (MobileConnection, error) {
	registryMu.Lock()
	f, ok := registry.mobiles[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown mobile connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewServerConnection returns a new server connection for the server connector
// with the given name.
func NewServerConnection(ctx context.Context, name string, conf *ServerConfig) (ServerConnection, error) {
	registryMu.Lock()
	f, ok := registry.servers[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown server connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewStorageConnection returns a new storage connection for the storage
// connector with the given name.
func NewStorageConnection(ctx context.Context, name string, conf *StorageConfig) (StorageConnection, error) {
	registryMu.Lock()
	f, ok := registry.storages[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown storage connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}

// NewWebsiteConnection returns a new website connection for the website
// connector with the given name.
func NewWebsiteConnection(ctx context.Context, name string, conf *WebsiteConfig) (WebsiteConnection, error) {
	registryMu.Lock()
	f, ok := registry.websites[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown website connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
