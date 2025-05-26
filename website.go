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

// WebsiteInfo represents a website connector info.
type WebsiteInfo struct {
	Name          string
	Icon          string // icon in SVG format
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the website connector
// info.
func (info WebsiteInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new website connector instance.
func (info WebsiteInfo) New(conf *WebsiteConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// WebsiteConfig represents the configuration of a website connector.
type WebsiteConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// WebsiteNewFunc represents functions that create new website connector
// instances.
type WebsiteNewFunc[T any] func(*WebsiteConfig) (T, error)
