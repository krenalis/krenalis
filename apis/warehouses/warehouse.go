//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"chichi/apis/types"
)

type Type int

const (
	BigQuery Type = iota + 1
	PostgreSQL
	Redshift
	Snowflake
)

// wrapError wraps err as an Error error.
// If err is nil, it returns a nil error.
func wrapError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{err}
}

// Error represents an error with a data warehouse. It could be for example an
// authentication error or a network error.
type Error struct {
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("cannot call the data warehouse: %s", e.Err)
}

// Warehouse is the interface implemented by data warehouses.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// CreateTables creates the data warehouse tables. schema is the schema of the
	// users table. If a table already exists it returns an Error error.
	CreateTables(ctx context.Context, schema types.Type) error

	// Exec executes a query without returning any rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Ping checks whether the connection to the data warehouse is active and, if
	// necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// Tables returns the tables of the data warehouse.
	// It returns only the tables 'users', 'groups', 'events', and the tables with
	// prefix 'users_', 'groups_' and 'events_'.
	Tables(ctx context.Context) ([]*Table, error)

	// Type returns the type of the warehouse.
	Type() Type

	// Query executes a query that returns rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Query(ctx context.Context, query string, args ...any) (*Rows, error)

	// QueryRow executes a query that should return at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// Users returns the users, with only the properties in schema, ordered by
	// order if order is not the zero Property, and in range [first,first+limit]
	// with first >= 0 and 0 < limit <= 1000.
	//
	// If a query to the warehouse fails, it returns an Error value.
	// If an argument is not valid, it panics.
	Users(ctx context.Context, schema types.Type, order types.Property, first, limit int) ([][]any, error)

	// Validate validates the settings and returns an error if they are not valid.
	Validate() error
}

// Table represents a table.
type Table struct {
	Name    string
	Columns []*Column
}

// Column represents a table column.
type Column struct {
	Name        string
	Description string
	Type        types.Type
	IsNullable  bool
	IsUpdatable bool
}

// IsValidType reports whether typ is a valid warehouse type.
func IsValidType(typ Type) bool {
	switch typ {
	case PostgreSQL:
		return true
	case BigQuery, Redshift, Snowflake:
		return false
	}
	return false
}

// Row returns a single row as a result of calling QueryRow.
type Row struct {
	row *sql.Row
	err error
}

func (row Row) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	err := row.row.Scan(dest...)
	if err == sql.ErrNoRows {
		return err
	}
	return wrapError(err)
}

func (row Row) Err() error {
	if row.err != nil {
		return row.err
	}
	err := row.row.Err()
	return wrapError(err)
}

// Rows represents the result of a query. Its methods, on error, return an
// Error value.
type Rows struct {
	rows *sql.Rows
}

func (rows Rows) Close() error {
	return wrapError(rows.rows.Close())
}

func (rows Rows) Err() error {
	return wrapError(rows.rows.Err())
}

func (rows Rows) Next() bool {
	return rows.rows.Next()
}

func (rows Rows) Scan(dest ...any) error {
	return wrapError(rows.rows.Scan(dest...))
}

// result implements the sql.Result interface but on error it returns an Error
// value.
type result struct {
	result sql.Result
}

func (r result) LastInsertId() (int64, error) {
	id, err := r.result.LastInsertId()
	if err != nil {
		return 0, wrapError(err)
	}
	return id, nil
}

func (r result) RowsAffected() (int64, error) {
	n, err := r.result.RowsAffected()
	if err != nil {
		return 0, wrapError(err)
	}
	return n, nil
}

// MarshalJSON implements the json.Marshaler interface.
// It panics if typ is not a valid Type value.
func (typ Type) MarshalJSON() ([]byte, error) {
	return []byte(`"` + typ.String() + `"`), nil
}

// Scan implements the sql.Scanner interface.
func (typ *Type) Scan(src any) error {
	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan a %T value into an warehouse.Type value", src)
	}
	var t Type
	switch s {
	case "BigQuery":
		t = BigQuery
	case "PostgreSQL":
		t = PostgreSQL
	case "Redshift":
		t = Redshift
	case "Snowflake":
		t = Snowflake
	default:
		return fmt.Errorf("invalid warehouse.Type: %s", s)
	}
	*typ = t
	return nil
}

// String returns the string representation of typ.
// It panics if typ is not a valid Type value.
func (typ Type) String() string {
	s, err := typ.Value()
	if err != nil {
		panic("invalid warehouse type")
	}
	return s.(string)
}

var null = []byte("null")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (typ *Type) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, null) {
		return nil
	}
	var s any
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("json: cannot unmarshal into a warehouse.Type value: %s", err)
	}
	return typ.Scan(s)
}

// Value implements driver.Valuer interface.
// It returns an error if typ is not a valid Type.
func (typ Type) Value() (driver.Value, error) {
	switch typ {
	case BigQuery:
		return "BigQuery", nil
	case PostgreSQL:
		return "PostgreSQL", nil
	case Redshift:
		return "Redshift", nil
	case Snowflake:
		return "Snowflake", nil
	}
	return nil, fmt.Errorf("not a valid Type: %d", typ)
}

// isValidTableName reports whether name is a valid table name.
// A valid table name must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func isValidTableName(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if !('a' <= c && c <= 'z' || c == '_' || 'A' <= c && c <= 'Z' || i > 0 && '0' <= c && c <= '9') {
			return false
		}
	}
	return true
}

// isValidSchemaName reports whether name is a valid schema name.
func isValidSchemaName(name string) bool {
	return isValidTableName(name)
}

// newError returns a new Error value with a fmt.Errorf(format, a...) error.
func newError(format string, a ...any) error {
	return &Error{Err: fmt.Errorf(format, a...)}
}
