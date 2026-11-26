// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/errors"
)

// legacySchemaVersion is the schema version of a database without the
// schema_version table, that is, a database of v0.48.0.
const legacySchemaVersion = 1

// ErrNotInitialized is returned when Krenalis's PostgreSQL database is empty,
// as it has not been initialized.
var ErrNotInitialized = errors.New("Krenalis's PostgreSQL database has not been initialized")

// schemaState is the state of the upgrade of the database, recorded in the
// schema_version table together with the schema version, which is always the
// highest one on which Krenalis can run. With stateMigrated, the releases at
// the previous schema version can also run on it.
type schemaState string

const (
	stateMigratingToNext schemaState = "MigratingToNext" // being migrated to the next schema version
	stateMigrated        schemaState = "Migrated"        // migrated, not finalized; the previous release can run
	stateFinalizing      schemaState = "Finalizing"      // being finalized; the previous release can no longer run
	stateFinalized       schemaState = "Finalized"       // finalized
)

// CheckSchemaVersion checks that this version of Krenalis can run on the
// database: its schema version must be the one of this version of Krenalis, or
// the next one, migrated and not yet being finalized, as while the Krenalis
// instances are being replaced or after a rollback to this version of
// Krenalis. It returns ErrNotInitialized if the database is empty.
func CheckSchemaVersion(ctx context.Context, database *db.DB) error {
	dbSchemaVersion, state, err := readSchemaVersion(ctx, database)
	if err != nil {
		return err
	}
	switch {
	case dbSchemaVersion == schemaVersion:
		// Same schema version as this Krenalis, in any state.
		return nil
	case dbSchemaVersion == schemaVersion+1 && state == stateMigrated:
		slog.Info("PostgreSQL database is at the next schema version, migrated and not yet being finalized,"+
			" on which this version of Krenalis can still run", "schema_version", dbSchemaVersion)
		return nil
	case dbSchemaVersion > schemaVersion:
		return fmt.Errorf("PostgreSQL database is at schema version %d, on which this version of Krenalis,"+
			" at schema version %d, can no longer run", dbSchemaVersion, schemaVersion)
	case dbSchemaVersion == schemaVersion-1 && state == stateMigratingToNext:
		return fmt.Errorf("PostgreSQL database is being migrated to schema version %d, the one of this version"+
			" of Krenalis; complete the migration first", schemaVersion)
	}
	return fmt.Errorf("PostgreSQL database is at schema version %d, but this version of Krenalis"+
		" requires schema version %d; migrate the database first", dbSchemaVersion, schemaVersion)
}

// Finalize finalizes Krenalis's PostgreSQL database at the schema version of
// this version of Krenalis, removing what only the Krenalis instances of the
// previous release need. It first records the database as being finalized,
// after which those instances no longer start on it, then runs finalize and
// records the database as finalized. If the database is already being
// finalized, as after an interrupted finalization, it resumes the finalization.
// It does nothing if the database is already finalized, even if it is being
// migrated to the next schema version, or if it is at the next schema version,
// migrated and not yet being finalized, as after a rollback to this version of
// Krenalis. It fails if the database is empty, if it has not been migrated to
// this schema version, or if it is at any other schema version. It does not
// check that no instance of the previous release is running: the caller must
// ensure it, since otherwise the behavior is undefined and may be destructive.
// It is safe to call it concurrently from several Krenalis instances. It fails,
// instead of finalizing the database, if finalize is nil, as in the releases
// after the one that introduced this schema version.
func Finalize(ctx context.Context, database *db.DB) error {

	conn, release, err := lockForUpgrade(ctx, database)
	if err != nil {
		return err
	}
	defer release()
	dbSchemaVersion, state, err := readSchemaVersion(ctx, conn)
	if err != nil {
		return err
	}

	switch {
	case dbSchemaVersion == schemaVersion && state == stateMigrated:
		// It is finalized below.
	case dbSchemaVersion == schemaVersion && state == stateFinalizing:
		slog.Info("PostgreSQL database is being finalized, so its finalization is resumed",
			"schema_version", dbSchemaVersion)
	case dbSchemaVersion == schemaVersion:
		// Finalized, possibly being migrated to the next schema version.
		slog.Info("PostgreSQL database is already finalized", "schema_version", dbSchemaVersion, "state", state)
		return nil
	case dbSchemaVersion < schemaVersion:
		return fmt.Errorf("PostgreSQL database is at schema version %d, which must be migrated to"+
			" schema version %d before finalizing it", dbSchemaVersion, schemaVersion)
	case dbSchemaVersion == schemaVersion+1 && state == stateMigrated:
		slog.Info("PostgreSQL database is at the next schema version, migrated and not yet being finalized,"+
			" so it won't be finalized", "schema_version", dbSchemaVersion)
		return nil
	default:
		return fmt.Errorf("PostgreSQL database is at schema version %d, newer than schema version %d of"+
			" this version of Krenalis", dbSchemaVersion, schemaVersion)
	}

	if finalize == nil {
		return fmt.Errorf("PostgreSQL database is at schema version %d, not finalized, and this version of"+
			" Krenalis cannot finalize it; finalize it with the release that introduced schema version %d",
			dbSchemaVersion, schemaVersion)
	}

	// Record that the finalization has started before changing anything, so
	// that the instances of the previous release no longer start, and that an
	// interrupted finalization can be resumed.
	if state == stateMigrated {
		err = writeSchemaVersion(ctx, conn, schemaVersion, stateFinalizing)
		if err != nil {
			return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
		}
	}

	err = finalize(ctx, conn)
	if err != nil {
		return err
	}
	err = writeSchemaVersion(ctx, conn, schemaVersion, stateFinalized)
	if err != nil {
		return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
	}
	slog.Info("PostgreSQL database finalized", "schema_version", schemaVersion)

	return nil
}

