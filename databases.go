//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package chichi

import (
	"context"
	"reflect"

	"github.com/open2b/chichi/types"
)

// DatabaseInfo represents a database connector info.
type DatabaseInfo struct {
	Name        string
	SampleQuery string // sample query
	Icon        string // icon in SVG format

	newFunc reflect.Value
	ct      reflect.Type
}

// ReflectType returns the type of the value implementing the database
// connector info.
func (info DatabaseInfo) ReflectType() reflect.Type {
	return info.ct
}

// New returns a new database connector instance.
func (info DatabaseInfo) New(conf *DatabaseConfig) (Database, error) {
	out := info.newFunc.Call([]reflect.Value{reflect.ValueOf(conf)})
	c := out[0].Interface().(Database)
	err, _ := out[1].Interface().(error)
	return c, err
}

// DatabaseConfig represents the configuration of a database connector.
type DatabaseConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// DatabaseNewFunc represents functions that create new database connector
// instances.
type DatabaseNewFunc[T Database] func(*DatabaseConfig) (T, error)

// Database is the interface implemented by database connectors.
type Database interface {

	// Close closes the database. When Close is called, no other calls to
	// connector's methods are in progress and no more will be made.
	Close() error

	// Columns returns the columns of the given table.
	Columns(ctx context.Context, table string) ([]types.Property, error)

	// Query executes the given query and returns the resulting rows and columns.
	Query(ctx context.Context, query string) (Rows, []types.Property, error)

	// Upsert creates or updates the provided rows in the specified table.
	// The columns parameter specifies the columns of the rows, including a column
	// named "id" that serves as the table's key. If a column's value is not
	// specified in a row, the default column value is used.
	Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
