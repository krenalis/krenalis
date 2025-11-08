// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connectors

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

var registry = struct {
	sync.RWMutex
	apis           map[string]APISpec
	databases      map[string]DatabaseSpec
	files          map[string]FileSpec
	messageBrokers map[string]MessageBrokerSpec
	sdks           map[string]SDKSpec
	storages       map[string]FileStorageSpec
	webhooks       map[string]WebhookSpec
	usedCodes      map[string]struct{} // used connector codes
}{
	apis:           make(map[string]APISpec),
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
	n := len(registry.apis) +
		len(registry.databases) +
		len(registry.files) +
		len(registry.storages) +
		len(registry.messageBrokers) +
		len(registry.sdks) +
		len(registry.webhooks)
	connectors := make(map[string]ConnectorSpec, n)
	for _, c := range registry.apis {
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

// RegisterAPI makes an API connector available by the provided code. If
// RegisterAPI is called twice with the same code or if new is nil, it panics.
func RegisterAPI[T any](api APISpec, new APINewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + api.Code)
	}
	api.newFunc = reflect.ValueOf(new)
	api.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateAPIConnector(api)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[api.Code]; ok {
		panic("meergo: RegisterAPI called with a connector code already registered: " + api.Code)
	}
	registry.apis[api.Code] = api
	registry.usedCodes[api.Code] = struct{}{}
}

// RegisterDatabase makes a database connector available by the provided code.
// If RegisterDatabase is called twice with the same code or if new is nil, it
// panics.
func RegisterDatabase[T any](database DatabaseSpec, new DatabaseNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + database.Code)
	}
	database.newFunc = reflect.ValueOf(new)
	database.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateDatabaseConnector(database)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[database.Code]; ok {
		panic("meergo: RegisterDatabase called with a connector code already registered: " + database.Code)
	}
	registry.databases[database.Code] = database
	registry.usedCodes[database.Code] = struct{}{}
}

// RegisterFile makes a file connector available by the provided code. If
// RegisterFile is called twice with the same code or if new is nil, it panics.
func RegisterFile[T any](file FileSpec, new FileNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + file.Code)
	}
	file.newFunc = reflect.ValueOf(new)
	file.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileConnector(file)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[file.Code]; ok {
		panic("meergo: RegisterFile called with a connector code already registered: " + file.Code)
	}
	registry.files[file.Code] = file
	registry.usedCodes[file.Code] = struct{}{}
}

// RegisterFileStorage makes a file storage connector available by the provided
// code. If RegisterFileStorage is called twice with the same code or if new is
// nil, it panics.
func RegisterFileStorage[T any](storage FileStorageSpec, new FileStorageNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + storage.Code)
	}
	storage.newFunc = reflect.ValueOf(new)
	storage.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateFileStorageConnector(storage)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[storage.Code]; ok {
		panic("meergo: RegisterFileStorage called with a connector code already registered: " + storage.Code)
	}
	registry.storages[storage.Code] = storage
	registry.usedCodes[storage.Code] = struct{}{}
}

// RegisterMessageBroker makes a message broker connector available by the
// provided code. If RegisterMessageBroker is called twice with the same code or
// if new is nil, it panics.
func RegisterMessageBroker[T any](broker MessageBrokerSpec, new MessageBrokerNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + broker.Code)
	}
	broker.newFunc = reflect.ValueOf(new)
	broker.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateMessageBrokerConnector(broker)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[broker.Code]; ok {
		panic("meergo: RegisterMessageBroker called with a connector code already registered: " + broker.Code)
	}
	registry.messageBrokers[broker.Code] = broker
	registry.usedCodes[broker.Code] = struct{}{}
}

