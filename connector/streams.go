//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"reflect"
)

// Stream represents a stream connector.
type Stream struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
}

// Open opens a stream connection.
func (stream Stream) Open(ctx context.Context, conf *StreamConfig) (StreamConnection, error) {
	out := stream.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
	c := out[0].Interface().(StreamConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// StreamConfig represents the configuration of a stream connection.
type StreamConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenStreamFunc represents functions that open stream connections.
type OpenStreamFunc[T StreamConnection] func(context.Context, *StreamConfig) (T, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the stream.
	OrderKey string
}

// StreamConnection is the interface implemented by stream connections.
// A StreamConnection value can be use for sending or receiving but not both.
type StreamConnection interface {

	// Close closes the stream. When Close is called, no other calls to connection
	// methods are in progress and no more will be made.
	Close() error

	// Receive receives an event from the stream. Callers call the ack function to
	// notify that the event has been received. The connector resends the event if
	// not acknowledged.
	//
	// Caller do not modify the event data, even temporarily, and event is not
	// retained after the ack function has been called.
	//
	// Receive can be used by multiple goroutines at the same time.
	Receive() (event []byte, ack func(), err error)

	// Send sends an event to the stream. If ack is not nil, connector calls ack
	// when the event has been stored or when an error occurred.
	//
	// Send can modify the event data, but event is not retained after the ack
	// function has been called.
	//
	// Send can be used by multiple goroutines at the same time.
	Send(event []byte, options SendOptions, ack func(err error)) error
}
