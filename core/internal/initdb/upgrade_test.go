// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestCheckSchemaVersionEmpty verifies that CheckSchemaVersion returns
// ErrNotInitialized on an empty database.
func TestCheckSchemaVersionEmpty(t *testing.T) {
	database := newTestDatabase(t)
	err := CheckSchemaVersion(t.Context(), database)
	if err != nil {
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("expected ErrNotInitialized, got %v", err)
		}
		return
	}
	t.Fatal("expected an error, got nil")
}

// TestCheckSchemaVersion verifies on which schema versions and states of the
// database CheckSchemaVersion lets this version of Krenalis run. Cases that set
// a schema version below 1 are skipped until schemaVersion is high enough.
func TestCheckSchemaVersion(t *testing.T) {

	type state struct {
		version int
		state   schemaState
	}

	const v = schemaVersion
	tests := []struct {
		name   string
		legacy bool   // drop the schema_version table and its type
		set    *state // state set on the newly initialized database, if not nil
		fails  bool
	}{
		{name: "new database"},
		{name: "legacy database", legacy: true, fails: legacySchemaVersion != v},
		{name: "current, migrating to next", set: &state{v, stateMigratingToNext}},
		{name: "current, migrated", set: &state{v, stateMigrated}},
		{name: "current, finalizing", set: &state{v, stateFinalizing}},
		{name: "next, migrated", set: &state{v + 1, stateMigrated}},
		{name: "next, finalizing", set: &state{v + 1, stateFinalizing}, fails: true},
		{name: "next, finalized", set: &state{v + 1, stateFinalized}, fails: true},
		{name: "next, migrating to next", set: &state{v + 1, stateMigratingToNext}, fails: true},
		{name: "two ahead", set: &state{v + 2, stateMigrated}, fails: true},
		{name: "previous, migrating to next", set: &state{v - 1, stateMigratingToNext}, fails: true},
		{name: "previous, finalized", set: &state{v - 1, stateFinalized}, fails: true},
		{name: "previous, migrated", set: &state{v - 1, stateMigrated}, fails: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.set != nil && test.set.version < 1 {
				t.Skipf("requires schemaVersion to be at least %d", v-test.set.version+1)
			}
			t.Parallel()

			ctx := t.Context()
			database := newInitializedTestDatabase(t)
			if test.legacy {
				_, err := database.Exec(ctx, "DROP TABLE schema_version; DROP TYPE schema_state")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, state = $2",
					test.set.version, test.set.state)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := CheckSchemaVersion(ctx, database)
			if err != nil {
				if !test.fails {
					t.Fatalf("expected no error, got %v", err)
				}
			}
			if err == nil && test.fails {
				t.Fatal("expected an error, got nil")
			}

		})
	}

}

