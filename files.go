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

	"github.com/meergo/meergo/core/types"
)

// ErrSheetNotExist indicates that a file does not contain a sheet.
var ErrSheetNotExist = errors.New("sheet does not exist")

// FileInfo represents a file connector info.
type FileInfo struct {
	Code          string
	Label         string
	Categories    Categories // categories
	AsSource      *AsSourceFile
	AsDestination *AsDestinationFile
	HasSheets     bool
	TimeLayouts   TimeLayouts // layouts for time values. If left empty, it is ISO 8601.
	Extension     string      // default extension of the file

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
func (info FileInfo) New(env *FileEnv) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(env)})
	c := out[0].Interface()
	err, _ := reflect.TypeAssert[error](out[1])
	return c, err
}

// FileEnv is the environment for a file connector.
type FileEnv struct {

	// Settings is the raw settings data.
	Settings []byte

	// SetSettings is the function used to update the settings.
	SetSettings SetSettingsFunc
}

// FileNewFunc represents functions that create new file connector instances.
type FileNewFunc[T any] func(*FileEnv) (T, error)

// A RecordReader interface is used by file connectors to read records
// before they are written.
type RecordReader interface {

	// Columns returns the columns of the records as properties.
	Columns() []types.Property

	// Record returns the next record. The keys of the record are column names.
	// A record may be empty or contain only a subset of columns.
	// It returns nil and io.EOF if there are no more records.
	Record(ctx context.Context) (map[string]any, error)
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
