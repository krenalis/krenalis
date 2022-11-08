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

	// Read reads the records from r and write them to records.
	Read(r io.Reader, records RecordWriter) error

	// Write writes to w the records read from records.
	Write(w io.Writer, records RecordReader) error
}

// A RecordReader interface is used by file connections to read the records to
// be written.
type RecordReader interface {

	// Columns returns the columns of the records.
	Columns() []Column

	// Record returns the next record as a slice of any.
	// It returns nil and io.EOF if there are no more records.
	Record() ([]any, error)

	// RecordString returns the next record as a string slice.
	// It returns nil and io.EOF if there are no more records.
	RecordString() ([]string, error)
}

// A RecordWriter interface is used by file connections to write read records.
type RecordWriter interface {

	// Columns sets the columns of the records.
	// It must be called before calling the other methods.
	Columns([]Column) error

	// Record writes a record as a slice of any.
	Record([]any) error

	// RecordMap writes a record as a string to any map.
	RecordMap(record map[string]any) error

	// RecordString writes a record as a string slice.
	RecordString([]string) error
}
