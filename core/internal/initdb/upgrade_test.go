// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
)

// TestUpgrade verifies that database upgrades are applied and are idempotent.
func TestUpgrade(t *testing.T) {
	ctx := t.Context()
	database := newTestDatabase(t)

	_, err := database.Exec(ctx, `
		CREATE TYPE notification_name AS ENUM ('EndPipelineRun');
		CREATE TABLE metadata (
			singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
			installation_id text UNIQUE NOT NULL,
			kms_encrypted_cookie_key bytea NOT NULL,
			kms_encrypted_oauth_key bytea NOT NULL,
			kms_encrypted_notification_key bytea NOT NULL,
			kms_encrypted_api_key_pepper bytea NOT NULL
		);
		CREATE TABLE notifications (
			id bigint NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
			name notification_name NOT NULL,
			payload jsonb NOT NULL,
			PRIMARY KEY (id)
		);
		CREATE TABLE organizations (
			id varchar(12) PRIMARY KEY,
			name varchar(255) NOT NULL DEFAULT '',
			enabled boolean NOT NULL DEFAULT FALSE
		);
		CREATE TABLE workspaces (
			id varchar(12) PRIMARY KEY,
			organization varchar(12) NOT NULL REFERENCES organizations (id)
		);
		CREATE TYPE role AS ENUM ('Source', 'Destination');
		CREATE TYPE pipeline_target AS ENUM ('Event', 'User', 'Group');
		CREATE TABLE connections (
			id varchar(12) PRIMARY KEY,
			workspace varchar(12) NOT NULL REFERENCES workspaces (id),
			connector varchar NOT NULL,
			role role NOT NULL
		);
		CREATE TABLE pipelines (
			id varchar(12) PRIMARY KEY,
			connection varchar(12) NOT NULL REFERENCES connections (id),
			target pipeline_target NOT NULL,
			filter jsonb,
			event_type varchar(100) NOT NULL,
			format varchar
		);
		CREATE TABLE pipelines_metrics (
			pipeline varchar(12) NOT NULL REFERENCES pipelines ON DELETE CASCADE,
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
		CREATE TABLE pipelines_runs (
			id varchar(12) PRIMARY KEY,
			pipeline varchar(12) NOT NULL REFERENCES pipelines (id),
			node uuid
		);
		CREATE TABLE election (
			number integer PRIMARY KEY,
			leader uuid NOT NULL,
			date timestamp NOT NULL
		);
		CREATE TABLE discontinued_functions (
			id varchar(200) PRIMARY KEY,
			discontinued_at timestamp(0) NOT NULL
		);
		CREATE INDEX pipelines_metrics_pipeline_idx ON pipelines_metrics (pipeline);
		INSERT INTO organizations (id, name, enabled) VALUES ('111111111111', 'ACME inc', true);
		INSERT INTO workspaces (id, organization) VALUES ('222222222222', '111111111111');
		INSERT INTO connections (id, workspace, connector, role) VALUES
			('333333333333', '222222222222', 'dummy', 'Source'),
			('999999999999', '222222222222', 'dummy', 'Destination');
		INSERT INTO pipelines (id, connection, target, event_type, filter, format) VALUES
			(
				'444444444444',
				'333333333333',
				'User',
				'',
				'{
					"logical": "And",
					"conditions": [
						{"property": ["a"], "operator": "OpIsNotBetween", "values": [5, 10]},
						{"property": ["b"], "operator": "IsNull"}
					]
				}',
				'csv'
			),
			(
				'666666666666',
				'333333333333',
				'User',
				'',
				'{"operator":"Or","rules":[{"property":["b"],"operator":"IsNotBetween","values":[15,20]}]}',
				NULL
			),
			(
				'777777777777',
				'333333333333',
				'User',
				'',
				'{"operator":"And","rules":[{"rules":[{"property":["b"],"operator":"OpIsNotBetween","values":[25,30]},{"property":["literal"],"operator":"Contains","values":["OpIsNotBetween"]}],"operator":"Or"}]}',
				NULL
			),
			(
				'888888888888',
				'999999999999',
				'Event',
				'send_event_with_no_schema',
				NULL,
				NULL
			);
		INSERT INTO pipelines_metrics (
			pipeline, timeslot,
			passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
			failed_0, failed_1, failed_2, failed_3, failed_4, failed_5
		) VALUES (
			'444444444444', 1,
			1, 2, 3, 4, 5, 6,
			7, 8, 9, 10, 11, 12
		);
		INSERT INTO pipelines_runs (id, pipeline, node) VALUES ('555555555555', '444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa');
		INSERT INTO election (number, leader, date) VALUES (1, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', NOW());
		INSERT INTO discontinued_functions (id, discontinued_at) VALUES ('arn:aws:lambda:eu-west-1:1:function:transform.js', NOW());
		INSERT INTO metadata (installation_id, kms_encrypted_cookie_key, kms_encrypted_oauth_key, kms_encrypted_notification_key, kms_encrypted_api_key_pepper)
			VALUES ('test-installation', '\x01'::bytea, '\x02'::bytea, '\x03'::bytea, '\x04'::bytea);
		INSERT INTO notifications (id, name, payload) VALUES (1, 'EndPipelineRun', '{}'::jsonb)`)
	if err != nil {
		t.Fatal(err)
	}

	if err := Upgrade(ctx, database); err != nil {
		t.Fatal(err)
	}
	assertOrganizationLimits(t, database)
	assertOrganizationLimitsHaveNoDefaults(t, database)
	assertIndexExists(t, database, workspacesOrganizationIndex)
	assertIndexExists(t, database, connectionsWorkspaceIndex)
	assertIndexNotExists(t, database, pipelinesMetricsPipelineIndex)
	assertIndexNotExists(t, database, pipelinesMetricsOrganizationWorkspaceTimeslotIndex)
	assertIndexNotExists(t, database, pipelinesMetricsOrganizationWorkspaceConnectionTargetTimeslotIndex)
	assertIndexNotExists(t, database, pipelinesMetricsOrganizationConnectionTimeslotIndex)
	assertIndexNotExists(t, database, pipelinesMetricsOrganizationTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsWorkspaceTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsConnectionTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsTimeslotIndex)
	assertOrganizationConnectorReferences(t, database)
	assertNodeIDsUpgraded(t, database)
	assertPipelineFiltersUpgraded(t, database)
	assertPipelineEventTypesUpgraded(t, database)
	assertPipelineMetricsUpgrade(t, database)
	assertPipelineMetricsColumnOrder(t, database)
	assertUsageMetricsUpgrade(t, database)
	assertStateRequestSyncSchemaUpgraded(t, database)
	assertDiscontinuedFunctionsUpgrade(t, database)
	assertRateLimitLeaseFunction(t, database)
	assertConsentStepColumns(t, database)

	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("expected second upgrade to succeed, got %s", err)
	}
	assertPipelineFiltersUpgraded(t, database)
	assertPipelineEventTypesUpgraded(t, database)
	assertPipelineMetricsSurvivePipelineDelete(t, database)
	assertUsageMetricsUpgrade(t, database)
}

