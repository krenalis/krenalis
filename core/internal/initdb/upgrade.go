// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/db"
)

// workspacesUpgrade adds the role columns of the profile schema alteration
// after alter_profile_schema_operations and the profile role columns after
// profile_schema, all empty, and drops the ui_profile_* columns. As
// PostgreSQL can only append columns, it recreates every column following
// alter_profile_schema_operations, as declared in schema.sql, and then restores
// its values.
const workspacesUpgrade = `
	DO $$
	BEGIN
		IF EXISTS (
			SELECT FROM pg_attribute
			WHERE attrelid = 'workspaces'::regclass AND attname = 'profile_role_first_name' AND NOT attisdropped
		) THEN
			RETURN;
		END IF;

		LOCK TABLE workspaces IN ACCESS EXCLUSIVE MODE;
		CREATE TEMP TABLE workspaces_upgrade ON COMMIT DROP AS SELECT * FROM workspaces;

		ALTER TABLE workspaces
			DROP COLUMN alter_profile_schema_start_time,
			DROP COLUMN alter_profile_schema_end_time,
			DROP COLUMN alter_profile_schema_error,
			DROP COLUMN profile_schema,
			DROP COLUMN resolve_identities_on_batch_import,
			DROP COLUMN identifiers,
			DROP COLUMN ir_id,
			DROP COLUMN ir_start_time,
			DROP COLUMN ir_end_time,
			DROP COLUMN ui_profile_image,
			DROP COLUMN ui_profile_first_name,
			DROP COLUMN ui_profile_last_name,
			DROP COLUMN ui_profile_extra,
			DROP COLUMN pipelines_to_purge;

		ALTER TABLE workspaces
			ADD COLUMN alter_profile_schema_role_first_name varchar NOT NULL DEFAULT '',
			ADD COLUMN alter_profile_schema_role_last_name varchar NOT NULL DEFAULT '',
			ADD COLUMN alter_profile_schema_role_email varchar NOT NULL DEFAULT '',
			ADD COLUMN alter_profile_schema_role_country varchar NOT NULL DEFAULT '',
			ADD COLUMN alter_profile_schema_role_photo varchar NOT NULL DEFAULT '',
			ADD COLUMN alter_profile_schema_start_time timestamp(3),
			ADD COLUMN alter_profile_schema_end_time timestamp(3),
			ADD COLUMN alter_profile_schema_error varchar,
			ADD COLUMN profile_schema jsonb NOT NULL DEFAULT 'null'::jsonb,
			ADD COLUMN profile_role_first_name varchar NOT NULL DEFAULT '',
			ADD COLUMN profile_role_last_name varchar NOT NULL DEFAULT '',
			ADD COLUMN profile_role_email varchar NOT NULL DEFAULT '',
			ADD COLUMN profile_role_country varchar NOT NULL DEFAULT '',
			ADD COLUMN profile_role_photo varchar NOT NULL DEFAULT '',
			ADD COLUMN resolve_identities_on_batch_import boolean NOT NULL DEFAULT false,
			ADD COLUMN identifiers text[] NOT NULL DEFAULT '{}',
			ADD COLUMN ir_id uuid,
			ADD COLUMN ir_start_time timestamp(3),
			ADD COLUMN ir_end_time timestamp(3),
			ADD COLUMN pipelines_to_purge varchar(12)[] NOT NULL DEFAULT '{}';

		UPDATE workspaces w
		SET alter_profile_schema_start_time = b.alter_profile_schema_start_time,
			alter_profile_schema_end_time = b.alter_profile_schema_end_time,
			alter_profile_schema_error = b.alter_profile_schema_error,
			profile_schema = b.profile_schema,
			resolve_identities_on_batch_import = b.resolve_identities_on_batch_import,
			identifiers = b.identifiers,
			ir_id = b.ir_id,
			ir_start_time = b.ir_start_time,
			ir_end_time = b.ir_end_time,
			pipelines_to_purge = b.pipelines_to_purge
		FROM pg_temp.workspaces_upgrade b
		WHERE w.id = b.id;
	END $$`

// Upgrade upgrades an existing Krenalis PostgreSQL database from the schema of
// release v0.46.0 to the current schema. It is idempotent.
func Upgrade(ctx context.Context, database *db.DB) error {

	initialized, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = 'organizations'
			AND c.relkind = 'r'`)
	if err != nil {
		return err
	}
	if !initialized {
		return fmt.Errorf("Krenalis's PostgreSQL database has not been initialized")
	}

	err = database.Transaction(ctx, func(tx *db.Tx) error {
		queries := []string{
			workspacesUpgrade,
			`CREATE TABLE IF NOT EXISTS identity_resolution_runs (
				id uuid NOT NULL,
				workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
				start_time timestamp(3) NOT NULL,
				end_time timestamp(3),
				error varchar,
				PRIMARY KEY (id)
			)`,
			`CREATE INDEX IF NOT EXISTS identity_resolution_runs_workspace_start_idx
				ON identity_resolution_runs (workspace, start_time DESC, id DESC)`,
			// Record the Identity Resolution in progress, so that it can be
			// finalized after the upgrade.
			`INSERT INTO identity_resolution_runs (id, workspace, start_time)
				SELECT ir_id, id, ir_start_time
				FROM workspaces
				WHERE ir_id IS NOT NULL
				ON CONFLICT (id) DO NOTHING`,
			`CREATE TABLE IF NOT EXISTS identity_resolution_metrics (
				workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
				day date NOT NULL,
				observed_at time without time zone NOT NULL,
				profiles_anonymous bigint NOT NULL,
				profiles_recognized bigint NOT NULL,
				identities_anonymous bigint NOT NULL,
				identities_recognized bigint NOT NULL,
				composition_one bigint NOT NULL,
				composition_two bigint NOT NULL,
				composition_three bigint NOT NULL,
				composition_four_to_ten bigint NOT NULL,
				composition_eleven_to_twenty bigint NOT NULL,
				composition_more_than_twenty bigint NOT NULL,
				PRIMARY KEY (workspace, day)
			)`,
		}
		for _, query := range queries {
			_, err := tx.Exec(ctx, query)
			if err != nil {
				return fmt.Errorf("cannot execute upgrade query %q: %s", query, err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}

	slog.Info("PostgreSQL database upgraded successfully")

	return nil
}
