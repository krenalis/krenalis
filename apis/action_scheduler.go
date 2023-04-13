//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"log"
	"sync"
	"time"

	"chichi/apis/errors"
	"chichi/apis/postgres"
	"chichi/apis/state"

	"golang.org/x/exp/slices"
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
	db      *postgres.DB
	state   *state.State
	mu      sync.Mutex // for the actions and indexes fields.
	actions [numPeriods]map[int16][]*state.Action
	indexes map[int]scIndex
	st      chan struct{}
}

// newScheduler returns a new scheduler.
//
// It is called during an elect leader notification if the current node is
// elected as leader
func newScheduler(db *postgres.DB, st *state.State) *scheduler {

	sc := &scheduler{
		db:      db,
		state:   st,
		indexes: map[int]scIndex{},
		st:      make(chan struct{}, 1),
	}

	for i := range sc.actions {
		sc.actions[i] = map[int16][]*state.Action{}
	}

	for _, action := range st.Actions() {
		if sc.toSchedule(action) {
			i := periodIndex(action.SchedulePeriod)
			j := action.ScheduleStart % action.SchedulePeriod
			sc.actions[i][j] = append(sc.actions[i][j], action)
			sc.indexes[action.ID] = scIndex{i, j}
		}
	}

	st.AddListener(sc.onAddAction)
	st.AddListener(sc.onDeleteAction)
	st.AddListener(sc.onDeleteConnection)
	st.AddListener(sc.onDeleteWorkspace)
	st.AddListener(sc.onSetActionSchedulePeriod)

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
						if sc.toExecute(action) {
							a := &Action{db: db, action: action}
							go func() {
								err := a.addExecution(false)
								if err != nil {
									if _, ok := err.(*errors.NotFoundError); ok {
										return
									}
									if _, ok := err.(*errors.UnprocessableError); ok {
										return
									}
									log.Printf("[error] cannot add execution for action %d: %s", a.ID, err)
								}
							}()
						}
					}
				}
			case <-sc.st:
				return
			}
		}

	}()

	return sc
}

// onAddAction is called when an action is added to the state.
func (sc *scheduler) onAddAction(n state.AddActionNotification) {
	action, _ := sc.state.Action(n.ID)
	if sc.toSchedule(action) {
		sc.mu.Lock()
		sc._addAction(action)
		sc.mu.Unlock()
	}
}

// onDeleteAction is called when an action is deleted from the state.
func (sc *scheduler) onDeleteAction(n state.DeleteActionNotification) {
	sc.mu.Lock()
	sc._removeAction(n.ID)
	sc.mu.Unlock()
}

// onDeleteConnection is called when a connection is deleted from the state.
func (sc *scheduler) onDeleteConnection(n state.DeleteConnectionNotification) {
	sc.mu.Lock()
	sc._removeActions()
	sc.mu.Unlock()
}

// onDeleteWorkspace is called when a workspace is deleted from the state.
func (sc *scheduler) onDeleteWorkspace(n state.DeleteWorkspaceNotification) {
	sc.mu.Lock()
	sc._removeActions()
	sc.mu.Unlock()
}

// onSetActionSchedulePeriod is called when the schedule period of an action is
// set.
func (sc *scheduler) onSetActionSchedulePeriod(n state.SetActionSchedulePeriodNotification) {
	action, _ := sc.state.Action(n.ID)
	index, ok := sc.indexes[n.ID]
	if !ok {
		return
	}
	if periods[index.i] == action.SchedulePeriod {
		return
	}
	sc.mu.Lock()
	sc._removeAction(n.ID)
	sc._addAction(action)
	sc.mu.Unlock()
}

// stop stops the scheduler.
//
// It is called during an elect leader notification.
func (sc *scheduler) stop() {
	sc.st <- struct{}{}
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
	if c.Connector().Type == state.FileType {
		if _, ok := c.Storage(); !ok {
			return false
		}
	}
	ws := c.Workspace()
	if ws.Warehouse == nil {
		return false
	}
	if _, ok := action.Execution(); ok {
		return false
	}
	return true
}

// toSchedule reports whether action can be scheduled.
func (sc *scheduler) toSchedule(action *state.Action) bool {
	return action.Target == state.UsersTarget || action.Target == state.GroupsTarget
}

// _addAction adds action to the scheduler.
//
// It must be called holding the sc.mu mutex.
func (sc *scheduler) _addAction(action *state.Action) {
	i := periodIndex(action.SchedulePeriod)
	j := action.ScheduleStart % action.SchedulePeriod
	sc.actions[i][j] = append(slices.Clone(sc.actions[i][j]), action)
	sc.indexes[action.ID] = scIndex{i, j}
}

// _removeAction removes the action with identifier id from the scheduler.
// If the action does not exist it does nothing.
//
// It must be called holding the sc.mu mutex.
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
// It must be called holding the sc.mu mutex.
func (sc *scheduler) _removeActions() {
	for id := range sc.indexes {
		if _, ok := sc.state.Action(id); !ok {
			sc._removeAction(id)
		}
	}
}
