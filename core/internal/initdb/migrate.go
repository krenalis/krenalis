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

// schemaGeneration is the generation of the database schema of this version of
// Krenalis. It is incremented only by the releases that change the database,
// whose migrate migrates a database of the previous generation to it. See
// .agents/skills/release-upgrade/SKILL.md.
const schemaGeneration = 1

// legacySchemaGeneration is the generation of a database without the
// schema_version table, that is, a database of v0.48.0.
const legacySchemaGeneration = 1

// CheckSchemaVersion checks that this version of Krenalis can run on the
// database: its schema generation must be the one of this version, finalized
// or not, or the next one not finalized, as while the Krenalis instances are
// being replaced or after a rollback. In the latter case it logs a warning. It
// returns nil for an empty database.
func CheckSchemaVersion(ctx context.Context, database *db.DB) error {
	isEmpty, err := databaseIsEmpty(ctx, database)
	if err != nil {
		return fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if isEmpty {
		return nil
	}
	generation, finalized, err := readSchemaVersion(ctx, database)
	if err != nil {
		return fmt.Errorf("cannot read schema generation of PostgreSQL database: %s", err)
	}
	switch {
	case generation == schemaGeneration:
		// This version runs on it.
	case generation == schemaGeneration+1 && !finalized:
		slog.Warn("PostgreSQL database is at the next schema generation, not finalized,"+
			" on which this version of Krenalis can still run", "generation", generation)
	case generation > schemaGeneration:
		return fmt.Errorf("PostgreSQL database is at schema generation %d, on which this version of Krenalis,"+
			" at generation %d, can no longer run", generation, schemaGeneration)
	default:
		return fmt.Errorf("PostgreSQL database is at schema generation %d, but this version of Krenalis"+
			" requires generation %d; migrate the database first", generation, schemaGeneration)
	}
	return nil
}

// Finalize finalizes, in a single transaction, the migration of Krenalis's
// PostgreSQL database to the schema generation of this version of Krenalis,
// removing what only the Krenalis instances of the previous version need, and
// records it as finalized. It does nothing if the database is already
// finalized, and fails if the database is empty or at another generation.
// After it, the instances of the previous version can no longer run on the
// database. It does not check that none of them is running: the caller must
// ensure it. It is safe to call it concurrently from several Krenalis
// instances.
func Finalize(ctx context.Context, database *db.DB) error {
	return database.Transaction(ctx, func(tx *db.Tx) error {

		generation, finalized, err := lockSchemaVersion(ctx, tx)
		if err != nil {
			return err
		}

		switch {
		case generation == schemaGeneration && finalized:
			slog.Info("PostgreSQL database is already finalized", "generation", generation)
			return nil
		case generation == schemaGeneration:
			// It is finalized below.
		case generation < schemaGeneration:
			return fmt.Errorf("PostgreSQL database is at schema generation %d, which must be migrated to"+
				" generation %d before finalizing it", generation, schemaGeneration)
		default:
			return fmt.Errorf("PostgreSQL database is at schema generation %d, newer than generation %d of this"+
				" version of Krenalis", generation, schemaGeneration)
		}

		err = finalize(ctx, tx)
		if err != nil {
			return err
		}
		err = writeSchemaVersion(ctx, tx, schemaGeneration, true)
		if err != nil {
			return fmt.Errorf("cannot write schema generation of PostgreSQL database: %s", err)
		}
		slog.Info("PostgreSQL database finalized", "generation", schemaGeneration)

		return nil
	})
}

// Migrate migrates Krenalis's PostgreSQL database, in a single transaction,
// from the previous schema generation to the one of this version of Krenalis,
// and records it as not finalized. It does nothing if the database is already
// at this generation, or at the next one but not finalized, as after a
// rollback. It fails if the database is empty, if the previous generation has
// not been finalized, or if the database is at any other generation. It is
// safe to call it concurrently from several Krenalis instances.
func Migrate(ctx context.Context, database *db.DB) error {
	err := database.Transaction(ctx, func(tx *db.Tx) error {

		generation, finalized, err := lockSchemaVersion(ctx, tx)
		if err != nil {
			return err
		}

		switch {
		case generation == schemaGeneration:
			slog.Info("PostgreSQL database is already at the schema generation of this version", "generation", generation)
			return nil
		case generation == schemaGeneration+1 && !finalized:
			slog.Warn("PostgreSQL database is at the next schema generation, not finalized, so it won't be migrated",
				"generation", generation)
			return nil
		case generation > schemaGeneration:
			return fmt.Errorf("PostgreSQL database is at schema generation %d, on which this version of Krenalis,"+
				" at generation %d, can no longer run", generation, schemaGeneration)
		case generation == schemaGeneration-1 && finalized:
			// It is migrated below.
		case generation == schemaGeneration-1:
			return fmt.Errorf("PostgreSQL database is at schema generation %d, which must be finalized before"+
				" migrating it to generation %d", generation, schemaGeneration)
		default:
			return fmt.Errorf("PostgreSQL database is at schema generation %d, which cannot be migrated directly"+
				" to generation %d; migrate it one generation at a time", generation, schemaGeneration)
		}

		err = migrate(ctx, tx)
		if err != nil {
			return err
		}
		err = writeSchemaVersion(ctx, tx, schemaGeneration, false)
		if err != nil {
			return fmt.Errorf("cannot write schema generation of PostgreSQL database: %s", err)
		}
		slog.Info("PostgreSQL database migrated", "generation", schemaGeneration)

		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database migration completed")
	return nil
}

// finalize removes, within the given transaction, what only the Krenalis
// instances of the previous schema generation need from a migrated database.
// It runs only when none of them is running anymore. The current version of
// Krenalis must run both before and after it.
//
// It is written together with migrate, when a release that changes the
// database is prepared, and emptied after the release is published. See
// .agents/skills/release-upgrade/SKILL.md.
func finalize(ctx context.Context, tx *db.Tx) error {
	return nil
}

// lockSchemaVersion takes, within the given transaction, the lock that
// serializes the initialization, the migration and the finalization of the
// database, and returns its schema generation and whether it has been
// finalized. It fails if the database is empty.
func lockSchemaVersion(ctx context.Context, tx *db.Tx) (generation int, finalized bool, err error) {

	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", initLockKey)
	if err != nil {
		return 0, false, fmt.Errorf("cannot lock PostgreSQL database for migration: %s", err)
	}

	// Fail, instead of blocking the running Krenalis instances, when a lock on
	// a database object cannot be acquired within a short time. It is set
	// after the advisory lock, so that concurrent migrations and finalizations
	// still wait for each other.
	_, err = tx.Exec(ctx, "SET LOCAL lock_timeout = '5s'")
	if err != nil {
		return 0, false, fmt.Errorf("cannot set lock timeout for migration: %s", err)
	}

	isEmpty, err := databaseIsEmpty(ctx, tx)
	if err != nil {
		return 0, false, fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if isEmpty {
		return 0, false, errors.New("Krenalis's PostgreSQL database has not been initialized")
	}
	generation, finalized, err = readSchemaVersion(ctx, tx)
	if err != nil {
		return 0, false, fmt.Errorf("cannot read schema generation of PostgreSQL database: %s", err)
	}

	return generation, finalized, nil
}

// migrate migrates, within the given transaction, a database of the previous
// schema generation to the current schema. It must not break the Krenalis
// instances of the previous version, which keep running on the database until
// they are replaced.
//
// It is written when a release that changes the database is prepared, from
// the notes in RELEASE-UPGRADE.md, and emptied after the release is published.
// See .agents/skills/release-upgrade/SKILL.md.
func migrate(ctx context.Context, tx *db.Tx) error {
	return nil
}

// readSchemaVersion returns the schema generation of the database and whether
// it has been finalized. A database without the schema_version table is at the
// legacy generation, finalized.
func readSchemaVersion(ctx context.Context, conn db.Connection) (generation int, finalized bool, err error) {
	var exists bool
	err = conn.QueryRow(ctx, "SELECT to_regclass('schema_version') IS NOT NULL").Scan(&exists)
	if err != nil {
		return 0, false, err
	}
	if !exists {
		return legacySchemaGeneration, true, nil
	}
	err = conn.QueryRow(ctx, "SELECT generation, finalized FROM schema_version").Scan(&generation, &finalized)
	return generation, finalized, err
}

// writeSchemaVersion records, within the given transaction, the schema
// generation of the database and whether it has been finalized. It creates the
// schema_version table if the database is at the legacy generation.
func writeSchemaVersion(ctx context.Context, tx *db.Tx, generation int, finalized bool) error {
	// The definition must match the one in schema.sql.
	_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_version (
		singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
		generation integer NOT NULL CHECK (generation > 0),
		finalized boolean NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, "INSERT INTO schema_version (generation, finalized) VALUES ($1, $2)"+
		" ON CONFLICT (singleton) DO UPDATE SET generation = EXCLUDED.generation, finalized = EXCLUDED.finalized",
		generation, finalized)
	return err
}
