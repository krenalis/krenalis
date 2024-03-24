//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"reflect"
)

// StreamInfo represents a stream connector info.
type StreamInfo struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the stream connector
// info.
func (info StreamInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new stream connector instance.
func (info StreamInfo) New(conf *StreamConfig) (Stream, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(Stream)
	err, _ := out[1].Interface().(error)
	return c, err
}

// StreamConfig represents the configuration of a stream connector.
type StreamConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// StreamNewFunc represents functions that create new stream connector
// instances.
type StreamNewFunc[T Stream] func(*StreamConfig) (T, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the stream.
	OrderKey string
}

// Stream is the interface implemented by stream connectors.
// A Stream value can be used for sending or receiving but not both.
type Stream interface {

	// Close closes the stream. When Close is called, no other calls to the
	// connector's methods are in progress and no more will be made.
	Close() error

	// Receive receives an event from the stream. Callers call the ack function to
	// notify that the event has been received. The connector resends the event if
	// not acknowledged.
	//
	// Caller do not modify the event data, even temporarily, and event is not
	// retained after the ack function has been called.
	//
	// Receive can be used by multiple goroutines at the same time.
	Receive(ctx context.Context) (event []byte, ack func(), err error)

	// Send sends an event to the stream. If ack is not nil, connector calls ack
	// when the event has been stored or when an error occurred.
	//
	// Send can modify the event data, but event is not retained after the ack
	// function has been called.
	//
	// Send can be used by multiple goroutines at the same time.
	Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error
}
