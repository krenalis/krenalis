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

// StreamInfo represents a stream connector info.
type StreamInfo struct {
	Code          string
	Label         string
	Categories    Categories // categories
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the stream connector
// info.
func (info StreamInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new stream connector instance.
func (info StreamInfo) New(env *StreamEnv) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// StreamEnv is the environment for a stream connector.
type StreamEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// StreamNewFunc represents functions that create new stream connector
// instances.
type StreamNewFunc[T any] func(*StreamEnv) (T, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the stream.
	OrderKey string
}
