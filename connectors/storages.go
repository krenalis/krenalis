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

// StorageConfig represents the configuration of a storage connection.
type StorageConfig struct {
	Settings []byte
	Firehose Firehose
}

// StorageConnectionFunc represents functions that create new storage
// connections.
type StorageConnectionFunc func(context.Context, *StorageConfig) (StorageConnection, error)

// RegisterStorageConnector makes a storage connector available by the provided
// name. If RegisterStorageConnector is called twice with the same name or if
// fn is nil, it panics.
func RegisterStorageConnector(name string, fn StorageConnectionFunc) {
	if fn == nil {
		panic("connectors: RegisterStorageConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.storages[name]; dup {
		panic("connectors: RegisterStorageConnector called twice for connector " + name)
	}
	connectors.storages[name] = fn
}

// StorageConnection is the interface implemented by storage connections.
type StorageConnection interface {
	Connection

	// Reader returns a Reader that reads from the given path.
	Reader(path string) (io.ReadCloser, error)

	// Writer returns a Writer that writes to the given path.
	Writer(path string) (io.WriteCloser, error)
}

// NewStorageConnection returns a new storage connection for the storage
// connector with the given name.
func NewStorageConnection(ctx context.Context, name string, conf *StorageConfig) (StorageConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.storages[name]
	if !ok {
		return nil, fmt.Errorf("connectors: unknown storage connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
