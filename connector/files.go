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
	"time"
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

// File represents a file connector.
type File struct {
	Name string
	Icon string // icon in SVG format
	Open OpenFileFunc
}

// FileConfig represents the configuration of a file connection.
type FileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenFileFunc represents functions that open file connections.
type OpenFileFunc func(context.Context, *FileConfig) (FileConnection, error)

// FileConnection is the interface implemented by file connections.
type FileConnection interface {
	Connection

	// Read reads the records from files and writes them to records.
	Read(files FileReader, records RecordWriter) error

	// Write writes to files the records read from records.
	Write(files FileWriter, records RecordReader) error
}

// A FileReader interface is used by file connections to read files.
type FileReader interface {

	// Reader returns a ReadCloser from which to read the file at the given
	// path and its last update time.
	// It is the caller's responsibility to close the returned reader.
	Reader(path string) (io.ReadCloser, time.Time, error)
}

// A FileWriter interface is used by file connections to write files.
type FileWriter interface {

	// Writer returns a Writer that writes to the file with the given path.
	// contentType is the file's content type.
	// It is the caller's responsibility to close the returned writer.
	Writer(path, contentType string) (io.WriteCloser, error)
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
	// Columns must be called before Record, RecordMap and RecordString.
	Columns([]Column) error

	// Record writes a record as a slice of any.
	Record([]any) error

	// RecordMap writes a record as a string to any map.
	RecordMap(record map[string]any) error

	// RecordString writes a record as a string slice.
	RecordString([]string) error

	// Timestamp sets the last modified time for all records.
	// If ts is zero time, it means that the timestamp is unknown.
	// Timestamp can be called before Record, RecordMap and RecordString.
	Timestamp(ts time.Time) error
}
