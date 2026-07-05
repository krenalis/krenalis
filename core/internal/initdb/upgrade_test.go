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
			workspace varchar(12) NOT NULL REFERENCES workspaces (id)
		);
		INSERT INTO organizations (id, name, enabled) VALUES ('111111111111', 'ACME inc', true);
		INSERT INTO workspaces (id, organization) VALUES ('222222222222', '111111111111');
		INSERT INTO connections (id, workspace) VALUES ('333333333333', '222222222222')`)
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

	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("second upgrade failed: %s", err)
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
