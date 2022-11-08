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

// DatabaseConfig represents the configuration of a database connection.
type DatabaseConfig struct {
	Settings []byte
	Firehose Firehose
}

// DatabaseConnectionFunc represents functions that create new database
// connections.
type DatabaseConnectionFunc func(context.Context, *DatabaseConfig) (DatabaseConnection, error)

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
