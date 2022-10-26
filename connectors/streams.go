//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package connectors

import (
	"context"
	"fmt"
	"io"
)

// StreamConnectionFunc represents functions that create new stream
// connections.
type StreamConnectionFunc func(context.Context, []byte, Firehose) (StreamConnection, error)

// RegisterStreamConnector makes a stream connector available by the provided
// name. If RegisterStreamConnector is called twice with the same name or if fn
// is nil, it panics.
func RegisterStreamConnector(name string, fn StreamConnectionFunc) {
	if fn == nil {
		panic("connectors: RegisterStreamConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.streams[name]; dup {
		panic("connectors: RegisterStreamConnector called twice for connector " + name)
	}
	connectors.streams[name] = fn
}

// StreamConnection is the interface implemented by stream connections.
type StreamConnection interface {
	Connection

	// Reader returns a Reader.
	// Callers should always call the Close method on it.
	Reader() (io.ReadCloser, error)

	// Writer returns a Writer.
	// Callers should always call the Close method on it.
	Writer() (io.WriteCloser, error)
}

// NewStreamConnection returns a new stream connection for the stream connector
// with the given name.
func NewStreamConnection(ctx context.Context, name string, settings []byte, fh Firehose) (StreamConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.streams[name]
	if !ok {
		return nil, fmt.Errorf("connectors: unknown stream connector %q (forgotten import?)", name)
	}
	return f(ctx, settings, fh)
}
