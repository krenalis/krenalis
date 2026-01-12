// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package streamstest provides mocks for the streams interfaces.
package streamstest

import (
	"context"

	"github.com/meergo/meergo/core/internal/streams"
)

// Connection is a mock for streams.Connection.
type Connection struct {
	CloseFunc  func() error
	StreamFunc func(context.Context) (streams.Stream, error)
	WaitUpFunc func(context.Context) bool

	StreamValue streams.Stream
	WaitUpValue bool
}

// Close implements streams.Connection.
func (c *Connection) Close() error {
	if c.CloseFunc != nil {
		return c.CloseFunc()
	}
	return nil
}

// Stream implements streams.Connection.
func (c *Connection) Stream(ctx context.Context) (streams.Stream, error) {
	if c.StreamFunc != nil {
		return c.StreamFunc(ctx)
	}
	if c.StreamValue != nil {
		return c.StreamValue, nil
	}
	return &Stream{}, nil
}

// WaitUp implements streams.Connection.
func (c *Connection) WaitUp(ctx context.Context) bool {
	if c.WaitUpFunc != nil {
		return c.WaitUpFunc(ctx)
	}
	return c.WaitUpValue
}

// Stream is a mock for streams.Stream.
type Stream struct {
	BatchFunc   func() streams.BatchPublisher
	ConsumeFunc func(pipeline, size int) streams.Consumer

	BatchValue   streams.BatchPublisher
	ConsumeValue streams.Consumer
}

// Batch implements streams.Stream.
func (s *Stream) Batch() streams.BatchPublisher {
	if s.BatchFunc != nil {
		return s.BatchFunc()
	}
	if s.BatchValue != nil {
		return s.BatchValue
	}
	return &batchPublisher{}
}

// Consume implements streams.Stream.
func (s *Stream) Consume(pipeline, size int) streams.Consumer {
	if s.ConsumeFunc != nil {
		return s.ConsumeFunc(pipeline, size)
	}
	if s.ConsumeValue != nil {
		return s.ConsumeValue
	}
	return &consumer{EventsCh: closedEvents}
}

// batchPublisher is a mock for streams.BatchPublisher.
type batchPublisher struct {
	PublishFunc func(pipelines []int, attributes map[string]any) error
	DoneFunc    func(context.Context) error
}

// Publish implements streams.BatchPublisher.
func (b *batchPublisher) Publish(pipelines []int, attributes map[string]any) error {
	if b.PublishFunc != nil {
		return b.PublishFunc(pipelines, attributes)
	}
	return nil
}

// Done implements streams.BatchPublisher.
func (b *batchPublisher) Done(ctx context.Context) error {
	if b.DoneFunc != nil {
		return b.DoneFunc(ctx)
	}
	return nil
}

// consumer is a mock for streams.Consumer.
type consumer struct {
	CloseFunc func()
	EventsCh  <-chan streams.Event
}

// Close implements streams.Consumer.
func (c *consumer) Close() {
	if c.CloseFunc != nil {
		c.CloseFunc()
	}
}

// Events implements streams.Consumer.
func (c *consumer) Events() <-chan streams.Event {
	if c.EventsCh != nil {
		return c.EventsCh
	}
	return closedEvents
}

// closedEvents provides a reusable closed channel for consumers with no events.
var closedEvents = func() <-chan streams.Event {
	ch := make(chan streams.Event)
	close(ch)
	return ch
}()
