// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/db"
)

const (
	workspacesOrganizationIndex = "workspaces_organization_idx"
	connectionsWorkspaceIndex   = "connections_workspace_idx"
)

const organizationConnectorReferencesView = `
	CREATE OR REPLACE VIEW organization_connector_references AS
	SELECT
		ws.organization,
		c.connector,
		'connection' AS resource_type,
		c.id AS resource
	FROM connections c
	JOIN workspaces ws ON ws.id = c.workspace
	UNION ALL
	SELECT
		ws.organization,
		p.format AS connector,
		'pipeline' AS resource_type,
		p.id AS resource
	FROM pipelines p
	JOIN connections c ON c.id = p.connection
	JOIN workspaces ws ON ws.id = c.workspace
	WHERE p.format IS NOT NULL`

const nodeIDUpgrade = `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = 'pipelines_runs'
				AND column_name = 'node'
				AND data_type = 'uuid'
		) THEN
			ALTER TABLE pipelines_runs
				ALTER COLUMN node TYPE varchar(22) USING NULL;
		END IF;

		IF EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = 'election'
				AND column_name = 'leader'
				AND data_type = 'uuid'
		) THEN
			ALTER TABLE election
				ALTER COLUMN leader TYPE varchar(22) USING '';
		END IF;

		IF NOT EXISTS (
			SELECT FROM pg_constraint
			WHERE conrelid = 'pipelines_runs'::regclass
				AND conname = 'pipelines_runs_node_check'
		) THEN
			ALTER TABLE pipelines_runs
				ADD CONSTRAINT pipelines_runs_node_check
				CHECK (node IS NULL OR node ~ '^[1-9A-HJ-NP-Za-km-z]{22}$');
		END IF;

		IF NOT EXISTS (
			SELECT FROM pg_constraint
			WHERE conrelid = 'election'::regclass
				AND conname = 'election_leader_check'
		) THEN
			ALTER TABLE election
				ADD CONSTRAINT election_leader_check
				CHECK (leader = '' OR leader ~ '^[1-9A-HJ-NP-Za-km-z]{22}$');
		END IF;
	END $$`

// Upgrade applies idempotent updates to an existing Krenalis PostgreSQL
// database.
func Upgrade(ctx context.Context, database *db.DB) error {

	initialized, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = 'organizations'
			AND c.relkind = 'r'`)
	if err != nil {
		return err
	}
	if !initialized {
		return fmt.Errorf("Krenalis's PostgreSQL database has not been initialized")
	}

	err = database.Transaction(ctx, func(tx *db.Tx) error {
		err := renameColumnIfExists(ctx, tx, "metadata", "kms_encrypted_cookie_key", "kms_encrypted_http_secret_key")
		if err != nil {
			return err
		}
		err = renameColumnIfExists(ctx, tx, "notifications", "id", "version")
		if err != nil {
			return err
		}
		err = renameConstraintIfExists(ctx, tx, "metadata", "metadata_kms_encrypted_cookie_key_not_null", "metadata_kms_encrypted_http_secret_key_not_null")
		if err != nil {
			return err
		}
		err = renameConstraintIfExists(ctx, tx, "notifications", "notifications_id_not_null", "notifications_version_not_null")
		if err != nil {
			return err
		}
		queries := []string{
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS members_limit integer NOT NULL DEFAULT 10000 CHECK (members_limit BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS access_keys_limit integer NOT NULL DEFAULT 1000 CHECK (access_keys_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspaces_limit integer NOT NULL DEFAULT 1000 CHECK (workspaces_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connectors_limit integer NOT NULL DEFAULT 1000 CHECK (connectors_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connections_limit integer NOT NULL DEFAULT 10000 CHECK (connections_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS pipelines_limit integer NOT NULL DEFAULT 10000 CHECK (pipelines_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ALTER COLUMN members_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN access_keys_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspaces_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connectors_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connections_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN pipelines_limit DROP DEFAULT`,
			`CREATE INDEX IF NOT EXISTS ` + workspacesOrganizationIndex + ` ON workspaces (organization)`,
			`CREATE INDEX IF NOT EXISTS ` + connectionsWorkspaceIndex + ` ON connections (workspace)`,
			organizationConnectorReferencesView,
			nodeIDUpgrade,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'InviteMember' AFTER 'EndPipelineRun'`,
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
				return fmt.Errorf("cannot execute upgrade query %q: %s", query, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	slog.Info("PostgreSQL database upgraded successfully")

	return nil
}

// renameColumnIfExists renames oldName to newName when oldName exists and
// newName does not. It is safe to run repeatedly.
func renameColumnIfExists(ctx context.Context, tx *db.Tx, table, oldName, newName string) error {
	oldExists, err := upgradeColumnExists(ctx, tx, table, oldName)
	if err != nil {
		return err
	}
	if !oldExists {
		return nil
	}
	newExists, err := upgradeColumnExists(ctx, tx, table, newName)
	if err != nil {
		return err
	}
	if newExists {
		return nil
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME COLUMN %s TO %s", table, oldName, newName))
	if err != nil {
		return fmt.Errorf("cannot rename column %s.%s to %s: %s", table, oldName, newName, err)
	}
	return nil
}

// renameConstraintIfExists renames oldName to newName when oldName exists. It
// is safe to run repeatedly.
func renameConstraintIfExists(ctx context.Context, tx *db.Tx, table, oldName, newName string) error {
	exists, err := upgradeConstraintExists(ctx, tx, table, oldName)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	_, err = tx.Exec(ctx, fmt.Sprintf("ALTER TABLE %s RENAME CONSTRAINT %s TO %s", table, oldName, newName))
	if err != nil {
		return fmt.Errorf("cannot rename constraint %s.%s to %s: %s", table, oldName, newName, err)
	}
	return nil
}

// upgradeConstraintExists reports whether table has a constraint named
// constraint.
func upgradeConstraintExists(ctx context.Context, tx *db.Tx, table, constraint string) (bool, error) {
	return tx.QueryExists(ctx, `
		SELECT FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
			AND t.relname = $1
			AND c.conname = $2`, table, constraint)
}

// upgradeColumnExists reports whether table has a column named column.
func upgradeColumnExists(ctx context.Context, tx *db.Tx, table, column string) (bool, error) {
	return tx.QueryExists(ctx, `
		SELECT FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = $1
			AND column_name = $2`, table, column)
}
