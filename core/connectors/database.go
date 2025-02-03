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
	"iter"
	"slices"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/schemas"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

type databaseConnector interface {

	// Close closes the database. When Close is called, no other calls to
	// connector's methods are in progress and no more will be made.
	Close() error

	// Columns returns the columns of the given table.
	// If a column type is not supported, it returns a *UnsupportedColumnTypeError
	// error.
	Columns(ctx context.Context, table string) ([]meergo.Column, error)

	// Merge performs batch insert and update operations on the specified table,
	// basing on the table keys. rows contains the rows to be inserted or updated;
	// rows with matching table keys are updated, while new rows are inserted.
	//
	// table.Columns (and consequently each row in rows) contains at least one
	// additional column besides the table key values.
	//
	// Some connectors may check that the table keys actually match the primary keys
	// of the table, returning an error if they do not.
	Merge(ctx context.Context, table meergo.Table, rows [][]any) error

	// Query executes the given query and returns the resulting rows and columns.
	// If a column type is not supported, it returns a *UnsupportedColumnTypeError
	// error.
	Query(ctx context.Context, query string) (meergo.Rows, []meergo.Column, error)

	// QuoteTime returns a quoted time value for the specified type or "NULL" if the
	// value is nil.
	//
	// value is the time value to quote, and typ specifies its type, which can be
	// DateTime, Date, Text, or JSON:
	//
	//   - For DateTime and Date types, value is a time.Time.
	//   - For Text and JSON types, value is a string.
	QuoteTime(value any, typ types.Type) string
}

// Database represents the database of a database connection.
type Database struct {
	closed      bool
	connector   string
	timeLayouts *state.TimeLayouts
	inner       databaseConnector
	err         error
}

// Database returns a database for the provided connection. Errors are deferred
// until a database's method is called. It panics if connection is not a
// database connection.
//
// The caller must call the database's Close method when the database is no
// longer needed.
func (connectors *Connectors) Database(connection *state.Connection) *Database {
	connector := connection.Connector()
	database := &Database{
		connector:   connection.Connector().Name,
		timeLayouts: &connector.TimeLayouts,
	}
	inner, err := meergo.RegisteredDatabase(connector.Name).New(&meergo.DatabaseConfig{
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(connectors.state, connection),
	})
	database.inner = inner.(databaseConnector)
	database.err = err
	return database
}

// Close closes the database. When Close is called, no other calls to the
// database's methods must be in progress, and no more calls must be made.
// Close is idempotent.
// It returns an *UnavailableError error if the connector returns an error.
func (database *Database) Close() error {
	if database.err != nil {
		return database.err
	}
	if database.closed {
		return nil
	}
	database.closed = true
	err := database.inner.Close()
	return connectorError(err)
}

// LastChangeTimePlaceholder returns the value used for the last_change_time
// placeholder for the provided action.
func (database *Database) LastChangeTimePlaceholder(action *state.Action) (string, error) {
	if database.err != nil {
		return "", database.err
	}
	if action == nil {
		return database.inner.QuoteTime(nil, types.Type{}), nil
	}
	execution, ok := action.Execution()
	if !ok {
		return "", errors.New("action is not currently executing")
	}
	cursor := execution.Cursor
	property := action.LastChangeTimeColumn
	if property == "" || cursor.IsZero() {
		return database.inner.QuoteTime(nil, types.Type{}), nil
	}
	p, _ := action.InSchema.Property(property)
	var value any
	switch p.Type.Kind() {
	case types.DateTimeKind:
		value = execution.Cursor
	case types.DateKind:
		value = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, time.UTC)
	case types.TextKind, types.JSONKind:
		value = formatLastChangeTimeColumn(action.LastChangeTimeFormat, cursor)
	}
	return database.inner.QuoteTime(value, p.Type), nil
}

