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

// Website represents a website connector.
type Website struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the website
// connection.
func (website Website) ConnectionReflectType() reflect.Type {
	return website.ct
}

// New returns a new website connection.
func (website Website) New(conf *WebsiteConfig) (WebsiteConnection, error) {
	out := website.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(WebsiteConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// WebsiteConfig represents the configuration of a website connection.
type WebsiteConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// WebsiteNewFunc represents functions that create new website connections.
type WebsiteNewFunc[T WebsiteConnection] func(*WebsiteConfig) (T, error)

// WebsiteConnection is the interface implemented by website connections.
type WebsiteConnection interface{}
