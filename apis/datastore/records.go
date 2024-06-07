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

// Seq is an iterator over sequences of individual values.
type Seq[V any] func(yield func(V) bool)

// Record represents a record.
type Record struct {
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// records executes a query on the data warehouse and returns an iterator to
// iterate on the resulting records. schema is the schema of the properties in
// Properties and Filter of query, and columnByProperty is the mapping from the
// path of a property to the relative column.
func (store *Store) records(ctx context.Context, query Query, schema types.Type, columnByProperty map[string]warehouses.Column) (*Records, error) {

	if err := checkSchemaAlignment(schema, columnByProperty); err != nil {
		return nil, err
	}

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty)

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

	records := &Records{
		columns:   columns,
		unflat:    unflat,
		rows:      rows,
		normalize: store.warehouse.Normalize,
	}

	return records, err
}

// Records represents records read from the data warehouse.
type Records struct {
	columns   []warehouses.Column
	normalize NormalizeFunc
	unflat    unflatRowFunc
	rows      warehouses.Rows
	last      bool
	err       error
	closed    bool
}

// All returns an iterator to iterate over the records. After All completes, it
// is also necessary to check the result of Err for any potential errors.
func (r *Records) All(ctx context.Context) Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("All called on a closed Records")
			return
		}
		defer r.Close()
		var record Record
		row := make([]any, len(r.columns))
		values := newScanValues(r.columns, row, r.normalize)
		for r.rows.Next() {
			select {
			case <-ctx.Done():
				r.err = ctx.Err()
				return
			default:
			}
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
				Properties: r.unflat(row),
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

// Close closes the iterator. It is automatically called by the For method
// before returning. Close is idempotent and does not impact the result of Err.
func (r *Records) Close() error {
	if r.closed {
		return nil
	}
	r.closed = true
	r.rows.Close()
	return nil
}

// Err returns any error encountered during iteration which may have occurred
// after an explicit or implicit Close.
func (r *Records) Err() error {
	return r.err
}

// Last reports whether the last record has been read.
func (r *Records) Last() bool {
	return r.last
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