// TestUpgradePipelineOrderingGroup verifies column order, schema and data
// preservation, and idempotence on a complete pipelines table.
func TestUpgradePipelineOrderingGroup(t *testing.T) {

	const snapshotQuery = `
		SELECT jsonb_build_object(
			'table', 'pipelines'::regclass::oid,
			'columns', (
				SELECT jsonb_agg(jsonb_build_array(a.attname, format_type(a.atttypid, a.atttypmod),
					a.attnotnull, pg_get_expr(d.adbin, d.adrelid)) ORDER BY a.attnum)
				FROM pg_attribute a
				LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
				WHERE a.attrelid = 'pipelines'::regclass AND a.attnum > 0 AND NOT a.attisdropped),
			'constraints', (
				SELECT jsonb_agg(jsonb_build_array(conname,
					CASE WHEN contype = 'c' THEN contype::text ELSE pg_get_constraintdef(oid) END,
					convalidated) ORDER BY conname)
				FROM pg_constraint WHERE conrelid = 'pipelines'::regclass OR confrelid = 'pipelines'::regclass),
			'indexes', (
				SELECT jsonb_agg(pg_get_indexdef(indexrelid) ORDER BY indexrelid::regclass::text)
				FROM pg_index WHERE indrelid = 'pipelines'::regclass),
			'data', (SELECT jsonb_agg(to_jsonb(p) ORDER BY id) FROM pipelines p),
			'runs', (SELECT jsonb_agg(to_jsonb(r) ORDER BY id) FROM pipelines_runs r),
			'errors', (SELECT jsonb_agg(to_jsonb(e) ORDER BY pipeline) FROM pipelines_errors e),
			'view', (SELECT jsonb_agg(to_jsonb(v) ORDER BY resource) FROM organization_connector_references v)
		)::text`

	for _, name := range []string{"missing", "appended", "delivery-appended", "installed", "empty", "brevo", "klaviyo"} {

		t.Run(name, func(t *testing.T) {

			database := newInitializedTestDatabase(t)
			if name != "empty" {

				_, err := database.Exec(t.Context(), `
					INSERT INTO workspaces (id, organization, name, warehouse_name, warehouse_mode,
						warehouse_settings, kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
					SELECT '222222222222', id, 'Workspace', 'postgresql', 'Normal', '\x', '\x', '\x'
					FROM organizations LIMIT 1;
					INSERT INTO connections (id, workspace, connector, role, kms_encrypted_settings_key)
					VALUES ('333333333333', '222222222222', 'dummy', 'Source', '\x');
					INSERT INTO pipelines (id, connection, target, event_type, ordering_group, delivery_endpoint,
						name, enabled,
						schedule_start, schedule_period, in_schema, out_schema, filter, required_consents,
						required_consents_operator, transformation_mapping, transformation_id, transformation_version,
						transformation_language, transformation_source, transformation_preserve_json,
						transformation_in_paths, transformation_out_paths, query, format, path, sheet, compression,
						order_by, format_settings, export_mode, matching_in, matching_out, update_on_duplicates,
						table_name, table_key, user_id_column, updated_at_column, updated_at_format, incremental,
						cursor, health, properties_to_unset)
					VALUES ('444444444444', '333333333333', 'Event', repeat('界', 99) || '-', 'events', '', 'Pipeline', true,
						17, 5, '{"in":1}', '{"out":2}', '{"operator":"And","rules":[]}', '{purpose}',
						'or', '{"mapping":3}', 'function', 'v1', 'Python', 'source', true,
						'{in}', '{out}', 'SELECT 1', 'json', '/path', 'Sheet', 'Gzip',
						'name', '{"setting":4}', 'CreateOnly', 'in', 'out', true,
						'profiles', 'id', 'uid', 'updated', 'format', true,
						'2026-01-02 03:04:05', 'RecentError', '{property}');
					INSERT INTO pipelines (id, connection, target, event_type, ordering_group, delivery_endpoint,
						transformation_language, matching_in, matching_out, table_key, update_on_duplicates)
					VALUES ('666666666666', '333333333333', 'User', '', '', '', 'JavaScript', '', '', '', false);
					INSERT INTO pipelines_runs (id, pipeline, start_time, ping_time)
					VALUES ('555555555555', '444444444444', '2026-01-02 03:04:05', '2026-01-02 03:04:06');
					INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message)
					VALUES ('444444444444', 1, 0, 1, 'error')`)
				if err != nil {
					t.Fatal(err)
				}

				if name == "brevo" || name == "klaviyo" {
					_, err = database.Exec(t.Context(), "UPDATE connections SET connector = $1", name)
					if err != nil {
						t.Fatal(err)
					}
					_, err = database.Exec(t.Context(), `
						UPDATE pipelines SET event_type = 'create_event', ordering_group = 'create_event'
						WHERE target = 'Event'`)
					if err != nil {
						t.Fatal(err)
					}
				}

			}

			var expected string
			err := database.QueryRow(t.Context(), snapshotQuery).Scan(&expected)
			if err != nil {
				t.Fatal(err)
			}
			if name != "delivery-appended" && name != "installed" {
				_, err = database.Exec(t.Context(), `
					ALTER TABLE pipelines DROP COLUMN ordering_group;
					ALTER TABLE pipelines ALTER COLUMN event_type TYPE varchar(100)`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "delivery-appended" {
				_, err = database.Exec(t.Context(), `
					ALTER TABLE pipelines DROP COLUMN delivery_endpoint;
					ALTER TABLE pipelines ADD COLUMN delivery_endpoint varchar(16);
					UPDATE pipelines SET delivery_endpoint = '';
					ALTER TABLE pipelines ALTER COLUMN delivery_endpoint SET NOT NULL`)
				if err != nil {
					t.Fatal(err)
				}
			}
			if name == "appended" {
				_, err = database.Exec(t.Context(), pipelineEventTypeUpgrade)
				if err != nil {
					t.Fatal(err)
				}
			}

			var previousPosition int
			err = database.QueryRow(t.Context(), `
				SELECT MAX(attnum) FROM pg_attribute
				WHERE attrelid = 'pipelines'::regclass AND NOT attisdropped`).Scan(&previousPosition)
			if err != nil {
				t.Fatal(err)
			}
			for attempt := range 2 {

				err = Upgrade(t.Context(), database)
				if err != nil {
					t.Fatal(err)
				}
				var got string
				err = database.QueryRow(t.Context(), snapshotQuery).Scan(&got)
				if err != nil {
					t.Fatal(err)
				}
				if got != expected {
					t.Fatalf("expected preserved schema and data %s, got %s", expected, got)
				}

				var position int
				err = database.QueryRow(t.Context(), `
					SELECT MAX(attnum) FROM pg_attribute
					WHERE attrelid = 'pipelines'::regclass AND NOT attisdropped`).Scan(&position)
				if err != nil {
					t.Fatal(err)
				}
				if (attempt > 0 || name == "installed") && position != previousPosition {
					t.Fatalf("expected unchanged final column position %d, got %d", previousPosition, position)
				}
				previousPosition = position

			}

			if name != "empty" {

				for _, test := range []struct {
					assignment string
					constraint string
				}{
					{"schedule_start = -1", "pipelines_schedule_start_check"},
					{"schedule_start = 1440", "pipelines_schedule_start_check"},
					{"schedule_period = 1", "pipelines_schedule_period_check"},
					{"required_consents_operator = 'xor'", "pipelines_required_consents_operator_check"},
				} {
					_, err = database.Exec(t.Context(), "UPDATE pipelines SET "+test.assignment)
					if err != nil {
						if db.ErrConstraintName(err) != test.constraint {
							t.Fatalf("expected violation of %s, got %s", test.constraint, err)
						}
						continue
					}
					t.Fatalf("expected violation of %s, got nil", test.constraint)
				}

			}

		})

	}

}

// assertUsageMetricsUpgrade verifies the schema and defaults of the upgraded
// usage metrics table.
func assertUsageMetricsUpgrade(t *testing.T, database *db.DB) {
	t.Helper()
	assertConstraintExists(t, database, "usage_metrics", "usage_metrics_pkey")
	for _, constraint := range []string{
		"usage_metrics_profile_values",
		"usage_metrics_events",
	} {
		assertConstraintDoesNotExist(t, database, "usage_metrics", constraint)
	}
	assertIndexExists(t, database, "usage_metrics_organization_day_idx")

	hasWorkspaceFK, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint AS con
		JOIN pg_attribute AS attr
			ON attr.attrelid = con.conrelid AND attr.attnum = ANY(con.conkey)
		WHERE con.conrelid = 'usage_metrics'::regclass
			AND con.contype = 'f'
			AND attr.attname = 'workspace'`)
	if err != nil {
		t.Fatal(err)
	}
	if hasWorkspaceFK {
		t.Fatal("expected usage_metrics.workspace to have no foreign key")
	}

	const count = 3_000_000_000
	_, err = database.Exec(t.Context(), `
		INSERT INTO usage_metrics
			(organization, workspace, day, profiles,
			 profile_seconds, observed_at, events)
		VALUES ('111111111111', '222222222222', '2026-08-01', $1, $1, '12:00:00', $1)
		ON CONFLICT (organization, workspace, day) DO NOTHING`, count)
	if err != nil {
		t.Fatal(err)
	}
	var profileCount, profileSeconds, ingestedEvents int
	err = database.QueryRow(t.Context(), `
		SELECT profiles, profile_seconds, events
		FROM usage_metrics
		WHERE organization = '111111111111'
			AND workspace = '222222222222'
			AND day = '2026-08-01'`).Scan(&profileCount, &profileSeconds, &ingestedEvents)
	if err != nil {
		t.Fatal(err)
	}
	if profileCount != count || profileSeconds != count || ingestedEvents != count {
		t.Fatalf("expected 64-bit usage counts %d, got profiles=%d profile-seconds=%d events=%d",
			count, profileCount, profileSeconds, ingestedEvents)
	}

	_, err = database.Exec(t.Context(), `
		INSERT INTO usage_metrics (organization, workspace, day, events)
		VALUES ('111111111111', '333333333333', '2026-08-01', 1)
		ON CONFLICT (organization, workspace, day) DO NOTHING`)
	if err != nil {
		t.Fatal(err)
	}
	var observedAt *time.Time
	err = database.QueryRow(t.Context(), `
		SELECT profiles, profile_seconds, observed_at
		FROM usage_metrics
		WHERE organization = '111111111111'
			AND workspace = '333333333333'
			AND day = '2026-08-01'`).Scan(&profileCount, &profileSeconds, &observedAt)
	if err != nil {
		t.Fatal(err)
	}
	if profileCount != 0 || profileSeconds != 0 || observedAt != nil {
		t.Fatalf("expected event-only usage defaults, got profiles=%d profile-seconds=%d observed-at=%v",
			profileCount, profileSeconds, observedAt)
	}
}

func assertRateLimitLeaseFunction(t *testing.T, database *db.DB) {
	t.Helper()
	if _, err := database.Exec(t.Context(), `
		SELECT granted_units
		FROM acquire_rate_limit_leases($1::jsonb)`, `[
			{"subject_kind":"organization","subject_id":"111111111111","requested_units":101}
		]`); err == nil {
		t.Fatal("lease request above 100 units succeeded")
	}
	if _, err := database.Exec(t.Context(), `
		SELECT granted_units
		FROM acquire_rate_limit_leases($1::jsonb)`, `[
			{"subject_kind":"events","subject_id":"222222222222","requested_units":20001}
		]`); err == nil {
		t.Fatal("event lease request above 20,000 events succeeded")
	}

	_, err := database.Exec(t.Context(), `
		UPDATE metadata
		SET requests_rate_per_minute = 60,
			requests_max_capacity = 100
		WHERE singleton;
		UPDATE organizations
		SET organization_requests_rate_per_minute = 60,
			organization_requests_max_capacity = 100,
			workspace_requests_rate_per_minute = 60,
			workspace_requests_max_capacity = 100,
			workspace_events_rate_per_minute = 1000,
			workspace_events_max_capacity = 20000
		WHERE id = '111111111111'`)
	if err != nil {
		t.Fatal(err)
	}

	rows, err := database.Query(t.Context(), `
		SELECT subject_kind, subject_id, granted_units, capacity_units,
			available_units, rate_per_minute, refill_remainder
		FROM acquire_rate_limit_leases($1::jsonb)`, `[
			{"subject_kind":"platform","subject_id":"platform","requested_units":100},
			{"subject_kind":"organization","subject_id":"111111111111","requested_units":100},
			{"subject_kind":"workspace","subject_id":"222222222222","requested_units":100},
			{"subject_kind":"events","subject_id":"222222222222","requested_units":20000}
		]`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	grants := map[string]int{}
	for rows.Next() {
		var kind, id string
		var granted, capacity, available, rate, remainder int
		if err := rows.Scan(&kind, &id, &granted, &capacity, &available, &rate, &remainder); err != nil {
			t.Fatal(err)
		}
		wantCapacity := 100
		if kind == "events" {
			wantCapacity = 20000
		}
		if capacity != wantCapacity {
			t.Fatalf("expected capacity for %s %s %d, got %d", kind, id, wantCapacity, capacity)
		}
		if available != 0 || remainder != 0 {
			t.Fatalf("expected exhausted bucket state for %s %s, got available=%d remainder=%d", kind, id, available, remainder)
		}
		wantRate := 60
		if kind == "events" {
			wantRate = 1000
		}
		if rate != wantRate {
			t.Fatalf("expected rate for %s %s %d, got %d", kind, id, wantRate, rate)
		}
		grants[kind+":"+id] = granted
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if grants["platform:platform"] != 100 || grants["organization:111111111111"] != 100 ||
		grants["workspace:222222222222"] != 100 || grants["events:222222222222"] != 20000 {
		t.Fatalf("expected mixed batch grants for platform=100, organization=100, workspace=100, events=20,000, got %#v", grants)
	}

	// A second limiter process would execute the same database function. Its
	// request cannot obtain the tokens already leased by the first process.
	var granted int
	err = database.QueryRow(t.Context(), `
		SELECT granted_units
		FROM acquire_rate_limit_leases($1::jsonb)`, `[
			{"subject_kind":"organization","subject_id":"111111111111","requested_units":100}
		]`).Scan(&granted)
	if err != nil {
		t.Fatal(err)
	}
	if granted != 0 {
		t.Fatalf("expected second organization lease to grant 0 units, got %d", granted)
	}

	_, err = database.Exec(t.Context(), `
		UPDATE rate_limit_buckets
		SET available_units = 0,
			last_refill_at = clock_timestamp() - INTERVAL '30 seconds',
			refill_remainder = 0
		WHERE subject_kind = 'organization'
		  AND subject_id = '111111111111'`)
	if err != nil {
		t.Fatal(err)
	}
	err = database.QueryRow(t.Context(), `
		SELECT granted_units
		FROM acquire_rate_limit_leases($1::jsonb)`, `[
			{"subject_kind":"organization","subject_id":"111111111111","requested_units":30}
		]`).Scan(&granted)
	if err != nil {
		t.Fatal(err)
	}
	if granted != 30 {
		t.Fatalf("expected 30-second refill at 60 units per minute to grant 30 units, got %d", granted)
	}

	_, err = database.Exec(t.Context(), `
		SELECT restore_rate_limit_capacity($1::jsonb)`, `[
			{"subject_kind":"organization","subject_id":"111111111111","units":90},
			{"subject_kind":"organization","subject_id":"222222222222","units":90}
		]`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(t.Context(), `
		SELECT restore_rate_limit_capacity($1::jsonb)`, `[
			{"subject_kind":"organization","subject_id":"111111111111","units":20}
		]`)
	if err != nil {
		t.Fatal(err)
	}
	err = database.QueryRow(t.Context(), `
		SELECT available_units
		FROM rate_limit_buckets
		WHERE subject_kind = 'organization'
		  AND subject_id = '111111111111'`).Scan(&granted)
	if err != nil {
		t.Fatal(err)
	}
	if granted != 100 {
		t.Fatalf("expected restored capacity to be capped at 100 units, got %d", granted)
	}
	var count int
	err = database.QueryRow(t.Context(), `
		SELECT COUNT(*)
		FROM rate_limit_buckets
		WHERE subject_kind = 'organization'
		  AND subject_id = '222222222222'`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected missing subject to remain absent, got %d rows", count)
	}
}

// assertPipelineEventTypesUpgraded verifies event type identifier limits and
// persisted ordering and delivery metadata.
func assertPipelineEventTypesUpgraded(t *testing.T, database *db.DB) {

	t.Helper()

	var adjacent bool
	err := database.QueryRow(t.Context(), `
		SELECT g.attnum > e.attnum AND d.attnum > g.attnum AND NOT EXISTS (
			SELECT FROM pg_attribute a
			WHERE a.attrelid = e.attrelid AND NOT a.attisdropped
				AND ((a.attnum > e.attnum AND a.attnum < g.attnum)
					OR (a.attnum > g.attnum AND a.attnum < d.attnum))
		)
		FROM pg_attribute e
		JOIN pg_attribute g ON g.attrelid = e.attrelid AND g.attname = 'ordering_group' AND NOT g.attisdropped
		JOIN pg_attribute d ON d.attrelid = e.attrelid AND d.attname = 'delivery_endpoint' AND NOT d.attisdropped
		WHERE e.attrelid = 'pipelines'::regclass AND e.attname = 'event_type' AND NOT e.attisdropped`).Scan(&adjacent)
	if err != nil {
		t.Fatal(err)
	}
	if !adjacent {
		t.Fatal("expected ordering_group and delivery_endpoint immediately after event_type, got intervening columns")
	}

	for _, column := range []struct {
		name   string
		length int
	}{
		{"event_type", 100},
		{"ordering_group", 16},
		{"delivery_endpoint", 16},
	} {
		var length int
		err := database.QueryRow(t.Context(), `
			SELECT character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = 'pipelines'
				AND column_name = $1`, column.name).Scan(&length)
		if err != nil {
			t.Fatal(err)
		}
		if length != column.length {
			t.Fatalf("expected pipelines.%s to have length %d, got %d", column.name, column.length, length)
		}

	}

	var eventType, orderingGroup, deliveryEndpoint string
	err = database.QueryRow(t.Context(), `
		SELECT event_type, ordering_group, delivery_endpoint
		FROM pipelines
		WHERE id = '888888888888'`).Scan(&eventType, &orderingGroup, &deliveryEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	if eventType != "send_event_with_no_schema" || orderingGroup != "events" {
		t.Fatalf(
			"expected event type %q and ordering group %q, got %q and %q",
			"send_event_with_no_schema", "events", eventType, orderingGroup)
	}
	if deliveryEndpoint != "" {
		t.Fatalf("expected empty delivery endpoint, got %q", deliveryEndpoint)
	}

	err = database.QueryRow(t.Context(), `
		SELECT event_type, ordering_group, delivery_endpoint
		FROM pipelines
		WHERE id = '444444444444'`).Scan(&eventType, &orderingGroup, &deliveryEndpoint)
	if err != nil {
		t.Fatal(err)
	}
	if eventType != "" || orderingGroup != "" {
		t.Fatalf("expected empty event type and ordering group, got %q and %q", eventType, orderingGroup)
	}
	if deliveryEndpoint != "" {
		t.Fatalf("expected empty delivery endpoint, got %q", deliveryEndpoint)
	}

	assertConstraintDoesNotExist(t, database, "pipelines", "pipelines_event_type_check")
	assertConstraintDoesNotExist(t, database, "pipelines", "pipelines_ordering_group_check")
	assertConstraintDoesNotExist(t, database, "pipelines", "pipelines_delivery_endpoint_check")
}

func assertStateRequestSyncSchemaUpgraded(t *testing.T, database *db.DB) {
	t.Helper()

	assertColumnExists(t, database, "metadata", "kms_encrypted_http_secret_key")
	assertColumnDoesNotExist(t, database, "metadata", "kms_encrypted_cookie_key")
	assertColumnExists(t, database, "notifications", "version")
	assertColumnDoesNotExist(t, database, "notifications", "id")
	assertConstraintExists(t, database, "metadata", "metadata_kms_encrypted_http_secret_key_not_null")
	assertConstraintDoesNotExist(t, database, "metadata", "metadata_kms_encrypted_cookie_key_not_null")
	assertConstraintExists(t, database, "notifications", "notifications_version_not_null")
	assertConstraintDoesNotExist(t, database, "notifications", "notifications_id_not_null")

	var httpSecretKey []byte
	err := database.QueryRow(t.Context(), "SELECT kms_encrypted_http_secret_key FROM metadata WHERE singleton").Scan(&httpSecretKey)
	if err != nil {
		t.Fatal(err)
	}
	if string(httpSecretKey) != "\x01" {
		t.Fatalf("expected HTTP secret key %v, got %v", []byte{0x01}, httpSecretKey)
	}

	var version int
	err = database.QueryRow(t.Context(), "SELECT version FROM notifications").Scan(&version)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("expected notification version %d, got %d", 1, version)
	}
}

// assertDiscontinuedFunctionsUpgrade verifies that discontinued functions gain
// their organization column, which references the organizations table and is
// left NULL for the rows already in the table, whose organization cannot be
// recovered, and that the upgraded table keeps the canonical column order.
func assertDiscontinuedFunctionsUpgrade(t *testing.T, database *db.DB) {
	t.Helper()

	var organizationPosition, discontinuedAtPosition int
	err := database.QueryRow(t.Context(), `
		SELECT
			MAX(CASE WHEN attname = 'organization' THEN attnum END),
			MAX(CASE WHEN attname = 'discontinued_at' THEN attnum END)
		FROM pg_attribute
		WHERE attrelid = 'discontinued_functions'::regclass
			AND attname IN ('organization', 'discontinued_at')
			AND NOT attisdropped`).Scan(&organizationPosition, &discontinuedAtPosition)
	if err != nil {
		t.Fatal(err)
	}
	if organizationPosition >= discontinuedAtPosition {
		t.Fatalf("expected organization column before discontinued_at column, got organization=%d discontinued_at=%d", organizationPosition, discontinuedAtPosition)
	}

	var organization *string
	err = database.QueryRow(t.Context(), `
		SELECT organization
		FROM discontinued_functions
		WHERE id = 'arn:aws:lambda:eu-west-1:1:function:transform.js'`).Scan(&organization)
	if err != nil {
		t.Fatal(err)
	}
	if organization != nil {
		t.Fatalf("expected discontinued function organization NULL, got %q", *organization)
	}

	hasOrganizationFK, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint con
		JOIN pg_attribute attr ON attr.attrelid = con.conrelid AND attr.attnum = ANY(con.conkey)
		WHERE con.conrelid = 'discontinued_functions'::regclass
			AND con.contype = 'f'
			AND con.confrelid = 'organizations'::regclass
			AND con.confdeltype = 'n'
			AND attr.attname = 'organization'`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOrganizationFK {
		t.Fatal("expected discontinued_functions.organization to reference organizations on delete set null, got no such foreign key")
	}

	isNotNull, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_attribute
		WHERE attrelid = 'discontinued_functions'::regclass
			AND attname = 'organization'
			AND NOT attisdropped
			AND attnotnull`)
	if err != nil {
		t.Fatal(err)
	}
	if isNotNull {
		t.Fatal("expected column discontinued_functions.organization to be nullable, got a not-null column")
	}
}

// assertNodeIDsUpgraded verifies that UUID node IDs were migrated to string
// IDs.
func assertNodeIDsUpgraded(t *testing.T, database *db.DB) {
	t.Helper()

	for _, column := range []struct {
		table string
		name  string
	}{
		{"pipelines_runs", "node"},
		{"election", "leader"},
	} {
		var (
			dataType string
			length   int
		)
		err := database.QueryRow(t.Context(), `
			SELECT data_type, character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = $1
				AND column_name = $2`, column.table, column.name).Scan(&dataType, &length)
		if err != nil {
			t.Fatal(err)
		}
		if dataType != "character varying" || length != 22 {
			t.Fatalf("expected %s.%s to be varchar(22), got %s(%d)", column.table, column.name, dataType, length)
		}
	}

	var node *string
	err := database.QueryRow(t.Context(), "SELECT node FROM pipelines_runs WHERE id = '555555555555'").Scan(&node)
	if err != nil {
		t.Fatal(err)
	}
	if node != nil {
		t.Fatalf("expected upgraded pipeline run node to be NULL, got %q", *node)
	}

	var leader string
	err = database.QueryRow(t.Context(), "SELECT leader FROM election WHERE number = 1").Scan(&leader)
	if err != nil {
		t.Fatal(err)
	}
	if leader != "" {
		t.Fatalf("expected upgraded election leader to be empty, got %q", leader)
	}
}

// assertPipelineFiltersUpgraded verifies that pipeline filters are converted
// and legacy operator names are corrected.
func assertPipelineFiltersUpgraded(t *testing.T, database *db.DB) {

	t.Helper()

	tests := []struct {
		id     string
		filter string
	}{
		{
			id: "444444444444",
			filter: `{"operator":"And","rules":[` +
				`{"property":["a"],"operator":"IsNotBetween","values":[5,10]},` +
				`{"property":["b"],"operator":"IsNull"}` +
				`]}`,
		},
		{
			id:     "666666666666",
			filter: `{"operator":"Or","rules":[{"property":["b"],"operator":"IsNotBetween","values":[15,20]}]}`,
		},
		{
			id:     "777777777777",
			filter: `{"operator":"And","rules":[{"rules":[{"property":["b"],"operator":"IsNotBetween","values":[25,30]},{"property":["literal"],"operator":"Contains","values":["OpIsNotBetween"]}],"operator":"Or"}]}`,
		},
	}
	for _, test := range tests {
		exists, err := database.QueryExists(t.Context(), `
			SELECT FROM pipelines
			WHERE id = $1
				AND filter = $2::jsonb`, test.id, test.filter)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected pipeline %s to have filter %s", test.id, test.filter)
		}
	}

}

// assertPipelineMetricsUpgrade verifies that old pipeline metrics rows gain
// their new scope columns and constraints.
func assertPipelineMetricsUpgrade(t *testing.T, database *db.DB) {
	t.Helper()

	var organization, workspace, connection, target string
	err := database.QueryRow(t.Context(), `
		SELECT organization, workspace, connection, target
		FROM pipelines_metrics
		WHERE pipeline = '444444444444'
			AND timeslot = 1`).Scan(&organization, &workspace, &connection, &target)
	if err != nil {
		t.Fatal(err)
	}
	if organization != "111111111111" || workspace != "222222222222" || connection != "333333333333" || target != "User" {
		t.Fatalf("expected pipeline metrics scope organization=%q workspace=%q connection=%q target=%q, got organization=%q workspace=%q connection=%q target=%q",
			"111111111111", "222222222222", "333333333333", "User", organization, workspace, connection, target)
	}

	hasPipelineFK, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint con
		JOIN pg_attribute attr ON attr.attrelid = con.conrelid AND attr.attnum = ANY(con.conkey)
		WHERE con.conrelid = 'pipelines_metrics'::regclass
			AND con.contype = 'f'
			AND attr.attname = 'pipeline'`)
	if err != nil {
		t.Fatal(err)
	}
	if hasPipelineFK {
		t.Fatal("expected pipelines_metrics.pipeline to have no foreign key, got one")
	}

	hasOrganizationFK, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint con
		JOIN pg_attribute attr ON attr.attrelid = con.conrelid AND attr.attnum = ANY(con.conkey)
		WHERE con.conrelid = 'pipelines_metrics'::regclass
			AND con.contype = 'f'
			AND attr.attname = 'organization'`)
	if err != nil {
		t.Fatal(err)
	}
	if !hasOrganizationFK {
		t.Fatal("expected pipelines_metrics.organization to have a foreign key, got none")
	}

	expectedConstraints := []string{"pipelines_metrics_pkey", "pipelines_metrics_organization_fkey"}
	for _, column := range []string{
		"organization",
		"workspace",
		"connection",
		"pipeline",
		"target",
		"timeslot",
		"passed_0",
		"passed_1",
		"passed_2",
		"passed_3",
		"passed_4",
		"passed_5",
		"passed_6",
		"failed_0",
		"failed_1",
		"failed_2",
		"failed_3",
		"failed_4",
		"failed_5",
		"failed_6",
	} {
		expectedConstraints = append(expectedConstraints, "pipelines_metrics_"+column+"_not_null")
	}

	for _, constraint := range expectedConstraints {
		exists, err := database.QueryExists(t.Context(), `
			SELECT FROM pg_constraint
			WHERE conrelid = 'pipelines_metrics'::regclass
				AND conname = $1`, constraint)
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("expected constraint %s to exist, got missing constraint", constraint)
		}
	}

	hasReorderedConstraint, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint
		WHERE conrelid = 'pipelines_metrics'::regclass
			AND conname LIKE 'pipelines_metrics_reordered_%'`)
	if err != nil {
		t.Fatal(err)
	}
	if hasReorderedConstraint {
		t.Fatal("expected no pipelines_metrics_reordered_* constraint names, got at least one")
	}
}

