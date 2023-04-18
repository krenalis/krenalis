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
	"reflect"
	"time"

	"chichi/apis/types"
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
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
	ct   reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the file
// connection.
func (file File) ConnectionReflectType() reflect.Type {
	return file.ct
}

// Open opens a file connection.
func (file File) Open(ctx context.Context, conf *FileConfig) (FileConnection, error) {
	out := file.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
	c := out[0].Interface().(FileConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// FileConfig represents the configuration of a file connection.
type FileConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenFileFunc represents functions that open file connections.
type OpenFileFunc[T FileConnection] func(context.Context, *FileConfig) (T, error)

// FileConnection is the interface implemented by file connections.
type FileConnection interface {

	// ContentType returns the content type of the file.
	ContentType() string

	// Path returns the path of the file.
	Path() string

	// Read reads the records from r, with their last update time, and writes
	// them to records.
	Read(r io.Reader, updateTime time.Time, records RecordWriter) error

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

// Column represents a column returned by RecordWriter.Columns.
type Column struct {
	Name string
	Type types.Type
}
