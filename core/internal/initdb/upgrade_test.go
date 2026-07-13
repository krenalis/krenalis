// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestUpgrade verifies that database upgrades are applied and are idempotent.
func TestUpgrade(t *testing.T) {
	const (
		databaseName = "krenalis"
		user         = "krenalis"
		password     = "krenalis"
	)

	ctx := t.Context()
	container, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(databaseName),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Error(err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	database, err := db.Open(&db.Options{
		Host:     host,
		Port:     int(port.Num()),
		Username: user,
		Password: password,
		Database: databaseName,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)

	_, err = database.Exec(ctx, `
		CREATE TYPE notification_name AS ENUM ('EndPipelineRun');
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
		CREATE INDEX pipelines_metrics_pipeline_idx ON pipelines_metrics (pipeline);
		INSERT INTO organizations (id, name, enabled) VALUES ('111111111111', 'ACME inc', true);
		INSERT INTO workspaces (id, organization) VALUES ('222222222222', '111111111111');
		INSERT INTO connections (id, workspace, connector, role) VALUES ('333333333333', '222222222222', 'dummy', 'Source');
		INSERT INTO pipelines (id, connection, target, format) VALUES ('444444444444', '333333333333', 'User', 'csv');
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
		INSERT INTO election (number, leader, date) VALUES (1, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', NOW())`)
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
	assertIndexExists(t, database, pipelinesMetricsOrganizationWorkspaceTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsOrganizationWorkspaceConnectionTargetTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsOrganizationTimeslotIndex)
	assertIndexExists(t, database, pipelinesMetricsTimeslotIndex)
	assertOrganizationConnectorReferences(t, database)
	assertNodeIDsUpgraded(t, database)
	assertPipelineMetricsUpgrade(t, database)
	assertPipelineMetricsColumnOrder(t, database)
	assertPipelineMetricsSurvivePipelineDelete(t, database)

	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("expected second upgrade to succeed, got %s", err)
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
		members     int
		accessKeys  int
		workspaces  int
		connectors  int
		connections int
		pipelines   int
	)
	err := database.QueryRow(t.Context(), `
		SELECT members_limit, access_keys_limit, workspaces_limit, connectors_limit, connections_limit, pipelines_limit
		FROM organizations
		WHERE id = '111111111111'`).Scan(&members, &accessKeys, &workspaces, &connectors, &connections, &pipelines)
	if err != nil {
		t.Fatal(err)
	}

	if members != 10000 || accessKeys != 1000 || workspaces != 1000 || connectors != 1000 ||
		connections != 10000 || pipelines != 10000 {
		t.Fatalf("expected limits members=%d access_keys=%d workspaces=%d connectors=%d connections=%d pipelines=%d, got members=%d access_keys=%d workspaces=%d connectors=%d connections=%d pipelines=%d",
			10000, 1000, 1000, 1000, 10000, 10000, members, accessKeys, workspaces, connectors, connections, pipelines)
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
	} {
		hasDefault, err := database.QueryExists(t.Context(), `
			SELECT FROM pg_attrdef d
			JOIN pg_attribute a ON a.attrelid = d.adrelid AND a.attnum = d.adnum
			JOIN pg_class c ON c.oid = d.adrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema()
				AND c.relname = 'organizations'
				AND a.attname = $1`, column)
		if err != nil {
			t.Fatal(err)
		}
		if hasDefault {
			t.Fatalf("expected column organizations.%s to have no default, got a default", column)
		}
	}
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
