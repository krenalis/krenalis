//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"fmt"
	"strings"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/state"
)

// Query represents a query on a table of a data warehouse.
type Query struct {

	// table is the table to query.
	table string

	// count retrieves the total number of rows that match the filter,
	// irrespective of the first and limit parameters. It is meaningful only if
	// the method has a count return parameter.
	count bool

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

// exprFromWhere returns a warehouses.Expr expression from a where.
func exprFromWhere(where *state.Where, columnFromProperty map[string]warehouses.Column) (warehouses.Expr, error) {
	exp := warehouses.NewMultiExpr(warehouses.LogicalOperator(where.Logical), make([]warehouses.Expr, len(where.Conditions)))
	for i, cond := range where.Conditions {
		path := strings.Join(cond.Property, ".") // TODO(marco): How can I avoid this allocation?
		column, ok := columnFromProperty[path]
		if !ok {
			return nil, fmt.Errorf("property path %s does not exist", path)
		}
		exp.Operands[i] = warehouses.NewBaseExpr(column, warehouses.Operator(cond.Operator), cond.Values...)
	}
	return exp, nil
}
