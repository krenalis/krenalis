// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package streams

import "context"

// Stream is an interface for event streams.
//
// A Stream is safe for concurrent use by multiple goroutines, except that
// Consume must not be called concurrently with another Consume call.
type Stream interface {

	// Close closes the stream. When Close is called, no other calls to
	// Stream's methods should be in progress and no other shall be made.
	Close() error

	// WaitUp blocks until the connection is up and the stream is available.
	// It returns false if the context is canceled, the connection is closed,
	// or the connection remains down for too long.
	WaitUp(context.Context) bool

	// Batch returns a batch publisher for the stream.
	//
	// It waits until the stream has been created. It returns an error only if ctx
	//	is canceled or if the stream has been closed.
	Batch(ctx context.Context) (BatchPublisher, error)

	// Consume returns a buffered channel of the given size that streams events for
	// the specified topic. Events belonging to the same shard are sent on the
	// channel in order, ensuring per-user ordering is preserved.
	Consume(topic string, size int) Consumer
}

// Consumer receives events for a pipeline.
//
// A Consumer must not be used concurrently by multiple goroutines. Different
// Consumers may be used concurrently.
type Consumer interface {

	// Close closes the consumer and closes the events channel.
	Close()

	// Events returns the channel of events.
	//
	// It waits until the stream has been created. It returns an error only if ctx
	//	is canceled or if the stream has been closed.
	Events(ctx context.Context) (<-chan Event, error)
}

// BatchPublisher publishes events in batches.
//
// A BatchPublisher must not be used concurrently by multiple goroutines.
// Different BatchPublishers may be used concurrently.
type BatchPublisher interface {

	// Publish adds an event to the current batch for the given topic.
	// If the topic begins with "connection-", destinations contains the destination
	// pipelines the event is sent to.
	Publish(ctx context.Context, topics []string, event map[string]any, destinations []int) error

	// Done publishes all buffered events.
	//
	// If Done returns nil, all events in the batch have been successfully
	// published. If Done returns an error, no guarantees are made about whether
	// or how many events have been published.
	//
	// After Done returns, the BatchPublisher must not be reused.
	Done(ctx context.Context) error
}

// Ack acknowledges an event.
type Ack func()

// Event represents an event read from the stream.
type Event struct {
	Attributes   map[string]any
	Destinations []int
	Ack          Ack
}
