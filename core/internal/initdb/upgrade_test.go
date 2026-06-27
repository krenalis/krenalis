// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/kms"

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
	CREATE TABLE connections (
		id text PRIMARY KEY,
		connector text NOT NULL,
		settings bytea,
		kms_encrypted_settings_key bytea NOT NULL
	);
	CREATE INDEX pipelines_runs_function_idx
		ON pipelines_runs (function)
		WHERE function != '' AND end_time IS NOT NULL`)
	if err != nil {
		t.Fatal(err)
	}

	testKMS := newUpgradeTestKMS(t)
	if err := Upgrade(ctx, database, testKMS); err != nil {
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
	if err := Upgrade(ctx, database, testKMS); err != nil {
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
	if !db.IsUniqueViolation(err) || db.ErrConstraintName(err) != oneLivePipelineRunIndex {
		t.Fatalf("expected violation of %s, got %v", oneLivePipelineRunIndex, err)
	}

	_, err = database.Exec(ctx, `DROP INDEX `+oneLivePipelineRunIndex)
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `INSERT INTO pipelines_runs (id, pipeline) VALUES ('run-2', 'pipeline-1')`)
	if err != nil {
		t.Fatal(err)
	}
	err = Upgrade(ctx, database, testKMS)
	if err == nil || !strings.Contains(err.Error(), "multiple live runs exist") {
		t.Fatalf("expected duplicate live runs error, got %v", err)
	}
}

func TestUpgradeDummyOperationDelaySettings(t *testing.T) {
	ctx := t.Context()
	database := openUpgradeTestDB(t)
	_, err := database.Exec(ctx, `CREATE TABLE pipelines_runs (
		id text PRIMARY KEY,
		pipeline text NOT NULL,
		function text NOT NULL DEFAULT '',
		end_time timestamp
	);
	CREATE TABLE connections (
		id text PRIMARY KEY,
		connector text NOT NULL,
		settings bytea,
		kms_encrypted_settings_key bytea NOT NULL
	)`)
	if err != nil {
		t.Fatal(err)
	}

	testKMS := newUpgradeTestKMS(t)
	c := cipher.New(testKMS)
	t.Cleanup(c.Close)
	insertConnection := func(id, settings string) {
		t.Helper()
		encryptedSettings, settingsKey, err := c.Encrypt(ctx, []byte(settings))
		if err != nil {
			t.Fatal(err)
		}
		_, err = database.Exec(ctx, "INSERT INTO connections (id, connector, settings, kms_encrypted_settings_key) VALUES ($1, 'dummy', $2, $3)",
			id, encryptedSettings, settingsKey)
		if err != nil {
			t.Fatal(err)
		}
	}
	insertConnection("enabled", `{"simulateHTTPDelay":true}`)
	insertConnection("disabled", `{"simulateHTTPDelay":false}`)
	insertConnection("existing", `{"simulateHTTPDelay":true,"operationDelay":"3s"}`)
	insertConnection("untouched", `{"operationDelay":"777ms"}`)

	if err := Upgrade(ctx, database, testKMS); err != nil {
		t.Fatal(err)
	}
	if err := Upgrade(ctx, database, testKMS); err != nil {
		t.Fatalf("second upgrade failed: %s", err)
	}

	assertSettings := func(id, want string) {
		t.Helper()
		var encryptedSettings, settingsKey []byte
		err := database.QueryRow(ctx, "SELECT settings, kms_encrypted_settings_key FROM connections WHERE id = $1", id).
			Scan(&encryptedSettings, &settingsKey)
		if err != nil {
			t.Fatal(err)
		}
		settings, err := c.Decrypt(ctx, encryptedSettings, settingsKey)
		if err != nil {
			t.Fatal(err)
		}
		if string(settings) != want {
			t.Fatalf("settings for connection %s = %s, want %s", id, settings, want)
		}
	}
	assertSettings("enabled", `{"operationDelay":"2s"}`)
	assertSettings("disabled", `{}`)
	assertSettings("existing", `{"operationDelay":"3s"}`)
	assertSettings("untouched", `{"operationDelay":"777ms"}`)
}

func openUpgradeTestDB(t *testing.T) *db.DB {
	t.Helper()
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
	return database
}

func newUpgradeTestKMS(t *testing.T) kms.Kms {
	t.Helper()
	kms, err := kms.New(t.Context(), "key:"+base64.RawStdEncoding.EncodeToString(make([]byte, 32)))
	if err != nil {
		t.Fatal(err)
	}
	return kms
}
