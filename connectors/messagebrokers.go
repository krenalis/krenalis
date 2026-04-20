// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"reflect"
)

// MessageBrokerSpec represents a message broker connector specification.
type MessageBrokerSpec struct {
	Code          string
	Label         string
	Categories    Categories
	Documentation Documentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the message broker
// connector specification.
func (spec MessageBrokerSpec) ReflectType() reflect.Type {
	return spec.ct
}

// New returns a new message broker connector instance.
func (spec MessageBrokerSpec) New(env *MessageBrokerEnv) (any, error) {
	out := spec.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// MessageBrokerEnv is the environment for a message broker connector.
type MessageBrokerEnv struct {

	// Settings holds the settings.
	Settings SettingsStore
}

// MessageBrokerNewFunc represents functions that create new message broker
// connector instances.
type MessageBrokerNewFunc[T any] func(*MessageBrokerEnv) (T, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the message broker.
	OrderKey string
}
