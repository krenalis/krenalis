//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"sync"
)

// accountsState contains the state of all accounts.
type accountsState struct {
	sync.Mutex
	ids map[int]*Account
}

// Count returns the total number of accounts.
func (state *accountsState) Count() int {
	state.Lock()
	count := len(state.ids)
	state.Unlock()
	return count
}

// Get returns the account with identifier id. The boolean return value reports
// whether the account exists.
func (state *accountsState) Get(id int) (*Account, bool) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	return c, ok
}

// List returns all accounts.
func (state *accountsState) List() []*Account {
	state.Lock()
	accounts := make([]*Account, len(state.ids))
	i := 0
	for _, account := range state.ids {
		accounts[i] = account
	}
	state.Unlock()
	return accounts
}

// connectorsState contains the state of all connectors.
type connectorsState struct {
	sync.Mutex
	ids map[int]*Connector
}

// Get returns the connector with identifier id. The boolean return value
// reports whether the connector exists.
func (state *connectorsState) Get(id int) (*Connector, bool) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	return c, ok
}

// List returns all the connectors.
func (state *connectorsState) List() []*Connector {
	state.Lock()
	connectors := make([]*Connector, len(state.ids))
	i := 0
	for _, c := range state.ids {
		connectors[i] = c
		i++
	}
	state.Unlock()
	return connectors
}

// resourcesState contains the state of a single workspace's resources.
type resourcesState struct {
	sync.Mutex
	ids map[int]*Resource
}

// add adds a resource to the state. If a resource with the same identifier
// already exists, add replaces it.
func (state *resourcesState) add(r *Resource) {
	state.Lock()
	state.ids[r.id] = r
	state.Unlock()
}

// delete deletes the resource with identifier id. If the resource does not
// exist, it does nothing.
func (state *resourcesState) delete(id int) {
	state.Lock()
	delete(state.ids, id)
	state.Unlock()
}

// Get returns the resource with identifier id. The boolean return value
// reports whether the resource exists.
func (state *resourcesState) Get(id int) (*Resource, bool) {
	state.Lock()
	r, ok := state.ids[id]
	state.Unlock()
	return r, ok
}

// GetByCode returns the resource with the given code. The boolean return value
// reports whether the resource exists.
func (state *resourcesState) GetByCode(code string) (*Resource, bool) {
	var r *Resource
	state.Lock()
	for _, resource := range state.ids {
		if resource.code == code {
			r = resource
			break
		}
	}
	state.Unlock()
	return r, r != nil
}

// workspacesState contains the state of a single account's workspaces.
type workspacesState struct {
	sync.Mutex
	ids map[int]*Workspace
}

// Get returns the workspace with identifier id. The boolean return value
// reports whether the workspace exists.
// Returns the errWorkspaceNotFound error if the workspace does not exist.
func (state *workspacesState) Get(id int) (*Workspace, bool) {
	state.Lock()
	w, ok := state.ids[id]
	state.Unlock()
	return w, ok
}

// List returns all the workspaces.
func (state *workspacesState) List() []*Workspace {
	state.Lock()
	workspaces := make([]*Workspace, len(state.ids))
	i := 0
	for _, c := range state.ids {
		workspaces[i] = c
		i++
	}
	state.Unlock()
	return workspaces
}

// connectionsState contains the state of a single workspace's collections.
type connectionsState struct {
	sync.Mutex
	ids map[int]*Connection
}

// Get returns the connection with identifier id. The boolean return value
// reports whether the connection exists.
func (state *connectionsState) Get(id int) (*Connection, bool) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	return c, ok
}

// List returns all the connections.
func (state *connectionsState) List() []*Connection {
	state.Lock()
	connections := make([]*Connection, len(state.ids))
	i := 0
	for _, c := range state.ids {
		connections[i] = c
		i++
	}
	state.Unlock()
	return connections
}

// typesState contains the state of a single workspace's types.
type typesState struct {
	sync.Mutex
	names map[string]*Type
}

// Get returns the type with the given name. The boolean return value reports
// whether the type exists.
func (state *typesState) Get(name string) (*Type, bool) {
	state.Lock()
	t, ok := state.names[name]
	state.Unlock()
	return t, ok
}

// List returns all the types.
func (state *typesState) List() []*Type {
	state.Lock()
	types := make([]*Type, len(state.names))
	i := 0
	for _, t := range state.names {
		types[i] = t
		i++
	}
	state.Unlock()
	return types
}
