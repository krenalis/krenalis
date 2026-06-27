// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package initdb

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/krenalis/krenalis/core/internal/cipher"
	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/kms"
)

const (
	oneActivePipelineRunIndex = "pipelines_one_active_run_idx"
	oneLivePipelineRunIndex   = "pipelines_one_live_run_idx"
)
const (
	pipelineRunsFunctionIndex       = "pipelines_runs_function_idx"
	legacyDummyOperationDelay       = "2s"
	legacyDummySimulateHTTPDelayKey = "simulateHTTPDelay"
	dummyOperationDelayKey          = "operationDelay"
)

// Upgrade applies idempotent updates to an existing Krenalis PostgreSQL
// database.
func Upgrade(ctx context.Context, database *db.DB, kms kms.Kms) error {

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

	c := cipher.New(kms)
	defer c.Close()
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
		// connectors/dummy: replace simulated HTTP delay with `operationDelay`.
		// https://github.com/krenalis/krenalis/pull/2316
		if err := upgradeDummyOperationDelaySettings(ctx, tx, c); err != nil {
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

// upgradeDummyOperationDelaySettings converts Dummy connections from the legacy
// simulated HTTP delay settings to the operation delay setting.
func upgradeDummyOperationDelaySettings(ctx context.Context, tx *db.Tx, c *cipher.Cipher) error {
	type update struct {
		connection string
		settings   []byte
	}
	updates := []update{}
	err := tx.QueryScan(ctx, `
		SELECT id, settings, kms_encrypted_settings_key
		FROM connections
		WHERE connector = 'dummy' AND settings IS NOT NULL`, func(rows *db.Rows) error {
		for rows.Next() {
			var id string
			var encryptedSettings, settingsKey []byte
			if err := rows.Scan(&id, &encryptedSettings, &settingsKey); err != nil {
				return err
			}
			settings, changed, err := upgradeDummyOperationDelaySetting(ctx, c, encryptedSettings, settingsKey)
			if err != nil {
				return fmt.Errorf("cannot upgrade Dummy settings for connection %s: %s", id, err)
			}
			if !changed {
				continue
			}
			updates = append(updates, update{id, settings})
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, update := range updates {
		if _, err := tx.Exec(ctx, "UPDATE connections SET settings = $1 WHERE id = $2", update.settings, update.connection); err != nil {
			return err
		}
	}
	return nil
}

// upgradeDummyOperationDelaySetting upgrades one Dummy connection setting.
func upgradeDummyOperationDelaySetting(ctx context.Context, c *cipher.Cipher, encryptedSettings, settingsKey []byte) ([]byte, bool, error) {
	settings, err := c.Decrypt(ctx, encryptedSettings, settingsKey)
	if err != nil {
		return nil, false, err
	}
	defer clear(settings)
	var s map[string]any
	if err := json.Unmarshal(settings, &s); err != nil {
		return nil, false, err
	}
	_, hasOperationDelay := s[dummyOperationDelayKey]
	changed := false
	if v, ok := s[legacyDummySimulateHTTPDelayKey]; ok {
		changed = true
		delete(s, legacyDummySimulateHTTPDelayKey)
		if enabled, ok := v.(bool); ok && enabled && !hasOperationDelay {
			s[dummyOperationDelayKey] = legacyDummyOperationDelay
		}
	}
	if !changed {
		return nil, false, nil
	}
	updated, err := json.Marshal(s)
	if err != nil {
		return nil, false, err
	}
	defer clear(updated)
	encrypted, err := c.EncryptWithExistingKey(ctx, updated, settingsKey)
	if err != nil {
		return nil, false, err
	}
	return encrypted, true, nil
}
