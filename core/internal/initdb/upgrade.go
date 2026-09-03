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

const consentPurposesTable = `
	CREATE TABLE IF NOT EXISTS consent_purposes (
		workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
		code varchar(100) NOT NULL CHECK (code ~ '^[A-Za-z_][0-9A-Za-z_]{0,99}$'),
		name varchar(100) NOT NULL,
		PRIMARY KEY (workspace, code)
	)`

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

// pipelineEventTypeUpgrade adds persisted ordering groups and event type
// identifier constraints.
const pipelineEventTypeUpgrade = `
	ALTER TABLE pipelines
		ADD COLUMN IF NOT EXISTS ordering_group varchar(25);

	UPDATE pipelines p
	SET ordering_group = CASE
		WHEN p.event_type = '' THEN ''
		WHEN c.connector IN ('dummy', 'google-analytics', 'mixpanel', 'posthog') THEN 'events'
		ELSE p.event_type
	END
	FROM connections c
	WHERE c.id = p.connection
		AND p.ordering_group IS NULL;

	ALTER TABLE pipelines
		ALTER COLUMN event_type TYPE varchar(25),
		ALTER COLUMN ordering_group TYPE varchar(25),
		ALTER COLUMN ordering_group SET NOT NULL;

	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT FROM pg_constraint
			WHERE conrelid = 'pipelines'::regclass
				AND conname = 'pipelines_event_type_check'
		) THEN
			ALTER TABLE pipelines
				ADD CONSTRAINT pipelines_event_type_check
				CHECK (event_type = '' OR event_type ~ '^[A-Za-z_][A-Za-z0-9_]*$');
		END IF;

		IF NOT EXISTS (
			SELECT FROM pg_constraint
			WHERE conrelid = 'pipelines'::regclass
				AND conname = 'pipelines_ordering_group_check'
		) THEN
			ALTER TABLE pipelines
				ADD CONSTRAINT pipelines_ordering_group_check
				CHECK ((event_type = '' AND ordering_group = '') OR
					(event_type <> '' AND ordering_group ~ '^[A-Za-z_][A-Za-z0-9_]*$'));
		END IF;
	END $$`

