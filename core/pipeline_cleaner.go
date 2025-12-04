// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
	"github.com/meergo/meergo/tools/backoff"
)

// backoffBase is the base used for the backoff.
const backoffBase = 1000

// functionDeletionInterval defines how often discontinued functions are
// deleted.
const functionDeletionInterval = 10 * time.Minute

// pipelineCleaner represents a pipeline cleaner. It performs the following
// tasks:
//
//   - Purges identities and destination profiles in the data warehouse that
//     are associated with deleted pipelines.
//   - Unsets identity properties in the data warehouse that are no longer
//     transformed.
//   - Deletes discontinued functions from its function provider.
//   - Terminates orphaned pipeline executions.
//
// Pipeline cleaning occurs only when the data warehouse is in Normal mode.
type pipelineCleaner struct {
	core             *Core
	functionProvider transformers.FunctionProvider
	workspaces       sync.Map // map of workspaces indicating if an operation is in progress
	pipelines        sync.Map // map of pipelines indicating if an operation is in progress
	close            struct {
		ctx    context.Context
		cancel context.CancelFunc
		atomic.Bool
		sync.WaitGroup
	}
}

// newPipelineCleaner returns a new instance of the pipeline cleaner. There is
// only one active pipeline cleaner at a time, and it exclusively runs on the
// leader node.
func newPipelineCleaner(core *Core, provider transformers.FunctionProvider) *pipelineCleaner {

	c := &pipelineCleaner{
		core:             core,
		functionProvider: provider,
	}
	c.close.ctx, c.close.cancel = context.WithCancel(context.Background())

	core.state.Freeze()
	core.state.AddListener(c.onDeleteConnection)
	core.state.AddListener(c.onDeletePipeline)
	core.state.AddListener(c.onUpdatePipeline)
	core.state.AddListener(c.onUpdateWarehouse)
	core.state.AddListener(c.onUpdateWarehouseMode)
	var workspaces []int
	for _, ws := range c.core.state.Workspaces() {
		if ws.NumPipelinesToPurge() > 0 {
			workspaces = append(workspaces, ws.ID)
		}
	}
	var pipelines []int
	for _, pipeline := range c.core.state.Pipelines() {
		if properties := pipeline.PropertiesToUnset(); len(properties) > 0 {
			pipelines = append(pipelines, pipeline.ID)
		}
	}
	core.state.Unfreeze()
	for _, ws := range workspaces {
		go c.purgeWorkspace(ws)
	}
	for _, pipeline := range pipelines {
		go c.unsetIdentityProperties(pipeline)
	}

	// Start a goroutine to delete functions that have been discontinued.
	if provider != nil {
		go c.deleteDiscontinuedFunctions()
	}

	// Start a goroutine to terminate orphaned pipeline executions.
	go c.terminateOrphanedPipelineExecutions()

	return c
}

// Close closes the pipeline cleaner, ensuring the completion of all ongoing
// operations. If the context is canceled, it interrupts ongoing operations and
// returns. If p is already closed, it does nothing and returns immediately.
func (c *pipelineCleaner) Close(ctx context.Context) {
	if c.close.Swap(true) {
		return
	}
	// Cancel c.close.ctx if ctx is cancelled.
	stop := context.AfterFunc(ctx, func() { c.close.cancel() })
	defer stop()
	// Waits for the ongoing operations to finish.
	c.close.Wait()
}

// complete calls f, ensuring it completes even if c is closed.
// If c is already closed, it does nothing.
func (c *pipelineCleaner) complete(f func() error) error {
	c.close.Add(1)
	if c.close.Load() {
		c.close.Done()
		return nil
	}
	err := f()
	c.close.Done()
	return err
}

