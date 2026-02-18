// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
	"time"

	"github.com/meergo/meergo/connectors"
	"github.com/meergo/meergo/core/internal/schemas"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/types"
)

type databaseConnection interface {

	// Close closes the database. When Close is called, no other calls to
	// connector's methods are in progress and no more will be made.
	Close() error

	// Columns returns the columns of the given table.
	Columns(ctx context.Context, table string) ([]connectors.Column, error)

	// Merge performs batch insert and update operations on the specified table,
	// basing on the table keys. rows contains the rows to be inserted or updated;
	// rows with matching table keys are updated, while new rows are inserted.
	//
	// table.Columns (and consequently each row in rows) contains at least one
	// additional column besides the table key values.
	//
	// Some connectors may check that the table keys actually match the primary keys
	// of the table, returning an error if they do not.
	Merge(ctx context.Context, table connectors.Table, rows [][]any) error

	// Query executes the given query and returns the resulting rows and columns.
	// If a column is unsupported, its type will be the invalid type, and the Issue
	// property will describe the problem.
	Query(ctx context.Context, query string) (connectors.Rows, []connectors.Column, error)

	// SQLLiteral returns the SQL literal representation of v according to the
	// provided Meergo type t. It supports nil values and the following Meergo
	// types: string, datetime, date, and json.
	//
	// Examples:
	//   (nil, types.Type{})
	//   ("foo", types.String())
	//   ("{\"boo\":5}", types.JSON())
	//   (time.Date(2025, 12, 9, 12, 53, 45, 730139838, time.UTC), types.DateTime())
	//   (time.Date(2025, 12, 9, 0, 0, 0, 0, time.UTC), types.Date())
	//
	// For the inputs above, the PostgreSQL connector returns:
	//   NULL
	//   'foo'
	//   '{"boo": 5}'
	//   '2025-12-09 12:53:45.730139'
	//   '2025-12-09'
	SQLLiteral(v any, t types.Type) string
}

// Database represents the database of a database connection.
type Database struct {
	connector   string
	closed      bool
	timeLayouts *state.TimeLayouts
	inner       databaseConnection
	err         error
}

