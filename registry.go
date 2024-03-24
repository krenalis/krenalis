//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"fmt"
	"reflect"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps      map[string]AppInfo
		databases map[string]DatabaseInfo
		files     map[string]FileInfo
		storages  map[string]FileStorageInfo
		mobiles   map[string]MobileInfo
		servers   map[string]ServerInfo
		streams   map[string]StreamInfo
		websites  map[string]WebsiteInfo
	}{
		apps:      make(map[string]AppInfo),
		databases: make(map[string]DatabaseInfo),
		files:     make(map[string]FileInfo),
		storages:  make(map[string]FileStorageInfo),
		mobiles:   make(map[string]MobileInfo),
		servers:   make(map[string]ServerInfo),
		streams:   make(map[string]StreamInfo),
		websites:  make(map[string]WebsiteInfo),
	}
)

// RegisterApp makes an app connector available by the provided name. If
// RegisterApp is called twice with the same name or if fn is nil, it panics.
func RegisterApp[T App](app AppInfo, new AppNewFunc[T]) {
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
func RegisterDatabase[T Database](database DatabaseInfo, new DatabaseNewFunc[T]) {
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
func RegisterFile[T File](file FileInfo, new FileNewFunc[T]) {
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

// RegisterFileStorage makes a file storage connector available by the provided
// name. If RegisterFileStorage is called twice with the same name or if fn is
// nil, it panics.
func RegisterFileStorage[T FileStorage](storage FileStorageInfo, new FileStorageNewFunc[T]) {
	if new == nil {
		panic("connector: new function is nil")
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.storages[storage.Name]; dup {
		panic("connector: RegisterFileStorage called twice for connector " + storage.Name)
	}
	registry.storages[storage.Name] = storage
}

// RegisterMobile makes a mobile connector available by the provided name. If
// RegisterDatabase is called twice with the same name or if fn is nil, it
// panics.
func RegisterMobile[T Mobile](mobile MobileInfo, new MobileNewFunc[T]) {
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
func RegisterServer[T Server](server ServerInfo, new ServerNewFunc[T]) {
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

// RegisterStream makes a stream connector available by the provided name.
// If RegisterStream is called twice with the same name or if fn is nil, it
// panics.
func RegisterStream[T Stream](stream StreamInfo, new StreamNewFunc[T]) {
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
func RegisterWebsite[T Website](website WebsiteInfo, new WebsiteNewFunc[T]) {
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
func RegisteredApp(name string) AppInfo {
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
func RegisteredDatabase(name string) DatabaseInfo {
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
func RegisteredStream(name string) StreamInfo {
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
func RegisteredFile(name string) FileInfo {
	registryMu.Lock()
	file, ok := registry.files[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown file connector %q (forgotten import?)", name))
	}
	return file
}

// RegisteredFileStorage returns the file storage registered with the given
// name. If a file storage with this name is not registered, it panics.
func RegisteredFileStorage(name string) FileStorageInfo {
	registryMu.Lock()
	storage, ok := registry.storages[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown file storage connector %q (forgotten import?)", name))
	}
	return storage
}

// RegisteredMobile returns the mobile registered with the given name.
// If a mobile with this name is not registered, it panics.
func RegisteredMobile(name string) MobileInfo {
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
func RegisteredServer(name string) ServerInfo {
	registryMu.Lock()
	server, ok := registry.servers[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown server connector %q (forgotten import?)", name))
	}
	return server
}

// RegisteredWebsite returns the website registered with the given name.
// If a website with this name is not registered, it panics.
func RegisteredWebsite(name string) WebsiteInfo {
	registryMu.Lock()
	website, ok := registry.websites[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("connector: unknown website connector %q (forgotten import?)", name))
	}
	return website
}
