//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package core

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
)

var periods = [...]int16{5, 15, 30, 60, 120, 180, 360, 480, 720, 1440}

type scIndex struct {
	i int8
	j int16
}

func periodIndex(period int16) int8 {
	for i, iv := range periods {
		if iv == period {
			return int8(i)
		}
	}
	panic("invalid period")
}

// actionScheduler is the action scheduler.
type actionScheduler struct {
	core     *Core
	executor *actionSchedulerExecutor
	ctx      context.Context    // context passes to the action executions.
	cancel   context.CancelFunc // function to cancel the action executions.
	wg       sync.WaitGroup     // waiting group that includes the schedulers and action executions.
}

// newActionScheduler returns a new action scheduler.
func newActionScheduler(core *Core) *actionScheduler {
	as := &actionScheduler{
		core: core,
	}
	as.ctx, as.cancel = context.WithCancel(context.Background())
	core.state.Freeze()
	core.state.AddListener(as.onCreateAction)
	core.state.AddListener(as.onDeleteAction)
	core.state.AddListener(as.onDeleteConnection)
	core.state.AddListener(as.onDeleteWorkspace)
	core.state.AddListener(as.onElectLeader)
	core.state.AddListener(as.onSetActionSchedulePeriod)
	core.state.Unfreeze()
	return as
}

// Close closes the action scheduler closing the executors and interrupting
// action executions.
func (as *actionScheduler) Close() {
	as.core.state.Freeze()
	as.core.state.Unfreeze()
	if as.executor != nil {
		as.executor.Close()
	}
	as.cancel()
	as.wg.Wait()
}

// onCreateAction is called when an action is created.
func (as *actionScheduler) onCreateAction(n state.CreateAction) {
	if as.executor == nil {
		return
	}
	action, _ := as.core.state.Action(n.ID)
	if action.SchedulePeriod == 0 {
		return
	}
	as.executor.AddAction(action)
}

// onDeleteAction is called when an action is deleted from the state.
func (as *actionScheduler) onDeleteAction(n state.DeleteAction) {
	if as.executor == nil {
		return
	}
	go func() {
		as.executor.RemoveAction(n.ID)
	}()
}

