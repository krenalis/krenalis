//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package state

import (
	"fmt"
	"sync"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// AddListener adds a notification listener.
// It must be called when the state is frozen, i.e., before Keep is called,
// during a listener execution, or between calls to Freeze and Unfreeze.
func (state *State) AddListener(listener any) {
	switch l := listener.(type) {
	case func(AddAction) func():
		state.listeners.AddAction = append(state.listeners.AddAction, l)
	case func(AddConnection) func():
		state.listeners.AddConnection = append(state.listeners.AddConnection, l)
	case func(DeleteAction) func():
		state.listeners.DeleteAction = append(state.listeners.DeleteAction, l)
	case func(DeleteConnection) func():
		state.listeners.DeleteConnection = append(state.listeners.DeleteConnection, l)
	case func(DeleteWorkspace) func():
		state.listeners.DeleteWorkspace = append(state.listeners.DeleteWorkspace, l)
	case func(ElectLeader) func():
		state.listeners.ElectLeader = append(state.listeners.ElectLeader, l)
	case func(ExecuteAction) func():
		state.listeners.ExecuteAction = append(state.listeners.ExecuteAction, l)
	case func(SetAction) func():
		state.listeners.SetAction = append(state.listeners.SetAction, l)
	case func(SetActionSchedulePeriod) func():
		state.listeners.SetActionSchedulePeriod = append(state.listeners.SetActionSchedulePeriod, l)
	case func(SetConnection) func():
		state.listeners.SetConnection = append(state.listeners.SetConnection, l)
	case func(SetConnectionSettings) func():
		state.listeners.SetConnectionSettings = append(state.listeners.SetConnectionSettings, l)
	case func(SetWarehouse) func():
		state.listeners.SetWarehouse = append(state.listeners.SetWarehouse, l)
	case func(SetWarehouseMode) func():
		state.listeners.SetWarehouseMode = append(state.listeners.SetWarehouseMode, l)
	case func(SetWorkspace) func():
		state.listeners.SetWorkspace = append(state.listeners.SetWorkspace, l)
	case func(schema SetWorkspaceUserSchema) func():
		state.listeners.SetWorkspaceUserSchema = append(state.listeners.SetWorkspaceUserSchema, l)
	default:
		panic(fmt.Sprintf("state: unexpected listener type %T", listener))
	}
}

// notifyListeners notifies all listeners of a notification.
func notifyListeners[T func(N) func(), N any](notification N, listeners []T) {
	if len(listeners) == 0 {
		return
	}
	wg := sync.WaitGroup{}
	for _, listener := range listeners {
		if f := listener(notification); f != nil {
			wg.Add(1)
			go func() {
				f()
				wg.Done()
			}()
		}
	}
	wg.Wait()
}
