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
	"github.com/krenalis/krenalis/tools/errors"
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
// "exists" and "does not exist" operators are mapped to warehouses.OpIsNotNull
// and warehouses.OpIsNull, respectively. For object properties, "exists"
// matches if any descendant column is non-null, while "does not exist" matches
// if all descendant columns are null.
func convertWhere(where *state.Where, columnByProperty map[string]warehouses.Column) (warehouses.Expr, error) {

	if where == nil {
		return nil, errors.New("where expression cannot be nil")
	}
	if where.Operator != state.OpAnd && where.Operator != state.OpOr {
		return nil, fmt.Errorf("invalid logical operator %d in where expression", int(where.Operator))
	}

	// state.WhereLogical values match the corresponding
	// warehouses.LogicalOperator values.
	expr := warehouses.NewMultiExpr(
		warehouses.LogicalOperator(where.Operator),
		make([]warehouses.Expr, len(where.Rules)),
	)

	for i, rule := range where.Rules {

		switch rule := rule.(type) {

		case *state.Where:
			if rule == nil {
				return nil, errors.New("where group cannot be nil")
			}
			operand, err := convertWhere(rule, columnByProperty)
			if err != nil {
				return nil, err
			}
			expr.Operands[i] = operand

		case *state.WhereCondition:
			if rule == nil {
				return nil, errors.New("where condition cannot be nil")
			}
			path := strings.Join(rule.Property, ".")
			if rule.Operator < state.OpIs || rule.Operator > state.OpDoesNotExist {
				return nil, fmt.Errorf("invalid operator %d for property %q in where expression",
					int(rule.Operator), path)
			}
			if column, ok := columnByProperty[path]; ok {
				var op warehouses.Operator
				switch rule.Operator {
				case state.OpExists:
					op = warehouses.OpIsNotNull
				case state.OpDoesNotExist:
					op = warehouses.OpIsNull
				default:
					// state.WhereOperator values through state.OpIsNotNull match the
					// corresponding warehouses.Operator values.
					op = warehouses.Operator(rule.Operator)
				}
				expr.Operands[i] = warehouses.NewBaseExpr(column, op, rule.Values...)
				continue
			}
			var descendantColumns []warehouses.Column
			n := len(path)
			for name, column := range columnByProperty {
				if len(name) > n && name[n] == '.' && name[:n] == path {
					descendantColumns = append(descendantColumns, column)
				}
			}
			if len(descendantColumns) == 0 {
				return nil, fmt.Errorf("property %q does not map to any warehouse columns", path)
			}
			// The property is an object; apply the condition to all descendant columns.
			var logicalOp warehouses.LogicalOperator
			var op warehouses.Operator
			switch rule.Operator {
			case state.OpExists:
				logicalOp = warehouses.OpOr
				op = warehouses.OpIsNotNull
			case state.OpDoesNotExist:
				logicalOp = warehouses.OpAnd
				op = warehouses.OpIsNull
			default:
				return nil, fmt.Errorf("operator %q cannot be used with object property %q in where expression",
					rule.Operator, path)
			}
			slices.SortFunc(descendantColumns, func(a, b warehouses.Column) int {
				return cmp.Compare(a.Name, b.Name)
			})
			operands := make([]warehouses.Expr, len(descendantColumns))
			for j, column := range descendantColumns {
				operands[j] = warehouses.NewBaseExpr(column, op)
			}
			expr.Operands[i] = warehouses.NewMultiExpr(logicalOp, operands)

		default:
			return nil, errors.New("unsupported where rule")

		}

	}

	return expr, nil
}
