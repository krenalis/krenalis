// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package streams

import "context"

// Stream is a transport-agnostic interface for event streams.
type Stream interface {
	Consumer
	Publisher
	Close() error
}

// Event represents a transport-agnostic event.
type Event struct {
	Attributes map[string]any
	Ack        Ack
}

// Ack acknowledges an event; errors are handled internally by the transport.
type Ack func()

// Consumer receives events for a pipeline.
type Consumer interface {
	Consume(ctx context.Context, pipeline, size int) (<-chan Event, error)
}

// Batch publishes events and waits for completion.
type Batch interface {
	Publish(pipelines []int, attributes map[string]any) error
	Done(ctx context.Context) error
}

// Publisher creates publish batches.
type Publisher interface {
	NewBatch() Batch
}
