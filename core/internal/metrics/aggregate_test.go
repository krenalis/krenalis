// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package metrics

import (
	"encoding/base64"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/dummy"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/kms"
	_ "github.com/krenalis/krenalis/warehouses/postgresql"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// newAggregateMetricsTestDatabase starts PostgreSQL and initializes the
// canonical schema for aggregate metric tests.
func newAggregateMetricsTestDatabase(t *testing.T) (*db.DB, string) {

	t.Helper()
	ctx := t.Context()
	container, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase("krenalis"),
		postgres.WithUsername("krenalis"),
		postgres.WithPassword("krenalis"),
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
		Host: host, Port: int(port.Num()), Username: "krenalis", Password: "krenalis", Database: "krenalis",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(database.Close)
	key, err := kms.New(ctx, "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if err := initdb.InitIfEmpty(ctx, database, key, false); err != nil {
		t.Fatal(err)
	}
	var organization string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations ORDER BY id LIMIT 1").Scan(&organization); err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO workspaces
		(id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
		kms_encrypted_warehouse_settings_key, kms_encrypted_warehouse_mcp_settings_key)
		VALUES ('workspace111', $1, 'metrics test', 'metrics test', 'Normal', $2, $2, $2)`,
		organization, []byte{1})
	if err != nil {
		t.Fatal(err)
	}

	return database, organization
}