// TestFinalizeResume verifies that Finalize records the database as being
// finalized before running finalize, resumes an interrupted finalization, runs
// finalize on a connection with the lock timeout while holding the upgrade
// lock, releases the lock when finalize fails, and records the database as
// finalized only once finalize succeeds. It replaces finalize, so it must not
// run in parallel with other tests.
func TestFinalizeResume(t *testing.T) {

	ctx := t.Context()
	database := newInitializedTestDatabase(t)
	_, err := database.Exec(ctx, "UPDATE schema_version SET state = 'Migrated'")
	if err != nil {
		t.Fatal(err)
	}

	saved := finalize
	t.Cleanup(func() { finalize = saved })
	var calls int
	var finalizeErr error
	finalize = func(ctx context.Context, conn *db.Conn) error {
		calls++
		assertSchemaState(t, conn, schemaVersion, stateFinalizing)
		assertLockTimeout(t, conn)
		waitForUpgradeLock(t, database, true)
		return finalizeErr
	}

	// Interrupt the finalization.
	finalizeErr = errors.New("finalization interrupted")
	err = Finalize(ctx, database)
	if err != nil {
		if !errors.Is(err, finalizeErr) {
			t.Fatalf("expected the error of finalize, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	waitForUpgradeLock(t, database, false)
	assertSchemaState(t, database, schemaVersion, stateFinalizing)
	err = CheckSchemaVersion(ctx, database)
	if err != nil {
		t.Fatalf("expected no error from CheckSchemaVersion, got %v", err)
	}

	// Resume the finalization.
	finalizeErr = nil
	err = Finalize(ctx, database)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertSchemaState(t, database, schemaVersion, stateFinalized)

	// Finalize again, which does nothing.
	err = Finalize(ctx, database)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected finalize to be called 2 times, got %d", calls)
	}

}

// TestFinalizeSchemaVersions verifies how Finalize handles each schema version
// and state of the database. When Finalize must run finalize, which must run on
// a migrated database, the case runs only if finalize is nil, and Finalize must
// fail without changing the state: upgrade_release_test.go tests finalize while
// a release is prepared. Cases that set a schema version below 1 are skipped
// until schemaVersion is high enough.
func TestFinalizeSchemaVersions(t *testing.T) {

	type state struct {
		version int
		state   schemaState
	}

	const v = schemaVersion
	tests := []struct {
		name          string
		empty         bool   // use an empty database
		legacy        bool   // drop the schema_version table and its type
		set           *state // state set on the newly initialized database, if not nil
		needsFinalize bool   // Finalize must run finalize
		fails         bool
		expected      *state // state after Finalize, if not nil
	}{
		{name: "empty database", empty: true, fails: true},
		{name: "new database", expected: &state{v, stateFinalized}},
		{name: "legacy database", legacy: true, fails: legacySchemaVersion != v, expected: &state{legacySchemaVersion, stateFinalized}},
		{name: "current, migrating to next", set: &state{v, stateMigratingToNext}, expected: &state{v, stateMigratingToNext}},
		{name: "current, migrated", set: &state{v, stateMigrated}, needsFinalize: true, fails: true, expected: &state{v, stateMigrated}},
		{name: "current, finalizing", set: &state{v, stateFinalizing}, needsFinalize: true, fails: true, expected: &state{v, stateFinalizing}},
		{name: "next, migrated", set: &state{v + 1, stateMigrated}, expected: &state{v + 1, stateMigrated}},
		{name: "next, finalizing", set: &state{v + 1, stateFinalizing}, fails: true, expected: &state{v + 1, stateFinalizing}},
		{name: "next, finalized", set: &state{v + 1, stateFinalized}, fails: true, expected: &state{v + 1, stateFinalized}},
		{name: "next, migrating to next", set: &state{v + 1, stateMigratingToNext}, fails: true, expected: &state{v + 1, stateMigratingToNext}},
		{name: "previous, migrating to next", set: &state{v - 1, stateMigratingToNext}, fails: true, expected: &state{v - 1, stateMigratingToNext}},
		{name: "previous, finalized", set: &state{v - 1, stateFinalized}, fails: true, expected: &state{v - 1, stateFinalized}},
		{name: "previous, migrated", set: &state{v - 1, stateMigrated}, fails: true, expected: &state{v - 1, stateMigrated}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.set != nil && test.set.version < 1 {
				t.Skipf("requires schemaVersion to be at least %d", v-test.set.version+1)
			}
			if test.needsFinalize && finalize != nil {
				t.Skip("Finalize would run finalize, which upgrade_release_test.go tests")
			}
			t.Parallel()

			ctx := t.Context()
			var database *db.DB
			if test.empty {
				database = newTestDatabase(t)
			} else {
				database = newInitializedTestDatabase(t)
			}
			if test.legacy {
				_, err := database.Exec(ctx, "DROP TABLE schema_version; DROP TYPE schema_state")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, state = $2",
					test.set.version, test.set.state)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := Finalize(ctx, database)
			if err != nil {
				if !test.fails {
					t.Fatalf("expected no error, got %v", err)
				}
			}
			if err == nil && test.fails {
				t.Fatal("expected an error, got nil")
			}

			if test.expected != nil {
				assertSchemaState(t, database, test.expected.version, test.expected.state)
			}

		})
	}

}

// TestMigrateEmpty verifies that Migrate returns ErrNotInitialized on an empty
// database.
func TestMigrateEmpty(t *testing.T) {
	database := newTestDatabase(t)
	err := Migrate(t.Context(), database)
	if err != nil {
		if !errors.Is(err, ErrNotInitialized) {
			t.Fatalf("expected ErrNotInitialized, got %v", err)
		}
		return
	}
	t.Fatal("expected an error, got nil")
}

// TestMigrateResume verifies that Migrate records the database, still at the
// previous schema version, as being migrated to the next one before running
// migrate, resumes an interrupted migration, runs migrate on a connection with
// the lock timeout while holding the upgrade lock, releases the lock when
// migrate fails, and records the database as migrated only once migrate
// succeeds. It replaces migrate, so it must not run in parallel with other
// tests.
func TestMigrateResume(t *testing.T) {

	// Set the database to the previous schema version, finalized, allowing
	// schema version 0 while schemaVersion is 1.
	ctx := t.Context()
	database := newInitializedTestDatabase(t)
	_, err := database.Exec(ctx, "ALTER TABLE schema_version DROP CONSTRAINT schema_version_version_check")
	if err != nil {
		t.Fatal(err)
	}
	_, err = database.Exec(ctx, "UPDATE schema_version SET version = $1", schemaVersion-1)
	if err != nil {
		t.Fatal(err)
	}

	saved := migrate
	t.Cleanup(func() { migrate = saved })
	var calls int
	var migrateErr error
	migrate = func(ctx context.Context, conn *db.Conn) error {
		calls++
		assertSchemaState(t, conn, schemaVersion-1, stateMigratingToNext)
		assertLockTimeout(t, conn)
		waitForUpgradeLock(t, database, true)
		return migrateErr
	}

	// Interrupt the migration.
	migrateErr = errors.New("migration interrupted")
	err = Migrate(ctx, database)
	if err != nil {
		if !errors.Is(err, migrateErr) {
			t.Fatalf("expected the error of migrate, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	waitForUpgradeLock(t, database, false)
	assertSchemaState(t, database, schemaVersion-1, stateMigratingToNext)
	err = CheckSchemaVersion(ctx, database)
	if err == nil {
		t.Fatal("expected an error from CheckSchemaVersion, got nil")
	}

	// Resume the migration.
	migrateErr = nil
	err = Migrate(ctx, database)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertSchemaState(t, database, schemaVersion, stateMigrated)

	// Migrate again, which does nothing.
	err = Migrate(ctx, database)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected migrate to be called 2 times, got %d", calls)
	}

}

// TestMigrateSchemaVersions verifies how Migrate handles each schema version
// and state of the database. When Migrate must run migrate, which must run on a
// database of the previous schema version, the case runs only if migrate is
// nil, and Migrate must fail without changing the state:
// upgrade_release_test.go tests migrate while a release is prepared. Cases that
// set a schema version below 1 are skipped until schemaVersion is high enough.
func TestMigrateSchemaVersions(t *testing.T) {

	type state struct {
		version int
		state   schemaState
	}

	const v = schemaVersion
	tests := []struct {
		name         string
		legacy       bool   // drop the schema_version table and its type
		set          *state // state set on the newly initialized database, if not nil
		needsMigrate bool   // Migrate must run migrate
		fails        bool
		expected     state // state after Migrate
	}{
		{name: "new database", expected: state{v, stateFinalized}},
		{name: "legacy database", legacy: true, needsMigrate: legacySchemaVersion == v-1, fails: legacySchemaVersion != v, expected: state{legacySchemaVersion, stateFinalized}},
		{name: "current, migrating to next", set: &state{v, stateMigratingToNext}, expected: state{v, stateMigratingToNext}},
		{name: "current, migrated", set: &state{v, stateMigrated}, expected: state{v, stateMigrated}},
		{name: "current, finalizing", set: &state{v, stateFinalizing}, expected: state{v, stateFinalizing}},
		{name: "next, migrated", set: &state{v + 1, stateMigrated}, expected: state{v + 1, stateMigrated}},
		{name: "next, finalizing", set: &state{v + 1, stateFinalizing}, fails: true, expected: state{v + 1, stateFinalizing}},
		{name: "next, finalized", set: &state{v + 1, stateFinalized}, fails: true, expected: state{v + 1, stateFinalized}},
		{name: "next, migrating to next", set: &state{v + 1, stateMigratingToNext}, fails: true, expected: state{v + 1, stateMigratingToNext}},
		{name: "two ahead", set: &state{v + 2, stateMigrated}, fails: true, expected: state{v + 2, stateMigrated}},
		{name: "previous, finalized", set: &state{v - 1, stateFinalized}, needsMigrate: true, fails: true, expected: state{v - 1, stateFinalized}},
		{name: "previous, migrating to next", set: &state{v - 1, stateMigratingToNext}, needsMigrate: true, fails: true, expected: state{v - 1, stateMigratingToNext}},
		{name: "previous, migrated", set: &state{v - 1, stateMigrated}, fails: true, expected: state{v - 1, stateMigrated}},
		{name: "previous, finalizing", set: &state{v - 1, stateFinalizing}, fails: true, expected: state{v - 1, stateFinalizing}},
		{name: "two behind", set: &state{v - 2, stateFinalized}, fails: true, expected: state{v - 2, stateFinalized}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.set != nil && test.set.version < 1 {
				t.Skipf("requires schemaVersion to be at least %d", v-test.set.version+1)
			}
			if test.needsMigrate && migrate != nil {
				t.Skip("Migrate would run migrate, which upgrade_release_test.go tests")
			}
			t.Parallel()

			ctx := t.Context()
			database := newInitializedTestDatabase(t)
			if test.legacy {
				_, err := database.Exec(ctx, "DROP TABLE schema_version; DROP TYPE schema_state")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, state = $2",
					test.set.version, test.set.state)
				if err != nil {
					t.Fatal(err)
				}
			}

			err := Migrate(ctx, database)
			if err != nil {
				if !test.fails {
					t.Fatalf("expected no error, got %v", err)
				}
			}
			if err == nil && test.fails {
				t.Fatal("expected an error, got nil")
			}

			assertSchemaState(t, database, test.expected.version, test.expected.state)

		})
	}

}

// TestMigrateWaitsForInitialization verifies that Migrate waits for an
// initialization in progress, holding the initialization lock, and then finds
// the database initialized, instead of failing because it was empty.
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

// TestWriteSchemaVersion verifies that writeSchemaVersion records the schema
// version and state, creating the schema_version table as schema.sql does if
// the database is at the legacy schema version.
func TestWriteSchemaVersion(t *testing.T) {

	tests := []struct {
		name   string
		legacy bool // drop the schema_version table and its type
	}{
		{name: "new database"},
		{name: "legacy database", legacy: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			t.Parallel()
			ctx := t.Context()
			database := newInitializedTestDatabase(t)
			expected := schemaSnapshot(t, database)
			if test.legacy {
				_, err := database.Exec(ctx, "DROP TABLE schema_version; DROP TYPE schema_state")
				if err != nil {
					t.Fatal(err)
				}
			}

			// Write a state other than the one recorded by initialize.
			conn, err := database.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			err = writeSchemaVersion(ctx, conn, schemaVersion+1, stateMigratingToNext)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			got := schemaSnapshot(t, database)
			if got != expected {
				t.Fatalf("expected schema %s, got %s", expected, got)
			}
			assertSchemaState(t, database, schemaVersion+1, stateMigratingToNext)

		})
	}

}

