// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"os"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/core/internal/db"
)

// schemaSnapshotQuery describes the objects of the current schema regardless
// of how they were created.
const schemaSnapshotQuery = `
	SELECT jsonb_build_object(
		'columns', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, a.attname, format_type(a.atttypid, a.atttypmod),
				a.attnotnull, pg_get_expr(d.adbin, d.adrelid)) ORDER BY c.relname, a.attnum)
			FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
			WHERE c.relnamespace = current_schema()::regnamespace AND a.attnum > 0 AND NOT a.attisdropped),
		'constraints', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, con.conname, pg_get_constraintdef(con.oid))
				ORDER BY c.relname, con.conname)
			FROM pg_constraint con
			JOIN pg_class c ON c.oid = con.conrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'indexes', (
			SELECT jsonb_agg(pg_get_indexdef(i.indexrelid) ORDER BY c.relname)
			FROM pg_index i
			JOIN pg_class c ON c.oid = i.indexrelid
			WHERE c.relnamespace = current_schema()::regnamespace),
		'enums', (
			SELECT jsonb_agg(jsonb_build_array(t.typname, e.enumlabel) ORDER BY t.typname, e.enumsortorder)
			FROM pg_enum e
			JOIN pg_type t ON t.oid = e.enumtypid
			WHERE t.typnamespace = current_schema()::regnamespace),
		'views', (
			SELECT jsonb_agg(jsonb_build_array(c.relname, pg_get_viewdef(c.oid)) ORDER BY c.relname)
			FROM pg_class c
			WHERE c.relnamespace = current_schema()::regnamespace AND c.relkind = 'v'),
		'functions', (
			SELECT jsonb_agg(pg_get_functiondef(p.oid) ORDER BY pg_get_functiondef(p.oid))
			FROM pg_proc p
			WHERE p.pronamespace = current_schema()::regnamespace)
	)::text`

