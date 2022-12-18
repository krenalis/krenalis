//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package apis

import (
	"errors"
	"sync"
)

var (
	errAccountNotFound       = errors.New("account does not exist")
	errConnectionNotFound    = errors.New("connection does not exist")
	errConnectorNotFound     = errors.New("connector does not exist")
	errEventDataTypeNotFound = errors.New("event data type does not exist")
	errEventTypeNotFound     = errors.New("event type does not exist")
	errWorkspaceNotFound     = errors.New("workspace does not exist")
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

// Get returns the account with identifier id.
// Returns the errAccountNotFound error if the account does not exist.
func (state *accountsState) Get(id int) (*Account, error) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	if ok {
		return c, nil
	}
	return nil, errAccountNotFound
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

// Get returns the connector with identifier id.
// Returns the errConnectorNotFound error if the connector does not exist.
func (state *connectorsState) Get(id int) (*Connector, error) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	if ok {
		return c, nil
	}
	return nil, errConnectorNotFound
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

// Get returns the workspace with identifier id.
// Returns the errWorkspaceNotFound error if the workspace does not exist.
func (state *workspacesState) Get(id int) (*Workspace, error) {
	state.Lock()
	w, ok := state.ids[id]
	state.Unlock()
	if ok {
		return w, nil
	}
	return nil, errWorkspaceNotFound
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

// Get returns the connection with identifier id.
// Returns the errConnectionNotFound error if the connection does not exist.
func (state *connectionsState) Get(id int) (*Connection, error) {
	state.Lock()
	c, ok := state.ids[id]
	state.Unlock()
	if ok {
		return c, nil
	}
	return nil, errConnectionNotFound
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

// transformationsState contains the state of a single connection's
// transformations.
type transformationsState struct {
	sync.Mutex
	ofConnection map[int][]*Transformation
}

// List returns the transformations of the given connection.
//
// If there are no transformations associated to the given connection, it
// returns nil.
func (state *transformationsState) List(connection int) []*Transformation {
	state.Lock()
	ts := state.ofConnection[connection]
	state.Unlock()
	return ts
}

// set sets the transformations of the given connection. If transformations is
// nil, then every transformation associated to the connection is removed.
func (state *transformationsState) set(connection int, transformations []*Transformation) {
	state.Lock()
	if transformations == nil {
		delete(state.ofConnection, connection)
	} else {
		state.ofConnection[connection] = transformations
	}
	state.Unlock()
}

// eventDataState contains the state of a single workspace's event types.
type eventTypesState struct {
	sync.Mutex
	ids map[int]*EventType
}

// Get returns the type with identifier id. It returns the
// errEventTypeNotFound error if the type does not exist.
func (state *eventTypesState) Get(id int) (*EventType, error) {
	state.Lock()
	t, ok := state.ids[id]
	state.Unlock()
	if !ok {
		return nil, errEventTypeNotFound
	}
	return t, nil
}

// List returns all event types.
func (state *eventTypesState) List() []*EventType {
	state.Lock()
	eventTypes := make([]*EventType, len(state.ids))
	i := 0
	for _, t := range state.ids {
		eventTypes[i] = t
	}
	state.Unlock()
	return eventTypes
}

// eventDataTypesState contains the state of a single workspace's event data
// types.
type eventDataTypesState struct {
	sync.Mutex
	names map[string]*EventDataType
}

// Get returns the data type with the given name. It returns the
// errEventDataTypeNotFound error if the type does not exist.
func (state *eventDataTypesState) Get(name string) (*EventDataType, error) {
	state.Lock()
	t, ok := state.names[name]
	state.Unlock()
	if !ok {
		return nil, errEventDataTypeNotFound
	}
	return t, nil
}

// List returns all the data types.
func (state *eventDataTypesState) List() []*EventDataType {
	state.Lock()
	dataTypes := make([]*EventDataType, len(state.names))
	i := 0
	for _, t := range state.names {
		dataTypes[i] = t
		i++
	}
	state.Unlock()
	return dataTypes
}
