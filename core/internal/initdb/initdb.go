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

	kmsEncryptedHTTPSecretKey, kmsEncryptedOAuthKey, kmsEncryptedNotificationKey, kmsEncryptedAPIKeyPepper, err := generateKmsEncryptedKeys(ctx, kms)
	if err != nil {
		return err
	}
	defer clear(kmsEncryptedHTTPSecretKey)
	defer clear(kmsEncryptedOAuthKey)
	defer clear(kmsEncryptedNotificationKey)
	defer clear(kmsEncryptedAPIKeyPepper)

	// Initialize the PostgreSQL database in a transaction, so if it is
	// fails, there is no need to manually empty the database.
	err = db.Transaction(ctx, func(tx *dbpkg.Tx) error {
		err := initialize(ctx, tx, dockerMember)
		if err != nil {
			return fmt.Errorf("cannot initialize PostgreSQL database: %s", err)
		}
		slog.Info("PostgreSQL database initialized correctly")
		err = initializeKmsEncryptedKeys(ctx, tx, kmsEncryptedHTTPSecretKey, kmsEncryptedOAuthKey, kmsEncryptedNotificationKey, kmsEncryptedAPIKeyPepper)
		if err != nil {
			return err
		}
		slog.Info("Admin console and notifications keys created")
		return nil
	})

	return err
}

// generateKmsEncryptedKeys generates the KMS-encrypted data keys used during
// database initialization.
func generateKmsEncryptedKeys(ctx context.Context, kms kms.Kms) (httpSecretKey, oauthKey, notificationKey, apiKeyPepper []byte, err error) {
	defer func() {
		if err != nil {
			clear(httpSecretKey)
			clear(oauthKey)
			clear(notificationKey)
			clear(apiKeyPepper)
		}
	}()

	httpSecretKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 64)
	if err != nil {
		err = fmt.Errorf("failed to generate HTTP secret key using KMS: %s", err)
		return
	}
	oauthKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		err = fmt.Errorf("failed to generate OAuth key using KMS: %s", err)
		return
	}
	notificationKey, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		err = fmt.Errorf("failed to generate notification key using KMS: %s", err)
		return
	}
	apiKeyPepper, err = kms.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		err = fmt.Errorf("failed to generate API key pepper using KMS: %s", err)
		return
	}
	return
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

// The PL/pgSQL function is kept outside schema.sql because schema.sql is split
// on SQL statement terminators during initialization.
//
//go:embed api_rate_limiter_leases.sql
var createAPIRateLimiterLeasesFunction string

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
	if _, err := tx.Exec(ctx, createAPIRateLimiterLeasesFunction); err != nil {
		return err
	}
	// Insert the organization.
	organizationID := base58.Generate(12)
	_, err := tx.Exec(ctx, `INSERT INTO organizations`+
		` (id, name, enabled, members_limit, access_keys_limit, workspaces_limit, connectors_limit, connections_limit, pipelines_limit,`+
		` api_workspace_quota_per_hour, api_workspace_burst_capacity, api_ingestion_quota_per_hour, api_ingestion_burst_capacity,`+
		` api_nonspecific_quota_per_hour, api_nonspecific_burst_capacity)`+
		` VALUES ($1, 'ACME inc', true, 10000, 1000, 1000, 1000, 10000, 10000, 25000, 1000, 25000, 1000, 25000, 1000)`,
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

// initializeKmsEncryptedKeys initializes the KMS-encrypted data keys used by
// metadata-scoped secrets.
func initializeKmsEncryptedKeys(ctx context.Context, tx *db.Tx, httpSecretKey, oauthKey, notificationKey, apiKeyPepper []byte) error {
	const query = `UPDATE metadata SET kms_encrypted_http_secret_key = $1, kms_encrypted_oauth_key = $2,
		kms_encrypted_notification_key = $3, kms_encrypted_api_key_pepper = $4 WHERE singleton`
	result, err := tx.Exec(ctx, query, httpSecretKey, oauthKey, notificationKey, apiKeyPepper)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("row is missing from table 'metadata'")
	}
	return nil
}
