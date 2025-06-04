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
func (info SDKInfo) New(conf *SDKConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// SDKConfig represents the configuration of an SDK connector.
type SDKConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// SDKNewFunc represents functions that create new SDK connector instances.
type SDKNewFunc[T any] func(*SDKConfig) (T, error)
