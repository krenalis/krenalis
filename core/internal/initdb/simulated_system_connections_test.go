// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

func TestSimulatedSystemConnectionForeignKeyProtectsConcurrentDelete(t *testing.T) {
	ctx := t.Context()
	database := newInitializedTestDatabase(t)
	_, err := database.Exec(ctx, `
		INSERT INTO organizations (
			id, name, enabled, members_limit, access_keys_limit, workspaces_limit, connectors_limit,
			connections_limit, pipelines_limit, organization_requests_rate_per_minute,
			organization_requests_max_capacity, workspace_requests_rate_per_minute,
			workspace_requests_max_capacity, workspace_events_rate_per_minute, workspace_events_max_capacity
		) VALUES (
			'111111111111', 'test', true, 1, 0, 1, 1, 1, 1, 60, 1, 60, 1, 1000, 20000
		);
		INSERT INTO workspaces (
			id, organization, name, environment, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key
		) VALUES (
			'222222222222', '111111111111', 'test', 'development', 'test', 'Normal', '\x00'::bytea, '\x01'::bytea, '\x02'::bytea
		);
		INSERT INTO simulated_systems (
			id, workspace, name, status, user_count, duplicate_record_percent, countries,
			generation_policy_version, generated_record_count, generation_error, created_at, updated_at
		) VALUES (
			'333333333333', '222222222222', 'test', 'Available', 1, 0, '{}'::jsonb, '', 1, '', NOW(), NOW()
		)`)
	if err != nil {
		t.Fatal(err)
	}

	deleter, err := database.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer deleter.Rollback(ctx)
	var accountID string
	err = deleter.QueryRow(ctx, "SELECT id FROM simulated_systems WHERE id = $1 FOR UPDATE",
		"333333333333").Scan(&accountID)
	if err != nil {
		t.Fatal(err)
	}

	binder, err := database.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer binder.Close()
	var binderPID int
	err = binder.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&binderPID)
	if err != nil {
		t.Fatal(err)
	}
	bindDone := make(chan error, 1)
	go func() {
		_, err := binder.Exec(ctx, `INSERT INTO connections (
			id, workspace, simulated_system, connector, role, kms_encrypted_settings_key
		) VALUES ('444444444444', '222222222222', '333333333333', 'dummy', 'Source', '\x00'::bytea)`)
		bindDone <- err
	}()
	waitForDatabaseLock(t, database, binderPID)

	_, err = deleter.Exec(ctx, "DELETE FROM simulated_systems WHERE id = $1", accountID)
	if err != nil {
		t.Fatal(err)
	}
	if err := deleter.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	err = <-bindDone
	if err == nil {
		t.Fatal("expected concurrent binding to fail, got nil error")
	}
	if !db.IsForeignKeyViolation(err) {
		t.Fatalf("expected foreign key violation, got %v", err)
	}
	if constraint := db.ErrConstraintName(err); constraint != connectionsWorkspaceSimulatedSystemForeignKey {
		t.Fatalf("expected constraint %s, got %s", connectionsWorkspaceSimulatedSystemForeignKey, constraint)
	}
	_, err = database.Exec(ctx, `
		INSERT INTO simulated_systems (
			id, workspace, name, status, user_count, duplicate_record_percent, countries,
			generation_policy_version, generated_record_count, generation_error, created_at, updated_at
		) VALUES (
			'555555555555', '222222222222', 'test', 'Available', 1, 0, '{}'::jsonb, '', 1, '', NOW(), NOW()
		);
		INSERT INTO connections (
			id, workspace, simulated_system, connector, role, kms_encrypted_settings_key
		) VALUES ('666666666666', '222222222222', '555555555555', 'dummy', 'Source', '\x00'::bytea)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", "222222222222")
	if err != nil {
		t.Fatalf("expected workspace deletion with bound simulated system to succeed, got %v", err)
	}
}

func TestWorkspaceDeleteWithBoundSimulatedSystemAfterMigrationOrder(t *testing.T) {
	ctx := t.Context()
	database := newInitializedTestDatabase(t)
	_, err := database.Exec(ctx, `
		ALTER TABLE simulated_systems DROP CONSTRAINT simulated_systems_workspace_fkey;
		ALTER TABLE connections DROP CONSTRAINT connections_workspace_fkey;
		ALTER TABLE connections ADD CONSTRAINT connections_workspace_fkey
			FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE;
		ALTER TABLE simulated_systems ADD CONSTRAINT simulated_systems_workspace_fkey
			FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE;
		INSERT INTO organizations (
			id, name, enabled, members_limit, access_keys_limit, workspaces_limit, connectors_limit,
			connections_limit, pipelines_limit, organization_requests_rate_per_minute,
			organization_requests_max_capacity, workspace_requests_rate_per_minute,
			workspace_requests_max_capacity, workspace_events_rate_per_minute, workspace_events_max_capacity
		) VALUES (
			'111111111111', 'test', true, 1, 0, 1, 1, 1, 1, 60, 1, 60, 1, 1000, 20000
		);
		INSERT INTO workspaces (
			id, organization, name, environment, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key
		) VALUES (
			'222222222222', '111111111111', 'test', 'development', 'test', 'Normal', '\x00'::bytea,
			'\x01'::bytea, '\x02'::bytea
		);
		INSERT INTO simulated_systems (
			id, workspace, name, status, user_count, duplicate_record_percent, countries,
			generation_policy_version, generated_record_count, generation_error, created_at, updated_at
		) VALUES (
			'333333333333', '222222222222', 'test', 'Available', 1, 0, '{}'::jsonb, '', 1, '', NOW(), NOW()
		);
		INSERT INTO connections (
			id, workspace, simulated_system, connector, role, kms_encrypted_settings_key
		) VALUES (
			'444444444444', '222222222222', '333333333333', 'dummy', 'Source', '\x00'::bytea
		)`)
	if err != nil {
		t.Fatal(err)
	}

	var accountTrigger, connectionTrigger uint32
	for constraint, destination := range map[string]*uint32{
		"simulated_systems_workspace_fkey": &accountTrigger,
		"connections_workspace_fkey":       &connectionTrigger,
	} {
		err = database.QueryRow(ctx, `
			SELECT t.oid
			FROM pg_trigger t
			INNER JOIN pg_constraint c ON c.oid = t.tgconstraint
			INNER JOIN pg_proc p ON p.oid = t.tgfoid
			WHERE t.tgrelid = 'workspaces'::regclass
				AND c.conname = $1
				AND p.proname = 'RI_FKey_cascade_del'`, constraint).Scan(destination)
		if err != nil {
			t.Fatal(err)
		}
	}
	if connectionTrigger >= accountTrigger {
		t.Fatalf("expected migration-order connection cascade trigger to precede account trigger; got %d and %d",
			connectionTrigger, accountTrigger)
	}

	_, err = database.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", "222222222222")
	if err != nil {
		t.Fatalf("expected workspace deletion with bound simulated system to succeed, got %v", err)
	}
}
