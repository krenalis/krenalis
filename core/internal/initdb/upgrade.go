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

// CheckSchemaVersion checks that this version of Krenalis can run on the
// database: its schema version must be the one of this version of Krenalis,
// finalized or not, or the next one not finalized, as while the Krenalis
// instances are being replaced or after a rollback to this version of
// Krenalis. It returns ErrNotInitialized if the database is empty.
func CheckSchemaVersion(ctx context.Context, database *db.DB) error {
	dbSchemaVersion, finalized, err := readSchemaVersion(ctx, database)
	if err != nil {
		return err
	}
	switch {
	case dbSchemaVersion == schemaVersion:
		// Same schema version as this Krenalis, finalized or not.
		return nil
	case dbSchemaVersion == schemaVersion+1 && !finalized:
		slog.Info("PostgreSQL database is at the next schema version, not finalized,"+
			" on which this version of Krenalis can still run", "schema_version", dbSchemaVersion)
		return nil
	case dbSchemaVersion > schemaVersion:
		return fmt.Errorf("PostgreSQL database is at schema version %d, on which this version of Krenalis,"+
			" at schema version %d, can no longer run", dbSchemaVersion, schemaVersion)
	}
	return fmt.Errorf("PostgreSQL database is at schema version %d, but this version of Krenalis"+
		" requires schema version %d; migrate the database first", dbSchemaVersion, schemaVersion)
}

