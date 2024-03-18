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
	"log/slog"
	"slices"
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
		SetSettings: setConnectionSettingsFunc(connectors.state, connection),
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
func (database *Database) Records(ctx context.Context, query string, schema types.Type, businessIDColumnName string) (Records, error) {
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
			property, _ := schema.Property("id")
			if c.Type.Kind() != property.Type.Kind() {
				return nil, &SchemaError{fmt.Sprintf(`identity column "id" has type %s instead of %s`, c.Type.Kind(), property.Type.Kind())}
			}
			hasIdentityColumn = true
		case "timestamp":
			if c.Type.Kind() != types.DateTimeKind {
				return nil, &SchemaError{fmt.Sprintf(`timestamp column "timestamp" has type %s instead of DateTime`, c.Type.Kind())}
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

	// Determine the Business ID, if necessary.
	var businessIDColumn types.Property
	if businessIDColumnName != "" {
		businessIDColumn, err = businessIDFromSchema(querySchema, businessIDColumnName)
		if err != nil {
			slog.Warn("cannot determine the Business ID column", "err", err)
		}
	}

	// Return the records.
	records = newDatabaseRecords(rows, columns, schema.Properties(), businessIDColumn)
	return records, nil
}

// Writer returns a Writer to create and update users.
func (database *Database) Writer(table string, schema types.Type, ack AckFunc) (Writer, error) {
	if database.err != nil {
		return nil, database.err
	}
	if ack == nil {
		return nil, errors.New("ack function is missing")
	}
	columns := append([]types.Property{{Name: "id", Type: types.Int(32)}}, schema.Properties()...)
	w := databaseWriter{
		ack:     ack,
		table:   table,
		schema:  schema,
		columns: columns,
		inner:   database.inner,
	}
	return &w, nil
}

// databaseWriter implements the Writer interface for databases.
type databaseWriter struct {
	ack     AckFunc
	table   string
	schema  types.Type
	columns []types.Property
	rows    []map[string]any
	inner   _connector.DatabaseConnection
	closed  bool
}

func (w *databaseWriter) Close(ctx context.Context) error {
	if w.closed {
		return nil
	}
	if len(w.rows) > 0 {
		w.upsert(ctx)
	}
	w.closed = true
	return nil
}

func (w *databaseWriter) Write(ctx context.Context, gid int, record Record) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	record.Properties["id"] = gid
	// Append the row.
	w.rows = append(w.rows, record.Properties)
	// Upsert the rows.
	if len(w.rows) == 100 {
		w.upsert(ctx)
	}
	return true
}

// upsert calls the Upsert method of the database connector with the collected
// records.
func (w *databaseWriter) upsert(ctx context.Context) {
	err := w.inner.Upsert(ctx, w.table, w.rows, w.columns)
	gids := make([]int, len(w.rows))
	for i, row := range w.rows {
		gids[i] = row["id"].(int)
	}
	w.ack(err, gids)
	w.rows = slices.Delete(w.rows, 0, len(w.rows))
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
		rs.dst[i] = queryScanValue{column: c, row: row}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, err
	}
	return row, nil
}

// queryScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type queryScanValue struct {
	column types.Property
	row    map[string]any
}

func (sv queryScanValue) Scan(src any) error {
	c := sv.column
	value, err := normalizeDatabaseFileProperty(c.Name, c.Type, src, c.Nullable)
	if err != nil {
		return err
	}
	sv.row[c.Name] = value
	return nil
}

// databaseRecords implements the Records interface for databases.
type databaseRecords struct {
	columns          []types.Property
	rows             _connector.Rows
	propertyOf       map[string]types.Property
	dst              []any
	businessIDColumn types.Property
	err              error
	closed           bool
}

func newDatabaseRecords(rows _connector.Rows, columns, properties []types.Property, businessIDColumn types.Property) *databaseRecords {
	records := databaseRecords{
		columns:          columns,
		rows:             rows,
		dst:              make([]any, len(columns)),
		propertyOf:       make(map[string]types.Property, len(properties)),
		businessIDColumn: businessIDColumn,
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
			r.dst[i] = recordsScanValue{
				property:         r.propertyOf[c.Name],
				record:           &record,
				businessIDColumn: r.businessIDColumn,
			}
		}
		if err := r.rows.Scan(r.dst...); err != nil {
			r.err = err
			return nil
		}
		if record.Timestamp.IsZero() {
			record.Timestamp = time.Now().UTC()
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

// recordsScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type recordsScanValue struct {
	property         types.Property
	record           *Record
	businessIDColumn types.Property
}

func (sv recordsScanValue) Scan(src any) error {
	p := sv.property
	if !p.Type.Valid() {
		return nil
	}

	switch p.Name {
	case "id":
		if src == nil {
			return errors.New("identity value is NULL")
		}
		id, err := parseIdentityColumn(p.Name, p.Type, src)
		if err != nil {
			return err
		}
		sv.record.ID = id
		return nil
	case "timestamp":
		if src == nil {
			return errors.New("timestamp value is NULL")
		}
	case sv.businessIDColumn.Name:
		col := sv.businessIDColumn
		normalizedValue, err := normalizeDatabaseFileProperty(col.Name, col.Type, src, col.Nullable)
		if err != nil {
			slog.Warn("Business ID value cannot be normalized", "err", err)
		} else {
			businessID, err := businessIDToString(normalizedValue)
			if err != nil {
				slog.Warn("invalid Business ID value", "err", err)
			} else {
				sv.record.BusinessID = businessID
			}
		}
	}
	value, err := normalizeDatabaseFileProperty(p.Name, p.Type, src, p.Nullable)
	if err != nil {
		return err
	}
	sv.record.Properties[p.Name] = value
	if p.Name == "timestamp" {
		timestamp := value.(time.Time)
		if err := validateTimestamp(timestamp); err != nil {
			return err
		}
		sv.record.Timestamp = timestamp
	}
	return nil
}
