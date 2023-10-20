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

	"chichi/apis/datastore/expr"
	"chichi/apis/postgres"
	"chichi/connector/types"
)

// MergeTable represents a table in which rows will be merged.
type MergeTable struct {
	Name        string           // Name of the table
	Columns     []types.Property // Columns to merge
	PrimaryKeys []types.Property // Primary keys
}

// Warehouse is the interface implemented by data warehouses.
//
// Methods return a DataWarehouseError error if an error occurs with the data
// warehouse.
type Warehouse interface {

	// Close closes the warehouse. It will not allow any new queries to run, and it
	// waits for the current ones to finish.
	Close() error

	// DestinationUser returns the external ID of the destination user of the
	// action that matches with the corresponding property. If it cannot be
	// found, then the empty string and false are returned.
	DestinationUser(ctx context.Context, action int, property string) (string, bool, error)

	// Exec executes a query without returning any rows. args are the placeholders.
	Exec(ctx context.Context, query string, args ...any) (Result, error)

	// Init initializes the data warehouse by creating the supporting tables.
	Init(ctx context.Context) error

	// Merge performs a table merge operation, handling row updates, inserts, and
	// deletions. table specifies the target table for the merge operation, rows
	// contains the rows to insert or update in the table, and deleted contains the
	// key values of rows to delete, if they exist.
	// rows or deleted can be empty but not both.
	Merge(ctx context.Context, table MergeTable, rows [][]any, deleted []any) error

	// Ping checks whether the connection to the data warehouse is active and, if
	// necessary, establishes a new connection.
	Ping(ctx context.Context) error

	// SetDestinationUser sets the destination user relative to the action, with
	// the given external user ID and external property.
	SetDestinationUser(ctx context.Context, connection int, externalUserID, externalProperty string) error

	// SetIdentity sets the identity id (which may have an anonymous ID) imported
	// from the action. fromEvents indicates if the identity has been imported from
	// an event or not.
	SetIdentity(ctx context.Context, identity map[string]any, id string, anonID string, action int, fromEvent bool) error

	// Settings returns the data warehouse settings.
	Settings() []byte

	// Tables returns the tables of the data warehouse.
	// It returns only the tables 'users', 'groups', 'events', and the tables with
	// prefix 'users_', 'groups_' and 'events_'.
	Tables(ctx context.Context) ([]*Table, error)

	// QueryRow executes a query that should return at most one row.
	QueryRow(ctx context.Context, query string, args ...any) Row

	// ResolveSyncUsers resolves and sync the users.
	// actions holds the identifiers of the actions of the workspace and must always
	// contain at least one action; identifiers are the columns of the
	// 'users_identities' table which are identifiers, ordered by priority;
	// usersColumns are the columns of the 'users' table which will be populated
	// during the users synchronization.
	ResolveSyncUsers(ctx context.Context, actions []int, identifiersColumns, usersColumns []types.Property) error

	// Select returns the rows from the given table that satisfies the where
	// condition with only the given columns, ordered by order if order is not the
	// zero Property, and in range [first,first+limit] with first >= 0 and
	// 0 < limit <= 1000.
	Select(ctx context.Context, table string, columns []types.Property, where expr.Expr, order types.Property, first, limit int) ([][]any, error)
}

// Table represents a table.
type Table struct {
	Name    string
	Columns []types.Property
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
	if err != nil {
		if err == sql.ErrNoRows {
			return err
		}
		return Error(err)
	}
	return nil
}

// Rows represents the result of a query. Its methods, on error, return a
// DataWarehouseError error.
type Rows struct {
	Rows *postgres.Rows
}

func (rows Rows) Close() {
	rows.Rows.Close()
}

func (rows Rows) Err() error {
	err := rows.Rows.Err()
	if err != nil {
		return Error(err)
	}
	return nil
}

func (rows Rows) Next() bool {
	return rows.Rows.Next()
}

func (rows Rows) Scan(dest ...any) error {
	err := rows.Rows.Scan(dest...)
	if err != nil {
		return Error(err)
	}
	return nil
}

// Result implements the sql.Result interface but on error it returns a
// DataWarehouseError error.
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
