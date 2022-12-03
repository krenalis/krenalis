//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package connector

import (
	"context"

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
	Name    string
	Icon    string // icon in SVG format
	Connect DatabaseConnectFunc
}

// DatabaseConfig represents the configuration of a database connection.
type DatabaseConfig struct {
	Role     Role
	Settings []byte
	Firehose Firehose
}

// DatabaseConnectFunc represents functions that create new database
// connections.
type DatabaseConnectFunc func(context.Context, *DatabaseConfig) (DatabaseConnection, error)

// DatabaseConnection is the interface implemented by database connections.
type DatabaseConnection interface {
	Connection

	// Query executes the given query and returns the resulting rows.
	Query(query string) ([]Column, Rows, error)
}

// Rows is the result of a database query. Its cursor starts before the first
// row of the result set. Use Next to advance from row to row.
type Rows interface {
	Close() error
	Err() error
	Next() bool
	Scan(dest ...any) error
}

// Column represents a column returned by a database query.
type Column struct {
	Name string
	Type types.Type
}
