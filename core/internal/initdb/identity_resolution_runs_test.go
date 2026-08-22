// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"
	"time"
)

// TestIdentityResolutionRunsSchema verifies keys, indexes, and workspace
// cascade behavior.
func TestIdentityResolutionRunsSchema(t *testing.T) {
	database := newInitializedTestDatabase(t)
	ctx := t.Context()

	for _, constraint := range []string{
		"identity_resolution_runs_pkey",
		"identity_resolution_runs_workspace_fkey",
	} {
		assertConstraintExists(t, database, "identity_resolution_runs", constraint)
	}
	assertIndexNotExists(t, database, "identity_resolution_runs_one_live_idx")
	assertIndexExists(t, database, "identity_resolution_runs_workspace_start_idx")

	var organization string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization); err != nil {
		t.Fatal(err)
	}
	_, err := database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('workspace111', $1, 'test', 'test', 'Normal', '\x01', '\x01', '\x01')`, organization)
	if err != nil {
		t.Fatal(err)
	}

	startedAt := time.Date(2026, time.August, 10, 1, 30, 0, 0, time.UTC)
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time)
		VALUES ('11111111-1111-1111-1111-111111111111', 'workspace111', $1)`, startedAt)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `UPDATE identity_resolution_runs SET end_time = $1, error = 'warehouse failure'
		WHERE id = '11111111-1111-1111-1111-111111111111'`, startedAt.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time, end_time)
		VALUES ('22222222-2222-2222-2222-222222222222', 'workspace111', $1, $2)`,
		startedAt.Add(2*time.Minute), startedAt.Add(3*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time)
		VALUES ('33333333-3333-3333-3333-333333333333', 'workspace111', $1)`,
		startedAt.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(ctx, "DELETE FROM workspaces WHERE id = 'workspace111'"); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_resolution_runs
		WHERE workspace = 'workspace111'`).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("expected workspace deletion to cascade run rows, got %d", rows)
	}
}