// Schema returns the schema of the provided table and role. If a column type is
// not supported, it returns a *meergo.UnsupportedColumnTypeError error. If the
// connector returns an error, it returns an *UnavailableError error.
// It panics if role is not Source or Destination.
func (database *Database) Schema(ctx context.Context, table string, role state.Role) (types.Type, error) {
	if database.err != nil {
		return types.Type{}, database.err
	}
	if role != state.Source && role != state.Destination {
		panic("invalid role")
	}
	columns, err := database.inner.Columns(ctx, table)
	if err != nil {
		return types.Type{}, connectorError(err)
	}
	if len(columns) == 0 {
		return types.Type{}, errors.New("no columns defined for table")
	}
	schema, err := types.ObjectOf(columnsToProperties(columns, role))
	if err != nil {
		return types.Type{}, rewriteColumnErrors(err)
	}
	return schema, nil
}

// Query executes a query and returns the resulting rows.
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a *PlaceholderError error may be returned in case of
// an error with placeholders. If a column type is not supported, it returns a
// *meergo.UnsupportedColumnTypeError error. If the connector returns an error,
// it returns an *UnavailableError error.
func (database *Database) Query(ctx context.Context, query string, queryReplacer PlaceholderReplacer) (*Rows, error) {
	if database.err != nil {
		return nil, database.err
	}
	if queryReplacer != nil {
		var err error
		query, err = ReplacePlaceholders(query, queryReplacer)
		if err != nil {
			return nil, err
		}
	}
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, connectorError(err)
	}
	if _, err = types.ObjectOf(columnsToProperties(columns, state.Source)); err != nil {
		_ = rows.Close()
		return nil, connectorError(rewriteColumnErrors(err))
	}
	return newRows(rows, columns, database.timeLayouts), nil
}

// Records executes the action's query and returns an iterator to iterate over
// the database's records. Each returned record will contain, in the Properties
// field, the properties of the action's input schema, with the same types.
//
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a *PlaceholderError error may be returned in case of
// an error with placeholders.
//
// If the action's input schema does not align with the query's results schema,
// it returns a *schemas.Error error. If a column type is not supported, it
// returns a *meergo.UnsupportedColumnTypeError error. If the connector returns
// an error, it returns an *UnavailableError error.
func (database *Database) Records(ctx context.Context, action *state.Action, queryReplacer PlaceholderReplacer) (Records, error) {
	if database.err != nil {
		return nil, database.err
	}
	if !action.InSchema.Valid() {
		return nil, fmt.Errorf("action's input schema is not valid")
	}
	query := action.Query
	// Replace the placeholders in query, if necessary.
	if queryReplacer != nil {
		var err error
		query, err = ReplacePlaceholders(query, queryReplacer)
		if err != nil {
			return nil, err
		}
	}
	// Execute the query.
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, connectorError(err)
	}
	var records Records
	defer func() {
		if records == nil {
			_ = rows.Close()
		}
	}()
	var identityColumn, lastChangeTimeColumn types.Property
	for _, c := range columns {
		if c.Name == action.IdentityColumn {
			identityColumn, _ = action.InSchema.Property(c.Name)
		}
		if c.Name == action.LastChangeTimeColumn {
			lastChangeTimeColumn, _ = action.InSchema.Property(c.Name)
		}
	}
	if identityColumn.Name == "" {
		return nil, &SchemaError{fmt.Sprintf("there is no identity column %q", action.IdentityColumn)}
	}
	if action.LastChangeTimeColumn != "" && lastChangeTimeColumn.Name == "" {
		return nil, &SchemaError{fmt.Sprintf("there is no last change time column %q", action.LastChangeTimeColumn)}
	}

	// Check that schema is aligned with the query's schema.
	querySchema, err := types.ObjectOf(columnsToProperties(columns, state.Source))
	if err != nil {
		return nil, rewriteColumnErrors(err)
	}
	err = schemas.CheckAlignment(action.InSchema, querySchema, nil)
	if err != nil {
		return nil, err
	}
	// Return the records.
	records = newDatabaseRecords(rows, columns, action, database.timeLayouts)
	return records, nil
}

