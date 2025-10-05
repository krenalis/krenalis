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

	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
)

var (
	registryMu sync.RWMutex
	registry   = struct {
		apps      map[string]AppInfo
		databases map[string]DatabaseInfo
		files     map[string]FileInfo
		storages  map[string]FileStorageInfo
		sdks      map[string]SDKInfo
		streams   map[string]StreamInfo
		usedCodes map[string]struct{} // used connector codes

		warehouses map[string]WarehouseDriver
	}{
		apps:      make(map[string]AppInfo),
		databases: make(map[string]DatabaseInfo),
		files:     make(map[string]FileInfo),
		storages:  make(map[string]FileStorageInfo),
		sdks:      make(map[string]SDKInfo),
		streams:   make(map[string]StreamInfo),
		usedCodes: make(map[string]struct{}),

		warehouses: make(map[string]WarehouseDriver),
	}
)

// Connectors returns the registered connectors as a map from the code to its
// ConnectorInfo.
func Connectors() map[string]ConnectorInfo {
	registryMu.Lock()
	defer registryMu.Unlock()
	n := len(registry.apps) + len(registry.databases) + len(registry.files) + len(registry.storages) +
		len(registry.sdks) + len(registry.streams)
	connectors := make(map[string]ConnectorInfo, n)
	for _, c := range registry.apps {
		connectors[c.Code] = c
	}
	for _, c := range registry.databases {
		connectors[c.Code] = c
	}
	for _, c := range registry.files {
		connectors[c.Code] = c
	}
	for _, c := range registry.storages {
		connectors[c.Code] = c
	}
	for _, c := range registry.sdks {
		connectors[c.Code] = c
	}
	for _, c := range registry.streams {
		connectors[c.Code] = c
	}
	return connectors
}

// RegisterApp makes an app connector available by the provided code. If
// RegisterApp is called twice with the same code or if new is nil, it panics.
func RegisterApp[T any](app AppInfo, new AppNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + app.Code)
	}
	app.newFunc = reflect.ValueOf(new)
	app.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateAppConnector(app)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[app.Code]; ok {
		panic("meergo: RegisterApp called with a connector code already registered: " + app.Code)
	}
	registry.apps[app.Code] = app
	registry.usedCodes[app.Code] = struct{}{}
}

// RegisterDatabase makes a database connector available by the provided code.
// If RegisterDatabase is called twice with the same code or if new is nil, it
// panics.
func RegisterDatabase[T any](database DatabaseInfo, new DatabaseNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + database.Code)
	}
	database.newFunc = reflect.ValueOf(new)
	database.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateDatabaseConnector(database)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[database.Code]; ok {
		panic("meergo: RegisterDatabase called with a connector code already registered: " + database.Code)
	}
	registry.databases[database.Code] = database
	registry.usedCodes[database.Code] = struct{}{}
}

// RegisterFile makes a file connector available by the provided code. If
// RegisterFile is called twice with the same code or if new is nil, it panics.
func RegisterFile[T any](file FileInfo, new FileNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + file.Code)
	}
	file.newFunc = reflect.ValueOf(new)
	file.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileConnector(file)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[file.Code]; ok {
		panic("meergo: RegisterFile called with a connector code already registered: " + file.Code)
	}
	registry.files[file.Code] = file
	registry.usedCodes[file.Code] = struct{}{}
}

// RegisterFileStorage makes a file storage connector available by the provided
// code. If RegisterFileStorage is called twice with the same code or if new is
// nil, it panics.
func RegisterFileStorage[T any](storage FileStorageInfo, new FileStorageNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + storage.Code)
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileStorageConnector(storage)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[storage.Code]; ok {
		panic("meergo: RegisterFileStorage called with a connector code already registered: " + storage.Code)
	}
	registry.storages[storage.Code] = storage
	registry.usedCodes[storage.Code] = struct{}{}
}

// RegisterSDK makes an SDK connector available by the provided code. If
// RegisterSDK is called twice with the same code or if new is nil, it
// panics.
func RegisterSDK[T any](sdk SDKInfo, new SDKNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + sdk.Code)
	}
	sdk.newFunc = reflect.ValueOf(new)
	sdk.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateSDKConnector(sdk)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[sdk.Code]; ok {
		panic("meergo: RegisterSDK called with a connector code already registered: " + sdk.Code)
	}
	registry.sdks[sdk.Code] = sdk
	registry.usedCodes[sdk.Code] = struct{}{}
}

