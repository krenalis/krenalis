//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/core/state"
)

// Query represents a query on a table of a data warehouse.
type Query struct {

	// table is the table to query.
	table string

	// total indicates the total number of rows that match the filter,
	// regardless of the 'first' and 'limit' parameters. It is only meaningful
	// if the method has a parameter that returns the total count.
	total bool

	// Properties are the paths of the properties to return. It cannot be empty
	// and cannot contain overlapped paths.
	Properties []string

	// Where, when not nil, represents the condition that the returned records
	// must satisfy.
	Where *state.Where

	// OrderBy, when non-empty, is the path of property for which the returned
	// rows are ordered.
	OrderBy string

	// OrderDesc, when true and OrderBy is provided, orders the returned records
	// in descending order instead of ascending order.
	OrderDesc bool

	// First is the index of the first returned record and must be >= 0.
	First int

	// Limit controls how many rows should be returned and must be >= 0. If 0,
	// it means that there is no limit.
	Limit int
}

// convertWhere converts a state.Where expression into a meergo.Expr.
// "exists" and "does not exist" operators are mapped to OpIsNotNull and
// OpIsNull, respectively.
func convertWhere(where *state.Where, columnFromProperty map[string]meergo.Column) (meergo.Expr, error) {
	exp := meergo.NewMultiExpr(meergo.LogicalOperator(where.Logical), make([]meergo.Expr, len(where.Conditions)))
	for i, cond := range where.Conditions {
		path := strings.Join(cond.Property, ".") // TODO(marco): How can I avoid this allocation?
		if column, ok := columnFromProperty[path]; ok {
			var op meergo.Operator
			switch cond.Operator {
			case state.OpExists:
				op = meergo.OpIsNotNull
			case state.OpDoesNotExist:
				op = meergo.OpIsNull
			default:
				op = meergo.Operator(cond.Operator)
			}
			exp.Operands[i] = meergo.NewBaseExpr(column, op, cond.Values...)
			continue
		}
		// The property is an object; apply it to all sub-property columns.
		var logical meergo.LogicalOperator
		var op meergo.Operator
		switch cond.Operator {
		case state.OpExists:
			logical = meergo.OpOr
			op = meergo.OpIsNotNull
		case state.OpDoesNotExist:
			logical = meergo.OpAnd
			op = meergo.OpIsNull
		default:
			return nil, fmt.Errorf("invalid operator %q for property %q in where expression", cond.Operator, path)
		}
		var operands []meergo.Expr
		for name, column := range columnFromProperty {
			if strings.HasPrefix(name, path) && name[len(path)] == '.' {
				operands = append(operands, meergo.NewBaseExpr(column, op))
			}
		}
		if operands == nil {
			return nil, fmt.Errorf("property %q does not exist in where expression", path)
		}
		slices.SortFunc(operands, func(a, b meergo.Expr) int {
			return cmp.Compare(a.(*meergo.BaseExpr).Column.Name, b.(*meergo.BaseExpr).Column.Name)
		})
		exp.Operands[i] = meergo.NewMultiExpr(logical, operands)
	}
	return exp, nil
}
