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
	"chichi/connector/ui"
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

	// Exec executes a query without returning any rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Exec(ctx context.Context, query string, args ...any) (sql.Result, error)

	// Type returns the type of the warehouse.
	Type() Type

	// ServeUI serves the data warehouse's user interface.
	ServeUI(ctx context.Context, event string, values []byte) (*ui.Form, *ui.Alert, []byte, error)

	// Query executes a query that returns rows. args are the placeholders.
	// If the query fails, it returns an Error value.
	Query(ctx context.Context, query string, args ...any) (*sql.Rows, error)

	// QueryRow executes a query that should return at most one row.
	// If the query fails, it returns an Error value.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// Users returns the users, with only the properties in schema, ordered by
	// order if order is not the zero Property, and in range [first,first+limit]
	// with first >= 0 and 0 < limit <= 1000.
	//
	// If a query to the warehouse fails, it returns an Error value.
	// If an argument is not valid, it panics.
	Users(ctx context.Context, schema types.Schema, order types.Property, first, limit int) ([][]any, error)
}

type Row interface {
	Scan(dest ...any) error
	Err() error
}

// Open opens a data warehouse specified by its type and settings.
// Open does not open a connection to the database.
func Open(typ Type, settings []byte) (Warehouse, error) {
	switch typ {
	case PostgreSQL:
		return openPostgres(settings)
	case BigQuery, Redshift, Snowflake:
		return nil, fmt.Errorf("warehouse type %s is not supported", typ)
	}
	return nil, fmt.Errorf("warehouse type %d does not exist", typ)
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
