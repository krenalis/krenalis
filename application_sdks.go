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

// SDKSpec represents an application SDK connector specification.
type SDKSpec struct {
	Code                string
	Label               string
	Categories          Categories // categories
	Strategies          bool       // whether this connector supports user strategies
	FallbackToRequestIP bool       // whether to use the request IP as the event IP if context.ip was not provided
	Documentation       ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the application SDK
// connector specification.
func (spec SDKSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new application SDK connector instance.
func (spec SDKSpec) New(env *SDKEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// SDKEnv is the environment for an application SDK connector.
type SDKEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// SDKNewFunc represents functions that create new application SDK connector
// instances.
type SDKNewFunc[T any] func(*SDKEnv) (T, error)
