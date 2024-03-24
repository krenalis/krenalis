//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"errors"
	"io"
	"reflect"

	"chichi/types"
)

// ErrSheetNotExist indicates that a file does not contain a sheet.
var ErrSheetNotExist = errors.New("sheet does not exist")

// FileInfo represents a file connector info.
type FileInfo struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format
	Extension              string // default extension of the file

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the file connector info.
func (info FileInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new file connector instance.
func (info FileInfo) New(conf *FileConfig) (File, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(File)
	err, _ := out[1].Interface().(error)
	return c, err
}

// FileConfig represents the configuration of a file connector.
type FileConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// FileNewFunc represents functions that create new file connector instances.
type FileNewFunc[T File] func(*FileConfig) (T, error)

// File is the interface implemented by file connectors.
type File interface {

	// ContentType returns the content type of the file.
	ContentType(ctx context.Context) string

	// Read reads the records from r and writes them to records. If the connector
	// has multiple sheets, sheet is the name of the sheet to be read.
	// If the provided sheet does not exist, it returns the ErrSheetNotExist error.
	Read(ctx context.Context, r io.Reader, sheet string, records RecordWriter) error

	// Write writes to w the records read from records. If the connector has
	// multiple sheets, sheet is the name of the sheet to be written to.
	Write(ctx context.Context, w io.Writer, sheet string, records RecordReader) error
}

// Sheets is implemented by file connectors that have multiple sheets.
type Sheets interface {
	File

	// Sheets returns the sheets of the file read from r.
	Sheets(ctx context.Context, r io.Reader) ([]string, error)
}

// A RecordReader interface is used by file connectors to read the records to be
// written.
type RecordReader interface {

	// Ack acknowledges the processing of the record with the given GID.
	// err is the error occurred processing the record, if any.
	Ack(gid int, err error)

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record as a slice of any with its GID.
	// It returns 0, nil, and io.EOF if there are no more records.
	//
	// After a record has been read and processed, the caller should call Ack
	// to acknowledge the processing of the record.
	Record(ctx context.Context) (gid int, record []any, err error)
}

// A RecordWriter interface is used by file connectors to write read records.
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
}
