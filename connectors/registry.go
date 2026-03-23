// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

var registry = struct {
	sync.RWMutex
	applications   map[string]ApplicationSpec
	databases      map[string]DatabaseSpec
	files          map[string]FileSpec
	messageBrokers map[string]MessageBrokerSpec
	sdks           map[string]SDKSpec
	storages       map[string]FileStorageSpec
	webhooks       map[string]WebhookSpec
	usedCodes      map[string]struct{} // used connector codes
}{
	applications:   make(map[string]ApplicationSpec),
	databases:      make(map[string]DatabaseSpec),
	files:          make(map[string]FileSpec),
	messageBrokers: make(map[string]MessageBrokerSpec),
	sdks:           make(map[string]SDKSpec),
	storages:       make(map[string]FileStorageSpec),
	webhooks:       make(map[string]WebhookSpec),
	usedCodes:      make(map[string]struct{}),
}

// Connectors returns the registered connectors as a map from the code to its
// ConnectorSpec.
func Connectors() map[string]ConnectorSpec {
	registry.Lock()
	n := len(registry.applications) +
		len(registry.databases) +
		len(registry.files) +
		len(registry.storages) +
		len(registry.messageBrokers) +
		len(registry.sdks) +
		len(registry.webhooks)
	connectors := make(map[string]ConnectorSpec, n)
	for _, c := range registry.applications {
		connectors[c.Code] = c
	}
	for _, c := range registry.databases {
		connectors[c.Code] = c
	}
	for _, c := range registry.files {
		connectors[c.Code] = c
	}
	for _, c := range registry.messageBrokers {
		connectors[c.Code] = c
	}
	for _, c := range registry.sdks {
		connectors[c.Code] = c
	}
	for _, c := range registry.storages {
		connectors[c.Code] = c
	}
	for _, c := range registry.webhooks {
		connectors[c.Code] = c
	}
	registry.Unlock()
	return connectors
}

// RegisterApplication makes an application connector available by the provided
// code. If RegisterApplication is called twice with the same code or if new is
// nil, it panics.
func RegisterApplication[T any](app ApplicationSpec, new ApplicationNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + app.Code)
	}
	app.newFunc = reflect.ValueOf(new)
	app.ct = reflect.TypeFor[T]()
	validateApplicationConnector(app)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[app.Code]; ok {
		panic("krenalis/connectors: RegisterApplication called with a connector code already registered: " + app.Code)
	}
	registry.applications[app.Code] = app
	registry.usedCodes[app.Code] = struct{}{}
}

// RegisterDatabase makes a database connector available by the provided code.
// If RegisterDatabase is called twice with the same code or if new is nil, it
// panics.
func RegisterDatabase[T any](database DatabaseSpec, new DatabaseNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + database.Code)
	}
	database.newFunc = reflect.ValueOf(new)
	database.ct = reflect.TypeFor[T]()
	validateDatabaseConnector(database)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[database.Code]; ok {
		panic("krenalis/connectors: RegisterDatabase called with a connector code already registered: " + database.Code)
	}
	registry.databases[database.Code] = database
	registry.usedCodes[database.Code] = struct{}{}
}

// RegisterFile makes a file connector available by the provided code. If
// RegisterFile is called twice with the same code or if new is nil, it panics.
func RegisterFile[T any](file FileSpec, new FileNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + file.Code)
	}
	file.newFunc = reflect.ValueOf(new)
	file.ct = reflect.TypeFor[T]()
	validateFileConnector(file)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[file.Code]; ok {
		panic("krenalis/connectors: RegisterFile called with a connector code already registered: " + file.Code)
	}
	registry.files[file.Code] = file
	registry.usedCodes[file.Code] = struct{}{}
}

// RegisterFileStorage makes a file storage connector available by the provided
// code. If RegisterFileStorage is called twice with the same code or if new is
// nil, it panics.
func RegisterFileStorage[T any](storage FileStorageSpec, new FileStorageNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + storage.Code)
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeFor[T]()
	validateFileStorageConnector(storage)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[storage.Code]; ok {
		panic("krenalis/connectors: RegisterFileStorage called with a connector code already registered: " + storage.Code)
	}
	registry.storages[storage.Code] = storage
	registry.usedCodes[storage.Code] = struct{}{}
}

