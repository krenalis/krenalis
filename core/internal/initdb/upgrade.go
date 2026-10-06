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

// pipelinesUpgrade adds the ordering_group and delivery_endpoint columns after
// event_type and the required consent columns after filter. As PostgreSQL can
// only append columns, it recreates every column following event_type, as
// declared in schema.sql, and then restores its values.
const pipelinesUpgrade = `
	DO $$
	DECLARE
		assignments text;
	BEGIN
		IF EXISTS (
			SELECT FROM pg_attribute
			WHERE attrelid = 'pipelines'::regclass AND attname = 'ordering_group' AND NOT attisdropped
		) THEN
			RETURN;
		END IF;

		LOCK TABLE pipelines IN ACCESS EXCLUSIVE MODE;
		CREATE TEMP TABLE pipelines_upgrade ON COMMIT DROP AS SELECT * FROM pipelines;
		DROP VIEW organization_connector_references;

		ALTER TABLE pipelines
			DROP COLUMN name,
			DROP COLUMN enabled,
			DROP COLUMN schedule_start,
			DROP COLUMN schedule_period,
			DROP COLUMN in_schema,
			DROP COLUMN out_schema,
			DROP COLUMN filter,
			DROP COLUMN transformation_mapping,
			DROP COLUMN transformation_id,
			DROP COLUMN transformation_version,
			DROP COLUMN transformation_language,
			DROP COLUMN transformation_source,
			DROP COLUMN transformation_preserve_json,
			DROP COLUMN transformation_in_paths,
			DROP COLUMN transformation_out_paths,
			DROP COLUMN query,
			DROP COLUMN format,
			DROP COLUMN path,
			DROP COLUMN sheet,
			DROP COLUMN compression,
			DROP COLUMN order_by,
			DROP COLUMN format_settings,
			DROP COLUMN export_mode,
			DROP COLUMN matching_in,
			DROP COLUMN matching_out,
			DROP COLUMN update_on_duplicates,
			DROP COLUMN table_name,
			DROP COLUMN table_key,
			DROP COLUMN user_id_column,
			DROP COLUMN updated_at_column,
			DROP COLUMN updated_at_format,
			DROP COLUMN incremental,
			DROP COLUMN cursor,
			DROP COLUMN health,
			DROP COLUMN properties_to_unset;

		-- Columns declared NOT NULL without a default are added as nullable and
		-- become NOT NULL once their values have been restored.
		ALTER TABLE pipelines
			ADD COLUMN ordering_group varchar(16),
			ADD COLUMN delivery_endpoint varchar(16),
			ADD COLUMN name varchar(60) NOT NULL DEFAULT '',
			ADD COLUMN enabled boolean NOT NULL DEFAULT FALSE,
			ADD COLUMN schedule_start smallint NOT NULL DEFAULT 0 CHECK (schedule_start >= 0 AND schedule_start < 1440),
			ADD COLUMN schedule_period smallint NOT NULL DEFAULT 0 CHECK(schedule_period IN (0, 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440)),
			ADD COLUMN in_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
			ADD COLUMN out_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
			ADD COLUMN filter jsonb,
			ADD COLUMN required_consents_operator varchar(3) NOT NULL DEFAULT 'and' CHECK (required_consents_operator IN ('and', 'or')),
			ADD COLUMN required_consents_purposes varchar(12)[] NOT NULL DEFAULT '{}',
			ADD COLUMN transformation_mapping jsonb,
			ADD COLUMN transformation_id varchar(200) NOT NULL DEFAULT '',
			ADD COLUMN transformation_version varchar(128) NOT NULL DEFAULT '',
			ADD COLUMN transformation_language transformation_language,
			ADD COLUMN transformation_source text NOT NULL DEFAULT '',
			ADD COLUMN transformation_preserve_json boolean NOT NULL DEFAULT false,
			ADD COLUMN transformation_in_paths varchar[],
			ADD COLUMN transformation_out_paths varchar[],
			ADD COLUMN query text NOT NULL DEFAULT '',
			ADD COLUMN format varchar,
			ADD COLUMN path varchar(1024) NOT NULL DEFAULT '',
			ADD COLUMN sheet varchar(31) NOT NULL DEFAULT '',
			ADD COLUMN compression compression NOT NULL DEFAULT '',
			ADD COLUMN order_by varchar(1024) NOT NULL DEFAULT '',
			ADD COLUMN format_settings jsonb,
			ADD COLUMN export_mode export_mode NOT NULL DEFAULT '',
			ADD COLUMN matching_in text,
			ADD COLUMN matching_out text,
			ADD COLUMN update_on_duplicates boolean,
			ADD COLUMN table_name varchar(1024) NOT NULL DEFAULT '',
			ADD COLUMN table_key text,
			ADD COLUMN user_id_column varchar(1024) NOT NULL DEFAULT '',
			ADD COLUMN updated_at_column varchar(1024) NOT NULL DEFAULT '',
			ADD COLUMN updated_at_format varchar(64) NOT NULL DEFAULT '',
			ADD COLUMN incremental boolean NOT NULL DEFAULT FALSE,
			ADD COLUMN cursor timestamp NOT NULL DEFAULT '0001-01-01 00:00:00+00',
			ADD COLUMN health health NOT NULL DEFAULT 'Healthy',
			ADD COLUMN properties_to_unset varchar[];

		SELECT string_agg(format('%1$I = b.%1$I', attname), ', ' ORDER BY attnum) INTO assignments
		FROM pg_attribute
		WHERE attrelid = 'pg_temp.pipelines_upgrade'::regclass AND attnum > 0 AND NOT attisdropped
			AND attname NOT IN ('id', 'connection', 'target', 'event_type');
		EXECUTE format('UPDATE pipelines p SET %s FROM pg_temp.pipelines_upgrade b WHERE p.id = b.id', assignments);

		UPDATE pipelines p
		SET ordering_group = CASE
				WHEN p.event_type = '' THEN ''
				WHEN c.connector IN ('dummy', 'google-analytics', 'mixpanel', 'posthog') THEN 'events'
				WHEN c.connector IN ('brevo', 'klaviyo') THEN 'create_event'
			END,
			delivery_endpoint = ''
		FROM connections c
		WHERE c.id = p.connection;

		ALTER TABLE pipelines
			ALTER COLUMN ordering_group SET NOT NULL,
			ALTER COLUMN delivery_endpoint SET NOT NULL,
			ALTER COLUMN transformation_language SET NOT NULL,
			ALTER COLUMN matching_in SET NOT NULL,
			ALTER COLUMN matching_out SET NOT NULL,
			ALTER COLUMN update_on_duplicates SET NOT NULL,
			ALTER COLUMN table_key SET NOT NULL;

		CREATE UNIQUE INDEX pipelines_transformation_id_idx ON pipelines (transformation_id) WHERE transformation_id <> '';
	END $$`

