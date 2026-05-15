// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"testing"

	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

type registrySnapshot struct {
	applications   map[string]ApplicationSpec
	databases      map[string]DatabaseSpec
	files          map[string]FileSpec
	storages       map[string]FileStorageSpec
	messageBrokers map[string]MessageBrokerSpec
	sdks           map[string]SDKSpec
	webhooks       map[string]WebhookSpec
	usedCodes      map[string]struct{}
}

func replaceRegistryForTest(t *testing.T) {
	t.Helper()

	registry.Lock()
	snapshot := registrySnapshot{
		applications:   maps.Clone(registry.applications),
		databases:      maps.Clone(registry.databases),
		files:          maps.Clone(registry.files),
		storages:       maps.Clone(registry.storages),
		messageBrokers: maps.Clone(registry.messageBrokers),
		sdks:           maps.Clone(registry.sdks),
		webhooks:       maps.Clone(registry.webhooks),
		usedCodes:      maps.Clone(registry.usedCodes),
	}
	registry.applications = make(map[string]ApplicationSpec)
	registry.databases = make(map[string]DatabaseSpec)
	registry.files = make(map[string]FileSpec)
	registry.storages = make(map[string]FileStorageSpec)
	registry.messageBrokers = make(map[string]MessageBrokerSpec)
	registry.sdks = make(map[string]SDKSpec)
	registry.webhooks = make(map[string]WebhookSpec)
	registry.usedCodes = make(map[string]struct{})
	registry.Unlock()

	t.Cleanup(func() {
		registry.Lock()
		registry.applications = snapshot.applications
		registry.databases = snapshot.databases
		registry.files = snapshot.files
		registry.storages = snapshot.storages
		registry.messageBrokers = snapshot.messageBrokers
		registry.sdks = snapshot.sdks
		registry.webhooks = snapshot.webhooks
		registry.usedCodes = snapshot.usedCodes
		registry.Unlock()
	})
}

func TestValidateConnectorCode(t *testing.T) {
	valid := []string{"a", "abc", "abc-123", "0", "-", "-a", "a-", "a-b-c", "z9-", "12345", "alpha-0-omega", "postgresql", "http-get"}
	invalid := map[string]string{
		"":     "krenalis/connectors: code is missing for a connector of type App",
		"ABC":  `krenalis/connectors: connector code "ABC" is not valid; valid codes contain only [a-z0-9-]`,
		"a_b":  `krenalis/connectors: connector code "a_b" is not valid; valid codes contain only [a-z0-9-]`,
		"a b":  `krenalis/connectors: connector code "a b" is not valid; valid codes contain only [a-z0-9-]`,
		"a.b":  `krenalis/connectors: connector code "a.b" is not valid; valid codes contain only [a-z0-9-]`,
		"a/b":  `krenalis/connectors: connector code "a/b" is not valid; valid codes contain only [a-z0-9-]`,
		"café": `krenalis/connectors: connector code "café" is not valid; valid codes contain only [a-z0-9-]`,
		"ç":    `krenalis/connectors: connector code "ç" is not valid; valid codes contain only [a-z0-9-]`,
		"🙂":    `krenalis/connectors: connector code "🙂" is not valid; valid codes contain only [a-z0-9-]`,
	}

	// Valid.
	for _, code := range valid {
		t.Run(fmt.Sprintf("valid_%q", code), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic for %q, got %v", code, r)
				}
			}()
			validateConnectorCode("App", code)
		})
	}

	// Invalid.
	for code, expected := range invalid {
		t.Run(fmt.Sprintf("invalid_%q", code), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("expected panic %q, got none", expected)
				} else if r != expected {
					t.Fatalf("expected %q, got %q", expected, r)
				}
			}()
			validateConnectorCode("App", code)
		})
	}
}