// RegisterStream makes a stream connector available by the provided code.
// If RegisterStream is called twice with the same code or if new is nil, it
// panics.
func RegisterStream[T any](stream StreamInfo, new StreamNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + stream.Code)
	}
	stream.newFunc = reflect.ValueOf(new)
	stream.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateStreamConnector(stream)
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry.usedCodes[stream.Code]; ok {
		panic("meergo: RegisterSDK called with a connector code already registered: " + stream.Code)
	}
	registry.streams[stream.Code] = stream
	registry.usedCodes[stream.Code] = struct{}{}
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

// RegisteredApp returns the app registered with the given code.
// If an app with this code is not registered, it panics.
func RegisteredApp(code string) AppInfo {
	registryMu.Lock()
	app, ok := registry.apps[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown app connector %q (forgotten import?)", code))
	}
	return app
}

// RegisteredDatabase returns the database registered with the given code.
// If a database with this code is not registered, it panics.
func RegisteredDatabase(code string) DatabaseInfo {
	registryMu.Lock()
	database, ok := registry.databases[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown database connector %q (forgotten import?)", code))
	}
	return database
}

// RegisteredStream returns the stream registered with the given code.
// If a stream with this code is not registered, it panics.
func RegisteredStream(code string) StreamInfo {
	registryMu.Lock()
	stream, ok := registry.streams[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown stream connector %q (forgotten import?)", code))
	}
	return stream
}

// RegisteredFile returns the file registered with the given code.
// If a file with this code is not registered, it panics.
func RegisteredFile(code string) FileInfo {
	registryMu.Lock()
	file, ok := registry.files[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown file connector %q (forgotten import?)", code))
	}
	return file
}

// RegisteredFileStorage returns the file storage registered with the given
// code. If a file storage with this code is not registered, it panics.
func RegisteredFileStorage(code string) FileStorageInfo {
	registryMu.Lock()
	storage, ok := registry.storages[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown file storage connector %q (forgotten import?)", code))
	}
	return storage
}

// RegisteredSDK returns the SDK registered with the given code.
// If an SDK with this code is not registered, it panics.
func RegisteredSDK(code string) SDKInfo {
	registryMu.Lock()
	sdk, ok := registry.sdks[code]
	registryMu.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown SDK connector %q (forgotten import?)", code))
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

	validateConnectorCode("App", app.Code)
	validateCategories(app.Code, app.Categories)

	if app.AsSource == nil && app.AsDestination == nil {
		panic(fmt.Sprintf("connector %s: AppInfo must include at least the AsSource and AsDestination fields", app.Code))
	}

	if app.AsSource != nil {
		targets := app.AsSource.Targets
		//if targets == 0 || (targets&^(TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^TargetUser) != 0 {
			panic(fmt.Sprintf("connector %s: AppInfo.AsSource.Target is not valid; possible value is meergo.TargetUser", app.Code))
		}
		if targets&TargetUser != 0 {
			if !app.ct.Implements(reflect.TypeFor[RecordFetcher]()) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
			}
		}
	}

	if app.AsDestination != nil {
		targets := app.AsDestination.Targets
		//if targets == 0 || (targets&^(TargetEvent|TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^(TargetEvent|TargetUser)) != 0 {
			panic(fmt.Sprintf("connector %s: AppInfo.AsDestination.Target is not valid; possible values are meergo.TargetEvent, meergo.TargetUser, or a combination of them using the bitwise OR operator", app.Code))
		}
		if targets&TargetEvent != 0 {
			if !app.ct.Implements(reflect.TypeFor[EventSender]()) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
			}
			if app.AsDestination.SendingMode == None {
				panic(fmt.Sprintf("connector %s is declared to support Event as destination, but it does not specify a sending mode", app.Code))
			}
			if targets&TargetUser != 0 {
				if !app.ct.Implements(reflect.TypeFor[RecordUpserter]()) {
					panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
				}
			}
		}
	}

	if app.Terms.User != "" || app.Terms.Users != "" {
		if (app.AsSource == nil || app.AsSource.Targets&TargetUser == 0) &&
			(app.AsDestination == nil || app.AsDestination.Targets&TargetUser == 0) {
			panic(fmt.Sprintf("connector %s: cannot specify a term for user and/or users"+
				" if it does not support the User target neither as source nor as destination", app.Code))
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
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
		}
	} else {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if app.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: ServeUI is implemented, but neither app.AsSource.HasSettings nor app.AsDestination.HasSettings is set to true", app.Code))
		}
	}

	if app.OAuth.AuthURL != "" {
		iface := reflect.TypeFor[interface {
			OAuthAccount(ctx context.Context) (string, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
		}
	}

	// Patterns are checked for validity when rate limiters are created; invalid patterns will cause construction to panic.
	var requireOAuth bool
	for _, group := range app.EndpointGroups {
		requireOAuth = requireOAuth || group.RequireOAuth
		if group.Patterns != nil && len(group.Patterns) == 0 {
			panic(fmt.Sprintf("connector %s: Patterns must be nil or contain at least one pattern", app.Code))
		}
		if group.RateLimit.RequestsPerSecond <= 0 {
			panic(fmt.Sprintf("connector %s: RequestsPerSecond must be > 0", app.Code))
		}
		if group.RateLimit.Burst <= 0 {
			panic(fmt.Sprintf("connector %s: Burst must be > 0", app.Code))
		}
		if group.RateLimit.MaxConcurrentRequests < 0 {
			panic(fmt.Sprintf("connector %s: MaxConcurrentRequests must be >= 0", app.Code))
		}
	}
	if app.OAuth.AuthURL == "" && requireOAuth {
		panic(fmt.Sprintf("connector %s: RequireOAuth cannot be true when OAuth is not supported", app.Code))
	}
	if app.OAuth.AuthURL != "" && !requireOAuth {
		panic(fmt.Sprintf("connector %s: OAuth is supported, but there are no endpoint groups that require it", app.Code))
	}

}

// validateDatabaseConnector validates the passed database connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateDatabaseConnector(database DatabaseInfo) {
	validateConnectorCode("Database", database.Code)
	validateCategories(database.Code, database.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Columns(ctx context.Context, table string) ([]Column, error)
		Merge(ctx context.Context, table Table, rows [][]any) error
		Query(ctx context.Context, query string) (Rows, []Column, error)
		QuoteTime(value any, typ types.Type) string
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !database.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the required methods", database.Code))
	}
}

