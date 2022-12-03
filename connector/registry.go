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
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps        map[string]App
		databases   map[string]Database
		eventStream map[string]EventStream
		files       map[string]File
		mobiles     map[string]Mobile
		servers     map[string]Server
		storages    map[string]Storage
		websites    map[string]Website
	}{
		apps:        make(map[string]App),
		databases:   make(map[string]Database),
		eventStream: make(map[string]EventStream),
		files:       make(map[string]File),
		mobiles:     make(map[string]Mobile),
		servers:     make(map[string]Server),
		storages:    make(map[string]Storage),
		websites:    make(map[string]Website),
	}
)

// Connector represent a connector.
type Connector struct {
	Name string
	Type Type
	Icon string
}

// Connectors returns a list sorted by name of the registered connectors.
func Connectors() []Connector {
	registryMu.Lock()
	defer registryMu.Unlock()
	n := len(registry.apps) + len(registry.databases) + len(registry.eventStream) +
		len(registry.files) + len(registry.mobiles) + len(registry.servers) +
		len(registry.storages) + len(registry.websites)
	connectors := make([]Connector, 0, n)
	for _, c := range registry.apps {
		connectors = append(connectors, Connector{Name: c.Name, Type: AppType, Icon: c.Icon})
	}
	for _, c := range registry.databases {
		connectors = append(connectors, Connector{Name: c.Name, Type: DatabaseType, Icon: c.Icon})
	}
	for _, c := range registry.eventStream {
		connectors = append(connectors, Connector{Name: c.Name, Type: EventStreamType, Icon: c.Icon})
	}
	for _, c := range registry.files {
		connectors = append(connectors, Connector{Name: c.Name, Type: FileType, Icon: c.Icon})
	}
	for _, c := range registry.mobiles {
		connectors = append(connectors, Connector{Name: c.Name, Type: MobileType, Icon: c.Icon})
	}
	for _, c := range registry.servers {
		connectors = append(connectors, Connector{Name: c.Name, Type: ServerType, Icon: c.Icon})
	}
	for _, c := range registry.storages {
		connectors = append(connectors, Connector{Name: c.Name, Type: StorageType, Icon: c.Icon})
	}
	for _, c := range registry.websites {
		connectors = append(connectors, Connector{Name: c.Name, Type: WebsiteType, Icon: c.Icon})
	}
	sort.Slice(connectors, func(i, j int) bool {
		ci, cj := connectors[i], connectors[j]
		return ci.Name < cj.Name || ci.Name == cj.Name && ci.Type < cj.Type
	})
	return connectors
}

