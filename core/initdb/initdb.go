// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"strings"

	_ "embed"

	"github.com/meergo/meergo/core/internal/db"
)

func DatabaseIsEmpty(ctx context.Context, db *db.DB) (bool, error) {
	const query = `SELECT COUNT(*)
	FROM
		"pg_class" "c"
		JOIN "pg_namespace" "n" ON "n"."oid" = "c"."relnamespace"
	WHERE
		"n"."nspname" = current_schema()
		AND "n"."nspname" NOT LIKE 'pg_\toast%'`
	var count int
	err := db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

//go:embed "DB_initialization_queries.sql"
var script1 string

// TODO: inizializzare anche l'utente Docker

func InitializeDB(ctx context.Context, db *db.DB) error {
	queries := strings.Split(string(script1), ";\n")
	for _, query := range queries {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err := db.Exec(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}
