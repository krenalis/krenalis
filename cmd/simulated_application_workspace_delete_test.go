// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/krenalis/krenalis/test/krenalistester"
	"github.com/krenalis/krenalis/test/syntheticworldtest"
	"github.com/krenalis/krenalis/tools/json"
)

func TestDeleteWorkspaceWithBoundSimulatedSystemFreshSchema(t *testing.T) {

	if testing.Short() {
		t.Skip()
	}

	assets := syntheticworldtest.Assets(t)
	server := httptest.NewServer(syntheticworldtest.Handler(t, assets, ""))
	defer server.Close()
	k := krenalistester.NewKrenalisInstance(t)
	k.SetSyntheticPhotosDir(simulatedSystemCatalogFixture(t))
	k.SetExternalAssetsURLs([]string{server.URL})
	k.Start()
	defer k.Stop()

	warehouseSettings := &krenalistester.DBSettings{}
	err := json.Unmarshal(krenalistester.PostgresWarehouseSettings(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected PostgreSQL warehouse settings, got %v", err)
	}
	adminPool, err := krenalistester.ConnectionPool(t.Context(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected PostgreSQL admin connection pool, got %v", err)
	}
	database := "test_simulated_workspace_delete_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	_, err = adminPool.Exec(t.Context(), "CREATE DATABASE "+database)
	adminPool.Close()
	if err != nil {
		t.Fatalf("expected isolated warehouse database, got %v", err)
	}
	cleanupSettings := *warehouseSettings
	defer func() {
		pool, err := krenalistester.ConnectionPool(context.Background(), &cleanupSettings)
		if err != nil {
			t.Errorf("expected PostgreSQL cleanup connection, got %v", err)
			return
		}
		defer pool.Close()
		_, err = pool.Exec(context.Background(), "DROP DATABASE "+database+" WITH (FORCE)")
		if err != nil {
			t.Errorf("expected isolated warehouse cleanup, got %v", err)
		}
	}()
	warehouseSettings.Database = database
	var workspace struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/workspaces", http.Header{"Krenalis-Workspace": nil}, map[string]any{
		"name": "Fresh-schema workspace deletion", "environment": "development",
		"profileSchema": simulatedApplicationProfileSchema(),
		"warehouse":     map[string]any{"platform": "PostgreSQL", "mode": "Normal", "settings": warehouseSettings},
	}, &workspace)
	k.SetWorkspaceID(workspace.ID)

	var account struct {
		ID string `json:"id"`
	}
	k.Call("POST", "/v1/simulated/systems", nil, map[string]any{
		"name": "Bound account", "userCount": 1, "countries": map[string]int{"IT": 100},
	}, &account)
	deadline := time.Now().Add(30 * time.Second)
	var status struct {
		Status string `json:"status"`
	}
	for {
		k.Call("GET", "/v1/simulated/systems/"+account.ID, nil, nil, &status)
		if status.Status == "Available" || status.Status == "Failed" || time.Now().After(deadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if status.Status != "Available" {
		t.Fatalf("expected simulated system to be Available, got %q", status.Status)
	}
	warehousePool, err := krenalistester.ConnectionPool(t.Context(), warehouseSettings)
	if err != nil {
		t.Fatalf("expected simulated-system warehouse connection, got %v", err)
	}
	defer warehousePool.Close()
	var warehouseRecords int
	err = warehousePool.QueryRow(
		t.Context(),
		"SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1",
		account.ID,
	).Scan(&warehouseRecords)
	if err != nil {
		t.Fatalf("expected simulated-system warehouse records, got %v", err)
	}
	if warehouseRecords != 1 {
		t.Fatalf("expected one simulated-system warehouse record, got %d", warehouseRecords)
	}

	mode := krenalistester.Server
	connection := k.CreateConnection(krenalistester.ConnectionToCreate{
		Name: "Bound Simulated Application destination", Role: krenalistester.Destination,
		Connector: "simulated-application", SendingMode: &mode,
		Settings: krenalistester.JSONEncodeSettings(map[string]any{"account": account.ID}),
	})
	var boundAccount string
	k.QueryRowTestDatabase(t.Context(), &boundAccount,
		"SELECT simulated_system FROM connections WHERE id = $1", connection)
	if boundAccount != account.ID {
		t.Fatalf("expected connection to bind account %q, got %q", account.ID, boundAccount)
	}

	var cascadeState string
	k.QueryRowTestDatabase(t.Context(), &cascadeState, `
		SELECT string_agg(c.conname || ' => ' || pg_get_constraintdef(c.oid) ||
			' / ' || t.tgname || ' / ' || p.proname, E'\n' ORDER BY t.tgname)
		FROM pg_trigger t
		INNER JOIN pg_constraint c ON c.oid = t.tgconstraint
		INNER JOIN pg_proc p ON p.oid = t.tgfoid
		WHERE t.tgrelid = 'workspaces'::regclass
			AND c.conname IN ('simulated_systems_workspace_fkey', 'connections_workspace_fkey')`)
	t.Logf("fresh-schema workspace cascade triggers:\n%s", cascadeState)

	assertHTTPError(t, k.TryCall("DELETE", "/v1/workspaces/current", nil, nil, nil),
		http.StatusUnprocessableEntity, "SimulatedSystemInUse")

	var workspaces, connections, accounts int
	k.QueryRowTestDatabase(t.Context(), &workspaces, "SELECT COUNT(*) FROM workspaces WHERE id = $1", workspace.ID)
	k.QueryRowTestDatabase(t.Context(), &connections, "SELECT COUNT(*) FROM connections WHERE id = $1", connection)
	k.QueryRowTestDatabase(t.Context(), &accounts, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account.ID)
	if workspaces != 1 || connections != 1 || accounts != 1 {
		t.Fatalf("expected workspace, connection, and account to remain; got %d, %d, %d",
			workspaces, connections, accounts)
	}

	k.DeleteConnection(connection)
	fenceLock, err := warehousePool.Begin(t.Context())
	if err != nil {
		t.Fatalf("expected warehouse fence lock transaction, got %v", err)
	}
	defer fenceLock.Rollback(context.Background())
	var generation int64
	err = fenceLock.QueryRow(t.Context(), "SELECT generation FROM krenalis_simulated_system_writers "+
		"WHERE simulated_system = $1 FOR UPDATE", account.ID).Scan(&generation)
	if err != nil {
		t.Fatalf("expected generation fence lock, got %v", err)
	}
	for _, status := range []string{"Preparing", "Failed", "Deleting"} {
		k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = $1 WHERE id = $2", status, account.ID)
		code := "SimulatedSystemInUse"
		if status == "Preparing" {
			code = "SimulatedSystemPreparing"
		}
		assertHTTPError(t, k.TryCall("DELETE", "/v1/workspaces/current", nil, nil, nil),
			http.StatusUnprocessableEntity, code)
	}
	var organization string
	k.QueryRowTestDatabase(t.Context(), &organization, "SELECT organization FROM workspaces WHERE id = $1", workspace.ID)
	assertHTTPError(t, k.TryDeleteOrganization(organization), http.StatusUnprocessableEntity, "SimulatedSystemInUse")
	k.ExecQueryTestDatabase(t.Context(), "UPDATE simulated_systems SET status = 'Available' WHERE id = $1", account.ID)
	err = fenceLock.Rollback(t.Context())
	if err != nil {
		t.Fatalf("expected warehouse fence lock release, got %v", err)
	}

	k.Call("DELETE", "/v1/simulated/systems/"+account.ID, nil, nil, nil)
	cleanupDeadline := time.Now().Add(30 * time.Second)
	for {
		k.QueryRowTestDatabase(t.Context(), &accounts, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account.ID)
		if accounts == 0 || time.Now().After(cleanupDeadline) {
			break
		}
		time.Sleep(25 * time.Millisecond)
	}
	if accounts != 0 {
		t.Fatalf("expected simulated system cleanup to finish, got %d remaining", accounts)
	}
	err = warehousePool.QueryRow(
		t.Context(),
		"SELECT COUNT(*) FROM krenalis_simulated_system_records WHERE simulated_system = $1",
		account.ID,
	).Scan(&warehouseRecords)
	if err != nil {
		t.Fatalf("expected simulated-system warehouse record count after cleanup, got %v", err)
	}
	if warehouseRecords != 0 {
		t.Fatalf("expected simulated-system warehouse records to be removed, got %d", warehouseRecords)
	}

	k.Call("DELETE", "/v1/workspaces/current", nil, nil, nil)
	k.QueryRowTestDatabase(t.Context(), &workspaces, "SELECT COUNT(*) FROM workspaces WHERE id = $1", workspace.ID)
	k.QueryRowTestDatabase(t.Context(), &connections, "SELECT COUNT(*) FROM connections WHERE id = $1", connection)
	k.QueryRowTestDatabase(t.Context(), &accounts, "SELECT COUNT(*) FROM simulated_systems WHERE id = $1", account.ID)
	if workspaces != 0 || connections != 0 || accounts != 0 {
		t.Fatalf("expected workspace and connection to be removed after account cleanup; got %d, %d, %d",
			workspaces, connections, accounts)
	}
	err = k.TryDeleteOrganization(organization)
	if err != nil {
		t.Fatalf("expected organization deletion after system cleanup, got %v", err)
	}
	var organizations int
	k.QueryRowTestDatabase(t.Context(), &organizations, "SELECT COUNT(*) FROM organizations WHERE id = $1", organization)
	if organizations != 0 {
		t.Fatalf("expected organization metadata removed after cleanup, got %d rows", organizations)
	}

}
