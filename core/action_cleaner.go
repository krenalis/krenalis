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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/meergo/meergo/backoff"
	"github.com/meergo/meergo/core/datastore"
	"github.com/meergo/meergo/core/db"
	"github.com/meergo/meergo/core/state"
)

// backoffBase is the base used for the backoff.
const backoffBase = 1000

// actionCleaner represents an action cleaner. It performs the following tasks
// in the data warehouse:
//
// - Purges user identities associated with deleted actions.
// - Unsets identity properties that are no longer transformed.
//
// Action cleaning occurs only when the data warehouse is in Normal mode.
type actionCleaner struct {
	state     *state.State
	datastore *datastore.Datastore
	close     struct {
		ctx    context.Context
		cancel context.CancelFunc
		closed atomic.Bool
		sync.WaitGroup
	}

	mu      sync.Mutex // for the 'backoff' field
	backoff struct {
		workspace map[int]*backoff.Backoff // backoff for workspace. access using 'mu'
		action    map[int]*backoff.Backoff // backoff for action. access using 'mu'
	}
}

// newActionCleaner returns a new instance of the action cleaner. There is only
// one active action cleaner at a time, and it exclusively runs on the leader
// node.
func newActionCleaner(state *state.State, datastore *datastore.Datastore) *actionCleaner {

	p := &actionCleaner{
		state:     state,
		datastore: datastore,
	}
	p.backoff.workspace = map[int]*backoff.Backoff{}
	p.backoff.action = map[int]*backoff.Backoff{}
	p.close.ctx, p.close.cancel = context.WithCancel(context.Background())

	state.Freeze()
	state.AddListener(p.onDeleteAction)
	state.AddListener(p.onDeleteConnection)
	state.AddListener(p.onUpdateAction)
	state.AddListener(p.onUpdateWarehouse)
	state.AddListener(p.onUpdateWarehouseMode)
	var workspaces []int
	for _, ws := range p.state.Workspaces() {
		if ws.NumActionsToPurge() > 0 {
			workspaces = append(workspaces, ws.ID)
		}
	}
	var actions []int
	for _, action := range p.state.Actions() {
		if properties := action.PropertiesToUnset(); len(properties) > 0 {
			actions = append(actions, action.ID)
		}
	}
	state.Unfreeze()
	for _, ws := range workspaces {
		p.close.Add(1)
		go p.purgeWorkspace(ws, nil)
	}
	for _, action := range actions {
		p.close.Add(1)
		go p.unsetIdentityProperties(action, nil)
	}

	return p
}

// Close closes the action cleaner, ensuring the completion of all ongoing
// operations. If the context is canceled, it interrupts ongoing operations and
// returns. If p is already closed, it does nothing and returns immediately.
func (c *actionCleaner) Close(ctx context.Context) {
	if c.close.closed.Load() {
		return
	}
	// Signals the closure.
	c.close.closed.Store(true)
	// Stop the backoff.
	c.mu.Lock()
	for _, bo := range c.backoff.workspace {
		if bo.Stop() {
			c.close.Done()
		}
	}
	for _, bo := range c.backoff.action {
		if bo.Stop() {
			c.close.Done()
		}
	}
	c.mu.Unlock()
	// Cancel p.close.ctx if ctx is cancelled.
	stop := context.AfterFunc(ctx, func() { c.close.cancel() })
	defer stop()
	// Waits for the ongoing operations to finish.
	c.close.Wait()
}

// onDeleteAction is called when an action is deleted.
func (c *actionCleaner) onDeleteAction(n state.DeleteAction) {
	action := n.Action()
	connection := action.Connection()
	if action.Target != state.Users || connection.Role != state.Source {
		return
	}
	ws := connection.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	c.close.Add(1)
	go c.purgeWorkspace(ws.ID, nil)
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
		if action.Target == state.Users {
			c.close.Add(1)
			go c.purgeWorkspace(ws.ID, nil)
			return
		}
	}
}

// onUpdateAction is called when an action is updated.
func (c *actionCleaner) onUpdateAction(n state.UpdateAction) {
	if len(n.PropertiesToUnset) == 0 {
		return
	}
	a, _ := c.state.Action(n.ID)
	ws := a.Connection().Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	c.close.Add(1)
	go c.unsetIdentityProperties(n.ID, nil)
}

