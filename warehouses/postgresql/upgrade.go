// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// MigrateBase58IDs updates Krenalis resource identifiers in the warehouse.
func (warehouse *PostgreSQL) MigrateBase58IDs(ctx context.Context, migration, workspace string, connectionIDs, pipelineIDs, pipelineRunIDs map[int]string) error {
	return warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS "krenalis_internal_migrations" (
			"id" character varying NOT NULL,
			"workspace" text NOT NULL,
			"completed_at" timestamp NOT NULL DEFAULT now(),
			PRIMARY KEY ("id", "workspace")
		)`)
		if err != nil {
			return err
		}
		var done bool
		err = tx.QueryRow(ctx, `SELECT EXISTS (
			SELECT FROM "krenalis_internal_migrations" WHERE "id" = $1 AND "workspace" = $2
		)`, migration, workspace).Scan(&done)
		if err != nil {
			return err
		}
		if done {
			return nil
		}

		if _, err := tx.Exec(ctx, `DROP VIEW IF EXISTS "events"`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `ALTER TABLE IF EXISTS "krenalis_destination_profiles" DROP CONSTRAINT IF EXISTS "krenalis_destination_profiles_pkey"`); err != nil {
			return err
		}
		if err := postgresqlMigrateBase58Column(ctx, tx, "krenalis_events", "connection_id", connectionIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateBase58Column(ctx, tx, "krenalis_identities", "_connection", connectionIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateBase58Column(ctx, tx, "krenalis_identities", "_pipeline", pipelineIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateBase58Column(ctx, tx, "krenalis_identities", "_run", pipelineRunIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateBase58Column(ctx, tx, "krenalis_destination_profiles", "_pipeline", pipelineIDs); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `ALTER TABLE IF EXISTS "krenalis_destination_profiles" ADD PRIMARY KEY ("_pipeline", "_external_id")`); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE OR REPLACE VIEW "events" AS SELECT * FROM "krenalis_events"`); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO "krenalis_internal_migrations" ("id", "workspace") VALUES ($1, $2)`, migration, workspace)
		return err
	})
}

func postgresqlMigrateBase58Column(ctx context.Context, tx pgx.Tx, table, column string, ids map[int]string) error {
	typ, err := postgresqlColumnType(ctx, tx, table, column)
	if err != nil {
		return err
	}
	if typ == "" || typ == "text" || strings.HasPrefix(typ, "character varying") {
		return nil
	}
	if typ != "integer" {
		return fmt.Errorf("%s.%s has type %s instead of integer", table, column, typ)
	}

	qTable := quoteIdent(table)
	qColumn := quoteIdent(column)
	qNewColumn := quoteIdent(column + "_base58")
	if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s text`, qTable, qNewColumn)); err != nil {
		return err
	}
	if len(ids) > 0 {
		query, args := postgresqlMigrateBase58ColumnQuery(table, column, ids)
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	var missing int
	err = tx.QueryRow(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s IS NOT NULL AND %s IS NULL`, qTable, qColumn, qNewColumn)).Scan(&missing)
	if err != nil {
		return err
	}
	if missing != 0 {
		return fmt.Errorf("cannot migrate %s.%s: %d value(s) have no ID mapping", table, column, missing)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, qTable, qColumn)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, qTable, qNewColumn, qColumn)); err != nil {
		return err
	}
	if column != "_run" {
		_, err = tx.Exec(ctx, fmt.Sprintf(`ALTER TABLE %s ALTER COLUMN %s SET NOT NULL`, qTable, qColumn))
		return err
	}
	return nil
}

func postgresqlColumnType(ctx context.Context, tx pgx.Tx, table, column string) (string, error) {
	var typ string
	err := tx.QueryRow(ctx, `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND a.attname = $2
			AND NOT a.attisdropped`, table, column).Scan(&typ)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return typ, err
}

func postgresqlMigrateBase58ColumnQuery(table, column string, ids map[int]string) (string, []any) {
	oldIDs := make([]int, 0, len(ids))
	for oldID := range ids {
		oldIDs = append(oldIDs, oldID)
	}
	sort.Ints(oldIDs)
	args := make([]any, 0, len(ids)*2)
	var values strings.Builder
	for i, oldID := range oldIDs {
		if i > 0 {
			values.WriteByte(',')
		}
		args = append(args, oldID, ids[oldID])
		fmt.Fprintf(&values, "($%d::integer, $%d::text)", len(args)-1, len(args))
	}
	qTable := quoteIdent(table)
	qColumn := quoteIdent(column)
	qNewColumn := quoteIdent(column + "_base58")
	query := fmt.Sprintf(`WITH "id_map" ("old_id", "new_id") AS (VALUES %s)
UPDATE %s SET %s = "id_map"."new_id"
FROM "id_map"
WHERE %s = "id_map"."old_id"`, values.String(), qTable, qNewColumn, qColumn)
	return query, args
}