// Migrate migrates Krenalis's PostgreSQL database from the previous schema
// version to the one of this version of Krenalis. It first records the
// database, still at the previous schema version, as being migrated to the next
// one, then runs migrate and records the database as migrated to this schema
// version. If the database is already being migrated to this schema version, as
// after an interrupted migration, it resumes the migration. It does nothing if
// the database is already migrated to this schema version, or if it is at the
// next one, migrated and not yet being finalized, as after a rollback to this
// version of Krenalis. It fails if the database is empty, if the previous
// schema version has not been finalized, or if the database is at any other
// schema version. It is safe to call it concurrently from several Krenalis
// instances. It fails, instead of migrating the database, if migrate is nil, as
// in the releases after the one that introduced this schema version.
func Migrate(ctx context.Context, database *db.DB) error {

	conn, release, err := lockForUpgrade(ctx, database)
	if err != nil {
		return err
	}
	defer release()
	dbSchemaVersion, state, err := readSchemaVersion(ctx, conn)
	if err != nil {
		return err
	}

	switch {
	case dbSchemaVersion == schemaVersion-1 && state == stateFinalized:
		// It is migrated below.
	case dbSchemaVersion == schemaVersion-1 && state == stateMigratingToNext:
		slog.Info("PostgreSQL database is being migrated to the next schema version, so its migration is resumed",
			"schema_version", dbSchemaVersion)
	case dbSchemaVersion == schemaVersion:
		slog.Info("PostgreSQL database is already migrated to the schema version of this version of Krenalis",
			"schema_version", dbSchemaVersion, "state", state)
		return nil
	case dbSchemaVersion == schemaVersion+1 && state == stateMigrated:
		slog.Info("PostgreSQL database is at the next schema version, migrated and not yet being finalized,"+
			" so it won't be migrated", "schema_version", dbSchemaVersion)
		return nil
	case dbSchemaVersion > schemaVersion:
		return fmt.Errorf("PostgreSQL database is at schema version %d, on which this version of Krenalis,"+
			" at schema version %d, can no longer run", dbSchemaVersion, schemaVersion)
	case dbSchemaVersion == schemaVersion-1:
		return fmt.Errorf("PostgreSQL database is at schema version %d, which must be finalized before"+
			" migrating it to schema version %d", dbSchemaVersion, schemaVersion)
	default:
		return fmt.Errorf("PostgreSQL database is at schema version %d, which cannot be migrated directly"+
			" to schema version %d; migrate it one schema version at a time", dbSchemaVersion, schemaVersion)
	}

	if migrate == nil {
		return fmt.Errorf("PostgreSQL database is at schema version %d, and this version of Krenalis cannot"+
			" migrate it to schema version %d; migrate it with the release that introduced schema version %d",
			dbSchemaVersion, schemaVersion, schemaVersion)
	}

	// Record that the migration has started before changing anything, so that
	// an interrupted migration can be resumed. The database stays at the
	// previous schema version until the migration is completed.
	if state == stateFinalized {
		err = writeSchemaVersion(ctx, conn, dbSchemaVersion, stateMigratingToNext)
		if err != nil {
			return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
		}
	}

	err = migrate(ctx, conn)
	if err != nil {
		return err
	}
	err = writeSchemaVersion(ctx, conn, schemaVersion, stateMigrated)
	if err != nil {
		return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
	}
	slog.Info("PostgreSQL database migrated", "schema_version", schemaVersion)

	return nil
}

