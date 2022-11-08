//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"errors"
	"fmt"
	"io"
)

var (
	ErrNoColumns                = errors.New("file does not contain columns")
	ErrEmptyColumnName          = errors.New("file contains an empty column name")
	ErrInvalidEncodedColumnName = errors.New("file contains a column name with an invalid UTF-8 encoding")
)

// A SameColumnNameError error is returned by the FileConnector.Read method when
// two columns have the same name.
type SameColumnNameError struct {
	Name string
}

func (err SameColumnNameError) Error() string {
	return fmt.Sprintf("there are two columns with the same name: %s", err.Name)
}

// A MissingIdentityColumnError error is returned by the FileConnector.Read method when
// the identity column is missing.
type MissingIdentityColumnError struct {
	Column string
}

func (err MissingIdentityColumnError) Error() string {
	return fmt.Sprintf("identity column is missing: %s", err.Column)
}

// A MissingTimestampColumnError error is returned by the FileConnector.Read method when
// the timestamp column is missing.
type MissingTimestampColumnError struct {
	Column string
}

func (err MissingTimestampColumnError) Error() string {
	return fmt.Sprintf("timestamp column is missing: %s", err.Column)
}

// FileConfig represents the configuration of a file connection.
type FileConfig struct {
	Settings []byte
	Firehose Firehose
}

// FileConnectionFunc represents functions that create new file connections.
type FileConnectionFunc func(context.Context, *FileConfig) (FileConnection, error)

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
