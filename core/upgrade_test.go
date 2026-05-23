// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/dummy"
	dbpkg "github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/initdb"
	statepkg "github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/test/testimages"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
	"github.com/krenalis/krenalis/tools/types"
	_ "github.com/krenalis/krenalis/warehouses/postgresql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestUpgradeDBRenumbersInternalIDs verifies the coordinated ID migration.
func TestUpgradeDBRenumbersInternalIDs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	internalDB, internalPool := newUpgradeTestPostgreSQL(t, ctx, "test_krenalis")
	warehouseDB, warehousePool := newUpgradeTestPostgreSQL(t, ctx, "test_warehouse")
	kmsURI := upgradeTestKMS()
	keyManager, err := kms.New(ctx, kmsURI)
	if err != nil {
		t.Fatal(err)
	}

	db, err := dbpkg.Open(&dbpkg.Options{
		Host:     internalDB.Host,
		Port:     internalDB.Port,
		Username: internalDB.Username,
		Password: internalDB.Password,
		Database: internalDB.Database,
		Schema:   internalDB.Schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	if err := initdb.InitIfEmpty(ctx, db, keyManager, false); err != nil {
		t.Fatal(err)
	}

	warehouseSettings, warehouseSettingsKey := encryptUpgradeTestWarehouseSettings(t, ctx, db, keyManager, warehouseDB)
	profileSchema, err := json.Marshal(types.Object([]types.Property{
		{Name: "email", Type: types.String().WithMaxLength(254)},
	}))
	if err != nil {
		t.Fatal(err)
	}
	emptyConnectorSettings, connectorSettingsKey := encryptUpgradeTestSettings(t, ctx, db, keyManager, json.Value(`{}`))

	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO workspaces (
			id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			kms_encrypted_warehouse_settings_key, warehouse_mcp_settings,
			kms_encrypted_warehouse_mcp_settings_key, profile_schema, pipelines_to_purge
		) VALUES (10, 1, 'Legacy workspace', 'PostgreSQL', 'Normal', $1, $2, NULL, $2, $3, ARRAY[1000,3000])`,
		warehouseSettings, warehouseSettingsKey, profileSchema)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO access_keys (id, organization, workspace, name, type, hmac, hint, created_at)
		VALUES (50, 1, 10, 'Legacy API key', 'API', $1, 'legacy', now())`,
		bytes.Repeat([]byte{7}, 32))
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO connections (
			id, workspace, name, connector, role, linked_connections, settings, kms_encrypted_settings_key
		) VALUES
			(100, 10, 'Legacy source', 'dummy', 'Source', ARRAY[300], $1, $2),
			(300, 10, 'Legacy destination', 'dummy', 'Destination', '{}', $1, $2)`,
		emptyConnectorSettings, connectorSettingsKey)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO pipelines (
			id, connection, target, event_type, name, transformation_language,
			in_schema, out_schema, matching_in, matching_out, update_on_duplicates, table_key
		) VALUES
			(1000, 100, 'User', 'legacy_user', 'Legacy users', 'JavaScript', $1, $1, 'email', 'email', false, 'email'),
			(3000, 300, 'Event', 'legacy_event', 'Legacy events', 'JavaScript', $1, $1, 'email', 'email', false, 'email')`,
		profileSchema)
	for _, table := range []string{"workspaces", "access_keys", "connections", "pipelines"} {
		mustExecUpgradeTestSQL(t, ctx, internalPool, `ALTER TABLE `+table+` ALTER COLUMN id DROP IDENTITY IF EXISTS`)
	}

	mustExecUpgradeTestSQL(t, ctx, warehousePool, `CREATE TABLE krenalis_events (connection_id integer)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `CREATE TABLE krenalis_identities (_connection integer, _pipeline integer)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `CREATE TABLE krenalis_destination_profiles (_pipeline integer)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_events (connection_id) VALUES (100)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_identities (_connection, _pipeline) VALUES (300, 1000)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_destination_profiles (_pipeline) VALUES (3000)`)

	err = UpgradeDB(ctx, &Config{
		DB: DBConfig{
			Host:     internalDB.Host,
			Port:     internalDB.Port,
			Username: internalDB.Username,
			Password: internalDB.Password,
			Database: internalDB.Database,
			Schema:   internalDB.Schema,
		},
		KMS: kmsURI,
	})
	if err != nil {
		t.Fatal(err)
	}

	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT id FROM workspaces WHERE name = 'Legacy workspace'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT id FROM access_keys WHERE name = 'Legacy API key'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT workspace FROM access_keys WHERE name = 'Legacy API key'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT id FROM connections WHERE name = 'Legacy source'`)
	assertUpgradeTestInt(t, ctx, internalPool, 2, `SELECT id FROM connections WHERE name = 'Legacy destination'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT workspace FROM connections WHERE name = 'Legacy source'`)
	assertUpgradeTestString(t, ctx, internalPool, "2", `SELECT array_to_string(linked_connections, ',') FROM connections WHERE name = 'Legacy source'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT id FROM pipelines WHERE name = 'Legacy users'`)
	assertUpgradeTestInt(t, ctx, internalPool, 2, `SELECT id FROM pipelines WHERE name = 'Legacy events'`)
	assertUpgradeTestInt(t, ctx, internalPool, 1, `SELECT connection FROM pipelines WHERE name = 'Legacy users'`)
	assertUpgradeTestInt(t, ctx, internalPool, 2, `SELECT connection FROM pipelines WHERE name = 'Legacy events'`)
	assertUpgradeTestString(t, ctx, internalPool, "1,2", `SELECT array_to_string(pipelines_to_purge, ',') FROM workspaces WHERE id = 1`)
	assertUpgradeTestSequence(t, ctx, internalPool, "workspaces", 2)
	assertUpgradeTestSequence(t, ctx, internalPool, "access_keys", 2)
	assertUpgradeTestSequence(t, ctx, internalPool, "connections", 3)
	assertUpgradeTestSequence(t, ctx, internalPool, "pipelines", 3)
	assertUpgradeTestBool(t, ctx, internalPool, false, `
		SELECT EXISTS (
			SELECT FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema() AND c.relname = 'krenalis_db_upgrade_internal_ids'
		)`)
	assertUpgradeTestBool(t, ctx, internalPool, true, `
		SELECT EXISTS (
			SELECT FROM krenalis_db_upgrades WHERE name = 'internal-id-renumbering-2026-05'
		)`)

	assertUpgradeTestInt(t, ctx, warehousePool, 1, `SELECT connection_id FROM krenalis_events`)
	assertUpgradeTestInt(t, ctx, warehousePool, 2, `SELECT _connection FROM krenalis_identities`)
	assertUpgradeTestInt(t, ctx, warehousePool, 1, `SELECT _pipeline FROM krenalis_identities`)
	assertUpgradeTestInt(t, ctx, warehousePool, 2, `SELECT _pipeline FROM krenalis_destination_profiles`)
	assertUpgradeTestInt(t, ctx, warehousePool, 0, `
		SELECT COUNT(*) FROM (
			SELECT connection_id AS id FROM krenalis_events
			UNION ALL SELECT _connection FROM krenalis_identities
			UNION ALL SELECT _pipeline FROM krenalis_identities
			UNION ALL SELECT _pipeline FROM krenalis_destination_profiles
		) ids WHERE id IN (100, 300, 1000, 3000)`)
	assertUpgradeTestBool(t, ctx, warehousePool, true, `
		SELECT EXISTS (
			SELECT FROM krenalis_internal_migrations
			WHERE id = 'internal-id-renumbering-2026-05' AND workspace = 1
		)`)
}

// newUpgradeTestPostgreSQL starts a PostgreSQL container for migration tests.
func newUpgradeTestPostgreSQL(t *testing.T, ctx context.Context, database string) (DBConfig, *pgxpool.Pool) {
	t.Helper()

	const user = "test_postgres"
	const password = "test_postgres"
	container, err := postgres.Run(ctx,
		testimages.PostgreSQL,
		postgres.WithDatabase(database),
		postgres.WithUsername(user),
		postgres.WithPassword(password),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			t.Error(err)
		}
	})
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}
	port, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatal(err)
	}
	conf := DBConfig{
		Host:     host,
		Port:     int(port.Num()),
		Username: user,
		Password: password,
		Database: database,
		Schema:   "public",
	}
	pool, err := pgxpool.New(ctx, upgradeTestPostgreSQLURL(conf))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return conf, pool
}

// upgradeTestKMS returns a deterministic local KMS URI.
func upgradeTestKMS() string {
	key := bytes.Repeat([]byte{1}, 32)
	return "key:" + base64.RawStdEncoding.EncodeToString(key)
}

// upgradeTestPostgreSQLURL returns the test PostgreSQL connection URL.
func upgradeTestPostgreSQLURL(conf DBConfig) string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(conf.Username, conf.Password),
		Host:   net.JoinHostPort(conf.Host, strconv.Itoa(conf.Port)),
		Path:   "/" + url.PathEscape(conf.Database),
	}
	if conf.Schema != "" {
		u.RawQuery = "search_path=" + url.QueryEscape(conf.Schema)
	}
	return u.String()
}

// encryptUpgradeTestWarehouseSettings encrypts PostgreSQL warehouse settings.
func encryptUpgradeTestWarehouseSettings(t *testing.T, ctx context.Context, db *dbpkg.DB, kms kms.Kms, conf DBConfig) ([]byte, []byte) {
	t.Helper()

	settings, err := json.Marshal(map[string]any{
		"host":     conf.Host,
		"port":     conf.Port,
		"username": conf.Username,
		"password": conf.Password,
		"database": conf.Database,
		"schema":   conf.Schema,
	})
	if err != nil {
		t.Fatal(err)
	}
	return encryptUpgradeTestSettings(t, ctx, db, kms, settings)
}

// encryptUpgradeTestSettings encrypts settings through the real state cipher.
func encryptUpgradeTestSettings(t *testing.T, ctx context.Context, db *dbpkg.DB, kms kms.Kms, settings json.Value) ([]byte, []byte) {
	t.Helper()

	st, err := statepkg.New(ctx, db, kms, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	encrypted, key, err := st.EncryptSettings(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	return encrypted, key
}

// mustExecUpgradeTestSQL executes SQL or fails the test.
func mustExecUpgradeTestSQL(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatalf("cannot execute SQL %q: %s", query, err)
	}
}

// assertUpgradeTestInt verifies a single integer result.
func assertUpgradeTestInt(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want int, query string, args ...any) {
	t.Helper()

	var got int
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("cannot scan integer from %q: %s", query, err)
	}
	if got != want {
		t.Fatalf("unexpected integer from %q: got %d, want %d", query, got, want)
	}
}

// assertUpgradeTestString verifies a single string result.
func assertUpgradeTestString(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want string, query string, args ...any) {
	t.Helper()

	var got string
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("cannot scan string from %q: %s", query, err)
	}
	if got != want {
		t.Fatalf("unexpected string from %q: got %q, want %q", query, got, want)
	}
}

// assertUpgradeTestBool verifies a single boolean result.
func assertUpgradeTestBool(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want bool, query string, args ...any) {
	t.Helper()

	var got bool
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("cannot scan bool from %q: %s", query, err)
	}
	if got != want {
		t.Fatalf("unexpected bool from %q: got %t, want %t", query, got, want)
	}
}

// assertUpgradeTestSequence verifies the next generated table ID.
func assertUpgradeTestSequence(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string, wantNext int) {
	t.Helper()

	query := fmt.Sprintf(`SELECT nextval(pg_get_serial_sequence('%s', 'id'))`, table)
	assertUpgradeTestInt(t, ctx, pool, wantNext, query)
}
