// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// Record represents a record.
type Record struct {
	ID         any            // Identifier.
	ExternalID string         // Application external ID.
	Attributes map[string]any // Attributes.
	// Err reports an error that occurred while reading the record.
	// If Err is not nil, only the ID field is significant.
	Err error
}

// Matching specifies criteria for the ProfileRecords method when exporting users
// to an application. It filters users based on whether they have or do not have a match
// with the application users.
type Matching struct {
	Pipeline           int
	InProperty         string
	ExportMode         state.ExportMode
	UpdateOnDuplicates bool
}

// records executes a query on the provided warehouse and returns an iterator to
// iterate on the resulting records. idProperty specifies the property whose
// value is returned as ID, columnByProperty is the mapping from the path of a
// property to the relative column, and omitNil indicates whether attributes
// with a nil value should be omitted from each record.
//
// pipeline and appExport parameters (if specified) represent the pipeline
// identifier and the export options for an application pipeline, respectively.
// When provided, the resulting records are compared against the destination
// users table.
//
// If matching is not nil and a matching application user exists for a record,
// the record's ExternalID will be set to the external ID of the matched
// application user.
func records(ctx context.Context, warehouse warehouses.Warehouse, query Query, idProperty string, columnByProperty map[string]warehouses.Column, omitNil bool, matching *Matching) (*Records, error) {

	columns, unflat := columnsFromProperties(query.Properties, columnByProperty, omitNil)

	var where warehouses.Expr
	if query.Where != nil {
		var err error
		where, err = convertWhere(query.Where, columnByProperty)
		if err != nil {
			return nil, err
		}
	}

	var joins []warehouses.Join
	var orderBy []warehouses.Column
	var orderDesc bool
	var matchingIndex int // index of matching column in columns slice; 0 if matching is nil

	if matching == nil {

		if query.OrderBy != "" {
			c, ok := columnByProperty[query.OrderBy]
			if !ok {
				return nil, fmt.Errorf("property path %s does not exist", query.OrderBy)
			}
			orderBy = []warehouses.Column{c}
			orderDesc = query.OrderDesc
		}

	} else {

		// Also select the _external_id column.
		externalIDColumn := warehouses.Column{Name: "_external_id", Type: types.String(), Nullable: true}
		columns = append(columns, externalIDColumn)
		// Update the WHERE condition and join the krenalis_destination_profiles table.
		inPropertyColumn, ok := columnByProperty[matching.InProperty]
		if !ok {
			return nil, fmt.Errorf("matching property %s does not exist in profile schema", matching.InProperty)
		}
		matchingIndex = slices.IndexFunc(columns, func(c warehouses.Column) bool {
			return c.Name == inPropertyColumn.Name
		})
		if matchingIndex == -1 {
			return nil, fmt.Errorf("matching property %s does not exist in the query properties", matching.InProperty)
		}
		joins = []warehouses.Join{
			{
				Table: "krenalis_destination_profiles",
				Condition: warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{
					warehouses.NewBaseExpr(warehouses.Column{Name: "_pipeline", Type: types.Int(32)}, warehouses.OpIs, matching.Pipeline),
					warehouses.NewBaseExpr(inPropertyColumn, warehouses.OpIs, warehouses.Column{Name: "_out_matching_value", Type: types.String()}),
				}),
			},
		}
		switch matching.ExportMode {
		case state.UpdateOnly:
			// Perform an INNER JOIN to return only users with a matching destination user.
			joins[0].Type = warehouses.InnerJoin
		case state.CreateOnly:
			// Include only users without a corresponding match.
			where = andExpressions(where, warehouses.NewBaseExpr(warehouses.Column{Name: "_pipeline", Type: types.Int(32)}, warehouses.OpIsNull))
			fallthrough
		case state.CreateOrUpdate:
			// Perform a LEFT JOIN to also return users without a matching destination user.
			joins[0].Type = warehouses.LeftJoin
			// Include only users with a value for the input matching property.
			where = andExpressions(where, warehouses.NewBaseExpr(inPropertyColumn, warehouses.OpIsNotNull))
		}
		// Sort the results by the input matching property, user ID, and external ID.
		orderBy = []warehouses.Column{
			inPropertyColumn,
			columnByProperty[idProperty],
			externalIDColumn,
		}
		query.OrderDesc = false

	}

	// Also select the property to be used as the record's ID.
	columns = append(columns, columnByProperty[idProperty])

	rows, _, err := warehouse.Query(ctx, warehouses.RowQuery{
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
		columns:       columns,
		unflat:        unflat,
		rows:          rows,
		matching:      matching,
		matchingIndex: matchingIndex,
	}

	return records, err
}

