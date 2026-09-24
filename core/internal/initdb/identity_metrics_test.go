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

	var defaults int
	err := database.QueryRow(ctx, `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name IN ('identity_metrics', 'identity_connection_metrics')
			AND column_name LIKE 'identities\_%' ESCAPE '\'
			AND column_default IS NOT NULL`).Scan(&defaults)
	if err != nil {
		t.Fatal(err)
	}
	if defaults != 0 {
		t.Fatalf("expected identity metric count columns without defaults, got %d defaults", defaults)
	}

	var organization string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization); err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('workspace111', $1, 'test', 'test', 'Normal', '\x01', '\x01', '\x01')`, organization)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES ('connection11', 'workspace111', 'Source', '\x01')`)
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, time.August, 9, 10, 30, 0, 0, time.UTC)
	day := observedAt.Truncate(24 * time.Hour)
	_, err = database.Exec(ctx, `INSERT INTO identity_metrics
		(workspace, day, observed_at,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('workspace111', $1, $2, 0, 0, 0)`, day, observedAt.Format("15:04:05.999999"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('connection11', $1, 0, 0, 0)`, day)
	if err != nil {
		t.Fatalf("expected an explicit zero connection row to be accepted, got %v", err)
	}

	var organizationColumns int
	if err := database.QueryRow(ctx, `SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name IN ('identity_metrics', 'identity_connection_metrics')
			AND column_name = 'organization'`).Scan(&organizationColumns); err != nil {
		t.Fatal(err)
	}
	if organizationColumns != 0 {
		t.Fatalf("expected identity metric tables without organization columns, got %d", organizationColumns)
	}
	assertColumnDoesNotExist(t, database, "identity_connection_metrics", "observed_at")
	assertColumnDoesNotExist(t, database, "identity_connection_metrics", "workspace")
	var dataType string
	if err := database.QueryRow(ctx, `SELECT data_type
		FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'identity_metrics'
			AND column_name = 'observed_at'`).Scan(&dataType); err != nil {
		t.Fatal(err)
	}
	if dataType != "time without time zone" {
		t.Fatalf("expected identity_metrics.observed_at to be time without time zone, got %q", dataType)
	}

	if _, err := database.Exec(ctx, "DELETE FROM connections WHERE id = 'connection11'"); err != nil {
		t.Fatal(err)
	}
	var connectionRows int
	if err := database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_connection_metrics
		WHERE connection = 'connection11'`).Scan(&connectionRows); err != nil {
		t.Fatal(err)
	}
	if connectionRows != 0 {
		t.Fatalf("expected deleting a connection to cascade its metric rows, got %d", connectionRows)
	}
	var parentRows int
	if err := database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_metrics
		WHERE workspace = 'workspace111'`).Scan(&parentRows); err != nil {
		t.Fatal(err)
	}
	if parentRows != 1 {
		t.Fatalf("expected the parent workspace metric to survive a connection delete, got %d", parentRows)
	}
	_, err = database.Exec(ctx, `INSERT INTO connections
		(id, workspace, role, kms_encrypted_settings_key)
		VALUES ('connection22', 'workspace111', 'Source', '\x01')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO identity_connection_metrics
		(connection, day,
		identities_anonymous, identities_recognized, identities_without_profile)
		VALUES ('connection22', $1, 0, 0, 0)`, day)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(ctx, "DELETE FROM workspaces WHERE id = 'workspace111'"); err != nil {
		t.Fatal(err)
	}
	var rows int
	if err := database.QueryRow(ctx,
		"SELECT COUNT(*) FROM identity_metrics WHERE workspace = 'workspace111'").Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if rows != 0 {
		t.Fatalf("expected deleting workspace to cascade to identity_metrics, got %d rows", rows)
	}
	if err := database.QueryRow(ctx, `SELECT COUNT(*) FROM identity_connection_metrics
		WHERE connection = 'connection22'`).Scan(&connectionRows); err != nil {
		t.Fatal(err)
	}
	if connectionRows != 0 {
		t.Fatalf("expected deleting workspace connections to cascade their metric rows, got %d", connectionRows)
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
		t.Fatal(err)
	}

	return columns
}
