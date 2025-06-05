//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps       map[string]AppInfo
		databases  map[string]DatabaseInfo
		files      map[string]FileInfo
		storages   map[string]FileStorageInfo
		sdks       map[string]SDKInfo
		streams    map[string]StreamInfo
		warehouses map[string]WarehouseDriver
	}{
		apps:       make(map[string]AppInfo),
		databases:  make(map[string]DatabaseInfo),
		files:      make(map[string]FileInfo),
		storages:   make(map[string]FileStorageInfo),
		sdks:       make(map[string]SDKInfo),
		streams:    make(map[string]StreamInfo),
		warehouses: make(map[string]WarehouseDriver),
	}
)

// Connectors returns the registered connectors as a map from the name to its
// ConnectorInfo.
func Connectors() map[string]ConnectorInfo {
	registryMu.Lock()
	defer registryMu.Unlock()
	n := len(registry.apps) + len(registry.databases) + len(registry.files) + len(registry.storages) +
		len(registry.sdks) + len(registry.streams)
	connectors := make(map[string]ConnectorInfo, n)
	for _, c := range registry.apps {
		connectors[c.Name] = c
	}
	for _, c := range registry.databases {
		connectors[c.Name] = c
	}
	for _, c := range registry.files {
		connectors[c.Name] = c
	}
	for _, c := range registry.storages {
		connectors[c.Name] = c
	}
	for _, c := range registry.sdks {
		connectors[c.Name] = c
	}
	for _, c := range registry.streams {
		connectors[c.Name] = c
	}
	return connectors
}

// RegisterApp makes an app connector available by the provided name. If
// RegisterApp is called twice with the same name or if new is nil, it panics.
func RegisterApp[T any](app AppInfo, new AppNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + app.Name)
	}
	app.newFunc = reflect.ValueOf(new)
	app.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateAppConnector(app)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.apps[app.Name]; dup {
		panic("meergo: RegisterApp called twice for connector " + app.Name)
	}
	registry.apps[app.Name] = app
}

// RegisterDatabase makes a database connector available by the provided name.
// If RegisterDatabase is called twice with the same name or if new is nil, it
// panics.
func RegisterDatabase[T any](database DatabaseInfo, new DatabaseNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + database.Name)
	}
	database.newFunc = reflect.ValueOf(new)
	database.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateDatabaseConnector(database)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.databases[database.Name]; dup {
		panic("meergo: RegisterDatabase called twice for connector " + database.Name)
	}
	registry.databases[database.Name] = database
}

// RegisterFile makes a file connector available by the provided name. If
// RegisterFile is called twice with the same name or if new is nil, it panics.
func RegisterFile[T any](file FileInfo, new FileNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + file.Name)
	}
	file.newFunc = reflect.ValueOf(new)
	file.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileConnector(file)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.files[file.Name]; dup {
		panic("meergo: RegisterFile called twice for connector " + file.Name)
	}
	registry.files[file.Name] = file
}

// RegisterFileStorage makes a file storage connector available by the provided
// name. If RegisterFileStorage is called twice with the same name or if new is
// nil, it panics.
func RegisterFileStorage[T any](storage FileStorageInfo, new FileStorageNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + storage.Name)
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileStorageConnector(storage)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.storages[storage.Name]; dup {
		panic("meergo: RegisterFileStorage called twice for connector " + storage.Name)
	}
	registry.storages[storage.Name] = storage
}

// RegisterSDK makes an SDK connector available by the provided name. If
// RegisterSDK is called twice with the same name or if new is nil, it
// panics.
func RegisterSDK[T any](sdk SDKInfo, new SDKNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + sdk.Name)
	}
	sdk.newFunc = reflect.ValueOf(new)
	sdk.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateSDKConnector(sdk)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.sdks[sdk.Name]; dup {
		panic("meergo: RegisterSDK called twice for connector " + sdk.Name)
	}
	registry.sdks[sdk.Name] = sdk
}

// RegisterStream makes a stream connector available by the provided name.
// If RegisterStream is called twice with the same name or if new is nil, it
// panics.
func RegisterStream[T any](stream StreamInfo, new StreamNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + stream.Name)
	}
	stream.newFunc = reflect.ValueOf(new)
	stream.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateStreamConnector(stream)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.streams[stream.Name]; dup {
		panic("meergo: RegisterStream called twice for connector " + stream.Name)
	}
	registry.streams[stream.Name] = stream
}

