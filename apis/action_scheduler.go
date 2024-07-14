//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/state"
)

const numPeriods = 10

var periods = [numPeriods]int16{5, 15, 30, 60, 120, 180, 360, 480, 720, 1440}

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
	apis      *APIs
	listeners []uint8
	executor  *actionSchedulerExecutor
	ctx       context.Context    // context passes to the action executions.
	cancel    context.CancelFunc // function to cancel the action executions.
	wg        sync.WaitGroup     // waiting group that includes the schedulers and action executions.
}

// newActionScheduler returns a new action scheduler.
func newActionScheduler(apis *APIs) *actionScheduler {
	as := &actionScheduler{
		apis: apis,
	}
	as.ctx, as.cancel = context.WithCancel(context.Background())
	apis.state.Freeze()
	as.listeners = []uint8{
		apis.state.AddListener(as.onAddAction),
		apis.state.AddListener(as.onDeleteAction),
		apis.state.AddListener(as.onDeleteConnection),
		apis.state.AddListener(as.onDeleteWorkspace),
		apis.state.AddListener(as.onElectLeader),
		apis.state.AddListener(as.onSetActionSchedulePeriod),
	}
	apis.state.Unfreeze()
	return as
}

// Close closes the action scheduler closing the executors and interrupting
// action executions.
func (as *actionScheduler) Close() {
	as.apis.state.Freeze()
	as.apis.state.RemoveListeners(as.listeners)
	as.apis.state.Unfreeze()
	if as.executor != nil {
		as.executor.Close()
	}
	as.cancel()
	as.wg.Wait()
}

// onAddAction is called when an action is added to the state.
func (as *actionScheduler) onAddAction(n state.AddAction) func() {
	if as.executor == nil {
		return nil
	}
	action, _ := as.apis.state.Action(n.ID)
	if !toSchedule(action) {
		return nil
	}
	return func() {
		as.executor.AddAction(action)
	}
}

// onDeleteAction is called when an action is deleted from the state.
func (as *actionScheduler) onDeleteAction(n state.DeleteAction) func() {
	if as.executor == nil {
		return nil
	}
	go func() {
		as.executor.RemoveAction(n.ID)
	}()
	return nil
}

// onDeleteConnection is called when a connection is deleted from the state.
func (as *actionScheduler) onDeleteConnection(n state.DeleteConnection) func() {
	if as.executor == nil {
		return nil
	}
	var actions []int
	for _, action := range n.Connection().Actions() {
		if toSchedule(action) {
			actions = append(actions, action.ID)
		}
	}
	if actions == nil {
		return nil
	}
	go func() {
		for _, action := range actions {
			as.executor.RemoveAction(action)
		}
	}()
	return nil
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (as *actionScheduler) onDeleteWorkspace(n state.DeleteWorkspace) func() {
	if as.executor == nil {
		return nil
	}
	var actions []int
	for _, connection := range n.Workspace().Connections() {
		for _, action := range connection.Actions() {
			if toSchedule(action) {
				actions = append(actions, action.ID)
			}
		}
	}
	if actions == nil {
		return nil
	}
	go func() {
		for _, action := range actions {
			as.executor.RemoveAction(action)
		}
	}()
	return nil
}

// ElectLeader is called when a leader is elected.
func (as *actionScheduler) onElectLeader(n state.ElectLeader) func() {
	if as.executor != nil {
		if !as.apis.state.IsLeader() {
			go as.executor.Close()
		}
		return nil
	}
	if !as.apis.state.IsLeader() {
		return nil
	}
	return func() {
		as.executor = newActionSchedulerExecutor(as.apis, &as.wg, as.ctx)
	}
}

// onSetActionSchedulePeriod is called when the schedule period of an action is
// set.
func (as *actionScheduler) onSetActionSchedulePeriod(n state.SetActionSchedulePeriod) func() {
	if as.executor != nil {
		return nil
	}
	action, _ := as.apis.state.Action(n.ID)
	if !toSchedule(action) {
		return nil
	}
	return func() {
		as.executor.SetPeriod(action)
	}
}

// actionSchedulerExecutor represents an executor for an action scheduler. When
// a node becomes the leader, it starts an executor, and the previous leader
// closes its executor.
type actionSchedulerExecutor struct {
	apis    *APIs
	mu      sync.Mutex // for the actions and indexes fields.
	actions [numPeriods]map[int16][]*state.Action
	indexes map[int]scIndex
	close   chan struct{}
}

// newActionSchedulerExecutor returns a new action scheduler executor. wg is the
// wait group that coordinates goroutines for the executor and action
// executions. ctx is the context to pass to the action executions.
func newActionSchedulerExecutor(apis *APIs, wg *sync.WaitGroup, ctx context.Context) *actionSchedulerExecutor {

	se := &actionSchedulerExecutor{
		apis:    apis,
		indexes: map[int]scIndex{},
		close:   make(chan struct{}),
	}
	for i := range len(se.actions) {
		se.actions[i] = map[int16][]*state.Action{}
	}
	for _, action := range se.apis.state.Actions() {
		if toSchedule(action) {
			i := periodIndex(action.SchedulePeriod)
			j := action.ScheduleStart % action.SchedulePeriod
			se.actions[i][j] = append(se.actions[i][j], action)
			se.indexes[action.ID] = scIndex{i, j}
		}
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
						store := se.apis.datastore.Store(connection.Workspace().ID)
						c := &Connection{apis: se.apis, connection: connection, store: store}
						a := &Action{apis: se.apis, action: action, connection: c}
						wg.Add(1)
						go func() {
							defer wg.Done()
							err := a.addExecution(ctx, false)
							if err != nil {
								if _, ok := err.(*errors.NotFoundError); ok {
									return
								}
								if _, ok := err.(*errors.UnprocessableError); ok {
									return
								}
								slog.Debug("cannot add execution for action", "action", a.ID, "err", err)
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
	if !ok {
		return
	}
	if periods[index.i] == action.SchedulePeriod {
		return
	}
	se.RemoveAction(action.ID)
	se.AddAction(action)
}

// toExecute reports whether action can be executed.
func toExecute(action *state.Action) bool {
	if !action.Enabled {
		return false
	}
	c := action.Connection()
	if !c.Enabled {
		return false
	}
	ws := c.Workspace()
	if wh := ws.Warehouse; wh == nil || wh.Mode != state.Normal {
		return false
	}
	if _, ok := action.Execution(); ok {
		return false
	}
	return true
}

// toSchedule reports whether action can be scheduled.
func toSchedule(action *state.Action) bool {
	if t := action.Target; t != state.Users && t != state.Groups {
		return false
	}
	typ := action.Connection().Connector().Type
	if typ != state.App && typ != state.Database && typ != state.FileStorage {
		return false
	}
	return true
}
