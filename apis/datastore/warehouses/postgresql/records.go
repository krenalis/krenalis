//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/types"
)

// Records returns an iterator over the results of the query.
func (warehouse *PostgreSQL) Records(ctx context.Context, query warehouses.RecordsQuery) (warehouses.Records, int, error) {

	// TODO(Gianluca): this code will be reviewed / rewritten / simplified as we
	// resolve issue 708.

	if !query.ID.Type.Valid() {
		return nil, 0, errors.New("invalid ID type")
	}
	if query.ID.Type.Kind() != types.IntKind {
		// TODO(Gianluca): see https://github.com/open2b/chichi/issues/555.
		return nil, 0, fmt.Errorf("expecting ID with Int kind, got %s", query.ID.Type.Kind())
	}
	if len(query.Properties) == 0 {
		return nil, 0, errors.New("properties cannot be empty")
	}
	if !warehouses.IsValidIdentifier(query.Table) {
		return nil, 0, fmt.Errorf("table name %q is not a valid identifier", query.Table)
	}
	if query.OrderBy.Name != "" && !types.IsValidPropertyName(query.OrderBy.Name) {
		return nil, 0, fmt.Errorf("order property name %q is not a valid property name", query.OrderBy.Name)
	}
	if !query.Schema.Valid() {
		return nil, 0, errors.New("schema must be valid")
	}

	// Transform the ID and the OrderBy properties into columns.
	idColumn := warehouses.PropertiesToColumns([]types.Property{query.ID})[0]
	var orderByColumn types.Property
	if query.OrderBy.Name != "" {
		orderByColumn = warehouses.PropertiesToColumns([]types.Property{query.OrderBy})[0]
	}

	db, err := warehouse.connection()
	if err != nil {
		return nil, 0, err
	}

	// Add the ID and the OrderBy properties to the schema.
	var schema types.Type
	{
		props := query.Schema.Properties()
		var hasID, hasOrderBy bool
		for _, p := range props {
			if p.Name == query.ID.Name {
				hasID = true
			}
			if p.Name == query.OrderBy.Name {
				hasOrderBy = true
			}
			if hasID && (hasOrderBy || query.OrderBy.Name == "") {
				break
			}
		}
		if !hasID {
			props = append(props, query.ID)
		}
		if !hasOrderBy && query.OrderBy.Name != "" && query.OrderBy.Name != query.ID.Name {
			props = append(props, query.OrderBy)
		}
		schema = types.Object(props)
	}

	// Determine the properties and the columns for the query.
	var properties, columns []types.Property
	{
		for _, path := range query.Properties {
			// TODO(Gianluca): this can be optimized to avoid fetching
			// unnecessary sub-properties from the data warehouse.
			p, ok := schema.Property(path[0])
			if !ok {
				return nil, 0, fmt.Errorf("property %q not found within query.Schema", path[0])
			}
			properties = append(properties, p)
		}
		columns = warehouses.PropertiesToColumns(properties)
	}
	hasID := false
	for _, c := range columns {
		if c.Name == idColumn.Name {
			hasID = true
			break
		}
	}
	removeIDFromProps := false
	if !hasID {
		columns = append([]types.Property{idColumn}, columns...)
		properties = append([]types.Property{idColumn}, properties...)
		// Since the ID has been added to the columns just to determine the
		// records IDs, and it is now explicitly requested by the user, it must
		// be removed from the returned properties.
		removeIDFromProps = true
	}

	// Build the WHERE expression, if necessary.
	var whereExpr string
	if query.Where != nil {
		whereExpr, err = renderExpr(schema, query.Where)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot build WHERE expression: %s", err)
		}
	}

	// Build and execute the COUNT query to determine the count of records.
	var count int
	var b strings.Builder
	b.WriteString(`SELECT COUNT(*) FROM "`)
	b.WriteString(query.Table)
	b.WriteByte('"')
	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}
	err = db.QueryRow(ctx, b.String()).Scan(&count)
	if err != nil {
		return nil, 0, warehouses.Error(err)
	}

	// Build the query.
	b.Reset()
	b.WriteString(`SELECT `)
	for i, c := range columns {
		if i > 0 {
			b.WriteString(", ")
		}
		if !types.IsValidPropertyName(c.Name) {
			return nil, 0, fmt.Errorf("column name %q is not a valid property name", c.Name)
		}
		b.WriteByte('"')
		b.WriteString(c.Name)
		b.WriteByte('"')
	}
	b.WriteString(` FROM "`)
	b.WriteString(query.Table)
	b.WriteByte('"')
	if query.Where != nil {
		b.WriteString(` WHERE `)
		b.WriteString(whereExpr)
	}

	if orderByColumn.Name != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(orderByColumn.Name)
		if query.OrderDesc {
			b.WriteString(" DESC")
		}
	}
	if query.Limit > 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(strconv.Itoa(query.Limit))
	}
	if query.First > 0 {
		b.WriteString(" OFFSET ")
		b.WriteString(strconv.Itoa(query.First))
	}

	// Execute the query.
	rows, err := db.Query(ctx, b.String())
	if err != nil {
		return nil, 0, warehouses.Error(err)
	}

	records, err := newRecords(rows, columns, properties, query.ID.Name, removeIDFromProps)
	if err != nil {
		return nil, 0, err
	}

	return records, count, nil
}

type records struct {
	columns           []types.Property
	properties        []types.Property
	rows              *postgres.Rows
	id                string
	dst               []any
	err               error
	closed            bool
	removeIDFromProps bool
}

var _ warehouses.Records = (*records)(nil)

// newRecords return a new records.
// It could change the columns slice and the column names.
// id is the name of the property used as Record.ID.
// removeIDFromProps controls whether the ID property should be removed from the
// Record.Properties.
func newRecords(rows *postgres.Rows, columns, properties []types.Property, id string, removeIDFromProps bool) (*records, error) {
	records := records{
		columns:           columns,
		properties:        properties,
		rows:              rows,
		id:                id,
		dst:               make([]any, len(columns)),
		removeIDFromProps: removeIDFromProps,
	}
	return &records, nil
}

func (r *records) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.rows.Close()
	return nil
}

func (r *records) Err() error {
	return r.err
}

func (r *records) For(yield func(warehouses.Record) error) error {
	if r.closed {
		r.err = errors.New("for called on a closed Records")
		return nil
	}
	defer r.Close()
	var rows [][]any
	values := newScanValues(r.columns, &rows)
	for r.rows.Next() {
		if err := r.rows.Scan(values...); err != nil {
			r.err = err
			return nil
		}
		var record warehouses.Record
		props, _ := warehouses.DeserializeRowAsMap(r.properties, rows[len(rows)-1])
		id, ok := props[r.id].(int)
		if !ok {
			r.err = fmt.Errorf("row has no integer ID %q", r.id)
			return nil
		}
		record.ID = id
		if r.removeIDFromProps {
			delete(props, r.id)
		}
		record.Properties = props
		if err := yield(record); err != nil {
			return err
		}
	}
	if err := r.rows.Err(); err != nil {
		r.err = err
	}
	return nil
}