// TestUpgrade verifies that upgrading a v0.46.0 database produces the schema
// of a newly initialized database, preserves and converts its data, and is
// idempotent.
func TestUpgrade(t *testing.T) {

	fresh := newInitializedTestDatabase(t)
	var expected string
	err := fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&expected)
	if err != nil {
		t.Fatal(err)
	}

	// Upgrading a newly initialized database must not change it.
	err = Upgrade(t.Context(), fresh)
	if err != nil {
		t.Fatal(err)
	}
	var got string
	err = fresh.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
	if err != nil {
		t.Fatal(err)
	}
	if got != expected {
		t.Fatalf("expected unchanged schema %s, got %s", expected, got)
	}

	database := newV0_46_0TestDatabase(t)

	for range 2 {
		err = Upgrade(t.Context(), database)
		if err != nil {
			t.Fatal(err)
		}
		err = database.QueryRow(t.Context(), schemaSnapshotQuery).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got != expected {
			t.Fatalf("expected schema %s, got %s", expected, got)
		}
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(to_jsonb(w) - ARRAY['organization', 'name', 'warehouse_name', 'warehouse_mode',
				'warehouse_settings', 'warehouse_mcp_settings', 'kms_encrypted_warehouse_settings_key',
				'kms_encrypted_warehouse_mcp_settings_key'] ORDER BY id)
			FROM workspaces w`, `[
			{
				"id": "Workspace111",
				"alter_profile_schema_id": null,
				"alter_profile_schema_schema": null,
				"alter_profile_schema_primary_sources": null,
				"alter_profile_schema_operations": null,
				"alter_profile_schema_role_first_name": "",
				"alter_profile_schema_role_last_name": "",
				"alter_profile_schema_role_email": "",
				"alter_profile_schema_role_country": "",
				"alter_profile_schema_role_photo": "",
				"alter_profile_schema_start_time": "2025-11-01T10:00:00.123",
				"alter_profile_schema_end_time": "2025-11-01T10:00:01.456",
				"alter_profile_schema_error": "alteration failed",
				"profile_schema": {"kind": "object", "properties": []},
				"profile_role_first_name": "",
				"profile_role_last_name": "",
				"profile_role_email": "",
				"profile_role_country": "",
				"profile_role_photo": "",
				"resolve_identities_on_batch_import": true,
				"identifiers": ["email", "phone"],
				"ir_id": "c0a80101-0000-4000-8000-000000000001",
				"ir_start_time": "2026-01-02T03:04:05.678",
				"ir_end_time": null,
				"pipelines_to_purge": ["pipeline1111"]
			},
			{
				"id": "Workspace222",
				"alter_profile_schema_id": "c0a80101-0000-4000-8000-000000000002",
				"alter_profile_schema_schema": {"kind": "object", "properties": []},
				"alter_profile_schema_primary_sources": {},
				"alter_profile_schema_operations": [],
				"alter_profile_schema_role_first_name": "",
				"alter_profile_schema_role_last_name": "",
				"alter_profile_schema_role_email": "",
				"alter_profile_schema_role_country": "",
				"alter_profile_schema_role_photo": "",
				"alter_profile_schema_start_time": "2026-01-03T00:00:00",
				"alter_profile_schema_end_time": null,
				"alter_profile_schema_error": null,
				"profile_schema": null,
				"profile_role_first_name": "",
				"profile_role_last_name": "",
				"profile_role_email": "",
				"profile_role_country": "",
				"profile_role_photo": "",
				"resolve_identities_on_batch_import": false,
				"identifiers": [],
				"ir_id": null,
				"ir_start_time": "2025-12-01T00:00:00",
				"ir_end_time": "2025-12-01T00:01:00",
				"pipelines_to_purge": []
			}
		]`)
		assertUpgradedData(t, database, `
			SELECT jsonb_agg(jsonb_build_array(id, workspace, start_time, end_time, error))
			FROM identity_resolution_runs`,
			`[["c0a80101-0000-4000-8000-000000000001", "Workspace111", "2026-01-02T03:04:05.678", null, null]]`)
	}

}

// assertUpgradedData verifies that query returns a single JSON value equal to
// expected.
func assertUpgradedData(t *testing.T, database *db.DB, query, expected string) {
	t.Helper()
	var equal bool
	var got string
	err := database.QueryRow(t.Context(), `SELECT q = $1::jsonb, q::text FROM (`+query+`) AS r(q)`, expected).
		Scan(&equal, &got)
	if err != nil {
		t.Fatal(err)
	}
	if !equal {
		t.Fatalf("expected %s, got %s", expected, got)
	}
}

// newV0_46_0TestDatabase returns a test database with the schema of release
// v0.46.0 and data covering every upgrade step.
func newV0_46_0TestDatabase(t *testing.T) *db.DB {

	t.Helper()

	schema, err := os.ReadFile("testdata/schema_v0.46.0.sql")
	if err != nil {
		t.Fatal(err)
	}
	database := newTestDatabase(t)
	for query := range strings.SplitSeq(string(schema), ";\n") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err = database.Exec(t.Context(), query)
		if err != nil {
			t.Fatal(err)
		}
	}
	// The rate-limit lease function has not changed since release v0.46.0.
	_, err = database.Exec(t.Context(), createRateLimiterLeasesFunction)
	if err != nil {
		t.Fatal(err)
	}

	// Workspace111 has an Identity Resolution in progress and a failed profile
	// schema alteration; Workspace222 has a profile schema alteration in progress
	// and a completed Identity Resolution. Both have UI profile preferences.
	_, err = database.Exec(t.Context(), `
		INSERT INTO organizations (id, name, enabled, members_limit, access_keys_limit, workspaces_limit,
			connectors_limit, connections_limit, pipelines_limit, organization_requests_rate_per_minute,
			organization_requests_max_capacity, workspace_requests_rate_per_minute, workspace_requests_max_capacity,
			workspace_events_rate_per_minute, workspace_events_max_capacity)
		VALUES ('organization', 'ACME inc', true, 10000, 1000, 1000, 1000, 10000, 10000, 1000, 1000, 1000, 1000,
			1000, 20000);
		INSERT INTO workspaces (id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key,
			alter_profile_schema_start_time, alter_profile_schema_end_time, alter_profile_schema_error,
			profile_schema, resolve_identities_on_batch_import, identifiers, ir_id, ir_start_time,
			ui_profile_image, ui_profile_first_name, ui_profile_last_name, ui_profile_extra, pipelines_to_purge)
		VALUES ('Workspace111', 'organization', 'Workspace 1', 'PostgreSQL', 'Normal', '\x01', '\x02', '\x03',
			'2025-11-01 10:00:00.123', '2025-11-01 10:00:01.456', 'alteration failed',
			'{"kind": "object", "properties": []}', true, '{email,phone}', 'c0a80101-0000-4000-8000-000000000001',
			'2026-01-02 03:04:05.678', 'photo', 'first_name', 'last_name', 'email', '{pipeline1111}');
		INSERT INTO workspaces (id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key, alter_profile_schema_id,
			alter_profile_schema_schema, alter_profile_schema_primary_sources, alter_profile_schema_operations,
			alter_profile_schema_start_time, ir_start_time, ir_end_time, ui_profile_image, ui_profile_first_name,
			ui_profile_last_name, ui_profile_extra)
		VALUES ('Workspace222', 'organization', 'Workspace 2', 'PostgreSQL', 'Normal', '\x01', '\x02', '\x03',
			'c0a80101-0000-4000-8000-000000000002', '{"kind": "object", "properties": []}', '{}', '[]',
			'2026-01-03 00:00:00', '2025-12-01 00:00:00', '2025-12-01 00:01:00', 'avatar', 'name', 'surname', '')`)
	if err != nil {
		t.Fatal(err)
	}

	return database
}
