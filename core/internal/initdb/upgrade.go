// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/db"
)

// Upgrade upgrades an existing Krenalis PostgreSQL database from the schema of
// release v0.44.0 to the current schema. It is idempotent.
func Upgrade(ctx context.Context, database *db.DB) error {

	initialized, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = 'organizations'
			AND c.relkind = 'r'`)
	if err != nil {
		return err
	}
	if !initialized {
		return fmt.Errorf("Krenalis's PostgreSQL database has not been initialized")
	}

	err = database.Transaction(ctx, func(tx *db.Tx) error {
		queries := []string{
			`CREATE TABLE IF NOT EXISTS identity_metrics (
				workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
				day date NOT NULL,
				observed_at time without time zone NOT NULL,
				identities_anonymous bigint NOT NULL,
				identities_recognized bigint NOT NULL,
				identities_without_profile bigint NOT NULL,
				PRIMARY KEY (workspace, day)
			)`,
			`CREATE TABLE IF NOT EXISTS identity_connection_metrics (
				connection varchar(12) NOT NULL REFERENCES connections ON DELETE CASCADE,
				day date NOT NULL,
				identities_anonymous bigint NOT NULL,
				identities_recognized bigint NOT NULL,
				identities_without_profile bigint NOT NULL,
				PRIMARY KEY (connection, day)
			)`,
			`ALTER TABLE election ALTER COLUMN number TYPE bigint`,
		}
		for _, query := range queries {
			_, err := tx.Exec(ctx, query)
			if err != nil {
				return fmt.Errorf("cannot execute upgrade query %q: %s", query, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	slog.Info("PostgreSQL database upgraded successfully")

	return nil
}
