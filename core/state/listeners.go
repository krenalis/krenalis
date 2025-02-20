//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package state

// AddListener adds a notification listener.
// It must be called on a frozen state.
func (state *State) AddListener(listener any) {
	if state.changing.TryLock() {
		state.changing.Unlock()
		panic("state: AddListener called on an unfrozen state")
	}
	state.listeners = append(state.listeners, listener)
}

// dispatchNotification dispatches the notification n to its listeners.
func dispatchNotification[N any](state *State, n N) {
	for _, listener := range state.listeners {
		if listener, ok := listener.(func(N)); ok {
			listener(n)
		}
	}
}
