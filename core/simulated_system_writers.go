// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"fmt"
	"time"

	"github.com/krenalis/krenalis/core/internal/db"
	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/tools/errors"
)

// recoverSimulatedSystemWriters fences expired claims even if their run rows
// disappeared; timeout identifies candidates, not permission to bypass writers.
func (core *Core) recoverSimulatedSystemWriters(ctx context.Context, expiredBefore time.Time) error {

	type claim struct {
		workspace, system, run string
		generation             int64
	}
	var claims []claim
	err := core.db.QueryScan(ctx, `SELECT workspace, id, claimed_run, writer_generation FROM simulated_systems
		WHERE claimed_run IS NOT NULL AND claim_ping_time < $1`, expiredBefore, func(rows *db.Rows) error {
		for rows.Next() {
			var claim claim
			if err := rows.Scan(&claim.workspace, &claim.system, &claim.run, &claim.generation); err != nil {
				return err
			}
			claims = append(claims, claim)
		}
		return rows.Err()
	})
	if err != nil {
		return err
	}

	var recoveryErr error
	for _, claim := range claims {
		claimCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		err = core.releaseSimulatedSystemWriter(claimCtx, claim.workspace, claim.system, claim.run, claim.generation,
			newPipelineError(metrics.ReceiveStep, errors.New("pipeline has been terminated because the node became unresponsive")))
		cancel()
		if err != nil {
			if recoveryErr == nil {
				recoveryErr = err
			}
		}
	}

	return recoveryErr
}

// releaseSimulatedSystemWriter seals warehouse authority before releasing the
// operational claim. A failed or uncertain seal retains the claim for recovery.
func (core *Core) releaseSimulatedSystemWriter(ctx context.Context, workspace, system, run string, generation int64, reason error) error {

	if generation <= 0 || generation%2 != 0 {
		return fmt.Errorf("invalid simulated system writer generation")
	}
	store, ok := core.datastore.Store(workspace)
	if !ok {
		return fmt.Errorf("workspace datastore is unavailable")
	}
	err := store.FenceSimulatedSystem(ctx, system, generation+1)
	if err != nil {
		return err
	}

	if reason != nil {
		if live, ok := core.state.LiveRun(run); ok {
			err = core.endLiveRun(ctx, live, reason)
			if err != nil {
				return err
			}
		}
	}

	_, err = core.db.Exec(ctx, `UPDATE simulated_systems SET claimed_run = NULL, claim_ping_time = NULL
		WHERE id = $1 AND claimed_run = $2 AND writer_generation = $3`, system, run, generation)

	return err
}