// onUpdateWarehouse is called when a warehouse is updated.
func (c *actionCleaner) onUpdateWarehouse(n state.UpdateWarehouse) {
	if n.Mode != state.Normal {
		return
	}
	if ws, _ := c.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return
	}
	c.close.Add(1)
	go c.purgeWorkspace(n.Workspace, nil)
}

// onUpdateWarehouseMode is called when the mode of a warehouse is updated.
func (c *actionCleaner) onUpdateWarehouseMode(n state.UpdateWarehouseMode) {
	if n.Mode != state.Normal {
		return
	}
	ws, _ := c.state.Workspace(n.Workspace)
	if ws.NumActionsToPurge() > 0 {
		c.close.Add(1)
		go c.purgeWorkspace(n.Workspace, nil)
	}
	for _, connection := range ws.Connections() {
		for _, action := range connection.Actions() {
			if paths := action.PropertiesToUnset(); paths != nil {
				c.close.Add(1)
				go c.unsetIdentityProperties(action.ID, nil)
			}
		}
	}
}

// purgeWorkspace purges the identities associated with the delete actions of
// a workspace. bo is non-nil only when a purge is being retried.
func (c *actionCleaner) purgeWorkspace(id int, bo *backoff.Backoff) {

	defer c.close.Done()
	if c.close.closed.Load() {
		return
	}

	c.mu.Lock()
	if bo, ok := c.backoff.workspace[id]; ok {
		if bo.Stop() {
			c.close.Done()
		}
		delete(c.backoff.workspace, id)
	}
	c.mu.Unlock()

	ws, ok := c.state.Workspace(id)
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

	err := c.purgeActions(id, actions)
	if err != nil {
		c.mu.Lock()
		if _, ok := c.backoff.workspace[id]; !ok {
			if bo == nil {
				bo = backoff.New(backoffBase)
			}
			c.close.Add(1)
			bo.AfterFunc(c.close.ctx, func(ctx context.Context) {
				c.purgeWorkspace(id, bo)
			})
			c.backoff.workspace[id] = bo
		}
		c.mu.Unlock()
	}

}

// purgeActions purges the identities and the destination users of the provided
// actions from the workspace's data warehouse.
func (c *actionCleaner) purgeActions(ws int, actions []int) error {

	store := c.datastore.Store(ws)
	err := store.PurgeActions(c.close.ctx, actions)
	if err != nil {
		return err
	}

	n := state.PurgeActions{
		Workspace: ws,
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

	err = c.state.Transaction(c.close.ctx, func(tx *state.Tx) error {
		var actions []int
		err := tx.QueryRow(c.close.ctx, update, ws).Scan(&actions)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		n.ActionsToPurge = actions
		return tx.Notify(c.close.ctx, n)
	})

	return err
}

// unsetIdentityProperties unsets the identity properties that are no longer
// being transformed, for the action with the provided ID.
func (c *actionCleaner) unsetIdentityProperties(id int, bo *backoff.Backoff) {

	defer c.close.Done()
	if c.close.closed.Load() {
		return
	}

	c.mu.Lock()
	if bo, ok := c.backoff.action[id]; ok {
		if bo.Stop() {
			c.close.Done()
		}
		delete(c.backoff.action, id)
	}
	c.mu.Unlock()

	action, ok := c.state.Action(id)
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
	err := func() error {

		store := c.datastore.Store(action.Connection().Workspace().ID)
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
			b.WriteString(db.QuoteValue(path))
			b.WriteByte(')')
		}
		b.WriteString("\nWHERE id = $1\nRETURNING properties_to_unset")
		update := b.String()

		err = c.state.Transaction(c.close.ctx, func(tx *state.Tx) error {
			err := tx.QueryRow(c.close.ctx, update, id).Scan(&n.Properties)
			if err != nil {
				if err == sql.ErrNoRows {
					return nil
				}
				return err
			}
			return tx.Notify(c.close.ctx, n)
		})

		return nil
	}()
	if err != nil {
		c.mu.Lock()
		if _, ok := c.backoff.action[id]; !ok {
			if bo == nil {
				bo = backoff.New(backoffBase)
			}
			c.close.Add(1)
			bo.AfterFunc(c.close.ctx, func(ctx context.Context) {
				c.unsetIdentityProperties(id, bo)
			})
			c.backoff.action[id] = bo
		}
		c.mu.Unlock()
	}

}
