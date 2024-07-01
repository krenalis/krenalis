//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package apis

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/open2b/chichi/apis/datastore"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/backoff"
)

// backoffBase is the base used for the backoff.
const backoffBase = 1000

// actionPurger represents an action purger. It purges user identities
// associated with deleted actions from the data warehouse. Identity purging
// occurs only when the data warehouse is in Normal mode.
type actionPurger struct {
	state     *state.State
	datastore *datastore.Datastore
	listeners []uint8
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
	p.listeners = []uint8{
		state.AddListener(p.onDeleteAction),
		state.AddListener(p.onDeleteConnection),
		state.AddListener(p.onSetWarehouse),
		state.AddListener(p.onSetWarehouseMode),
	}
	var workspaces []int
	for _, ws := range p.state.Workspaces() {
		if ws.NumActionsToPurge() > 0 {
			workspaces = append(workspaces, ws.ID)
		}
	}
	state.Unfreeze()
	if workspaces != nil {
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
	p.state.Freeze()
	p.state.RemoveListeners(p.listeners)
	p.state.Unfreeze()
	// Signals the closure.
	p.close.closed.Store(true)
	// Stop the backoff.
	p.mu.Lock()
	for _, bo := range p.backoff {
		bo.Stop()
	}
	p.mu.Unlock()
	// Cancel p.close.ctx if ctx is cancelled.
	stop := context.AfterFunc(ctx, func() { p.close.cancel() })
	defer stop()
	// Waits for the ongoing purges to finish.
	p.close.Wait()
}

// onDeleteAction is called when an action is deleted.
func (p *actionPurger) onDeleteAction(n state.DeleteAction) func() {
	a := n.Action()
	c := a.Connection()
	if a.Target != state.Users || c.Role != state.Source {
		return nil
	}
	ws := c.Workspace()
	if ws.Warehouse == nil || ws.Warehouse.Mode != state.Normal {
		return nil
	}
	go p.purgeWorkspace(ws.ID, nil)
	return nil
}

// onDeleteConnection is called when a connection is deleted.
func (p *actionPurger) onDeleteConnection(n state.DeleteConnection) func() {
	c := n.Connection()
	if c.Role != state.Source {
		return nil
	}
	ws := c.Workspace()
	if ws.Warehouse == nil || ws.Warehouse.Mode != state.Normal {
		return nil
	}
	for _, action := range c.Actions() {
		if action.Target == state.Users {
			go p.purgeWorkspace(ws.ID, nil)
			return nil
		}
	}
	return nil
}

// onSetWarehouse is called when the settings of a data warehouse are changed.
func (p *actionPurger) onSetWarehouse(n state.SetWarehouse) func() {
	if n.Warehouse == nil || n.Warehouse.Mode != state.Normal {
		return nil
	}
	if ws, _ := p.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return nil
	}
	go p.purgeWorkspace(n.Workspace, nil)
	return nil
}

// onSetWarehouseMode is called when the mode of a data warehouse is changed.
func (p *actionPurger) onSetWarehouseMode(n state.SetWarehouseMode) func() {
	if n.Mode != state.Normal {
		return nil
	}
	if ws, _ := p.state.Workspace(n.Workspace); ws.NumActionsToPurge() == 0 {
		return nil
	}
	go p.purgeWorkspace(n.Workspace, nil)
	return nil
}

// purgeWorkspace purges the identities associated with the delete actions of
// a workspace. bo is non-nil only when a purge is being retried.
func (p *actionPurger) purgeWorkspace(id int, bo *backoff.Backoff) {

	p.close.Add(1)
	defer p.close.Done()
	if p.close.closed.Load() {
		return
	}

	p.mu.Lock()
	if bo, ok := p.backoff[id]; ok {
		bo.Stop()
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
	store := p.datastore.Store(ws.ID)
	if store == nil {
		return
	}
	actions := ws.ActionsToPurge()
	if len(actions) == 0 {
		return
	}

	err := p.purgeIdentities(id, actions)
	if err != nil {
		p.mu.Lock()
		if _, ok := p.backoff[id]; !ok {
			if bo == nil {
				bo = backoff.New(backoffBase)
			}
			bo.AfterFunc(p.close.ctx, func(ctx context.Context) {
				p.purgeWorkspace(id, bo)
			})
			p.backoff[id] = bo
		}
		p.mu.Unlock()
	}

}

// purgeIdentities purges the identities of the provided actions from the
// workspace's data warehouse.
func (p *actionPurger) purgeIdentities(ws int, actions []int) error {

	store := p.datastore.Store(ws)
	err := store.PurgeIdentities(p.close.ctx, actions)
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