// RegisterWarehouseDriver makes a warehouse driver available by the provided
// name. If RegisterWarehouseDriver is called twice with the same name or if new
// is nil, it panics.
func RegisterWarehouseDriver[T Warehouse](typ WarehouseDriver, new WarehouseDriverNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for warehouse driver " + typ.Name)
	}
	typ.newFunc = reflect.ValueOf(new)
	typ.ct = reflect.TypeOf((*T)(nil)).Elem()
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, dup := registry.warehouses[typ.Name]; dup {
		panic("meergo: RegisterWarehouseDriver called twice for type " + typ.Name)
	}
	registry.warehouses[typ.Name] = typ
}

// RegisteredApp returns the app registered with the given name.
// If an app with this name is not registered, it panics.
func RegisteredApp(name string) AppInfo {
	registryMu.Lock()
	app, ok := registry.apps[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown app connector %q (forgotten import?)", name))
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
		panic(fmt.Errorf("meergo: unknown database connector %q (forgotten import?)", name))
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
		panic(fmt.Errorf("meergo: unknown stream connector %q (forgotten import?)", name))
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
		panic(fmt.Errorf("meergo: unknown file connector %q (forgotten import?)", name))
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
		panic(fmt.Errorf("meergo: unknown file storage connector %q (forgotten import?)", name))
	}
	return storage
}

// RegisteredSDK returns the SDK registered with the given name.
// If an SDK with this name is not registered, it panics.
func RegisteredSDK(name string) SDKInfo {
	registryMu.Lock()
	sdk, ok := registry.sdks[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown SDK connector %q (forgotten import?)", name))
	}
	return sdk
}

// RegisteredWarehouseDriver returns the warehouse driver registered with the
// given name. If a warehouse driver with this name is not registered, it
// panics.
func RegisteredWarehouseDriver(name string) WarehouseDriver {
	registryMu.Lock()
	warehouse, ok := registry.warehouses[name]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown warehouse driver %q (forgotten import?)", name))
	}
	return warehouse
}

// WarehouseDrivers returns the warehouse drivers.
func WarehouseDrivers() []WarehouseDriver {
	registryMu.Lock()
	drivers := make([]WarehouseDriver, 0, len(registry.warehouses))
	for _, t := range registry.warehouses {
		drivers = append(drivers, t)
	}
	registryMu.Unlock()
	return drivers
}

// validateCategories validates the categories of a connector.
func validateCategories(connectorName string, categories Categories) {
	if categories == 0 {
		panic(fmt.Sprintf("connector %s: at least one category must be specified", connectorName))
	}
}

