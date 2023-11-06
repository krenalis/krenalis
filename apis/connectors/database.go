//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"

	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// Database represents the database of a database connection.
type Database struct {
	closed bool
	inner  _connector.DatabaseConnection
	err    error
}

// Database returns a database for the provided connection. Errors are deferred
// until a database's method is called. It panics if connection is not a
// database connection.
//
// The caller must call the database's Close method when the database is no
// longer needed.
func (connectors *Connectors) Database(connection *state.Connection) *Database {
	database := &Database{}
	database.inner, database.err = _connector.RegisteredDatabase(connection.Connector().Name).New(&_connector.DatabaseConfig{
		Role:        _connector.Role(connection.Role),
		Settings:    connection.Settings,
		SetSettings: setSettingsFunc(connectors.db, connection),
	})
	return database
}

// Close closes the database. When Close is called, no other calls to the
// database's methods must be in progress, and no more calls must be made.
// Close is idempotent.
func (database *Database) Close() error {
	if database.err != nil {
		return database.err
	}
	if database.closed {
		return nil
	}
	database.closed = true
	return database.inner.Close()
}

// Columns returns the columns of the provided table.
func (database *Database) Columns(ctx context.Context, table string) ([]types.Property, error) {
	if database.err != nil {
		return nil, database.err
	}
	return database.inner.Columns(ctx, table)
}

// Query executes a query and returns the resulting rows.
func (database *Database) Query(ctx context.Context, query string) (*Rows, error) {
	if database.err != nil {
		return nil, database.err
	}
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	return newRows(rows, columns), nil
}

// Upsert creates or updates the provided rows in the specified table.
// The columns parameter specifies the columns of the rows, including a column
// named "id" that serves as the table's key.
func (database *Database) Upsert(ctx context.Context, table string, rows [][]any, columns []types.Property) error {
	if database.err != nil {
		return database.err
	}
	return database.inner.Upsert(ctx, table, rows, columns)
}

// Rows is the result of a query.
type Rows struct {
	rows    _connector.Rows
	columns []types.Property
	dst     []any
	closed  bool
}

func newRows(rows _connector.Rows, columns []types.Property) *Rows {
	rs := &Rows{
		rows:    rows,
		columns: columns,
		dst:     make([]any, len(columns)),
	}
	return rs
}

// Close closes the rows. Close is idempotent.
func (rs *Rows) Close() error {
	if rs.closed {
		return nil
	}
	rs.closed = true
	return rs.rows.Close()
}

// Columns returns the columns.
func (rs *Rows) Columns() []types.Property {
	return rs.columns
}

// Err returns the error encountered during iteration, if any. It can be called
// after an explicit or implicit Close
func (rs *Rows) Err() error {
	return rs.rows.Err()
}

// Next prepares the next result row for reading with the Scan method.
// It returns true on success, signaling the availability of a result row, or
// false in cases where there is no next result row or an error occurred during
// preparation.
//
// Every call to Scan, even the first one, must be preceded by a call to Next.
func (rs *Rows) Next() bool {
	return rs.rows.Next()
}

// Scan returns the current row.
func (rs *Rows) Scan() (map[string]any, error) {
	row := make(map[string]any, len(rs.columns))
	for i, p := range rs.columns {
		rs.dst[i] = databaseScanValue{property: p, row: row}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, err
	}
	return row, nil
}

// databaseScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type databaseScanValue struct {
	property types.Property
	row      map[string]any
}

func (sv databaseScanValue) Scan(src any) error {
	p := sv.property
	value, err := normalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
	if err != nil {
		return err
	}
	sv.row[sv.property.Name] = value
	return nil
}
