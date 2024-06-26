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

// scheduler is the action scheduler.
type scheduler struct {
	apis      *APIs
	mu        sync.Mutex // for the actions and indexes fields.
	actions   [numPeriods]map[int16][]*state.Action
	indexes   map[int]scIndex
	listeners []uint8
	close     struct {
		ctx       context.Context
		cancelCtx context.CancelFunc
		shutdown  chan struct{}
		sync.WaitGroup
	}
}

// newScheduler returns a new scheduler.
//
// It is called during an elect leader notification if the current node is
// elected as leader.
func newScheduler(apis *APIs) *scheduler {

	sc := &scheduler{
		apis:    apis,
		indexes: map[int]scIndex{},
	}

	sc.close.ctx, sc.close.cancelCtx = context.WithCancel(context.Background())
	sc.close.shutdown = make(chan struct{})

	for i := range sc.actions {
		sc.actions[i] = map[int16][]*state.Action{}
	}

	for _, action := range apis.state.Actions() {
		if sc.toSchedule(action) {
			i := periodIndex(action.SchedulePeriod)
			j := action.ScheduleStart % action.SchedulePeriod
			sc.actions[i][j] = append(sc.actions[i][j], action)
			sc.indexes[action.ID] = scIndex{i, j}
		}
	}

	sc.listeners = []uint8{
		apis.state.AddListener(sc.onAddAction),
		apis.state.AddListener(sc.onDeleteAction),
		apis.state.AddListener(sc.onDeleteConnection),
		apis.state.AddListener(sc.onDeleteWorkspace),
		apis.state.AddListener(sc.onSetActionSchedulePeriod),
	}

	go func() {

		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

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
						if !sc.toExecute(action) {
							continue
						}
						connection := action.Connection()
						store := apis.datastore.Store(connection.Workspace().ID)
						c := &Connection{apis: apis, connection: connection, store: store}
						a := &Action{apis: apis, action: action, connection: c}
						sc.close.Add(1)
						go func() {
							defer sc.close.Done()
							err := a.addExecution(sc.close.ctx, false)
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
			case <-sc.close.shutdown:
				return
			}
		}

	}()

	return sc
}

// Close closes the scheduler.
func (sc *scheduler) Close() {
	close(sc.close.shutdown)
	sc.close.cancelCtx()
	sc.close.Wait()
	sc.apis.state.RemoveListeners(sc.listeners)
}

// Shutdown gracefully shuts down the scheduler without interrupting any action
// that is executing. If the provided context expires before the shutdown is
// complete, Shutdown interrupts any ongoing action and returns.
func (sc *scheduler) Shutdown(ctx context.Context) {
	close(sc.close.shutdown)
	stop := context.AfterFunc(ctx, func() { sc.close.cancelCtx() })
	defer stop()
	sc.close.Wait()
	sc.apis.state.RemoveListeners(sc.listeners)
}

// onAddAction is called when an action is added to the state.
func (sc *scheduler) onAddAction(n state.AddAction) func() {
	action, _ := sc.apis.state.Action(n.ID)
	if sc.toSchedule(action) {
		return func() {
			sc.mu.Lock()
			sc._addAction(action)
			sc.mu.Unlock()
		}
	}
	return nil
}

// onDeleteAction is called when an action is deleted from the state.
func (sc *scheduler) onDeleteAction(n state.DeleteAction) func() {
	return func() {
		sc.mu.Lock()
		sc._removeAction(n.ID)
		sc.mu.Unlock()
	}
}

// onDeleteConnection is called when a connection is deleted from the state.
func (sc *scheduler) onDeleteConnection(n state.DeleteConnection) func() {
	return func() {
		sc.mu.Lock()
		sc._removeActions()
		sc.mu.Unlock()
	}
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (sc *scheduler) onDeleteWorkspace(n state.DeleteWorkspace) func() {
	return func() {
		sc.mu.Lock()
		sc._removeActions()
		sc.mu.Unlock()
	}
}

// onSetActionSchedulePeriod is called when the schedule period of an action is
// set.
func (sc *scheduler) onSetActionSchedulePeriod(n state.SetActionSchedulePeriod) func() {
	action, _ := sc.apis.state.Action(n.ID)
	index, ok := sc.indexes[n.ID]
	if !ok {
		return nil
	}
	if periods[index.i] == action.SchedulePeriod {
		return nil
	}
	return func() {
		sc.mu.Lock()
		sc._removeAction(n.ID)
		sc._addAction(action)
		sc.mu.Unlock()
	}
}

// toExecute reports whether action can be executed.
func (sc *scheduler) toExecute(action *state.Action) bool {
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
func (sc *scheduler) toSchedule(action *state.Action) bool {
	if t := action.Target; t != state.Users && t != state.Groups {
		return false
	}
	typ := action.Connection().Connector().Type
	if typ != state.AppType && typ != state.DatabaseType && typ != state.FileStorageType {
		return false
	}
	return true
}

// _addAction adds action to the scheduler.
//
// It must be called when the state is frozen and holding the sc.mu mutex.
func (sc *scheduler) _addAction(action *state.Action) {
	i := periodIndex(action.SchedulePeriod)
	j := action.ScheduleStart % action.SchedulePeriod
	sc.actions[i][j] = append(slices.Clone(sc.actions[i][j]), action)
	sc.indexes[action.ID] = scIndex{i, j}
}

// _removeAction removes the action with identifier id from the scheduler.
// If the action does not exist it does nothing.
//
// It must be called when the state is frozen and holding the sc.mu mutex.
func (sc *scheduler) _removeAction(id int) {
	index, ok := sc.indexes[id]
	if !ok {
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
}

// _removeActions removes from the scheduler the actions that no longer exist.
//
// It must be called when the state is frozen and holding the sc.mu mutex.
func (sc *scheduler) _removeActions() {
	for id := range sc.indexes {
		if _, ok := sc.apis.state.Action(id); !ok {
			sc._removeAction(id)
		}
	}
}
