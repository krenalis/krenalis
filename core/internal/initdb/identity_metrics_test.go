// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
)

// TestIdentityMetricsSchema verifies keys, indexes, metric columns,
// count-column defaults, workspace relationships, and cascade behavior in the
// canonical schema.
func TestIdentityMetricsSchema(t *testing.T) {

	database := newInitializedTestDatabase(t)
	ctx := t.Context()

	for table, want := range map[string]string{
		"identity_resolution_metrics": "workspace,day",
		"identity_metrics":            "workspace,day",
		"identity_connection_metrics": "connection,day",
	} {
		if got := identityMetricPrimaryKey(t, database, table); got != want {
			t.Fatalf("expected %s primary key %q, got %q", table, want, got)
		}
	}
	for table, constraints := range map[string][]string{
		"identity_metrics": {
			"identity_metrics_workspace_fkey",
		},
		"identity_connection_metrics": {
			"identity_connection_metrics_connection_fkey",
		},
	} {
		for _, constraint := range constraints {
			assertConstraintExists(t, database, table, constraint)
		}
	}
	assertIndexNotExists(t, database, "identity_connection_metrics_connection_idx")

	var workspaceConstraint string
	err := database.QueryRow(ctx, `SELECT pg_get_constraintdef(oid)
		FROM pg_constraint
		WHERE conrelid = 'identity_resolution_metrics'::regclass
			AND conname = 'identity_resolution_metrics_workspace_fkey'`).Scan(&workspaceConstraint)
	if err != nil {
		t.Fatalf("expected identity_resolution_metrics workspace constraint definition, got %v", err)
	}
	if want := "FOREIGN KEY (workspace) REFERENCES workspaces(id) ON DELETE CASCADE"; workspaceConstraint != want {
		t.Fatalf("expected identity_resolution_metrics workspace constraint %q, got %q", want, workspaceConstraint)
	}

	for _, column := range []string{
		"profiles_anonymous", "profiles_recognized", "identities_anonymous", "identities_recognized",
		"composition_one", "composition_two", "composition_three", "composition_four_to_ten",
		"composition_eleven_to_twenty", "composition_more_than_twenty",
	} {
		var dataType, nullable string
		err := database.QueryRow(ctx, `SELECT data_type, is_nullable
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = 'identity_resolution_metrics'
				AND column_name = $1`, column).Scan(&dataType, &nullable)
		if err != nil {
			t.Fatalf("expected identity_resolution_metrics.%s column definition, got %v", column, err)
		}
		if dataType != "bigint" || nullable != "NO" {
			t.Errorf("expected identity_resolution_metrics.%s to be bigint NOT NULL, got data_type=%q, is_nullable=%q",
				column, dataType, nullable)
		}
	}

	var defaults int
	err = database.QueryRow(ctx, `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name IN ('identity_resolution_metrics', 'identity_metrics', 'identity_connection_metrics')
			AND (column_name LIKE 'identities\_%' ESCAPE '\'
				OR column_name LIKE 'profiles\_%' ESCAPE '\'
				OR column_name LIKE 'composition\_%' ESCAPE '\')
			AND column_default IS NOT NULL`).Scan(&defaults)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if defaults != 0 {
		t.Fatalf("expected identity metric count columns without defaults, got %d defaults", defaults)
	}

	var organization string
	err = database.QueryRow(ctx, "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('workspace111', $1, 'test', 'test', 'Normal', '\x01', '\x01', '\x01')`, organization)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES ('connection11', 'workspace111', 'Source', '\x01')`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	observedAt := time.Date(2026, time.August, 9, 10, 30, 0, 0, time.UTC)
	day := observedAt.Truncate(24 * time.Hour)
	_, err = database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('workspace111', $1, $2, 0, 0, 0)`, day, observedAt.Format("15:04:05.999999"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('connection11', $1, 0, 0, 0)`, day)
	if err != nil {
		t.Fatalf("expected an explicit zero connection row to be accepted, got %v", err)
	}

	_, err = database.Exec(ctx, `INSERT INTO identity_resolution_metrics
		(workspace, day, observed_at,
		profiles_anonymous, profiles_recognized, identities_anonymous, identities_recognized,
		composition_one, composition_two, composition_three,
		composition_four_to_ten,
		composition_eleven_to_twenty, composition_more_than_twenty)
		VALUES ('workspace111', $1, $2, 1, 1, 2, 1, 1, 1, 0, 0, 0, 0)`,
		day, observedAt.Format("15:04:05.999999"))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var organizationColumns int
	err = database.QueryRow(ctx, `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name IN ('identity_resolution_metrics', 'identity_metrics', 'identity_connection_metrics')
			AND column_name = 'organization'`).Scan(&organizationColumns)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if organizationColumns != 0 {
		t.Fatalf("expected identity metric tables without organization columns, got %d", organizationColumns)
	}
	assertColumnDoesNotExist(t, database, "identity_resolution_metrics", "operation_id")
	assertColumnDoesNotExist(t, database, "identity_resolution_metrics", "started_at")
	assertColumnDoesNotExist(t, database, "identity_resolution_metrics", "completed_at")
	assertColumnDoesNotExist(t, database, "identity_connection_metrics", "observed_at")
	assertColumnDoesNotExist(t, database, "identity_connection_metrics", "workspace")
	for _, table := range []string{"identity_resolution_metrics", "identity_metrics"} {
		var dataType string
		err = database.QueryRow(ctx, `SELECT data_type
			FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = $1
				AND column_name = 'observed_at'`, table).Scan(&dataType)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if dataType != "time without time zone" {
			t.Fatalf("expected %s.observed_at to be time without time zone, got %q", table, dataType)
		}
	}

	_, err = database.Exec(ctx, "DELETE FROM connections WHERE id = 'connection11'")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	var connectionRows int
	err = database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_connection_metrics
		WHERE connection = 'connection11'`).Scan(&connectionRows)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if connectionRows != 0 {
		t.Fatalf("expected deleting a connection to cascade its metric rows, got %d", connectionRows)
	}
	for _, table := range []string{"identity_resolution_metrics", "identity_metrics"} {
		var parentRows int
		err = database.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE workspace = 'workspace111'").
			Scan(&parentRows)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if parentRows != 1 {
			t.Fatalf("expected the %s workspace metric to survive a connection delete, got %d rows", table, parentRows)
		}
	}
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES ('connection22', 'workspace111', 'Source', '\x01')`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('connection22', $1, 0, 0, 0)`, day)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = database.Exec(ctx, "DELETE FROM workspaces WHERE id = 'workspace111'")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	for _, table := range []string{"identity_resolution_metrics", "identity_metrics"} {
		var rows int
		err = database.QueryRow(ctx, "SELECT COUNT(*) FROM "+table+" WHERE workspace = 'workspace111'").Scan(&rows)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if rows != 0 {
			t.Fatalf("expected deleting workspace to cascade to %s, got %d rows", table, rows)
		}
	}
	err = database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_connection_metrics
		WHERE connection = 'connection22'`).Scan(&connectionRows)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if connectionRows != 0 {
		t.Fatalf("expected deleting workspace connections to cascade their metric rows, got %d", connectionRows)
	}

}