// pipelineDeliveryEndpointUpgrade adds persisted delivery endpoints and their
// identifier constraint.
const pipelineDeliveryEndpointUpgrade = `
	ALTER TABLE pipelines
		ADD COLUMN IF NOT EXISTS delivery_endpoint varchar(25);

	UPDATE pipelines
	SET delivery_endpoint = ''
	WHERE delivery_endpoint IS NULL;

	ALTER TABLE pipelines
		ALTER COLUMN delivery_endpoint TYPE varchar(25),
		ALTER COLUMN delivery_endpoint SET NOT NULL;

	DO $$
	BEGIN
		IF NOT EXISTS (
			SELECT FROM pg_constraint
			WHERE conrelid = 'pipelines'::regclass
				AND conname = 'pipelines_delivery_endpoint_check'
		) THEN
			ALTER TABLE pipelines
				ADD CONSTRAINT pipelines_delivery_endpoint_check
				CHECK (delivery_endpoint = '' OR
					(event_type <> '' AND delivery_endpoint ~ '^[A-Za-z_][A-Za-z0-9_]*$'));
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
			`ALTER TABLE metadata ADD COLUMN IF NOT EXISTS requests_rate_per_minute integer NOT NULL DEFAULT 100 CHECK (requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE metadata ADD COLUMN IF NOT EXISTS requests_max_capacity integer NOT NULL DEFAULT 100 CHECK (requests_max_capacity BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS members_limit integer NOT NULL DEFAULT 10000 CHECK (members_limit BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS access_keys_limit integer NOT NULL DEFAULT 1000 CHECK (access_keys_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspaces_limit integer NOT NULL DEFAULT 1000 CHECK (workspaces_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connectors_limit integer NOT NULL DEFAULT 1000 CHECK (connectors_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connections_limit integer NOT NULL DEFAULT 10000 CHECK (connections_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS pipelines_limit integer NOT NULL DEFAULT 10000 CHECK (pipelines_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS organization_requests_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (organization_requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS organization_requests_max_capacity integer NOT NULL DEFAULT 1000 CHECK (organization_requests_max_capacity BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_requests_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (workspace_requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_requests_max_capacity integer NOT NULL DEFAULT 1000 CHECK (workspace_requests_max_capacity BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_events_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (workspace_events_rate_per_minute BETWEEN 1000 AND 1000000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_events_max_capacity integer NOT NULL DEFAULT 20000 CHECK (workspace_events_max_capacity BETWEEN 20000 AND 100000)`,
			`ALTER TABLE organizations ALTER COLUMN members_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN access_keys_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspaces_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connectors_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connections_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN pipelines_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN organization_requests_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN organization_requests_max_capacity DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_requests_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_requests_max_capacity DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_events_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_events_max_capacity DROP DEFAULT`,
			`ALTER TABLE metadata ALTER COLUMN requests_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE metadata ALTER COLUMN requests_max_capacity DROP DEFAULT`,
			`CREATE TABLE IF NOT EXISTS rate_limit_buckets (
				subject_kind varchar(12) NOT NULL CHECK (subject_kind IN ('platform', 'organization', 'workspace', 'events')),
				subject_id varchar(12) NOT NULL CHECK (
					(subject_kind = 'platform' AND subject_id = 'platform')
					OR (subject_kind <> 'platform' AND subject_id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$')
				),
				organization varchar(12) REFERENCES organizations ON DELETE CASCADE,
				workspace varchar(12) REFERENCES workspaces ON DELETE CASCADE,
				available_units integer NOT NULL,
				capacity_units integer NOT NULL,
				rate_per_minute integer NOT NULL,
				last_refill_at timestamptz NOT NULL,
				refill_remainder integer NOT NULL,
				PRIMARY KEY (subject_kind, subject_id),
				CHECK (available_units >= 0),
				CHECK (
					(subject_kind IN ('platform', 'organization', 'workspace') AND capacity_units BETWEEN 1 AND 10000)
					OR (subject_kind = 'events' AND capacity_units BETWEEN 20000 AND 100000)
				),
				CHECK (available_units <= capacity_units),
				CHECK (
					(subject_kind IN ('platform', 'organization', 'workspace') AND rate_per_minute BETWEEN 60 AND 20000)
					OR (subject_kind = 'events' AND rate_per_minute BETWEEN 1000 AND 1000000)
				),
				CHECK (refill_remainder >= 0 AND refill_remainder < 60000000),
				CHECK (
					(
						subject_kind = 'platform'
						AND subject_id = 'platform'
						AND organization IS NULL
						AND workspace IS NULL
					)
					OR
					(
						subject_kind = 'organization'
						AND subject_id = organization
						AND workspace IS NULL
					)
					OR
					(
						subject_kind IN ('workspace', 'events')
						AND subject_id = workspace
						AND organization IS NULL
					)
				)
			)`,
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
			`DO $$
				BEGIN
					IF NOT EXISTS (
						SELECT FROM pg_attribute
						WHERE attrelid = 'discontinued_functions'::regclass
							AND attname = 'organization'
							AND NOT attisdropped
					) THEN
						ALTER TABLE discontinued_functions
							ADD COLUMN organization varchar(12) REFERENCES organizations ON DELETE SET NULL;

						ALTER TABLE discontinued_functions ADD COLUMN discontinued_at_reordered timestamp(0);
						UPDATE discontinued_functions SET discontinued_at_reordered = discontinued_at;
						ALTER TABLE discontinued_functions DROP COLUMN discontinued_at;
						ALTER TABLE discontinued_functions
							RENAME COLUMN discontinued_at_reordered TO discontinued_at;
						ALTER TABLE discontinued_functions ALTER COLUMN discontinued_at SET NOT NULL;
					END IF;
				END $$`,
			organizationConnectorReferencesView,
			nodeIDUpgrade,
			pipelineEventTypeUpgrade,
			pipelineDeliveryEndpointUpgrade,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'InviteMember' AFTER 'EndPipelineRun'`,
			consentPurposesTable,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'AddConsentPurpose'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'DeleteConsentPurpose'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'UpdateConsentPurpose'`,
			`ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS required_consents varchar(100)[] NOT NULL DEFAULT '{}'`,
			`ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS required_consents_operator varchar(3) NOT NULL DEFAULT 'and' CHECK (required_consents_operator IN ('and', 'or'))`,
			`UPDATE pipelines
				SET filter = regexp_replace(
					(
						CASE
							WHEN filter ? 'logical'
								AND filter ? 'conditions'
								AND NOT (filter ? 'operator')
								AND NOT (filter ? 'rules')
							THEN jsonb_build_object(
								'operator', filter->'logical',
								'rules', (
									SELECT COALESCE(
										jsonb_agg(rule ORDER BY position),
										'[]'::jsonb
									)
									FROM jsonb_array_elements(filter->'conditions') WITH ORDINALITY AS rules(rule, position)
								)
							)
							ELSE filter
						END
					)::text,
					'"operator"[[:space:]]*:[[:space:]]*"OpIsNotBetween"',
					'"operator":"IsNotBetween"',
					'g'
				)::jsonb
				WHERE filter IS NOT NULL
					AND (
						(
							filter ? 'logical'
							AND filter ? 'conditions'
							AND NOT (filter ? 'operator')
							AND NOT (filter ? 'rules')
						)
						OR filter::text ~ '"operator"[[:space:]]*:[[:space:]]*"OpIsNotBetween"'
					)`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS passed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS failed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN passed_6 DROP DEFAULT`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN failed_6 DROP DEFAULT`,
			`ALTER TABLE pipelines_runs ADD COLUMN IF NOT EXISTS passed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_runs ADD COLUMN IF NOT EXISTS failed_6 integer NOT NULL DEFAULT 0`,
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
				return fmt.Errorf("cannot execute upgrade query %q: %s", query, err)
			}
		}
		if _, err := tx.Exec(ctx, createRateLimiterLeasesFunction); err != nil {
			return fmt.Errorf("cannot create rate-limit lease function: %s", err)
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