func TestRegisterApplicationConnector(t *testing.T) {
	replaceRegistryForTest(t)

	app := ApplicationSpec{
		Code:       "test-application",
		Label:      "Test Application",
		Categories: CategorySaaS,
		AsDestination: &AsApplicationDestination{
			Targets:     TargetEvent,
			SendingMode: Server,
		},
	}
	RegisterApplication(app, newTestApplication)

	got := RegisteredApplication("test-application")
	if got.Code != "test-application" {
		t.Fatalf("expected code test-application, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-application"]; !ok {
		t.Fatalf("expected code test-application to be tracked in used codes")
	}
}

func TestRegisterDatabaseRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	database := DatabaseSpec{
		Code:       "test-database",
		Label:      "Test Database",
		Categories: CategoryDatabase,
	}
	RegisterDatabase(database, newTestDatabase)

	got := RegisteredDatabase("test-database")
	if got.Code != "test-database" {
		t.Fatalf("expected code test-database, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-database"]; !ok {
		t.Fatalf("expected code test-database to be tracked in used codes")
	}
}

func TestRegisterFileRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	file := FileSpec{
		Code:          "test-file",
		Label:         "Test File",
		Categories:    CategoryFile,
		AsDestination: &AsDestinationFile{},
	}
	RegisterFile(file, newTestFile)

	got := RegisteredFile("test-file")
	if got.Code != "test-file" {
		t.Fatalf("expected code test-file, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-file"]; !ok {
		t.Fatalf("expected code test-file to be tracked in used codes")
	}
}

// TestRegisterSourceFileReadVariants verifies the supported Read signatures for
// source file connectors.
func TestRegisterSourceFileReadVariants(t *testing.T) {
	testCases := []struct {
		name      string
		code      string
		register  func(FileSpec)
		wantPanic bool
	}{
		{
			name: "ReadReader",
			code: "test-source-file-reader",
			register: func(file FileSpec) {
				RegisterFile(file, func(*FileEnv) (testSourceFileReaderConnector, error) {
					return testSourceFileReaderConnector{}, nil
				})
			},
		},
		{
			name: "ReadSeeker",
			code: "test-source-file-read-seeker",
			register: func(file FileSpec) {
				RegisterFile(file, func(*FileEnv) (testSourceFileReadSeekerConnector, error) {
					return testSourceFileReadSeekerConnector{}, nil
				})
			},
		},
		{
			name: "MissingRead",
			code: "test-source-file-without-read",
			register: func(file FileSpec) {
				RegisterFile(file, newTestFile)
			},
			wantPanic: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			replaceRegistryForTest(t)
			file := FileSpec{
				Code:       tc.code,
				Label:      tc.name,
				Categories: CategoryFile,
				AsSource:   &AsSourceFile{},
			}
			if tc.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Fatal("expected panic when registering source file without a supported Read method")
					}
				}()
			}
			tc.register(file)
			if tc.wantPanic {
				return
			}
			got := RegisteredFile(file.Code)
			if got.Code != file.Code {
				t.Fatalf("expected code %s, got %s", file.Code, got.Code)
			}
		})
	}
}

func TestRegisterFileStorageRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	storage := FileStorageSpec{
		Code:          "test-file-storage",
		Label:         "Test File Storage",
		Categories:    CategoryFileStorage,
		AsDestination: &AsFileStorageDestination{},
	}
	RegisterFileStorage(storage, newTestFileStorage)

	got := RegisteredFileStorage("test-file-storage")
	if got.Code != "test-file-storage" {
		t.Fatalf("expected code test-file-storage, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-file-storage"]; !ok {
		t.Fatalf("expected code test-file-storage to be tracked in used codes")
	}
}

func TestRegisterMessageBrokerRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	broker := MessageBrokerSpec{
		Code:       "test-broker",
		Label:      "Test Message Broker",
		Categories: CategoryMessageBroker,
	}
	RegisterMessageBroker(broker, newTestMessageBroker)

	got := RegisteredMessageBroker("test-broker")
	if got.Code != "test-broker" {
		t.Fatalf("expected code test-broker, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-broker"]; !ok {
		t.Fatalf("expected code test-broker to be tracked in used codes")
	}
}

func TestRegisterSDKRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	sdk := SDKSpec{
		Code:       "test-sdk",
		Label:      "Test SDK",
		Categories: CategorySDK,
	}
	RegisterSDK(sdk, newTestSDK)

	got := RegisteredSDK("test-sdk")
	if got.Code != "test-sdk" {
		t.Fatalf("expected code test-sdk, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-sdk"]; !ok {
		t.Fatalf("expected code test-sdk to be tracked in used codes")
	}
}

func TestRegisterWebhookRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	webhook := WebhookSpec{
		Code:       "test-webhook",
		Label:      "Test Webhook",
		Categories: CategoryWebhook,
	}
	RegisterWebhook(webhook, newTestWebhook)

	got := RegisteredWebhook("test-webhook")
	if got.Code != "test-webhook" {
		t.Fatalf("expected code test-webhook, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-webhook"]; !ok {
		t.Fatalf("expected code test-webhook to be tracked in used codes")
	}
}

