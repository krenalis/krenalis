// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/kms"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	upgradeTestDatabase = "krenalis"
	upgradeTestUser     = "krenalis"
	upgradeTestPassword = "krenalis"
)

func Test_UpgradeToKMS(t *testing.T) {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(upgradeTestDatabase),
		postgres.WithUsername(upgradeTestUser),
		postgres.WithPassword(upgradeTestPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			t.Error(err)
		}
	}()

	host, err := postgresContainer.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := postgresContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}

	conn, err := db.Open(&db.Options{
		Host:           host,
		Port:           int(port.Num()),
		Username:       upgradeTestUser,
		Password:       upgradeTestPassword,
		Database:       upgradeTestDatabase,
		Schema:         "public",
		MaxConnections: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err := seedLegacyUpgradeSchema(ctx, conn); err != nil {
		t.Fatal(err)
	}

	keyManager, err := kms.New(ctx, "local:"+base64.RawStdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}

	if err := UpgradeToKMS(ctx, conn, keyManager); err != nil {
		t.Fatalf("upgrade failed: %s", err)
	}
	if err := UpgradeToKMS(ctx, conn, keyManager); err != nil {
		t.Fatalf("second upgrade should be a no-op: %s", err)
	}

	state, err := detectUpgradeDBState(ctx, conn)
	if err != nil {
		t.Fatal(err)
	}
	if state != upgradeDBStateNew {
		t.Fatalf("unexpected state after upgrade: %v", state)
	}

	c := cipher.New(keyManager)
	assertWorkspaceSettings(t, ctx, conn, c)
	assertConnectionSettings(t, ctx, conn, c, keyManager)
	assertMetadata(t, ctx, conn, keyManager)
}

func seedLegacyUpgradeSchema(ctx context.Context, conn *db.DB) error {
	return conn.Transaction(ctx, func(tx *db.Tx) error {
		queries := []string{
			`CREATE TABLE "workspaces" (
				"id" integer PRIMARY KEY,
				"warehouse_name" varchar NOT NULL DEFAULT '',
				"warehouse_mode" text NOT NULL DEFAULT 'Normal',
				"warehouse_settings" jsonb NOT NULL,
				"warehouse_mcp_settings" jsonb NOT NULL DEFAULT 'null'::jsonb
			)`,
			`CREATE TABLE "connections" (
				"id" integer PRIMARY KEY,
				"workspace" integer NOT NULL,
				"name" varchar NOT NULL DEFAULT '',
				"connector" varchar,
				"role" text NOT NULL DEFAULT 'Source',
				"account" integer NOT NULL DEFAULT 0,
				"strategy" text,
				"sending_mode" text,
				"linked_connections" integer[],
				"settings" jsonb,
				"health" text NOT NULL DEFAULT 'Healthy'
			)`,
			`CREATE TABLE "metadata" (
				"key" text PRIMARY KEY,
				"value" text
			)`,
			`INSERT INTO "workspaces" ("id", "warehouse_name", "warehouse_mode", "warehouse_settings", "warehouse_mcp_settings") VALUES
				(1, 'PostgreSQL', 'Normal', '{"host":"warehouse","port":5432}', 'null'::jsonb),
				(2, 'PostgreSQL', 'Normal', '{"host":"warehouse-2","port":15432}', '{"readonly":true}')`,
			`INSERT INTO "connections" ("id", "workspace", "name", "connector", "role", "settings", "health") VALUES
				(11, 1, 'db source', 'postgresql', 'Source', '{"database":"db1","ssl":false}', 'Healthy'),
				(12, 1, 'db source 2', 'postgresql', 'Source', NULL, 'Healthy')`,
			`INSERT INTO "metadata" ("key", "value") VALUES
				('installation_id', 'install-123'),
				('encryption_key', 'ignored')`,
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
}

func assertWorkspaceSettings(t *testing.T, ctx context.Context, conn *db.DB, c *cipher.Cipher) {
	t.Helper()
	rows, err := conn.Query(ctx, `SELECT
		"id",
		"warehouse_settings",
		"warehouse_mcp_settings",
		"kms_encrypted_warehouse_settings_key",
		"kms_encrypted_warehouse_mcp_settings_key"
		FROM "workspaces" ORDER BY "id"`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	type expectedRow struct {
		settings map[string]any
		mcp      map[string]any
	}
	expected := map[int]expectedRow{
		1: {settings: map[string]any{"host": "warehouse", "port": float64(5432)}, mcp: nil},
		2: {settings: map[string]any{"host": "warehouse-2", "port": float64(15432)}, mcp: map[string]any{"readonly": true}},
	}

	for rows.Next() {
		var id int
		var settingsCiphertext []byte
		var mcpCiphertext []byte
		var settingsKey []byte
		var mcpKey []byte
		if err := rows.Scan(&id, &settingsCiphertext, &mcpCiphertext, &settingsKey, &mcpKey); err != nil {
			t.Fatal(err)
		}
		settings, err := c.Decrypt(ctx, settingsCiphertext, settingsKey)
		if err != nil {
			t.Fatal(err)
		}
		if !sameJSON(settings, expected[id].settings) {
			t.Fatalf("workspace %d settings mismatch: %s", id, settings)
		}
		if len(mcpKey) == 0 {
			t.Fatalf("workspace %d missing MCP encrypted key", id)
		}
		if expected[id].mcp == nil {
			if mcpCiphertext != nil {
				t.Fatalf("workspace %d expected nil MCP settings", id)
			}
			continue
		}
		mcpSettings, err := c.Decrypt(ctx, mcpCiphertext, mcpKey)
		if err != nil {
			t.Fatal(err)
		}
		if !sameJSON(mcpSettings, expected[id].mcp) {
			t.Fatalf("workspace %d MCP settings mismatch: %s", id, mcpSettings)
		}
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertConnectionSettings(t *testing.T, ctx context.Context, conn *db.DB, c *cipher.Cipher, keyManager kms.Kms) {
	t.Helper()
	rows, err := conn.Query(ctx, `SELECT "id", "settings", "kms_encrypted_settings_key" FROM "connections" ORDER BY "id"`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var settingsCiphertext []byte
		var settingsKey []byte
		if err := rows.Scan(&id, &settingsCiphertext, &settingsKey); err != nil {
			t.Fatal(err)
		}
		if len(settingsKey) == 0 {
			t.Fatalf("connection %d missing encrypted settings key", id)
		}
		if id == 11 {
			settings, err := c.Decrypt(ctx, settingsCiphertext, settingsKey)
			if err != nil {
				t.Fatal(err)
			}
			if !sameJSON(settings, map[string]any{"database": "db1", "ssl": false}) {
				t.Fatalf("connection 11 settings mismatch: %s", settings)
			}
			continue
		}
		if settingsCiphertext != nil {
			t.Fatalf("connection %d expected nil ciphertext", id)
		}
		key, err := keyManager.DecryptDataKey(ctx, settingsKey)
		if err != nil {
			t.Fatal(err)
		}
		if len(key) != 32 {
			t.Fatalf("connection %d expected 32-byte data key, got %d", id, len(key))
		}
		clear(key)
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
}

func sameJSON(data []byte, expected any) bool {
	var got any
	if err := json.Unmarshal(data, &got); err != nil {
		return false
	}
	return reflect.DeepEqual(got, expected)
}

func assertMetadata(t *testing.T, ctx context.Context, conn *db.DB, keyManager kms.Kms) {
	t.Helper()
	var singleton bool
	var installationID string
	var cookieKey, oAuthKey, notificationKey []byte
	err := conn.QueryRow(ctx, `SELECT
		"singleton",
		"installation_id",
		"kms_encrypted_cookie_key",
		"kms_encrypted_oauth_key",
		"kms_encrypted_notification_key"
		FROM "metadata"`).Scan(&singleton, &installationID, &cookieKey, &oAuthKey, &notificationKey)
	if err != nil {
		t.Fatal(err)
	}
	if !singleton {
		t.Fatal("expected metadata.singleton to be true")
	}
	if installationID != "install-123" {
		t.Fatalf("unexpected installation_id: %s", installationID)
	}
	cookiePlain, err := keyManager.DecryptDataKey(ctx, cookieKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(cookiePlain) != 64 {
		t.Fatalf("expected 64-byte cookie key, got %d", len(cookiePlain))
	}
	clear(cookiePlain)
	for name, encrypted := range map[string][]byte{
		"oauth":        oAuthKey,
		"notification": notificationKey,
	} {
		plain, err := keyManager.DecryptDataKey(ctx, encrypted)
		if err != nil {
			t.Fatal(err)
		}
		if len(plain) != 32 {
			t.Fatalf("expected 32-byte %s key, got %d", name, len(plain))
		}
		clear(plain)
	}
}
