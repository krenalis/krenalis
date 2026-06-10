// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestUpgradePipelineRunIndexes(t *testing.T) {
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

	_, err = database.Exec(ctx, `CREATE TABLE pipelines_runs (
		id text PRIMARY KEY,
		pipeline text NOT NULL,
		function text NOT NULL DEFAULT '',
		end_time timestamp
	);
	CREATE INDEX pipelines_runs_function_idx
		ON pipelines_runs (function)
		WHERE function != '' AND end_time IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}

	if err := Upgrade(ctx, database); err != nil {
		t.Fatal(err)
	}
	var predicate string
	err = database.QueryRow(ctx, `
		SELECT pg_get_expr(i.indpred, i.indrelid)
		FROM pg_index i
		JOIN pg_class c ON c.oid = i.indexrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema() AND c.relname = $1`,
		pipelineRunsFunctionIndex).Scan(&predicate)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(predicate, "end_time IS NULL") || strings.Contains(predicate, "end_time IS NOT NULL") {
		t.Fatalf("unexpected %s predicate: %s", pipelineRunsFunctionIndex, predicate)
	}
	if err := Upgrade(ctx, database); err != nil {
		t.Fatalf("second upgrade failed: %s", err)
	}

	_, err = database.Exec(ctx, `INSERT INTO pipelines_runs (id, pipeline) VALUES ('run-1', 'pipeline-1')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO pipelines_runs (id, pipeline, end_time) VALUES ('run-ended', 'pipeline-1', now())`)
	if err != nil {
		t.Fatalf("cannot insert completed run: %s", err)
	}
	_, err = database.Exec(ctx, `INSERT INTO pipelines_runs (id, pipeline) VALUES ('run-2', 'pipeline-1')`)
	if !db.IsUniqueViolation(err) || db.ErrConstraintName(err) != oneActivePipelineRunIndex {
		t.Fatalf("expected violation of %s, got %v", oneActivePipelineRunIndex, err)
	}

	_, err = database.Exec(ctx, `DROP INDEX `+oneActivePipelineRunIndex)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO pipelines_runs (id, pipeline) VALUES ('run-2', 'pipeline-1')`)
	if err != nil {
		t.Fatal(err)
	}
	err = Upgrade(ctx, database)
	if err == nil || !strings.Contains(err.Error(), "multiple active runs exist") {
		t.Fatalf("expected duplicate active runs error, got %v", err)
	}
}