// deleteDiscontinuedFunctions starts the deletion task for function that have
// been discontinued and are no longer in use by any executing pipelines.
// It must be called in its own goroutine.
//
// A function is considered discontinued if:
//
//   - The associated pipeline has been deleted.
//   - The transformation type has changed from function-based to mapping-based.
//   - The function has switched between JavaScript and Python.
func (c *pipelineCleaner) deleteDiscontinuedFunctions() {
	var ids, deleted []string
	var d = 2 * time.Second // initial waiting time.
	timer := time.NewTimer(d)
	for {
		select {
		case <-c.close.ctx.Done():
			return
		case <-timer.C:
		}
		d = functionDeletionInterval
		err := c.complete(func() error {
			// Read the functions. These are the discontinued ones from over ten
			// minutes ago, with no pipelines still using them.
			rows, err := c.core.db.Query(c.close.ctx, "SELECT f.id\n"+
				"FROM discontinued_functions AS f\n"+
				"LEFT JOIN pipelines_executions AS e ON f.id = e.function AND e.end_time IS NULL\n"+
				"WHERE f.discontinued_at < $1 AND e.id IS NULL",
				time.Now().Add(-10*time.Minute))
			if err != nil {
				return err
			}
			ids = ids[:0]
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err != nil {
					break
				}
				ids = append(ids, id)
			}
			if err = rows.Err(); err != nil {
				return err
			}
			if len(ids) == 0 {
				return nil
			}
			// Delete the functions.
			deleted = deleted[:0]
			for _, id := range ids {
				err = c.functionProvider.Delete(c.close.ctx, id)
				if err != nil {
					return fmt.Errorf("deleting function %q: %s", id, err)
				}
				deleted = append(deleted, id)
			}
			if len(deleted) == 0 {
				return nil
			}
			// Delete the functions from the database.
			bo := backoff.New(1000)
			bo.SetCap(functionDeletionInterval)
			for bo.Next(c.close.ctx) {
				_, err = c.core.db.Exec(c.close.ctx,
					fmt.Sprintf("DELETE FROM discontinued_functions WHERE id IN %s", db.Quote(deleted)))
				if err == nil {
					break
				}
			}
			return err
		})
		if err != nil {
			slog.Error("core: an error occurred deleting discontinued functions", "err", err)
		}
		timer.Reset(d)
	}
}

// onDeleteConnection is called when a connection is deleted.
func (c *pipelineCleaner) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if connection.Role != state.Source {
		return
	}
	ws := connection.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	for _, pipeline := range connection.Pipelines() {
		if pipeline.Target == state.TargetUser {
			go c.purgeWorkspace(ws.ID)
			return
		}
	}
}

// onDeletePipeline is called when a pipeline is deleted.
func (c *pipelineCleaner) onDeletePipeline(n state.DeletePipeline) {
	pipeline := n.Pipeline()
	connection := pipeline.Connection()
	if pipeline.Target != state.TargetUser || connection.Role != state.Source {
		return
	}
	ws := connection.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	go c.purgeWorkspace(ws.ID)
}

// onUpdatePipeline is called when a pipeline is updated.
func (c *pipelineCleaner) onUpdatePipeline(n state.UpdatePipeline) {
	if len(n.PropertiesToUnset) == 0 {
		return
	}
	p, _ := c.core.state.Pipeline(n.ID)
	ws := p.Connection().Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	go c.unsetIdentityProperties(n.ID)
}

// onUpdateWarehouse is called when a warehouse is updated.
func (c *pipelineCleaner) onUpdateWarehouse(n state.UpdateWarehouse) {
	if n.Mode != state.Normal {
		return
	}
	if ws, _ := c.core.state.Workspace(n.Workspace); ws.NumPipelinesToPurge() == 0 {
		return
	}
	go c.purgeWorkspace(n.Workspace)
}

// onUpdateWarehouseMode is called when the mode of a warehouse is updated.
func (c *pipelineCleaner) onUpdateWarehouseMode(n state.UpdateWarehouseMode) {
	if n.Mode != state.Normal {
		return
	}
	ws, _ := c.core.state.Workspace(n.Workspace)
	if ws.NumPipelinesToPurge() > 0 {
		go c.purgeWorkspace(n.Workspace)
	}
	for _, connection := range ws.Connections() {
		for _, pipeline := range connection.Pipelines() {
			if paths := pipeline.PropertiesToUnset(); paths != nil {
				go c.unsetIdentityProperties(pipeline.ID)
			}
		}
	}
}

