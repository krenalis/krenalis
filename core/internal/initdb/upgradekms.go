// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/kms"
)

type upgradeDBState int

const (
	upgradeDBStateUnexpected upgradeDBState = iota
	upgradeDBStateEmpty
	upgradeDBStateOld
	upgradeDBStateNew
)

type workspaceSettingsRow struct {
	ID           int
	SettingsJSON string
	MCPJSON      string
}

type connectionSettingsRow struct {
	ID           int
	SettingsJSON sql.NullString
}

// UpgradeToKMS upgrades an existing PostgreSQL database from the legacy schema
// to the KMS-backed encrypted schema used by the current codebase.
func UpgradeToKMS(ctx context.Context, database *db.DB, keyManager kms.Kms) error {
	state, err := detectUpgradeDBState(ctx, database)
	if err != nil {
		return err
	}
	switch state {
	case upgradeDBStateEmpty:
		return errors.New("database is empty; use -init-db-if-empty")
	case upgradeDBStateNew:
		slog.Info("the PostgreSQL database is already upgraded")
		return nil
	case upgradeDBStateUnexpected:
		return errors.New("database schema is not in the expected pre-upgrade format")
	}

	slog.Info("upgrading PostgreSQL database to KMS-backed encrypted settings...")
	err = database.Transaction(ctx, func(tx *db.Tx) error {
		return upgradeToKMS(ctx, tx, keyManager)
	})
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database upgraded successfully")
	return nil
}

func upgradeToKMS(ctx context.Context, tx *db.Tx, keyManager kms.Kms) error {
	installationID, err := loadLegacyInstallationID(ctx, tx)
	if err != nil {
		return err
	}

	err = renameLegacyColumns(ctx, tx)
	if err != nil {
		return err
	}
	err = addUpgradedColumns(ctx, tx)
	if err != nil {
		return err
	}
	err = createUpgradedMetadataTable(ctx, tx)
	if err != nil {
		return err
	}

	c := cipher.New(keyManager)
	if err = migrateWorkspaceSettings(ctx, tx, c); err != nil {
		return err
	}
	if err = migrateConnectionSettings(ctx, tx, c); err != nil {
		return err
	}
	if err = populateUpgradedMetadata(ctx, tx, keyManager, installationID); err != nil {
		return err
	}
	if err = dropLegacyColumns(ctx, tx); err != nil {
		return err
	}
	if err = setUpgradedNotNullConstraints(ctx, tx); err != nil {
		return err
	}
	return nil
}

func detectUpgradeDBState(ctx context.Context, conn db.Connection) (upgradeDBState, error) {
	if dbconn, ok := conn.(*db.DB); ok {
		isEmpty, err := databaseIsEmpty(ctx, dbconn)
		if err != nil {
			return upgradeDBStateUnexpected, fmt.Errorf("cannot check if PostgreSQL database is empty or not: %s", err)
		}
		if isEmpty {
			return upgradeDBStateEmpty, nil
		}
	}

	legacy, err := hasLegacyUpgradeSchema(ctx, conn)
	if err != nil {
		return upgradeDBStateUnexpected, err
	}
	upgraded, err := hasUpgradedSchema(ctx, conn)
	if err != nil {
		return upgradeDBStateUnexpected, err
	}

	switch {
	case legacy && !upgraded:
		return upgradeDBStateOld, nil
	case upgraded && !legacy:
		ok, err := hasUsableUpgradedData(ctx, conn)
		if err != nil {
			return upgradeDBStateUnexpected, err
		}
		if !ok {
			return upgradeDBStateUnexpected, nil
		}
		return upgradeDBStateNew, nil
	default:
		return upgradeDBStateUnexpected, nil
	}
}

func hasLegacyUpgradeSchema(ctx context.Context, conn db.Connection) (bool, error) {
	return allColumnsMatch(ctx, conn, map[[2]string]string{
		{"metadata", "key"}:                      "text",
		{"metadata", "value"}:                    "text",
		{"workspaces", "warehouse_settings"}:     "jsonb",
		{"workspaces", "warehouse_mcp_settings"}: "jsonb",
		{"connections", "settings"}:              "jsonb",
	})
}

