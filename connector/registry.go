//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"fmt"
	"sort"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps      map[string]App
		databases map[string]Database
		streams   map[string]Stream
		files     map[string]File
		mobiles   map[string]Mobile
		servers   map[string]Server
		storages  map[string]Storage
		websites  map[string]Website
	}{
		apps:      make(map[string]App),
		databases: make(map[string]Database),
		streams:   make(map[string]Stream),
		files:     make(map[string]File),
		mobiles:   make(map[string]Mobile),
		servers:   make(map[string]Server),
		storages:  make(map[string]Storage),
		websites:  make(map[string]Website),
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
	n := len(registry.apps) + len(registry.databases) + len(registry.streams) +
		len(registry.files) + len(registry.mobiles) + len(registry.servers) +
		len(registry.storages) + len(registry.websites)
	connectors := make([]Connector, 0, n)
	for _, c := range registry.apps {
		connectors = append(connectors, Connector{Name: c.Name, Type: AppType, Icon: c.Icon})
	}
	for _, c := range registry.databases {
		connectors = append(connectors, Connector{Name: c.Name, Type: DatabaseType, Icon: c.Icon})
	}
	for _, c := range registry.streams {
		connectors = append(connectors, Connector{Name: c.Name, Type: StreamType, Icon: c.Icon})
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

// RegisterStream makes a stream connector available by the provided name.
// If RegisterStream is called twice with the same name or if fn is nil, it
// panics.
func RegisterStream(stream Stream) {
	if stream.Connect == nil {
		panic("connector: RegisterStream function is nil")
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[stream.Name]; dup {
		panic("connector: RegisterStream called twice for connector " + stream.Name)
	}
	registry.streams[stream.Name] = stream
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

// RegisteredApp returns the app registered with the given name.
// If an app with this name is not registered, it panics.
func RegisteredApp(name string) App {
	registryMu.Lock()
	app, ok := registry.apps[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown app connector %q (forgotten import?)", name))
	}
	return app
}

// RegisteredDatabase returns the database registered with the given name.
// If a database with this name is not registered, it panics.
func RegisteredDatabase(name string) Database {
	registryMu.Lock()
	database, ok := registry.databases[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown database connector %q (forgotten import?)", name))
	}
	return database
}

// RegisteredStream returns the stream registered with the given name.
// If a stream with this name is not registered, it panics.
func RegisteredStream(name string) Stream {
	registryMu.Lock()
	stream, ok := registry.streams[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown stream connector %q (forgotten import?)", name))
	}
	return stream
}

// RegisteredFile returns the file registered with the given name.
// If a file with this name is not registered, it panics.
func RegisteredFile(name string) File {
	registryMu.Lock()
	file, ok := registry.files[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown file connector %q (forgotten import?)", name))
	}
	return file
}

// RegisteredMobile returns the mobile registered with the given name.
// If a mobile with this name is not registered, it panics.
func RegisteredMobile(name string) Mobile {
	registryMu.Lock()
	mobile, ok := registry.mobiles[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown mobile connector %q (forgotten import?)", name))
	}
	return mobile
}

// RegisteredServer returns the server registered with the given name.
// If a server with this name is not registered, it panics.
func RegisteredServer(name string) Server {
	registryMu.Lock()
	server, ok := registry.servers[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown server connector %q (forgotten import?)", name))
	}
	return server
}

// RegisteredStorage returns the storage registered with the given name.
// If a storage with this name is not registered, it panics.
func RegisteredStorage(name string) Storage {
	registryMu.Lock()
	storage, ok := registry.storages[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown storage connector %q (forgotten import?)", name))
	}
	return storage
}

// RegisteredWebsite returns the website registered with the given name.
// If a website with this name is not registered, it panics.
func RegisteredWebsite(name string) Website {
	registryMu.Lock()
	website, ok := registry.websites[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown website connector %q (forgotten import?)", name))
	}
	return website
}
