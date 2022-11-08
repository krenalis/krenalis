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

// StreamConfig represents the configuration of a stream connection.
type StreamConfig struct {
	Settings []byte
	Firehose Firehose
}

// StreamConnectionFunc represents functions that create new stream
// connections.
type StreamConnectionFunc func(context.Context, *StreamConfig) (StreamConnection, error)

// StreamConnection is the interface implemented by stream connections.
type StreamConnection interface {
	Connection

	// Reader returns a ReadCloser from which to read the data and its last update time.
	// It is the caller's responsibility to close the returned reader.
	Reader() (io.ReadCloser, time.Time, error)

	// Write writes the data read from p. contentType is the data's content type.
	Write(p io.Reader, contentType string) error
}
