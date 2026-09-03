// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"

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
	assertStateRequestSyncSchemaUpgraded(t, database)
	assertDiscontinuedFunctionsUpgrade(t, database)
	assertRateLimitLeaseFunction(t, database)
	assertConsentStepColumns(t, database)

	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("expected second upgrade to succeed, got %s", err)
	}
	assertPipelineFiltersUpgraded(t, database)
	assertPipelineMetricsSurvivePipelineDelete(t, database)
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
// persisted ordering groups.
func assertPipelineEventTypesUpgraded(t *testing.T, database *db.DB) {
	t.Helper()

	for _, column := range []string{"event_type", "ordering_group"} {
		var length int
		err := database.QueryRow(t.Context(), `
			SELECT character_maximum_length
			FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = 'pipelines'
				AND column_name = $1`, column).Scan(&length)
		if err != nil {
			t.Fatal(err)
		}
		if length != 25 {
			t.Fatalf("expected pipelines.%s to have length 25, got %d", column, length)
		}
	}

	var eventType, orderingGroup string
	err := database.QueryRow(t.Context(), `
		SELECT event_type, ordering_group
		FROM pipelines
		WHERE id = '888888888888'`).Scan(&eventType, &orderingGroup)
	if err != nil {
		t.Fatal(err)
	}
	if eventType != "send_event_with_no_schema" || orderingGroup != "events" {
		t.Fatalf("expected event type %q and ordering group %q, got %q and %q", "send_event_with_no_schema", "events", eventType, orderingGroup)
	}

	err = database.QueryRow(t.Context(), `
		SELECT event_type, ordering_group
		FROM pipelines
		WHERE id = '444444444444'`).Scan(&eventType, &orderingGroup)
	if err != nil {
		t.Fatal(err)
	}
	if eventType != "" || orderingGroup != "" {
		t.Fatalf("expected empty event type and ordering group, got %q and %q", eventType, orderingGroup)
	}

	assertConstraintExists(t, database, "pipelines", "pipelines_event_type_check")
	assertConstraintExists(t, database, "pipelines", "pipelines_ordering_group_check")
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
