//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2022 Open2b
//

package warehouses

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"sync"

	"chichi/apis/types"
)

// Warehouse is the interface implemented by data warehouses.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// Exec executes a query without returning any rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Ping checks whether the connection to the data warehouse is active and, if
	// necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// PrepareBatch creates a prepared batch statement for inserting rows in
	// batch and returns it. table specifies the table in which the rows will be
	// inserted, and columns specifies the columns.
	PrepareBatch(ctx context.Context, table string, columns []string) (Batch, error)

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Tables returns the tables of the data warehouse.
	// It returns only the tables 'users', 'groups', 'events', and the tables with
	// prefix 'users_', 'groups_' and 'events_'. Also, it does not return columns
	// starting with an underscore.
	Tables(ctx context.Context) ([]*Table, error)

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
}

// Batch is implemented by values returned by PrepareBatch.
type Batch interface {

	// Abort aborts the batch.
	Abort() error

	// Append appends the values of a row to batch.
	Append(v ...interface{}) error

	// AppendStruct appends the values of a row, read from the fields of the struct
	// v, to batch. It returns an error if v is not a struct.
	AppendStruct(v interface{}) error

	// Send sends the batch to the data warehouse.
	Send() error
}

// Error represents an error with a data warehouse. It could be for example an
// authentication error or a network error.
type Error struct {
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("cannot call the data warehouse: %s", e.Err)
}

// NewError returns a new Error value with a fmt.Errorf(format, a...) error.
func NewError(format string, a ...any) error {
	return &Error{Err: fmt.Errorf(format, a...)}
}

// WrapError wraps err as an Error error.
// If err is nil, it returns a nil error.
func WrapError(err error) error {
	if err == nil {
		return nil
	}
	return &Error{err}
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

// Row returns a single row as a result of calling QueryRow.
type Row struct {
	Row   *sql.Row
	Error error
}

func (row Row) Scan(dest ...any) error {
	if row.Error != nil {
		return row.Error
	}
	err := row.Row.Scan(dest...)
	if err == sql.ErrNoRows {
		return err
	}
	return WrapError(err)
}

func (row Row) Err() error {
	if row.Error != nil {
		return row.Error
	}
	err := row.Row.Err()
	return WrapError(err)
}

// Rows represents the result of a query. Its methods, on error, return an
// Error value.
type Rows struct {
	Rows *sql.Rows
}

func (rows Rows) Close() error {
	return WrapError(rows.Rows.Close())
}

func (rows Rows) Err() error {
	return WrapError(rows.Rows.Err())
}

func (rows Rows) Next() bool {
	return rows.Rows.Next()
}

func (rows Rows) Scan(dest ...any) error {
	return WrapError(rows.Rows.Scan(dest...))
}

// Result implements the sql.Result interface but on error it returns an Error
// value.
type Result struct {
	Result sql.Result
}

func (r Result) LastInsertId() (int64, error) {
	id, err := r.Result.LastInsertId()
	if err != nil {
		return 0, WrapError(err)
	}
	return id, nil
}

func (r Result) RowsAffected() (int64, error) {
	n, err := r.Result.RowsAffected()
	if err != nil {
		return 0, WrapError(err)
	}
	return n, nil
}

// IsValidIdentifier reports whether name is a valid identifier.
// A valid identifier must:
//   - start with [A-Za-z_]
//   - subsequently contain only [A-Za-z0-9_]
func IsValidIdentifier(name string) bool {
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

// IsValidSchemaName reports whether name is a valid schema name.
func IsValidSchemaName(name string) bool {
	return IsValidIdentifier(name)
}

// columnsIndexes contains the column indexes of the struct passed as argument
// to AppendStruct of Batch.
var columnsIndexes = sync.Map{}

// ColumnsIndex returns a map from a column name to its index in the struct t.
func ColumnsIndex(t reflect.Type) (map[string][]int, error) {
	idx, ok := columnsIndexes.Load(t)
	if ok {
		return idx.(map[string][]int), nil
	}
	index := map[string][]int{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		column := field.Tag.Get("column")
		if column == "" {
			column = field.Name
		}
		if !IsValidIdentifier(column) {
			return nil, fmt.Errorf("column name %q is not a valid identifier", column)
		}
		index[column] = field.Index
	}
	columnsIndexes.Store(t, index)
	return index, nil
}