// validateFileConnector validates the passed file connector, performing checks
// to detect errors that could cause panic or errors in the Meergo code that
// uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileConnector(file FileInfo) {

	validateConnectorCode("File", file.Code)
	validateCategories(file.Code, file.Categories)

	if file.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: inconsistency between the declared functionalities and the methods it actually implements", file.Code))
		}
	}

	if file.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
			ContentType(ctx context.Context) string
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

	if file.HasSheets {
		iface := reflect.TypeFor[interface {
			Sheets(ctx context.Context, r io.Reader) ([]string, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

	if (file.AsSource != nil && file.AsSource.HasSettings) ||
		(file.AsDestination != nil && file.AsDestination.HasSettings) {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

}

// validateFileStorageConnector validates the passed file storage connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileStorageConnector(fileStorage FileStorageInfo) {

	validateConnectorCode("File Storage", fileStorage.Code)
	validateCategories(fileStorage.Code, fileStorage.Categories)

	iface := reflect.TypeFor[interface {
		AbsolutePath(ctx context.Context, name string) (string, error)
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !fileStorage.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the minimum required methods", fileStorage.Code))
	}

	if fileStorage.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Code))
		}
	}

	if fileStorage.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, r io.Reader, name, contentType string) error
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Code))
		}
	}

}

// validateSDKConnector validates the passed SDK connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateSDKConnector(sdk SDKInfo) {
	validateConnectorCode("SDK", sdk.Code)
	validateCategories(sdk.Code, sdk.Categories)
}

// validateStreamConnector validates the passed stream connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateStreamConnector(stream StreamInfo) {
	validateConnectorCode("Stream", stream.Code)
	validateCategories(stream.Code, stream.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Receive(ctx context.Context) (event []byte, ack func(), err error)
		Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error
	}]()
	if !stream.ct.Implements(iface) {
		panic(fmt.Sprintf("connector %s: it does not implement the required methods", stream.Code))
	}
}

// validateConnectorCode validates a connector code. Valid codes contain only
// [a-z0-9-].
func validateConnectorCode(typ string, code string) {
	if code == "" {
		panic(fmt.Sprintf("code is missing for a connector of type %s", typ))
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-' {
			continue
		}
		panic(fmt.Sprintf("connector code %q is not valid; valid codes contain only [a-z0-9-]", code))
	}
}
