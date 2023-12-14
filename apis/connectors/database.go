//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
	"context"
	"errors"
	"fmt"
	"time"

	"chichi/apis/state"
	_connector "chichi/connector"
	"chichi/connector/types"
)

// Database represents the database of a database connection.
type Database struct {
	closed    bool
	connector int
	inner     _connector.DatabaseConnection
	err       error
}

// Database returns a database for the provided connection. Errors are deferred
// until a database's method is called. It panics if connection is not a
// database connection.
//
// The caller must call the database's Close method when the database is no
// longer needed.
func (connectors *Connectors) Database(connection *state.Connection) *Database {
	database := &Database{
		connector: connection.Connector().ID,
	}
	database.inner, database.err = _connector.RegisteredDatabase(connection.Connector().Name).New(&_connector.DatabaseConfig{
		Role:        _connector.Role(connection.Role),
		Settings:    connection.Settings,
		SetSettings: setSettingsFunc(connectors.state, connection),
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

// Records executes a query and returns an iterator to iterate over the
// database's records, conforming to the provided schema.
//
// The query must be such that its execution returns a column named "id" (the
// identity column) with type Int, Uint, UUID, or Text. Additionally, if the
// query execution returns a column named "timestamp", that column is considered
// the timestamp column and must have the DateTime type.
//
// If the provided schema, which must be valid, does not conform to the query's
// results schema, it returns a *SchemaError error.
func (database *Database) Records(ctx context.Context, query string, schema types.Type) (Records, error) {
	if database.err != nil {
		return nil, database.err
	}
	if !schema.Valid() {
		return nil, fmt.Errorf("schema is not valid")
	}
	// Execute the query.
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var records Records
	defer func() {
		if records == nil {
			_ = rows.Close()
		}
	}()
	// Validate the identity and timestamp columns.
	var hasIdentityColumn bool
	for _, c := range columns {
		switch c.Name {
		case "id":
			switch k := c.Type.Kind(); k {
			case types.IntKind, types.UintKind, types.UUIDKind, types.TextKind:
			default:
				return nil, &SchemaError{fmt.Sprintf(`identity column "id" has type %s instead of Int, Uint, UUID, or Text`, c.Type)}
			}
			hasIdentityColumn = true
		case "timestamp":
			if c.Type.Kind() != types.DateTimeKind {
				return nil, &SchemaError{fmt.Sprintf(`timestamp column "timestamp" has type %s instead of DateTime`, c.Type)}
			}
		}
	}
	if !hasIdentityColumn {
		return nil, &SchemaError{`there is no identity column "id"`}
	}
	// Check that schema is compatible with the query's schema.
	querySchema, err := types.ObjectOf(columns)
	if err != nil {
		return nil, fmt.Errorf("connector %d has returned invalid columns: %s", database.connector, err)
	}
	err = checkConformity("", schema, querySchema)
	if err != nil {
		return nil, err
	}
	// Return the records.
	records = newDatabaseRecords(rows, columns, schema.Properties())
	return records, nil
}

// Upsert creates or updates the provided rows in the specified table.
// The columns parameter specifies the columns of the rows, including a column
// named "id" that serves as the table's key. If a column's value is not
// specified in a row, the default column value is used.
func (database *Database) Upsert(ctx context.Context, table string, rows []map[string]any, columns []types.Property) error {
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
	for i, c := range rs.columns {
		rs.dst[i] = databaseScanValue{property: c, record: &Record{Properties: row}}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, err
	}
	return row, nil
}

// databaseRecords implements the Records interface for databases.
type databaseRecords struct {
	columns    []types.Property
	rows       _connector.Rows
	propertyOf map[string]types.Property
	dst        []any
	err        error
	closed     bool
}

func newDatabaseRecords(rows _connector.Rows, columns, properties []types.Property) *databaseRecords {
	records := databaseRecords{
		columns:    columns,
		rows:       rows,
		dst:        make([]any, len(columns)),
		propertyOf: make(map[string]types.Property, len(properties)),
	}
	for _, p := range properties {
		records.propertyOf[p.Name] = p
	}
	return &records
}

func (r *databaseRecords) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	err := r.rows.Close()
	if err != nil && r.err == nil {
		r.err = err
	}
	return err
}

func (r *databaseRecords) Err() error {
	return r.err
}

func (r *databaseRecords) For(yield func(Record) error) error {
	if r.closed {
		r.err = errors.New("connectors: For called on a closed Records")
		return nil
	}
	defer r.Close()
	for r.rows.Next() {
		record := Record{
			Properties: make(map[string]any, len(r.propertyOf)),
		}
		for i, c := range r.columns {
			sv := databaseScanValue{
				property: r.propertyOf[c.Name],
				record:   &record,
			}
			sv.property.Name = c.Name
			if c.Name == "id" {
				sv.identityType = c.Type
			}
			r.dst[i] = sv
		}
		if err := r.rows.Scan(r.dst...); err != nil {
			r.err = err
			return nil
		}
		if err := yield(record); err != nil {
			return err
		}
	}
	if err := r.rows.Err(); err != nil {
		r.err = err
	}
	return nil
}

// databaseScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type databaseScanValue struct {
	property     types.Property
	identityType types.Type
	record       *Record
}

func (sv databaseScanValue) Scan(src any) error {
	p := sv.property
	if p.Name == "" {
		return nil
	}
	if p.Type.Valid() {
		value, err := normalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
		if err != nil {
			return err
		}
		sv.record.Properties[p.Name] = value
	}
	switch p.Name {
	case "id":
		if !sv.identityType.Valid() {
			return nil
		}
		if src == nil {
			return errors.New("identity value is NULL")
		}
		value, err := parseIdentityColumn(p.Name, sv.identityType, src)
		if err != nil {
			return err
		}
		sv.record.ID = value
	case "timestamp":
		if src == nil {
			return errors.New("timestamp value is NULL")
		}
		value, err := normalizeDatabaseFileProperty(p.Name, types.DateTime(), src, false)
		if err != nil {
			return err
		}
		sv.record.Timestamp = value.(time.Time)
	}
	return nil
}
