// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package meergo

import (
	"reflect"
)

// WebhookSpec represents a webhook connector specification.
type WebhookSpec struct {
	Code          string
	Label         string
	Categories    Categories // categories
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the webhook connector
// specification.
func (spec WebhookSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new webhook connector instance.
func (spec WebhookSpec) New(env *WebhookEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// WebhookEnv is the environment for a webhook connector.
type WebhookEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// WebhookNewFunc represents functions that create new webhook connector
// instances.
type WebhookNewFunc[T any] func(*WebhookEnv) (T, error)