// RegisterMessageBroker makes a message broker connector available by the
// provided code. If RegisterMessageBroker is called twice with the same code or
// if new is nil, it panics.
func RegisterMessageBroker[T any](broker MessageBrokerSpec, new MessageBrokerNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + broker.Code)
	}
	broker.newFunc = reflect.ValueOf(new)
	broker.ct = reflect.TypeFor[T]()
	validateMessageBrokerConnector(broker)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[broker.Code]; ok {
		panic("krenalis/connectors: RegisterMessageBroker called with a connector code already registered: " + broker.Code)
	}
	registry.messageBrokers[broker.Code] = broker
	registry.usedCodes[broker.Code] = struct{}{}
}

// RegisterSDK makes an SDK connector available by the provided code. If
// RegisterSDK is called twice with the same code or if new is nil, it
// panics.
func RegisterSDK[T any](sdk SDKSpec, new SDKNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + sdk.Code)
	}
	sdk.newFunc = reflect.ValueOf(new)
	sdk.ct = reflect.TypeFor[T]()
	validateSDKConnector(sdk)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[sdk.Code]; ok {
		panic("krenalis/connectors: RegisterSDK called with a connector code already registered: " + sdk.Code)
	}
	registry.sdks[sdk.Code] = sdk
	registry.usedCodes[sdk.Code] = struct{}{}
}

// RegisterWebhook makes a webhook connector available by the provided code. If
// RegisterWebhook is called twice with the same code or if new is nil, it
// panics.
func RegisterWebhook[T any](webhook WebhookSpec, new WebhookNewFunc[T]) {
	if new == nil {
		panic("krenalis/connectors: new function is nil for connector " + webhook.Code)
	}
	webhook.newFunc = reflect.ValueOf(new)
	webhook.ct = reflect.TypeFor[T]()
	validateWebhookConnector(webhook)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[webhook.Code]; ok {
		panic("krenalis/connectors: RegisterWebhook called with a connector code already registered: " + webhook.Code)
	}
	registry.webhooks[webhook.Code] = webhook
	registry.usedCodes[webhook.Code] = struct{}{}
}

// RegisteredApplication returns the application registered with the given code.
// If an application with this code is not registered, it panics.
func RegisteredApplication(code string) ApplicationSpec {
	registry.Lock()
	app, ok := registry.applications[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown application connector %q (forgotten import?)", code))
	}
	return app
}

// RegisteredDatabase returns the database registered with the given code.
// If a database with this code is not registered, it panics.
func RegisteredDatabase(code string) DatabaseSpec {
	registry.Lock()
	database, ok := registry.databases[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown database connector %q (forgotten import?)", code))
	}
	return database
}

// RegisteredFile returns the file registered with the given code.
// If a file with this code is not registered, it panics.
func RegisteredFile(code string) FileSpec {
	registry.Lock()
	file, ok := registry.files[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown file connector %q (forgotten import?)", code))
	}
	return file
}

// RegisteredFileStorage returns the file storage registered with the given
// code. If a file storage with this code is not registered, it panics.
func RegisteredFileStorage(code string) FileStorageSpec {
	registry.Lock()
	storage, ok := registry.storages[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown file storage connector %q (forgotten import?)", code))
	}
	return storage
}

// RegisteredMessageBroker returns the message broker registered with the given
// code. If a message broker with this code is not registered, it panics.
func RegisteredMessageBroker(code string) MessageBrokerSpec {
	registry.Lock()
	broker, ok := registry.messageBrokers[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown message broker connector %q (forgotten import?)", code))
	}
	return broker
}

// RegisteredSDK returns the SDK registered with the given code.
// If an SDK with this code is not registered, it panics.
func RegisteredSDK(code string) SDKSpec {
	registry.Lock()
	sdk, ok := registry.sdks[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown SDK connector %q (forgotten import?)", code))
	}
	return sdk
}

// RegisteredWebhook returns the webhook registered with the given code.
// If a webhook with this code is not registered, it panics.
func RegisteredWebhook(code string) WebhookSpec {
	registry.Lock()
	webhook, ok := registry.webhooks[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("krenalis/connectors: unknown webhook connector %q (forgotten import?)", code))
	}
	return webhook
}