// onDeleteConnection is called when a connection is deleted from the state.
func (as *actionScheduler) onDeleteConnection(n state.DeleteConnection) {
	if as.executor == nil {
		return
	}
	var actions []int
	for _, action := range n.Connection().Actions() {
		if action.SchedulePeriod != 0 {
			actions = append(actions, action.ID)
		}
	}
	if actions == nil {
		return
	}
	go func() {
		for _, action := range actions {
			as.executor.RemoveAction(action)
		}
	}()
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (as *actionScheduler) onDeleteWorkspace(n state.DeleteWorkspace) {
	if as.executor == nil {
		return
	}
	var actions []int
	for _, connection := range n.Workspace().Connections() {
		for _, action := range connection.Actions() {
			if action.SchedulePeriod != 0 {
				actions = append(actions, action.ID)
			}
		}
	}
	if actions == nil {
		return
	}
	go func() {
		for _, action := range actions {
			as.executor.RemoveAction(action)
		}
	}()
}

// ElectLeader is called when a leader is elected.
func (as *actionScheduler) onElectLeader(n state.ElectLeader) {
	if as.executor != nil {
		if !as.core.state.IsLeader() {
			go as.executor.Close()
		}
		return
	}
	if !as.core.state.IsLeader() {
		return
	}
	as.executor = newActionSchedulerExecutor(as.core, &as.wg, as.ctx)
}

// onSetActionSchedulePeriod is called when the schedule period of an action is
// set.
func (as *actionScheduler) onSetActionSchedulePeriod(n state.SetActionSchedulePeriod) {
	if as.executor == nil {
		return
	}
	action, _ := as.core.state.Action(n.ID)
	as.executor.SetPeriod(action)
}

// actionSchedulerExecutor represents an executor for an action scheduler. When
// a node becomes the leader, it starts an executor, and the previous leader
// closes its executor.
type actionSchedulerExecutor struct {
	core    *Core
	mu      sync.Mutex // for the actions and indexes fields.
	actions [len(periods)]map[int16][]*state.Action
	indexes map[int]scIndex
	close   chan struct{}
}

// newActionSchedulerExecutor returns a new action scheduler executor. wg is the
// wait group that coordinates goroutines for the executor and action
// executions. ctx is the context to pass to the action executions.
func newActionSchedulerExecutor(core *Core, wg *sync.WaitGroup, ctx context.Context) *actionSchedulerExecutor {

	se := &actionSchedulerExecutor{
		core:    core,
		indexes: map[int]scIndex{},
		close:   make(chan struct{}),
	}
	for i := range len(se.actions) {
		se.actions[i] = map[int16][]*state.Action{}
	}
	for _, action := range se.core.state.Actions() {
		if action.SchedulePeriod == 0 {
			continue
		}
		i := periodIndex(action.SchedulePeriod)
		j := action.ScheduleStart % action.SchedulePeriod
		se.actions[i][j] = append(se.actions[i][j], action)
		se.indexes[action.ID] = scIndex{i, j}
	}

	wg.Add(1)

	go func() {
		ticker := time.NewTicker(time.Minute)
		for {
			select {
			case t := <-ticker.C:
				minute := int16(t.Hour()*60 + t.Minute())
				for i, period := range periods {
					j := minute % period
					se.mu.Lock()
					actions := se.actions[i][j]
					se.mu.Unlock()
					for _, action := range actions {
						if !toExecute(action) {
							continue
						}
						connection := action.Connection()
						store := se.core.datastore.Store(connection.Workspace().ID)
						c := &Connection{core: se.core, connection: connection, store: store}
						a := &Action{core: se.core, action: action, connection: c}
						wg.Add(1)
						go func() {
							defer wg.Done()
							_, err := a.createExecution(ctx, nil)
							if err != nil {
								if _, ok := err.(*errors.NotFoundError); ok {
									return
								}
								if _, ok := err.(*errors.UnprocessableError); ok {
									return
								}
								slog.Debug("core: cannot add execution for action", "action", a.ID, "err", err)
							}
						}()
					}
				}
			case <-se.close:
				wg.Done()
				return
			}
		}
	}()

	return se
}

// Close closes the scheduler executor but does not interrupt any action
// execution.
func (se *actionSchedulerExecutor) Close() {
	close(se.close)
}

// AddAction adds action to the scheduler executor.
func (se *actionSchedulerExecutor) AddAction(action *state.Action) {
	i := periodIndex(action.SchedulePeriod)
	j := action.ScheduleStart % action.SchedulePeriod
	se.mu.Lock()
	se.actions[i][j] = append(slices.Clone(se.actions[i][j]), action)
	se.indexes[action.ID] = scIndex{i, j}
	se.mu.Unlock()
}

// RemoveAction removes the action with identifier id from the scheduler
// executor. If the action does not exist it does nothing.
func (se *actionSchedulerExecutor) RemoveAction(id int) {
	se.mu.Lock()
	index, ok := se.indexes[id]
	if !ok {
		se.mu.Unlock()
		return
	}
	i, j := index.i, index.j
	actions := se.actions[i][j]
	for k, action := range actions {
		if action.ID == id {
			actions = slices.Delete(actions, k, k+1)
			if len(actions) == 0 {
				delete(se.actions[i], j)
			} else {
				se.actions[i][j] = actions
			}
			break
		}
	}
	se.mu.Unlock()
}

// SetPeriod sets the period of an action.
func (se *actionSchedulerExecutor) SetPeriod(action *state.Action) {
	se.mu.Lock()
	index, ok := se.indexes[action.ID]
	se.mu.Unlock()
	if ok {
		if periods[index.i] == action.SchedulePeriod {
			return
		}
		se.RemoveAction(action.ID)
	}
	if action.SchedulePeriod != 0 {
		se.AddAction(action)
	}
}

// toExecute reports whether action can be executed.
func toExecute(action *state.Action) bool {
	if !action.Enabled || action.SchedulePeriod == 0 {
		return false
	}
	c := action.Connection()
	if c.Workspace().Warehouse.Mode != state.Normal {
		return false
	}
	if _, ok := action.Execution(); ok {
		return false
	}
	return true
}
