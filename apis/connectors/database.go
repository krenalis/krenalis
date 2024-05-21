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

	"github.com/google/uuid"
	"github.com/open2b/chichi"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"
)

// Database represents the database of a database connection.
type Database struct {
	closed      bool
	connector   string
	timeLayouts *state.TimeLayouts
	inner       chichi.Database
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
	database.inner, database.err = chichi.RegisteredDatabase(connector.Name).New(&chichi.DatabaseConfig{
		Role:        chichi.Role(connection.Role),
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
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a PlaceholderError error may be returned in case of
// an error with placeholders.
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
		return nil, err
	}
	return newRows(rows, columns, database.timeLayouts), nil
}

// Records executes the action's query and returns an iterator to iterate over
// the database's records. Each returned record will contain, in the Properties
// field, the properties of the action's input schema, with the same types.
//
// If queryReplacer is not nil, then the placeholders in the query are replaced
// using it; in this case, a PlaceholderError error may be returned in case of
// an error with placeholders.
//
// If the action's input schema, which must be valid, does not conform to the
// query's results schema, it returns a *SchemaError error.
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
		return nil, err
	}
	var records Records
	defer func() {
		if records == nil {
			_ = rows.Close()
		}
	}()
	// Validate the identity and the last change time properties.
	var identityProperty, lastChangeTimeProperty types.Property
	for _, c := range columns {
		if c.Name == action.IdentityProperty {
			property, _ := action.InSchema.Property(action.IdentityProperty)
			if c.Type.Kind() != property.Type.Kind() {
				return nil, &SchemaError{""}
			}
			identityProperty = property
		}
		if action.LastChangeTimeProperty != "" && c.Name == action.LastChangeTimeProperty {
			property, _ := action.InSchema.Property(action.LastChangeTimeProperty)
			if c.Type.Kind() != property.Type.Kind() {
				return nil, &SchemaError{fmt.Sprintf(`last change time property %q has type %s instead of %s`,
					action.LastChangeTimeProperty, c.Type.Kind(), property.Type.Kind())}
			}
			lastChangeTimeProperty = property
		}
	}
	if identityProperty.Name == "" {
		return nil, &SchemaError{fmt.Sprintf("there is no identity property %q", action.IdentityProperty)}
	}
	if action.LastChangeTimeProperty != "" && lastChangeTimeProperty.Name == "" {
		return nil, &SchemaError{fmt.Sprintf("there is no last change time property %q", action.LastChangeTimeProperty)}
	}
	// Check that schema is aligned with the query's schema.
	querySchema, err := types.ObjectOf(columns)
	if err != nil {
		return nil, fmt.Errorf("connector %s has returned invalid columns: %s", database.connector, err)
	}
	err = checkSchemaAlignment(action.InSchema, querySchema)
	if err != nil {
		return nil, err
	}

	// Determine the displayed property, if necessary.
	var displayedProperty types.Property
	if action.DisplayedProperty != "" {
		displayedProperty, err = displayedPropertyFromSchema(querySchema, action.DisplayedProperty)
		if err != nil {
			slog.Warn("cannot determine the displayed property", "err", err)
		}
	}

	// Return the records.
	records = newDatabaseRecords(rows, columns, types.Properties(action.InSchema), identityProperty,
		lastChangeTimeProperty, action.LastChangeTimeFormat, displayedProperty, database.timeLayouts)
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
	columns := append([]types.Property{{Name: "id", Type: types.Int(32)}}, types.Properties(schema)...)
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
	inner   chichi.Database
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

func (w *databaseWriter) Write(ctx context.Context, gid uuid.UUID, record Record) bool {
	if w.closed {
		panic("connectors: Write called on a closed writer")
	}
	record.Properties["id"] = gid.String()
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
	gids := make([]uuid.UUID, len(w.rows))
	for i, row := range w.rows {
		gids[i] = uuid.MustParse(row["id"].(string))
	}
	w.ack(err, gids)
	w.rows = slices.Delete(w.rows, 0, len(w.rows))
}

// Rows is the result of a query.
type Rows struct {
	rows        chichi.Rows
	columns     []types.Property
	timeLayouts *state.TimeLayouts
	dst         []any
	closed      bool
}

func newRows(rows chichi.Rows, columns []types.Property, layouts *state.TimeLayouts) *Rows {
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
		rs.dst[i] = queryScanValue{column: c, row: row, timeLayouts: rs.timeLayouts}
	}
	if err := rs.rows.Scan(rs.dst...); err != nil {
		return nil, err
	}
	return row, nil
}

// queryScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type queryScanValue struct {
	column      types.Property
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

// databaseRecords implements the Records interface for databases.
type databaseRecords struct {
	columns              []types.Property
	rows                 chichi.Rows
	propertyOf           map[string]types.Property
	dst                  []any
	identityProperty     types.Property
	lastChangeTime       types.Property
	lastChangeTimeFormat string
	displayedProperty    types.Property
	timeLayouts          *state.TimeLayouts
	last                 bool
	err                  error
	closed               bool
}

func newDatabaseRecords(rows chichi.Rows, columns, properties []types.Property,
	identityProperty, lastChangeTimeProperty types.Property, lastChangeTimeFormat string,
	displayedProperty types.Property, layouts *state.TimeLayouts) *databaseRecords {
	records := databaseRecords{
		columns:              columns,
		rows:                 rows,
		dst:                  make([]any, len(columns)),
		propertyOf:           make(map[string]types.Property, len(properties)),
		identityProperty:     identityProperty,
		lastChangeTime:       lastChangeTimeProperty,
		lastChangeTimeFormat: lastChangeTimeFormat,
		displayedProperty:    displayedProperty,
		timeLayouts:          layouts,
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

func (r *databaseRecords) Last() bool {
	return r.last
}

func (r *databaseRecords) Seq() Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("connectors: For called on a closed Records")
			return
		}
		var record Record
		defer r.Close()
		for r.rows.Next() {
			if record.Properties != nil || record.Err != nil {
				if !yield(record) {
					return
				}
			}
			record = Record{
				Properties: make(map[string]any, len(r.propertyOf)),
			}
			for i, c := range r.columns {
				p := r.propertyOf[c.Name]
				if c.Name == r.displayedProperty.Name {
					// This is necessary as the displayed property is not
					// necessarily included in "propertyOf"; even if it is, the type
					// of its property must be taken from the query.
					p = r.displayedProperty
				}
				r.dst[i] = recordsScanValue{
					property:             p,
					record:               &record,
					identityProperty:     r.identityProperty,
					lastChangeTime:       r.lastChangeTime,
					lastChangeTimeFormat: r.lastChangeTimeFormat,
					displayedProperty:    r.displayedProperty,
					timeLayouts:          r.timeLayouts,
				}
			}
			if err := r.rows.Scan(r.dst...); err != nil {
				r.err = err
				return
			}
			if record.LastChangeTime.IsZero() {
				record.LastChangeTime = time.Now().UTC()
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

// recordsScanValue implements the sql.Scanner interface to read the database
// values from a database connector.
type recordsScanValue struct {
	property             types.Property
	record               *Record
	identityProperty     types.Property
	lastChangeTime       types.Property
	lastChangeTimeFormat string
	displayedProperty    types.Property
	timeLayouts          *state.TimeLayouts
}

func (sv recordsScanValue) Scan(src any) error {
	p := sv.property

	if !p.Type.Valid() {
		return nil
	}

	if p.Name == sv.displayedProperty.Name {
		col := sv.displayedProperty
		normalizedValue, err := normalize(col.Name, col.Type, src, col.Nullable, sv.timeLayouts)
		if err != nil {
			slog.Warn("displayed property value cannot be normalized", "err", err)
		} else {
			dp, err := displayedPropertyToString(normalizedValue)
			if err != nil {
				slog.Warn("invalid displayed property value", "err", err)
			} else {
				sv.record.DisplayedProperty = dp
			}
		}
	}

	switch p.Name {
	case sv.identityProperty.Name:
		if src == nil {
			return errors.New("identity value is NULL")
		}
		id, err := parseIdentityProperty(p.Name, p.Type, src, sv.timeLayouts)
		if err != nil {
			return err
		}
		sv.record.ID = id
		return nil
	case sv.lastChangeTime.Name:
		if src == nil {
			return errors.New("last change time value is NULL")
		}
		lastChangeTime, err := parseTimestampColumn(p.Name, p.Type, sv.lastChangeTimeFormat, src, sv.timeLayouts)
		if err != nil {
			return err
		}
		sv.record.LastChangeTime = lastChangeTime
		return nil
	}
	value, err := normalize(p.Name, p.Type, src, p.Nullable, sv.timeLayouts)
	if err != nil {
		return err
	}
	sv.record.Properties[p.Name] = value
	return nil
}
