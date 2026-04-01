// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/warehouses"
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

// convertWhere converts a state.Where expression into a warehouses.Expr.
// "exists" and "does not exist" operators are mapped to OpIsNotNull and
// OpIsNull, respectively.
func convertWhere(where *state.Where, columnFromProperty map[string]warehouses.Column) (warehouses.Expr, error) {
	exp := warehouses.NewMultiExpr(warehouses.LogicalOperator(where.Logical), make([]warehouses.Expr, len(where.Conditions)))
	for i, cond := range where.Conditions {
		path := strings.Join(cond.Property, ".") // TODO(marco): How can I avoid this allocation?
		if column, ok := columnFromProperty[path]; ok {
			var op warehouses.Operator
			switch cond.Operator {
			case state.OpExists:
				op = warehouses.OpIsNotNull
			case state.OpDoesNotExist:
				op = warehouses.OpIsNull
			default:
				op = warehouses.Operator(cond.Operator)
			}
			exp.Operands[i] = warehouses.NewBaseExpr(column, op, cond.Values...)
			continue
		}
		// The property is an object; apply it to all sub-property columns.
		var logical warehouses.LogicalOperator
		var op warehouses.Operator
		switch cond.Operator {
		case state.OpExists:
			logical = warehouses.OpOr
			op = warehouses.OpIsNotNull
		case state.OpDoesNotExist:
			logical = warehouses.OpAnd
			op = warehouses.OpIsNull
		default:
			return nil, fmt.Errorf("invalid operator %q for property %q in where expression", cond.Operator, path)
		}
		var operands []warehouses.Expr
		for name, column := range columnFromProperty {
			if strings.HasPrefix(name, path) && name[len(path)] == '.' {
				operands = append(operands, warehouses.NewBaseExpr(column, op))
			}
		}
		if operands == nil {
			return nil, fmt.Errorf("property %q does not exist in where expression", path)
		}
		slices.SortFunc(operands, func(a, b warehouses.Expr) int {
			return cmp.Compare(a.(*warehouses.BaseExpr).Column.Name, b.(*warehouses.BaseExpr).Column.Name)
		})
		exp.Operands[i] = warehouses.NewMultiExpr(logical, operands)
	}
	return exp, nil
}