// andExpressions returns an expression resulting from the AND of expr with
// base.
func andExpressions(expr warehouses.Expr, base *warehouses.BaseExpr) warehouses.Expr {
	if expr == nil {
		return base
	}
	if e, ok := expr.(*warehouses.MultiExpr); ok && e.Operator == warehouses.OpAnd {
		e.Operands = append(e.Operands, base)
		return e
	}
	return warehouses.NewMultiExpr(warehouses.OpAnd, []warehouses.Expr{expr, base})
}

// Records represents records read from the data warehouse.
type Records struct {
	columns       []warehouses.Column
	unflat        unflatRowFunc
	rows          warehouses.Rows
	matching      *Matching
	matchingIndex int
	last          bool
	err           error
	closed        bool
}

// All returns an iterator to iterate over the records. After All completes, it
// is also necessary to check the result of Err for any potential errors.
func (r *Records) All(ctx context.Context) iter.Seq[Record] {

	if r.matching == nil {
		return func(yield func(Record) bool) {
			if r.closed {
				r.err = errors.New("All called on a closed Records")
				return
			}
			defer r.Close()
			// Read the records.
			var previous Record
			last := len(r.columns) - 1
			row := make([]any, len(r.columns))
			for r.rows.Next() {
				if err := ctx.Err(); err != nil {
					r.err = err
					return
				}
				if err := r.rows.Scan(row...); err != nil {
					r.err = err
					return
				}
				if previous.Attributes != nil {
					if !yield(previous) {
						return
					}
				}
				previous = Record{
					ID:         row[last],
					Attributes: r.unflat(row[:last]),
				}
			}
			r.last = true
			if previous.Attributes != nil {
				yield(previous)
			}
			if err := r.rows.Err(); err != nil {
				r.err = err
			}
		}
	}

	return func(yield func(Record) bool) {

		if r.closed {
			r.err = errors.New("All called on a closed Records")
			return
		}
		defer r.Close()

		// previous contains all records previously read that share the same matching property value.
		var previous struct {
			records []Record
			value   any // matching property value
		}

		// yieldPrevious processes previously read records with the same value for the matching property,
		// and calls the yield function. last reports whether this is the last call to yieldPrevious.
		// If it returns false, the iteration should be stopped.
		yieldPrevious := func(last bool) bool {
			if len(previous.records) == 0 {
				return true
			}
			if len(previous.records) == 1 {
				r.last = last
				return yield(previous.records[0])
			}
			// Verify if the previous records have the same user ID.
			sameUserID := true
			id := previous.records[0].ID
			for _, record := range previous.records[1:] {
				if record.ID != id {
					sameUserID = false
					break
				}
			}
			if sameUserID {
				// If updating duplicates is not allowed, return a single record with an error;
				// otherwise, return all the previous records.
				if !r.matching.UpdateOnDuplicates {
					previous.records = previous.records[:1]
					previous.records[0].Err = fmt.Errorf("duplicates found for the matching property %s in the app users", r.matching.InProperty)
				}
				for i, record := range previous.records {
					r.last = last && i == len(previous.records)-1
					if !yield(record) {
						return false
					}
				}
				return true
			}
			// The previous records do not have the same user ID.
			// Remove duplicates and return the records with an error.
			previous.records = slices.CompactFunc(previous.records, func(a, b Record) bool {
				return a.ID == b.ID
			})
			for i, record := range previous.records {
				r.last = last && i == len(previous.records)-1
				record.Err = fmt.Errorf("profile has the same «%s» (the matching property) as other profiles selected for export", r.matching.InProperty)
				if !yield(record) {
					return false
				}
			}
			return true
		}

		// Read the records.
		last := len(r.columns) - 1
		row := make([]any, len(r.columns))
		for r.rows.Next() {
			if err := ctx.Err(); err != nil {
				r.err = err
				return
			}
			if err := r.rows.Scan(row...); err != nil {
				r.err = err
				return
			}
			var value any // value of the matching property
			if r.matching != nil {
				value = row[r.matchingIndex]
			}
			record := Record{
				ID:         row[last],              // the user ID is the last column.
				Attributes: r.unflat(row[:last-1]), // skip the last 2 columns: the external ID and the user ID.
			}
			// If there is no matching application user and the external ID is nil, assign an empty string.
			record.ExternalID, _ = row[last-1].(string)
			// Process the previous records if the value of the matching property differs from the previous records.
			if len(previous.records) > 0 && value != previous.value {
				yieldPrevious(false)
				previous.records = previous.records[0:0]
				previous.value = nil
			}
			previous.records = append(previous.records, record)
			previous.value = value
		}
		// If there was an error, don't process the previous records as they could be incomplete.
		if err := r.rows.Err(); err != nil {
			r.err = err
			return
		}
		yieldPrevious(true)

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
