//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package postgresql

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/json"
	"github.com/meergo/meergo/core/types"
	"github.com/meergo/meergo/testimages"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestUpdatesSQL just runs some basic tests on the 'updates.sql' file to ensure
// it doesn't contain syntax errors or obvious execution mistakes.
func TestUpdatesSQL(t *testing.T) {

	// Run the PostgreSQL container.
	ctx := context.Background()
	postgresContainer, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(testDatabase),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Error(err)
		}
	}()
	if err != nil {
		t.Fatal(err)
	}
	testHost, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testPort, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	settings, err := json.Marshal(map[string]any{
		"Host":     testHost,
		"Port":     testPort.Int(),
		"Username": testUser,
		"Password": testPassword,
		"Database": testDatabase,
		"Schema":   "public",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Open the data warehouse.
	wh, err := meergo.RegisteredWarehouseDriver("PostgreSQL").New(&meergo.WarehouseConfig{
		Settings: settings,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wh.Close()

	var columns = []meergo.Column{
		{Name: "email", Type: types.Text(), Nullable: true},
		{Name: "first_name", Type: types.Text(), Nullable: true},
		{Name: "last_name", Type: types.Text(), Nullable: true},
		{Name: "notes", Type: types.Array(types.Text()), Nullable: true},
	}

	err = wh.Initialize(ctx, columns)
	if err != nil {
		t.Fatalf("cannot initialize warehouse: %s", err)
	}

	pool, err := wh.(*PostgreSQL).connectionPool(context.Background())
	if err != nil {
		t.Fatalf("cannot open the warehouse: %s", err)
	}

	queries, err := os.ReadFile("updates.sql")
	if err != nil {
		t.Fatalf("cannot read queries SQL file: %s", err)
	}

	for i := range 3 {
		for query := range bytes.SplitSeq(queries, []byte(";\n")) {
			query := string(query)
			_, err := pool.Exec(ctx, query)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("repetition %d: query executed correctly: %q", i+1, query)
		}
	}
	t.Logf("all queries have been executed")
}
