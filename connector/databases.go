//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"
	"reflect"

	"chichi/connector/types"
)

// Database represents a database connector.
type Database struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
	ct   reflect.Type
}

// ConnectionReflectType returns the type of the value implementing the database
// connection.
func (database Database) ConnectionReflectType() reflect.Type {
	return database.ct
}

// Open opens a database connection.
func (database Database) Open(ctx context.Context, conf *DatabaseConfig) (DatabaseConnection, error) {
	out := database.open.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(conf)})
	c := out[0].Interface().(DatabaseConnection)
	err, _ := out[1].Interface().(error)
	return c, err
}

// DatabaseConfig represents the configuration of a database connection.
type DatabaseConfig struct {
	Role        Role
	Settings    []byte
	SetSettings SetSettingsFunc
}

// OpenDatabaseFunc represents functions that open database connections.
type OpenDatabaseFunc[T DatabaseConnection] func(context.Context, *DatabaseConfig) (T, error)

// DatabaseConnection is the interface implemented by database connections.
type DatabaseConnection interface {

	// Close closes the database. When Close is called, no other calls to connection
	// methods are in progress and no more will be made.
	Close() error

	// Columns returns the columns of the given table.
	Columns(table string) ([]types.Property, error)

	// Query executes the given query and returns the resulting rows and properties.
	Query(query string) (Rows, []types.Property, error)

	// Upsert creates or updates the provided rows in the specified table.
	// The columns parameter specifies the columns of the rows, including a column
	// named "id" that serves as the table's key.
	Upsert(table string, rows [][]any, columns []types.Property) error
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
