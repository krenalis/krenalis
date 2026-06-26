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
	oneActivePipelineRunIndex = "pipelines_one_active_run_idx"
	oneLivePipelineRunIndex   = "pipelines_one_live_run_idx"
)
const pipelineRunsFunctionIndex = "pipelines_runs_function_idx"

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

	err = database.Transaction(ctx, func(tx *db.Tx) error {
		// core: prevent concurrent runs for the same pipeline.
		// https://github.com/krenalis/krenalis/pull/2275
		if _, err := tx.Exec(ctx, `DROP INDEX IF EXISTS `+oneActivePipelineRunIndex); err != nil {
			return err
		}
		// core: rename active pipeline run index to live
		// https://github.com/krenalis/krenalis/pull/2308
		if _, err := tx.Exec(ctx, `DROP INDEX IF EXISTS `+oneLivePipelineRunIndex); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `CREATE UNIQUE INDEX `+oneLivePipelineRunIndex+`
				ON pipelines_runs (pipeline)
				WHERE end_time IS NULL`); err != nil {
			if db.IsUniqueViolation(err) {
				err = fmt.Errorf("cannot create %s: multiple live runs exist for the same pipeline; try it later", oneLivePipelineRunIndex)
			}
			return err
		}
		// core: fix pipeline run function index.
		// https://github.com/krenalis/krenalis/pull/2276
		if _, err := tx.Exec(ctx, `DROP INDEX IF EXISTS `+pipelineRunsFunctionIndex); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `CREATE INDEX `+pipelineRunsFunctionIndex+`
			ON pipelines_runs (function)
			WHERE function != '' AND end_time IS NULL`); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	slog.Info("PostgreSQL database upgraded successfully")

	return nil
}
