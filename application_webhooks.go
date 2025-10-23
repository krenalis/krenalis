//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"reflect"
)

// WebhookSpec represents an application webhook connector specification.
type WebhookSpec struct {
	Code          string
	Label         string
	Categories    Categories // categories
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the application
// webhook connector specification.
func (spec WebhookSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new application webhook connector instance.
func (spec WebhookSpec) New(env *WebhookEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// WebhookEnv is the environment for an application webhook connector.
type WebhookEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// WebhookNewFunc represents functions that create new application webhook
// connector instances.
type WebhookNewFunc[T any] func(*WebhookEnv) (T, error)
