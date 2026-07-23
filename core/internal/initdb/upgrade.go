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
	workspacesOrganizationIndex                                        = "workspaces_organization_idx"
	connectionsWorkspaceIndex                                          = "connections_workspace_idx"
	pipelinesMetricsPipelineIndex                                      = "pipelines_metrics_pipeline_idx"
	pipelinesMetricsOrganizationWorkspaceTimeslotIndex                 = "pipelines_metrics_organization_workspace_timeslot_idx"
	pipelinesMetricsOrganizationWorkspaceConnectionTargetTimeslotIndex = "pipelines_metrics_org_ws_conn_target_timeslot_idx"
	pipelinesMetricsOrganizationConnectionTimeslotIndex                = "pipelines_metrics_organization_connection_timeslot_idx"
	pipelinesMetricsOrganizationTimeslotIndex                          = "pipelines_metrics_organization_timeslot_idx"
	pipelinesMetricsWorkspaceTimeslotIndex                             = "pipelines_metrics_workspace_timeslot_idx"
	pipelinesMetricsConnectionTimeslotIndex                            = "pipelines_metrics_connection_timeslot_idx"
	pipelinesMetricsTimeslotIndex                                      = "pipelines_metrics_timeslot_idx"
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
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS organization varchar(12) REFERENCES organizations ON DELETE CASCADE`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS workspace varchar(12)`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS connection varchar(12)`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS target pipeline_target`,
			`UPDATE pipelines_metrics m
				SET organization = w.organization,
					workspace = c.workspace,
					connection = c.id,
					target = p.target
				FROM pipelines p
				JOIN connections c ON c.id = p.connection
				JOIN workspaces w ON w.id = c.workspace
				WHERE m.pipeline = p.id`,
			`DELETE FROM pipelines_metrics WHERE organization IS NULL OR workspace IS NULL OR connection IS NULL OR target IS NULL`,
			`DO $$
				DECLARE
					pipeline_position integer;
					connection_position integer;
				BEGIN
					SELECT attnum INTO pipeline_position
					FROM pg_attribute
					WHERE attrelid = 'pipelines_metrics'::regclass
						AND attname = 'pipeline'
						AND NOT attisdropped;

					SELECT attnum INTO connection_position
					FROM pg_attribute
					WHERE attrelid = 'pipelines_metrics'::regclass
						AND attname = 'connection'
						AND NOT attisdropped;

					IF pipeline_position < connection_position THEN
						CREATE TABLE pipelines_metrics_reordered (
							organization varchar(12) NOT NULL REFERENCES organizations ON DELETE CASCADE,
							workspace varchar(12) NOT NULL,
							connection varchar(12) NOT NULL,
							pipeline varchar(12) NOT NULL,
							target pipeline_target NOT NULL,
							timeslot integer NOT NULL,
							passed_0 integer NOT NULL,
							passed_1 integer NOT NULL,
							passed_2 integer NOT NULL,
							passed_3 integer NOT NULL,
							passed_4 integer NOT NULL,
							passed_5 integer NOT NULL,
							failed_0 integer NOT NULL,
							failed_1 integer NOT NULL,
							failed_2 integer NOT NULL,
							failed_3 integer NOT NULL,
							failed_4 integer NOT NULL,
							failed_5 integer NOT NULL,
							PRIMARY KEY (pipeline, timeslot)
						);

						INSERT INTO pipelines_metrics_reordered (
							organization, workspace, connection, pipeline, target, timeslot,
							passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
							failed_0, failed_1, failed_2, failed_3, failed_4, failed_5
						)
						SELECT
							organization, workspace, connection, pipeline, target, timeslot,
							passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
							failed_0, failed_1, failed_2, failed_3, failed_4, failed_5
						FROM pipelines_metrics;

						DROP TABLE pipelines_metrics;
						ALTER TABLE pipelines_metrics_reordered RENAME TO pipelines_metrics;
					END IF;
				END $$`,
			`DO $$
				DECLARE
					c record;
				BEGIN
					FOR c IN
						SELECT
							conname AS old_name,
							'pipelines_metrics_' ||
								substr(conname, length('pipelines_metrics_reordered_') + 1) AS new_name
						FROM pg_constraint
						WHERE conrelid = 'pipelines_metrics'::regclass
							AND left(conname, length('pipelines_metrics_reordered_')) =
								'pipelines_metrics_reordered_'
					LOOP
						IF NOT EXISTS (
							SELECT FROM pg_constraint
							WHERE conrelid = 'pipelines_metrics'::regclass
								AND conname = c.new_name
						) THEN
							EXECUTE format('ALTER TABLE pipelines_metrics RENAME CONSTRAINT %I TO %I', c.old_name, c.new_name);
						END IF;
					END LOOP;
				END $$`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN organization SET NOT NULL`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN workspace SET NOT NULL`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN connection SET NOT NULL`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN target SET NOT NULL`,
			`ALTER TABLE pipelines_metrics DROP CONSTRAINT IF EXISTS pipelines_metrics_pipeline_fkey`,
			`DROP INDEX IF EXISTS ` + pipelinesMetricsPipelineIndex,
			`DROP INDEX IF EXISTS ` + pipelinesMetricsOrganizationWorkspaceTimeslotIndex,
			`DROP INDEX IF EXISTS ` + pipelinesMetricsOrganizationWorkspaceConnectionTargetTimeslotIndex,
			`DROP INDEX IF EXISTS ` + pipelinesMetricsOrganizationConnectionTimeslotIndex,
			`DROP INDEX IF EXISTS ` + pipelinesMetricsOrganizationTimeslotIndex,
			`CREATE INDEX IF NOT EXISTS ` + pipelinesMetricsWorkspaceTimeslotIndex + ` ON pipelines_metrics (workspace, timeslot)`,
			`CREATE INDEX IF NOT EXISTS ` + pipelinesMetricsConnectionTimeslotIndex + ` ON pipelines_metrics (connection, timeslot)`,
			`CREATE INDEX IF NOT EXISTS ` + pipelinesMetricsTimeslotIndex + ` ON pipelines_metrics (timeslot)`,
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
