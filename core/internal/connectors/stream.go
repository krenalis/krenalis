//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/internal/state"
)

// Stream is the interface implemented by stream connectors.
// A Stream value can be used for sending or receiving but not both.
type streamConnector interface {

	// Close closes the stream. When Close is called, no other calls to the
	// connector's methods are in progress and no more will be made.
	Close() error

	// Receive receives an event from the stream. Callers call the ack function to
	// notify that the event has been received. The connector resends the event if
	// not acknowledged.
	//
	// Callers must not modify the event data, even temporarily, and the event is
	// not retained after the ack function has been called.
	//
	// Receive can be used by multiple goroutines at the same time.
	Receive(ctx context.Context) (event []byte, ack func(), err error)

	// Send sends an event to the stream. If ack is not nil, connector calls ack
	// when the event has been stored or when an error occurred.
	//
	// Send may modify the event data, but the event slice is not retained after the
	// ack function has been called.
	//
	// Send can be used by multiple goroutines at the same time.
	Send(ctx context.Context, event []byte, options meergo.SendOptions, ack func(err error)) error
}

// Stream represents the stream of a stream connection.
type Stream struct {
	connector string
	closed    bool
	inner     streamConnector
}

// Stream returns a stream for the provided connection. It panics if connection
// is not a stream connection.
//
// The caller must call the stream's Close method when the stream is no
// longer needed.
func (connectors *Connectors) Stream(connection *state.Connection) (*Stream, error) {
	stream := &Stream{
		connector: connection.Connector().Name,
	}
	inner, err := meergo.RegisteredStream(connection.Connector().Name).New(&meergo.StreamEnv{
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, connection),
	})
	if err != nil {
		return nil, connectorError(err)
	}
	stream.inner = inner.(streamConnector)
	return stream, nil
}

// Close closes the stream. When Close is called, no other calls to the
// stream's methods must be in progress, and no more calls must be made.
// It returns an *UnavailableError error if the connector returns an error.
// Close is idempotent.
func (stream *Stream) Close() error {
	if stream.closed {
		return nil
	}
	stream.closed = true
	err := stream.inner.Close()
	return connectorError(err)
}

// Connector returns the name of the stream connector.
func (stream *Stream) Connector() string {
	return stream.connector
}

// Receive receives an event from the stream. The caller can call the ack
// function to notify that the event has been received. The stream resends the
// event if not acknowledged.
//
// The caller must not modify the event data, even temporarily, and must not
// retain the event slice after the ack function has been called.
//
// If the connector returns an error, it returns a *UnavailableError error.
//
// Receive can be used by multiple goroutines at the same time.
func (stream *Stream) Receive(ctx context.Context) (event []byte, ack func(), err error) {
	event, ack, err = stream.inner.Receive(ctx)
	if err != nil {
		return nil, nil, connectorError(err)
	}
	return event, ack, nil
}

// Send sends an event to the stream. If ack is not nil, the stream calls ack
// when the event has been stored or when an error occurred.
//
// Send may modify the event data, but the event slice is not retained after the
// ack function has been called.
//
// If the connector returns an error, it returns a *UnavailableError error.
//
// Send can be used by multiple goroutines at the same time.
func (stream *Stream) Send(ctx context.Context, event []byte, options meergo.SendOptions, ack func(err error)) error {
	err := stream.inner.Send(ctx, event, options, ack)
	return connectorError(err)
}