// pipelineMetricStepsUpgrade adds the consent steps to the six-step counters of
// pipeline metrics, runs, and errors. Released step indices 3, 4, and 5 become
// 5, 6, and 8. As no consent was required before, the EventConsent and
// ImportProfileConsent steps pass whatever reached them in the pipelines that
// count them, and no step fails. As PostgreSQL can only append columns, it
// recreates the columns following passed_5, as declared in schema.sql, and
// then restores their values.
const pipelineMetricStepsUpgrade = `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT FROM pg_attribute
			WHERE attrelid = 'pipelines_metrics'::regclass AND attname = 'passed_8' AND NOT attisdropped
		) THEN
			RETURN;
		END IF;

		-- Upgrade the metrics.
		LOCK TABLE pipelines_metrics IN ACCESS EXCLUSIVE MODE;
		CREATE TEMP TABLE pipelines_metrics_upgrade ON COMMIT DROP AS SELECT * FROM pipelines_metrics;
		ALTER TABLE pipelines_metrics
			DROP COLUMN failed_0, DROP COLUMN failed_1, DROP COLUMN failed_2,
			DROP COLUMN failed_3, DROP COLUMN failed_4, DROP COLUMN failed_5,
			ADD COLUMN passed_6 integer NOT NULL DEFAULT 0,
			ADD COLUMN passed_7 integer NOT NULL DEFAULT 0,
			ADD COLUMN passed_8 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_0 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_1 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_2 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_3 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_4 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_5 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_6 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_7 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_8 integer NOT NULL DEFAULT 0;
		-- Event-based imports write the records without a transformation as soon
		-- as they pass the filter, without counting them, and transform the
		-- others asynchronously, possibly in a later timeslot. Up to a timeslot,
		-- the untransformed records cannot exceed the records that passed the
		-- filter minus those that entered the transformation, up to that or any
		-- later timeslot, and the smallest of these differences is used.
		CREATE TEMP TABLE pipelines_metrics_untransformed ON COMMIT DROP AS
		SELECT pipeline, timeslot,
			total - COALESCE(LAG(total) OVER (PARTITION BY pipeline ORDER BY timeslot), 0) AS untransformed
		FROM (
			SELECT pipeline, timeslot,
				GREATEST(MIN(difference) OVER (PARTITION BY pipeline ORDER BY timeslot DESC), 0) AS total
			FROM (
				SELECT b.pipeline, b.timeslot,
					SUM(b.passed_2::bigint - b.passed_3 - b.failed_3) OVER (PARTITION BY b.pipeline ORDER BY b.timeslot)
						AS difference
				FROM pg_temp.pipelines_metrics_upgrade b
				JOIN connections c ON c.id = b.connection
				WHERE b.target = 'User' AND c.role = 'Source' AND c.connector IN ('android', 'dotnet', 'go', 'ios',
					'java', 'javascript', 'nodejs', 'python', 'rudderstack', 'segment', 'webhook')
			) AS d
		) AS t;
		UPDATE pipelines_metrics m SET
			passed_3 = CASE WHEN b.target = 'Event' THEN b.passed_2 ELSE 0 END,
			passed_4 = 0, passed_5 = b.passed_3, passed_6 = b.passed_4,
			passed_7 = CASE
				WHEN b.target <> 'User' OR c.role IS DISTINCT FROM 'Source' THEN 0
				ELSE b.passed_4 + COALESCE(u.untransformed, 0)
			END,
			passed_8 = b.passed_5,
			failed_0 = b.failed_0, failed_1 = b.failed_1, failed_2 = b.failed_2,
			failed_5 = b.failed_3, failed_6 = b.failed_4, failed_8 = b.failed_5
		FROM pg_temp.pipelines_metrics_upgrade b
		LEFT JOIN connections c ON c.id = b.connection
		LEFT JOIN pg_temp.pipelines_metrics_untransformed u ON u.pipeline = b.pipeline AND u.timeslot = b.timeslot
		WHERE m.pipeline = b.pipeline AND m.timeslot = b.timeslot;
		ALTER TABLE pipelines_metrics
			ALTER COLUMN passed_6 DROP DEFAULT,
			ALTER COLUMN passed_7 DROP DEFAULT,
			ALTER COLUMN passed_8 DROP DEFAULT,
			ALTER COLUMN failed_0 DROP DEFAULT,
			ALTER COLUMN failed_1 DROP DEFAULT,
			ALTER COLUMN failed_2 DROP DEFAULT,
			ALTER COLUMN failed_3 DROP DEFAULT,
			ALTER COLUMN failed_4 DROP DEFAULT,
			ALTER COLUMN failed_5 DROP DEFAULT,
			ALTER COLUMN failed_6 DROP DEFAULT,
			ALTER COLUMN failed_7 DROP DEFAULT,
			ALTER COLUMN failed_8 DROP DEFAULT;

		-- Upgrade the runs.
		LOCK TABLE pipelines_runs IN ACCESS EXCLUSIVE MODE;
		CREATE TEMP TABLE pipelines_runs_upgrade ON COMMIT DROP AS SELECT * FROM pipelines_runs;
		ALTER TABLE pipelines_runs
			DROP COLUMN failed_0, DROP COLUMN failed_1, DROP COLUMN failed_2,
			DROP COLUMN failed_3, DROP COLUMN failed_4, DROP COLUMN failed_5,
			DROP COLUMN error,
			ADD COLUMN passed_6 integer NOT NULL DEFAULT 0,
			ADD COLUMN passed_7 integer NOT NULL DEFAULT 0,
			ADD COLUMN passed_8 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_0 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_1 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_2 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_3 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_4 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_5 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_6 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_7 integer NOT NULL DEFAULT 0,
			ADD COLUMN failed_8 integer NOT NULL DEFAULT 0,
			ADD COLUMN error varchar NOT NULL DEFAULT '';
		UPDATE pipelines_runs r SET
			passed_3 = CASE WHEN p.target = 'Event' THEN b.passed_2 ELSE 0 END,
			passed_4 = 0, passed_5 = b.passed_3, passed_6 = b.passed_4,
			passed_7 = CASE WHEN p.target = 'User' AND c.role = 'Source' THEN b.passed_4 ELSE 0 END,
			passed_8 = b.passed_5,
			failed_0 = b.failed_0, failed_1 = b.failed_1, failed_2 = b.failed_2,
			failed_5 = b.failed_3, failed_6 = b.failed_4, failed_8 = b.failed_5,
			error = b.error
		FROM pg_temp.pipelines_runs_upgrade b
		JOIN pipelines p ON p.id = b.pipeline
		JOIN connections c ON c.id = p.connection
		WHERE r.id = b.id;

		-- Upgrade the errors.
		UPDATE pipelines_errors SET step = CASE step
			WHEN 3 THEN 5
			WHEN 4 THEN 6
			WHEN 5 THEN 8
		END
		WHERE step BETWEEN 3 AND 5;
	END $$`

