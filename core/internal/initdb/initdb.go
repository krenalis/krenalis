// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	_ "embed"
	"strings"

	"github.com/meergo/meergo/core/internal/db"
)

// DatabaseIsEmpty reports whether the given PostgreSQL database is empty, that
// is, if it does not contain any database objects (such as tables, views,
// types, etc.).
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
var initSQLQueries string

// Initialize initializes the provided PostgreSQL database, creating all the
// database objects (tables, types, etc.) needed to run Meergo.
//
// This function must be called on an empty database. Otherwise, the behavior is
// undefined.
func Initialize(ctx context.Context, db *db.DB) error {
	for query := range strings.SplitSeq(initSQLQueries, ";\n") {
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

// InitializeDockerMember initializes a Meergo member on the given PostgreSQL
// database for certain scenarios where Meergo is running with Docker, e.g.,
// with the configuration we provide in Docker Compose (this user is treated
// differently, for example, by the Admin).
//
// This function is intended to be called after a successful call to Initialize.
//
// Specifically, this function:
//
//  1. Deletes the members already present in the PostgreSQL database;
//
//  2. Creates a new member whose email is "docker@meergo.com" and whose
//     password is "meergo-password".
func InitializeDockerMember(ctx context.Context, db *db.DB) error {
	_, err := db.Exec(ctx, "TRUNCATE members")
	if err != nil {
		return err
	}
	const query = `INSERT INTO members (organization, name, avatar, email, password, created_at)
		SELECT id, 'User', NULL, 'docker@meergo.com', '$2a$10$dGlVroo3N23Vn99edSPe..xo1hhKzGLYafIjFQjazu3faeFizvW7m', now() at time zone 'utc'
		FROM organizations`
	_, err = db.Exec(ctx, query)
	if err != nil {
		return err
	}
	return err
}
