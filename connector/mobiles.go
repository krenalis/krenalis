//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"reflect"
)

// Mobile represents a mobile connector.
type Mobile struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the mobile
// connection.
func (mobile Mobile) ConnectionReflectType() reflect.Type {
	return mobile.ct
}

// New returns a new mobile connection.
func (mobile Mobile) New(conf *MobileConfig) (MobileConnection, error) {
	out := mobile.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(MobileConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// MobileConfig represents the configuration of a mobile connection.
type MobileConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// MobileNewFunc represents functions that create new mobile connections.
type MobileNewFunc[T MobileConnection] func(*MobileConfig) (T, error)

// MobileConnection is the interface implemented by mobile connections.
type MobileConnection interface{}