// RegisterSDK makes an SDK connector available by the provided code. If
// RegisterSDK is called twice with the same code or if new is nil, it
// panics.
func RegisterSDK[T any](sdk SDKSpec, new SDKNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + sdk.Code)
	}
	sdk.newFunc = reflect.ValueOf(new)
	sdk.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateSDKConnector(sdk)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[sdk.Code]; ok {
		panic("meergo: RegisterSDK called with a connector code already registered: " + sdk.Code)
	}
	registry.sdks[sdk.Code] = sdk
	registry.usedCodes[sdk.Code] = struct{}{}
}

// RegisterWebhook makes a webhook connector available by the provided code. If
// RegisterWebhook is called twice with the same code or if new is nil, it
// panics.
func RegisterWebhook[T any](webhook WebhookSpec, new WebhookNewFunc[T]) {
	if new == nil {
		panic("meergo: new function is nil for connector " + webhook.Code)
	}
	webhook.newFunc = reflect.ValueOf(new)
	webhook.ct = reflect.TypeOf((*T)(nil)).Elem()
	validateWebhookConnector(webhook)
	registry.Lock()
	defer registry.Unlock()
	if _, ok := registry.usedCodes[webhook.Code]; ok {
		panic("meergo: RegisterWebhook called with a connector code already registered: " + webhook.Code)
	}
	registry.webhooks[webhook.Code] = webhook
	registry.usedCodes[webhook.Code] = struct{}{}
}

// RegisteredAPI returns the API registered with the given code. If an API with
// this code is not registered, it panics.
func RegisteredAPI(code string) APISpec {
	registry.Lock()
	api, ok := registry.apis[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown API connector %q (forgotten import?)", code))
	}
	return api
}

// RegisteredDatabase returns the database registered with the given code.
// If a database with this code is not registered, it panics.
func RegisteredDatabase(code string) DatabaseSpec {
	registry.Lock()
	database, ok := registry.databases[code]
	registry.Unlock()
	if !ok {
		panic(fmt.Errorf("meergo: unknown database connector %q (forgotten import?)", code))
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
		panic(fmt.Errorf("meergo: unknown file connector %q (forgotten import?)", code))
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
		panic(fmt.Errorf("meergo: unknown file storage connector %q (forgotten import?)", code))
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
		panic(fmt.Errorf("meergo: unknown message broker connector %q (forgotten import?)", code))
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
		panic(fmt.Errorf("meergo: unknown SDK connector %q (forgotten import?)", code))
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
		panic(fmt.Errorf("meergo: unknown webhook connector %q (forgotten import?)", code))
	}
	return webhook
}

// validateCategories validates the categories of a connector.
func validateCategories(connectorName string, categories Categories) {
	if categories == 0 {
		panic(fmt.Sprintf("connector %s: at least one category must be specified", connectorName))
	}
}

