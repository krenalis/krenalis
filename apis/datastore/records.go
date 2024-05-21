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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"
)

// Seq represents a sequence of V values.
type Seq[V any] func(yield func(V) bool)

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

	// Last reports whether the last record has been read.
	Last() bool

	// Seq returns an iterator to iterate over the records. After Seq completes, it
	// is also necessary to check the result of Err for any potential errors.
	Seq() Seq[Record]
}

// Record represents a record.
type Record struct {
	ID         int            // Identifier, whose value must fall within an Int(32).
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// records executes a query on the data warehouse and returns a Records iterator
// to iterate on the resulting records. schema is the schema of the properties
// in Properties and Filter of query, and columnByProperty is the mapping from
// the path of a property to the relative column.
func (store *Store) records(ctx context.Context, query Query, schema types.Type, columnByProperty map[string]warehouses.Column) (Records, error) {

	if err := checkSchemaAlignment(schema, columnByProperty); err != nil {
		return nil, err
	}

	columns, explode := columnsFromProperties(query.Properties, columnByProperty)
	columns = append(columns, columnByProperty[query.id])

	var where warehouses.Expr
	if query.Filter != nil {
		var err error
		where, err = exprFromFilter(query.Filter, columnByProperty)
		if err != nil {
			return nil, err
		}
	}

	var orderBy warehouses.Column
	var orderDesc bool
	if query.OrderBy != "" {
		var ok bool
		orderBy, ok = columnByProperty[query.OrderBy]
		if !ok {
			return nil, fmt.Errorf("property path %s does not exist", query.OrderBy)
		}
		orderDesc = query.OrderDesc
	}

	rows, _, err := store.warehouse.Query(ctx, warehouses.RowQuery{
		Columns:   columns,
		Table:     query.table,
		Where:     where,
		OrderBy:   orderBy,
		OrderDesc: orderDesc,
		First:     query.First,
		Limit:     query.Limit,
	})
	if err != nil {
		return nil, err
	}

	records := &records{
		columns:   columns,
		explode:   explode,
		rows:      rows,
		normalize: store.warehouse.Normalize,
	}

	return records, err
}

var _ Records = (*records)(nil)

type records struct {
	columns   []warehouses.Column
	normalize NormalizeFunc
	explode   explodeRowFunc
	rows      warehouses.Rows
	last      bool
	err       error
	closed    bool
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

func (r *records) Last() bool {
	return r.last
}

func (r *records) Seq() Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("Seq called on a closed Query")
			return
		}
		defer r.Close()
		var record Record
		last := len(r.columns) - 1
		row := make([]any, len(r.columns))
		values := newScanValues(r.columns, row, r.normalize)
		for r.rows.Next() {
			if record.Properties != nil || record.Err != nil {
				if !yield(record) {
					return
				}
			}
			if err := r.rows.Scan(values...); err != nil {
				record = Record{Err: err}
				continue
			}
			record = Record{
				ID:         row[last].(int),
				Properties: r.explode(row),
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
		return
	}
}

// NormalizeFunc is a function type representing the normalization function
// exposed by data warehouse drivers to normalize values returned by them.
type NormalizeFunc func(name string, typ types.Type, v any, nullable bool) (any, error)

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	columns   []warehouses.Column
	row       []any
	normalize NormalizeFunc
	index     int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []warehouses.Column, row []any, normalize NormalizeFunc) []any {
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

// SchemaError represents an error with a schema.
type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}

// checkSchemaAlignment checks whether schema is aligned with the properties and
// types of columnByProperty. It returns a *SchemaError error if it is not
// aligned. It panics if a schema is not valid.
func checkSchemaAlignment(schema types.Type, columnByProperty map[string]warehouses.Column) error {
	for path, p := range types.Walk(schema) {
		if p.Type.Kind() == types.ObjectKind {
			continue
		}
		c, ok := columnByProperty[path]
		if !ok {
			return &SchemaError{Msg: fmt.Sprintf(`%q property no longer exists`, path)}
		}
		if !p.Type.EqualTo(c.Type) {
			return &SchemaError{Msg: fmt.Sprintf(`type of the %q property has been changed from %s to %s`, path, c.Type, p.Type)}
		}
	}
	return nil
}
