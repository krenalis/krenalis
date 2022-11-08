//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"io"
	"time"
)

// StorageConfig represents the configuration of a storage connection.
type StorageConfig struct {
	Direction Direction
	Settings  []byte
	Firehose  Firehose
}

// StorageConnectionFunc represents functions that create new storage
// connections.
type StorageConnectionFunc func(context.Context, *StorageConfig) (StorageConnection, error)

// StorageConnection is the interface implemented by storage connections.
type StorageConnection interface {
	Connection

	// Reader returns a ReadCloser from which to read the data and its last update time.
	// It is the caller's responsibility to close the returned reader.
	Reader() (io.ReadCloser, time.Time, error)

	// Write writes the data read from p. contentType is the data's content type.
	Write(p io.Reader, contentType string) error
}
