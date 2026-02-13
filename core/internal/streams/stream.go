// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package streams

import "context"

// Connection represents a connection to an event stream.
type Connection interface {

	// Close closes the connection. When Close is called, no other calls to
	// Connection's methods should be in progress and no other shall be made.
	Close() error

	// Stream returns the stream. It waits until the stream has been created.
	// It returns an error only if ctx is canceled or if c has been closed.
	Stream(context.Context) (Stream, error)

	// WaitUp blocks until the connection is up and the stream is available.
	// It returns false if the context is canceled, the connection is closed,
	// or the connection remains down for too long.
	WaitUp(context.Context) bool
}

// Consumer receives events for a pipeline.
type Consumer interface {

	// Close closes the consumer and closes the events channel.
	Close()

	// Events returns the channel of events.
	Events() <-chan Event
}

// BatchPublisher publishes events in batches.
type BatchPublisher interface {

	// Publish adds an event to the current batch for the given topics.
	Publish(topics []string, event map[string]any, destinations []int) error

	// Done publishes all buffered events.
	//
	// If Done returns nil, all events in the batch have been successfully
	// published. If Done returns an error, no guarantees are made about whether
	// or how many events have been published.
	//
	// After Done returns, the BatchPublisher must not be reused.
	Done(ctx context.Context) error
}

// Stream is an interface for event streams.
type Stream interface {

	// Batch returns a batch publisher for the stream.
	Batch() BatchPublisher

	// Consume returns a buffered channel of the given size that streams events for
	// the specified topic. Events belonging to the same shard are sent on the
	// channel in order, ensuring per-user ordering is preserved.
	Consume(topic string, size int) Consumer
}

// Ack acknowledges an event.
type Ack func()

// Event represents an event read from the stream.
type Event struct {
	Attributes   map[string]any
	Destinations []int
	Ack          Ack
}
