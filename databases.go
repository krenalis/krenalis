//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package meergo

import (
	"context"
	"reflect"

	"github.com/meergo/meergo/types"
)

// DatabaseInfo represents a database connector info.
type DatabaseInfo struct {
	Name        string
	SampleQuery string      // sample query
	TimeLayouts TimeLayouts // layouts for time values. If left empty, it is ISO 8601.
	Icon        string      // icon in SVG format

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
	// If a column type is not supported, it returns a *UnsupportedColumnTypeError
	// error.
	Columns(ctx context.Context, table string) ([]types.Property, error)

	// LastChangeTimeCondition returns the query condition used for the
	// last_change_time placeholder in the form "column >= value" or, if column is
	// empty, a true value.
	//
	// column, if not empty, is the name of the column, typ is its type (can be
	// DateTime, Date, JSON, or Text), and value is the value for the condition:
	//
	//   - for DateTime and Date types, it is a time.Time value.
	//   - for JSON and Text types, it is a string value.
	//
	// For example, it could return `"updated_at" >= '2024-06-18 16:12:25'` or
	// `TRUE`.
	LastChangeTimeCondition(column string, typ types.Type, value any) string

	// Query executes the given query and returns the resulting rows and columns.
	// If a column type is not supported, it returns a *UnsupportedColumnTypeError
	// error.
	Query(ctx context.Context, query string) (Rows, []types.Property, error)

	// Upsert inserts or updates the rows provided in the specified table. If a row
	// with the same key value already exists in the database, it updates that row;
	// otherwise, it inserts a new one. When inserting or updating a row, if a
	// column is not provided, the column's default value is used.
	Upsert(ctx context.Context, table Table, rows []map[string]any) error
}

// Table represents a table.
type Table struct {
	Name    string
	Columns []types.Property
	Key     string
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
