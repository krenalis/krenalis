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

	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/state"
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

// schedulerManager is the action scheduler manager.
type schedulerManager struct {
	apis      *APIs
	scheduler *scheduler
	ctx       context.Context    // context passes to the action executions.
	cancel    context.CancelFunc // function to cancel the action executions.
	wg        sync.WaitGroup     // waiting group that includes the schedulers and action executions.
}

type scheduler struct {
	apis    *APIs
	mu      sync.Mutex // for the actions and indexes fields.
	actions [numPeriods]map[int16][]*state.Action
	indexes map[int]scIndex
	close   chan struct{}
}

// newSchedulerManager returns a new scheduler manager.
func newSchedulerManager(apis *APIs) *schedulerManager {
	sc := &schedulerManager{
		apis: apis,
	}
	apis.state.AddListener(sc.onAddAction)
	apis.state.AddListener(sc.onDeleteAction)
	apis.state.AddListener(sc.onDeleteConnection)
	apis.state.AddListener(sc.onDeleteWorkspace)
	apis.state.AddListener(sc.onElectLeader)
	apis.state.AddListener(sc.onSetActionSchedulePeriod)
	sc.ctx, sc.cancel = context.WithCancel(context.Background())
	return sc
}

// Close closes the scheduler interrupting action executions.
func (sm *schedulerManager) Close() {
	if sm.scheduler != nil {
		sm.scheduler.Close()
	}
	sm.cancel()
	sm.wg.Wait()
}

// onAddAction is called when an action is added to the state.
func (sm *schedulerManager) onAddAction(n state.AddAction) func() {
	if sm.scheduler == nil {
		return nil
	}
	action, _ := sm.apis.state.Action(n.ID)
	if !toSchedule(action) {
		return nil
	}
	return func() {
		sm.scheduler.AddAction(action)
	}
}

// onDeleteAction is called when an action is deleted from the state.
func (sm *schedulerManager) onDeleteAction(n state.DeleteAction) func() {
	if sm.scheduler == nil {
		return nil
	}
	go func() {
		sm.scheduler.RemoveAction(n.ID)
	}()
	return nil
}

// onDeleteConnection is called when a connection is deleted from the state.
func (sm *schedulerManager) onDeleteConnection(n state.DeleteConnection) func() {
	if sm.scheduler == nil {
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
			sm.scheduler.RemoveAction(action)
		}
	}()
	return nil
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (sm *schedulerManager) onDeleteWorkspace(n state.DeleteWorkspace) func() {
	if sm.scheduler == nil {
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
			sm.scheduler.RemoveAction(action)
		}
	}()
	return nil
}

// ElectLeader is called when a leader is elected.
func (sm *schedulerManager) onElectLeader(n state.ElectLeader) func() {
	if sm.scheduler != nil {
		if !sm.apis.state.IsLeader() {
			go sm.scheduler.Close()
		}
		return nil
	}
	if !sm.apis.state.IsLeader() {
		return nil
	}
	return func() {
		sm.scheduler = newScheduler(sm.apis, &sm.wg, sm.ctx)
	}
}

// onSetActionSchedulePeriod is called when the schedule period of an action is
// set.
func (sm *schedulerManager) onSetActionSchedulePeriod(n state.SetActionSchedulePeriod) func() {
	if sm.scheduler != nil {
		return nil
	}
	action, _ := sm.apis.state.Action(n.ID)
	if !toSchedule(action) {
		return nil
	}
	return func() {
		sm.scheduler.SetPeriod(action)
	}
}

// newScheduler returns a new action scheduler. wg is the wait group that
// includes both the scheduler and all action executions. ctx is the context
// to pass to the action executions.
func newScheduler(apis *APIs, wg *sync.WaitGroup, ctx context.Context) *scheduler {

	sc := &scheduler{
		apis:    apis,
		indexes: map[int]scIndex{},
		close:   make(chan struct{}),
	}
	for i := range len(sc.actions) {
		sc.actions[i] = map[int16][]*state.Action{}
	}
	for _, action := range sc.apis.state.Actions() {
		if toSchedule(action) {
			i := periodIndex(action.SchedulePeriod)
			j := action.ScheduleStart % action.SchedulePeriod
			sc.actions[i][j] = append(sc.actions[i][j], action)
			sc.indexes[action.ID] = scIndex{i, j}
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
					sc.mu.Lock()
					actions := sc.actions[i][j]
					sc.mu.Unlock()
					for _, action := range actions {
						if !toExecute(action) {
							continue
						}
						connection := action.Connection()
						store := sc.apis.datastore.Store(connection.Workspace().ID)
						c := &Connection{apis: sc.apis, connection: connection, store: store}
						a := &Action{apis: sc.apis, action: action, connection: c}
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
			case <-sc.close:
				wg.Done()
				return
			}
		}
	}()

	return sc
}

// Close closes the scheduler but does not interrupt any executing action.
func (sc *scheduler) Close() {
	close(sc.close)
}

// AddAction adds action to the scheduler. It must be called when the state is
// frozen.
func (sc *scheduler) AddAction(action *state.Action) {
	i := periodIndex(action.SchedulePeriod)
	j := action.ScheduleStart % action.SchedulePeriod
	sc.mu.Lock()
	sc.actions[i][j] = append(slices.Clone(sc.actions[i][j]), action)
	sc.indexes[action.ID] = scIndex{i, j}
	sc.mu.Unlock()
}

// RemoveAction removes the action with identifier id from the scheduler. If the
// action does not exist it does nothing. It must be called when the state is
// frozen.
func (sc *scheduler) RemoveAction(id int) {
	sc.mu.Lock()
	index, ok := sc.indexes[id]
	if !ok {
		sc.mu.Unlock()
		return
	}
	i, j := index.i, index.j
	actions := sc.actions[i][j]
	for k, action := range actions {
		if action.ID == id {
			actions = slices.Delete(actions, k, k+1)
			if len(actions) == 0 {
				delete(sc.actions[i], j)
			} else {
				sc.actions[i][j] = actions
			}
			break
		}
	}
	sc.mu.Unlock()
}

// SetPeriod sets the period of an action.
func (sc *scheduler) SetPeriod(action *state.Action) {
	sc.mu.Lock()
	index, ok := sc.indexes[action.ID]
	sc.mu.Unlock()
	if !ok {
		return
	}
	if periods[index.i] == action.SchedulePeriod {
		return
	}
	sc.RemoveAction(action.ID)
	sc.AddAction(action)
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
	if typ != state.AppType && typ != state.DatabaseType && typ != state.FileStorageType {
		return false
	}
	return true
}
