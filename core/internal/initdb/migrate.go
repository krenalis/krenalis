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

// Migrate migrates Krenalis's PostgreSQL database to the current version, in a
// single transaction. Between releases there is nothing to migrate, so it only
// checks that the database is initialized. It is idempotent, and it is safe to
// call it concurrently from several Krenalis instances.
func Migrate(ctx context.Context, database *db.DB) error {
	err := database.Transaction(ctx, func(tx *db.Tx) error {
		// Serialize the migration with the initialization and the migration
		// of other Krenalis instances.
		_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", initLockKey)
		if err != nil {
			return fmt.Errorf("cannot lock PostgreSQL database for migration: %s", err)
		}
		isEmpty, err := databaseIsEmpty(ctx, tx)
		if err != nil {
			return fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
		}
		if isEmpty {
			return errors.New("Krenalis's PostgreSQL database has not been initialized")
		}
		return migrate(ctx, tx)
	})
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database migration completed")
	return nil
}

// migrate migrates, within the given transaction, a database of the base
// release to the current schema. It must be idempotent, as it also runs on
// databases already migrated and on newly initialized ones.
//
// It is written when a release is prepared, from the notes in
// RELEASE-UPGRADE.md, and emptied after the release is published. See
// .agents/skills/release-upgrade/SKILL.md.
func migrate(ctx context.Context, tx *db.Tx) error {
	return nil
}
