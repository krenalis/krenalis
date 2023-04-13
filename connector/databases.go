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

	"chichi/apis/types"
)

// A DatabaseQueryError error is returned from a database connector if an error
// occurs when executing a query.
type DatabaseQueryError struct {
	Message string
}

func (err *DatabaseQueryError) Error() string {
	return err.Message
}

func NewDatabaseQueryError(msg string) error {
	return &DatabaseQueryError{Message: msg}
}

// Database represents a database connector.
type Database struct {
	Name                   string
	SourceDescription      string // It should complete the sentence "Add an action to ..."
	DestinationDescription string // It should complete the sentence "Add an action to ..."
	Icon                   string // icon in SVG format

	open reflect.Value
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
	Role     Role
	Settings []byte
	Firehose Firehose
}

// OpenDatabaseFunc represents functions that open database connections.
type OpenDatabaseFunc[T DatabaseConnection] func(context.Context, *DatabaseConfig) (T, error)

// DatabaseConnection is the interface implemented by database connections.
type DatabaseConnection interface {

	// Query executes the given query and returns the resulting schema and rows.
	Query(query string) (types.Type, Rows, error)
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}