// assertPipelineMetricsColumnOrder verifies that the upgraded metrics table
// keeps the canonical column order.
func assertPipelineMetricsColumnOrder(t *testing.T, database *db.DB) {
	t.Helper()

	var connectionPosition, pipelinePosition int
	err := database.QueryRow(t.Context(), `
		SELECT
			MAX(CASE WHEN attname = 'connection' THEN attnum END),
			MAX(CASE WHEN attname = 'pipeline' THEN attnum END)
		FROM pg_attribute
		WHERE attrelid = 'pipelines_metrics'::regclass
			AND attname IN ('connection', 'pipeline')
			AND NOT attisdropped`).Scan(&connectionPosition, &pipelinePosition)
	if err != nil {
		t.Fatal(err)
	}
	if connectionPosition >= pipelinePosition {
		t.Fatalf("expected connection column before pipeline column, got connection=%d pipeline=%d", connectionPosition, pipelinePosition)
	}
}

// assertPipelineMetricsSurvivePipelineDelete verifies that historical metrics
// are not deleted with their pipeline.
func assertPipelineMetricsSurvivePipelineDelete(t *testing.T, database *db.DB) {
	t.Helper()

	if _, err := database.Exec(t.Context(), `DELETE FROM pipelines_runs WHERE pipeline = '444444444444'`); err != nil {
		t.Fatal(err)
	}

	if _, err := database.Exec(t.Context(), `DELETE FROM pipelines WHERE id = '444444444444'`); err != nil {
		t.Fatal(err)
	}

	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM pipelines_metrics
		WHERE pipeline = '444444444444'
			AND timeslot = 1`)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected pipeline metrics to survive pipeline deletion, got no metrics row")
	}
}

// assertOrganizationLimits verifies that organization resource limits were
// backfilled.
func assertOrganizationLimits(t *testing.T, database *db.DB) {
	t.Helper()

	var (
		members                           int
		accessKeys                        int
		workspaces                        int
		connectors                        int
		connections                       int
		pipelines                         int
		organizationRequestsRatePerMinute int
		organizationRequestsMaxCapacity   int
		workspaceRequestsRatePerMinute    int
		workspaceRequestsMaxCapacity      int
		workspaceEventsRatePerMinute      int
		workspaceEventsMaxCapacity        int
	)
	err := database.QueryRow(t.Context(), `
			SELECT members_limit, access_keys_limit, workspaces_limit, connectors_limit, connections_limit, pipelines_limit,
				organization_requests_rate_per_minute, organization_requests_max_capacity,
				workspace_requests_rate_per_minute, workspace_requests_max_capacity,
				workspace_events_rate_per_minute, workspace_events_max_capacity
			FROM organizations
			WHERE id = '111111111111'`).Scan(&members, &accessKeys, &workspaces, &connectors, &connections, &pipelines,
		&organizationRequestsRatePerMinute, &organizationRequestsMaxCapacity,
		&workspaceRequestsRatePerMinute, &workspaceRequestsMaxCapacity, &workspaceEventsRatePerMinute, &workspaceEventsMaxCapacity)
	if err != nil {
		t.Fatal(err)
	}

	if members != 10000 || accessKeys != 1000 || workspaces != 1000 || connectors != 1000 ||
		connections != 10000 || pipelines != 10000 || organizationRequestsRatePerMinute != 1000 || organizationRequestsMaxCapacity != 1000 ||
		workspaceRequestsRatePerMinute != 1000 || workspaceRequestsMaxCapacity != 1000 ||
		workspaceEventsRatePerMinute != 1000 || workspaceEventsMaxCapacity != 20000 {
		t.Fatalf("expected default organization limits, got members=%d access_keys=%d workspaces=%d connectors=%d connections=%d pipelines=%d organization_requests_rate_per_minute=%d organization_requests_max_capacity=%d workspace_requests_rate_per_minute=%d workspace_requests_max_capacity=%d workspace_events_rate_per_minute=%d workspace_events_max_capacity=%d",
			members, accessKeys, workspaces, connectors, connections, pipelines, organizationRequestsRatePerMinute, organizationRequestsMaxCapacity,
			workspaceRequestsRatePerMinute, workspaceRequestsMaxCapacity, workspaceEventsRatePerMinute, workspaceEventsMaxCapacity)
	}
}

// assertOrganizationLimitsHaveNoDefaults verifies that resource limits no
// longer have database defaults.
func assertOrganizationLimitsHaveNoDefaults(t *testing.T, database *db.DB) {
	t.Helper()

	for _, column := range []string{
		"members_limit",
		"access_keys_limit",
		"workspaces_limit",
		"connectors_limit",
		"connections_limit",
		"pipelines_limit",
		"organization_requests_rate_per_minute",
		"organization_requests_max_capacity",
		"workspace_requests_rate_per_minute",
		"workspace_requests_max_capacity",
		"workspace_events_rate_per_minute",
		"workspace_events_max_capacity",
	} {
		if hasDefault(t, database, "organizations", column) {
			t.Fatalf("expected column organizations.%s to have no default, got a default", column)
		}
	}
}

// assertConsentStepColumns verifies that the consent step columns were added,
// keeping their default on pipelines_runs and dropping it on
// pipelines_metrics.
func assertConsentStepColumns(t *testing.T, database *db.DB) {
	t.Helper()

	for _, column := range []string{"passed_6", "failed_6"} {
		if !hasDefault(t, database, "pipelines_runs", column) {
			t.Fatalf("expected column pipelines_runs.%s to have a default, got no default", column)
		}
		if hasDefault(t, database, "pipelines_metrics", column) {
			t.Fatalf("expected column pipelines_metrics.%s to have no default, got a default", column)
		}
	}
}

// hasDefault reports whether table.column has a database default.
func hasDefault(t *testing.T, database *db.DB, table, column string) bool {
	t.Helper()

	found, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_attrdef d
		JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
		JOIN pg_class c ON c.oid = d.adrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND a.attname = $2`, table, column)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

