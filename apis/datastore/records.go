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
	"iter"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"
)

// Record represents a record.
type Record struct {
	ID         any            // Identifier.
	MatchingID string         // Matching identifier.
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// Records represents records read from the data warehouse.
type Records struct {
	columns   []warehouses.Column
	normalize NormalizeFunc
	unflat    unflatRowFunc
	rows      warehouses.Rows
	matching  *Matching
	last      bool
	err       error
	closed    bool
}

// Matching specifies criteria for the UserRecords method when exporting users
// to an app. It filters users based on whether they have or do not have a match
// with the app users.
type Matching struct {
	Action          int
	Property        string
	ExportMode      state.ExportMode
	AllowDuplicates bool
}

// records executes a query on the data warehouse and returns an iterator to
// iterate on the resulting records. idProperty specifies the property whose
// value is returned as ID, columnByProperty is the mapping from the path of a
// property to the relative column, and omitNil indicates whether properties
// with a nil value should be omitted from each record.
//
// It returns, in Record.MatchingID, the matching ID if matching is not nil.
func (store *Store) records(ctx context.Context, query Query, idProperty string, columnByProperty map[string]warehouses.Column, omitNil bool, matching *Matching) (*Records, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty, omitNil)

	var where warehouses.Expr
	if query.Where != nil {
		var err error
		where, err = exprFromWhere(query.Where, columnByProperty)
		if err != nil {
			return nil, err
		}
	}

	var joins []warehouses.Join

	if matching != nil {
		c, ok := columnByProperty[matching.Property]
		if !ok {
			return nil, fmt.Errorf("matching property %s does not exist in user schema", matching.Property)
		}
		columns = append(columns, warehouses.Column{Name: "__user__", Type: types.Text(), Nullable: true})
		joins = []warehouses.Join{
			{
				Type:  warehouses.Left,
				Table: "_destinations_users",
				Condition: warehouses.NewMultiExpr(warehouses.LogicalOperatorAnd, []warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "__action__", Type: types.Int(32)}, warehouses.OperatorEqual, matching.Action),
					warehouses.NewBaseExpr(c, warehouses.OperatorEqual, warehouses.Column{Name: "__property__", Type: types.Text()}),
				}),
			},
		}
		switch matching.ExportMode {
		case state.UpdateOnly:
			joins[0].Type = warehouses.Inner
		case state.CreateOnly:
			expr := warehouses.NewBaseExpr(warehouses.Column{Name: "__action__", Type: types.Int(32)}, warehouses.OperatorIsNull, nil)
			if where == nil {
				where = expr
			} else {
				where = warehouses.NewMultiExpr(warehouses.LogicalOperatorAnd, []warehouses.Expr{expr, where})
			}
		}
		query.OrderBy = matching.Property
		query.OrderDesc = false
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

	columns = append(columns, columnByProperty[idProperty])

	rows, _, err := store.warehouse.Query(ctx, warehouses.RowQuery{
		Columns:   columns,
		Table:     query.table,
		Joins:     joins,
		Where:     where,
		OrderBy:   orderBy,
		OrderDesc: orderDesc,
		First:     query.First,
		Limit:     query.Limit,
	}, false)
	if err != nil {
		return nil, err
	}

	records := &Records{
		columns:   columns,
		unflat:    unflat,
		rows:      rows,
		normalize: store.warehouse.Normalize,
		matching:  matching,
	}

	return records, err
}

// All returns an iterator to iterate over the records. After All completes, it
// is also necessary to check the result of Err for any potential errors.
func (r *Records) All(ctx context.Context) iter.Seq[Record] {
	return func(yield func(Record) bool) {
		if r.closed {
			r.err = errors.New("All called on a closed Records")
			return
		}
		defer r.Close()
		var record Record
		last := len(r.columns) - 1
		row := make([]any, len(r.columns))
		values := newScanValues(r.columns, row, r.normalize)
		i := 0
		for r.rows.Next() {
			select {
			case <-ctx.Done():
				r.err = ctx.Err()
				return
			default:
			}
			if err := r.rows.Scan(values...); err != nil {
				r.err = err
				return
			}
			id := row[last]
			if r.matching == nil {
				if i > 0 {
					if !yield(record) {
						return
					}
				}
				record = Record{
					ID:         id,
					Properties: r.unflat(row[:last]),
				}
				i++
				continue
			}
			previous := record
			record = Record{ID: id}
			if v := row[last-1]; v != nil {
				record.MatchingID = v.(string)
				if id == previous.ID {
					record.Err = fmt.Errorf("duplicates found for the matching property %q in the exported users", r.matching.Property)
				} else if i > 0 && !r.matching.AllowDuplicates && record.MatchingID == previous.MatchingID {
					record.Err = fmt.Errorf("duplicates found for the matching property %q in the app users", r.matching.Property)
				}
			}
			if record.Err == nil {
				record.Properties = r.unflat(row[:last-1])
			}
			if i > 0 {
				if record.Err != nil && previous.Err == nil {
					previous.Properties = nil
					previous.Err = record.Err
				}
				if !yield(previous) {
					return
				}
			}
			i++
		}
		if i > 0 {
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

// SchemaError represents an error with a schema.
type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}
