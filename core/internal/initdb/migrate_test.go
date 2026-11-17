// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"sync"
	"testing"
	"time"
)

// TestMigrateEmpty verifies that Migrate fails on an empty database.
func TestMigrateEmpty(t *testing.T) {
	database := newTestDatabase(t)
	err := Migrate(t.Context(), database)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// TestMigrateConcurrent verifies that concurrent calls to Migrate on the same
// initialized database succeed.
func TestMigrateConcurrent(t *testing.T) {
	database := newInitializedTestDatabase(t)
	const instances = 4
	if n := database.PoolStats().MaxConns(); n < instances {
		t.Fatalf("expected at least %d pool connections, got %d", instances, n)
	}
	errs := make([]error, instances)
	var wg sync.WaitGroup
	for i := range instances {
		wg.Go(func() {
			errs[i] = Migrate(t.Context(), database)
		})
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("expected no error from instance %d, got %v", i, err)
		}
	}
}

// TestMigrateWaitsForInitialization verifies that Migrate waits for an
// initialization in progress, holding the initialization lock, and then
// migrates the initialized database instead of failing because it was empty.
func TestMigrateWaitsForInitialization(t *testing.T) {

	ctx := t.Context()
	database := newTestDatabase(t)
	if n := database.PoolStats().MaxConns(); n < 3 {
		t.Fatalf("expected at least 3 pool connections, got %d", n)
	}

	// Initialize the database under the lock, as InitIfEmpty does, without
	// committing.
	initializer, err := database.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer initializer.Rollback(ctx)
	_, err = initializer.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", initLockKey)
	if err != nil {
		t.Fatal(err)
	}
	err = initialize(ctx, initializer, false)
	if err != nil {
		t.Fatal(err)
	}

	migrated := make(chan error, 1)
	go func() {
		migrated <- Migrate(ctx, database)
	}()

	// Wait until Migrate waits for the initialization lock.
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case err := <-migrated:
			t.Fatalf("expected Migrate to wait for the initialization lock, got return with error %v", err)
		default:
		}
		var waiting bool
		err = database.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM pg_locks
				WHERE locktype = 'advisory' AND objid::bigint = $1 AND objsubid = 1 AND NOT granted
			)`, initLockKey).Scan(&waiting)
		if err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("expected Migrate to wait for the initialization lock, got no lock wait")
		}
		time.Sleep(10 * time.Millisecond)
	}

	err = initializer.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = <-migrated
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

}
