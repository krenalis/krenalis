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

	"chichi/apis/normalization"
	"chichi/apis/postgres"
	"chichi/connector/types"
)

// Warehouse is the interface implemented by data warehouses.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// DestinationUser returns the external ID of the destination user of the
	// action that matches with the corresponding property. If it cannot be
	// found, then the empty string and false are returned.
	DestinationUser(ctx context.Context, action int, property string) (string, bool, error)

	// Exec executes a query without returning any rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Exec(ctx context.Context, query string, args ...any) (Result, error)

	// Init initializes the data warehouse by creating the supporting tables.
	Init(ctx context.Context) error

	// Ping checks whether the connection to the data warehouse is active and, if
	// necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// PrepareBatch creates a prepared batch statement for inserting rows in
	// batch and returns it. table specifies the table in which the rows will be
	// inserted, and columns specifies the columns.
	PrepareBatch(ctx context.Context, table string, columns []string) (Batch, error)

	// SetDestinationUser sets the destination user relative to the action, with
	// the given external user ID and external property.
	SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Tables returns the tables of the data warehouse.
	// It returns only the tables 'users', 'groups', 'events', and the tables with
	// prefix 'users_', 'groups_' and 'events_'.
	Tables(ctx context.Context) ([]*Table, error)

	// Query executes a query that returns rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Query(ctx context.Context, query string, args ...any) (*Rows, error)

	// QueryRow executes a query that should return at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// Select returns the rows from the given table that satisfies the where
	// condition with only the given columns, ordered by order if order is not the
	// zero Property, and in range [first,first+limit] with first >= 0 and
	// 0 < limit <= 1000.
	//
	// If a query to the warehouse fails, it returns an Error value.
	// If an argument is not valid, it panics.
	Select(ctx context.Context, table string, columns []types.Property, where Where, order types.Property, first, limit int) ([][]any, error)
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
	Nullable    bool
	IsUpdatable bool
}

// Row returns a single row as a result of calling QueryRow.
type Row struct {
	Row   *postgres.Row
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

// Rows represents the result of a query. Its methods, on error, return an
// Error value.
type Rows struct {
	Rows *postgres.Rows
}

func (rows Rows) Close() {
	rows.Rows.Close()
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
	Result *postgres.Result
}

func (r Result) RowsAffected() (int64, error) {
	n := r.Result.RowsAffected()
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

// ScanValue implements the sql.Scanner interface to read the database values.
type ScanValue struct {
	property    types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// NewScanValues returns a slice containing scan values to be used to scan rows.
func NewScanValues(properties []types.Property, rows *[][]any) []any {
	values := make([]any, len(properties))
	for i, p := range properties {
		values[i] = ScanValue{
			property:    p,
			rows:        rows,
			columnIndex: i,
			columnCount: len(properties),
		}
	}
	return values
}

func (sv ScanValue) Scan(src any) error {
	p := sv.property
	value, err := normalization.NormalizeDatabaseFileProperty(p.Name, p.Nullable, p.Type, src)
	if err != nil {
		return err
	}
	var row []any
	if sv.columnIndex == 0 {
		row = make([]any, sv.columnCount)
		*sv.rows = append(*sv.rows, row)
	} else {
		row = (*sv.rows)[len(*sv.rows)-1]
	}
	row[sv.columnIndex] = value
	return nil
}
