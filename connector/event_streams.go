//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import "context"

// EventStreamConfig represents the configuration of an event stream
// connection.
type EventStreamConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// EventStreamConnectionFunc represents functions that create new event stream
// connections.
type EventStreamConnectionFunc func(context.Context, *EventStreamConfig) (EventStreamConnection, error)

// SendOptions are the send options.
type SendOptions struct {

	// OrderKey, if not empty, ensures that all events with the same order key
	// are received in the order they were sent to the stream.
	OrderKey string
}

// EventStreamConnection is the interface implemented by event stream
// connections.
type EventStreamConnection interface {
	Connection

	// Close closes the stream.
	// A call to Receive or Send opens the stream if it is closed.
	Close() error

	// Receive receives an event from the stream. Callers call the ack function to
	// notify when and if the event has been consumed. Connector resend the event
	// if the event has not been consumed.
	//
	// Caller do not modify the event data, even temporarily, and event is not
	// retained after the ack function has been called.
	Receive() (event []byte, ack func(consumed bool), err error)

	// Send sends an event to the stream. If ack is not null, connector calls ack
	// when the event has been stored or when an error occurred.
	//
	// Send can modify the event data, but event is not retained after the ack
	// function has been called.
	Send(event []byte, options SendOptions, ack func(err error)) error
}