// Database returns a database for the provided connection. Errors are deferred
// until a database's method is called. It panics if connection is not a
// database connection.
//
// The caller must call the database's Close method when the database is no
// longer needed.
func (c *Connections) Database(connection *state.Connection) *Database {
	connector := connection.Connector()
	database := &Database{
		connector:   connector.Code,
		timeLayouts: &connector.TimeLayouts,
	}
	inner, err := connectors.RegisteredDatabase(connector.Code).New(&connectors.DatabaseEnv{
		Settings:    connection.Settings,
		SetSettings: setConnectionSettingsFunc(c.state, connection),
	})
	database.inner = inner.(databaseConnection)
	database.err = connectorError(err)
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

// Connector returns the name of the database connector.
func (database *Database) Connector() string {
	return database.connector
}

// Schema returns the schema for the given table and role, along with any issues
// encountered while reading the schema, such as unsupported column types.
// The returned schema may be invalid if there are no supported columns.
//
// If the connector returns an error, it will be an *UnavailableError error.
// It panics if role is not Source or Destination.
func (database *Database) Schema(ctx context.Context, table string, role state.Role) (types.Type, []string, error) {
	if database.err != nil {
		return types.Type{}, nil, database.err
	}
	if role != state.Source && role != state.Destination {
		panic("invalid role")
	}
	columns, err := database.inner.Columns(ctx, table)
	if err != nil {
		return types.Type{}, nil, connectorError(err)
	}
	if len(columns) == 0 {
		return types.Type{}, nil, errors.New("no columns defined for table")
	}
	properties := columnsProperties(columns, role)
	var schema types.Type
	if properties != nil {
		schema, err = types.ObjectOf(properties)
		if err != nil {
			return types.Type{}, nil, rewriteColumnErrors(err)
		}
	}
	issues, err := columnsIssues(columns)
	if err != nil {
		return types.Type{}, nil, connectorError(fmt.Errorf("connector %s %s", database.connector, err))
	}
	return schema, issues, nil
}

// Query executes a query and returns the resulting rows and schema, along with
// any issues encountered while reading the schema, such as unsupported column
// types. The returned schema can be the invalid schema.
//
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a *PlaceholderError error may be returned in case of
// an error with placeholders. If the connector returns an error, it returns an
// *UnavailableError error.
func (database *Database) Query(ctx context.Context, query string, queryReplacer PlaceholderReplacer) (*Rows, types.Type, []string, error) {
	if database.err != nil {
		return nil, types.Type{}, nil, database.err
	}
	if queryReplacer != nil {
		var err error
		query, err = ReplacePlaceholders(query, queryReplacer)
		if err != nil {
			return nil, types.Type{}, nil, err
		}
	}
	rows, columns, err := database.inner.Query(ctx, query)
	if err != nil {
		return nil, types.Type{}, nil, connectorError(err)
	}
	properties := columnsProperties(columns, state.Source)
	var schema types.Type
	if properties != nil {
		schema, err = types.ObjectOf(properties)
		if err != nil {
			_ = rows.Close()
			return nil, types.Type{}, nil, connectorError(rewriteColumnErrors(err))
		}
	}
	issues, err := columnsIssues(columns)
	if err != nil {
		_ = rows.Close()
		return nil, types.Type{}, nil, connectorError(fmt.Errorf("connector %s %s", database.connector, err))
	}
	for i, c := range columns {
		if !c.Type.Valid() {
			columns[i] = connectors.Column{}
		}
	}
	return newRows(rows, columns, database.timeLayouts), schema, issues, nil
}

// Records executes the pipeline's query and returns an iterator to iterate over
// the database's records. Each returned record will contain, in the Attributes
// field, the properties of the pipeline's input schema, with the same types.
//
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a *PlaceholderError error may be returned in case of
// an error with placeholders.
//
// If the pipeline's input schema does not align with the query's results schema,
// or if the identity or timestamp columns defined in the pipeline are not
// returned by the query, it returns a *schemas.Error error. If the connector
// returns an error, it returns an *UnavailableError error.
func (database *Database) Records(ctx context.Context, pipeline *state.Pipeline, queryReplacer PlaceholderReplacer) (Records, error) {
	if database.err != nil {
		return nil, database.err
	}
	if !pipeline.InSchema.Valid() {
		return nil, fmt.Errorf("pipeline's input schema is not valid")
	}
	query := pipeline.Query
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
	properties := pipeline.InSchema.Properties()
	var identityColumn, updatedAtColumn types.Property
	for i, c := range columns {
		if !c.Type.Valid() {
			columns[i] = connectors.Column{}
			continue
		}
		p, ok := properties.ByName(c.Name)
		if !ok {
			if !types.IsValidPropertyName(c.Name) {
				return nil, connectorError(fmt.Errorf("connector %s has returned an invalid column name %q", database.connector, c.Name))
			}
			columns[i] = connectors.Column{}
			continue
		}
		if c.Name == pipeline.IdentityColumn {
			identityColumn = p
		}
		if c.Name == pipeline.UpdatedAtColumn {
			updatedAtColumn = p
		}
	}
	if identityColumn.Name == "" {
		return nil, &schemas.Error{Msg: fmt.Sprintf("there is no identity column %q", pipeline.IdentityColumn)}
	}
	if pipeline.UpdatedAtColumn != "" && updatedAtColumn.Name == "" {
		return nil, &schemas.Error{Msg: fmt.Sprintf("there is no update time column %q", pipeline.UpdatedAtColumn)}
	}
	// Check that schema is aligned with the query's schema.
	schema, err := types.ObjectOf(columnsProperties(columns, state.Source))
	if err != nil {
		return nil, connectorError(rewriteColumnErrors(err))
	}
	err = schemas.CheckAlignment(pipeline.InSchema, schema, nil)
	if err != nil {
		return nil, err
	}
	// Return the records.
	records = newDatabaseRecords(rows, columns, pipeline, database.timeLayouts)
	return records, nil
}

// UpdatedAtPlaceholder returns the value used for the updated_at placeholder
// for the provided pipeline.
func (database *Database) UpdatedAtPlaceholder(pipeline *state.Pipeline) (string, error) {
	if database.err != nil {
		return "", database.err
	}
	if pipeline == nil {
		return database.inner.SQLLiteral(nil, types.Type{}), nil
	}
	run, ok := pipeline.Run()
	if !ok {
		return "", errors.New("pipeline is not currently running")
	}
	cursor := run.Cursor
	property := pipeline.UpdatedAtColumn
	if property == "" || cursor.IsZero() {
		return database.inner.SQLLiteral(nil, types.Type{}), nil
	}
	p, _ := pipeline.InSchema.Properties().ByName(property)
	var value any
	switch p.Type.Kind() {
	case types.StringKind, types.JSONKind:
		value = formatUpdatedAtColumn(pipeline.UpdatedAtFormat, cursor)
	case types.DateTimeKind:
		value = run.Cursor
	case types.DateKind:
		value = time.Date(cursor.Year(), cursor.Month(), cursor.Day(), 0, 0, 0, 0, time.UTC)
	}
	return database.inner.SQLLiteral(value, p.Type), nil
}

// Writer returns a Writer to create and update users.
//
// If the pipeline's output schema does not align with the table's schema, it
// returns a *schemas.Error error.
func (database *Database) Writer(ctx context.Context, pipeline *state.Pipeline, ack AckFunc) (Writer, error) {
	if database.err != nil {
		return nil, database.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	columns, err := database.inner.Columns(ctx, pipeline.TableName)
	if err != nil {
		return nil, connectorError(err)
	}
	properties := columnsProperties(columns, state.Destination)
	if properties == nil {
		return nil, &UnavailableError{Err: fmt.Errorf("table has no supported columns")}
	}
	for i, p := range properties {
		// The table key cannot be nullable. This sets it as not nullable as a
		// temporary workaround, until we can ensure that all connector
		// implementations correctly handle the Nullable attribute for each property
		// (see issue #374).
		if p.Name == pipeline.TableKey {
			properties[i].Nullable = false
		}
	}
	tableSchema, err := types.ObjectOf(properties)
	if err != nil {
		return nil, err
	}
	err = schemas.CheckAlignment(pipeline.OutSchema, tableSchema, nil)
	if err != nil {
		return nil, err
	}
	w := databaseWriter{
		ack: ack,
		table: connectors.Table{
			Name:    pipeline.TableName,
			Columns: columnsOfType(pipeline.OutSchema),
			Keys:    []string{pipeline.TableKey},
		},
		schema: pipeline.OutSchema,
		inner:  database.inner,
	}
	return &w, nil
}

// columnsIssues returns the issues found in the provided columns. It returns a
// nil slice if there are no issues. If a column has a valid type but an issue,
// or an invalid type but no issue, it returns an error.
func columnsIssues(columns []connectors.Column) ([]string, error) {
	var n int
	for i := range columns {
		valid := columns[i].Type.Valid()
		if columns[i].Issue == "" {
			if !valid {
				return nil, fmt.Errorf("has returned column %q with the invalid type but without the issue", columns[i].Name)
			}
			continue
		} else if valid {
			return nil, fmt.Errorf("has returned column %q with a valid type but with an issue", columns[i].Name)
		}
		n++
	}
	if n == 0 {
		return nil, nil
	}
	issues := make([]string, n)
	j := 0
	for i := range columns {
		if columns[i].Issue == "" {
			continue
		}
		issues[j] = columns[i].Issue
		j++
	}
	return issues, nil
}

// columnsOfType returns the properties of a type as connectors.Column values.
func columnsOfType(t types.Type) []connectors.Column {
	properties := t.Properties()
	columns := make([]connectors.Column, properties.Len())
	for i, p := range properties.All() {
		columns[i].Name = p.Name
		columns[i].Type = p.Type
		columns[i].Nullable = p.Nullable
	}
	return columns
}

// columnsProperties returns the provided columns as types.Property values.
// It excludes columns with an invalid type and, if the role is Destination,
// also excludes non-writable columns. If there are no properties to return,
// it returns a nil slice.
func columnsProperties(columns []connectors.Column, role state.Role) []types.Property {
	var n int // number of valid columns to return
	for i := range columns {
		if columns[i].Type.Valid() && (role == state.Source || columns[i].Writable) {
			n++
		}
	}
	if n == 0 {
		return nil
	}
	properties := make([]types.Property, n)
	j := 0
	for i := range columns {
		if !columns[i].Type.Valid() {
			continue
		}
		if role == state.Destination && !columns[i].Writable {
			continue
		}
		properties[j].Name = columns[i].Name
		properties[j].Type = columns[i].Type
		properties[j].Nullable = columns[i].Nullable
		j++
	}
	return properties
}

// databaseWriter implements the Writer interface for databases.
type databaseWriter struct {
	ack    AckFunc
	table  connectors.Table
	schema types.Type
	rows   [][]any
	ids    []string
	inner  databaseConnection
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

func (w *databaseWriter) Write(ctx context.Context, id string, attributes map[string]any) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	// Append the row and the ack ids.
	row := make([]any, len(w.table.Columns))
	for i, c := range w.table.Columns {
		row[i] = attributes[c.Name]
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
	rows        connectors.Rows
	columns     []connectors.Column
	timeLayouts *state.TimeLayouts
	dst         []any
	closed      bool
}

func newRows(rows connectors.Rows, columns []connectors.Column, layouts *state.TimeLayouts) *Rows {
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
		if c.Name == "" {
			rs.dst[i] = queryScanValue{}
		} else {
			rs.dst[i] = queryScanValue{column: c, row: row, timeLayouts: rs.timeLayouts}
		}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, &UnavailableError{Err: err}
	}
	return row, nil
}

// databaseRecords implements the Records interface for databases.
type databaseRecords struct {
	rows        connectors.Rows
	columns     []connectors.Column
	pipeline    *state.Pipeline
	timeLayouts *state.TimeLayouts
	last        bool
	err         error
	closed      bool
}

func newDatabaseRecords(rows connectors.Rows, columns []connectors.Column, pipeline *state.Pipeline, layouts *state.TimeLayouts) *databaseRecords {
	// Unused columns are represented by the zero value of connectors.Column in columns.
	records := databaseRecords{
		rows:        rows,
		columns:     columns,
		pipeline:    pipeline,
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
		run, _ := r.pipeline.Run()
		n := 0 // number of properties per record
		var identityIndex = -1
		var updatedAtIndex = -1
		scanner := scanner{
			values: make([]any, len(r.columns)),
		}
		dest := make([]any, len(r.columns))
		inSchemaProperties := r.pipeline.InSchema.Properties()
		properties := make([]types.Property, len(r.columns))
		for i, c := range r.columns {
			dest[i] = &scanner
			if c.Name == "" { // skip unused columns
				continue
			}
			p, _ := inSchemaProperties.ByName(c.Name)
			properties[i] = p
			if p.Name == r.pipeline.IdentityColumn {
				identityIndex = i
			}
			if p.Name == r.pipeline.UpdatedAtColumn {
				updatedAtIndex = i
			}
			n++
		}
		// Read the rows.
		var record Record
	Rows:
		for r.rows.Next() {
			if record.Attributes != nil || record.Err != nil {
				if !yield(record) {
					return
				}
				record.ID = ""
				record.Attributes = nil
				record.Err = nil
			}
			if err := ctx.Err(); err != nil {
				r.err = err
				return
			}
			if err := r.rows.Scan(dest...); err != nil {
				record.Err = err
				scanner.reset()
				continue Rows
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
			// Get the update time.
			if updatedAtIndex >= 0 {
				v := scanner.values[updatedAtIndex]
				if v == nil {
					record.Err = errors.New("update time value is NULL")
					continue Rows
				}
				p := properties[updatedAtIndex]
				var err error
				record.UpdatedAt, err = parseUpdatedAtColumn(p.Name, p.Type, r.pipeline.UpdatedAtFormat, v, p.Nullable, r.timeLayouts)
				if err != nil {
					record.Err = err
					continue Rows
				}
				if !record.UpdatedAt.IsZero() && record.UpdatedAt.Before(run.Cursor) {
					continue Rows
				}
			}
			if record.UpdatedAt.IsZero() {
				record.UpdatedAt = time.Now().UTC()
			}
			// Get the attributes.
			record.Attributes = make(map[string]any, n)
			for i, v := range scanner.values {
				p := properties[i]
				if p.Name == "" { // skip unused properties
					continue
				}
				value, err := normalize(p.Name, p.Type, v, p.Nullable, r.timeLayouts)
				if err != nil {
					record.Err = err
					continue Rows
				}
				record.Attributes[p.Name] = value
			}
		}
		if record.Attributes != nil || record.Err != nil {
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
	column      connectors.Column
	row         map[string]any
	timeLayouts *state.TimeLayouts
}

func (sv queryScanValue) Scan(src any) error {
	c := sv.column
	if c.Name == "" {
		return nil
	}
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
