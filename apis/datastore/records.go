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

	"github.com/meergo/meergo"
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
	columns   []meergo.Column
	normalize meergo.NormalizeFunc
	unflat    unflatRowFunc
	rows      meergo.Rows
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

// SchemaError represents an error with a schema.
type SchemaError struct {
	Msg string
}

func (err *SchemaError) Error() string {
	return err.Msg
}

// records executes a query on the data warehouse and returns an iterator to
// iterate on the resulting records. idProperty specifies the property whose
// value is returned as ID, columnByProperty is the mapping from the path of a
// property to the relative column, and omitNil indicates whether properties
// with a nil value should be omitted from each record.
//
// It returns, in Record.MatchingID, the matching ID if matching is not nil.
func (store *Store) records(ctx context.Context, query Query, idProperty string, columnByProperty map[string]meergo.Column, omitNil bool, matching *Matching) (*Records, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty, omitNil)

	var where meergo.Expr
	if query.Where != nil {
		var err error
		where, err = exprFromWhere(query.Where, columnByProperty)
		if err != nil {
			return nil, err
		}
	}

	var joins []meergo.Join

	if matching != nil {
		c, ok := columnByProperty[matching.Property]
		if !ok {
			return nil, fmt.Errorf("matching property %s does not exist in user schema", matching.Property)
		}
		columns = append(columns, meergo.Column{Name: "__user__", Type: types.Text(), Nullable: true})
		joins = []meergo.Join{
			{
				Type:  meergo.Inner,
				Table: "_destinations_users",
				Condition: meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIs, matching.Action),
					meergo.NewBaseExpr(c, meergo.OpIs, meergo.Column{Name: "__property__", Type: types.Text()}),
				}),
			},
		}
		if matching.ExportMode == state.CreateOnly || matching.ExportMode == state.CreateOrUpdate {
			// Use a Left JOIN instead.
			joins[0].Type = meergo.Left
			// Add 'property IS NOT NULL' to the WHERE condition to exclude users with a NULL value for the matching property.
			expr := meergo.NewBaseExpr(c, meergo.OpIsNotNull)
			if where == nil {
				where = expr
			} else if w, ok := where.(*meergo.MultiExpr); ok && w.Operator == meergo.OpAnd {
				w.Operands = append(w.Operands, expr)
			} else {
				where = meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{expr, where})
			}
			if matching.ExportMode == state.CreateOnly {
				// Add '__action__ IS NULL' to the WHERE condition to include only users without a corresponding match.
				expr = meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIsNull)
				if w, ok := where.(*meergo.MultiExpr); ok && w.Operator == meergo.OpAnd {
					w.Operands = append(w.Operands, expr)
				} else {
					where = meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{expr, where})
				}
			}
		}
		query.OrderBy = matching.Property
		query.OrderDesc = false
	}

	var orderBy meergo.Column
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

	rows, _, err := store.warehouse.Query(ctx, meergo.RowQuery{
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

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	columns   []meergo.Column
	row       []any
	normalize meergo.NormalizeFunc
	index     int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []meergo.Column, row []any, normalize meergo.NormalizeFunc) []any {
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
