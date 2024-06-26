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
// It must be called when the state is frozen, i.e., before Keep is called,
// during a listener execution, or between calls to Freeze and Unfreeze.
func (state *State) AddListener(listener any) uint8 {
	if len(state.listeners) == math.MaxUint8-1 {
		panic("state: too many listeners")
	}
	var ok bool
	id := state.lastListenerID
	for {
		id++
		if id == 0 {
			id = 1
		}
		_, ok = state.listeners[id]
		if !ok {
			break
		}
	}
	state.lastListenerID = id
	state.listeners[id] = listener
	return id
}

// RemoveListeners removes the listeners with the provided identifiers.
// It must be called when the state is frozen, i.e., before Keep is called,
// during a listener execution, or between calls to Freeze and Unfreeze.
func (state *State) RemoveListeners(ids []uint8) {
	for _, id := range ids {
		if _, ok := state.listeners[id]; !ok {
			panic("state: listener to remove not found")
		}
		delete(state.listeners, id)
	}
}

// dispatchNotification dispatches a notification to the provided listeners.
func dispatchNotification[N any](notification N, listeners map[uint8]any) {
	if len(listeners) == 0 {
		return
	}
	var wg sync.WaitGroup
	for _, listener := range listeners {
		listener, ok := listener.(func(N) func())
		if !ok {
			continue
		}
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