func hasUpgradedSchema(ctx context.Context, conn db.Connection) (bool, error) {
	return allColumnsMatch(ctx, conn, map[[2]string]string{
		{"metadata", "singleton"}:                                  "bool",
		{"metadata", "installation_id"}:                            "text",
		{"metadata", "kms_encrypted_cookie_key"}:                   "bytea",
		{"metadata", "kms_encrypted_oauth_key"}:                    "bytea",
		{"metadata", "kms_encrypted_notification_key"}:             "bytea",
		{"workspaces", "warehouse_settings"}:                       "bytea",
		{"workspaces", "warehouse_mcp_settings"}:                   "bytea",
		{"workspaces", "kms_encrypted_warehouse_settings_key"}:     "bytea",
		{"workspaces", "kms_encrypted_warehouse_mcp_settings_key"}: "bytea",
		{"connections", "settings"}:                                "bytea",
		{"connections", "kms_encrypted_settings_key"}:              "bytea",
	})
}

func hasUsableUpgradedData(ctx context.Context, conn db.Connection) (bool, error) {
	ok, err := hasValidUpgradedMetadata(ctx, conn)
	if err != nil || !ok {
		return ok, err
	}
	return hasValidUpgradedSettingsKeys(ctx, conn)
}

func columnType(ctx context.Context, conn db.Connection, table, column string) (typ string, ok bool, err error) {
	const query = `SELECT "t"."typname"
		FROM "pg_attribute" "a"
		JOIN "pg_class" "c" ON "c"."oid" = "a"."attrelid"
		JOIN "pg_namespace" "n" ON "n"."oid" = "c"."relnamespace"
		JOIN "pg_type" "t" ON "t"."oid" = "a"."atttypid"
		WHERE
			"n"."nspname" = current_schema()
			AND "c"."relname" = $1
			AND "a"."attname" = $2
			AND "a"."attnum" > 0
			AND NOT "a"."attisdropped"`
	err = conn.QueryRow(ctx, query, table, column).Scan(&typ)
	switch {
	case err == nil:
		return typ, true, nil
	case err == sql.ErrNoRows:
		return "", false, nil
	default:
		return "", false, err
	}
}

func allColumnsMatch(ctx context.Context, conn db.Connection, expected map[[2]string]string) (bool, error) {
	for column, want := range expected {
		got, ok, err := columnType(ctx, conn, column[0], column[1])
		if err != nil {
			return false, err
		}
		if !ok || got != want {
			return false, nil
		}
	}
	return true, nil
}

func hasValidUpgradedMetadata(ctx context.Context, conn db.Connection) (bool, error) {
	var count int
	var installationID string
	var cookieKey, oAuthKey, notificationKey []byte
	err := conn.QueryRow(ctx, `SELECT
		COUNT(*),
		COALESCE(MIN("installation_id"), ''),
		COALESCE(MIN("kms_encrypted_cookie_key"), '\x'::bytea),
		COALESCE(MIN("kms_encrypted_oauth_key"), '\x'::bytea),
		COALESCE(MIN("kms_encrypted_notification_key"), '\x'::bytea)
		FROM "metadata"`).Scan(&count, &installationID, &cookieKey, &oAuthKey, &notificationKey)
	if err != nil {
		return false, err
	}
	return count == 1 &&
		installationID != "" &&
		len(cookieKey) > 0 &&
		len(oAuthKey) > 0 &&
		len(notificationKey) > 0, nil
}

func hasValidUpgradedSettingsKeys(ctx context.Context, conn db.Connection) (bool, error) {
	var invalidWorkspaces int
	err := conn.QueryRow(ctx, `SELECT COUNT(*)
		FROM "workspaces"
		WHERE
			"kms_encrypted_warehouse_settings_key" IS NULL
			OR octet_length("kms_encrypted_warehouse_settings_key") = 0
			OR "kms_encrypted_warehouse_mcp_settings_key" IS NULL
			OR octet_length("kms_encrypted_warehouse_mcp_settings_key") = 0`).Scan(&invalidWorkspaces)
	if err != nil {
		return false, err
	}
	if invalidWorkspaces > 0 {
		return false, nil
	}

	var invalidConnections int
	err = conn.QueryRow(ctx, `SELECT COUNT(*)
		FROM "connections"
		WHERE
			"kms_encrypted_settings_key" IS NULL
			OR octet_length("kms_encrypted_settings_key") = 0`).Scan(&invalidConnections)
	if err != nil {
		return false, err
	}
	return invalidConnections == 0, nil
}

