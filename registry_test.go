//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"testing"

	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
)

type registrySnapshot struct {
	apps       map[string]AppInfo
	databases  map[string]DatabaseInfo
	files      map[string]FileInfo
	storages   map[string]FileStorageInfo
	sdks       map[string]SDKInfo
	streams    map[string]StreamInfo
	usedCodes  map[string]struct{}
	warehouses map[string]WarehouseDriver
}

func replaceRegistryForTest(t *testing.T) {
	t.Helper()

	registryMu.Lock()
	snapshot := registrySnapshot{
		apps:       maps.Clone(registry.apps),
		databases:  maps.Clone(registry.databases),
		files:      maps.Clone(registry.files),
		storages:   maps.Clone(registry.storages),
		sdks:       maps.Clone(registry.sdks),
		streams:    maps.Clone(registry.streams),
		usedCodes:  maps.Clone(registry.usedCodes),
		warehouses: maps.Clone(registry.warehouses),
	}
	registry.apps = make(map[string]AppInfo)
	registry.databases = make(map[string]DatabaseInfo)
	registry.files = make(map[string]FileInfo)
	registry.storages = make(map[string]FileStorageInfo)
	registry.sdks = make(map[string]SDKInfo)
	registry.streams = make(map[string]StreamInfo)
	registry.usedCodes = make(map[string]struct{})
	registry.warehouses = make(map[string]WarehouseDriver)
	registryMu.Unlock()

	t.Cleanup(func() {
		registryMu.Lock()
		registry.apps = snapshot.apps
		registry.databases = snapshot.databases
		registry.files = snapshot.files
		registry.storages = snapshot.storages
		registry.sdks = snapshot.sdks
		registry.streams = snapshot.streams
		registry.usedCodes = snapshot.usedCodes
		registry.warehouses = snapshot.warehouses
		registryMu.Unlock()
	})
}

func TestValidateConnectorCode(t *testing.T) {
	valid := []string{"a", "abc", "abc-123", "0", "-", "-a", "a-", "a-b-c", "z9-", "12345", "alpha-0-omega", "postgresql", "http-get"}
	invalid := map[string]string{
		"":     "code is missing for a connector of type App",
		"ABC":  "connector code ABC is not valid; valid codes contain only [a-z0-9-]",
		"a_b":  "connector code a_b is not valid; valid codes contain only [a-z0-9-]",
		"a b":  "connector code a b is not valid; valid codes contain only [a-z0-9-]",
		"a.b":  "connector code a.b is not valid; valid codes contain only [a-z0-9-]",
		"a/b":  "connector code a/b is not valid; valid codes contain only [a-z0-9-]",
		"café": "connector code café is not valid; valid codes contain only [a-z0-9-]",
		"ç":    "connector code ç is not valid; valid codes contain only [a-z0-9-]",
		"🙂":    "connector code 🙂 is not valid; valid codes contain only [a-z0-9-]",
	}

	// Valid.
	for _, code := range valid {
		code := code
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
		code, expected := code, expected
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

func TestRegisterAppRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	app := AppInfo{
		Code:       "test-app",
		Label:      "Test App",
		Categories: CategoryTest,
		AsDestination: &AsAppDestination{
			Targets:     TargetEvent,
			SendingMode: Server,
		},
	}
	RegisterApp(app, newTestApp)

	got := RegisteredApp("test-app")
	if got.Code != "test-app" {
		t.Fatalf("expected code test-app, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-app"]; !ok {
		t.Fatalf("expected code test-app to be tracked in used codes")
	}
}

func TestRegisterDatabaseRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	database := DatabaseInfo{
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

	file := FileInfo{
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

func TestRegisterFileStorageRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	storage := FileStorageInfo{
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

func TestRegisterSDKRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	sdk := SDKInfo{
		Code:       "test-sdk",
		Label:      "Test SDK",
		Categories: CategoryTest,
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

func TestRegisterStreamRegistersConnector(t *testing.T) {
	replaceRegistryForTest(t)

	stream := StreamInfo{
		Code:       "test-stream",
		Label:      "Test Stream",
		Categories: CategoryEventStreaming,
	}
	RegisterStream(stream, newTestStream)

	got := RegisteredStream("test-stream")
	if got.Code != "test-stream" {
		t.Fatalf("expected code test-stream, got %s", got.Code)
	}
	if _, ok := registry.usedCodes["test-stream"]; !ok {
		t.Fatalf("expected code test-stream to be tracked in used codes")
	}
}

func TestRegisterConnectorDuplicateCodePanics(t *testing.T) {
	replaceRegistryForTest(t)

	RegisterSDK(SDKInfo{
		Code:       "duplicate",
		Label:      "Duplicate",
		Categories: CategoryTest,
	}, newTestSDK)

	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic when registering duplicate code")
		}
	}()
	RegisterStream(StreamInfo{
		Code:       "duplicate",
		Label:      "Duplicate Stream",
		Categories: CategoryEventStreaming,
	}, newTestStream)
}

func newTestApp(*AppEnv) (testAppConnector, error) {
	return testAppConnector{}, nil
}

type testAppConnector struct{}

func (testAppConnector) EventTypeSchema(context.Context, string) (types.Type, error) {
	return types.Text(), nil
}

func (testAppConnector) EventTypes(context.Context) ([]*EventType, error) {
	return nil, nil
}

func (testAppConnector) PreviewSendEvents(context.Context, Events) (*http.Request, error) {
	return nil, nil
}

func (testAppConnector) SendEvents(context.Context, Events) error {
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

func (testDatabaseConnector) QuoteTime(any, types.Type) string {
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

func newTestSDK(*SDKEnv) (testSDKConnector, error) {
	return testSDKConnector{}, nil
}

type testSDKConnector struct{}

func newTestStream(*StreamEnv) (testStreamConnector, error) {
	return testStreamConnector{}, nil
}

type testStreamConnector struct{}

func (testStreamConnector) Close() error {
	return nil
}

func (testStreamConnector) Receive(context.Context) ([]byte, func(), error) {
	return nil, func() {}, nil
}

func (testStreamConnector) Send(context.Context, []byte, SendOptions, func(error)) error {
	return nil
}