// Writer returns a Writer to create and update users.
//
// If the action's output schema does not align with the table's schema, it
// returns a *schemas.Error error.
func (database *Database) Writer(ctx context.Context, action *state.Action, ack AckFunc) (Writer, error) {
	if database.err != nil {
		return nil, database.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	columns, err := database.inner.Columns(ctx, action.TableName)
	if err != nil {
		return nil, connectorError(err)
	}
	properties := columnsToProperties(columns, state.Destination)
	for i, p := range properties {
		// The table key cannot be nullable. This sets it as not nullable as a
		// temporary workaround, until we can ensure that all connector
		// implementations correctly handle the Nullable attribute for each property
		// (see issue #374).
		if p.Name == action.TableKey {
			properties[i].Nullable = false
		}
	}
	tableSchema, err := types.ObjectOf(properties)
	if err != nil {
		return nil, err
	}
	err = schemas.CheckAlignment(action.OutSchema, tableSchema, nil)
	if err != nil {
		return nil, err
	}
	w := databaseWriter{
		ack: ack,
		table: meergo.Table{
			Name:    action.TableName,
			Columns: columnsOfType(action.OutSchema),
			Keys:    []string{action.TableKey},
		},
		schema: action.OutSchema,
		inner:  database.inner,
	}
	return &w, nil
}

// columnsOfType returns the properties of a type as meergo.Column values.
func columnsOfType(t types.Type) []meergo.Column {
	columns := make([]meergo.Column, types.NumProperties(t))
	for i, p := range t.Properties() {
		columns[i].Name = p.Name
		columns[i].Type = p.Type
		columns[i].Nullable = p.Nullable
	}
	return columns
}

// columnsToProperties returns the provided columns as types.Property values.
// If role is Destination, it excludes non-writable columns.
func columnsToProperties(columns []meergo.Column, role state.Role) []types.Property {
	properties := make([]types.Property, len(columns))
	for i, c := range columns {
		if role == state.Destination && !c.Writable {
			continue
		}
		properties[i].Name = c.Name
		properties[i].Type = c.Type
		properties[i].Nullable = c.Nullable
	}
	return properties
}

// databaseWriter implements the Writer interface for databases.
type databaseWriter struct {
	ack    AckFunc
	table  meergo.Table
	schema types.Type
	rows   [][]any
	ids    []string
	inner  databaseConnector
	closed bool
}

func (w *databaseWriter) Close(ctx context.Context) error {
	if w.closed {
		return nil
	}
	if len(w.rows) > 0 {
		w.merge(ctx)
	}
	w.closed = true
	return nil
}

func (w *databaseWriter) Write(ctx context.Context, id string, properties map[string]any) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	// Append the row and the ack ids.
	row := make([]any, len(w.table.Columns))
	for i, c := range w.table.Columns {
		row[i] = properties[c.Name]
	}
	w.rows = append(w.rows, row)
	w.ids = append(w.ids, id)
	// Upsert the rows.
	if len(w.rows) == 100 {
		w.merge(ctx)
	}
	return true
}

// merge calls the Merge method of the database connector with the collected
// records.
func (w *databaseWriter) merge(ctx context.Context) {
	err := w.inner.Merge(ctx, w.table, w.rows)
	w.ack(w.ids, connectorError(err))
	w.rows = slices.Delete(w.rows, 0, len(w.rows))
	w.ids = slices.Delete(w.ids, 0, len(w.ids))
}

// Rows is the result of a query.
type Rows struct {
	rows        meergo.Rows
	columns     []meergo.Column
	timeLayouts *state.TimeLayouts
	dst         []any
	closed      bool
}

