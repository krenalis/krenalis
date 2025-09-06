//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

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

	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/core/internal/db"
	"github.com/meergo/meergo/core/internal/metrics"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/transformers"
)

// backoffBase is the base used for the backoff.
const backoffBase = 1000

// functionDeletionInterval defines how often discontinued functions are
// deleted.
const functionDeletionInterval = 10 * time.Minute

// actionCleaner represents an action cleaner. It performs the following tasks:
//
//   - Purges user identities in the data warehouse that are associated with
//     deleted actions.
//   - Unsets identity properties in the data warehouse that are no longer
//     transformed.
//   - Deletes discontinued functions from its function provider.
//   - Terminates orphaned action executions.
//
// Action cleaning occurs only when the data warehouse is in Normal mode.
type actionCleaner struct {
	core             *Core
	functionProvider transformers.FunctionProvider
	workspaces       sync.Map // map of workspaces indicating if an operation is in progress
	actions          sync.Map // map of actions indicating if an operation is in progress
	close            struct {
		ctx    context.Context
		cancel context.CancelFunc
		atomic.Bool
		sync.WaitGroup
	}
}

// newActionCleaner returns a new instance of the action cleaner. There is only
// one active action cleaner at a time, and it exclusively runs on the leader
// node.
func newActionCleaner(core *Core, provider transformers.FunctionProvider) *actionCleaner {

	c := &actionCleaner{
		core:             core,
		functionProvider: provider,
	}
	c.close.ctx, c.close.cancel = context.WithCancel(context.Background())

	core.state.Freeze()
	core.state.AddListener(c.onDeleteAction)
	core.state.AddListener(c.onDeleteConnection)
	core.state.AddListener(c.onUpdateAction)
	core.state.AddListener(c.onUpdateWarehouse)
	core.state.AddListener(c.onUpdateWarehouseMode)
	var workspaces []int
	for _, ws := range c.core.state.Workspaces() {
		if ws.NumActionsToPurge() > 0 {
			workspaces = append(workspaces, ws.ID)
		}
	}
	var actions []int
	for _, action := range c.core.state.Actions() {
		if properties := action.PropertiesToUnset(); len(properties) > 0 {
			actions = append(actions, action.ID)
		}
	}
	core.state.Unfreeze()
	for _, ws := range workspaces {
		go c.purgeWorkspace(ws)
	}
	for _, action := range actions {
		go c.unsetIdentityProperties(action)
	}

	// Start a goroutine to delete functions that have been discontinued.
	if provider != nil {
		go c.deleteDiscontinuedFunctions()
	}

	// Start a goroutine to terminate orphaned action executions.
	go c.terminateOrphanedActionExecutions()

	return c
}