func TestRegisterConnectorDuplicateCodePanics(t *testing.T) {
	replaceRegistryForTest(t)

	RegisterSDK(SDKSpec{
		Code:       "duplicate",
		Label:      "Duplicate",
		Categories: CategorySDK,
	}, newTestSDK)

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when registering duplicate code")
		}
	}()
	RegisterMessageBroker(MessageBrokerSpec{
		Code:       "duplicate",
		Label:      "Duplicate Message Broker",
		Categories: CategoryMessageBroker,
	}, newTestMessageBroker)
}

func newTestApplication(*ApplicationEnv) (testApplicationConnector, error) {
	return testApplicationConnector{}, nil
}

type testApplicationConnector struct{}

func (testApplicationConnector) EventTypeSchema(context.Context, string) (types.Type, error) {
	return types.String(), nil
}

func (testApplicationConnector) EventTypes(context.Context) ([]*EventType, error) {
	return nil, nil
}

func (testApplicationConnector) PreviewSendEvents(context.Context, Events) (*http.Request, error) {
	return nil, nil
}

func (testApplicationConnector) SendEvents(context.Context, Events) error {
	return nil
}

func newTestDatabase(*DatabaseEnv) (testDatabaseConnector, error) {
	return testDatabaseConnector{}, nil
}

type testDatabaseConnector struct{}

func (testDatabaseConnector) Close() error {
	return nil
}

func (testDatabaseConnector) Columns(context.Context, string) ([]Column, error) {
	return nil, nil
}

func (testDatabaseConnector) Merge(context.Context, Table, [][]any) error {
	return nil
}

func (testDatabaseConnector) Query(context.Context, string) (Rows, []Column, error) {
	return nil, nil, nil
}

func (testDatabaseConnector) SQLLiteral(any, types.Type) string {
	return ""
}

func (testDatabaseConnector) ServeUI(context.Context, string, json.Value, Role) (*UI, error) {
	return nil, nil
}

func newTestFile(*FileEnv) (testFileConnector, error) {
	return testFileConnector{}, nil
}

type testFileConnector struct{}

func (testFileConnector) Write(context.Context, io.Writer, string, RecordReader) error {
	return nil
}

func (testFileConnector) ContentType(context.Context) string {
	return "text/plain"
}

// testSourceFileReaderConnector reads from an io.Reader.
type testSourceFileReaderConnector struct{}

// Read implements the source file reader signature.
func (testSourceFileReaderConnector) Read(context.Context, io.Reader, string, RecordWriter) error {
	return nil
}

// testSourceFileReadSeekerConnector reads from an io.ReadSeeker.
type testSourceFileReadSeekerConnector struct{}

// Read implements the source file read seeker signature.
func (testSourceFileReadSeekerConnector) Read(context.Context, io.ReadSeeker, string, RecordWriter) error {
	return nil
}

func newTestFileStorage(*FileStorageEnv) (testFileStorageConnector, error) {
	return testFileStorageConnector{}, nil
}

type testFileStorageConnector struct{}

func (testFileStorageConnector) AbsolutePath(context.Context, string) (string, error) {
	return "", nil
}

func (testFileStorageConnector) ServeUI(context.Context, string, json.Value, Role) (*UI, error) {
	return nil, nil
}

func (testFileStorageConnector) Write(context.Context, io.Reader, string, string) error {
	return nil
}

func newTestMessageBroker(*MessageBrokerEnv) (testMessageBrokerConnector, error) {
	return testMessageBrokerConnector{}, nil
}

type testMessageBrokerConnector struct{}

func (testMessageBrokerConnector) Close() error {
	return nil
}

func (testMessageBrokerConnector) Receive(context.Context) ([]byte, func(), error) {
	return nil, func() {}, nil
}

func (testMessageBrokerConnector) Send(context.Context, []byte, SendOptions, func(error)) error {
	return nil
}

func newTestSDK(*SDKEnv) (testSDKConnector, error) {
	return testSDKConnector{}, nil
}

type testSDKConnector struct{}

func newTestWebhook(*WebhookEnv) (testWebhookConnector, error) {
	return testWebhookConnector{}, nil
}

type testWebhookConnector struct{}