// RegisterApp makes an app connector available by the provided name. If
// RegisterApp is called twice with the same name or if fn is nil, it panics.
func RegisterApp(app App) {
	if app.Connect == nil {
		panic("connector: RegisterApp function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.apps[app.Name]; dup {
		panic("connector: RegisterApp called twice for connector " + app.Name)
	}
	registry.apps[app.Name] = app
}

// RegisterDatabase makes a database connector available by the provided name.
// If RegisterDatabase is called twice with the same name or if fn is nil, it
// panics.
func RegisterDatabase(database Database) {
	if database.Connect == nil {
		panic("connector: RegisterDatabase function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.databases[database.Name]; dup {
		panic("connector: RegisterDatabase called twice for connector " + database.Name)
	}
	registry.databases[database.Name] = database
}

// RegisterEventStream makes an event stream connector available by the
// provided name. If RegisterEventStream is called twice with the same name or
// if fn is nil, it panics.
func RegisterEventStream(stream EventStream) {
	if stream.Connect == nil {
		panic("connector: RegisterEventStream function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[stream.Name]; dup {
		panic("connector: RegisterEventStream called twice for connector " + stream.Name)
	}
	registry.eventStream[stream.Name] = stream
}

// RegisterFile makes a file connector available by the provided name. If
// RegisterFile is called twice with the same name or if fn is nil, it panics.
func RegisterFile(file File) {
	if file.Connect == nil {
		panic("connector: RegisterFile function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[file.Name]; dup {
		panic("connector: RegisterFile called twice for connector " + file.Name)
	}
	registry.files[file.Name] = file
}

// RegisterMobile makes a mobile connector available by the provided name. If
// RegisterDatabase is called twice with the same name or if fn is nil, it
// panics.
func RegisterMobile(mobile Mobile) {
	if mobile.Connect == nil {
		panic("connector: RegisterMobile function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.mobiles[mobile.Name]; dup {
		panic("connector: RegisterMobile called twice for connector " + mobile.Name)
	}
	registry.mobiles[mobile.Name] = mobile
}

// RegisterServer makes a server connector available by the provided name. If
// RegisterServer is called twice with the same name or if fn is nil, it
// panics.
func RegisterServer(server Server) {
	if server.Connect == nil {
		panic("connector: RegisterServer function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.servers[server.Name]; dup {
		panic("connector: RegisterServer called twice for connector " + server.Name)
	}
	registry.servers[server.Name] = server
}

// RegisterStorage makes a storage connector available by the provided name. If
// RegisterStorage is called twice with the same name or if fn is nil, it
// panics.
func RegisterStorage(storage Storage) {
	if storage.Connect == nil {
		panic("connector: RegisterStorage function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.storages[storage.Name]; dup {
		panic("connector: RegisterStorage called twice for connector " + storage.Name)
	}
	registry.storages[storage.Name] = storage
}

// RegisterWebsite makes a website connector available by the provided name. If
// RegisterWebsite is called twice with the same name or if fn is nil, it
// panics.
func RegisterWebsite(website Website) {
	if website.Connect == nil {
		panic("connector: RegisterWebsite function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.websites[website.Name]; dup {
		panic("connector: RegisterWebsite called twice for connector " + website.Name)
	}
	registry.websites[website.Name] = website
}

// NewAppConnection returns a new app connection for the app connector with the
// given name.
func NewAppConnection(ctx context.Context, name string, conf *AppConfig) (AppConnection, error) {
	registryMu.Lock()
	app, ok := registry.apps[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown app connector %q (forgotten import?)", name)
	}
	return app.Connect(ctx, conf)
}

// NewDatabaseConnection returns a new database connection for the database
// connector with the given name.
func NewDatabaseConnection(ctx context.Context, name string, conf *DatabaseConfig) (DatabaseConnection, error) {
	registryMu.Lock()
	database, ok := registry.databases[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown database connector %q (forgotten import?)", name)
	}
	return database.Connect(ctx, conf)
}

// NewEventStreamConnection returns a new event stream connection for the event
// stream connector with the given name.
func NewEventStreamConnection(ctx context.Context, name string, conf *EventStreamConfig) (EventStreamConnection, error) {
	registryMu.Lock()
	stream, ok := registry.eventStream[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown event stream connector %q (forgotten import?)", name)
	}
	return stream.Connect(ctx, conf)
}

// NewFileConnection returns a new file connection for the file connector with
// the given name.
func NewFileConnection(ctx context.Context, name string, conf *FileConfig) (FileConnection, error) {
	registryMu.Lock()
	file, ok := registry.files[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown file connector %q (forgotten import?)", name)
	}
	return file.Connect(ctx, conf)
}

// NewMobileConnection returns a new mobile connection for the mobile connector
// with the given name.
func NewMobileConnection(ctx context.Context, name string, conf *MobileConfig) (MobileConnection, error) {
	registryMu.Lock()
	mobile, ok := registry.mobiles[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown mobile connector %q (forgotten import?)", name)
	}
	return mobile.Connect(ctx, conf)
}

// NewServerConnection returns a new server connection for the server connector
// with the given name.
func NewServerConnection(ctx context.Context, name string, conf *ServerConfig) (ServerConnection, error) {
	registryMu.Lock()
	server, ok := registry.servers[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown server connector %q (forgotten import?)", name)
	}
	return server.Connect(ctx, conf)
}

// NewStorageConnection returns a new storage connection for the storage
// connector with the given name.
func NewStorageConnection(ctx context.Context, name string, conf *StorageConfig) (StorageConnection, error) {
	registryMu.Lock()
	storage, ok := registry.storages[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown storage connector %q (forgotten import?)", name)
	}
	return storage.Connect(ctx, conf)
}

// NewWebsiteConnection returns a new website connection for the website
// connector with the given name.
func NewWebsiteConnection(ctx context.Context, name string, conf *WebsiteConfig) (WebsiteConnection, error) {
	registryMu.Lock()
	website, ok := registry.websites[name]
	registryMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("connector: unknown website connector %q (forgotten import?)", name)
	}
	return website.Connect(ctx, conf)
}
