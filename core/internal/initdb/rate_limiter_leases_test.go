// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestRateLimitLeaseTimestampMonotonic verifies that an older batch cannot move
// a bucket's refill timestamp backwards after waiting for a lock on another
// row.
func TestRateLimitLeaseTimestampMonotonic(t *testing.T) {
	const workspaceID = "222222222222"

	ctx := t.Context()
	database := newInitializedTestDatabase(t)

	var organizationID string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations").Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	_, err := database.Exec(ctx, `
		INSERT INTO workspaces (
			id,
			organization,
			name,
			warehouse_name,
			warehouse_mode,
			warehouse_settings,
			kms_encrypted_warehouse_settings_key,
			kms_encrypted_warehouse_mcp_settings_key
		) VALUES ($1, $2, 'test', 'test', 'Normal', '\x01', '\x01', '\x01')`,
		workspaceID, organizationID)
	if err != nil {
		t.Fatal(err)
	}

	olderRequests := fmt.Sprintf(`[
		{"subject_kind":"events","subject_id":%q,"requested_units":1},
		{"subject_kind":"organization","subject_id":%q,"requested_units":1}
	]`, workspaceID, organizationID)
	newerRequest := fmt.Sprintf(`[
		{"subject_kind":"organization","subject_id":%q,"requested_units":1}
	]`, organizationID)

	// Create both authoritative buckets, then give them a common empty state.
	if err := acquireTestLeases(ctx, database, olderRequests); err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, `
		UPDATE rate_limit_buckets
		SET available_units = 0,
			last_refill_at = clock_timestamp(),
			refill_remainder = 0`)
	if err != nil {
		t.Fatal(err)
	}

	// Block the first subject in key order so a newer acquisition can update the
	// second subject before the older acquisition reaches it.
	blocker, err := database.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var kind string
	if err := blocker.QueryRow(ctx, `
		SELECT subject_kind
		FROM rate_limit_buckets
		WHERE subject_kind = 'events'
			AND subject_id = $1
		FOR UPDATE`, workspaceID).Scan(&kind); err != nil {
		_ = blocker.Rollback(ctx)
		t.Fatal(err)
	}

	older, err := database.Conn(ctx)
	if err != nil {
		_ = blocker.Rollback(ctx)
		t.Fatal(err)
	}
	defer older.Close()
	defer blocker.Rollback(ctx)
	var olderPID int
	if err := older.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&olderPID); err != nil {
		t.Fatal(err)
	}
	olderDone := make(chan error, 1)
	go func() {
		olderDone <- acquireTestLeases(ctx, older, olderRequests)
	}()

	waitForDatabaseLock(t, database, olderPID)

	if err := acquireTestLeases(ctx, database, newerRequest); err != nil {
		t.Fatal(err)
	}
	var newerTimestamp time.Time
	if err := database.QueryRow(ctx, `
		SELECT last_refill_at
		FROM rate_limit_buckets
		WHERE subject_kind = 'organization'
			AND subject_id = $1`, organizationID).Scan(&newerTimestamp); err != nil {
		t.Fatal(err)
	}

	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-olderDone; err != nil {
		t.Fatal(err)
	}

	var finalTimestamp time.Time
	if err := database.QueryRow(ctx, `
		SELECT last_refill_at
		FROM rate_limit_buckets
		WHERE subject_kind = 'organization'
			AND subject_id = $1`, organizationID).Scan(&finalTimestamp); err != nil {
		t.Fatal(err)
	}
	if finalTimestamp.Before(newerTimestamp) {
		t.Fatalf("rate-limit refill timestamp moved backwards from %s to %s", newerTimestamp, finalTimestamp)
	}
}

// TestRestoreRateLimitCapacityAfterConcurrentUpdate verifies that restoration
// follows an updated row version after waiting for its lock.
func TestRestoreRateLimitCapacityAfterConcurrentUpdate(t *testing.T) {
	ctx := t.Context()
	database := newInitializedTestDatabase(t)

	var organizationID string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations").Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	requests := fmt.Sprintf(`[
		{"subject_kind":"organization","subject_id":%q,"requested_units":1}
	]`, organizationID)
	if err := acquireTestLeases(ctx, database, requests); err != nil {
		t.Fatal(err)
	}
	_, err := database.Exec(ctx, `
		UPDATE rate_limit_buckets
		SET available_units = 10,
			capacity_units = 100
		WHERE subject_kind = 'organization'
			AND subject_id = $1`, organizationID)
	if err != nil {
		t.Fatal(err)
	}

	blocker, err := database.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var restorer *db.Conn
	defer func() {
		_ = blocker.Rollback(ctx)
		if restorer != nil {
			_ = restorer.Close()
		}
	}()
	_, err = blocker.Exec(ctx, `
		UPDATE rate_limit_buckets
		SET available_units = 5
		WHERE subject_kind = 'organization'
			AND subject_id = $1`, organizationID)
	if err != nil {
		t.Fatal(err)
	}

	restorer, err = database.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var restorerPID int
	if err := restorer.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&restorerPID); err != nil {
		t.Fatal(err)
	}
	restored := make(chan error, 1)
	go func() {
		_, err := restorer.Exec(ctx, "SELECT restore_rate_limit_capacity($1::jsonb)", fmt.Sprintf(`[
			{"subject_kind":"organization","subject_id":%q,"units":20}
		]`, organizationID))
		restored <- err
	}()

	waitForDatabaseLock(t, database, restorerPID)
	if err := blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-restored; err != nil {
		t.Fatal(err)
	}

	var available int
	if err := database.QueryRow(ctx, `
		SELECT available_units
		FROM rate_limit_buckets
		WHERE subject_kind = 'organization'
			AND subject_id = $1`, organizationID).Scan(&available); err != nil {
		t.Fatal(err)
	}
	if available != 25 {
		t.Fatalf("expected restored capacity 25, got %d", available)
	}
}

// TestAcquireRateLimitLeaseRejectsMissingBucket verifies that acquisition does
// not grant capacity when the authoritative bucket cannot be locked.
func TestAcquireRateLimitLeaseRejectsMissingBucket(t *testing.T) {
	ctx := t.Context()
	database := newInitializedTestDatabase(t)

	var organizationID string
	if err := database.QueryRow(ctx, "SELECT id FROM organizations").Scan(&organizationID); err != nil {
		t.Fatal(err)
	}
	_, err := database.Exec(ctx, `
		CREATE FUNCTION suppress_rate_limit_bucket_insert()
		RETURNS trigger
		LANGUAGE plpgsql
		AS $$
		BEGIN
			RETURN NULL;
		END;
		$$;

		CREATE TRIGGER suppress_rate_limit_bucket_insert
		BEFORE INSERT ON rate_limit_buckets
		FOR EACH ROW
		EXECUTE FUNCTION suppress_rate_limit_bucket_insert()`)
	if err != nil {
		t.Fatal(err)
	}
	requests := fmt.Sprintf(`[
		{"subject_kind":"organization","subject_id":%q,"requested_units":1}
	]`, organizationID)
	err = acquireTestLeases(ctx, database, requests)
	if err == nil || !strings.Contains(err.Error(), "rate-limit bucket for subject "+organizationID+" is not available") {
		t.Fatalf("expected missing rate-limit bucket error, got %v", err)
	}
}

// newInitializedTestDatabase creates a test database initialized with the
// current schema and database functions.
func newInitializedTestDatabase(t *testing.T) *db.DB {
	t.Helper()
	ctx := t.Context()
	database := newTestDatabase(t)
	if err := database.Transaction(ctx, func(tx *db.Tx) error {
		return initialize(ctx, tx, false)
	}); err != nil {
		t.Fatal(err)
	}
	return database
}

// waitForDatabaseLock waits until a PostgreSQL backend is blocked on a lock.
func waitForDatabaseLock(t *testing.T, database *db.DB, pid int) {
	t.Helper()
	ctx := t.Context()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting bool
		if err := database.QueryRow(ctx, `
			SELECT COALESCE((
				SELECT wait_event_type = 'Lock'
				FROM pg_stat_activity
				WHERE pid = $1
			), false)`, pid).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected backend %d to wait for a database lock, got no lock wait", pid)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func acquireTestLeases(ctx context.Context, connection db.Connection, requests string) error {
	var count int
	return connection.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM acquire_rate_limit_leases($1::jsonb)`, requests).Scan(&count)
}

func newTestDatabase(t *testing.T) *db.DB {
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
