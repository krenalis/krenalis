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
	Name          string
	Categories    Category // bitmask of connector's categories
	Icon          string   // icon in SVG format
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
func (info StreamInfo) New(conf *StreamConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// StreamConfig represents the configuration of a stream connector.
type StreamConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// StreamNewFunc represents functions that create new stream connector
// instances.
type StreamNewFunc[T any] func(*StreamConfig) (T, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the stream.
	OrderKey string
}
