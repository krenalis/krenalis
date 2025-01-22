//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"reflect"
)

// ServerInfo represents a server connector info.
type ServerInfo struct {
	Name                   string
	SourceDescription      string
	DestinationDescription string
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the server connector
// info.
func (info ServerInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new server connector instance.
func (info ServerInfo) New(conf *ServerConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// ServerConfig represents the configuration of a server connector.
type ServerConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// ServerNewFunc represents functions that create new server connector
// instances.
type ServerNewFunc[T any] func(*ServerConfig) (T, error)
