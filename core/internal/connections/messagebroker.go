// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/state"
)

// messageBrokerConnection is the interface implemented by message broker
// connections. A messageBrokerConnection instance can be used for sending or
// receiving but not both.
type messageBrokerConnection interface {

	// Close closes the message broker. When Close is called, no other calls to the
	// connection's methods are in progress and no more will be made.
	Close() error

	// Receive receives an event from the message broker. Callers call the ack
	// function to notify that the event has been received. The connection resends
	// the event if not acknowledged.
	//
	// Callers must not modify the event data, even temporarily, and the event is
	// not retained after the ack function has been called.
	//
	// Receive can be used by multiple goroutines at the same time.
	Receive(ctx context.Context) (event []byte, ack func(), err error)

	// Send sends an event to the message broker. If ack is not nil, connection
	// calls ack when the event has been stored or when an error occurred.
	//
	// Send may modify the event data, but the event slice is not retained after the
	// ack function has been called.
	//
	// Send can be used by multiple goroutines at the same time.
	Send(ctx context.Context, event []byte, options connectors.SendOptions, ack func(err error)) error
}

// MessageBroker represents the broker of a message broker connection.
type MessageBroker struct {
	connector string
	closed    bool
	inner     messageBrokerConnection
}

// MessageBroker returns a message broker for the provided connection. It panics
// if connection is not a message broker connection.
//
// The caller must call the message broker's Close method when the broker is no
// longer needed.
func (c *Connections) MessageBroker(connection *state.Connection) (*MessageBroker, error) {
	broker := &MessageBroker{
		connector: connection.Connector().Code,
	}
	inner, err := connectors.RegisteredMessageBroker(connection.Connector().Code).New(&connectors.MessageBrokerEnv{
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(c.state, connection),
	})
	if err != nil {
		return nil, connectorError(err)
	}
	broker.inner = inner.(messageBrokerConnection)
	return broker, nil
}

// Close closes the message broker. When Close is called, no other calls to the
// broker's methods must be in progress, and no more calls must be made.
// It returns an *UnavailableError error if the connector returns an error.
// Close is idempotent.
func (broker *MessageBroker) Close() error {
	if broker.closed {
		return nil
	}
	broker.closed = true
	err := broker.inner.Close()
	return connectorError(err)
}

// Connector returns the name of the message broker connector.
func (broker *MessageBroker) Connector() string {
	return broker.connector
}

// Receive receives an event from the message broker. The caller can call the
// ack function to notify that the event has been received. The broker resends
// the event if not acknowledged.
//
// The caller must not modify the event data, even temporarily, and must not
// retain the event slice after the ack function has been called.
//
// If the connector returns an error, it returns a *UnavailableError error.
//
// Receive can be used by multiple goroutines at the same time.
func (broker *MessageBroker) Receive(ctx context.Context) (event []byte, ack func(), err error) {
	event, ack, err = broker.inner.Receive(ctx)
	if err != nil {
		return nil, nil, connectorError(err)
	}
	return event, ack, nil
}

// Send sends an event to the message broker. If ack is not nil, the broker
// calls ack when the event has been stored or when an error occurred.
//
// Send may modify the event data, but the event slice is not retained after the
// ack function has been called.
//
// If the connector returns an error, it returns a *UnavailableError error.
//
// Send can be used by multiple goroutines at the same time.
func (broker *MessageBroker) Send(ctx context.Context, event []byte, options connectors.SendOptions, ack func(err error)) error {
	err := broker.inner.Send(ctx, event, options, ack)
	return connectorError(err)
}
