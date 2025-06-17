//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package collector

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/meergo/meergo/core/connectors"
	"github.com/meergo/meergo/core/events"
	"github.com/meergo/meergo/core/events/collector/sender"
	"github.com/meergo/meergo/core/metrics"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/transformers"
	"github.com/meergo/meergo/types"
)

// destinations is responsible for dispatching events to destination apps.
// Use the QueueEvent method to enqueue events for delivery.
type destinations struct {
	state      *state.State
	connectors *connectors.Connectors
	provider   transformers.FunctionProvider
	metrics    *metrics.Collector

	// senders maps a connection ID to its sender.
	// No mutex is needed since all accesses occur while the state is frozen.
	senders map[int]*sender.Sender

	mu      sync.Mutex
	actions map[int][]*destinationAction // maps a destination connection ID to its actions; it is protected by mu.

	close struct {
		closed    atomic.Bool             // indicates if the writer has been closed
		ctx       context.Context         // context passes to iterators
		cancel    context.CancelCauseFunc // function to cancel iterators executions
		completed sync.Cond               // signal the completion of the current iteration
		iterators sync.WaitGroup          // waiting group for the iterators
	}
}

// newDestinations returns a new destinations instance.
func newDestinations(st *state.State, connectors *connectors.Connectors, provider transformers.FunctionProvider, metrics *metrics.Collector) *destinations {

	d := destinations{
		state:      st,
		connectors: connectors,
		provider:   provider,
		metrics:    metrics,
		senders:    map[int]*sender.Sender{},
		actions:    map[int][]*destinationAction{},
	}
	d.close.ctx, d.close.cancel = context.WithCancelCause(context.Background())

	// Keeps all destination connections whose connector supports events.
	d.state.Freeze()
	d.state.AddListener(d.onCreateAction)
	d.state.AddListener(d.onCreateConnection)
	d.state.AddListener(d.onDeleteAction)
	d.state.AddListener(d.onDeleteConnection)
	d.state.AddListener(d.onDeleteWorkspace)
	d.state.AddListener(d.onSetActionStatus)
	d.state.AddListener(d.onUpdateAction)
	for _, c := range st.Connections() {
		if c.Role != state.Destination {
			continue
		}
		if !c.Connector().DestinationTargets.Contains(state.TargetEvent) {
			continue
		}
		app := connectors.App(c)
		sender := sender.New(c.Connector().Name, app.SendEvents, d.sentAcks)
		actions := make([]*destinationAction, 0, 1)
		// Keeps all actions active on the connection's events.
		for _, a := range c.Actions() {
			if !a.Enabled || a.Target != state.TargetEvent {
				continue
			}
			action := d.newDestinationAction(a, sender)
			actions = append(actions, action)
		}
		d.senders[c.ID] = sender
		d.actions[c.ID] = actions
	}
	d.state.Unfreeze()

	return &d
}

// QueueEvent queues the given event to be sent on the specified destination
// connection.
func (d *destinations) QueueEvent(connection int, event events.Event) {
	d.mu.Lock()
	for _, action := range d.actions[connection] {
		d.metrics.ReceivePassed(action.id, 1)
		action.QueueEvent(event)
	}
	d.mu.Unlock()
}

// newDestinationAction returns a new destination action for the provided action
// with the provided sender.
func (d *destinations) newDestinationAction(action *state.Action, sender *sender.Sender) *destinationAction {
	return newDestinationAction(d, action, sender)
}

// onCreateAction is called when an action is created.
func (d *destinations) onCreateAction(n state.CreateAction) {
	if !n.Enabled || n.Target != state.TargetEvent {
		return
	}
	a, _ := d.state.Action(n.ID)
	c := a.Connection()
	if c.Role != state.Destination {
		return
	}
	// No lock is needed for reading d.senders since the state is frozen,
	// ensuring there are no concurrent writes.
	action := d.newDestinationAction(a, d.senders[c.ID])
	d.mu.Lock()
	d.actions[c.ID] = append(d.actions[c.ID], action)
	d.mu.Unlock()
}

// onCreateConnection is called when a connection is created.
func (d *destinations) onCreateConnection(n state.CreateConnection) {
	if n.Role != state.Destination {
		return
	}
	c, _ := d.state.Connection(n.ID)
	connector := c.Connector()
	if !connector.DestinationTargets.Contains(state.TargetEvent) {
		return
	}
	app := d.connectors.App(c)
	d.senders[n.ID] = sender.New(connector.Name, app.SendEvents, d.sentAcks)
	actions := make([]*destinationAction, 0, 1)
	d.mu.Lock()
	d.actions[n.ID] = actions
	d.mu.Unlock()
}

