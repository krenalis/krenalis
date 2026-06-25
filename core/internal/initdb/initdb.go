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
	"github.com/krenalis/krenalis/tools/base58"
	"github.com/krenalis/krenalis/tools/kms"
)

const kmsEncryptedKeys = 4

// InitIfEmpty initializes the PostgreSQL database if it is empty.
// If dockerMember is true, the initial member name and email are set to
// "User" and "docker@krenalis.com" instead of "ACME inc" and
// "acme@krenalis.com", respectively.
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
		err := initialize(ctx, tx, dockerMember)
		if err != nil {
			return fmt.Errorf("cannot initialize PostgreSQL database: %s", err)
		}
		slog.Info("PostgreSQL database initialized correctly")
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

var errWorkOSUserIDAlreadyExists = errors.New("workos_user_id column already exists")

// UpgradeWorkOS adds WorkOS support to the database: it adds the workos_user_id
// column to the members table, creates the corresponding unique index, widens
// the name and email columns to varchar(255), and adds 'DeleteMembers' to the
// notification_name enum.
func UpgradeWorkOS(ctx context.Context, db *db.DB) error {
	err := db.Transaction(ctx, func(tx *dbpkg.Tx) error {
		_, err := tx.Exec(ctx, "ALTER TABLE members ADD COLUMN workos_user_id varchar(255) NOT NULL DEFAULT ''")
		if err != nil {
			if dbpkg.IsDuplicateColumn(err) {
				return errWorkOSUserIDAlreadyExists
			}
			return err
		}
		_, err = tx.Exec(ctx, "CREATE UNIQUE INDEX members_workos_user_id_idx ON members (organization, workos_user_id) WHERE workos_user_id <> ''")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "ALTER TABLE organizations ALTER COLUMN name TYPE varchar(255)")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "ALTER TABLE members ALTER COLUMN name TYPE varchar(255)")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "ALTER TABLE members ALTER COLUMN email TYPE varchar(255)")
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "ALTER TYPE notification_name ADD VALUE 'DeleteMembers' AFTER 'DeleteMember'")
		return err
	})
	if err != nil {
		if err == errWorkOSUserIDAlreadyExists {
			slog.Info("PostgreSQL database is already up to date")
			return nil
		}
		return err
	}
	slog.Info("PostgreSQL database updated successfully")
	return nil
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

// initialize initializes the provided PostgreSQL database by executing the
// required queries within the given transaction. It creates all database
// objects (tables, types, etc.) needed to run Krenalis, as well as an
// organization and a member.
//
// If dockerMember is true, the initial member name and email are set to
// "User" and "docker@krenalis.com" instead of "ACME inc" and
// "acme@krenalis.com", respectively.
//
// This function must be called on a transaction opened on an empty database.
// Otherwise, the behavior is undefined.
func initialize(ctx context.Context, tx *db.Tx, dockerMember bool) error {
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
	// Insert the organization.
	organizationID := base58.Generate(12)
	_, err := tx.Exec(ctx, "INSERT INTO organizations (id, name, enabled) VALUES ($1, 'ACME inc', true)",
		organizationID)
	if err != nil {
		return err
	}
	// Insert the member with password "krenalis-password".
	memberID := base58.Generate(12)
	memberName := "ACME inc"
	memberEmail := "acme@krenalis.com"
	if dockerMember {
		memberName = "User"
		memberEmail = "docker@krenalis.com"
	}
	_, err = tx.Exec(ctx, "INSERT INTO members (id, organization, name, email, password, created_at)\n"+
		"VALUES ($1, $2, $3, $4, '$2a$10$1arUoJQAeIVLAuNiErG29ex2r43n/4bJZWmW/PPOiWaSt4ZCH5Ysm', now() at time zone 'utc')",
		memberID, organizationID, memberName, memberEmail)
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
