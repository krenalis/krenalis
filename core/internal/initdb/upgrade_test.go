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

func TestUpgradeOrganizationResourceLimits(t *testing.T) {
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
		CREATE TABLE connections (
			id varchar(12) PRIMARY KEY,
			workspace varchar(12) NOT NULL REFERENCES workspaces (id),
			connector varchar NOT NULL
		);
		CREATE TABLE pipelines (
			id varchar(12) PRIMARY KEY,
			connection varchar(12) NOT NULL REFERENCES connections (id),
			format varchar
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
		INSERT INTO organizations (id, name, enabled) VALUES ('111111111111', 'ACME inc', true);
		INSERT INTO workspaces (id, organization) VALUES ('222222222222', '111111111111');
		INSERT INTO connections (id, workspace, connector) VALUES ('333333333333', '222222222222', 'dummy');
		INSERT INTO pipelines (id, connection, format) VALUES ('444444444444', '333333333333', 'csv');
		INSERT INTO pipelines_runs (id, pipeline, node) VALUES ('555555555555', '444444444444', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa');
		INSERT INTO election (number, leader, date) VALUES (1, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', NOW());
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
	assertOrganizationConnectorReferences(t, database)
	assertNodeIDsUpgraded(t, database)
	assertStateRequestSyncSchemaUpgraded(t, database)

	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("second upgrade failed: %s", err)
	}
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
		t.Fatalf("unexpected limits: members=%d access_keys=%d workspaces=%d connectors=%d connections=%d pipelines=%d",
			members, accessKeys, workspaces, connectors, connections, pipelines)
	}
}

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
			t.Fatalf("column organizations.%s has a default", column)
		}
	}
}

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
		t.Fatalf("index %s does not exist", name)
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
		t.Fatalf("unexpected organization connector references count: %d", count)
	}
}