// validateAPIConnector validates the passed API connector, performing checks to
// detect errors that could cause panic or errors in the Meergo code that uses
// the connectors.
//
// In case of a validation error, this function panics.
func validateAPIConnector(api APISpec) {

	validateConnectorCode("API", api.Code)
	validateCategories(api.Code, api.Categories)

	if api.AsSource == nil && api.AsDestination == nil {
		panic(fmt.Sprintf("connector %s: APISpec must include at least the AsSource and AsDestination fields", api.Code))
	}

	if api.AsSource != nil {
		targets := api.AsSource.Targets
		//if targets == 0 || (targets&^(TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^TargetUser) != 0 {
			panic(fmt.Sprintf("connector %s: APISpec.AsSource.Target is not valid; possible value is connectors.TargetUser", api.Code))
		}
		if targets&TargetUser != 0 {
			if !api.ct.Implements(reflect.TypeFor[RecordFetcher]()) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", api.Code))
			}
		}
	}

	if api.AsDestination != nil {
		targets := api.AsDestination.Targets
		//if targets == 0 || (targets&^(TargetEvent|TargetUser|GroupTarget)) != 0 { TODO(marco): Implement groups
		if targets == 0 || (targets&^(TargetEvent|TargetUser)) != 0 {
			panic(fmt.Sprintf("connector %s: APISpec.AsDestination.Target is not valid; possible values are connectors.TargetEvent, connectors.TargetUser, or a combination of them using the bitwise OR operator", api.Code))
		}
		if targets&TargetEvent != 0 {
			if !api.ct.Implements(reflect.TypeFor[EventSender]()) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", api.Code))
			}
			if api.AsDestination.SendingMode == None {
				panic(fmt.Sprintf("connector %s is declared to support Event as destination, but it does not specify a sending mode", api.Code))
			}
		}
		if targets&TargetUser != 0 {
			if !api.ct.Implements(reflect.TypeFor[RecordUpserter]()) {
				panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", api.Code))
			}
		}
	}

	if api.Terms.User != "" || api.Terms.Users != "" {
		if (api.AsSource == nil || api.AsSource.Targets&TargetUser == 0) &&
			(api.AsDestination == nil || api.AsDestination.Targets&TargetUser == 0) {
			panic(fmt.Sprintf("connector %s: cannot specify a term for user and/or users"+
				" if it does not support the User target neither as source nor as destination", api.Code))
		}
	}

	// TODO(marco): Implement groups
	//if api.Terms.Group != "" || api.Terms.Groups != "" {
	//	if (api.AsSource == nil || api.AsSource.Targets&GroupTarget == 0) &&
	//		(api.AsDestination == nil || api.AsDestination.Targets&GroupTarget == 0) {
	//		panic(fmt.Sprintf("connector %s: cannot specify a term for group and/or groups"+
	//			" if it does not support the Group target neither as source nor as destination", api.Name))
	//	}
	//}

	var hasSourceSettings = api.AsSource != nil && api.AsSource.HasSettings
	var hasDestinationSettings = api.AsDestination != nil && api.AsDestination.HasSettings
	if hasSourceSettings || hasDestinationSettings {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if !api.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", api.Code))
		}
	} else {
		iface := reflect.TypeFor[interface {
			ServeUI(ctx context.Context, event string, settings json.Value, role Role) (*UI, error)
		}]()
		if api.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: ServeUI is implemented, but neither APISpec.AsSource.HasSettings nor APISpec.AsDestination.HasSettings is set to true", api.Code))
		}
	}

	if api.OAuth.AuthURL != "" {
		iface := reflect.TypeFor[interface {
			OAuthAccount(ctx context.Context) (string, error)
		}]()
		if !api.ct.Implements(iface) {
			panic(fmt.Sprintf("connector %s: there's a mismatch between the declared functionalities and the methods actually implemented", api.Code))
		}
	}

	// Patterns are checked for validity when rate limiters are created; invalid patterns will cause construction to panic.
	var requireOAuth bool
	for _, group := range api.EndpointGroups {
		requireOAuth = requireOAuth || group.RequireOAuth
		if group.Patterns != nil && len(group.Patterns) == 0 {
			panic(fmt.Sprintf("connector %s: Patterns must be nil or contain at least one pattern", api.Code))
		}
		if group.RateLimit.RequestsPerSecond <= 0 {
			panic(fmt.Sprintf("connector %s: RequestsPerSecond must be > 0", api.Code))
		}
		if group.RateLimit.Burst <= 0 {
			panic(fmt.Sprintf("connector %s: Burst must be > 0", api.Code))
		}
		if group.RateLimit.MaxConcurrentRequests < 0 {
			panic(fmt.Sprintf("connector %s: MaxConcurrentRequests must be >= 0", api.Code))
		}
	}
	if api.OAuth.AuthURL == "" && requireOAuth {
		panic(fmt.Sprintf("connector %s: RequireOAuth cannot be true when OAuth is not supported", api.Code))
	}
	if api.OAuth.AuthURL != "" && !requireOAuth {
		panic(fmt.Sprintf("connector %s: OAuth is supported, but there are no endpoint groups that require it", api.Code))
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
func validateFileConnector(file FileSpec) {

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
func validateFileStorageConnector(fileStorage FileStorageSpec) {

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
		panic(fmt.Sprintf("connector %s: it does not implement the required methods", broker.Code))
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
