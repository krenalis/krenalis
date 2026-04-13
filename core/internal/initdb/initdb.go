// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"strings"

	"github.com/krenalis/krenalis/core/internal/db"
	dbpkg "github.com/krenalis/krenalis/core/internal/db"
)

// InitIfEmpty initializes the PostgreSQL database if it is empty.
// If dockerMember is true, it also initializes the Docker member.
func InitIfEmpty(ctx context.Context, db *db.DB, dockerMember bool) error {
	isEmpty, err := databaseIsEmpty(ctx, db)
	if err != nil {
		return fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if !isEmpty {
		slog.Info("the PostgreSQL database is not empty, so it won't be initialized")
		return nil
	}
	slog.Info("the PostgreSQL database is empty, so the database will be initialized...")
	// Initialize the PostgreSQL database in a transaction, so if it is
	// fails, there is no need to manually empty the database.
	err = db.Transaction(ctx, func(tx *dbpkg.Tx) error {
		err := initialize(ctx, tx)
		if err != nil {
			return fmt.Errorf("cannot initialize PostgreSQL database: %s", err)
		}
		slog.Info("PostgreSQL database initialized correctly")
		// Also initialize the Docker member, if requested.
		if dockerMember {
			slog.Info("initializing Docker member...")
			err := initializeDockerMember(ctx, tx)
			if err != nil {
				return fmt.Errorf("cannot initialize the Docker member: %s", err)
			}
			slog.Info("Docker member initialized")
		}
		return nil
	})
	return err
}

// databaseIsEmpty reports whether the given PostgreSQL database is empty, that
// is, if it does not contain any database objects (such as tables, views,
// types, etc.).
func databaseIsEmpty(ctx context.Context, db *db.DB) (bool, error) {
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

// initialize initializes the provided PostgreSQL database by executing queries
// in the given transaction, creating all the database objects (tables, types,
// etc.) needed to run Krenalis.
//
// This function must be called on a transaction opened on an empty database.
// Otherwise, the behavior is undefined.
func initialize(ctx context.Context, tx *db.Tx) error {
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

// initializeDockerMember initializes a Krenalis member on the given PostgreSQL
// database (by executing queries in the given transaction) for certain
// scenarios where Krenalis is running with Docker, e.g., with the configuration
// we provide in Docker Compose (this user is treated differently, for example,
// by the Admin).
//
// This function is intended to be called after a successful call to initialize,
// on its same transaction.
//
// Specifically, this function:
//
//  1. Deletes the members already present in the PostgreSQL database;
//
//  2. Creates a new member whose email is "docker@krenalis.com" and whose
//     password is "krenalis-password".
func initializeDockerMember(ctx context.Context, tx *db.Tx) error {
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
