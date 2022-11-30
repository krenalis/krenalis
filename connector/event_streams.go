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

	// Receive receives an event from the stream. Callers call the ack function to
	// notify that the event has been received. The connector resends the event if
	// not acknowledged.
	//
	// Caller do not modify the event data, even temporarily, and event is not
	// retained after the ack function has been called.
	Receive() (event []byte, ack func(), err error)

	// Send sends an event to the stream. If ack is not nil, connector calls ack
	// when the event has been stored or when an error occurred.
	//
	// Send can modify the event data, but event is not retained after the ack
	// function has been called.
	Send(event []byte, options SendOptions, ack func(err error)) error
}
