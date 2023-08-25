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
	"reflect"
	"time"

	"chichi/connector/types"
)

// File represents a file connector.
type File struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format
	Extension              string // default extension of the file

	open reflect.Value
	ct   reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the file
// connection.
func (file File) ConnectionReflectType() reflect.Type {
	return file.ct
}

// Open opens a file connection.
func (file File) Open(conf *FileConfig) (FileConnection, error) {
	out := file.open.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(FileConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// FileConfig represents the configuration of a file connection.
type FileConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// OpenFileFunc represents functions that open file connections.
type OpenFileFunc[T FileConnection] func(*FileConfig) (T, error)

// FileConnection is the interface implemented by file connections.
type FileConnection interface {

	// ContentType returns the content type of the file.
	ContentType(ctx context.Context) string

	// Read reads the records from r and writes them to records. If the connection
	// has multiple sheets, sheet is the name of the sheet to be read.
	Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error

	// Write writes to w the records read from records. If the connection has
	// multiple sheets, sheet is the name of the sheet to be written to.
	Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
}

// Sheets is implemented by file connectors that have multiple sheets.
type Sheets interface {
	FileConnection

	// Sheets returns the sheets of the file read from r.
	Sheets(ctx context.Context, r io.Reader) ([]string, error)
}

// A RecordReader interface is used by file connections to read the records to
// be written.
type RecordReader interface {

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record as a slice of any.
	// It returns nil and io.EOF if there are no more records.
	Record() ([]any, error)
}

// A RecordWriter interface is used by file connections to write read records.
type RecordWriter interface {

	// Columns sets the columns of the records as properties.
	// Columns must be called before Record, RecordMap and RecordString.
	Columns([]types.Property) error

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