// validateAppConnector validates the passed app connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateAppConnector(app AppInfo) {

	validateCategories(app.Name, app.Categories)

	if app.AsSource == nil && app.AsDestination == nil {
		panic(fmt.Sprintf("connector %s: AppInfo must include at least the AsSource and AsDestination fields", app.Name))
	}

	if app.AsSource != nil {
		targets := app.AsSource.Targets
		//if targets == 0 || (targets&^(TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^TargetUser) != 0 {
			panic(fmt.Sprintf("connector %s: AppInfo.AsSource.Target is not valid; possible value is meergo.TargetUser", app.Name))
		}
		if targets&TargetUser != 0 {
			iface := reflect.TypeFor[interface {
				RecordSchema(ctx context.Context, target Targets, role Role) (types.Type, error)
				Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]Record, string, error)
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Name))
			}
		}
	}

	if app.AsDestination != nil {
		targets := app.AsDestination.Targets
		//if targets == 0 || (targets&^(TargetEvent|TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^(TargetEvent|TargetUser)) != 0 {
			panic(fmt.Sprintf("connector %s: AppInfo.AsDestination.Target is not valid; possible values are meergo.TargetEvent, meergo.TargetUser, or a combination of them using the bitwise OR operator", app.Name))
		}
		if targets&TargetUser != 0 {
			iface := reflect.TypeFor[interface {
				RecordSchema(ctx context.Context, target Targets, role Role) (types.Type, error)
				Records(ctx context.Context, target Targets, lastChangeTime time.Time, ids, properties []string, cursor string, schema types.Type) ([]Record, string, error)
				Upsert(ctx context.Context, target Targets, records Records) error
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Name))
			}
		}
		if targets&TargetEvent != 0 {
			iface := reflect.TypeFor[interface {
				EventRequest(ctx context.Context, event RawEvent, eventType string, schema types.Type, properties map[string]any, redacted bool) (*EventRequest, error)
				EventTypeSchema(ctx context.Context, eventType string) (types.Type, error)
				EventTypes(ctx context.Context) ([]*EventType, error)
			}]()
			if !app.ct.Implements(iface) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Name))
			}
			if app.AsDestination.SendingMode == None {
				panic(fmt.Sprintf("connector %s is declared to support Event as destination, but it does not specify a sending mode", app.Name))
			}
		}
	}

	if app.Terms.User != "" || app.Terms.Users != "" {
		if (app.AsSource == nil || app.AsSource.Targets&TargetUser == 0) &&
			(app.AsDestination == nil || app.AsDestination.Targets&TargetUser == 0) {
			panic(fmt.Sprintf("connector %s: cannot specify a term for user and/or users"+
				" if it does not support the User target neither as source nor as destination", app.Name))
		}
	}

	// TODO(marco): Implement groups
	//if app.Terms.Group != "" || app.Terms.Groups != "" {
	//	if (app.AsSource == nil || app.AsSource.Targets&GroupTarget == 0) &&
	//		(app.AsDestination == nil || app.AsDestination.Targets&GroupTarget == 0) {
	//		panic(fmt.Sprintf("connector %s: cannot specify a term for group and/or groups"+
	//			" if it does not support the Group target neither as source nor as destination", app.Name))
	//	}
	//}

	var hasSourceSettings = app.AsSource != nil && app.AsSource.HasSettings
	var hasDestinationSettings = app.AsDestination != nil && app.AsDestination.HasSettings
	if hasSourceSettings || hasDestinationSettings {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Name))
		}
	} else if !hasSourceSettings && !hasDestinationSettings {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if app.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: ServeUI is implemented, but neither app.AsSource.HasSettings nor app.AsDestination.HasSettings is set to true", app.Name))
		}
	}

	if app.OAuth.AuthURL != "" {
		iface := reflect.TypeFor[interface {
			OAuthAccount(ctx context.Context) (string, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Name))
		}
	}

}

// validateDatabaseConnector validates the passed database connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateDatabaseConnector(database DatabaseInfo) {
	validateCategories(database.Name, database.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Columns(ctx context.Context, table string) ([]Column, error)
		Merge(ctx context.Context, table Table, rows [][]any) error
		Query(ctx context.Context, query string) (Rows, []Column, error)
		QuoteTime(value any, typ types.Type) string
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !database.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the required methods", database.Name))
	}
}

// validateFileConnector validates the passed file connector, performing checks
// to detect errors that could cause panic or errors in the Meergo code that
// uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileConnector(file FileInfo) {

	validateCategories(file.Name, file.Categories)

	if file.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: inconsistency between the declared functionalities and the methods it actually implements", file.Name))
		}
	}

	if file.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
			ContentType(ctx context.Context) string
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Name))
		}
	}

	if file.HasSheets {
		iface := reflect.TypeFor[interface {
			Sheets(ctx context.Context, r io.Reader) ([]string, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Name))
		}
	}

	if (file.AsSource != nil && file.AsSource.HasSettings) ||
		(file.AsDestination != nil && file.AsDestination.HasSettings) {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Name))
		}
	}

}

// validateFileStorageConnector validates the passed file storage connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileStorageConnector(fileStorage FileStorageInfo) {

	validateCategories(fileStorage.Name, fileStorage.Categories)

	iface := reflect.TypeFor[interface {
		AbsolutePath(ctx context.Context, name string) (string, error)
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !fileStorage.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the minimum required methods", fileStorage.Name))
	}

	if fileStorage.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Name))
		}
	}

	if fileStorage.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, r io.Reader, name, contentType string) error
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Name))
		}
	}

}

// validateSDKConnector validates the passed SDK connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateSDKConnector(sdk SDKInfo) {
	validateCategories(sdk.Name, sdk.Categories)
}

// validateStreamConnector validates the passed stream connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateStreamConnector(stream StreamInfo) {
	validateCategories(stream.Name, stream.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Receive(ctx context.Context) (event []byte, ack func(), err error)
		Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error
	}]()
	if !stream.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the required methods", stream.Name))
	}
}
