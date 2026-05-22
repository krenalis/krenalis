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

var errOrganizationIDAlreadyInteger = errors.New("organization ID is already integer")

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

// UpgradeOrganizationIDToInt changes the organization ID from UUID to integer.
// The new integer IDs preserve the previous organization ordering by UUID.
func UpgradeOrganizationIDToInt(ctx context.Context, db *db.DB) error {
	err := db.Transaction(ctx, func(tx *dbpkg.Tx) error {
		var typ string
		err := tx.QueryRow(ctx, `
			SELECT format_type(a.atttypid, a.atttypmod)
			FROM pg_attribute a
			JOIN pg_class c ON c.oid = a.attrelid
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema()
				AND c.relname = 'organizations'
				AND a.attname = 'id'
				AND NOT a.attisdropped`).Scan(&typ)
		if err != nil {
			return err
		}
		if typ == "integer" {
			return errOrganizationIDAlreadyInteger
		}
		if typ != "uuid" {
			return fmt.Errorf("organizations.id has type %s instead of uuid", typ)
		}

		lockQuery := `LOCK TABLE organizations, members, workspaces`
		accessKeysExists, err := tx.QueryExists(ctx, `
			SELECT FROM pg_class c
			JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema()
				AND c.relname = 'access_keys'
				AND c.relkind = 'r'`)
		if err != nil {
			return err
		}
		if accessKeysExists {
			lockQuery += `, access_keys`
		}
		lockQuery += ` IN ACCESS EXCLUSIVE MODE`

		queries := []string{
			lockQuery,
			`CREATE TEMP TABLE organization_id_map ON COMMIT DROP AS
				SELECT id AS old_id, row_number() OVER (ORDER BY id)::integer AS new_id
				FROM organizations`,
			`ALTER TABLE organizations ADD COLUMN id_new integer`,
			`UPDATE organizations
				SET id_new = organization_id_map.new_id
				FROM organization_id_map
				WHERE organizations.id = organization_id_map.old_id`,
			`ALTER TABLE organizations ALTER COLUMN id_new SET NOT NULL`,

			`ALTER TABLE members ADD COLUMN organization_new integer`,
			`UPDATE members
				SET organization_new = organization_id_map.new_id
				FROM organization_id_map
				WHERE members.organization = organization_id_map.old_id`,
			`ALTER TABLE members ALTER COLUMN organization_new SET NOT NULL`,

			`ALTER TABLE workspaces ADD COLUMN organization_new integer`,
			`UPDATE workspaces
				SET organization_new = organization_id_map.new_id
				FROM organization_id_map
				WHERE workspaces.organization = organization_id_map.old_id`,
			`ALTER TABLE workspaces ALTER COLUMN organization_new SET NOT NULL`,
		}
		if accessKeysExists {
			queries = append(queries,
				`ALTER TABLE access_keys ADD COLUMN organization_new integer`,
				`UPDATE access_keys
					SET organization_new = organization_id_map.new_id
					FROM organization_id_map
					WHERE access_keys.organization = organization_id_map.old_id`,
				`ALTER TABLE access_keys ALTER COLUMN organization_new SET NOT NULL`,
			)
		}
		queries = append(queries,
			`ALTER TABLE members DROP CONSTRAINT IF EXISTS members_organization_fkey`,
			`ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS workspaces_organization_fkey`,
			`ALTER TABLE members DROP CONSTRAINT IF EXISTS members_organization_email_key`,
		)
		if accessKeysExists {
			queries = append(queries, `ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS access_keys_organization_fkey`)
		}
		queries = append(queries,
			`ALTER TABLE organizations DROP CONSTRAINT IF EXISTS organizations_pkey`,
			`ALTER TABLE members DROP COLUMN organization`,
			`ALTER TABLE members RENAME COLUMN organization_new TO organization`,
			`ALTER TABLE workspaces DROP COLUMN organization`,
			`ALTER TABLE workspaces RENAME COLUMN organization_new TO organization`,
		)
		if accessKeysExists {
			queries = append(queries,
				`ALTER TABLE access_keys DROP COLUMN organization`,
				`ALTER TABLE access_keys RENAME COLUMN organization_new TO organization`,
			)
		}
		queries = append(queries,
			`ALTER TABLE organizations DROP COLUMN id`,
			`ALTER TABLE organizations RENAME COLUMN id_new TO id`,

			`ALTER TABLE organizations ADD CONSTRAINT organizations_pkey PRIMARY KEY (id)`,
			`ALTER TABLE organizations ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY`,
			`SELECT setval(pg_get_serial_sequence('organizations', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM organizations`,
			`ALTER TABLE members ADD CONSTRAINT members_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`,
			`ALTER TABLE members ADD CONSTRAINT members_organization_email_key UNIQUE (organization, email)`,
			`ALTER TABLE workspaces ADD CONSTRAINT workspaces_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`,
		)
		if accessKeysExists {
			queries = append(queries, `ALTER TABLE access_keys ADD CONSTRAINT access_keys_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`)
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == errOrganizationIDAlreadyInteger {
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
