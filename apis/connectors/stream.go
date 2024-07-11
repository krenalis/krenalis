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
	"github.com/meergo/meergo/apis/state"
)

// SendOptions are the stream's Send options.
type SendOptions = meergo.SendOptions

// Stream represents the stream of a stream connection.
type Stream struct {
	closed bool
	inner  meergo.Stream
}

// Stream returns a stream for the provided connection. It panics if connection
// is not a stream connection.
//
// The caller must call the stream's Close method when the stream is no
// longer needed.
func (connectors *Connectors) Stream(connection *state.Connection) (*Stream, error) {
	stream := &Stream{}
	var err error
	stream.inner, err = meergo.RegisteredStream(connection.Connector().Name).New(&meergo.StreamConfig{
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, connection),
	})
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// Close closes the stream. When Close is called, no other calls to the
// stream's methods must be in progress, and no more calls must be made.
// Close is idempotent.
func (stream *Stream) Close() error {
	if stream.closed {
		return nil
	}
	stream.closed = true
	return stream.inner.Close()
}

// Receive receives an event from the stream. The caller can call the ack
// function to notify that the event has been received. The stream resends the
// event if not acknowledged.
//
// The caller must not modify the event data, even temporarily, and must not
// retain the event slice after the ack function has been called.
//
// Receive can be used by multiple goroutines at the same time.
func (stream *Stream) Receive(ctx context.Context) (event []byte, ack func(), err error) {
	return stream.inner.Receive(ctx)
}

// Send sends an event to the stream. If ack is not nil, the stream calls ack
// when the event has been stored or when an error occurred.
//
// Send can modify the event data, but the event slice is not retained after the
// ack function has been called.
//
// Send can be used by multiple goroutines at the same time.
func (stream *Stream) Send(ctx context.Context, event []byte, options SendOptions, ack func(err error)) error {
	return stream.inner.Send(ctx, event, options, ack)
}
