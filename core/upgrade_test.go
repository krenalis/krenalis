// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/krenalis/krenalis/connectors/dummy"
	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/initdb"
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

func TestUpgradeDBMigratesBase58IDs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	internalDB, internalPool := newUpgradeTestPostgreSQL(t, ctx, "test_krenalis")
	warehouseDB, warehousePool := newUpgradeTestPostgreSQL(t, ctx, "test_warehouse")

	kmsURI := upgradeTestKMS()
	keyManager, err := kms.New(ctx, kmsURI)
	if err != nil {
		t.Fatal(err)
	}
	initializeUpgradeTestLegacyDB(t, ctx, internalPool, keyManager)
	initializeUpgradeTestLegacyWarehouse(t, ctx, warehousePool)

	warehouseSettings, warehouseSettingsKey := encryptUpgradeTestWarehouseSettings(t, ctx, keyManager, warehouseDB)
	emptyConnectorSettings, connectorSettingsKey := encryptUpgradeTestSettings(t, ctx, keyManager, json.Value(`{}`))
	profileSchema, err := json.Marshal(types.Object([]types.Property{
		{Name: "email", Type: types.String().WithMaxLength(254)},
	}))
	if err != nil {
		t.Fatal(err)
	}

	const organizationUUID = "11111111-1111-1111-1111-111111111111"
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO organizations (id, name)
		VALUES ($1, 'Legacy organization')`, organizationUUID)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO members (id, organization, name, email, created_at)
		VALUES (7, $1, 'Legacy member', 'legacy@example.com', now())`, organizationUUID)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO workspaces (
			id, organization, name, warehouse_name, warehouse_mode, warehouse_settings,
			warehouse_mcp_settings, kms_encrypted_warehouse_settings_key,
			kms_encrypted_warehouse_mcp_settings_key, alter_profile_schema_primary_sources,
			profile_schema, pipelines_to_purge
		) VALUES (10, $1, 'Legacy workspace', 'PostgreSQL', 'Normal', $2, $2, $3, $3, '{"primary":"100"}', $4, ARRAY[1000,9000])`,
		organizationUUID, warehouseSettings, warehouseSettingsKey, profileSchema)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO access_keys (id, organization, workspace, name, type, hmac, hint, created_at)
		VALUES (50, $1, 10, 'Legacy API key', 'API', $2, 'legacy', now())`,
		organizationUUID, bytes.Repeat([]byte{7}, 32))
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
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO pipelines_runs (id, pipeline, start_time, ping_time, end_time)
		VALUES (7000, 1000, now(), now(), now())`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO event_write_keys (connection, key, created_at)
		VALUES (100, '12345678901234567890123456789012', now())`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO accounts (id, workspace, connector, code, access_token, refresh_token, expires_in)
		VALUES (80, 10, 'dummy', 'legacy-account', '', '', now())`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO primary_sources (source, path)
		VALUES (100, 'email')`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO pipelines_errors (pipeline, timeslot, step, count, message)
		VALUES (1000, 1, 0, 2, 'legacy error')`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO pipelines_metrics (pipeline, timeslot, passed_0, passed_1, passed_2, passed_3, passed_4, passed_5,
			failed_0, failed_1, failed_2, failed_3, failed_4, failed_5)
		VALUES (1000, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)`)
	mustExecUpgradeTestSQL(t, ctx, internalPool, `
		INSERT INTO notifications (id, name, payload)
		VALUES (1, 'CreateWorkspace', '{"legacy":true}')`)

	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_events (connection_id) VALUES (100)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_identities (_connection, _pipeline, _run) VALUES (300, 1000, 7000)`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_destination_profiles (_pipeline, _external_id, _out_matching_value) VALUES (3000, 'event-1', 'legacy')`)
	mustExecUpgradeTestSQL(t, ctx, warehousePool, `INSERT INTO krenalis_destination_profiles (_pipeline, _external_id, _out_matching_value) VALUES (9000, 'purged-1', 'legacy')`)

	err = UpgradeDB(ctx, &Config{
		DB:  internalDB,
		KMS: kmsURI,
	})
	if err != nil {
		t.Fatal(err)
	}

	organizationID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM organizations WHERE name = 'Legacy organization'`)
	memberID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM members WHERE name = 'Legacy member'`)
	workspaceID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM workspaces WHERE name = 'Legacy workspace'`)
	accessKeyID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM access_keys WHERE name = 'Legacy API key'`)
	sourceConnectionID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM connections WHERE name = 'Legacy source'`)
	destinationConnectionID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM connections WHERE name = 'Legacy destination'`)
	userPipelineID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM pipelines WHERE name = 'Legacy users'`)
	eventPipelineID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM pipelines WHERE name = 'Legacy events'`)
	pipelineRunID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT id FROM pipelines_runs`)
	purgedPipelineID := assertUpgradeTestBase58ID(t, ctx, internalPool, `SELECT pipelines_to_purge[2] FROM workspaces WHERE name = 'Legacy workspace'`)
	assertUpgradeTestString(t, ctx, internalPool, organizationUUID, `SELECT global_id::text FROM organizations WHERE id = $1`, organizationID)
	assertUpgradeTestString(t, ctx, internalPool, organizationID, `SELECT organization FROM members WHERE id = $1`, memberID)
	assertUpgradeTestString(t, ctx, internalPool, organizationID, `SELECT organization FROM workspaces WHERE id = $1`, workspaceID)
	assertUpgradeTestString(t, ctx, internalPool, accessKeyID, `SELECT id FROM access_keys WHERE workspace = $1`, workspaceID)
	assertUpgradeTestString(t, ctx, internalPool, organizationID, `SELECT organization FROM access_keys WHERE id = $1`, accessKeyID)
	assertUpgradeTestString(t, ctx, internalPool, workspaceID, `SELECT workspace FROM accounts WHERE code = 'legacy-account'`)
	assertUpgradeTestString(t, ctx, internalPool, workspaceID, `SELECT workspace FROM connections WHERE id = $1`, sourceConnectionID)
	assertUpgradeTestString(t, ctx, internalPool, destinationConnectionID, `SELECT linked_connections[1] FROM connections WHERE id = $1`, sourceConnectionID)
	assertUpgradeTestString(t, ctx, internalPool, sourceConnectionID, `SELECT connection FROM pipelines WHERE id = $1`, userPipelineID)
	assertUpgradeTestString(t, ctx, internalPool, destinationConnectionID, `SELECT connection FROM pipelines WHERE id = $1`, eventPipelineID)
	assertUpgradeTestString(t, ctx, internalPool, userPipelineID, `SELECT pipeline FROM pipelines_runs WHERE id = $1`, pipelineRunID)
	assertUpgradeTestString(t, ctx, internalPool, sourceConnectionID, `SELECT connection FROM event_write_keys`)
	assertUpgradeTestString(t, ctx, internalPool, sourceConnectionID, `SELECT source FROM primary_sources WHERE path = 'email'`)
	assertUpgradeTestString(t, ctx, internalPool, userPipelineID, `SELECT pipeline FROM pipelines_errors`)
	assertUpgradeTestString(t, ctx, internalPool, userPipelineID, `SELECT pipeline FROM pipelines_metrics`)
	assertUpgradeTestString(t, ctx, internalPool, userPipelineID+","+purgedPipelineID, `SELECT array_to_string(pipelines_to_purge, ',') FROM workspaces WHERE id = $1`, workspaceID)
	assertUpgradeTestString(t, ctx, internalPool, sourceConnectionID, `SELECT alter_profile_schema_primary_sources->>'primary' FROM workspaces WHERE id = $1`, workspaceID)
	assertUpgradeTestInt(t, ctx, internalPool, 0, `SELECT COUNT(*) FROM notifications`)
	assertUpgradeTestBool(t, ctx, internalPool, false, `
		SELECT EXISTS (
			SELECT FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
			WHERE n.nspname = current_schema() AND c.relname = 'krenalis_db_upgrade_base58_ids'
		)`)
	assertUpgradeTestBool(t, ctx, internalPool, true, `SELECT EXISTS (SELECT FROM krenalis_db_upgrades WHERE name = $1)`, initdb.Base58IDsMigrationName)
	assertUpgradeTestString(t, ctx, internalPool, "character varying(12)[]", `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'workspaces' AND a.attname = 'pipelines_to_purge' AND NOT a.attisdropped`)
	assertUpgradeTestString(t, ctx, internalPool, "'{}'::character varying(12)[]", `
		SELECT pg_get_expr(d.adbin, d.adrelid)
		FROM pg_attrdef d
		JOIN pg_class c ON c.oid = d.adrelid
		JOIN pg_attribute a ON a.attrelid = c.oid AND a.attnum = d.adnum
		WHERE c.relname = 'workspaces' AND a.attname = 'pipelines_to_purge'`)

	assertUpgradeTestString(t, ctx, warehousePool, sourceConnectionID, `SELECT connection_id FROM krenalis_events`)
	assertUpgradeTestString(t, ctx, warehousePool, destinationConnectionID, `SELECT _connection FROM krenalis_identities`)
	assertUpgradeTestString(t, ctx, warehousePool, userPipelineID, `SELECT _pipeline FROM krenalis_identities`)
	assertUpgradeTestString(t, ctx, warehousePool, pipelineRunID, `SELECT _run FROM krenalis_identities`)
	assertUpgradeTestString(t, ctx, warehousePool, eventPipelineID+","+purgedPipelineID, `SELECT string_agg(_pipeline, ',' ORDER BY _external_id) FROM krenalis_destination_profiles`)
	assertUpgradeTestString(t, ctx, warehousePool, "text", `
		SELECT format_type(a.atttypid, a.atttypmod)
		FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		WHERE c.relname = 'krenalis_events' AND a.attname = 'connection_id' AND NOT a.attisdropped`)
	assertUpgradeTestBool(t, ctx, warehousePool, true, `
		SELECT EXISTS (
			SELECT FROM krenalis_internal_migrations
			WHERE id = $1 AND workspace = $2
		)`, initdb.Base58IDsMigrationName, workspaceID)

	if err := UpgradeDB(ctx, &Config{DB: internalDB, KMS: kmsURI}); err != nil {
		t.Fatalf("second UpgradeDB failed: %s", err)
	}
	assertUpgradeTestString(t, ctx, internalPool, workspaceID, `SELECT id FROM workspaces WHERE name = 'Legacy workspace'`)
	assertUpgradeTestString(t, ctx, warehousePool, sourceConnectionID, `SELECT connection_id FROM krenalis_events`)
}

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

func initializeUpgradeTestLegacyDB(t *testing.T, ctx context.Context, pool *pgxpool.Pool, keyManager kms.Kms) {
	t.Helper()

	for _, statement := range strings.Split(upgradeTestLegacySchema, ";\n") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		mustExecUpgradeTestSQL(t, ctx, pool, statement)
	}
	cookieKey, err := keyManager.GenerateDataKeyWithoutPlaintext(ctx, 64)
	if err != nil {
		t.Fatal(err)
	}
	oauthKey, err := keyManager.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		t.Fatal(err)
	}
	notificationKey, err := keyManager.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		t.Fatal(err)
	}
	apiKeyPepper, err := keyManager.GenerateDataKeyWithoutPlaintext(ctx, 32)
	if err != nil {
		t.Fatal(err)
	}
	mustExecUpgradeTestSQL(t, ctx, pool, `
		UPDATE metadata SET
			kms_encrypted_cookie_key = $1,
			kms_encrypted_oauth_key = $2,
			kms_encrypted_notification_key = $3,
			kms_encrypted_api_key_pepper = $4`,
		cookieKey, oauthKey, notificationKey, apiKeyPepper)
}

func initializeUpgradeTestLegacyWarehouse(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	mustExecUpgradeTestSQL(t, ctx, pool, `CREATE TABLE krenalis_events (connection_id integer NOT NULL)`)
	mustExecUpgradeTestSQL(t, ctx, pool, `CREATE TABLE krenalis_identities (_connection integer NOT NULL, _pipeline integer NOT NULL, _run integer)`)
	mustExecUpgradeTestSQL(t, ctx, pool, `CREATE TABLE krenalis_destination_profiles (
		_pipeline integer NOT NULL,
		_external_id text NOT NULL,
		_out_matching_value text NOT NULL,
		PRIMARY KEY (_pipeline, _external_id)
	)`)
}

func upgradeTestKMS() string {
	key := bytes.Repeat([]byte{1}, 32)
	return "key:" + base64.RawStdEncoding.EncodeToString(key)
}

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

func encryptUpgradeTestWarehouseSettings(t *testing.T, ctx context.Context, keyManager kms.Kms, conf DBConfig) ([]byte, []byte) {
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
	return encryptUpgradeTestSettings(t, ctx, keyManager, settings)
}

func encryptUpgradeTestSettings(t *testing.T, ctx context.Context, keyManager kms.Kms, settings json.Value) ([]byte, []byte) {
	t.Helper()

	c := cipher.New(keyManager)
	defer c.Close()
	encrypted, key, err := c.Encrypt(ctx, settings)
	if err != nil {
		t.Fatal(err)
	}
	return encrypted, key
}

func mustExecUpgradeTestSQL(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()

	if _, err := pool.Exec(ctx, query, args...); err != nil {
		t.Fatalf("cannot execute SQL %q: %s", query, err)
	}
}

func assertUpgradeTestBase58ID(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()

	id := assertUpgradeTestStringResult(t, ctx, pool, query, args...)
	if !IsValidID(id) {
		t.Fatalf("unexpected ID from %q: got %q", query, id)
	}
	return id
}

func assertUpgradeTestString(t *testing.T, ctx context.Context, pool *pgxpool.Pool, want string, query string, args ...any) {
	t.Helper()

	got := assertUpgradeTestStringResult(t, ctx, pool, query, args...)
	if got != want {
		t.Fatalf("unexpected string from %q: got %q, want %q", query, got, want)
	}
}

func assertUpgradeTestStringResult(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, args ...any) string {
	t.Helper()

	var got string
	if err := pool.QueryRow(ctx, query, args...).Scan(&got); err != nil {
		t.Fatalf("cannot scan string from %q: %s", query, err)
	}
	return got
}

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

const upgradeTestLegacySchema = `
CREATE COLLATION case_insensitive (provider = icu, locale = 'und-u-ks-level2', deterministic = false);
CREATE TYPE pipeline_target AS ENUM ('Event', 'User', 'Group');
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE TABLE organizations (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    name varchar(45) NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);
CREATE TYPE avatar_mime_type AS ENUM ('image/jpeg', 'image/png');
CREATE TYPE avatar AS (
    image bytea,
    mime_type avatar_mime_type
);
CREATE TABLE members (
    id integer GENERATED BY DEFAULT AS IDENTITY,
    organization uuid NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(45) NOT NULL DEFAULT '',
    avatar avatar,
    email varchar(120) NOT NULL COLLATE case_insensitive,
    password varchar(72) NOT NULL DEFAULT '',
    invitation_token varchar(44) NOT NULL DEFAULT '',
    reset_password_token varchar(44) NOT NULL DEFAULT '',
    reset_password_token_created_at timestamp,
    created_at timestamp,
    UNIQUE (organization, email),
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX invitation_token_index ON members (invitation_token) WHERE invitation_token <> '';
CREATE UNIQUE INDEX reset_password_token_index ON members (reset_password_token) WHERE reset_password_token <> '';
CREATE TYPE warehouse_mode AS ENUM ('Normal', 'Inspection', 'Maintenance');
CREATE TABLE workspaces (
    id integer NOT NULL,
    organization uuid NOT NULL REFERENCES organizations ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    warehouse_name varchar NOT NULL,
    warehouse_mode warehouse_mode NOT NULL,
    warehouse_settings bytea NOT NULL,
    warehouse_mcp_settings bytea DEFAULT NULL,
    kms_encrypted_warehouse_settings_key bytea NOT NULL,
    kms_encrypted_warehouse_mcp_settings_key bytea NOT NULL,
    alter_profile_schema_id uuid,
    alter_profile_schema_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    alter_profile_schema_primary_sources jsonb,
    alter_profile_schema_operations jsonb,
    alter_profile_schema_start_time timestamp(3),
    alter_profile_schema_end_time timestamp(3),
    alter_profile_schema_error varchar,
    profile_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    resolve_identities_on_batch_import boolean NOT NULL DEFAULT false,
    identifiers text[] NOT NULL DEFAULT '{}',
    ir_id uuid,
    ir_start_time timestamp(3),
    ir_end_time timestamp(3),
    ui_profile_image varchar(100) NOT NULL DEFAULT '',
    ui_profile_first_name varchar(100) NOT NULL DEFAULT '',
    ui_profile_last_name varchar(100) NOT NULL DEFAULT '',
    ui_profile_extra varchar(100) NOT NULL DEFAULT '',
    pipelines_to_purge int[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (id)
);
CREATE TYPE access_key_type AS ENUM ('API', 'MCP');
CREATE TABLE access_keys (
    id integer NOT NULL,
    organization uuid NOT NULL REFERENCES organizations ON DELETE CASCADE,
    workspace integer REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL,
    type access_key_type NOT NULL,
    hmac bytea NOT NULL UNIQUE,
    hint varchar(13) NOT NULL,
    created_at timestamp(0) NOT NULL,
    PRIMARY KEY (id)
);
CREATE TYPE role AS ENUM ('Source', 'Destination');
CREATE TYPE health AS ENUM ('Healthy', 'NoRecentData', 'RecentError');
CREATE TYPE compression AS ENUM ('', 'Zip', 'Gzip', 'Snappy');
CREATE TYPE strategy AS ENUM ('Conversion', 'Fusion', 'Isolation', 'Preservation');
CREATE TYPE sending_mode AS ENUM ('Client', 'Server', 'ClientAndServer');
CREATE TABLE connections (
    id integer NOT NULL,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    name varchar(100) NOT NULL DEFAULT '',
    connector varchar,
    role role NOT NULL,
    account integer NOT NULL DEFAULT 0,
    strategy strategy,
    sending_mode sending_mode,
    linked_connections integer[],
    settings bytea,
    kms_encrypted_settings_key bytea NOT NULL,
    health health NOT NULL DEFAULT 'Healthy',
    PRIMARY KEY (id)
);
CREATE TYPE export_mode AS ENUM ('', 'CreateOnly', 'UpdateOnly', 'CreateOrUpdate');
CREATE TYPE transformation_language AS ENUM ('JavaScript', 'Python');
CREATE TABLE pipelines (
    id integer GENERATED BY DEFAULT AS IDENTITY,
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    target pipeline_target NOT NULL,
    event_type varchar(100) NOT NULL,
    name varchar(60) NOT NULL DEFAULT '',
    enabled boolean NOT NULL DEFAULT FALSE,
    schedule_start smallint NOT NULL DEFAULT 0 CHECK (schedule_start >= 0 AND schedule_start < 1440),
    schedule_period smallint NOT NULL DEFAULT 0 CHECK(schedule_period IN (0, 5, 15, 30, 60, 120, 180, 360, 480, 720, 1440)),
    in_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    out_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
    filter jsonb,
    transformation_mapping jsonb,
    transformation_id varchar(200) NOT NULL DEFAULT '',
    transformation_version varchar(128) NOT NULL DEFAULT '',
    transformation_language transformation_language NOT NULL,
    transformation_source text NOT NULL DEFAULT '',
    transformation_preserve_json boolean NOT NULL DEFAULT false,
    transformation_in_paths varchar[],
    transformation_out_paths varchar[],
    query text NOT NULL DEFAULT '',
    format varchar,
    path varchar(1024) NOT NULL DEFAULT '',
    sheet varchar(31) NOT NULL DEFAULT '',
    compression compression NOT NULL DEFAULT '',
    order_by varchar(1024) NOT NULL DEFAULT '',
    format_settings jsonb,
    export_mode export_mode NOT NULL DEFAULT '',
    matching_in text NOT NULL,
    matching_out text NOT NULL,
    update_on_duplicates boolean NOT NULL,
    table_name varchar(1024) NOT NULL DEFAULT '',
    table_key text NOT NULL,
    user_id_column varchar(1024) NOT NULL DEFAULT '',
    updated_at_column varchar(1024) NOT NULL DEFAULT '',
    updated_at_format varchar(64) NOT NULL DEFAULT '',
    incremental boolean NOT NULL DEFAULT FALSE,
    cursor timestamp NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    health health NOT NULL DEFAULT 'Healthy',
    properties_to_unset varchar[],
    PRIMARY KEY (id)
);
CREATE UNIQUE INDEX pipelines_transformation_id_idx ON pipelines (transformation_id) WHERE transformation_id <> '';
CREATE TABLE pipelines_runs (
    id integer GENERATED BY DEFAULT AS IDENTITY,
    pipeline integer NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    function varchar(200) NOT NULL DEFAULT '',
    node uuid,
    incremental boolean NOT NULL DEFAULT FALSE,
    cursor timestamp NOT NULL DEFAULT '0001-01-01 00:00:00+00',
    start_time timestamp NOT NULL,
    ping_time timestamp NOT NULL,
    end_time timestamp,
    passed_0 integer NOT NULL DEFAULT 0,
    passed_1 integer NOT NULL DEFAULT 0,
    passed_2 integer NOT NULL DEFAULT 0,
    passed_3 integer NOT NULL DEFAULT 0,
    passed_4 integer NOT NULL DEFAULT 0,
    passed_5 integer NOT NULL DEFAULT 0,
    failed_0 integer NOT NULL DEFAULT 0,
    failed_1 integer NOT NULL DEFAULT 0,
    failed_2 integer NOT NULL DEFAULT 0,
    failed_3 integer NOT NULL DEFAULT 0,
    failed_4 integer NOT NULL DEFAULT 0,
    failed_5 integer NOT NULL DEFAULT 0,
    error varchar NOT NULL DEFAULT '',
    PRIMARY KEY (id)
);
CREATE INDEX pipelines_runs_function_idx ON pipelines_runs (function) WHERE function != '' AND end_time IS NOT NULL;
CREATE TABLE pipelines_errors (
    pipeline integer NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    timeslot integer NOT NULL,
    step smallint NOT NULL,
    count integer NOT NULL,
    message varchar NOT NULL
);
CREATE INDEX ON pipelines_errors (pipeline);
CREATE INDEX ON pipelines_errors (timeslot);
CREATE INDEX ON pipelines_errors (step);
CREATE TABLE pipelines_metrics (
    pipeline integer NOT NULL REFERENCES pipelines ON DELETE CASCADE,
    timeslot integer NOT NULL,
    passed_0 integer NOT NULL,
    passed_1 integer NOT NULL,
    passed_2 integer NOT NULL,
    passed_3 integer NOT NULL,
    passed_4 integer NOT NULL,
    passed_5 integer NOT NULL,
    failed_0 integer NOT NULL,
    failed_1 integer NOT NULL,
    failed_2 integer NOT NULL,
    failed_3 integer NOT NULL,
    failed_4 integer NOT NULL,
    failed_5 integer NOT NULL,
    PRIMARY KEY (pipeline, timeslot)
);
CREATE INDEX ON pipelines_metrics (pipeline);
CREATE INDEX ON pipelines_metrics (timeslot);
CREATE TABLE discontinued_functions (
    id varchar(200) NOT NULL,
    discontinued_at timestamp(0) NOT NULL,
    PRIMARY KEY (id)
);
CREATE TABLE election (
    number integer NOT NULL,
    leader uuid NOT NULL,
    date timestamp NOT NULL,
    PRIMARY KEY (number)
);
INSERT INTO election (number, leader, date) VALUES (1, '00000000-0000-0000-0000-000000000000', '2023-01-01 00:00:00.000000');
CREATE TABLE event_write_keys (
    connection integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    key char(32) NOT NULL,
    created_at timestamp NOT NULL,
    PRIMARY KEY (connection, key)
);
CREATE TABLE primary_sources (
    source integer NOT NULL REFERENCES connections ON DELETE CASCADE,
    path varchar NOT NULL,
    PRIMARY KEY (source, path)
);
CREATE TABLE accounts (
    id integer GENERATED BY DEFAULT AS IDENTITY,
    workspace integer NOT NULL REFERENCES workspaces ON DELETE CASCADE,
    connector varchar NOT NULL,
    code varchar(100) NOT NULL,
    access_token varchar(500) NOT NULL DEFAULT '',
    refresh_token varchar(500) NOT NULL DEFAULT '',
    expires_in timestamp(0),
    PRIMARY KEY (id)
);
CREATE INDEX ON accounts (connector);
CREATE TYPE notification_name AS ENUM (
    'AcceptInvitation',
    'AddMember',
    'CreateAccessKey',
    'CreateConnection',
    'CreateEventWriteKey',
    'CreateOrganization',
    'CreatePipeline',
    'CreateWorkspace',
    'DeleteAccessKey',
    'DeleteConnection',
    'DeleteEventWriteKey',
    'DeleteMember',
    'DeleteOrganization',
    'DeletePipeline',
    'DeleteWorkspace',
    'EndAlterProfileSchema',
    'EndIdentityResolution',
    'EndPipelineRun',
    'LinkConnection',
    'PurgePipelines',
    'RenameConnection',
    'RenameWorkspace',
    'RunPipeline',
    'SetAccount',
    'SetConnectionSettings',
    'SetPipelineFormatSettings',
    'SetPipelineSchedulePeriod',
    'SetPipelineStatus',
    'StartAlterProfileSchema',
    'StartIdentityResolution',
    'UnlinkConnection',
    'UpdateConnection',
    'UpdateIdentityPropertiesToUnset',
    'UpdateIdentityResolutionSettings',
    'UpdateOrganization',
    'UpdatePipeline',
    'UpdateWarehouse',
    'UpdateWarehouseMode',
    'UpdateWorkspace'
);
CREATE TABLE notifications (
    id bigint NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    name notification_name NOT NULL,
    payload jsonb NOT NULL,
    PRIMARY KEY (id)
);
CREATE TABLE metadata (
    singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
    installation_id text UNIQUE NOT NULL,
    kms_encrypted_cookie_key bytea NOT NULL,
    kms_encrypted_oauth_key bytea NOT NULL,
    kms_encrypted_notification_key bytea NOT NULL,
    kms_encrypted_api_key_pepper bytea NOT NULL
);
INSERT INTO metadata (
    installation_id,
    kms_encrypted_cookie_key,
    kms_encrypted_oauth_key,
    kms_encrypted_notification_key,
    kms_encrypted_api_key_pepper
) VALUES (
    '22222222-2222-2222-2222-222222222222',
    '\x'::bytea,
    '\x'::bytea,
    '\x'::bytea,
    '\x'::bytea
)`
