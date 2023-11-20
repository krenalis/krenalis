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

// Records executes a query and returns the resulting records. The returned
// records conform to the provided schema, which must be valid and compatible
// with the query's schema.
//
// The query execution must return a column named "id", the identity column,
// with type Int, Uint, UUID, or Text. If the query execution returns a column
// named "timestamp," that column is considered the timestamp column and must
// have the type DateTime.
func (database *Database) Records(ctx context.Context, query string, schema types.Type) (*Records, error) {
	if database.err != nil {
		return nil, database.err
	}
	// Execute the query.
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	// Validate the identity and timestamp columns.
	var hasIdentityColumn bool
	for _, c := range columns {
		switch c.Name {
		case "id":
			switch k := c.Type.Kind(); k {
			case types.IntKind, types.UintKind, types.UUIDKind, types.TextKind:
			default:
				return nil, fmt.Errorf(`identity column "id" has type %s instead of Int, Uint, UUID, or Text`, c.Type)
			}
			hasIdentityColumn = true
		case "timestamp":
			if c.Type.Kind() != types.DateTimeKind {
				return nil, fmt.Errorf(`timestamp column "timestamp" has type %s instead of DateTime`, c.Type)
			}
		}
	}
	if !hasIdentityColumn {
		return nil, fmt.Errorf(`there is no identity column "id"`)
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
	return newRecords(rows, columns, schema.Properties()), nil
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
	for i, c := range rs.columns {
		rs.dst[i] = databaseScanValue{property: c, record: &Record{Properties: row}}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, err
	}
	return row, nil
}

// Records is the result of a query.
type Records struct {
	columns    []types.Property
	rows       _connector.Rows
	propertyOf map[string]types.Property
	dst        []any
	closed     bool
}

func newRecords(rows _connector.Rows, columns, properties []types.Property) *Records {
	records := Records{
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

// Close closes the records. Close is idempotent.
func (records *Records) Close() error {
	if records.closed {
		return nil
	}
	records.closed = true
	return records.rows.Close()
}

// Err returns the error encountered during iteration, if any. It can be called
// after an explicit or implicit Close
func (records *Records) Err() error {
	return records.rows.Err()
}

// Next prepares the next record for reading with the Scan method.
// It returns true on success, signaling the availability of a record, or
// false in cases where there is no next record or an error occurred during
// preparation.
//
// Every call to Scan, even the first one, must be preceded by a call to Next.
func (records *Records) Next() bool {
	return records.rows.Next()
}

// Scan returns the current record.
func (records *Records) Scan() (Record, error) {
	record := Record{
		Properties: make(map[string]any, len(records.propertyOf)),
	}
	for i, c := range records.columns {
		sv := databaseScanValue{
			property: records.propertyOf[c.Name],
			record:   &record,
		}
		sv.property.Name = c.Name
		if c.Name == "id" {
			sv.identityType = c.Type
		}
		records.dst[i] = sv
	}
	if err := records.rows.Scan(records.dst...); err != nil {
		return Record{}, err
	}
	return record, nil
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
