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
	if err != nil {
		pool.Close()
		t.Fatalf("expected isolated warehouse database, got %v", err)
	}
	var productionRecordsTable, productionWritersTable bool
	err = pool.QueryRow(t.Context(), "SELECT to_regclass('krenalis_simulated_system_records') IS NOT NULL, "+
		"to_regclass('krenalis_simulated_system_writers') IS NOT NULL").Scan(&productionRecordsTable, &productionWritersTable)
	if err != nil {
		pool.Close()
		t.Fatalf("expected production simulated system records table query, got %v", err)
	}
	if productionRecordsTable || productionWritersTable {
		pool.Close()
		t.Fatal("expected production workspace not to create simulated system records table, got present")
	}
	pool.Close()
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
		"name": "development", "environment": "development", "profileSchema": krenalistester.ProfileSchema{Schema: profileSchema},
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
	pool, err = krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected development warehouse connection pool, got %v", err)
	}
	var developmentRecordsTable, developmentWritersTable bool
	err = pool.QueryRow(t.Context(), "SELECT to_regclass('krenalis_simulated_system_records') IS NOT NULL, "+
		"to_regclass('krenalis_simulated_system_writers') IS NOT NULL").Scan(&developmentRecordsTable, &developmentWritersTable)
	pool.Close()
	if err != nil {
		t.Fatalf("expected development simulated system records table query, got %v", err)
	}
	if !developmentRecordsTable || !developmentWritersTable {
		t.Fatal("expected development workspace to create simulated system records table, got absent")
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
	assertHTTPError(t, k.TryCall("POST", "/v1/simulated/systems", nil, accountRequest, nil), http.StatusUnprocessableEntity,
		"SimulatedSystemsRequireDevelopment")
	var productionAccountCount int
	k.QueryRowTestDatabase(t.Context(), &productionAccountCount, "SELECT COUNT(*) FROM simulated_systems WHERE workspace = $1", production.ID)
	if productionAccountCount != 0 {
		t.Fatalf("expected no simulated systems in production, got %d", productionAccountCount)
	}

	var productionCatalog struct {
		Connectors []struct {
			Code string `json:"code"`
		} `json:"connectors"`
	}
	k.SetWorkspaceID(production.ID)
	k.Call("GET", "/v1/connectors", nil, nil, &productionCatalog)
	for _, connector := range productionCatalog.Connectors {
		if connector.Code == "simulated-application" {
			t.Fatal("expected Simulated Application to be absent from the production catalog, got present")
		}
	}
	assertHTTPError(t, k.TryCall("POST", "/v1/ui", nil, map[string]any{
		"connector": "simulated-application", "event": "load", "role": "Source",
	}, nil), http.StatusUnprocessableEntity, "SimulatedSystemsRequireDevelopment")
	assertHTTPError(t, k.TryCall("POST", "/v1/ui-event", nil, map[string]any{
		"connector": "simulated-application", "event": "refresh-accounts", "role": "Source",
		"settings": map[string]any{"account": ""},
	}, nil), http.StatusUnprocessableEntity, "SimulatedSystemsRequireDevelopment")
	_, err = k.TryCreateConnection(krenalistester.ConnectionToCreate{
		Name: "Production simulated application", Role: krenalistester.Source,
		Connector: "simulated-application", Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": ""}),
	})
	assertHTTPError(t, err, http.StatusUnprocessableEntity, "SimulatedSystemsRequireDevelopment")

	k.SetWorkspaceID(created.ID)
	apiKey := k.CreateWorkspaceRestrictedAPIKey("simulated system test")
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated/systems", http.Header{
		"Authorization": {"Bearer " + apiKey},
	}, nil, nil), http.StatusUnauthorized, "")
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated/systems", http.Header{
		"Krenalis-Workspace": nil,
	}, nil, nil), http.StatusForbidden, "Krenalis-Workspace header is missing")

	var account struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, accountRequest, &account)
	var detail struct {
		ID                     string         `json:"id"`
		Name                   string         `json:"name"`
		Status                 string         `json:"status"`
		UserCount              int            `json:"userCount"`
		DuplicateRecordPercent float64        `json:"duplicateRecordPercent"`
		Countries              map[string]int `json:"countries"`
		SourceConnection       *string        `json:"sourceConnection"`
		DestinationConnection  *string        `json:"destinationConnection"`
	}
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.ID != account.ID || detail.Name != "Empty account" || detail.Status != "Available" ||
		detail.UserCount != 0 || detail.DuplicateRecordPercent != 0 || detail.Countries["IT"] != 40 ||
		detail.SourceConnection != nil || detail.DestinationConnection != nil {
		t.Fatalf("expected empty available simulated system metadata, got %+v", detail)
	}
	var detailFields map[string]json.Value
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detailFields)
	for _, field := range []string{"connector", "generationPolicyVersion", "generationError", "generationCheckpoint", "claimedRun", "claimPingTime", "writerGeneration"} {
		if _, ok := detailFields[field]; ok {
			t.Fatalf("expected simulated system response without %s, got %s", field, detailFields[field])
		}
	}
	for _, field := range []string{"sourceConnection", "destinationConnection"} {
		if value, ok := detailFields[field]; !ok || value.String() != "null" {
			t.Fatalf("expected %s to be null, got %s", field, value)
		}
	}
	var accounts struct {
		Accounts []struct {
			ID                    string  `json:"id"`
			Status                string  `json:"status"`
			SourceConnection      *string `json:"sourceConnection"`
			DestinationConnection *string `json:"destinationConnection"`
		} `json:"simulatedSystems"`
	}
	k.Call("GET", "/v1/simulated/systems", nil, nil, &accounts)
	if len(accounts.Accounts) != 1 || accounts.Accounts[0].ID != account.ID ||
		accounts.Accounts[0].SourceConnection != nil || accounts.Accounts[0].DestinationConnection != nil {
		t.Fatalf("expected one listed simulated system, got %+v", accounts.Accounts)
	}

	var developmentCatalog struct {
		Connectors []struct {
			Code     string `json:"code"`
			AsSource *struct {
				Targets     []string `json:"targets"`
				HasSettings bool     `json:"hasSettings"`
			} `json:"asSource"`
			AsDestination *struct {
				Targets     []string `json:"targets"`
				HasSettings bool     `json:"hasSettings"`
				SendingMode string   `json:"sendingMode"`
			} `json:"asDestination"`
		} `json:"connectors"`
	}
	k.Call("GET", "/v1/connectors", nil, nil, &developmentCatalog)
	var simulatedApplicationFound bool
	for _, connector := range developmentCatalog.Connectors {
		if connector.Code != "simulated-application" {
			continue
		}
		simulatedApplicationFound = true
		if connector.AsSource == nil || len(connector.AsSource.Targets) != 1 || connector.AsSource.Targets[0] != "User" ||
			!connector.AsSource.HasSettings || connector.AsDestination == nil || len(connector.AsDestination.Targets) != 1 ||
			connector.AsDestination.Targets[0] != "User" || !connector.AsDestination.HasSettings || connector.AsDestination.SendingMode != "Server" {
			t.Fatalf("expected Simulated Application capabilities, got %+v", connector)
		}
	}
	if !simulatedApplicationFound {
		t.Fatal("expected Simulated Application in the development catalog, got absent")
	}

	var simulatedApplicationUI struct {
		Fields []struct {
			Name    string `json:"name"`
			Options []struct {
				Value string `json:"value"`
			} `json:"options"`
		} `json:"fields"`
	}
	k.Call("POST", "/v1/ui", nil, map[string]any{
		"connector": "simulated-application", "event": "load", "role": "Source",
	}, &simulatedApplicationUI)
	if len(simulatedApplicationUI.Fields) != 1 || simulatedApplicationUI.Fields[0].Name != "account" ||
		len(simulatedApplicationUI.Fields[0].Options) != 1 || simulatedApplicationUI.Fields[0].Options[0].Value != account.ID {
		t.Fatalf("expected one available simulated system option, got %+v", simulatedApplicationUI)
	}
	k.Call("POST", "/v1/ui", nil, map[string]any{
		"connector": "simulated-application", "event": "save", "role": "Source",
		"settings": map[string]any{"account": account.ID},
	}, nil)

	simulatedConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Simulated Application", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.SourceConnection == nil || *detail.SourceConnection != simulatedConnection || detail.DestinationConnection != nil {
		t.Fatalf("expected account with source connection %s and no destination, got %+v", simulatedConnection, detail)
	}
	var accountOptions struct {
		Fields []struct {
			Options []struct {
				Value    string `json:"value"`
				Disabled bool   `json:"disabled"`
			} `json:"options"`
		} `json:"fields"`
	}
	for _, role := range []struct {
		name     string
		disabled bool
	}{{"Source", true}, {"Destination", false}} {
		k.Call("POST", "/v1/ui", nil, map[string]any{
			"connector": "simulated-application", "event": "load", "role": role.name,
		}, &accountOptions)
		if len(accountOptions.Fields) != 1 || len(accountOptions.Fields[0].Options) != 1 ||
			accountOptions.Fields[0].Options[0].Value != account.ID ||
			accountOptions.Fields[0].Options[0].Disabled != role.disabled {
			t.Fatalf("expected %s account option disabled=%t, got %+v", role.name, role.disabled, accountOptions)
		}
	}
	_, err = k.TryCreateConnection(krenalistester.ConnectionToCreate{
		Name: "Duplicate source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	assertHTTPError(t, err, http.StatusUnprocessableEntity, "InvalidSettings")
	var simulatedConnectionDetail map[string]any
	k.Call("GET", "/v1/connections/"+simulatedConnection, nil, nil, &simulatedConnectionDetail)
	if value, ok := simulatedConnectionDetail["simulatedSystem"]; ok {
		t.Fatalf("expected simulated system binding to be absent from connection response, got %v", value)
	}
	var boundAccount string
	k.QueryRowTestDatabase(t.Context(), &boundAccount,
		"SELECT simulated_system FROM connections WHERE id = $1", simulatedConnection)
	if boundAccount != account.ID {
		t.Fatalf("expected structured Simulated Application binding %q, got %q", account.ID, boundAccount)
	}

	var newAccount struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "Additional account", "userCount": 0,
	}, &newAccount)
	_, err = k.TryCreateConnection(krenalistester.ConnectionToCreate{
		Name: strings.Repeat("x", 101), Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": newAccount.ID}),
	})
	assertHTTPError(t, err, http.StatusBadRequest, "name")
	newConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Simulated Application new account", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": newAccount.ID}),
	})
	k.QueryRowTestDatabase(t.Context(), &boundAccount,
		"SELECT simulated_system FROM connections WHERE id = $1", newConnection)
	if boundAccount != newAccount.ID {
		t.Fatalf("expected new connection to bind account %q, got %q", newAccount.ID, boundAccount)
	}

	var secondSelectableAccount struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{"name": "Second account", "userCount": 0}, &secondSelectableAccount)
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+simulatedConnection, nil, map[string]any{
		"name": "Simulated Application", "simulatedSystem": secondSelectableAccount.ID,
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+simulatedConnection, nil, map[string]any{
		"name": "Simulated Application", "simulatedSystem": account.ID,
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	var authoritativeUI struct {
		Fields []struct {
			ComponentType string `json:"componentType"`
			Text          string `json:"text"`
		} `json:"fields"`
		Buttons []any `json:"buttons"`
	}
	k.Call("GET", "/v1/connections/"+simulatedConnection+"/ui", nil, nil, &authoritativeUI)
	if len(authoritativeUI.Fields) != 1 || authoritativeUI.Fields[0].ComponentType != "Text" ||
		authoritativeUI.Fields[0].Text != "Empty account" || len(authoritativeUI.Buttons) != 0 {
		t.Fatalf("expected existing connection UI to display the account name without a selector, got %+v", authoritativeUI)
	}
	assertHTTPError(t, k.TryCall("POST", "/v1/connections/"+simulatedConnection+"/ui-event", nil, map[string]any{
		"event": "save", "settings": map[string]any{"account": ""},
	}, nil), http.StatusUnprocessableEntity, "InvalidSettings")
	assertHTTPError(t, k.TryCall("POST", "/v1/connections/"+simulatedConnection+"/ui-event", nil, map[string]any{
		"event": "save", "settings": map[string]any{"account": secondSelectableAccount.ID},
	}, nil), http.StatusUnprocessableEntity, "InvalidSettings")
	assertHTTPError(t, k.TryCall("POST", "/v1/connections/"+simulatedConnection+"/ui-event", nil, map[string]any{
		"event": "save", "settings": map[string]any{"account": account.ID},
	}, nil), http.StatusUnprocessableEntity, "InvalidSettings")
	k.QueryRowTestDatabase(t.Context(), &boundAccount,
		"SELECT simulated_system FROM connections WHERE id = $1", simulatedConnection)
	if boundAccount != account.ID {
		t.Fatalf("expected rejected settings saves to preserve structured binding %q, got %q", account.ID, boundAccount)
	}

	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = 'Preparing' WHERE id = $1", secondSelectableAccount.ID)
	k.RestartServer(t.Context())
	_, err = k.TryCreateConnection(krenalistester.ConnectionToCreate{
		Name: "Preparing account source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": secondSelectableAccount.ID}),
	})
	assertHTTPError(t, err, http.StatusUnprocessableEntity, "InvalidSettings")
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = 'Available' WHERE id = $1", secondSelectableAccount.ID)
	k.RestartServer(t.Context())
	k.ExecQueryTestDatabase(t.Context(), `INSERT INTO connections
		(id, workspace, simulated_system, connector, role, kms_encrypted_settings_key)
		VALUES ('AAAAAAAAAAAA', $1, $2, 'simulated-application', 'Source', '\x00'::bytea)`,
		created.ID, secondSelectableAccount.ID)
	_, err = k.TryCreateConnection(krenalistester.ConnectionToCreate{
		Name: "Concurrent source", Role: krenalistester.Source, Connector: "simulated-application",
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": secondSelectableAccount.ID}),
	})
	assertHTTPError(t, err, http.StatusUnprocessableEntity, "InvalidSettings")
	k.ExecQueryTestDatabase(t.Context(), "DELETE FROM connections WHERE id = 'AAAAAAAAAAAA'")
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+simulatedConnection, nil, map[string]any{
		"name": "Simulated Application", "simulatedSystem": "",
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	k.Call("PUT", "/v1/connections/"+simulatedConnection, nil, map[string]any{"name": "Simulated Application"}, nil)
	k.Call("PUT", "/v1/simulated/systems/"+account.ID, nil, map[string]string{"name": "Renamed account"}, nil)
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.Name != "Renamed account" {
		t.Fatalf("expected renamed simulated system, got %q", detail.Name)
	}

	k.SetWorkspaceID(production.ID)
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated/systems/"+account.ID, nil, nil, nil), http.StatusNotFound, "")
	k.SetWorkspaceID(created.ID)
	unboundConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Unbound source", Role: krenalistester.Source, Connector: "dummy", Settings: json.Value("{}"),
	})
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+unboundConnection, nil, map[string]any{
		"name": "Unbound source", "simulatedSystem": account.ID,
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	k.DeleteConnection(unboundConnection)
	mode := krenalistester.Server
	firstConnection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Simulated destination", Role: krenalistester.Destination, Connector: "simulated-application",
		SendingMode: &mode,
		Settings:    krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	k.SetWorkspaceID(production.ID)
	assertHTTPError(t, k.TryCall("POST", "/v1/connections", nil, map[string]any{
		"name": "Legacy account binding", "role": "Source", "connector": "dummy",
		"settings": map[string]any{}, "simulatedSystem": account.ID,
	}, nil), http.StatusBadRequest, "simulatedSystem must be provided in settings")
	k.SetWorkspaceID(created.ID)
	k.RestartServer(t.Context())
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.SourceConnection == nil || *detail.SourceConnection != simulatedConnection ||
		detail.DestinationConnection == nil || *detail.DestinationConnection != firstConnection {
		t.Fatalf("expected account with source %s and destination %s after restart, got %+v",
			simulatedConnection, firstConnection, detail)
	}
	k.Call("GET", "/v1/simulated/systems", nil, nil, &accounts)
	var foundAccount bool
	for _, listed := range accounts.Accounts {
		if listed.ID != account.ID {
			continue
		}
		foundAccount = true
		if listed.SourceConnection == nil || *listed.SourceConnection != simulatedConnection ||
			listed.DestinationConnection == nil || *listed.DestinationConnection != firstConnection {
			t.Fatalf("expected listed account with both connections, got %+v", listed)
		}
	}
	if !foundAccount {
		t.Fatalf("expected account %s in list, got %+v", account.ID, accounts.Accounts)
	}
	var connections struct {
		Connections []map[string]any `json:"connections"`
	}
	k.Call("GET", "/v1/connections", nil, nil, &connections)
	for _, connection := range connections.Connections {
		if value, ok := connection["simulatedSystem"]; ok {
			t.Fatalf("expected simulated system binding to be absent from connection list, got %v", value)
		}
	}
	var refs int
	k.QueryRowTestDatabase(t.Context(), &refs,
		"SELECT COUNT(*) FROM connections WHERE workspace = $1 AND simulated_system = $2", created.ID, account.ID)
	if refs != 2 {
		t.Fatalf("expected 2 simulated system references after restart, got %d", refs)
	}
	assertHTTPError(t, k.TryCall("PUT", "/v1/connections/"+firstConnection, nil, map[string]any{
		"name": "Simulated destination", "sendingMode": "Server", "simulatedSystem": "",
	}, nil), http.StatusBadRequest, "simulatedSystem cannot be updated")
	var referenceCount int
	k.QueryRowTestDatabase(t.Context(), &referenceCount,
		"SELECT COUNT(*) FROM connections WHERE workspace = $1 AND simulated_system = $2", created.ID, account.ID)
	if referenceCount != 2 {
		t.Fatalf("expected 2 simulated system references after rejected clearing, got %d", referenceCount)
	}
	k.Call("PUT", "/v1/connections/"+firstConnection, nil, map[string]any{
		"name": "Simulated destination", "sendingMode": "Server",
	}, nil)

	warehousePool, err := krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected simulated system warehouse pool, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), `INSERT INTO krenalis_simulated_system_records
		(simulated_system, external_id, updated_at, data) VALUES ($1, 'account-record', TIMESTAMP '2024-10-03 15:38:36.920638', '{}'::jsonb),
		('other-account', 'other-record', TIMESTAMP '2024-10-03 15:38:36.920638', '{}'::jsonb)`, account.ID)
	if err != nil {
		warehousePool.Close()
		t.Fatalf("expected simulated system warehouse fixture, got %v", err)
	}
	assertHTTPError(t, k.TryCall("DELETE", "/v1/simulated/systems/"+account.ID, nil, nil, nil),
		http.StatusUnprocessableEntity, "SimulatedSystemInUse")
	var accountRecords, otherRecords int
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1", account.ID).Scan(&accountRecords)
	if err != nil {
		warehousePool.Close()
		t.Fatalf("expected deleted account record count, got %v", err)
	}
	err = warehousePool.QueryRow(t.Context(), "SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = 'other-account'").Scan(&otherRecords)
	warehousePool.Close()
	if err != nil {
		t.Fatalf("expected preserved account record count, got %v", err)
	}
	if accountRecords != 1 || otherRecords != 1 {
		t.Fatalf("expected referenced account cleanup isolation, got account=%d other=%d", accountRecords, otherRecords)
	}
	k.DeleteConnection(firstConnection)
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.SourceConnection == nil || *detail.SourceConnection != simulatedConnection || detail.DestinationConnection != nil {
		t.Fatalf("expected only source connection after destination deletion, got %+v", detail)
	}
	k.QueryRowTestDatabase(t.Context(), &referenceCount,
		"SELECT COUNT(*) FROM connections WHERE workspace = $1 AND simulated_system = $2", created.ID, account.ID)
	if referenceCount != 1 {
		t.Fatalf("expected one simulated system reference after deleting one connection, got %d", referenceCount)
	}
	var metadataCount int
	k.QueryRowTestDatabase(t.Context(), &metadataCount, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account.ID)
	if metadataCount != 1 {
		t.Fatalf("expected simulated system metadata to remain, got %d", metadataCount)
	}
	k.DeleteConnection(simulatedConnection)
	k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &detail)
	if detail.SourceConnection != nil || detail.DestinationConnection != nil {
		t.Fatalf("expected no connections after source deletion, got %+v", detail)
	}
	k.QueryRowTestDatabase(t.Context(), &referenceCount,
		"SELECT COUNT(*) FROM connections WHERE workspace = $1 AND simulated_system = $2", created.ID, account.ID)
	if referenceCount != 0 {
		t.Fatalf("expected no simulated system references after connection deletion, got %d", referenceCount)
	}
	k.Call("DELETE", "/v1/simulated/systems/"+account.ID, nil, nil, nil)

	var secondAccount struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "Failed cleanup", "userCount": 0,
	}, &secondAccount)
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = 'Preparing' WHERE id = $1", secondAccount.ID)
	assertHTTPError(t, k.TryCall("DELETE", "/v1/simulated/systems/"+secondAccount.ID, nil, nil, nil), http.StatusUnprocessableEntity,
		"SimulatedSystemPreparing")
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = 'Available' WHERE id = $1", secondAccount.ID)
	warehousePool, err = krenalistester.ConnectionPool(t.Context(), settings)
	if err != nil {
		t.Fatalf("expected simulated system warehouse pool, got %v", err)
	}
	_, err = warehousePool.Exec(t.Context(), "DROP TABLE krenalis_simulated_system_records")
	warehousePool.Close()
	if err != nil {
		t.Fatalf("expected simulated system warehouse fixture removal, got %v", err)
	}
	k.Call("DELETE", "/v1/simulated/systems/"+secondAccount.ID, nil, nil, nil)
	var status string
	k.QueryRowTestDatabase(t.Context(), &status, "SELECT status::text FROM simulated_systems WHERE id = $1", secondAccount.ID)
	if status != "Deleting" {
		t.Fatalf("expected metadata to remain Deleting during failed cleanup, got %q", status)
	}
	var deleting struct {
		Status string `json:"status"`
	}
	k.Call("GET", "/v1/simulated/systems/"+secondAccount.ID, nil, nil, &deleting)
	if deleting.Status != "Deleting" {
		t.Fatalf("expected deleting account to remain readable, got %q", deleting.Status)
	}
	k.Call("GET", "/v1/simulated/systems", nil, nil, &accounts)
	var foundDeleting bool
	for _, listed := range accounts.Accounts {
		if listed.ID == secondAccount.ID {
			if listed.Status != "Deleting" {
				t.Fatalf("expected listed account to be Deleting, got %q", listed.Status)
			}
			foundDeleting = true
			break
		}
	}
	if !foundDeleting {
		t.Fatalf("expected deleting account %s in list, got %+v", secondAccount.ID, accounts.Accounts)
	}
	k.RestartServer(t.Context())
	k.Call("GET", "/v1/simulated/systems/"+secondAccount.ID, nil, nil, &deleting)
	if deleting.Status != "Deleting" {
		t.Fatalf("expected failed cleanup to remain Deleting after restart, got %q", deleting.Status)
	}
	k.Call("POST", "/v1/warehouse/repair", nil, nil, nil)
	deadline := time.Now().Add(30 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &metadataCount, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", secondAccount.ID)
		if metadataCount == 0 || time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if metadataCount != 0 {
		t.Fatalf("expected account cleanup after warehouse repair, got %d metadata rows", metadataCount)
	}
	assertHTTPError(t, k.TryCall("GET", "/v1/simulated/systems/"+secondAccount.ID, nil, nil, nil),
		http.StatusNotFound, "")
	k.Call("GET", "/v1/simulated/systems", nil, nil, &accounts)
	for _, listed := range accounts.Accounts {
		if listed.ID == secondAccount.ID {
			t.Fatalf("expected deleted account to be absent from list, got %+v", listed)
		}
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
