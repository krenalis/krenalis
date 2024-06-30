//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package state

import (
	"math"
	"sync"
)

// logNotifications controls the logging of notifications on the log.
const logNotifications = false

// AddListener adds a notification listener and returns its identifier.
// It must be called on a frozen state.
func (state *State) AddListener(listener any) uint8 {
	if state.changing.TryLock() {
		state.changing.Unlock()
		panic("state: AddListener called on an unfrozen state")
	}
	if len(state.listeners.all) == math.MaxUint8-1 {
		panic("state: too many listeners")
	}
	var ok bool
	id := state.listeners.id
	for {
		id++
		if id == 0 {
			id = 1
		}
		_, ok = state.listeners.all[id]
		if !ok {
			break
		}
	}
	state.listeners.id = id
	state.listeners.all[id] = listener
	return id
}

// RemoveListeners removes the listeners with the provided identifiers.
// It must be called on a frozen state.
// It panics if a listener to remove does not exist.
func (state *State) RemoveListeners(ids []uint8) {
	if state.changing.TryLock() {
		state.changing.Unlock()
		panic("state: RemoveListeners called on an unfrozen state")
	}
	for _, id := range ids {
		if _, ok := state.listeners.all[id]; !ok {
			panic("state: listener to remove not found")
		}
		delete(state.listeners.all, id)
	}
}

// dispatchNotification dispatches the notification n to its listeners.
func dispatchNotification[N any](state *State, n N) {
	if len(state.listeners.all) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, listener := range state.listeners.all {
		listener, ok := listener.(func(N) func())
		if !ok {
			continue
		}
		if f := listener(n); f != nil {
			wg.Add(1)
			go func() {
				f()
				wg.Done()
			}()
		}
	}
	wg.Wait()
}
