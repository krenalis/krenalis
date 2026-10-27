// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"
	"time"
)

// TestIdentityResolutionRunsSchema verifies constraints, the history index,
// and cascading deletion of runs when a workspace is deleted.
func TestIdentityResolutionRunsSchema(t *testing.T) {

	database := newInitializedTestDatabase(t)
	ctx := t.Context()

	for constraint, want := range map[string]string{
		"identity_resolution_runs_pkey":           "PRIMARY KEY (id)",
		"identity_resolution_runs_workspace_fkey": "FOREIGN KEY (workspace) REFERENCES workspaces(id) ON DELETE CASCADE",
	} {
		var definition string
		err := database.QueryRow(ctx, `SELECT pg_get_constraintdef(oid)
			FROM pg_constraint
			WHERE conrelid = 'identity_resolution_runs'::regclass AND conname = $1`, constraint).Scan(&definition)
		if err != nil {
			t.Fatalf("expected definition of constraint %s, got %v", constraint, err)
		}
		if definition != want {
			t.Fatalf("expected constraint %s definition %q, got %q", constraint, want, definition)
		}
	}

	var indexDefinition string
	err := database.QueryRow(ctx, `SELECT indexdef FROM pg_indexes
		WHERE schemaname = current_schema() AND tablename = 'identity_resolution_runs'
			AND indexname = 'identity_resolution_runs_workspace_start_idx'`).Scan(&indexDefinition)
	if err != nil {
		t.Fatalf("expected history index definition, got %v", err)
	}
	if want := "CREATE INDEX identity_resolution_runs_workspace_start_idx " +
		"ON public.identity_resolution_runs USING btree (workspace, start_time DESC, id DESC)"; indexDefinition != want {
		t.Fatalf("expected history index definition %q, got %q", want, indexDefinition)
	}

	var organization string
	err = database.QueryRow(ctx, "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization)
	if err != nil {
		t.Fatalf("expected initial organization, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('workspace111', $1, 'test', 'test', 'Normal', '\x01', '\x01', '\x01')`, organization)
	if err != nil {
		t.Fatalf("expected workspace insertion to succeed, got %v", err)
	}

	startedAt := time.Date(2026, time.August, 10, 1, 30, 0, 0, time.UTC)
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time)
		VALUES ('11111111-1111-1111-1111-111111111111', 'workspace111', $1)`, startedAt)
	if err != nil {
		t.Fatalf("expected first run insertion to succeed, got %v", err)
	}
	result, err := database.Exec(ctx, `UPDATE identity_resolution_runs SET end_time = $1, error = 'warehouse failure'
		WHERE id = '11111111-1111-1111-1111-111111111111'`, startedAt.Add(time.Minute))
	if err != nil {
		t.Fatalf("expected run completion to succeed, got %v", err)
	}
	if rows := result.RowsAffected(); rows != 1 {
		t.Fatalf("expected 1 updated run, got %d", rows)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time, end_time)
		VALUES ('22222222-2222-2222-2222-222222222222', 'workspace111', $1, $2)`,
		startedAt.Add(2*time.Minute), startedAt.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("expected successful run insertion to succeed, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_runs (id, workspace, start_time)
		VALUES ('33333333-3333-3333-3333-333333333333', 'workspace111', $1)`,
		startedAt.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("expected running run insertion to succeed, got %v", err)
	}

	result, err = database.Exec(ctx, "DELETE FROM workspaces WHERE id = 'workspace111'")
	if err != nil {
		t.Fatalf("expected workspace deletion to succeed, got %v", err)
	}
	if rows := result.RowsAffected(); rows != 1 {
		t.Fatalf("expected 1 deleted workspace, got %d", rows)
	}
	var rows int
	err = database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_resolution_runs
		WHERE workspace = 'workspace111'`).Scan(&rows)
	if err != nil {
		t.Fatalf("expected remaining run count, got %v", err)
	}
	if rows != 0 {
		t.Fatalf("expected workspace deletion to cascade run rows, got %d", rows)
	}

}