// lockForUpgrade acquires from database a connection reserved to the upgrade,
// takes on it the session lock that serializes the initialization, the
// migration, and the finalization of the database, and limits the wait for the
// locks on database objects taken afterwards on it. The returned function
// releases the lock and the connection.
func lockForUpgrade(ctx context.Context, database *db.DB) (*db.Conn, func(), error) {

	conn, err := database.Conn(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot connect to PostgreSQL: %s", err)
	}
	// Close the connection, instead of returning it to the pool, so that the
	// lock and the lock timeout end with it.
	release := func() {
		_ = conn.Underlying().Close(ctx)
		_ = conn.Close()
	}

	_, err = conn.Exec(ctx, "SELECT pg_advisory_lock($1)", initLockKey)
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("cannot lock PostgreSQL database: %s", err)
	}

	// Fail, instead of blocking the running Krenalis instances, when a lock on
	// a database object cannot be acquired within a short time. It is set
	// after the advisory lock, so that concurrent migrations and finalizations
	// still wait for each other.
	_, err = conn.Exec(ctx, "SET lock_timeout = '5s'")
	if err != nil {
		release()
		return nil, nil, fmt.Errorf("cannot set lock timeout: %s", err)
	}

	return conn, release, nil
}

// readSchemaVersion returns the schema version of the database and its state.
// A database without the schema_version table is at the legacy schema version,
// finalized. It returns ErrNotInitialized if the database is empty.
func readSchemaVersion(ctx context.Context, conn db.Connection) (version int, state schemaState, err error) {
	isEmpty, err := databaseIsEmpty(ctx, conn)
	if err != nil {
		return 0, "", fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if isEmpty {
		return 0, "", ErrNotInitialized
	}
	var exists bool
	err = conn.QueryRow(ctx, "SELECT to_regclass('schema_version') IS NOT NULL").Scan(&exists)
	if err != nil {
		return 0, "", fmt.Errorf("cannot read schema version of PostgreSQL database: %s", err)
	}
	if !exists {
		return legacySchemaVersion, stateFinalized, nil
	}
	err = conn.QueryRow(ctx, "SELECT version, state FROM schema_version").Scan(&version, &state)
	if err != nil {
		return 0, "", fmt.Errorf("cannot read schema version of PostgreSQL database: %s", err)
	}
	switch state {
	case stateMigratingToNext, stateMigrated, stateFinalizing, stateFinalized:
	default:
		return 0, "", fmt.Errorf("core/internal/initdb: invalid schema state %q", state)
	}
	return version, state, nil
}

// writeSchemaVersion records, in a transaction on conn, the schema version of
// the database and its state. It creates the schema_version table if the
// database is at the legacy schema version.
func writeSchemaVersion(ctx context.Context, conn *db.Conn, version int, state schemaState) error {
	return conn.Transaction(ctx, func(tx *db.Tx) error {
		var exists bool
		err := tx.QueryRow(ctx, "SELECT to_regclass('schema_version') IS NOT NULL").Scan(&exists)
		if err != nil {
			return err
		}
		if !exists {
			// The definitions must match the ones in schema.sql.
			_, err = tx.Exec(ctx, "CREATE TYPE schema_state AS ENUM"+
				" ('MigratingToNext', 'Migrated', 'Finalizing', 'Finalized')")
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `CREATE TABLE schema_version (
				singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
				version integer NOT NULL CHECK (version > 0),
				state schema_state NOT NULL
			)`)
			if err != nil {
				return err
			}
		}
		_, err = tx.Exec(ctx, "INSERT INTO schema_version (version, state) VALUES ($1, $2)"+
			" ON CONFLICT (singleton) DO UPDATE SET version = EXCLUDED.version, state = EXCLUDED.state",
			version, state)
		return err
	})
}
