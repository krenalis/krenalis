// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	_ "embed"
	"strings"

	"github.com/krenalis/krenalis/core/internal/db"
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
		AND "n"."nspname" NOT LIKE 'pg\_toast%'`
	var count int
	err := db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		return false, err
	}
	return count == 0, nil
}

//go:embed "schema.sql"
var schema string

// Initialize initializes the provided PostgreSQL database by executing queries
// in the given transaction, creating all the database objects (tables, types,
// etc.) needed to run Krenalis.
//
// This function must be called on a transaction opened on an empty database.
// Otherwise, the behavior is undefined.
func Initialize(ctx context.Context, tx *db.Tx) error {
	for query := range strings.SplitSeq(schema, ";\n") {
		query = strings.TrimSpace(query)
		if query == "" {
			continue
		}
		_, err := tx.Exec(ctx, query)
		if err != nil {
			return err
		}
	}
	return nil
}

// InitializeDockerMember initializes a Krenalis member on the given PostgreSQL
// database (by executing queries in the given transaction) for certain
// scenarios where Krenalis is running with Docker, e.g., with the configuration
// we provide in Docker Compose (this user is treated differently, for example,
// by the Admin).
//
// This function is intended to be called after a successful call to Initialize,
// on its same transaction.
//
// Specifically, this function:
//
//  1. Deletes the members already present in the PostgreSQL database;
//
//  2. Creates a new member whose email is "docker@krenalis.com" and whose
//     password is "krenalis-password".
func InitializeDockerMember(ctx context.Context, tx *db.Tx) error {
	_, err := tx.Exec(ctx, "TRUNCATE members")
	if err != nil {
		return err
	}
	const query = `INSERT INTO members (organization, name, avatar, email, password, created_at)
		SELECT id, 'User', NULL, 'docker@krenalis.com', '$2a$10$1arUoJQAeIVLAuNiErG29ex2r43n/4bJZWmW/PPOiWaSt4ZCH5Ysm', now() at time zone 'utc'
		FROM organizations`
	_, err = tx.Exec(ctx, query)
	if err != nil {
		return err
	}
	return err
}
