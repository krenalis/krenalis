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

// MobileInfo represents a mobile connector info.
type MobileInfo struct {
	Name                   string
	SourceDescription      string
	DestinationDescription string
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the mobile connector
// info.
func (info MobileInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new mobile connector instance.
func (info MobileInfo) New(conf *MobileConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// MobileConfig represents the configuration of a mobile connector.
type MobileConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// MobileNewFunc represents functions that create new mobile connector
// instances.
type MobileNewFunc[T any] func(*MobileConfig) (T, error)
