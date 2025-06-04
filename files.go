//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"errors"
	"reflect"

	"github.com/meergo/meergo/types"
)

// ErrSheetNotExist indicates that a file does not contain a sheet.
var ErrSheetNotExist = errors.New("sheet does not exist")

// FileInfo represents a file connector info.
type FileInfo struct {
	Name          string
	Categories    Categories // categories
	AsSource      *AsSourceFile
	AsDestination *AsDestinationFile
	HasSheets     bool
	TimeLayouts   TimeLayouts // layouts for time values. If left empty, it is ISO 8601.
	Extension     string      // default extension of the file
	Icon          string      // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// AsSourceFile represents the specific information of a file connector used as
// a source.
type AsSourceFile struct {
	HasSettings   bool
	Documentation ConnectorRoleDocumentation
}

// AsDestinationFile represents the specific information of a file connector
// used as a destination.
type AsDestinationFile struct {
	HasSettings   bool
	Documentation ConnectorRoleDocumentation
}

// ReflectType returns the type of the value implementing the file connector info.
func (info FileInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new file connector instance.
func (info FileInfo) New(conf *FileConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// FileConfig represents the configuration of a file connector.
type FileConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// FileNewFunc represents functions that create new file connector instances.
type FileNewFunc[T any] func(*FileConfig) (T, error)

// A RecordReader interface is used by file connectors to read the records to be
// written.
type RecordReader interface {

	// Ack acknowledges the processing of the record with the given identifier.
	// err is the error occurred processing the record, if any.
	Ack(id string, err error)

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record with its ack ID. The keys of record represent
	// column names. A record may be empty or contain only a subset of columns.
	// It returns "", nil, and io.EOF if there are no more records.
	//
	// After a record has been read and processed, the caller should call Ack
	// to acknowledge the processing of the record.
	Record(ctx context.Context) (ackID string, record map[string]any, err error)
}

// A RecordWriter interface is used by file connectors to write read records.
type RecordWriter interface {

	// Columns sets the columns of the records as properties.
	// Columns must be called before Record, RecordSlice, and RecordStrings.
	Columns(columns []types.Property) error

	// Issue reports an issue encountered during file reading that did not
	// prevent the file from being processed. For instance, an issue might occur
	// if a column is excluded due to an unsupported data type.
	//
	// The format and its arguments are formatted using the syntax and rules of
	// fmt.Sprintf.
	//
	// Subsequent calls will append new issues to the ones previously reported.
	Issue(format string, a ...any)

	// Record writes a record represented as a string to any map.
	// The record's length must equal to the number of columns.
	Record(record map[string]any) error

	// RecordSlice writes a record represented as a slice of any.
	// The record's length must equal to the number of columns.
	RecordSlice(record []any) error

	// RecordStrings writes a record represented as a string slice.
	// The record's length must be less than or equal to the number of columns, and
	// record cannot be nil.
	//
	// RecordStrings may modify the elements of the record.
	RecordStrings(record []string) error
}
