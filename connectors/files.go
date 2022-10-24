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

// FileConnectionFunc represents functions that create new file connections.
type FileConnectionFunc func(context.Context, []byte, Firehose) (FileConnection, error)

// RegisterFileConnector makes a file connector available by the provided name.
// If RegisterFileConnector is called twice with the same name or if fn is nil,
// it panics.
func RegisterFileConnector(name string, fn FileConnectionFunc) {
	if fn == nil {
		panic("connectors: RegisterFileConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.files[name]; dup {
		panic("connectors: RegisterFileConnector called twice for connector " + name)
	}
	connectors.files[name] = fn

}

// FileConnection is the interface implemented by file connections.
type FileConnection interface {
	Connection

	// Read reads the records from r and calls put for each record read.
	Read(r io.Reader, put func(record []string) error) error

	// Write writes the records read from get into w.
	Write(w io.Writer, get func() ([]string, error)) error
}

// NewFileConnection returns a new file connection for the file connector with
// the given name.
func NewFileConnection(ctx context.Context, name string, settings []byte, fh Firehose) (FileConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.files[name]
	if !ok {
		return nil, fmt.Errorf("connectors: unknown file connector %q (forgotten import?)", name)
	}
	return f(ctx, settings, fh)
}
