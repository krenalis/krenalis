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
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
)

const kmsEncryptedKeys = 4

var errOrganizationIDAlreadyInteger = errors.New("organization ID is already integer")

// RenumberInternalIDsMigrationName identifies the temporary ID renumbering migration.
const RenumberInternalIDsMigrationName = "internal-id-renumbering-2026-05"
const renumberInternalIDsMigrationTable = "krenalis_db_upgrade_internal_ids"
const completedMigrationsTable = "krenalis_db_upgrades"

// RenumberInternalIDsWarehouseMigration is a pending warehouse ID migration.
type RenumberInternalIDsWarehouseMigration struct {
	Workspace     int
	ConnectionMap map[int]int
	PipelineMap   map[int]int
}

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

// MigrateRenumberedInternalIDs prepares the internal ID renumbering migration.
func MigrateRenumberedInternalIDs(ctx context.Context, database *db.DB) error {
	if err := UpgradeOrganizationIDToInt(ctx, database); err != nil {
		return err
	}
	completed, err := renumberInternalIDsMigrationCompleted(ctx, database)
	if err != nil {
		return err
	}
	if completed {
		slog.Info("PostgreSQL database is already up to date")
		return nil
	}
	exists, err := RenumberInternalIDsMigrationInProgress(ctx, database)
	if err != nil {
		return err
	}
	if exists {
		slog.Info("PostgreSQL database migration already prepared; resuming warehouse migrations")
		return nil
	}
	err = database.Transaction(ctx, func(tx *dbpkg.Tx) error {
		if err := validateRenumberInternalIDsMigrationPreconditions(ctx, tx); err != nil {
			return err
		}
		queries := []string{
			`LOCK TABLE workspaces, access_keys, connections, pipelines, accounts, event_write_keys,
				primary_sources, pipelines_runs, pipelines_errors, pipelines_metrics, notifications IN ACCESS EXCLUSIVE MODE`,
			`CREATE TEMP TABLE workspace_id_map ON COMMIT DROP AS
				SELECT id AS old_id, row_number() OVER (ORDER BY id)::integer AS new_id FROM workspaces`,
			`CREATE TEMP TABLE access_key_id_map ON COMMIT DROP AS
				SELECT id AS old_id, row_number() OVER (ORDER BY id)::integer AS new_id FROM access_keys`,
			`CREATE TEMP TABLE connection_id_map ON COMMIT DROP AS
				SELECT id AS old_id, row_number() OVER (ORDER BY id)::integer AS new_id FROM connections`,
			`CREATE TEMP TABLE pipeline_id_map ON COMMIT DROP AS
				SELECT id AS old_id, row_number() OVER (ORDER BY id)::integer AS new_id FROM pipelines`,
			`CREATE TABLE ` + renumberInternalIDsMigrationTable + ` (
				workspace integer PRIMARY KEY,
				connection_map jsonb NOT NULL,
				pipeline_map jsonb NOT NULL,
				warehouse_done boolean NOT NULL DEFAULT false
			)`,
			`INSERT INTO ` + renumberInternalIDsMigrationTable + ` (workspace, connection_map, pipeline_map)
				SELECT wm.new_id,
					COALESCE((
						SELECT jsonb_object_agg(cm.old_id::text, cm.new_id)
						FROM connection_id_map cm
						INNER JOIN connections c ON c.id = cm.old_id
						WHERE c.workspace = wm.old_id
					), '{}'::jsonb),
					COALESCE((
						SELECT jsonb_object_agg(pm.old_id::text, pm.new_id)
						FROM pipeline_id_map pm
						INNER JOIN pipelines p ON p.id = pm.old_id
						INNER JOIN connections c ON c.id = p.connection
						WHERE c.workspace = wm.old_id
					), '{}'::jsonb)
				FROM workspace_id_map wm`,

			`ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS access_keys_workspace_fkey`,
			`ALTER TABLE connections DROP CONSTRAINT IF EXISTS connections_workspace_fkey`,
			`ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_workspace_fkey`,
			`ALTER TABLE pipelines DROP CONSTRAINT IF EXISTS pipelines_connection_fkey`,
			`ALTER TABLE event_write_keys DROP CONSTRAINT IF EXISTS event_write_keys_connection_fkey`,
			`ALTER TABLE primary_sources DROP CONSTRAINT IF EXISTS primary_sources_source_fkey`,
			`ALTER TABLE pipelines_runs DROP CONSTRAINT IF EXISTS pipelines_runs_pipeline_fkey`,
			`ALTER TABLE pipelines_errors DROP CONSTRAINT IF EXISTS pipelines_errors_pipeline_fkey`,
			`ALTER TABLE pipelines_metrics DROP CONSTRAINT IF EXISTS pipelines_metrics_pipeline_fkey`,

			`UPDATE workspaces SET id = -workspace_id_map.new_id FROM workspace_id_map WHERE workspaces.id = workspace_id_map.old_id`,
			`UPDATE access_keys SET id = -access_key_id_map.new_id FROM access_key_id_map WHERE access_keys.id = access_key_id_map.old_id`,
			`UPDATE connections SET id = -connection_id_map.new_id FROM connection_id_map WHERE connections.id = connection_id_map.old_id`,
			`UPDATE pipelines SET id = -pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines.id = pipeline_id_map.old_id`,

			`UPDATE access_keys SET workspace = -workspace_id_map.new_id FROM workspace_id_map WHERE access_keys.workspace = workspace_id_map.old_id`,
			`UPDATE connections SET workspace = -workspace_id_map.new_id FROM workspace_id_map WHERE connections.workspace = workspace_id_map.old_id`,
			`UPDATE accounts SET workspace = -workspace_id_map.new_id FROM workspace_id_map WHERE accounts.workspace = workspace_id_map.old_id`,
			`UPDATE pipelines SET connection = -connection_id_map.new_id FROM connection_id_map WHERE pipelines.connection = connection_id_map.old_id`,
			`UPDATE event_write_keys SET connection = -connection_id_map.new_id FROM connection_id_map WHERE event_write_keys.connection = connection_id_map.old_id`,
			`UPDATE primary_sources SET source = -connection_id_map.new_id FROM connection_id_map WHERE primary_sources.source = connection_id_map.old_id`,
			`UPDATE pipelines_runs SET pipeline = -pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_runs.pipeline = pipeline_id_map.old_id`,
			`UPDATE pipelines_errors SET pipeline = -pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_errors.pipeline = pipeline_id_map.old_id`,
			`UPDATE pipelines_metrics SET pipeline = -pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_metrics.pipeline = pipeline_id_map.old_id`,
			`UPDATE connections SET linked_connections = COALESCE((
				SELECT array_agg(-connection_id_map.new_id ORDER BY ord)
				FROM unnest(connections.linked_connections) WITH ORDINALITY AS linked(id, ord)
				INNER JOIN connection_id_map ON connection_id_map.old_id = linked.id
			), '{}'::integer[]) WHERE linked_connections IS NOT NULL`,
			`UPDATE workspaces SET pipelines_to_purge = COALESCE((
				SELECT array_agg(-pipeline_id_map.new_id ORDER BY ord)
				FROM unnest(workspaces.pipelines_to_purge) WITH ORDINALITY AS purge(id, ord)
				INNER JOIN pipeline_id_map ON pipeline_id_map.old_id = purge.id
			), '{}'::integer[])`,

			`UPDATE workspaces SET id = -id WHERE id < 0`,
			`UPDATE access_keys SET id = -id WHERE id < 0`,
			`UPDATE connections SET id = -id WHERE id < 0`,
			`UPDATE pipelines SET id = -id WHERE id < 0`,
			`UPDATE access_keys SET workspace = -workspace WHERE workspace < 0`,
			`UPDATE connections SET workspace = -workspace WHERE workspace < 0`,
			`UPDATE accounts SET workspace = -workspace WHERE workspace < 0`,
			`UPDATE pipelines SET connection = -connection WHERE connection < 0`,
			`UPDATE event_write_keys SET connection = -connection WHERE connection < 0`,
			`UPDATE primary_sources SET source = -source WHERE source < 0`,
			`UPDATE pipelines_runs SET pipeline = -pipeline WHERE pipeline < 0`,
			`UPDATE pipelines_errors SET pipeline = -pipeline WHERE pipeline < 0`,
			`UPDATE pipelines_metrics SET pipeline = -pipeline WHERE pipeline < 0`,
			`UPDATE connections SET linked_connections = ARRAY(
				SELECT CASE WHEN id < 0 THEN -id ELSE id END FROM unnest(linked_connections) AS id
			) WHERE linked_connections IS NOT NULL`,
			`UPDATE workspaces SET pipelines_to_purge = ARRAY(
				SELECT CASE WHEN id < 0 THEN -id ELSE id END FROM unnest(pipelines_to_purge) AS id
			)`,

			`ALTER TABLE access_keys ADD CONSTRAINT access_keys_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
			`ALTER TABLE connections ADD CONSTRAINT connections_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
			`ALTER TABLE accounts ADD CONSTRAINT accounts_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
			`ALTER TABLE pipelines ADD CONSTRAINT pipelines_connection_fkey FOREIGN KEY (connection) REFERENCES connections ON DELETE CASCADE`,
			`ALTER TABLE event_write_keys ADD CONSTRAINT event_write_keys_connection_fkey FOREIGN KEY (connection) REFERENCES connections ON DELETE CASCADE`,
			`ALTER TABLE primary_sources ADD CONSTRAINT primary_sources_source_fkey FOREIGN KEY (source) REFERENCES connections ON DELETE CASCADE`,
			`ALTER TABLE pipelines_runs ADD CONSTRAINT pipelines_runs_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,
			`ALTER TABLE pipelines_errors ADD CONSTRAINT pipelines_errors_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,
			`ALTER TABLE pipelines_metrics ADD CONSTRAINT pipelines_metrics_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,

			addIdentityIfMissingForRenumberQuery("workspaces"),
			addIdentityIfMissingForRenumberQuery("access_keys"),
			addIdentityIfMissingForRenumberQuery("connections"),
			addIdentityIfMissingForRenumberQuery("pipelines"),
			`SELECT setval(pg_get_serial_sequence('workspaces', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM workspaces`,
			`SELECT setval(pg_get_serial_sequence('access_keys', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM access_keys`,
			`SELECT setval(pg_get_serial_sequence('connections', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM connections`,
			`SELECT setval(pg_get_serial_sequence('pipelines', 'id'), COALESCE(MAX(id), 0) + 1, false) FROM pipelines`,
			`TRUNCATE notifications`,
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database internal ID migration prepared successfully")
	return nil
}

// RenumberInternalIDsMigrationInProgress reports whether the migration must resume.
func RenumberInternalIDsMigrationInProgress(ctx context.Context, database *db.DB) (bool, error) {
	return database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'r'`, renumberInternalIDsMigrationTable)
}

// PendingRenumberInternalIDsWarehouseMigrations returns pending warehouse migrations.
func PendingRenumberInternalIDsWarehouseMigrations(ctx context.Context, database *db.DB) ([]RenumberInternalIDsWarehouseMigration, error) {
	var migrations []RenumberInternalIDsWarehouseMigration
	err := database.QueryScan(ctx, `SELECT workspace, connection_map, pipeline_map FROM `+renumberInternalIDsMigrationTable+`
		WHERE NOT warehouse_done ORDER BY workspace`, func(rows *db.Rows) error {
		for rows.Next() {
			var migration RenumberInternalIDsWarehouseMigration
			var connectionMap, pipelineMap []byte
			if err := rows.Scan(&migration.Workspace, &connectionMap, &pipelineMap); err != nil {
				return err
			}
			migration.ConnectionMap = map[int]int{}
			migration.PipelineMap = map[int]int{}
			if err := unmarshalRenumberIDMap(connectionMap, migration.ConnectionMap); err != nil {
				return err
			}
			if err := unmarshalRenumberIDMap(pipelineMap, migration.PipelineMap); err != nil {
				return err
			}
			migrations = append(migrations, migration)
		}
		return nil
	})
	return migrations, err
}

// MarkRenumberInternalIDsWarehouseMigrationDone marks a workspace warehouse as migrated.
func MarkRenumberInternalIDsWarehouseMigrationDone(ctx context.Context, database *db.DB, workspace int) error {
	_, err := database.Exec(ctx, `UPDATE `+renumberInternalIDsMigrationTable+` SET warehouse_done = true WHERE workspace = $1`, workspace)
	return err
}

// CompleteRenumberInternalIDsMigration removes state after all warehouses are migrated.
func CompleteRenumberInternalIDsMigration(ctx context.Context, database *db.DB) error {
	exists, err := database.QueryExists(ctx, `SELECT FROM `+renumberInternalIDsMigrationTable+` WHERE NOT warehouse_done`)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = database.Exec(ctx, `CREATE TABLE IF NOT EXISTS `+completedMigrationsTable+` (
		name text PRIMARY KEY,
		completed_at timestamp NOT NULL DEFAULT now()
	)`)
	if err != nil {
		return err
	}
	_, err = database.Exec(ctx, `INSERT INTO `+completedMigrationsTable+` (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, RenumberInternalIDsMigrationName)
	if err != nil {
		return err
	}
	_, err = database.Exec(ctx, `DROP TABLE `+renumberInternalIDsMigrationTable)
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database migration completed")
	return nil
}

// renumberInternalIDsMigrationCompleted reports whether the ID migration is complete.
func renumberInternalIDsMigrationCompleted(ctx context.Context, database *db.DB) (bool, error) {
	exists, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'r'`, completedMigrationsTable)
	if err != nil || !exists {
		return false, err
	}
	return database.QueryExists(ctx, `SELECT FROM `+completedMigrationsTable+` WHERE name = $1`, RenumberInternalIDsMigrationName)
}

// validateRenumberInternalIDsMigrationPreconditions checks whether migration can start.
func validateRenumberInternalIDsMigrationPreconditions(ctx context.Context, tx *dbpkg.Tx) error {
	exists, err := tx.QueryExists(ctx, `SELECT FROM workspaces WHERE alter_profile_schema_id IS NOT NULL OR ir_id IS NOT NULL`)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("cannot upgrade database while alter schema or identity resolution operations are in progress")
	}
	exists, err = tx.QueryExists(ctx, `SELECT FROM pipelines_runs WHERE end_time IS NULL`)
	if err != nil {
		return err
	}
	if exists {
		return errors.New("cannot upgrade database while pipeline runs are in progress")
	}
	return nil
}

// addIdentityIfMissingForRenumberQuery returns SQL that adds an identity to table.id if needed.
func addIdentityIfMissingForRenumberQuery(table string) string {
	return fmt.Sprintf(`DO $$
BEGIN
	IF NOT EXISTS (
		SELECT FROM pg_attribute a
		JOIN pg_class c ON c.oid = a.attrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = '%s'
			AND a.attname = 'id'
			AND a.attidentity <> ''
	) THEN
		ALTER TABLE %s ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY;
	END IF;
END$$`, table, table)
}

// unmarshalRenumberIDMap decodes a JSON object with integer string keys.
func unmarshalRenumberIDMap(data []byte, out map[int]int) error {
	var raw map[string]int
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		var id int
		if _, err := fmt.Sscanf(k, "%d", &id); err != nil {
			return err
		}
		out[id] = v
	}
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
