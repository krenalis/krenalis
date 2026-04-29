// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/krenalis/krenalis/core/internal/db"
	dbpkg "github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/kms"
)

const kmsEncryptedKeys = 4

var errAPIKeyPepperAlreadyExists = errors.New("API key pepper already exists")

// InitIfEmpty initializes the PostgreSQL database if it is empty.
// If dockerMember is true, it also initializes the Docker member.
func InitIfEmpty(ctx context.Context, db *db.DB, kms kms.Kms, dockerMember bool) error {

	isEmpty, err := databaseIsEmpty(ctx, db)
	if err != nil {
		return fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
	}
	if !isEmpty {
		slog.Info("the PostgreSQL database is not empty, so it won't be initialized")
		return nil
	}
	slog.Info("the PostgreSQL database is empty, so the database will be initialized...")

	// Generate the kms-encrypted data keys.
	var kmsEncryptedCookieKey, kmsEncryptedOAuthKey, kmsEncryptedNotificationKey, kmsEncryptedAPIKeyPepper []byte
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, kmsEncryptedKeys)
	go func() {
		var err error
		kmsEncryptedCookieKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 64)
		done <- err
	}()
	go func() {
		var err error
		kmsEncryptedOAuthKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
		done <- err
	}()
	go func() {
		var err error
		kmsEncryptedNotificationKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
		done <- err
	}()
	go func() {
		var err error
		kmsEncryptedAPIKeyPepper, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
		done <- err
	}()

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
		for range kmsEncryptedKeys {
			if err = <-done; err != nil {
				return fmt.Errorf("failed to generate key using KMS: %s", err)
			}
		}
		err = initializeKmsEncryptedKeys(ctx, tx, kmsEncryptedCookieKey, kmsEncryptedOAuthKey, kmsEncryptedNotificationKey, kmsEncryptedAPIKeyPepper)
		if err != nil {
			return err
		}
		slog.Info("Admin console and notifications keys created")
		return nil
	})

	return err
}

// UpgradeAddAPIKeyPepper adds the KMS-encrypted API key pepper to the database.
// Existing access keys are deleted.
func UpgradeAddAPIKeyPepper(ctx context.Context, db *db.DB, kms kms.Kms) error {
	key, err := kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		return fmt.Errorf("failed to generate API key pepper using KMS: %s", err)
	}
	err = db.Transaction(ctx, func(tx *dbpkg.Tx) error {
		_, err := tx.Exec(ctx, "ALTER TABLE metadata ADD COLUMN kms_encrypted_api_key_pepper bytea")
		if err != nil {
			if dbpkg.IsDuplicateColumn(err) {
				return errAPIKeyPepperAlreadyExists
			}
			return err
		}
		result, err := tx.Exec(ctx, "UPDATE metadata SET kms_encrypted_api_key_pepper = $1", key)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return errors.New("row is missing from table 'metadata'")
		}
		_, err = tx.Exec(ctx, "ALTER TABLE metadata ALTER COLUMN kms_encrypted_api_key_pepper SET NOT NULL")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "DROP TABLE IF EXISTS access_keys")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `CREATE TABLE access_keys (
			id integer NOT NULL,
			organization uuid NOT NULL REFERENCES organizations ON DELETE CASCADE,
			workspace integer REFERENCES workspaces ON DELETE CASCADE,
			name varchar(100) NOT NULL,
			type access_key_type NOT NULL,
			hmac bytea NOT NULL UNIQUE,
			hint varchar(13) NOT NULL,
			created_at timestamp(0) NOT NULL,
			PRIMARY KEY (id)
		)`)
		if err != nil {
			return err
		}
		return nil
	})
	if err == errAPIKeyPepperAlreadyExists {
		return nil
	}
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

// initializeKmsEncryptedKeys initializes the KMS-encrypted data keys
// used for cookies, OAuth, notifications, and API keys.
func initializeKmsEncryptedKeys(ctx context.Context, tx *db.Tx, cookieKey, oauthKey, notificationKey, apiKeyPepper []byte) error {
	const query = `UPDATE metadata SET kms_encrypted_cookie_key = $1, kms_encrypted_oauth_key = $2,
		kms_encrypted_notification_key = $3, kms_encrypted_api_key_pepper = $4 WHERE singleton`
	result, err := tx.Exec(ctx, query, cookieKey, oauthKey, notificationKey, apiKeyPepper)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("row is missing from table 'metadata'")
	}
	return nil
}