// assertIndexExists verifies that an index with name exists.
func assertIndexExists(t *testing.T, database *db.DB, name string) {
	t.Helper()

	exists, err := database.QueryExists(t.Context(), `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'i'`, name)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected index %s to exist, got no index", name)
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
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected index %s to not exist, got existing index", name)
	}
}

func assertColumnExists(t *testing.T, database *db.DB, table, column string) {
	t.Helper()

	exists, err := columnExists(t, database, table, column)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected column %s.%s to exist, got missing column", table, column)
	}
}

func assertColumnDoesNotExist(t *testing.T, database *db.DB, table, column string) {
	t.Helper()

	exists, err := columnExists(t, database, table, column)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected column %s.%s to be missing, got existing column", table, column)
	}
}

func columnExists(t *testing.T, database *db.DB, table, column string) (bool, error) {
	t.Helper()

	return database.QueryExists(t.Context(), `
		SELECT FROM information_schema.columns
		WHERE table_schema = current_schema()
			AND table_name = $1
			AND column_name = $2`, table, column)
}

func assertConstraintExists(t *testing.T, database *db.DB, table, constraint string) {
	t.Helper()

	exists, err := constraintExists(t, database, table, constraint)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("expected constraint %s.%s to exist, got missing constraint", table, constraint)
	}
}