// validateCategories validates the categories of a connector.
func validateCategories(connectorName string, categories Categories) {
	if categories == 0 {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: at least one category must be specified", connectorName))
	}
}

// validateApplicationConnector validates the passed application connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateApplicationConnector(app ApplicationSpec) {

	validateConnectorCode("Application", app.Code)
	validateCategories(app.Code, app.Categories)

	if app.AsSource == nil && app.AsDestination == nil {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: ApplicationSpec must include at least the AsSource and AsDestination fields", app.Code))
	}

	if app.AsSource != nil {
		targets := app.AsSource.Targets
		//if targets == 0 || (targets&^(TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^TargetUser) != 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: ApplicationSpec.AsSource.Target is not valid; possible value is connectors.TargetUser", app.Code))
		}
		if targets&TargetUser != 0 {
			if !app.ct.Implements(reflect.TypeFor[RecordFetcher]()) {
				panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
			}
		}
	}

	if app.AsDestination != nil {
		targets := app.AsDestination.Targets
		//if targets == 0 || (targets&^(TargetEvent|TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^(TargetEvent|TargetUser)) != 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: ApplicationSpec.AsDestination.Target is not valid; possible values are connectors.TargetEvent, connectors.TargetUser, or a combination of them using the bitwise OR operator", app.Code))
		}
		if targets&TargetEvent != 0 {
			if !app.ct.Implements(reflect.TypeFor[EventSender]()) {
				panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
			}
			if app.AsDestination.SendingMode == None {
				panic(fmt.Sprintf("krenalis/connectors: connector %s is declared to support Event as destination, but it does not specify a sending mode", app.Code))
			}
		}
		if targets&TargetUser != 0 {
			if !app.ct.Implements(reflect.TypeFor[RecordUpserter]()) {
				panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
			}
		}
	}

	if app.Terms.User != "" || app.Terms.Users != "" || app.Terms.UserID != "" {
		if (app.AsSource == nil || app.AsSource.Targets&TargetUser == 0) &&
			(app.AsDestination == nil || app.AsDestination.Targets&TargetUser == 0) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: cannot specify terms for users"+
				" if it does not support the User target as either source or destination", app.Code))
		}
	}

	// TODO(marco): Implement groups
	//if app.Terms.Group != "" || app.Terms.Groups != "" {
	//	if (app.AsSource == nil || app.AsSource.Targets&GroupTarget == 0) &&
	//		(app.AsDestination == nil || app.AsDestination.Targets&GroupTarget == 0) {
	//		panic(fmt.Sprintf("krenalis/connectors: connector %s: cannot specify a term for group and/or groups"+
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
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
		}
	} else {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if app.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: ServeUI is implemented, but neither ApplicationSpec.AsSource.HasSettings nor ApplicationSpec.AsDestination.HasSettings is set to true", app.Code))
		}
	}

	if app.OAuth.AuthURL != "" {
		iface := reflect.TypeFor[interface {
			OAuthAccount(ctx context.Context) (string, error)
		}]()
		if !app.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", app.Code))
		}
	}

	// Patterns are checked for validity when rate limiters are created; invalid patterns will cause construction to panic.
	var requireOAuth bool
	for _, group := range app.EndpointGroups {
		requireOAuth = requireOAuth || group.RequireOAuth
		if group.Patterns != nil && len(group.Patterns) == 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: Patterns must be nil or contain at least one pattern", app.Code))
		}
		if group.RateLimit.RequestsPerSecond <= 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: RequestsPerSecond must be > 0", app.Code))
		}
		if group.RateLimit.Burst <= 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: Burst must be > 0", app.Code))
		}
		if group.RateLimit.MaxConcurrentRequests < 0 {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: MaxConcurrentRequests must be >= 0", app.Code))
		}
	}
	if app.OAuth.AuthURL == "" && requireOAuth {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: RequireOAuth cannot be true when OAuth is not supported", app.Code))
	}
	if app.OAuth.AuthURL != "" && !requireOAuth {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: OAuth is supported, but there are no endpoint groups that require it", app.Code))
	}

}