func loadLegacyInstallationID(ctx context.Context, tx *db.Tx) (string, error) {
	var count int
	var installationID string
	err := tx.QueryRow(ctx, `SELECT COUNT(*), COALESCE(MIN("value"), '')
		FROM "metadata"
		WHERE "key" = 'installation_id'`).Scan(&count, &installationID)
	if err != nil {
		return "", err
	}
	if count != 1 {
		return "", errors.New("metadata must contain exactly one installation_id entry")
	}
	if installationID == "" {
		return "", errors.New("installation_id in metadata is empty")
	}
	return installationID, nil
}

func renameLegacyColumns(ctx context.Context, tx *db.Tx) error {
	queries := []string{
		`ALTER TABLE "workspaces" RENAME COLUMN "warehouse_settings" TO "warehouse_settings_old"`,
		`ALTER TABLE "workspaces" RENAME COLUMN "warehouse_mcp_settings" TO "warehouse_mcp_settings_old"`,
		`ALTER TABLE "connections" RENAME COLUMN "settings" TO "settings_old"`,
		`ALTER TABLE "metadata" RENAME TO "metadata_old"`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func addUpgradedColumns(ctx context.Context, tx *db.Tx) error {
	queries := []string{
		`ALTER TABLE "workspaces" ADD COLUMN "warehouse_settings" bytea`,
		`ALTER TABLE "workspaces" ADD COLUMN "warehouse_mcp_settings" bytea`,
		`ALTER TABLE "workspaces" ADD COLUMN "kms_encrypted_warehouse_settings_key" bytea`,
		`ALTER TABLE "workspaces" ADD COLUMN "kms_encrypted_warehouse_mcp_settings_key" bytea`,
		`ALTER TABLE "connections" ADD COLUMN "settings" bytea`,
		`ALTER TABLE "connections" ADD COLUMN "kms_encrypted_settings_key" bytea`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func createUpgradedMetadataTable(ctx context.Context, tx *db.Tx) error {
	const query = `CREATE TABLE "metadata" (
		"singleton" boolean PRIMARY KEY DEFAULT true CHECK ("singleton"),
		"installation_id" text UNIQUE NOT NULL,
		"kms_encrypted_cookie_key" bytea NOT NULL,
		"kms_encrypted_oauth_key" bytea NOT NULL,
		"kms_encrypted_notification_key" bytea NOT NULL
	)`
	_, err := tx.Exec(ctx, query)
	return err
}

func migrateWorkspaceSettings(ctx context.Context, tx *db.Tx, c *cipher.Cipher) error {
	var rows []workspaceSettingsRow
	err := tx.QueryScan(ctx,
		`SELECT "id", "warehouse_settings_old"::text, "warehouse_mcp_settings_old"::text FROM "workspaces" ORDER BY "id"`,
		func(dbRows *db.Rows) error {
			for dbRows.Next() {
				var row workspaceSettingsRow
				if err := dbRows.Scan(&row.ID, &row.SettingsJSON, &row.MCPJSON); err != nil {
					return err
				}
				rows = append(rows, row)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	for _, row := range rows {
		settingsCiphertext, settingsKey, err := c.Encrypt(ctx, []byte(row.SettingsJSON), 32)
		if err != nil {
			return fmt.Errorf("failed while upgrading workspace %d: %w", row.ID, err)
		}
		var mcpCiphertext []byte
		var mcpKey []byte
		if row.MCPJSON == "null" {
			mcpKey, err = generateEncryptedDataKey(ctx, c.KeyManager(), 32)
		} else {
			mcpCiphertext, mcpKey, err = c.Encrypt(ctx, []byte(row.MCPJSON), 32)
		}
		if err != nil {
			return fmt.Errorf("failed while upgrading workspace %d: %w", row.ID, err)
		}
		result, err := tx.Exec(ctx, `UPDATE "workspaces"
			SET
				"warehouse_settings" = $1,
				"warehouse_mcp_settings" = $2,
				"kms_encrypted_warehouse_settings_key" = $3,
				"kms_encrypted_warehouse_mcp_settings_key" = $4
			WHERE "id" = $5`,
			settingsCiphertext, mcpCiphertext, settingsKey, mcpKey, row.ID,
		)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("failed while upgrading workspace %d: row not found", row.ID)
		}
	}
	return nil
}

func migrateConnectionSettings(ctx context.Context, tx *db.Tx, c *cipher.Cipher) error {
	var rows []connectionSettingsRow
	err := tx.QueryScan(ctx,
		`SELECT "id", "settings_old"::text FROM "connections" ORDER BY "id"`,
		func(dbRows *db.Rows) error {
			for dbRows.Next() {
				var row connectionSettingsRow
				if err := dbRows.Scan(&row.ID, &row.SettingsJSON); err != nil {
					return err
				}
				rows = append(rows, row)
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	for _, row := range rows {
		var settingsCiphertext []byte
		var settingsKey []byte
		if row.SettingsJSON.Valid {
			var err error
			settingsCiphertext, settingsKey, err = c.Encrypt(ctx, []byte(row.SettingsJSON.String), 32)
			if err != nil {
				return fmt.Errorf("failed while upgrading connection %d: %w", row.ID, err)
			}
		} else {
			var err error
			settingsKey, err = generateEncryptedDataKey(ctx, c.KeyManager(), 32)
			if err != nil {
				return fmt.Errorf("failed while upgrading connection %d: %w", row.ID, err)
			}
		}
		result, err := tx.Exec(ctx, `UPDATE "connections"
			SET "settings" = $1, "kms_encrypted_settings_key" = $2
			WHERE "id" = $3`,
			settingsCiphertext, settingsKey, row.ID,
		)
		if err != nil {
			return err
		}
		if result.RowsAffected() != 1 {
			return fmt.Errorf("failed while upgrading connection %d: row not found", row.ID)
		}
	}
	return nil
}

func populateUpgradedMetadata(ctx context.Context, tx *db.Tx, keyManager kms.Kms, installationID string) error {
	cookieKey, err := generateEncryptedDataKey(ctx, keyManager, 64)
	if err != nil {
		return err
	}
	oAuthKey, err := generateEncryptedDataKey(ctx, keyManager, 32)
	if err != nil {
		return err
	}
	notificationKey, err := generateEncryptedDataKey(ctx, keyManager, 32)
	if err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `INSERT INTO "metadata"
		("singleton", "installation_id", "kms_encrypted_cookie_key", "kms_encrypted_oauth_key", "kms_encrypted_notification_key")
		VALUES (true, $1, $2, $3, $4)`,
		installationID, cookieKey, oAuthKey, notificationKey,
	)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return errors.New("failed to insert upgraded metadata")
	}
	return nil
}

func dropLegacyColumns(ctx context.Context, tx *db.Tx) error {
	queries := []string{
		`ALTER TABLE "workspaces" DROP COLUMN "warehouse_settings_old"`,
		`ALTER TABLE "workspaces" DROP COLUMN "warehouse_mcp_settings_old"`,
		`ALTER TABLE "connections" DROP COLUMN "settings_old"`,
		`DROP TABLE "metadata_old"`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func setUpgradedNotNullConstraints(ctx context.Context, tx *db.Tx) error {
	queries := []string{
		`ALTER TABLE "workspaces" ALTER COLUMN "warehouse_settings" SET NOT NULL`,
		`ALTER TABLE "workspaces" ALTER COLUMN "kms_encrypted_warehouse_settings_key" SET NOT NULL`,
		`ALTER TABLE "workspaces" ALTER COLUMN "kms_encrypted_warehouse_mcp_settings_key" SET NOT NULL`,
		`ALTER TABLE "connections" ALTER COLUMN "kms_encrypted_settings_key" SET NOT NULL`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	return nil
}

func generateEncryptedDataKey(ctx context.Context, keyManager kms.Kms, keyLen int) ([]byte, error) {
	encryptedDataKey, err := keyManager.GenerateDataKeyWithoutPlaintext(ctx, keyLen)
	if err != nil {
		return nil, err
	}
	if len(encryptedDataKey) == 0 {
		return nil, errors.New("KMS returned an empty encrypted data key")
	}
	return encryptedDataKey, nil
}