// assertColumnDoesNotExist verifies that table has no column with the given name.
func assertColumnDoesNotExist(t *testing.T, database *db.DB, table, column string) {
	t.Helper()
	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = $1
			AND column_name = $2`, table, column)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected column %s.%s to be missing, got existing column", table, column)
	}
}

// assertConstraintExists verifies that table has a constraint with the given name.
func assertConstraintExists(t *testing.T, database *db.DB, table, constraint string) {
	t.Helper()
	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
			AND t.relname = $1
			AND c.conname = $2`, table, constraint)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !exists {
		t.Fatalf("expected constraint %s.%s to exist, got missing constraint", table, constraint)
	}
}

// assertIndexNotExists verifies that an index with name does not exist.
func assertIndexNotExists(t *testing.T, database *db.DB, name string) {
	t.Helper()
	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'i'`, name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exists {
		t.Fatalf("expected index %s to not exist, got existing index", name)
	}
}

// identityMetricPrimaryKey returns the ordered primary-key columns for table.
func identityMetricPrimaryKey(t *testing.T, database *db.DB, table string) string {
	t.Helper()
	var columns string
	err := database.QueryRow(t.Context(), `SELECT string_agg(a.attname, ',' ORDER BY k.ordinality)
		FROM pg_constraint AS c
		CROSS JOIN LATERAL unnest(c.conkey) WITH ORDINALITY AS k(attnum, ordinality)
		INNER JOIN pg_attribute AS a ON a.attrelid = c.conrelid AND a.attnum = k.attnum
		WHERE c.contype = 'p' AND c.conrelid = $1::regclass`, table).Scan(&columns)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	return columns
}