// Finalize finalizes, in a single transaction, Krenalis's PostgreSQL database
// at the schema version of this version of Krenalis, removing what only the
// Krenalis instances of the previous release need, and records it as
// finalized. It does nothing if the database is already finalized, or if it is
// at the next schema version but not finalized, as after a rollback to this
// version of Krenalis. It fails if the database is empty or at any other
// schema version. After it, the instances of the previous release can no
// longer run on the database. It does not check that none of them is
// running: the caller must ensure it, since otherwise the behavior is
// undefined and may be destructive. It is safe to call it concurrently from
// several Krenalis instances. It fails, instead of finalizing the database, if
// finalize is nil, as in the releases after the one that introduced this schema
// version.
func Finalize(ctx context.Context, database *db.DB) error {
	var done bool
	err := database.Transaction(ctx, func(tx *db.Tx) error {

		err := lockForUpgrade(ctx, tx)
		if err != nil {
			return err
		}
		dbSchemaVersion, finalized, err := readSchemaVersion(ctx, tx)
		if err != nil {
			return err
		}

		switch {
		case dbSchemaVersion == schemaVersion && finalized:
			slog.Info("PostgreSQL database is already finalized", "schema_version", dbSchemaVersion)
			return nil
		case dbSchemaVersion == schemaVersion:
			// It is finalized below.
		case dbSchemaVersion < schemaVersion:
			return fmt.Errorf("PostgreSQL database is at schema version %d, which must be migrated to"+
				" schema version %d before finalizing it", dbSchemaVersion, schemaVersion)
		case dbSchemaVersion == schemaVersion+1 && !finalized:
			slog.Info("PostgreSQL database is at the next schema version, not finalized, so it won't be finalized",
				"schema_version", dbSchemaVersion)
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
		err = finalize(ctx, tx)
		if err != nil {
			return err
		}
		err = writeSchemaVersion(ctx, tx, schemaVersion, true)
		if err != nil {
			return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
		}
		done = true

		return nil
	})
	if err != nil {
		return err
	}
	if done {
		slog.Info("PostgreSQL database finalized", "schema_version", schemaVersion)
	}
	return nil
}

// Migrate migrates Krenalis's PostgreSQL database, in a single transaction,
// from the previous schema version to the one of this version of Krenalis, and
// records it as not finalized. It does nothing if the database is already at
// this schema version, or at the next one but not finalized, as after a
// rollback to this version of Krenalis. It fails if the database is empty, if
// the previous schema version has not been finalized, or if the database is at
// any other schema version. It is safe to call it concurrently from several
// Krenalis instances. It fails, instead of migrating the database, if migrate
// is nil, as in the releases after the one that introduced this schema version.
func Migrate(ctx context.Context, database *db.DB) error {
	var done bool
	err := database.Transaction(ctx, func(tx *db.Tx) error {

		err := lockForUpgrade(ctx, tx)
		if err != nil {
			return err
		}
		dbSchemaVersion, finalized, err := readSchemaVersion(ctx, tx)
		if err != nil {
			return err
		}

		switch {
		case dbSchemaVersion == schemaVersion:
			slog.Info("PostgreSQL database is already at the schema version of this version of Krenalis",
				"schema_version", dbSchemaVersion)
			return nil
		case dbSchemaVersion == schemaVersion+1 && !finalized:
			slog.Info("PostgreSQL database is at the next schema version, not finalized, so it won't be migrated",
				"schema_version", dbSchemaVersion)
			return nil
		case dbSchemaVersion > schemaVersion:
			return fmt.Errorf("PostgreSQL database is at schema version %d, on which this version of Krenalis,"+
				" at schema version %d, can no longer run", dbSchemaVersion, schemaVersion)
		case dbSchemaVersion == schemaVersion-1 && finalized:
			// It is migrated below.
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
		err = migrate(ctx, tx)
		if err != nil {
			return err
		}
		err = writeSchemaVersion(ctx, tx, schemaVersion, false)
		if err != nil {
			return fmt.Errorf("cannot write schema version of PostgreSQL database: %s", err)
		}
		done = true

		return nil
	})
	if err != nil {
		return err
	}
	if done {
		slog.Info("PostgreSQL database migrated", "schema_version", schemaVersion)
	}
	return nil
}

// lockForUpgrade takes, within the given transaction, the lock that serializes
// the initialization, the migration, and the finalization of the database, and
// limits the wait for the locks on database objects taken afterwards.
func lockForUpgrade(ctx context.Context, tx *db.Tx) error {

	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", initLockKey)
	if err != nil {
		return fmt.Errorf("cannot lock PostgreSQL database: %s", err)
	}

	// Fail, instead of blocking the running Krenalis instances, when a lock on
	// a database object cannot be acquired within a short time. It is set
	// after the advisory lock, so that concurrent migrations and finalizations
	// still wait for each other.
	_, err = tx.Exec(ctx, "SET LOCAL lock_timeout = '5s'")
	if err != nil {
		return fmt.Errorf("cannot set lock timeout: %s", err)
	}

	return nil
}

// readSchemaVersion returns the schema version of the database and whether it
// has been finalized. A database without the schema_version table is at the
// legacy schema version, finalized. It returns ErrNotInitialized if the
// database is empty.
func readSchemaVersion(ctx context.Context, conn db.Connection) (version int, finalized bool, err error) {
	isEmpty, err := databaseIsEmpty(ctx, conn)
	if err != nil {
		return 0, false, fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if isEmpty {
		return 0, false, ErrNotInitialized
	}
	var exists bool
	err = conn.QueryRow(ctx, "SELECT to_regclass('schema_version') IS NOT NULL").Scan(&exists)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read schema version of PostgreSQL database: %s", err)
	}
	if !exists {
		return legacySchemaVersion, true, nil
	}
	err = conn.QueryRow(ctx, "SELECT version, finalized FROM schema_version").Scan(&version, &finalized)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read schema version of PostgreSQL database: %s", err)
	}
	return version, finalized, nil
}

// writeSchemaVersion records, within the given transaction, the schema version
// of the database and whether it has been finalized. It creates the
// schema_version table if the database is at the legacy schema version.
func writeSchemaVersion(ctx context.Context, tx *db.Tx, version int, finalized bool) error {
	// The definition must match the one in schema.sql.
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_version (
		singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
		version integer NOT NULL CHECK (version > 0),
		finalized boolean NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO schema_version (version, finalized) VALUES ($1, $2)"+
		" ON CONFLICT (singleton) DO UPDATE SET version = EXCLUDED.version, finalized = EXCLUDED.finalized",
		version, finalized)
	return err
}
