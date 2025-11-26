// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"strings"

	_ "embed"

	"github.com/meergo/meergo/core/internal/db"
)

func isEmpty(ctx context.Context, db *db.DB, schema string) (bool, error) {
	const query = `SELECT COUNT(*)
	FROM
		"pg_class" "c"
		JOIN "pg_namespace" "n" ON "n"."oid" = "c"."relnamespace"
	WHERE
		"n"."nspname" = $1
		AND "n"."nspname" NOT LIKE 'pg_\toast%'
	ORDER BY
		"c"."relname"`
	var count int
	err := db.QueryRow(ctx, query, schema).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

//go:embed "database_init_scripts/1 - postgres.sql"
var script1 string

// TODO: c'è davvero bisogno di questo secondo script? O riusciamo a fare il
// tutto semplicemente eseguendo una query codificata nel codice in Go?

//go:embed "database_init_scripts/2 - docker user.sql"
var script2 string

func initializeDB(ctx context.Context, db *db.DB) error {
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
