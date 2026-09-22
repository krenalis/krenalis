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
	stopped := false
	defer func() {
		if !stopped {
			k.Stop()
		}
	}()

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
		_, err = pool.Exec(ctx, "DROP DATABASE "+database+" WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse database cleanup, got %v", err)
		}
		if !stopped {
			k.Stop()
			stopped = true
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

	accountRequest := map[string]any{
		"name": "Empty account", "userCount": 0,
		"duplicateRecordPercent": 0, "countries": map[string]int{"IT": 40},
	}
	k.SetWorkspaceID(production.ID)
	assertHTTPError(t, k.TryCall("POST", "/v1/simulated-accounts", nil, accountRequest, nil), http.StatusUnprocessableEntity,
		"SimulatedAccountsRequireDevelopment")
	var productionAccountCount int
	k.QueryRowTestDatabase(t.Context(), &productionAccountCount, "SELECT COUNT(*) FROM simulated_accounts WHERE workspace = $1", production.ID)
	if productionAccountCount != 0 {
		t.Fatalf("expected no simulated accounts in production, got %d", productionAccountCount)
	}

	k.SetWorkspaceID(created.ID)
	apiKey := k.CreateWorkspaceRestrictedAPIKey("simulated account test")
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated-accounts", http.Header{
		"Authorization": {"Bearer " + apiKey},
	}, nil, nil), http.StatusUnauthorized, "")
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated-accounts", http.Header{
		"Krenalis-Workspace": nil,
	}, nil, nil), http.StatusForbidden, "Krenalis-Workspace header is missing")

	var account struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, accountRequest, &account)
	var detail struct {
		ID                      string         `json:"id"`
		Name                    string         `json:"name"`
		Status                  string         `json:"status"`
		UserCount               int            `json:"userCount"`
		DuplicateRecordPercent  float64        `json:"duplicateRecordPercent"`
		Countries               map[string]int `json:"countries"`
		GenerationPolicyVersion string         `json:"generationPolicyVersion"`
		GeneratedRecordCount    int            `json:"generatedRecordCount"`
		GenerationError         string         `json:"generationError"`
	}
	k.Call("GET", "/v1/simulated-accounts/"+account.ID, nil, nil, &detail)
	if detail.ID != account.ID || detail.Name != "Empty account" || detail.Status != "Ready" ||
		detail.UserCount != 0 || detail.DuplicateRecordPercent != 0 || detail.Countries["IT"] != 40 ||
		detail.GenerationPolicyVersion != "empty-v1" || detail.GeneratedRecordCount != 0 || detail.GenerationError != "" {
		t.Fatalf("expected empty ready simulated account metadata, got %+v", detail)
	}
	var detailFields map[string]json.Value
	k.Call("GET", "/v1/simulated-accounts/"+account.ID, nil, nil, &detailFields)
	if _, ok := detailFields["connector"]; ok {
		t.Fatalf("expected simulated account response without connector, got %s", detailFields["connector"])
	}
	var accounts struct {
		Accounts []struct {
			ID string `json:"id"`
		} `json:"simulatedAccounts"`
	}
	k.Call("GET", "/v1/simulated-accounts", nil, nil, &accounts)
	if len(accounts.Accounts) != 1 || accounts.Accounts[0].ID != account.ID {
		t.Fatalf("expected one listed simulated account, got %+v", accounts.Accounts)
	}
	k.Call("PUT", "/v1/simulated-accounts/"+account.ID, nil, map[string]string{"name": "Renamed account"}, nil)
	k.Call("GET", "/v1/simulated-accounts/"+account.ID, nil, nil, &detail)
	if detail.Name != "Renamed account" {
		t.Fatalf("expected renamed simulated account, got %q", detail.Name)
	}

	k.SetWorkspaceID(production.ID)
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated-accounts/"+account.ID, nil, nil, nil), http.StatusNotFound, "")
	k.SetWorkspaceID(created.ID)

	warehousePool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected simulated account warehouse pool, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), `INSERT INTO krenalis_simulated_account_records
		(simulated_account_id, external_id, data) VALUES ($1, 'account-record', '{}'::jsonb),
		('other-account', 'other-record', '{}'::jsonb)`, account.ID)
	if err != nil {
		warehousePool.Close()
		t.Fatalf("expected simulated account warehouse fixture, got %v", err)
	}
	k.Call("DELETE", "/v1/simulated-accounts/"+account.ID, nil, nil, nil)
	var accountRecords, otherRecords int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records WHERE simulated_account_id = $1", account.ID).Scan(&accountRecords)
	if err != nil {
		warehousePool.Close()
		t.Fatalf("expected deleted account record count, got %v", err)
	}
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_account_records WHERE simulated_account_id = 'other-account'").Scan(&otherRecords)
	warehousePool.Close()
	if err != nil {
		t.Fatalf("expected preserved account record count, got %v", err)
	}
	if accountRecords != 0 || otherRecords != 1 {
		t.Fatalf("expected account cleanup isolation, got account=%d other=%d", accountRecords, otherRecords)
	}

	var secondAccount struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated-accounts", nil, map[string]any{
		"name": "Failed cleanup", "userCount": 0,
	}, &secondAccount)
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_accounts SET status = 'Preparing' WHERE id = $1", secondAccount.ID)
	assertHTTPError(t, k.TryCall("DELETE", "/v1/simulated-accounts/"+secondAccount.ID, nil, nil, nil), http.StatusUnprocessableEntity,
		"SimulatedAccountPreparing")
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_accounts SET status = 'Ready' WHERE id = $1", secondAccount.ID)
	warehousePool, err = krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected simulated account warehouse pool, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TABLE krenalis_simulated_account_records")
	warehousePool.Close()
	if err != nil {
		t.Fatalf("expected simulated account warehouse fixture removal, got %v", err)
	}
	assertHTTPError(t, k.TryCall("DELETE", "/v1/simulated-accounts/"+secondAccount.ID, nil, nil, nil), http.StatusServiceUnavailable, "")
	var metadataCount int
	k.QueryRowTestDatabase(t.Context(), &metadataCount, "SELECT COUNT(*) FROM simulated_accounts WHERE id = $1", secondAccount.ID)
	if metadataCount != 1 {
		t.Fatalf("expected metadata to survive failed cleanup, got %d rows", metadataCount)
	}
	k.Call("POST", "/v1/warehouse/repair", nil, nil, nil)
	k.Call("DELETE", "/v1/simulated-accounts/"+secondAccount.ID, nil, nil, nil)
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

func assertHTTPError(t *testing.T, err error, code int, message string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected HTTP %d, got success", code)
	}
	status, ok := err.(*krenalistester.StatusCodeError)
	if !ok {
		t.Fatalf("expected HTTP status error, got %T", err)
	}
	if status.Response.Code != code {
		t.Fatalf("expected HTTP %d, got %d: %s", code, status.Response.Code, status.Response.Text)
	}
	if message != "" && !strings.Contains(status.Response.Text, message) {
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
