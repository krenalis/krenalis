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

	// Read reads the records from r and calls put for each record read.
	Read(r io.Reader, put func(record []string) error) error

	// Write writes the records read from get into w.
	// get should return io.EOF when there are no more records.
	Write(w io.Writer, get func() ([]string, error)) error
}