// purgeWorkspace purges identities and destination users associated to
// pipelines deleted from the workspace with the identifier id.
func (c *pipelineCleaner) purgeWorkspace(id int) {

	if _, ok := c.workspaces.Swap(id, true); ok {
		return
	}
	defer c.workspaces.Delete(id)

	bo := backoff.New(backoffBase)
	for bo.Next(c.close.ctx) {

		// Purge the workspace.
		ws, ok := c.core.state.Workspace(id)
		if !ok {
			return
		}
		if ws.Warehouse.Mode != state.Normal {
			return
		}
		pipelines := ws.PipelinesToPurge()
		if len(pipelines) == 0 {
			return
		}
		err := c.complete(func() error {

			store := c.core.datastore.Store(id)
			err := store.PurgePipelines(c.close.ctx, pipelines)
			if err != nil {
				return fmt.Errorf("cannot purge pipelines: %s", err)
			}

			n := state.PurgePipelines{
				Workspace: id,
			}

			// Build the query that updates the pipelines to purge of the workspace.
			var b strings.Builder
			b.WriteString("UPDATE workspaces\nSET pipelines_to_purge = ")
			for range len(pipelines) {
				b.WriteString("array_remove(")
			}
			b.WriteString("pipelines_to_purge")
			for _, pipeline := range pipelines {
				b.WriteByte(',')
				b.WriteString(strconv.Itoa(pipeline))
				b.WriteByte(')')
			}
			b.WriteString("\nWHERE id = $1\nRETURNING pipelines_to_purge")
			update := b.String()

			err = c.core.state.Transaction(c.close.ctx, func(tx *db.Tx) (any, error) {
				err := tx.QueryRow(c.close.ctx, update, id).Scan(&n.PipelinesToPurge)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, nil
					}
					return nil, err
				}
				return n, nil
			})
			if err != nil {
				return fmt.Errorf("cannot set pipelines as purged: %s", err)
			}

			return nil
		})
		if err != nil {
			slog.Error(err.Error(), "workspace", ws.ID)
		} else {
			// Try one last time to check whether any pipelines were added in the meantime and still need purging.
			// Note that we pass 1ns to SetNextWaitTime because it does not accept 0ns.
			bo.SetNextWaitTime(1)
		}

	}

}

// terminateOrphanedPipelineExecutions starts a termination task for pipeline
// executions whose node is no longer running or unresponsive.
// It must be called in its own goroutine.
//
// A pipeline execution is considered orphaned if it is not yet terminated, and
// its last ping time is older than 15 seconds.
func (c *pipelineCleaner) terminateOrphanedPipelineExecutions() {
	pipelineErr := newPipelineError(metrics.ReceiveStep, errors.New("pipeline has been terminated because the node became unresponsive"))
	ctx := c.close.ctx
	tick := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		pingTime := time.Now().UTC().Add(-15 * time.Second)
		err := c.core.db.QueryScan(ctx, "SELECT pipeline FROM pipelines_executions WHERE end_time IS NULL AND ping_time < $1",
			pingTime, func(rows *db.Rows) error {
				var pipelineID int
				for rows.Next() {
					if err := rows.Scan(&pipelineID); err != nil {
						return err
					}
					pipeline, ok := c.core.state.Pipeline(pipelineID)
					if !ok {
						continue
					}
					c2 := pipeline.Connection()
					ws := c2.Workspace()
					store := c.core.datastore.Store(ws.ID)
					connection := &Connection{core: c.core, store: store, connection: c2}
					p := &Pipeline{core: c.core, pipeline: pipeline, connection: connection}
					go p.endExecution(pipelineErr)
				}
				return nil
			})
		if err != nil && ctx.Err() == nil {
			slog.Error("core: cannot terminate orphaned pipeline executions", "err", err)
		}
	}
}

// unsetIdentityProperties unsets the identity properties that are no longer
// being transformed, for the pipeline with the provided ID.
func (c *pipelineCleaner) unsetIdentityProperties(id int) {

	if _, ok := c.pipelines.Swap(id, true); ok {
		return
	}
	defer c.pipelines.Delete(id)

	bo := backoff.New(backoffBase)
	for bo.Next(c.close.ctx) {

		pipeline, ok := c.core.state.Pipeline(id)
		if !ok {
			return
		}
		ws := pipeline.Connection().Workspace()
		if ws.Warehouse.Mode != state.Normal {
			return
		}
		paths := pipeline.PropertiesToUnset()
		if len(paths) == 0 {
			return
		}

		// Unset the properties.
		err := c.complete(func() error {

			store := c.core.datastore.Store(pipeline.Connection().Workspace().ID)
			err := store.UnsetIdentityProperties(c.close.ctx, pipeline.ID, paths)
			if err != nil {
				return err
			}
			n := state.UpdateIdentityPropertiesToUnset{
				Pipeline: pipeline.ID,
			}

			// Build the query that updates the identity properties to unset.
			var b strings.Builder
			b.WriteString("UPDATE pipelines\nSET properties_to_unset = ")
			for range len(paths) {
				b.WriteString("array_remove(")
			}
			b.WriteString("properties_to_unset")
			for _, path := range paths {
				b.WriteByte(',')
				b.WriteString(db.Quote(path))
				b.WriteByte(')')
			}
			b.WriteString("\nWHERE id = $1\nRETURNING properties_to_unset")
			update := b.String()

			err = c.core.state.Transaction(c.close.ctx, func(tx *db.Tx) (any, error) {
				err := tx.QueryRow(c.close.ctx, update, id).Scan(&n.Properties)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, nil
					}
					return nil, err
				}
				return n, nil
			})

			return err
		})
		if err == nil {
			break
		}

	}

}
