// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/krenalis/krenalis/core/internal/db"
	dbpkg "github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/base58"
	"github.com/krenalis/krenalis/tools/json"
)

const (
	// Base58IDsMigrationName identifies the public resource ID migration.
	Base58IDsMigrationName   = "base58-resource-identifiers-2026-05"
	base58IDsMigrationTable  = "krenalis_db_upgrade_base58_ids"
	completedMigrationsTable = "krenalis_db_upgrades"
	pipelineIDMapQuery       = "SELECT id FROM pipelines UNION SELECT unnest(pipelines_to_purge) AS id FROM workspaces ORDER BY id"
)

var errBase58IDsAlreadyApplied = errors.New("Base58 ID migration is already applied")

// Base58IDsWarehouseMigration is a pending warehouse ID migration.
type Base58IDsWarehouseMigration struct {
	Workspace      string
	ConnectionMap  map[int]string
	PipelineMap    map[int]string
	PipelineRunMap map[int]string
}

// MigrateBase58IDs prepares the public Base58 ID migration for the Krenalis
// PostgreSQL database. Warehouse migrations are intentionally performed later,
// so this function can be safely resumed.
func MigrateBase58IDs(ctx context.Context, database *db.DB) error {
	completed, err := base58IDsMigrationCompleted(ctx, database)
	if err != nil {
		return err
	}
	if completed {
		slog.Info("PostgreSQL database is already up to date")
		return nil
	}
	inProgress, err := Base58IDsMigrationInProgress(ctx, database)
	if err != nil {
		return err
	}
	if inProgress {
		slog.Info("PostgreSQL database migration already prepared; resuming warehouse migrations")
		return nil
	}
	err = database.Transaction(ctx, func(tx *dbpkg.Tx) error {
		if err := validateBase58IDsMigrationPreconditions(ctx, tx); err != nil {
			return err
		}
		if err := prepareBase58IDMaps(ctx, tx); err != nil {
			return err
		}
		for _, query := range base58IDMigrationQueries() {
			if _, err := tx.Exec(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		if err == errBase58IDsAlreadyApplied {
			slog.Info("PostgreSQL database is already up to date")
			return nil
		}
		return err
	}
	slog.Info("PostgreSQL database Base58 ID migration prepared successfully")
	return nil
}

// Base58IDsMigrationInProgress reports whether the migration must resume.
func Base58IDsMigrationInProgress(ctx context.Context, database *db.DB) (bool, error) {
	return database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'r'`, base58IDsMigrationTable)
}

// PendingBase58IDsWarehouseMigrations returns pending warehouse migrations.
func PendingBase58IDsWarehouseMigrations(ctx context.Context, database *db.DB) ([]Base58IDsWarehouseMigration, error) {
	var migrations []Base58IDsWarehouseMigration
	err := database.QueryScan(ctx, `SELECT workspace, connection_map, pipeline_map, pipeline_run_map FROM `+base58IDsMigrationTable+`
		WHERE NOT warehouse_done ORDER BY workspace`, func(rows *db.Rows) error {
		for rows.Next() {
			var migration Base58IDsWarehouseMigration
			var connectionMap, pipelineMap, pipelineRunMap []byte
			if err := rows.Scan(&migration.Workspace, &connectionMap, &pipelineMap, &pipelineRunMap); err != nil {
				return err
			}
			migration.ConnectionMap = map[int]string{}
			migration.PipelineMap = map[int]string{}
			migration.PipelineRunMap = map[int]string{}
			if err := unmarshalBase58IDMap(connectionMap, migration.ConnectionMap); err != nil {
				return err
			}
			if err := unmarshalBase58IDMap(pipelineMap, migration.PipelineMap); err != nil {
				return err
			}
			if err := unmarshalBase58IDMap(pipelineRunMap, migration.PipelineRunMap); err != nil {
				return err
			}
			migrations = append(migrations, migration)
		}
		return nil
	})
	return migrations, err
}

// MarkBase58IDsWarehouseMigrationDone marks a workspace warehouse as migrated.
func MarkBase58IDsWarehouseMigrationDone(ctx context.Context, database *db.DB, workspace string) error {
	_, err := database.Exec(ctx, `UPDATE `+base58IDsMigrationTable+` SET warehouse_done = true WHERE workspace = $1`, workspace)
	return err
}

// CompleteBase58IDsMigration removes state after all warehouses are migrated.
func CompleteBase58IDsMigration(ctx context.Context, database *db.DB) error {
	exists, err := database.QueryExists(ctx, `SELECT FROM `+base58IDsMigrationTable+` WHERE NOT warehouse_done`)
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
	_, err = database.Exec(ctx, `INSERT INTO `+completedMigrationsTable+` (name) VALUES ($1) ON CONFLICT (name) DO NOTHING`, Base58IDsMigrationName)
	if err != nil {
		return err
	}
	_, err = database.Exec(ctx, `DROP TABLE `+base58IDsMigrationTable)
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database migration completed")
	return nil
}

func base58IDsMigrationCompleted(ctx context.Context, database *db.DB) (bool, error) {
	exists, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'r'`, completedMigrationsTable)
	if err != nil || !exists {
		return false, err
	}
	return database.QueryExists(ctx, `SELECT FROM `+completedMigrationsTable+` WHERE name = $1`, Base58IDsMigrationName)
}

func validateBase58IDsMigrationPreconditions(ctx context.Context, tx *dbpkg.Tx) error {
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
	if typ == "character varying(12)" {
		return errBase58IDsAlreadyApplied
	}
	if typ != "uuid" {
		return fmt.Errorf("organizations.id has type %s instead of uuid", typ)
	}
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

func prepareBase58IDMaps(ctx context.Context, tx *dbpkg.Tx) error {
	queries := []string{
		`LOCK TABLE organizations, members, workspaces, access_keys, connections, pipelines, pipelines_runs,
			accounts, event_write_keys, primary_sources, pipelines_errors, pipelines_metrics, notifications IN ACCESS EXCLUSIVE MODE`,
		`CREATE TEMP TABLE organization_id_map (old_id uuid PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE member_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE workspace_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE access_key_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE connection_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE pipeline_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
		`CREATE TEMP TABLE pipeline_run_id_map (old_id integer PRIMARY KEY, new_id varchar(12) NOT NULL UNIQUE) ON COMMIT DROP`,
	}
	for _, query := range queries {
		if _, err := tx.Exec(ctx, query); err != nil {
			return err
		}
	}
	used := map[string]struct{}{}
	if err := insertUUIDBase58Map(ctx, tx, "organization_id_map", "SELECT id FROM organizations ORDER BY id", used); err != nil {
		return err
	}
	for _, table := range []struct {
		mapTable string
		query    string
	}{
		{"member_id_map", "SELECT id FROM members ORDER BY id"},
		{"workspace_id_map", "SELECT id FROM workspaces ORDER BY id"},
		{"access_key_id_map", "SELECT id FROM access_keys ORDER BY id"},
		{"connection_id_map", "SELECT id FROM connections ORDER BY id"},
		{"pipeline_id_map", pipelineIDMapQuery},
		{"pipeline_run_id_map", "SELECT id FROM pipelines_runs ORDER BY id"},
	} {
		if err := insertIntBase58Map(ctx, tx, table.mapTable, table.query, used); err != nil {
			return err
		}
	}
	return nil
}

func insertUUIDBase58Map(ctx context.Context, tx *dbpkg.Tx, table, query string, used map[string]struct{}) error {
	return tx.QueryScan(ctx, query, func(rows *db.Rows) error {
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "INSERT INTO "+table+" (old_id, new_id) VALUES ($1, $2)", id, generateMigrationID(used)); err != nil {
				return err
			}
		}
		return nil
	})
}

func insertIntBase58Map(ctx context.Context, tx *dbpkg.Tx, table, query string, used map[string]struct{}) error {
	return tx.QueryScan(ctx, query, func(rows *db.Rows) error {
		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, "INSERT INTO "+table+" (old_id, new_id) VALUES ($1, $2)", id, generateMigrationID(used)); err != nil {
				return err
			}
		}
		return nil
	})
}

func generateMigrationID(used map[string]struct{}) string {
	for {
		id := base58.Generate(12)
		if _, ok := used[id]; ok {
			continue
		}
		used[id] = struct{}{}
		return id
	}
}

func base58IDMigrationQueries() []string {
	return []string{
		`CREATE TABLE ` + base58IDsMigrationTable + ` (
			workspace varchar(12) PRIMARY KEY,
			connection_map jsonb NOT NULL,
			pipeline_map jsonb NOT NULL,
			pipeline_run_map jsonb NOT NULL,
			warehouse_done boolean NOT NULL DEFAULT false
		)`,
		`INSERT INTO ` + base58IDsMigrationTable + ` (workspace, connection_map, pipeline_map, pipeline_run_map)
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
				), '{}'::jsonb) ||
				COALESCE((
					SELECT jsonb_object_agg(pm.old_id::text, pm.new_id)
					FROM workspaces w
					CROSS JOIN LATERAL unnest(w.pipelines_to_purge) AS purge(id)
					INNER JOIN pipeline_id_map pm ON pm.old_id = purge.id
					WHERE w.id = wm.old_id
				), '{}'::jsonb),
				COALESCE((
					SELECT jsonb_object_agg(rm.old_id::text, rm.new_id)
					FROM pipeline_run_id_map rm
					INNER JOIN pipelines_runs r ON r.id = rm.old_id
					INNER JOIN pipelines p ON p.id = r.pipeline
					INNER JOIN connections c ON c.id = p.connection
					WHERE c.workspace = wm.old_id
				), '{}'::jsonb)
			FROM workspace_id_map wm`,

		`ALTER TABLE members ADD COLUMN id_new varchar(12)`,
		`UPDATE members SET id_new = member_id_map.new_id FROM member_id_map WHERE members.id = member_id_map.old_id`,
		`ALTER TABLE members ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE members ADD COLUMN organization_new varchar(12)`,
		`UPDATE members SET organization_new = organization_id_map.new_id FROM organization_id_map WHERE members.organization = organization_id_map.old_id`,
		`ALTER TABLE members ALTER COLUMN organization_new SET NOT NULL`,

		`ALTER TABLE organizations ADD COLUMN id_new varchar(12)`,
		`UPDATE organizations SET id_new = organization_id_map.new_id FROM organization_id_map WHERE organizations.id = organization_id_map.old_id`,
		`ALTER TABLE organizations ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE organizations ADD COLUMN global_id uuid`,
		`UPDATE organizations SET global_id = id`,
		`ALTER TABLE organizations ALTER COLUMN global_id SET NOT NULL`,

		`ALTER TABLE workspaces ADD COLUMN id_new varchar(12)`,
		`UPDATE workspaces SET id_new = workspace_id_map.new_id FROM workspace_id_map WHERE workspaces.id = workspace_id_map.old_id`,
		`ALTER TABLE workspaces ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE workspaces ADD COLUMN organization_new varchar(12)`,
		`UPDATE workspaces SET organization_new = organization_id_map.new_id FROM organization_id_map WHERE workspaces.organization = organization_id_map.old_id`,
		`ALTER TABLE workspaces ALTER COLUMN organization_new SET NOT NULL`,
		`ALTER TABLE workspaces ADD COLUMN pipelines_to_purge_new varchar(12)[]`,
		`UPDATE workspaces SET pipelines_to_purge_new = COALESCE((
			SELECT array_agg(pipeline_id_map.new_id ORDER BY ord)
			FROM unnest(workspaces.pipelines_to_purge) WITH ORDINALITY AS purge(id, ord)
			INNER JOIN pipeline_id_map ON pipeline_id_map.old_id = purge.id
		), '{}'::varchar(12)[])`,
		`ALTER TABLE workspaces ALTER COLUMN pipelines_to_purge_new SET NOT NULL`,
		`UPDATE workspaces SET alter_profile_schema_primary_sources = COALESCE((
			SELECT jsonb_object_agg(sources.key, connection_id_map.new_id)
			FROM jsonb_each_text(workspaces.alter_profile_schema_primary_sources) AS sources(key, value)
			INNER JOIN connection_id_map ON connection_id_map.old_id::text = sources.value
		), '{}'::jsonb) WHERE alter_profile_schema_primary_sources IS NOT NULL`,

		`ALTER TABLE access_keys ADD COLUMN id_new varchar(12)`,
		`UPDATE access_keys SET id_new = access_key_id_map.new_id FROM access_key_id_map WHERE access_keys.id = access_key_id_map.old_id`,
		`ALTER TABLE access_keys ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE access_keys ADD COLUMN organization_new varchar(12)`,
		`UPDATE access_keys SET organization_new = organization_id_map.new_id FROM organization_id_map WHERE access_keys.organization = organization_id_map.old_id`,
		`ALTER TABLE access_keys ALTER COLUMN organization_new SET NOT NULL`,
		`ALTER TABLE access_keys ADD COLUMN workspace_new varchar(12)`,
		`UPDATE access_keys SET workspace_new = workspace_id_map.new_id FROM workspace_id_map WHERE access_keys.workspace = workspace_id_map.old_id`,

		`ALTER TABLE connections ADD COLUMN id_new varchar(12)`,
		`UPDATE connections SET id_new = connection_id_map.new_id FROM connection_id_map WHERE connections.id = connection_id_map.old_id`,
		`ALTER TABLE connections ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE connections ADD COLUMN workspace_new varchar(12)`,
		`UPDATE connections SET workspace_new = workspace_id_map.new_id FROM workspace_id_map WHERE connections.workspace = workspace_id_map.old_id`,
		`ALTER TABLE connections ALTER COLUMN workspace_new SET NOT NULL`,
		`ALTER TABLE connections ADD COLUMN linked_connections_new varchar(12)[]`,
		`UPDATE connections SET linked_connections_new = COALESCE((
			SELECT array_agg(connection_id_map.new_id ORDER BY ord)
			FROM unnest(connections.linked_connections) WITH ORDINALITY AS linked(id, ord)
			INNER JOIN connection_id_map ON connection_id_map.old_id = linked.id
		), '{}'::varchar(12)[]) WHERE linked_connections IS NOT NULL`,

		`ALTER TABLE accounts ADD COLUMN workspace_new varchar(12)`,
		`UPDATE accounts SET workspace_new = workspace_id_map.new_id FROM workspace_id_map WHERE accounts.workspace = workspace_id_map.old_id`,
		`ALTER TABLE accounts ALTER COLUMN workspace_new SET NOT NULL`,

		`ALTER TABLE pipelines ADD COLUMN id_new varchar(12)`,
		`UPDATE pipelines SET id_new = pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines.id = pipeline_id_map.old_id`,
		`ALTER TABLE pipelines ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE pipelines ADD COLUMN connection_new varchar(12)`,
		`UPDATE pipelines SET connection_new = connection_id_map.new_id FROM connection_id_map WHERE pipelines.connection = connection_id_map.old_id`,
		`ALTER TABLE pipelines ALTER COLUMN connection_new SET NOT NULL`,

		`ALTER TABLE pipelines_runs ADD COLUMN id_new varchar(12)`,
		`UPDATE pipelines_runs SET id_new = pipeline_run_id_map.new_id FROM pipeline_run_id_map WHERE pipelines_runs.id = pipeline_run_id_map.old_id`,
		`ALTER TABLE pipelines_runs ALTER COLUMN id_new SET NOT NULL`,
		`ALTER TABLE pipelines_runs ADD COLUMN pipeline_new varchar(12)`,
		`UPDATE pipelines_runs SET pipeline_new = pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_runs.pipeline = pipeline_id_map.old_id`,
		`ALTER TABLE pipelines_runs ALTER COLUMN pipeline_new SET NOT NULL`,

		`ALTER TABLE event_write_keys ADD COLUMN connection_new varchar(12)`,
		`UPDATE event_write_keys SET connection_new = connection_id_map.new_id FROM connection_id_map WHERE event_write_keys.connection = connection_id_map.old_id`,
		`ALTER TABLE event_write_keys ALTER COLUMN connection_new SET NOT NULL`,
		`ALTER TABLE primary_sources ADD COLUMN source_new varchar(12)`,
		`UPDATE primary_sources SET source_new = connection_id_map.new_id FROM connection_id_map WHERE primary_sources.source = connection_id_map.old_id`,
		`ALTER TABLE primary_sources ALTER COLUMN source_new SET NOT NULL`,
		`ALTER TABLE pipelines_errors ADD COLUMN pipeline_new varchar(12)`,
		`UPDATE pipelines_errors SET pipeline_new = pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_errors.pipeline = pipeline_id_map.old_id`,
		`ALTER TABLE pipelines_errors ALTER COLUMN pipeline_new SET NOT NULL`,
		`ALTER TABLE pipelines_metrics ADD COLUMN pipeline_new varchar(12)`,
		`UPDATE pipelines_metrics SET pipeline_new = pipeline_id_map.new_id FROM pipeline_id_map WHERE pipelines_metrics.pipeline = pipeline_id_map.old_id`,
		`ALTER TABLE pipelines_metrics ALTER COLUMN pipeline_new SET NOT NULL`,

		`ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS access_keys_workspace_fkey`,
		`ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS access_keys_organization_fkey`,
		`ALTER TABLE connections DROP CONSTRAINT IF EXISTS connections_workspace_fkey`,
		`ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_workspace_fkey`,
		`ALTER TABLE pipelines DROP CONSTRAINT IF EXISTS pipelines_connection_fkey`,
		`ALTER TABLE event_write_keys DROP CONSTRAINT IF EXISTS event_write_keys_connection_fkey`,
		`ALTER TABLE primary_sources DROP CONSTRAINT IF EXISTS primary_sources_source_fkey`,
		`ALTER TABLE pipelines_runs DROP CONSTRAINT IF EXISTS pipelines_runs_pipeline_fkey`,
		`ALTER TABLE pipelines_errors DROP CONSTRAINT IF EXISTS pipelines_errors_pipeline_fkey`,
		`ALTER TABLE pipelines_metrics DROP CONSTRAINT IF EXISTS pipelines_metrics_pipeline_fkey`,
		`ALTER TABLE members DROP CONSTRAINT IF EXISTS members_organization_fkey`,
		`ALTER TABLE members DROP CONSTRAINT IF EXISTS members_organization_email_key`,
		`ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS workspaces_organization_fkey`,

		`ALTER TABLE event_write_keys DROP CONSTRAINT IF EXISTS event_write_keys_pkey`,
		`ALTER TABLE primary_sources DROP CONSTRAINT IF EXISTS primary_sources_pkey`,
		`ALTER TABLE pipelines_metrics DROP CONSTRAINT IF EXISTS pipelines_metrics_pkey`,
		`ALTER TABLE pipelines_runs DROP CONSTRAINT IF EXISTS pipelines_runs_pkey`,
		`ALTER TABLE pipelines DROP CONSTRAINT IF EXISTS pipelines_pkey`,
		`ALTER TABLE connections DROP CONSTRAINT IF EXISTS connections_pkey`,
		`ALTER TABLE access_keys DROP CONSTRAINT IF EXISTS access_keys_pkey`,
		`ALTER TABLE workspaces DROP CONSTRAINT IF EXISTS workspaces_pkey`,
		`ALTER TABLE members DROP CONSTRAINT IF EXISTS members_pkey`,
		`ALTER TABLE organizations DROP CONSTRAINT IF EXISTS organizations_pkey`,

		`ALTER TABLE organizations DROP COLUMN id`,
		`ALTER TABLE organizations RENAME COLUMN id_new TO id`,
		`ALTER TABLE members DROP COLUMN id`,
		`ALTER TABLE members RENAME COLUMN id_new TO id`,
		`ALTER TABLE members DROP COLUMN organization`,
		`ALTER TABLE members RENAME COLUMN organization_new TO organization`,
		`ALTER TABLE workspaces DROP COLUMN id`,
		`ALTER TABLE workspaces RENAME COLUMN id_new TO id`,
		`ALTER TABLE workspaces DROP COLUMN organization`,
		`ALTER TABLE workspaces RENAME COLUMN organization_new TO organization`,
		`ALTER TABLE workspaces DROP COLUMN pipelines_to_purge`,
		`ALTER TABLE workspaces RENAME COLUMN pipelines_to_purge_new TO pipelines_to_purge`,
		`ALTER TABLE workspaces ALTER COLUMN pipelines_to_purge SET DEFAULT '{}'::varchar(12)[]`,
		`ALTER TABLE access_keys DROP COLUMN id`,
		`ALTER TABLE access_keys RENAME COLUMN id_new TO id`,
		`ALTER TABLE access_keys DROP COLUMN organization`,
		`ALTER TABLE access_keys RENAME COLUMN organization_new TO organization`,
		`ALTER TABLE access_keys DROP COLUMN workspace`,
		`ALTER TABLE access_keys RENAME COLUMN workspace_new TO workspace`,
		`ALTER TABLE connections DROP COLUMN id`,
		`ALTER TABLE connections RENAME COLUMN id_new TO id`,
		`ALTER TABLE connections DROP COLUMN workspace`,
		`ALTER TABLE connections RENAME COLUMN workspace_new TO workspace`,
		`ALTER TABLE connections DROP COLUMN linked_connections`,
		`ALTER TABLE connections RENAME COLUMN linked_connections_new TO linked_connections`,
		`ALTER TABLE accounts DROP COLUMN workspace`,
		`ALTER TABLE accounts RENAME COLUMN workspace_new TO workspace`,
		`ALTER TABLE pipelines DROP COLUMN id`,
		`ALTER TABLE pipelines RENAME COLUMN id_new TO id`,
		`ALTER TABLE pipelines DROP COLUMN connection`,
		`ALTER TABLE pipelines RENAME COLUMN connection_new TO connection`,
		`ALTER TABLE pipelines_runs DROP COLUMN id`,
		`ALTER TABLE pipelines_runs RENAME COLUMN id_new TO id`,
		`ALTER TABLE pipelines_runs DROP COLUMN pipeline`,
		`ALTER TABLE pipelines_runs RENAME COLUMN pipeline_new TO pipeline`,
		`ALTER TABLE event_write_keys DROP COLUMN connection`,
		`ALTER TABLE event_write_keys RENAME COLUMN connection_new TO connection`,
		`ALTER TABLE primary_sources DROP COLUMN source`,
		`ALTER TABLE primary_sources RENAME COLUMN source_new TO source`,
		`ALTER TABLE pipelines_errors DROP COLUMN pipeline`,
		`ALTER TABLE pipelines_errors RENAME COLUMN pipeline_new TO pipeline`,
		`ALTER TABLE pipelines_metrics DROP COLUMN pipeline`,
		`ALTER TABLE pipelines_metrics RENAME COLUMN pipeline_new TO pipeline`,

		addBase58Check("organizations", "id"),
		addBase58Check("members", "id"),
		addBase58Check("workspaces", "id"),
		addBase58Check("access_keys", "id"),
		addBase58Check("connections", "id"),
		addBase58Check("pipelines", "id"),
		addBase58Check("pipelines_runs", "id"),

		`ALTER TABLE organizations ADD CONSTRAINT organizations_pkey PRIMARY KEY (id)`,
		`ALTER TABLE organizations ADD CONSTRAINT organizations_global_id_key UNIQUE (global_id)`,
		`ALTER TABLE members ADD CONSTRAINT members_pkey PRIMARY KEY (id)`,
		`ALTER TABLE workspaces ADD CONSTRAINT workspaces_pkey PRIMARY KEY (id)`,
		`ALTER TABLE access_keys ADD CONSTRAINT access_keys_pkey PRIMARY KEY (id)`,
		`ALTER TABLE connections ADD CONSTRAINT connections_pkey PRIMARY KEY (id)`,
		`ALTER TABLE pipelines ADD CONSTRAINT pipelines_pkey PRIMARY KEY (id)`,
		`ALTER TABLE pipelines_runs ADD CONSTRAINT pipelines_runs_pkey PRIMARY KEY (id)`,
		`ALTER TABLE event_write_keys ADD CONSTRAINT event_write_keys_pkey PRIMARY KEY (connection, key)`,
		`ALTER TABLE primary_sources ADD CONSTRAINT primary_sources_pkey PRIMARY KEY (source, path)`,
		`ALTER TABLE pipelines_metrics ADD CONSTRAINT pipelines_metrics_pkey PRIMARY KEY (pipeline, timeslot)`,

		`ALTER TABLE members ADD CONSTRAINT members_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`,
		`ALTER TABLE members ADD CONSTRAINT members_organization_email_key UNIQUE (organization, email)`,
		`ALTER TABLE workspaces ADD CONSTRAINT workspaces_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`,
		`ALTER TABLE access_keys ADD CONSTRAINT access_keys_organization_fkey FOREIGN KEY (organization) REFERENCES organizations ON DELETE CASCADE`,
		`ALTER TABLE access_keys ADD CONSTRAINT access_keys_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
		`ALTER TABLE connections ADD CONSTRAINT connections_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
		`ALTER TABLE accounts ADD CONSTRAINT accounts_workspace_fkey FOREIGN KEY (workspace) REFERENCES workspaces ON DELETE CASCADE`,
		`ALTER TABLE pipelines ADD CONSTRAINT pipelines_connection_fkey FOREIGN KEY (connection) REFERENCES connections ON DELETE CASCADE`,
		`ALTER TABLE event_write_keys ADD CONSTRAINT event_write_keys_connection_fkey FOREIGN KEY (connection) REFERENCES connections ON DELETE CASCADE`,
		`ALTER TABLE primary_sources ADD CONSTRAINT primary_sources_source_fkey FOREIGN KEY (source) REFERENCES connections ON DELETE CASCADE`,
		`ALTER TABLE pipelines_runs ADD CONSTRAINT pipelines_runs_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,
		`ALTER TABLE pipelines_errors ADD CONSTRAINT pipelines_errors_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,
		`ALTER TABLE pipelines_metrics ADD CONSTRAINT pipelines_metrics_pipeline_fkey FOREIGN KEY (pipeline) REFERENCES pipelines ON DELETE CASCADE`,
		`CREATE INDEX ON pipelines_errors (pipeline)`,
		`CREATE INDEX ON pipelines_metrics (pipeline)`,
		`TRUNCATE notifications`,
	}
}

func addBase58Check(table, column string) string {
	name := table + "_" + column + "_base58_check"
	return fmt.Sprintf(`ALTER TABLE %s ADD CONSTRAINT %s CHECK (%s ~ '^[1-9A-HJ-NP-Za-km-z]{12}$')`, table, name, column)
}

func unmarshalBase58IDMap(data []byte, out map[int]string) error {
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		id, err := strconv.Atoi(k)
		if err != nil {
			return err
		}
		out[id] = v
	}
	return nil
}
