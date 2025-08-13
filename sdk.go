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

// SDKInfo represents an SDK connector info.
type SDKInfo struct {
	Name          string
	Categories    Categories // categories
	Icon          string     // icon in SVG format
	Strategies    bool       // whether this connector supports users strategies
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the SDK connector
// info.
func (info SDKInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new SDK connector instance.
func (info SDKInfo) New(env *SDKEnv) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// SDKEnv is the environment for an SDK connector.
type SDKEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// SDKNewFunc represents functions that create new SDK connector instances.
type SDKNewFunc[T any] func(*SDKEnv) (T, error)