func newRows(rows meergo.Rows, columns []meergo.Column, layouts *state.TimeLayouts) *Rows {
	rs := &Rows{
		rows:        rows,
		columns:     columns,
		timeLayouts: layouts,
		dst:         make([]any, len(columns)),
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

// Columns returns the columns as properties.
func (rs *Rows) Columns() []types.Property {
	return columnsToProperties(rs.columns, state.Source)
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
		rs.dst[i] = queryScanValue{column: c, row: row, timeLayouts: rs.timeLayouts}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, &UnavailableError{Err: err}
	}
	return row, nil
}

// databaseRecords implements the Records interface for databases.
type databaseRecords struct {
	rows        meergo.Rows
	columns     []meergo.Column
	action      *state.Action
	timeLayouts *state.TimeLayouts
	last        bool
	err         error
	closed      bool
}

func newDatabaseRecords(rows meergo.Rows, columns []meergo.Column, action *state.Action, layouts *state.TimeLayouts) *databaseRecords {
	records := databaseRecords{
		rows:        rows,
		columns:     columns,
		action:      action,
		timeLayouts: layouts,
	}
	return &records
}

func (r *databaseRecords) All(ctx context.Context) iter.Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			return
		}
		defer r.Close()
		execution, _ := r.action.Execution()
		n := 0 // number of properties per record
		var identityIndex = -1
		var lastChangeTimeIndex = -1
		scanner := scanner{
			values: make([]any, len(r.columns)),
		}
		dest := make([]any, len(r.columns))
		properties := make([]types.Property, len(r.columns))
		for i, c := range r.columns {
			dest[i] = &scanner
			p, ok := r.action.InSchema.Property(c.Name)
			if !ok {
				continue
			}
			properties[i] = p
			if p.Name == r.action.IdentityColumn {
				identityIndex = i
			}
			if p.Name == r.action.LastChangeTimeColumn {
				lastChangeTimeIndex = i
			}
			n++
		}
		// Read the rows.
		var record Record
	Rows:
		for r.rows.Next() {
			if record.Properties != nil || record.Err != nil {
				if !yield(record) {
					return
				}
				record.Properties = nil
				record.Err = nil
			}
			select {
			case <-ctx.Done():
				r.err = ctx.Err()
				return
			default:
			}
			if err := r.rows.Scan(dest...); err != nil {
				record.Err = err
				scanner.reset()
				continue Rows
			}
			// Get the last change time.
			if lastChangeTimeIndex >= 0 {
				v := scanner.values[lastChangeTimeIndex]
				if v == nil {
					record.Err = errors.New("last change time value is NULL")
					continue Rows
				}
				p := properties[lastChangeTimeIndex]
				var err error
				record.LastChangeTime, err = parseLastChangeTimeColumn(p.Name, p.Type, r.action.LastChangeTimeFormat, v, p.Nullable, r.timeLayouts)
				if err != nil {
					record.Err = err
					continue Rows
				}
				if !record.LastChangeTime.IsZero() && record.LastChangeTime.Before(execution.Cursor) {
					continue Rows
				}
			}
			if record.LastChangeTime.IsZero() {
				record.LastChangeTime = time.Now().UTC()
			}
			// Get the identity.
			if identityIndex >= 0 {
				v := scanner.values[identityIndex]
				if v == nil {
					record.Err = errors.New("identity value is NULL")
					continue Rows
				}
				p := properties[identityIndex]
				id, err := parseIdentityColumn(p.Name, p.Type, v, r.timeLayouts)
				if err != nil {
					record.Err = err
					continue Rows
				}
				record.ID = id
			}
			// Get the properties.
			record.Properties = make(map[string]any, n)
			for i, v := range scanner.values {
				p := properties[i]
				if p.Name == "" {
					continue
				}
				value, err := normalize(p.Name, p.Type, v, p.Nullable, r.timeLayouts)
				if err != nil {
					record.Err = err
					continue Rows
				}
				record.Properties[p.Name] = value
			}
		}
		if record.Properties != nil || record.Err != nil {
			r.last = true
			if !yield(record) {
				return
			}
		}
		if err := r.rows.Err(); err != nil {
			r.err = err
		}
	}
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

func (r *databaseRecords) Last() bool {
	return r.last
}

// queryScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type queryScanValue struct {
	column      meergo.Column
	row         map[string]any
	timeLayouts *state.TimeLayouts
}

func (sv queryScanValue) Scan(src any) error {
	c := sv.column
	value, err := normalize(c.Name, c.Type, src, c.Nullable, sv.timeLayouts)
	if err != nil {
		return err
	}
	sv.row[c.Name] = value
	return nil
}

// scanner implements the sql.Scanner interface to read the database values from
// a database connector.
type scanner struct {
	index  int
	values []any
}

func (sv *scanner) Scan(src any) error {
	sv.values[sv.index] = src
	sv.index++
	sv.index %= len(sv.values)
	return nil
}

func (sv *scanner) reset() {
	sv.index = 0
}
