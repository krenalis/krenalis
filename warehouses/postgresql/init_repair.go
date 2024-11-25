//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package postgresql

import (
	"context"
	"fmt"
	"strings"

	"github.com/meergo/meergo"
)

// CanInitialize checks whether the data warehouse can be initialized.
func (warehouse *PostgreSQL) CanInitialize(ctx context.Context) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	const query = `SELECT
		"c"."relname",
		CASE
			"c"."relkind"
			WHEN 'r' THEN 'table'
			WHEN 'm' THEN 'materialized view'
			WHEN 'i' THEN 'index'
			WHEN 'S' THEN 'sequence'
			WHEN 'v' THEN 'view'
			WHEN 'c' THEN 'type'
			ELSE "c"."relkind"::text
		END
	FROM
		"pg_class" "c"
		JOIN "pg_namespace" "n" ON "n"."oid" = "c"."relnamespace"
	WHERE
		"n"."nspname" = $1
		AND "n"."nspname" NOT LIKE 'pg_toast%'
	ORDER BY
		"c"."relname"`
	rows, err := pool.Query(ctx, query, warehouse.settings.Schema)
	if err != nil {
		return err
	}
	defer rows.Close()
	var objects []string
	for rows.Next() {
		var name, typ string
		err := rows.Scan(&name, &typ)
		if err != nil {
			return err
		}
		objects = append(objects, fmt.Sprintf("%s '%s'", typ, name))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if objects != nil {
		reason := fmt.Sprintf("expected an empty database, got: %s", strings.Join(objects, ", "))
		return meergo.NewWarehouseNonInitializableError(reason)
	}
	return nil
}

// Initialize initializes the database objects on the data warehouse in order to
// make it work with Meergo.
func (warehouse *PostgreSQL) Initialize(ctx context.Context) error {
	return warehouse.initRepair(ctx, false)
}

// Repair repairs the database objects on the data warehouse needed by Meergo.
func (warehouse *PostgreSQL) Repair(ctx context.Context) error {
	return warehouse.initRepair(ctx, true)
}

// initRepair initializes (or repairs) the database objects (as tables, types,
// etc...) on the data warehouse.
func (warehouse *PostgreSQL) initRepair(ctx context.Context, repair bool) error {
	pool, err := warehouse.connectionPool(ctx)
	if err != nil {
		return err
	}
	queries := []string{
		createDestinationUsersTable,
		createEventsTable,
		createOperationsTable,
		createUserIdentitiesTable,
		createUsersTable,
	}
	if !repair {
		// Since the "CREATE VIEW IF NOT EXISTS" statement does not exist in
		// PostgreSQL, the view is recreated only if initializing, not when
		// repairing, otherwise a "cannot drop columns from view" error is
		// returned by PostgreSQL in cases where the users table has different
		// columns than the default one.
		//
		// TODO(Gianluca): clarify this.
		queries = append(queries, createUsersView)
	}
	for _, query := range queries {
		_, err := pool.Exec(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}
