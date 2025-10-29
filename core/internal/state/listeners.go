// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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
