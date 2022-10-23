//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2002-2022 Open2b
//

package connectors

import (
	"context"
	"fmt"
)

// DatabaseConfig represents the configuration of a database connection.
type DatabaseConfig struct {
	Settings []byte
	Firehose Firehose
}

// DatabaseConnectionFunc represents functions that create new database
// connections.
type DatabaseConnectionFunc func(context.Context, *DatabaseConfig) (DatabaseConnection, error)

// RegisterDatabaseConnector makes a database connector available by the
// provided name. If RegisterDatabaseConnector is called twice with the same
// name or if fn is nil, it panics.
func RegisterDatabaseConnector(name string, f DatabaseConnectionFunc) {
	if f == nil {
		panic("connectors: RegisterDatabaseConnector function is nil")
	}
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	if _, dup := connectors.databases[name]; dup {
		panic("connectors: RegisterDatabaseConnector called twice for connector " + name)
	}
	connectors.databases[name] = f
}

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
	Type string
}

// NewDatabaseConnection returns a new database connection for the database
// connector with the given name.
func NewDatabaseConnection(ctx context.Context, name string, conf *DatabaseConfig) (DatabaseConnection, error) {
	connectorsMu.Lock()
	defer connectorsMu.Unlock()
	f, ok := connectors.databases[name]
	if !ok {
		return nil, fmt.Errorf("connectors: unknown database connector %q (forgotten import?)", name)
	}
	return f(ctx, conf)
}