// assertLockTimeout verifies that conn has the lock timeout set by
// lockForUpgrade.
func assertLockTimeout(t *testing.T, conn *db.Conn) {
	t.Helper()
	var timeout string
	err := conn.QueryRow(t.Context(), "SHOW lock_timeout").Scan(&timeout)
	if err != nil {
		t.Fatal(err)
	}
	if timeout != "5s" {
		t.Fatalf("expected lock timeout 5s, got %s", timeout)
	}
}

// assertSchemaState verifies that the database is at the given schema version
// and state.
func assertSchemaState(t *testing.T, conn db.Connection, version int, state schemaState) {
	t.Helper()
	gotVersion, gotState, err := readSchemaVersion(t.Context(), conn)
	if err != nil {
		t.Fatal(err)
	}
	if gotVersion != version || gotState != state {
		t.Fatalf("expected schema version %d, %s, got %d, %s", version, state, gotVersion, gotState)
	}
}

// waitForUpgradeLock waits until the lock taken by lockForUpgrade is held by a
// session, if held is true, or by none, and fails the test after 5 seconds. It
// reads pg_locks, since a connection of the pool can be the one holding the
// lock, on which acquiring it again succeeds. Waiting is needed for the
// release, because the server releases the lock when it ends the session of
// the closed connection, which may happen just after the upgrade step returns.
func waitForUpgradeLock(t *testing.T, database *db.DB, held bool) {
	t.Helper()
	ctx := t.Context()
	deadline := time.Now().Add(5 * time.Second)
	for {
		var got bool
		err := database.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT FROM pg_locks
				WHERE locktype = 'advisory' AND objid::bigint = $1 AND objsubid = 1 AND granted
			)`, initLockKey).Scan(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got == held {
			return
		}
		if time.Now().After(deadline) {
			if held {
				t.Fatal("expected the upgrade lock to be held, got it free")
			}
			t.Fatal("expected the upgrade lock to be free, got it held")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