// validateDatabaseConnector validates the passed database connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateDatabaseConnector(database DatabaseSpec) {
	validateConnectorCode("Database", database.Code)
	validateCategories(database.Code, database.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Columns(ctx context.Context, table string) ([]Column, error)
		Merge(ctx context.Context, table Table, rows [][]any) error
		Query(ctx context.Context, query string) (Rows, []Column, error)
		SQLLiteral(value any, typ types.Type) string
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !database.ct.Implements(iface) {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: it does not implement the required methods", database.Code))
	}
}

// validateFileConnector validates the passed file connector, performing checks
// to detect errors that could cause panic or errors in the Meergo code that
// uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileConnector(file FileSpec) {

	validateConnectorCode("File", file.Code)
	validateCategories(file.Code, file.Categories)

	if file.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: inconsistency between the declared functionalities and the methods it actually implements", file.Code))
		}
	}

	if file.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
			ContentType(ctx context.Context) string
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

	if file.HasSheets {
		iface := reflect.TypeFor[interface {
			Sheets(ctx context.Context, r io.Reader) ([]string, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

	if (file.AsSource != nil && file.AsSource.HasSettings) ||
		(file.AsDestination != nil && file.AsDestination.HasSettings) {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !file.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", file.Code))
		}
	}

}

// validateFileStorageConnector validates the passed file storage connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateFileStorageConnector(fileStorage FileStorageSpec) {

	validateConnectorCode("File Storage", fileStorage.Code)
	validateCategories(fileStorage.Code, fileStorage.Categories)

	iface := reflect.TypeFor[interface {
		AbsolutePath(ctx context.Context, name string) (string, error)
		ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
	}]()
	if !fileStorage.ct.Implements(iface) {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: it does not implement the minimum required methods", fileStorage.Code))
	}

	if fileStorage.AsSource != nil {
		iface := reflect.TypeFor[interface {
			Reader(ctx context.Context, name string) (io.ReadCloser, time.Time, error)
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Code))
		}
	}

	if fileStorage.AsDestination != nil {
		iface := reflect.TypeFor[interface {
			Write(ctx context.Context, r io.Reader, name, contentType string) error
		}]()
		if !fileStorage.ct.Implements(iface) {
			panic(fmt.Sprintf("krenalis/connectors: connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", fileStorage.Code))
		}
	}

}

// validateMessageBrokerConnector validates the passed message broker connector,
// performing checks to detect errors that could cause panic or errors in the
// Meergo code that uses the connectors.
//
// In case of a validation error, this function panics.
func validateMessageBrokerConnector(broker MessageBrokerSpec) {
	validateConnectorCode("Message Broker", broker.Code)
	validateCategories(broker.Code, broker.Categories)
	iface := reflect.TypeFor[interface {
		Close() error
		Receive(ctx context.Context) (event []byte, ack func(), err error)
		Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error
	}]()
	if !broker.ct.Implements(iface) {
		panic(fmt.Sprintf("krenalis/connectors: connector %s: it does not implement the required methods", broker.Code))
	}
}

// validateSDKConnector validates the passed SDK connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateSDKConnector(sdk SDKSpec) {
	validateConnectorCode("SDK", sdk.Code)
	validateCategories(sdk.Code, sdk.Categories)
}

// validateWebhookConnector validates the passed webhook connector, performing
// checks to detect errors that could cause panic or errors in the Meergo code
// that uses the connectors.
//
// In case of a validation error, this function panics.
func validateWebhookConnector(webhook WebhookSpec) {
	validateConnectorCode("Webhook", webhook.Code)
	validateCategories(webhook.Code, webhook.Categories)
}

// validateConnectorCode validates a connector code. Valid codes contain only
// [a-z0-9-].
func validateConnectorCode(typ string, code string) {
	if code == "" {
		panic(fmt.Sprintf("krenalis/connectors: code is missing for a connector of type %s", typ))
	}
	for i := 0; i < len(code); i++ {
		c := code[i]
		if 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-' {
			continue
		}
		panic(fmt.Sprintf("krenalis/connectors: connector code %q is not valid; valid codes contain only [a-z0-9-]", code))
	}
}
