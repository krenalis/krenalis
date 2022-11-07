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
)

// FileConnectionFunc represents functions that create new file connections.
type FileConnectionFunc func(context.Context, []byte, Firehose) (FileConnection, error)

// FileConnection is the interface implemented by file connections.
type FileConnection interface {
	Connection

	// ContentType returns the content type of the data to write.
	ContentType() string

	// Read reads the records from r.
	Read(r io.Reader) error

	// Write writes the records to w.
	Write(w io.Writer) error
}