func assertConstraintDoesNotExist(t *testing.T, database *db.DB, table, constraint string) {
	t.Helper()

	exists, err := constraintExists(t, database, table, constraint)
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatalf("expected constraint %s.%s to be missing, got existing constraint", table, constraint)
	}
}

func constraintExists(t *testing.T, database *db.DB, table, constraint string) (bool, error) {
	t.Helper()

	return database.QueryExists(t.Context(), `
		SELECT FROM pg_constraint c
		JOIN pg_class t ON t.oid = c.conrelid
		JOIN pg_namespace n ON n.oid = t.relnamespace
		WHERE n.nspname = current_schema()
			AND t.relname = $1
			AND c.conname = $2`, table, constraint)
}

// assertOrganizationConnectorReferences verifies the upgraded organization
// connector references view.
func assertOrganizationConnectorReferences(t *testing.T, database *db.DB) {
	t.Helper()

	var count int
	err := database.QueryRow(t.Context(), `
		SELECT COUNT(*)
		FROM organization_connector_references
		WHERE organization = '111111111111'
			AND (
				(resource_type = 'connection' AND resource = '333333333333' AND connector = 'dummy')
				OR (resource_type = 'pipeline' AND resource = '444444444444' AND connector = 'csv')
			)`).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected two organization connector references, got %d", count)
	}
}
