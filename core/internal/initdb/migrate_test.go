// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"sync"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
)

// TestCheckSchemaVersion verifies on which schema versions of the database
// CheckSchemaVersion lets this version of Krenalis run. Cases that set a
// schema version below 1 are skipped until schemaVersion is high enough.
func TestCheckSchemaVersion(t *testing.T) {

	type state struct {
		version   int
		finalized bool
	}

	const v = schemaVersion
	tests := []struct {
		name   string
		empty  bool   // use an empty database
		legacy bool   // drop the schema_version table
		set    *state // state set on the newly initialized database, if not nil
		fails  bool
	}{
		{name: "empty database", empty: true},
		{name: "new database"},
		{name: "legacy database", legacy: true, fails: legacySchemaVersion != v},
		{name: "current, not finalized", set: &state{v, false}},
		{name: "next, not finalized", set: &state{v + 1, false}},
		{name: "next, finalized", set: &state{v + 1, true}, fails: true},
		{name: "two ahead", set: &state{v + 2, false}, fails: true},
		{name: "previous, finalized", set: &state{v - 1, true}, fails: true},
		{name: "previous, not finalized", set: &state{v - 1, false}, fails: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.set != nil && test.set.version < 1 {
				t.Skipf("requires schemaVersion to be at least %d", v-test.set.version+1)
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
				_, err := database.Exec(ctx, "DROP TABLE schema_version")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, finalized = $2",
					test.set.version, test.set.finalized)
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

// TestFinalizeSchemaVersions verifies how Finalize handles each schema version
// of the database. Cases that set a schema version below 1 are skipped until
// schemaVersion is high enough.
func TestFinalizeSchemaVersions(t *testing.T) {

	type state struct {
		version   int
		finalized bool
	}

	const v = schemaVersion
	tests := []struct {
		name     string
		empty    bool   // use an empty database
		legacy   bool   // drop the schema_version table
		set      *state // state set on the newly initialized database, if not nil
		fails    bool
		expected *state // state after Finalize, if not nil
	}{
		{name: "empty database", empty: true, fails: true},
		{name: "new database", expected: &state{v, true}},
		{name: "legacy database", legacy: true, fails: legacySchemaVersion != v, expected: &state{legacySchemaVersion, true}},
		{name: "current, not finalized", set: &state{v, false}, expected: &state{v, true}},
		{name: "next, not finalized", set: &state{v + 1, false}, fails: true, expected: &state{v + 1, false}},
		{name: "next, finalized", set: &state{v + 1, true}, fails: true, expected: &state{v + 1, true}},
		{name: "previous, finalized", set: &state{v - 1, true}, fails: true, expected: &state{v - 1, true}},
		{name: "previous, not finalized", set: &state{v - 1, false}, fails: true, expected: &state{v - 1, false}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			if test.set != nil && test.set.version < 1 {
				t.Skipf("requires schemaVersion to be at least %d", v-test.set.version+1)
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
				_, err := database.Exec(ctx, "DROP TABLE schema_version")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, finalized = $2",
					test.set.version, test.set.finalized)
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
				var got state
				got.version, got.finalized, err = readSchemaVersion(ctx, database)
				if err != nil {
					t.Fatal(err)
				}
				if got != *test.expected {
					t.Fatalf("expected version %+v, got %+v", *test.expected, got)
				}
			}

		})
	}

}

// TestMigrateEmpty verifies that Migrate fails on an empty database.
func TestMigrateEmpty(t *testing.T) {
	database := newTestDatabase(t)
	err := Migrate(t.Context(), database)
	if err != nil {
		const expected = "Krenalis's PostgreSQL database has not been initialized"
		if err.Error() != expected {
			t.Fatalf("expected error %q, got %q", expected, err)
		}
		return
	}
	t.Fatal("expected an error, got nil")
}

// TestMigrateSchemaVersions verifies how Migrate handles each schema version of
// the database. Cases that set a schema version below 1 are skipped until
// schemaVersion is high enough.
func TestMigrateSchemaVersions(t *testing.T) {

	type state struct {
		version   int
		finalized bool
	}

	// A database without the schema_version table is handled as a finalized
	// database of the legacy schema version.
	legacyExpected, legacyFails := state{legacySchemaVersion, true}, false
	switch legacySchemaVersion {
	case schemaVersion:
		// There is nothing to migrate.
	case schemaVersion - 1:
		legacyExpected = state{schemaVersion, false}
	default:
		legacyFails = true
	}

	const v = schemaVersion
	tests := []struct {
		name     string
		legacy   bool   // drop the schema_version table
		set      *state // state set on the newly initialized database, if not nil
		fails    bool
		expected state // state after Migrate
	}{
		{name: "new database", expected: state{v, true}},
		{name: "legacy database", legacy: true, fails: legacyFails, expected: legacyExpected},
		{name: "current, not finalized", set: &state{v, false}, expected: state{v, false}},
		{name: "next, not finalized", set: &state{v + 1, false}, expected: state{v + 1, false}},
		{name: "next, finalized", set: &state{v + 1, true}, fails: true, expected: state{v + 1, true}},
		{name: "two ahead", set: &state{v + 2, false}, fails: true, expected: state{v + 2, false}},
		{name: "previous, finalized", set: &state{v - 1, true}, expected: state{v, false}},
		{name: "previous, not finalized", set: &state{v - 1, false}, fails: true, expected: state{v - 1, false}},
		{name: "two behind", set: &state{v - 2, true}, fails: true, expected: state{v - 2, true}},
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
				_, err := database.Exec(ctx, "DROP TABLE schema_version")
				if err != nil {
					t.Fatal(err)
				}
			}
			if test.set != nil {
				_, err := database.Exec(ctx, "UPDATE schema_version SET version = $1, finalized = $2",
					test.set.version, test.set.finalized)
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

			var got state
			got.version, got.finalized, err = readSchemaVersion(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			if got != test.expected {
				t.Fatalf("expected version %+v, got %+v", test.expected, got)
			}

		})
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
