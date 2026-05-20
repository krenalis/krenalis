// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package streamstest provides mocks for the streams interfaces.
package streamstest

import (
	"context"

	"github.com/krenalis/krenalis/core/internal/streams"
)

// Stream is a mock for streams.Stream.
type Stream struct {
	ConsumeFunc func(string, int) streams.Consumer
	WaitUpFunc  func(context.Context) bool
	BatchFunc   func(context.Context) (streams.BatchPublisher, error)
	CloseFunc   func() error

	WaitUpValue  bool
	BatchValue   streams.BatchPublisher
	BatchError   error
	ConsumeValue streams.Consumer
	EventsError  error
}

func (s *Stream) Batch(ctx context.Context) (streams.BatchPublisher, error) {
	if s.BatchFunc != nil {
		return s.BatchFunc(ctx)
	}
	if s.BatchError != nil {
		return nil, s.BatchError
	}
	if s.BatchValue != nil {
		return s.BatchValue, nil
	}
	return &batchPublisher{}, nil
}

func (s *Stream) Close() error {
	if s.CloseFunc != nil {
		return s.CloseFunc()
	}
	return nil
}

func (s *Stream) Consume(topic string, size int) streams.Consumer {
	if s.ConsumeFunc != nil {
		return s.ConsumeFunc(topic, size)
	}
	if s.ConsumeValue != nil {
		return s.ConsumeValue
	}
	return &consumer{EventsCh: closedEvents, EventsError: s.EventsError}
}

func (s *Stream) WaitUp(ctx context.Context) bool {
	if s.WaitUpFunc != nil {
		return s.WaitUpFunc(ctx)
	}
	return s.WaitUpValue
}

// batchPublisher is a mock for streams.BatchPublisher.
type batchPublisher struct {
	PublishFunc func(context.Context, []string, map[string]any, []int) error
	DoneFunc    func(context.Context) error
}

// Publish implements streams.BatchPublisher.
func (b *batchPublisher) Publish(ctx context.Context, topics []string, attributes map[string]any, destinations []int) error {
	if b.PublishFunc != nil {
		return b.PublishFunc(ctx, topics, attributes, destinations)
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
	EventsFunc  func(context.Context) (<-chan streams.Event, error)
	EventsCh    <-chan streams.Event
	EventsError error
	CloseFunc   func()
}

// Close implements streams.Consumer.
func (c *consumer) Close() {
	if c.CloseFunc != nil {
		c.CloseFunc()
	}
}

// Events implements streams.Consumer.
func (c *consumer) Events(ctx context.Context) (<-chan streams.Event, error) {
	if c.EventsFunc != nil {
		return c.EventsFunc(ctx)
	}
	if c.EventsError != nil {
		return nil, c.EventsError
	}
	if c.EventsCh != nil {
		return c.EventsCh, nil
	}
	return closedEvents, nil
}

// closedEvents provides a reusable closed channel for consumers with no events.
var closedEvents = func() <-chan streams.Event {
	ch := make(chan streams.Event)
	close(ch)
	return ch
}()
