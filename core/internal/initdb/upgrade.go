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

const oneActivePipelineRunIndex = "pipelines_one_active_run_idx"

// Upgrade applies idempotent updates to an existing Krenalis PostgreSQL
// database.
func Upgrade(ctx context.Context, database *db.DB) error {
	initialized, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = 'pipelines_runs'
			AND c.relkind = 'r'`)
	if err != nil {
		return err
	}
	if !initialized {
		return fmt.Errorf("Krenalis's PostgreSQL database has not been initialized")
	}

	indexExists, err := database.QueryExists(ctx, `
		SELECT FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema()
			AND c.relname = $1
			AND c.relkind = 'i'`, oneActivePipelineRunIndex)
	if err != nil {
		return err
	}
	if indexExists {
		slog.Info("PostgreSQL database is already up to date")
		return nil
	}

	duplicateRuns, err := database.QueryExists(ctx, `
		SELECT FROM pipelines_runs
		WHERE end_time IS NULL
		GROUP BY pipeline
		HAVING COUNT(*) > 1`)
	if err != nil {
		return err
	}
	if duplicateRuns {
		return fmt.Errorf("cannot create %s: multiple active runs exist for the same pipeline", oneActivePipelineRunIndex)
	}

	_, err = database.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS `+oneActivePipelineRunIndex+`
		ON pipelines_runs (pipeline)
		WHERE end_time IS NULL`)
	if db.IsUniqueViolation(err) {
		return fmt.Errorf("cannot create %s: multiple active runs exist for the same pipeline", oneActivePipelineRunIndex)
	}
	if err != nil {
		return err
	}
	slog.Info("PostgreSQL database upgraded successfully")
	return nil
}