// Close closes the action cleaner, ensuring the completion of all ongoing
// operations. If the context is canceled, it interrupts ongoing operations and
// returns. If p is already closed, it does nothing and returns immediately.
func (c *actionCleaner) Close(ctx context.Context) {
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
func (c *actionCleaner) complete(f func() error) error {
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
// been discontinued and are no longer in use by any executing actions.
// It must be called in its own goroutine.
//
// A function is considered discontinued if:
//
//   - The associated action has been deleted.
//   - The transformation type has changed from function-based to mapping-based.
//   - The function has switched between JavaScript and Python.
func (c *actionCleaner) deleteDiscontinuedFunctions() {
	var ids, deleted []string
	var d = 2 * time.Second // initial waiting time.
	for {
		tick := time.NewTicker(d)
		select {
		case <-c.close.ctx.Done():
			return
		case <-tick.C:
		}
		d = functionDeletionInterval
		err := c.complete(func() error {
			// Read the functions. These are the discontinued ones from over ten
			// minutes ago, with no actions still using them.
			rows, err := c.core.db.Query(c.close.ctx, "SELECT f.id\n"+
				"FROM discontinued_functions AS f\n"+
				"LEFT JOIN actions_executions AS e ON f.id = e.function AND e.end_time IS NULL\n"+
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
	}
}

// onDeleteAction is called when an action is deleted.
func (c *actionCleaner) onDeleteAction(n state.DeleteAction) {
	action := n.Action()
	connection := action.Connection()
	if action.Target != state.TargetUser || connection.Role != state.Source {
		return
	}
	ws := connection.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	go c.purgeWorkspace(ws.ID)
}

// onDeleteConnection is called when a connection is deleted.
func (c *actionCleaner) onDeleteConnection(n state.DeleteConnection) {
	connection := n.Connection()
	if connection.Role != state.Source {
		return
	}
	ws := connection.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	for _, action := range connection.Actions() {
		if action.Target == state.TargetUser {
			go c.purgeWorkspace(ws.ID)
			return
		}
	}
}

// onUpdateAction is called when an action is updated.
func (c *actionCleaner) onUpdateAction(n state.UpdateAction) {
	if len(n.PropertiesToUnset) == 0 {
		return
	}
	a, _ := c.core.state.Action(n.ID)
	ws := a.Connection().Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	go c.unsetIdentityProperties(n.ID)
}

// onUpdateWarehouse is called when a warehouse is updated.
func (c *actionCleaner) onUpdateWarehouse(n state.UpdateWarehouse) {
	if n.Mode != state.Normal {
		return
	}
	if ws, _ := c.core.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return
	}
	go c.purgeWorkspace(n.Workspace)
}

// onUpdateWarehouseMode is called when the mode of a warehouse is updated.
func (c *actionCleaner) onUpdateWarehouseMode(n state.UpdateWarehouseMode) {
	if n.Mode != state.Normal {
		return
	}
	ws, _ := c.core.state.Workspace(n.Workspace)
	if ws.NumActionsToPurge() > 0 {
		go c.purgeWorkspace(n.Workspace)
	}
	for _, connection := range ws.Connections() {
		for _, action := range connection.Actions() {
			if paths := action.PropertiesToUnset(); paths != nil {
				go c.unsetIdentityProperties(action.ID)
			}
		}
	}
}

// purgeWorkspace purges the identities associated with actions that have been
// deleted for the workspace with the identifier id.
func (c *actionCleaner) purgeWorkspace(id int) {

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
		actions := ws.ActionsToPurge()
		if len(actions) == 0 {
			return
		}
		err := c.complete(func() error {

			store := c.core.datastore.Store(id)
			err := store.PurgeActions(c.close.ctx, actions)
			if err != nil {
				return err
			}

			n := state.PurgeActions{
				Workspace: id,
			}

			// Build the query that updates the actions to purge of the workspace.
			var b strings.Builder
			b.WriteString("UPDATE workspaces\nSET actions_to_purge = ")
			for range len(actions) {
				b.WriteString("array_remove(")
			}
			b.WriteString("actions_to_purge")
			for _, action := range actions {
				b.WriteByte(',')
				b.WriteString(strconv.Itoa(action))
				b.WriteByte(')')
			}
			b.WriteString("\nWHERE id = $1 AND actions_to_purge IS NOT NULL\nRETURNING actions_to_purge")
			update := b.String()

			err = c.core.state.Transaction(c.close.ctx, func(tx *db.Tx) (any, error) {
				var actions []int
				err := tx.QueryRow(c.close.ctx, update, id).Scan(&actions)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, nil
					}
					return nil, err
				}
				n.ActionsToPurge = actions
				return n, nil
			})

			return err
		})
		if err == nil {
			break
		}

	}

}

// terminateOrphanedActionExecutions starts a termination task for action
// executions whose node is no longer running or unresponsive.
// It must be called in its own goroutine.
//
// An action execution is considered orphaned if it is not yet terminated, and
// its last ping time is older than 15 seconds.
func (c *actionCleaner) terminateOrphanedActionExecutions() {
	actionErr := newActionError(metrics.ReceiveStep, errors.New("action has been terminated because the node became unresponsive"))
	ctx := c.close.ctx
	tick := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		pingTime := time.Now().UTC().Add(-15 * time.Second)
		err := c.core.db.QueryScan(ctx, "SELECT action FROM actions_executions WHERE end_time IS NULL AND ping_time < $1",
			pingTime, func(rows *db.Rows) error {
				var actionID int
				for rows.Next() {
					if err := rows.Scan(&actionID); err != nil {
						return err
					}
					action, ok := c.core.state.Action(actionID)
					if !ok {
						continue
					}
					c2 := action.Connection()
					ws := c2.Workspace()
					store := c.core.datastore.Store(ws.ID)
					connection := &Connection{core: c.core, store: store, connection: c2}
					a := &Action{core: c.core, action: action, connection: connection}
					go a.endExecution(actionErr)
				}
				return nil
			})
		if err != nil && ctx.Err() == nil {
			slog.Error("core: cannot terminate orphaned action executions", "err", err)
		}
	}
}

// unsetIdentityProperties unsets the identity properties that are no longer
// being transformed, for the action with the provided ID.
func (c *actionCleaner) unsetIdentityProperties(id int) {

	if _, ok := c.actions.Swap(id, true); ok {
		return
	}
	defer c.actions.Delete(id)

	bo := backoff.New(backoffBase)
	for bo.Next(c.close.ctx) {

		action, ok := c.core.state.Action(id)
		if !ok {
			return
		}
		ws := action.Connection().Workspace()
		if ws.Warehouse.Mode != state.Normal {
			return
		}
		paths := action.PropertiesToUnset()
		if len(paths) == 0 {
			return
		}

		// Unset the properties.
		err := c.complete(func() error {

			store := c.core.datastore.Store(action.Connection().Workspace().ID)
			err := store.UnsetIdentityProperties(c.close.ctx, action.ID, paths)
			if err != nil {
				return err
			}
			n := state.UpdateIdentityPropertiesToUnset{
				Action: action.ID,
			}

			// Build the query that updates the identity properties to unset.
			var b strings.Builder
			b.WriteString("UPDATE actions\nSET properties_to_unset = ")
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
