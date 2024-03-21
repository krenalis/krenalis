//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"reflect"
)

// Server represents a server connector.
type Server struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the server
// connection.
func (server Server) ConnectionReflectType() reflect.Type {
	return server.ct
}

// New returns a new server connection.
func (server Server) New(conf *ServerConfig) (ServerConnection, error) {
	out := server.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(ServerConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// ServerConfig represents the configuration of a server connection.
type ServerConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// ServerNewFunc represents functions that create new server connections.
type ServerNewFunc[T ServerConnection] func(*ServerConfig) (T, error)

// ServerConnection is the interface implemented by server connections.
type ServerConnection interface{}