// onDeleteAction is called when an action is deleted
func (d *destinations) onDeleteAction(n state.DeleteAction) {
	a := n.Action()
	if !a.Enabled || a.Target != state.TargetEvent {
		return
	}
	c := a.Connection()
	if c.Role != state.Destination {
		return
	}
	var i int
	var action *destinationAction
	actions := d.actions[c.ID]
	for i, action = range actions {
		if action.id == a.ID {
			break
		}
	}
	if i == len(actions) {
		panic("unexpected missing action")
	}
	d.mu.Lock()
	d.actions[c.ID] = slices.Delete(actions, i, i+1)
	d.mu.Unlock()
	go action.Discard(errors.New("action has been deleted"))
}

// onDeleteConnection is called when a connection is deleted.
func (d *destinations) onDeleteConnection(n state.DeleteConnection) {
	c := n.Connection()
	if c.Role != state.Destination {
		return
	}
	connector := c.Connector()
	if !connector.DestinationTargets.Contains(state.TargetEvent) {
		return
	}
	delete(d.senders, c.ID)
	actions := d.actions[c.ID]
	d.mu.Lock()
	delete(d.actions, c.ID)
	d.mu.Unlock()
	go func() {
		for _, action := range actions {
			action.Discard(errors.New("connection has been deleted"))
		}
	}()
}

// onDeleteWorkspace is called when a workspace is deleted.
func (d *destinations) onDeleteWorkspace(n state.DeleteWorkspace) {
	ws := n.Workspace()
	var actions []*destinationAction
	for _, c := range ws.Connections() {
		if c.Role != state.Destination {
			continue
		}
		connector := c.Connector()
		if !connector.DestinationTargets.Contains(state.TargetEvent) {
			continue
		}
		delete(d.senders, c.ID)
		actions = append(actions, d.actions[c.ID]...)
		d.mu.Lock()
		delete(d.actions, c.ID)
		d.mu.Unlock()
	}
	if len(actions) > 0 {
		go func() {
			for _, action := range actions {
				action.Discard(errors.New("workspace has been deleted"))
			}
		}()
	}
}

// onSetActionStatus is called when the status of an action is set.
func (d *destinations) onSetActionStatus(n state.SetActionStatus) {
	a, _ := d.state.Action(n.ID)
	if a.Target != state.TargetEvent {
		return
	}
	c := a.Connection()
	if c.Role != state.Destination {
		return
	}
	actions := d.actions[c.ID]
	if n.Enabled {
		// Add the action.
		action := d.newDestinationAction(a, d.senders[c.ID])
		d.mu.Lock()
		d.actions[c.ID] = append(actions, action)
		d.mu.Unlock()
		return
	}
	// Remove the action.
	var i int
	var action *destinationAction
	for i, action = range actions {
		if action.id == a.ID {
			break
		}
	}
	if i == len(actions) {
		panic("unexpected missing action")
	}
	d.mu.Lock()
	actions = slices.Delete(actions, i, i+1)
	d.actions[c.ID] = actions
	d.mu.Unlock()
	go action.Discard(errors.New("action has been disabled"))
}

// onUpdateAction is called when an action is updated.
func (d *destinations) onUpdateAction(n state.UpdateAction) {
	a, _ := d.state.Action(n.ID)
	if a.Target != state.TargetEvent {
		return
	}
	c := a.Connection()
	if c.Role != state.Destination {
		return
	}
	actions := d.actions[c.ID]
	var current *destinationAction
	var index int
	for i, action := range actions {
		if action.id == a.ID {
			current = action
			index = i
			break
		}
	}
	// Removes it if is not enabled but present.
	if !a.Enabled {
		if current != nil {
			d.mu.Lock()
			d.actions[c.ID] = slices.Delete(actions, index, index+1)
			d.mu.Unlock()
			go current.Discard(errors.New("action has been disabled"))
		}
		return
	}
	// Adds it if it wasn't present.
	if current == nil {
		action := d.newDestinationAction(a, d.senders[c.ID])
		d.mu.Lock()
		d.actions[c.ID] = append(actions, action)
		d.mu.Unlock()
		return
	}
	// If Filter, or Transformation has changed, replace the destination action
	// with a new version.
	sameFilter := current.filter.Equal(a.Filter)
	sameSchema := types.Equal(current.schema, a.OutSchema)
	sameTransformation := current.transformation.Equal(a.Transformation)
	if sameFilter && sameSchema && sameTransformation {
		return
	}
	action := *current
	if !sameFilter {
		action.filter = a.Filter
	}
	if !sameSchema {
		action.schema = a.OutSchema
	}
	if !sameTransformation {
		t := a.Transformation
		action.transformation = t
		if t.Mapping == nil && t.Function == nil {
			action.transformer = nil
		} else {
			action.transformer, _ = transformers.New(a, d.provider, nil)
		}
	}
	d.mu.Lock()
	actions[index] = &action
	d.mu.Unlock()
}

// sentAcks is the function passed to the sender, which calls it to acknowledge
// the delivery of events (or report errors).
func (d *destinations) sentAcks(acks []sender.Ack, err error) {
	for _, ack := range acks {
		if err != nil {
			d.metrics.FinalizeFailed(ack.Action, 1, err.Error())
			continue
		}
		d.metrics.FinalizePassed(ack.Action, 1)
	}
}
