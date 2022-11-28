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

// EventStreamConnection is the interface implemented by event stream
// connections.
type EventStreamConnection interface {
	Connection

	// Close closes the stream.
	// A call to Receive or Send opens the stream if it is closed.
	Close() error

	// Commit commits a received event.
	// Caller does not retain the slice after Commit has been called.
	Commit(ctx context.Context, event []byte) error

	// Receive receives an event from the stream.
	//
	// Caller does not modify the event data, even temporarily, and event is
	// not retained after the Commit method has been called.
	Receive(ctx context.Context) (event []byte, err error)

	// Send sends an event to the stream.
	//
	// Send must not modify the event data, even temporarily, and implementations
	// must not retain event.
	Send(ctx context.Context, event []byte) error
}
