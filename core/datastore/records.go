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
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/types"
)

// Record represents a record.
type Record struct {
	ID         any            // Identifier.
	ExternalID string         // App external ID.
	Properties map[string]any // Properties.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// Matching specifies criteria for the UserRecords method when exporting users
// to an app. It filters users based on whether they have or do not have a match
// with the app users.
type Matching struct {
	Action             int
	InProperty         string
	ExportMode         state.ExportMode
	ExportOnDuplicates bool
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
// action and appExport parameters (if specified) represent the action
// identifier and the export options for an app action, respectively. When
// provided, the resulting records are compared against the destination users
// table.
//
// If matching is not nil and a matching app user exists for a record, the
// record's ExternalID will be set to the external ID of the matched app user.
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
		// Also select the __external_id__ column.
		columns = append(columns, meergo.Column{Name: "__external_id__", Type: types.Text(), Nullable: true})
		// Update the WHERE condition and join the _destinations_users table.
		inPropertyColumn, ok := columnByProperty[matching.InProperty]
		if !ok {
			return nil, fmt.Errorf("matching property %s does not exist in user schema", matching.InProperty)
		}
		joins = []meergo.Join{
			{
				Table: "_destinations_users",
				Condition: meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{
					meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIs, matching.Action),
					meergo.NewBaseExpr(inPropertyColumn, meergo.OpIs, meergo.Column{Name: "__out_matching_value__", Type: types.Text()}),
				}),
			},
		}
		switch matching.ExportMode {
		case state.UpdateOnly:
			// Perform an INNER JOIN to return only users with a matching destination user.
			joins[0].Type = meergo.InnerJoin
		case state.CreateOnly:
			// Include only users without a corresponding match.
			where = andExpressions(where, meergo.NewBaseExpr(meergo.Column{Name: "__action__", Type: types.Int(32)}, meergo.OpIsNull))
			fallthrough
		case state.CreateOrUpdate:
			// Perform a LEFT JOIN to also return users without a matching destination user.
			joins[0].Type = meergo.LeftJoin
			// Include only users with a value for the input matching property.
			where = andExpressions(where, meergo.NewBaseExpr(inPropertyColumn, meergo.OpIsNotNull))
		}
		// Sort the results by the input matching property.
		query.OrderBy = matching.InProperty
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

	// Also select the property to be used as the record's ID.
	columns = append(columns, columnByProperty[idProperty])

	rows, _, err := store.warehouse().Query(ctx, meergo.RowQuery{
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
		columns:  columns,
		unflat:   unflat,
		rows:     rows,
		matching: matching,
	}

	return records, err
}

// andExpressions returns an expression resulting from the AND of expr with
// base.
func andExpressions(expr meergo.Expr, base *meergo.BaseExpr) meergo.Expr {
	if expr == nil {
		return base
	}
	if e, ok := expr.(*meergo.MultiExpr); ok && e.Operator == meergo.OpAnd {
		e.Operands = append(e.Operands, base)
		return e
	}
	return meergo.NewMultiExpr(meergo.OpAnd, []meergo.Expr{expr, base})
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
		i := 0
		for r.rows.Next() {
			select {
			case <-ctx.Done():
				r.err = ctx.Err()
				return
			default:
			}
			if err := r.rows.Scan(row...); err != nil {
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
			// If the record has a matching app user, check for duplicate entries.
			if externalID := row[last-1]; externalID != nil {
				record.ExternalID = externalID.(string)
				if id == previous.ID {
					record.Err = fmt.Errorf("duplicates found for the matching property %q in the exported users", r.matching.InProperty)
				} else if i > 0 && !r.matching.ExportOnDuplicates && record.ExternalID == previous.ExternalID {
					record.Err = fmt.Errorf("duplicates found for the matching property %q in the app users", r.matching.InProperty)
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
