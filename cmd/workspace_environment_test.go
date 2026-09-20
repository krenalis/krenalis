// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/s3"
	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// TestWorkspaceEnvironment verifies that the immutable workspace environment
// does not limit ordinary connector behavior.
func TestWorkspaceEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	storageDir, err := filepath.Abs("../test/testdata/storage")
	if err != nil {
		t.Fatalf("expected storage directory, got %v", err)
	}
	k := krenalistester.NewKrenalisInstance(t)
	k.SetFileSystemRoot(storageDir)
	k.Start()
	defer k.Stop()

	var production struct {
		ID          string `json:"id"`
		Environment string `json:"environment"`
	}
	k.Call("GET", "/v1/workspaces/current", nil, nil, &production)
	if production.Environment != "production" {
		t.Fatalf("expected default production environment, got %q", production.Environment)
	}
	assertEnvironmentBadRequest(t, k.TryCall("PUT", "/v1/workspaces/current", nil, map[string]any{
		"name": "production", "environment": "development",
	}, nil), "environment is immutable")
	k.Call("PUT", "/v1/workspaces/current", nil, map[string]any{"name": "production-renamed"}, nil)
	k.Call("GET", "/v1/workspaces/current", nil, nil, &production)
	if production.Environment != "production" {
		t.Fatalf("expected preserved production environment, got %q", production.Environment)
	}

	for _, value := range []any{nil, "preview", 1} {
		assertEnvironmentBadRequest(t, k.TryCall("POST", "/v1/workspaces/test", http.Header{"Krenalis-Workspace": nil}, map[string]any{
			"environment": value,
		}, nil), "environment")
	}
	assertEnvironmentBadRequest(t, k.TryCall("POST", "/v1/workspaces/test", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"synthetic": true,
	}, nil), "synthetic is not supported")

	settings := &krenalistester.DBSettings{}
	err = json.Unmarshal(krenalistester.PostgresWarehouseSettings(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL settings, got %v", err)
	}
	database := "test_workspace_environment_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	pool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected PostgreSQL connection pool, got %v", err)
	}
	_, err = pool.Exec(t.Context(), "CREATE DATABASE "+database)
	pool.Close()
	if err != nil {
		t.Fatalf("expected isolated warehouse database, got %v", err)
	}
	cleanupSettings := *settings
	defer func() {

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		pool, err := krenalistester.ConnectionPool(ctx, &cleanupSettings)
		if err != nil {
			t.Errorf("expected PostgreSQL connection pool for cleanup, got %v", err)
			return
		}
		defer pool.Close()
		_, err = pool.Exec(ctx, "DROP DATABASE "+database)
		if err != nil {
			t.Errorf("expected isolated warehouse database cleanup, got %v", err)
		}
	}()
	settings.Database = database

	profileSchema := types.Object([]types.Property{{
		Name: "email", Type: types.String().AsEmail().WithMaxLength(254), ReadOptional: true,
	}})
	request := map[string]any{
		"name": "development", "environment": "development", "profileSchema": profileSchema,
		"warehouse": map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": settings},
	}
	var created struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, request, &created)
	k.SetWorkspaceID(created.ID)

	var development struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Environment string `json:"environment"`
	}
	k.Call("GET", "/v1/workspaces/current", nil, nil, &development)
	if development.Environment != "development" {
		t.Fatalf("expected development environment, got %q", development.Environment)
	}
	assertEnvironmentBadRequest(t, k.TryCall("PUT", "/v1/workspaces/current", nil, map[string]any{
		"name": "development", "environment": "production",
	}, nil), "environment is immutable")
	k.Call("PUT", "/v1/workspaces/current", nil, map[string]any{"name": "development-renamed"}, nil)
	k.Call("GET", "/v1/workspaces/current", nil, nil, &development)
	if development.Name != "development-renamed" || development.Environment != "development" {
		t.Fatalf("expected preserved development environment, got %+v", development)
	}

	var list struct {
		Workspaces []struct {
			ID          string `json:"id"`
			Environment string `json:"environment"`
		} `json:"workspaces"`
	}
	k.Call("GET", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, nil, &list)
	environments := map[string]string{}
	for _, workspace := range list.Workspaces {
		environments[workspace.ID] = workspace.Environment
	}
	if environments[production.ID] != "production" || environments[created.ID] != "development" {
		t.Fatalf("expected persisted workspace environments, got %v", environments)
	}

	var catalog struct {
		Connectors []struct {
			Code          string    `json:"code"`
			AsSource      *struct{} `json:"asSource"`
			AsDestination *struct{} `json:"asDestination"`
		} `json:"connectors"`
	}
	k.Call("GET", "/v1/connectors", nil, nil, &catalog)
	if len(catalog.Connectors) <= 2 {
		t.Fatalf("expected full connector catalog, got %d entries", len(catalog.Connectors))
	}
	for _, connector := range []struct {
		code          string
		asSource      bool
		asDestination bool
	}{
		{code: "filesystem", asSource: true, asDestination: true},
		{code: "s3", asSource: true, asDestination: true},
		{code: "csv", asSource: true, asDestination: true},
	} {
		if !hasConnector(catalog.Connectors, connector.code, connector.asSource, connector.asDestination) {
			t.Fatalf("expected connector %+v in full catalog, got %+v", connector, catalog.Connectors)
		}
	}

	filesystem := k.CreateSourceFileSystem()
	rows, _ := k.File(filesystem, "users.csv", "csv", "", krenalistester.NoCompression,
		krenalistester.JSONEncodeSettings(map[string]any{"separator": ",", "hasColumnNames": true}), 100)
	if len(rows) != 2 {
		t.Fatalf("expected 2 FileSystem rows in development, got %d", len(rows))
	}
	k.CreateDestinationFilesystem()

	s3 := k.CreateConnection(krenalistester.ConnectionToCreate{Name: "S3", Role: krenalistester.Source,
		Connector: "s3", Settings: krenalistester.JSONEncodeSettings(map[string]any{
			"accessKeyID": "AAAAAAAAAAAAAAAAAAAA", "secretAccessKey": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
			"region": "us-east-1", "bucket": "demo",
		})})
	if path := k.AbsolutePath(s3, "customers-a.csv"); path != "s3://demo/customers-a.csv" {
		t.Fatalf("expected normal S3 absolute path, got %q", path)
	}
}

func assertEnvironmentBadRequest(t *testing.T, err error, message string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected HTTP 400, got success")
	}
	status, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected HTTP status error, got %T", err)
	}
	if status.Response.Code != http.StatusBadRequest {
		t.Fatalf("expected HTTP 400, got %d: %s", status.Response.Code, status.Response.Text)
	}
	if !strings.Contains(status.Response.Text, message) {
		t.Fatalf("expected error containing %q, got %q", message, status.Response.Text)
	}
}

func hasConnector(connectors []struct {
	Code          string    `json:"code"`
	AsSource      *struct{} `json:"asSource"`
	AsDestination *struct{} `json:"asDestination"`
}, code string, asSource, asDestination bool) bool {
	for _, connector := range connectors {
		if connector.Code == code && (connector.AsSource != nil) == asSource && (connector.AsDestination != nil) == asDestination {
			return true
		}
	}
	return false
}
