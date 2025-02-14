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
	"github.com/meergo/meergo/core/state"
)

// backoffBase is the base used for the backoff.
const backoffBase = 1000

// actionPurger represents an action purger. It purges user identities
// associated with deleted actions from the data warehouse. Identity purging
// occurs only when the data warehouse is in Normal mode.
type actionPurger struct {
	state     *state.State
	datastore *datastore.Datastore
	close     struct {
		ctx    context.Context
		cancel context.CancelFunc
		closed atomic.Bool
		sync.WaitGroup
	}

	mu      sync.Mutex               // for the 'backoff' field
	backoff map[int]*backoff.Backoff // backoff for workspace. access using 'mu'
}

// newActionPurger returns a new instance of the action purger. There is only
// one active action purger at a time, and it exclusively runs on the leader
// node.
func newActionPurger(state *state.State, datastore *datastore.Datastore) *actionPurger {

	p := &actionPurger{
		state:     state,
		datastore: datastore,
		backoff:   map[int]*backoff.Backoff{},
	}
	p.close.ctx, p.close.cancel = context.WithCancel(context.Background())

	state.Freeze()
	state.AddListener(p.onDeleteAction)
	state.AddListener(p.onDeleteConnection)
	state.AddListener(p.onUpdateWarehouse)
	state.AddListener(p.onUpdateWarehouseMode)
	var workspaces []int
	for _, ws := range p.state.Workspaces() {
		if ws.NumActionsToPurge() > 0 {
			workspaces = append(workspaces, ws.ID)
		}
	}
	state.Unfreeze()
	if workspaces != nil {
		p.close.Add(len(workspaces))
		go func() {
			for _, ws := range workspaces {
				p.purgeWorkspace(ws, nil)
			}
		}()
	}

	return p
}

// Close closes the action purger, ensuring the completion of all ongoing
// purges. If the context is canceled, it interrupts ongoing purges and returns.
// If p is already closed, it does nothing and returns immediately.
func (p *actionPurger) Close(ctx context.Context) {
	if p.close.closed.Load() {
		return
	}
	// Signals the closure.
	p.close.closed.Store(true)
	// Stop the backoff.
	p.mu.Lock()
	for _, bo := range p.backoff {
		if bo.Stop() {
			p.close.Done()
		}
	}
	p.mu.Unlock()
	// Cancel p.close.ctx if ctx is cancelled.
	stop := context.AfterFunc(ctx, func() { p.close.cancel() })
	defer stop()
	// Waits for the ongoing purges to finish.
	p.close.Wait()
}

// onDeleteAction is called when an action is deleted.
func (p *actionPurger) onDeleteAction(n state.DeleteAction) {
	a := n.Action()
	c := a.Connection()
	if a.Target != state.Users || c.Role != state.Source {
		return
	}
	ws := c.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	p.close.Add(1)
	go p.purgeWorkspace(ws.ID, nil)
}

// onDeleteConnection is called when a connection is deleted.
func (p *actionPurger) onDeleteConnection(n state.DeleteConnection) {
	c := n.Connection()
	if c.Role != state.Source {
		return
	}
	ws := c.Workspace()
	if ws.Warehouse.Mode != state.Normal {
		return
	}
	for _, action := range c.Actions() {
		if action.Target == state.Users {
			p.close.Add(1)
			go p.purgeWorkspace(ws.ID, nil)
			return
		}
	}
}

// onUpdateWarehouse is called when a warehouse is updated.
func (p *actionPurger) onUpdateWarehouse(n state.UpdateWarehouse) {
	if n.Mode != state.Normal {
		return
	}
	if ws, _ := p.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return
	}
	p.close.Add(1)
	go p.purgeWorkspace(n.Workspace, nil)
}

// onUpdateWarehouseMode is called when the mode of a warehouse is updated.
func (p *actionPurger) onUpdateWarehouseMode(n state.UpdateWarehouseMode) {
	if n.Mode != state.Normal {
		return
	}
	if ws, _ := p.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return
	}
	p.close.Add(1)
	go p.purgeWorkspace(n.Workspace, nil)
}

// purgeWorkspace purges the identities associated with the delete actions of
// a workspace. bo is non-nil only when a purge is being retried.
func (p *actionPurger) purgeWorkspace(id int, bo *backoff.Backoff) {

	defer p.close.Done()
	if p.close.closed.Load() {
		return
	}

	p.mu.Lock()
	if bo, ok := p.backoff[id]; ok {
		if bo.Stop() {
			p.close.Done()
		}
		delete(p.backoff, id)
	}
	p.mu.Unlock()

	ws, ok := p.state.Workspace(id)
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

	err := p.purgeActions(id, actions)
	if err != nil {
		p.mu.Lock()
		if _, ok := p.backoff[id]; !ok {
			if bo == nil {
				bo = backoff.New(backoffBase)
			}
			p.close.Add(1)
			bo.AfterFunc(p.close.ctx, func(ctx context.Context) {
				p.purgeWorkspace(id, bo)
			})
			p.backoff[id] = bo
		}
		p.mu.Unlock()
	}

}

// purgeActions purges the identities and the destination users of the provided
// actions from the workspace's data warehouse.
func (p *actionPurger) purgeActions(ws int, actions []int) error {

	store := p.datastore.Store(ws)
	err := store.PurgeActions(p.close.ctx, actions)
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

	err = p.state.Transaction(p.close.ctx, func(tx *state.Tx) error {
		var actions []int
		err := tx.QueryRow(p.close.ctx, update, ws).Scan(&actions)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil
			}
			return err
		}
		n.ActionsToPurge = actions
		return tx.Notify(p.close.ctx, n)
	})

	return err
}