// Upgrade upgrades an existing Krenalis PostgreSQL database from the schema of
// release v0.43.0 to the current schema. It is idempotent.
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
		queries := []string{
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS organization_requests_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (organization_requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS organization_requests_max_capacity integer NOT NULL DEFAULT 1000 CHECK (organization_requests_max_capacity BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_requests_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (workspace_requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_requests_max_capacity integer NOT NULL DEFAULT 1000 CHECK (workspace_requests_max_capacity BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_events_rate_per_minute integer NOT NULL DEFAULT 1000 CHECK (workspace_events_rate_per_minute BETWEEN 1000 AND 1000000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspace_events_max_capacity integer NOT NULL DEFAULT 20000 CHECK (workspace_events_max_capacity BETWEEN 20000 AND 100000)`,
			`ALTER TABLE organizations ALTER COLUMN organization_requests_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN organization_requests_max_capacity DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_requests_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_requests_max_capacity DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_events_rate_per_minute DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspace_events_max_capacity DROP DEFAULT`,
			`ALTER TABLE metadata ADD COLUMN IF NOT EXISTS requests_rate_per_minute integer NOT NULL DEFAULT 100 CHECK (requests_rate_per_minute BETWEEN 60 AND 20000)`,
			`ALTER TABLE metadata ADD COLUMN IF NOT EXISTS requests_max_capacity integer NOT NULL DEFAULT 100 CHECK (requests_max_capacity BETWEEN 1 AND 10000)`,
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
			`CREATE TABLE IF NOT EXISTS consent_purposes (
				id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
				workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
				name varchar(100) NOT NULL,
				event_purpose_codes varchar(1024)[] NOT NULL DEFAULT '{}',
				profile_property varchar(1024) NOT NULL DEFAULT '',
				profile_json_key varchar(1024) NOT NULL DEFAULT '',
				PRIMARY KEY (id)
			)`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'AddConsentPurpose' AFTER 'AcceptInvitation'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'DeleteConsentPurpose' AFTER 'DeleteConnection'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'UpdateConsentPurpose' AFTER 'UpdateConnection'`,
			pipelinesUpgrade,
			organizationConnectorReferencesView,
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
			pipelineMetricStepsUpgrade,
			`CREATE TABLE IF NOT EXISTS usage_metrics (
				organization varchar(12) NOT NULL REFERENCES organizations ON DELETE CASCADE,
				workspace varchar(12) NOT NULL,
				day date NOT NULL,
				profiles bigint NOT NULL DEFAULT 0,
				profile_seconds bigint NOT NULL DEFAULT 0,
				observed_at time without time zone,
				events bigint NOT NULL DEFAULT 0,
				PRIMARY KEY (organization, workspace, day)
			)`,
			`CREATE INDEX IF NOT EXISTS usage_metrics_organization_day_idx ON usage_metrics (organization, day)`,
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
