//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package datastore

import (
	"context"
	"errors"
	"fmt"

	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"
)

// Records is the iterator interface used to iterate over the records read from
// a data warehouse.
type Records interface {

	// Close closes the iterator. It is automatically called by the For method
	// before returning. Close is idempotent and does not impact the result of Err.
	Close() error

	// Err returns any error encountered during iteration, excluding errors returned
	// by the yield function, which may have occurred after an explicit or implicit
	// Close.
	Err() error

	// For calls the yield function for each record (r) in the sequence. If yield
	// returns an error, For stops and returns the error. After For completes, it
	// is also necessary to check the result of Err for any potential errors.
	For(yield func(Record) error) error
}

// Record represents a record.
type Record struct {
	ID         int            // Identifier.
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// queryParams holds information to pass to the 'records' method.
type queryParams struct {

	// Table is the table to query.
	Table string

	// TableSchema is the schema with the properties of the table specified in
	// Table.
	TableSchema types.Type

	// IDColumn is the name of the column which contains the ID returned in the
	// records. It cannot be the empty string.
	IDColumn string

	// Properties are the properties to return for each record in the
	// Record.Properties field.
	Properties []types.Path

	// Where, when not nil, filters the records to return.
	Where expr.Expr

	// OrderBy, when provided, is the name of property for which the returned
	// records are ordered.
	OrderBy string

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many records should be returned and must be >= 0. If
	// 0, it means that there is no limit.
	Limit int
}

// records performs a query on the store and return its result as a Records
// iterator.
func (store *Store) records(ctx context.Context, query queryParams) (Records, int, error) {

	// Determine the properties.
	var properties []types.Property
	for _, path := range query.Properties {
		p, ok := query.TableSchema.Property(path[0])
		if !ok {
			return nil, 0, fmt.Errorf("property %q not found in the schema of the table %q", path[0], query.Table)
		}
		properties = append(properties, p)
	}

	// Determine the columns to query.
	columns := warehouses.PropertiesToColumns(properties)
	var columnNames []string
	for _, c := range columns {
		columnNames = append(columnNames, c.Name)
	}

	// If the ID is not already present in the columns, add it.
	hasID := false
	for _, c := range columns {
		if c.Name == query.IDColumn {
			hasID = true
			break
		}
	}
	removeIDFromProps := false
	if !hasID {
		id, ok := query.TableSchema.Property(query.IDColumn)
		if !ok {
			return nil, 0, fmt.Errorf("ID column %q not found in the schema of the table %q", query.IDColumn, query.Table)
		}
		columns = append([]types.Property{id}, columns...)
		columnNames = append([]string{query.IDColumn}, columnNames...)
		properties = append([]types.Property{id}, properties...)
		// Ensure that the ID is subsequently removed from the properties, as it
		// is only required by the driver to determine the ID and should not be
		// returned.
		removeIDFromProps = true
	}

	// Transform the properties of the table schema to columns.
	tableColumnsSchema := types.Object(
		warehouses.PropertiesToColumns(query.TableSchema.Properties()),
	)

	rows, count, err := store.warehouse.Query(ctx, warehouses.RowQuery{
		Columns:            columnNames,
		Table:              query.Table,
		TableColumnsSchema: tableColumnsSchema,
		// TODO(Gianluca): see the issue
		// https://github.com/open2b/chichi/issues/727, where we revise the
		// "where" expressions and the filters.
		Where:     query.Where,
		OrderBy:   query.OrderBy,
		OrderDesc: query.OrderDesc,
		First:     query.First,
		Limit:     query.Limit,
	})
	if err != nil {
		return nil, 0, err
	}

	records, err := newRecords(rows, columns, properties,
		query.IDColumn, store.warehouse.Normalize, removeIDFromProps)
	if err != nil {
		return nil, 0, err
	}

	return records, count, err
}

var _ Records = (*records)(nil)

type records struct {
	columns           []types.Property
	properties        []types.Property
	rows              warehouses.Rows
	id                string
	dst               []any
	err               error
	closed            bool
	normalize         NormalizeFunc
	removeIDFromProps bool
}

// newRecords return a new records.
// It could change the columns slice and the column names.
// id is the name of the property used as Record.ID.
func newRecords(rows warehouses.Rows, columns, properties []types.Property, id string, normalize NormalizeFunc, removeIDFromProps bool) (*records, error) {
	records := records{
		columns:           columns,
		properties:        properties,
		rows:              rows,
		id:                id,
		dst:               make([]any, len(columns)),
		normalize:         normalize,
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

func (r *records) For(yield func(Record) error) error {
	if r.closed {
		r.err = errors.New("for called on a closed Query")
		return nil
	}
	defer r.Close()
	row := make([]any, len(r.columns))
	values := newScanValues(r.columns, row, r.normalize)
	for r.rows.Next() {
		if err := r.rows.Scan(values...); err != nil {
			r.err = err
			return nil
		}
		var record Record
		props, _ := warehouses.DeserializeRowAsMap(r.properties, row)
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

// NormalizeFunc is a function type representing the normalization function
// exposed by data warehouse drivers to normalize values returned by them.
type NormalizeFunc func(name string, typ types.Type, v any, nullable bool) (any, error)

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	columns   []types.Property
	row       []any
	normalize NormalizeFunc
	index     int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []types.Property, row []any, normalize NormalizeFunc) []any {
	values := make([]any, len(columns))
	value := &scanValue{
		columns:   columns,
		row:       row,
		normalize: normalize,
	}
	for i := range columns {
		values[i] = value
	}
	return values
}

func (sv *scanValue) Scan(src any) error {
	c := sv.columns[sv.index]
	value, err := sv.normalize(c.Name, c.Type, src, c.Nullable)
	if err != nil {
		return err
	}
	sv.row[sv.index] = value
	sv.index = (sv.index + 1) % len(sv.columns)
	return nil
}
