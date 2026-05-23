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

// MigrateRenumberedInternalIDs updates renumbered Krenalis IDs in the warehouse.
func (warehouse *PostgreSQL) MigrateRenumberedInternalIDs(ctx context.Context, migration string, workspace int, connectionIDs, pipelineIDs map[int]int) error {
	return warehouse.execTransaction(ctx, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS "krenalis_internal_migrations" (
			"id" character varying NOT NULL,
			"workspace" integer NOT NULL,
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
		if err := postgresqlMigrateIntColumn(ctx, tx, `"krenalis_events"`, `"connection_id"`, connectionIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateIntColumn(ctx, tx, `"krenalis_identities"`, `"_connection"`, connectionIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateIntColumn(ctx, tx, `"krenalis_identities"`, `"_pipeline"`, pipelineIDs); err != nil {
			return err
		}
		if err := postgresqlMigrateIntColumn(ctx, tx, `"krenalis_destination_profiles"`, `"_pipeline"`, pipelineIDs); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO "krenalis_internal_migrations" ("id", "workspace") VALUES ($1, $2)`, migration, workspace)
		return err
	})
}

// postgresqlMigrateIntColumn remaps integer IDs in table.column.
func postgresqlMigrateIntColumn(ctx context.Context, tx pgx.Tx, table, column string, ids map[int]int) error {
	if len(ids) == 0 {
		return nil
	}
	query, args := postgresqlMigrateIntColumnQuery(table, column, ids)
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, fmt.Sprintf(`UPDATE %s SET %s = -%s WHERE %s < 0`, table, column, column, column))
	return err
}

// postgresqlMigrateIntColumnQuery returns the first phase of an ID remap query.
func postgresqlMigrateIntColumnQuery(table, column string, ids map[int]int) (string, []any) {
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
		fmt.Fprintf(&values, "($%d::integer, $%d::integer)", len(args)-1, len(args))
	}
	query := fmt.Sprintf(`WITH "id_map" ("old_id", "new_id") AS (VALUES %s)
UPDATE %s SET %s = -"id_map"."new_id"
FROM "id_map"
WHERE %s = "id_map"."old_id"`, values.String(), table, column, column)
	return query, args
}
