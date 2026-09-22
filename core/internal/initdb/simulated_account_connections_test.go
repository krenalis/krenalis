// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

func TestSimulatedAccountConnectionForeignKeyProtectsConcurrentDelete(t *testing.T) {
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
		INSERT INTO simulated_accounts (
			id, workspace, name, status, user_count, duplicate_record_percent, countries,
			generation_policy_version, generated_record_count, generation_error, created_at, updated_at
		) VALUES (
			'333333333333', '222222222222', 'test', 'Ready', 1, 0, '{}'::jsonb, '', 1, '', NOW(), NOW()
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
	err = deleter.QueryRow(ctx, "SELECT id FROM simulated_accounts WHERE id = $1 FOR UPDATE",
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
			id, workspace, simulated_account, connector, role, kms_encrypted_settings_key
		) VALUES ('444444444444', '222222222222', '333333333333', 'dummy', 'Source', '\x00'::bytea)`)
		bindDone <- err
	}()
	waitForDatabaseLock(t, database, binderPID)

	_, err = deleter.Exec(ctx, "DELETE FROM simulated_accounts WHERE id = $1", accountID)
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
	if constraint := db.ErrConstraintName(err); constraint != connectionsWorkspaceSimulatedAccountForeignKey {
		t.Fatalf("expected constraint %s, got %s", connectionsWorkspaceSimulatedAccountForeignKey, constraint)
	}
	_, err = database.Exec(ctx, `
		INSERT INTO simulated_accounts (
			id, workspace, name, status, user_count, duplicate_record_percent, countries,
			generation_policy_version, generated_record_count, generation_error, created_at, updated_at
		) VALUES (
			'555555555555', '222222222222', 'test', 'Ready', 1, 0, '{}'::jsonb, '', 1, '', NOW(), NOW()
		);
		INSERT INTO connections (
			id, workspace, simulated_account, connector, role, kms_encrypted_settings_key
		) VALUES ('666666666666', '222222222222', '555555555555', 'dummy', 'Source', '\x00'::bytea)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, "DELETE FROM workspaces WHERE id = $1", "222222222222")
	if err != nil {
		t.Fatalf("expected workspace deletion with bound simulated account to succeed, got %v", err)
	}
}
