//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps      map[string]App
		databases map[string]Database
		files     map[string]File
		mobiles   map[string]Mobile
		servers   map[string]Server
		storages  map[string]Storage
		streams   map[string]Stream
		websites  map[string]Website
	}{
		apps:      make(map[string]App),
		databases: make(map[string]Database),
		files:     make(map[string]File),
		mobiles:   make(map[string]Mobile),
		servers:   make(map[string]Server),
		storages:  make(map[string]Storage),
		streams:   make(map[string]Stream),
		websites:  make(map[string]Website),
	}
)

// RegisterApp makes an app connector available by the provided name. If
// RegisterApp is called twice with the same name or if fn is nil, it panics.
func RegisterApp[T AppConnection](app App, new AppNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	app.newFunc = reflect.ValueOf(new)
	app.ct = reflect.TypeOf((*T)(nil)).Elem()
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
func RegisterDatabase[T DatabaseConnection](database Database, new DatabaseNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	database.newFunc = reflect.ValueOf(new)
	database.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.databases[database.Name]; dup {
		panic("connector: RegisterDatabase called twice for connector " + database.Name)
	}
	registry.databases[database.Name] = database
}

// RegisterFile makes a file connector available by the provided name. If
// RegisterFile is called twice with the same name or if fn is nil, it panics.
func RegisterFile[T FileConnection](file File, new FileNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	file.newFunc = reflect.ValueOf(new)
	file.ct = reflect.TypeOf((*T)(nil)).Elem()
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
func RegisterMobile[T MobileConnection](mobile Mobile, new MobileNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	mobile.newFunc = reflect.ValueOf(new)
	mobile.ct = reflect.TypeOf((*T)(nil)).Elem()
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
func RegisterServer[T ServerConnection](server Server, new ServerNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	server.newFunc = reflect.ValueOf(new)
	server.ct = reflect.TypeOf((*T)(nil)).Elem()
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
func RegisterStorage[T StorageConnection](storage Storage, new StorageNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.storages[storage.Name]; dup {
		panic("connector: RegisterStorage called twice for connector " + storage.Name)
	}
	registry.storages[storage.Name] = storage
}

// RegisterStream makes a stream connector available by the provided name.
// If RegisterStream is called twice with the same name or if fn is nil, it
// panics.
func RegisterStream[T StreamConnection](stream Stream, new StreamNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	stream.newFunc = reflect.ValueOf(new)
	stream.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[stream.Name]; dup {
		panic("connector: RegisterStream called twice for connector " + stream.Name)
	}
	registry.streams[stream.Name] = stream
}

// RegisterWebsite makes a website connector available by the provided name. If
// RegisterWebsite is called twice with the same name or if fn is nil, it
// panics.
func RegisterWebsite[T WebsiteConnection](website Website, new WebsiteNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	website.newFunc = reflect.ValueOf(new)
	website.ct = reflect.TypeOf((*T)(nil)).Elem()
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
