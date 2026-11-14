// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

const (
	connectionsWorkspaceSimulatedSystemIndex      = "connections_workspace_simulated_system_idx"
	connectionsSimulatedSystemRoleKey             = "connections_simulated_system_role_key"
	connectionsWorkspaceSimulatedSystemForeignKey = "connections_workspace_simulated_system_fkey"
	simulatedSystemsWorkspaceIDKey                = "simulated_systems_workspace_id_key"
)

func TestInitializeSimulatedSystemsSchema(t *testing.T) {
	database := newInitializedTestDatabase(t)
	assertSimulatedSystemsTable(t, database)
	assertColumnExists(t, database, "connections", "simulated_system")
	var nullable string
	err := database.QueryRow(t.Context(), `SELECT is_nullable FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'connections' AND
			column_name = 'simulated_system'`).Scan(&nullable)
	if err != nil {
		t.Fatalf("expected simulated system nullability query, got %v", err)
	}
	if nullable != "YES" {
		t.Fatalf("expected connections.simulated_system to be nullable, got %q", nullable)
	}
	assertConstraintExists(t, database, "simulated_systems", simulatedSystemsWorkspaceIDKey)
	assertConstraintExists(t, database, "connections", connectionsWorkspaceSimulatedSystemForeignKey)
	assertIndexExists(t, database, connectionsWorkspaceSimulatedSystemIndex)
	assertIndexExists(t, database, connectionsSimulatedSystemRoleKey)
	assertSimulatedSystemLiveRunSchema(t, database)
}

func assertColumnExists(t *testing.T, database *db.DB, table, column string) {
	t.Helper()

	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = $1
			AND column_name = $2`, table, column)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected column %s.%s to exist, got missing column", table, column)
	}
}

// assertIndexExists verifies that an index with name exists.
func assertIndexExists(t *testing.T, database *db.DB, name string) {
	t.Helper()

	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'i'`, name)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected index %s to exist, got no index", name)
	}
}

func assertSimulatedSystemLiveRunSchema(t *testing.T, database *db.DB) {
	t.Helper()
	assertColumnExists(t, database, "pipelines_runs", "simulated_system")
	assertIndexExists(t, database, "pipelines_one_live_simulated_system_run_idx")
	assertColumnExists(t, database, "simulated_systems", "writer_generation")
	assertColumnExists(t, database, "simulated_systems", "claimed_run")
	assertColumnExists(t, database, "simulated_systems", "claim_ping_time")
	assertConstraintExists(t, database, "simulated_systems", "simulated_systems_claimed_run_key")
}

func assertSimulatedSystemsTable(t *testing.T, database *db.DB) {
	t.Helper()
	var exists bool
	err := database.QueryRow(t.Context(), "SELECT to_regclass('simulated_systems') IS NOT NULL").Scan(&exists)
	if err != nil {
		t.Fatalf("expected simulated systems table query, got %v", err)
	}
	if !exists {
		t.Fatal("expected simulated systems table to exist")
	}
	var statusType string
	err = database.QueryRow(t.Context(), "SELECT typname FROM pg_type WHERE typname = 'simulated_system_status'").Scan(&statusType)
	if err != nil {
		t.Fatalf("expected simulated system status type query, got %v", err)
	}
	if statusType != "simulated_system_status" {
		t.Fatalf("expected simulated system status type, got %q", statusType)
	}
	var available string
	err = database.QueryRow(t.Context(), "SELECT 'Available'::simulated_system_status::text").Scan(&available)
	if err != nil {
		t.Fatalf("expected Available simulated system status, got %v", err)
	}
	var deleting string
	err = database.QueryRow(t.Context(), "SELECT 'Deleting'::simulated_system_status::text").Scan(&deleting)
	if err != nil {
		t.Fatalf("expected Deleting simulated system status, got %v", err)
	}
	assertColumnDoesNotExist(t, database, "simulated_systems", "connector")
}
