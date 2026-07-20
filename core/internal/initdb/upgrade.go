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

const (
	workspacesOrganizationIndex = "workspaces_organization_idx"
	connectionsWorkspaceIndex   = "connections_workspace_idx"
)

const consentPurposesTable = `
	CREATE TABLE IF NOT EXISTS consent_purposes (
		id varchar(12) NOT NULL CHECK (id ~ '^[1-9A-HJ-NP-Za-km-z]{12}$'),
		workspace varchar(12) NOT NULL REFERENCES workspaces ON DELETE CASCADE,
		name varchar(100) NOT NULL,
		code varchar(100) NOT NULL,
		UNIQUE (workspace, code),
		PRIMARY KEY (id)
	)`

const organizationConnectorReferencesView = `
	CREATE OR REPLACE VIEW organization_connector_references AS
	SELECT
		ws.organization,
		c.connector,
		'connection' AS resource_type,
		c.id AS resource
	FROM connections c
	JOIN workspaces ws ON ws.id = c.workspace
	UNION ALL
	SELECT
		ws.organization,
		p.format AS connector,
		'pipeline' AS resource_type,
		p.id AS resource
	FROM pipelines p
	JOIN connections c ON c.id = p.connection
	JOIN workspaces ws ON ws.id = c.workspace
	WHERE p.format IS NOT NULL`

// Upgrade applies idempotent updates to an existing Krenalis PostgreSQL
// database.
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
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS members_limit integer NOT NULL DEFAULT 10000 CHECK (members_limit BETWEEN 1 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS access_keys_limit integer NOT NULL DEFAULT 1000 CHECK (access_keys_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS workspaces_limit integer NOT NULL DEFAULT 1000 CHECK (workspaces_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connectors_limit integer NOT NULL DEFAULT 1000 CHECK (connectors_limit BETWEEN 0 AND 1000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS connections_limit integer NOT NULL DEFAULT 10000 CHECK (connections_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ADD COLUMN IF NOT EXISTS pipelines_limit integer NOT NULL DEFAULT 10000 CHECK (pipelines_limit BETWEEN 0 AND 10000)`,
			`ALTER TABLE organizations ALTER COLUMN members_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN access_keys_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN workspaces_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connectors_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN connections_limit DROP DEFAULT`,
			`ALTER TABLE organizations ALTER COLUMN pipelines_limit DROP DEFAULT`,
			`CREATE INDEX IF NOT EXISTS ` + workspacesOrganizationIndex + ` ON workspaces (organization)`,
			`CREATE INDEX IF NOT EXISTS ` + connectionsWorkspaceIndex + ` ON connections (workspace)`,
			organizationConnectorReferencesView,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'InviteMember' AFTER 'EndPipelineRun'`,
			consentPurposesTable,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'AddConsentPurpose'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'DeleteConsentPurpose'`,
			`ALTER TYPE notification_name ADD VALUE IF NOT EXISTS 'UpdateConsentPurpose'`,
			`ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS required_consents varchar(12)[] NOT NULL DEFAULT '{}'`,
			`ALTER TABLE pipelines ADD COLUMN IF NOT EXISTS required_consents_logical varchar(3) NOT NULL DEFAULT '' CHECK (required_consents_logical IN ('', 'and', 'or'))`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS passed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_metrics ADD COLUMN IF NOT EXISTS failed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN passed_6 DROP DEFAULT`,
			`ALTER TABLE pipelines_metrics ALTER COLUMN failed_6 DROP DEFAULT`,
			`ALTER TABLE pipelines_runs ADD COLUMN IF NOT EXISTS passed_6 integer NOT NULL DEFAULT 0`,
			`ALTER TABLE pipelines_runs ADD COLUMN IF NOT EXISTS failed_6 integer NOT NULL DEFAULT 0`,
		}
		for _, query := range queries {
			if _, err := tx.Exec(ctx, query); err != nil {
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
