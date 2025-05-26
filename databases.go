//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"reflect"

	"github.com/meergo/meergo/types"
)

// DatabaseInfo represents a database connector info.
type DatabaseInfo struct {
	Name          string
	SampleQuery   string      // sample query
	TimeLayouts   TimeLayouts // layouts for time values. If left empty, it is ISO 8601.
	Icon          string      // icon in SVG format
	Documentation ConnectorDocumentation

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the database
// connector info.
func (info DatabaseInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new database connector instance.
func (info DatabaseInfo) New(conf *DatabaseConfig) (any, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface()
	err, _ := out[1].Interface().(error)
	return c, err
}

// DatabaseConfig represents the configuration of a database connector.
type DatabaseConfig struct {
	Settings    []byte
	SetSettings SetSettingsFunc
}

// DatabaseNewFunc represents functions that create new database connector
// instances.
type DatabaseNewFunc[T any] func(*DatabaseConfig) (T, error)

// Table represents a database table.
type Table struct {
	Name    string
	Columns []Column
	Keys    []string
}

// Column represents a database table column. If Type is invalid, Issue
// describes the problem, and the other fields are not meaningful.
type Column struct {
	Name     string     // column name
	Type     types.Type // data type of the column
	Nullable bool       // true if the column can contain NULL values
	Writable bool       // true if the column is writable
	Issue    string     // issue message
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
