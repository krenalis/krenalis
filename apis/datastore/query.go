//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package datastore

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
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
	Where *Where

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

// Where represents a condition in a query.
type Where struct {
	Logical    WhereLogical     // can be "all" or "any".
	Conditions []WhereCondition // cannot be empty.
}

// WhereLogical represents the logical operator of a where.
// It can be "all" or "any".
type WhereLogical string

// WhereCondition represents the condition of a where.
type WhereCondition struct {
	Property string // A property identifier or selector (e.g. "street1" or "traits.address.street1").
	Operator string // "is", "is not".
	Value    string // "Track", "Page", ...
}

// exprFromWhere returns a warehouses.Expr expression from a where.
func exprFromWhere(where *Where, columnFromProperty map[string]warehouses.Column) (warehouses.Expr, error) {
	op := warehouses.OpAnd
	if where.Logical == "any" {
		op = warehouses.OpOr
	}
	exp := warehouses.NewMultiExpr(op, make([]warehouses.Expr, len(where.Conditions)))
	for i, cond := range where.Conditions {
		column, ok := columnFromProperty[cond.Property]
		if !ok {
			return nil, fmt.Errorf("property path %s does not exist", cond.Property)
		}
		var op warehouses.Operator
		switch cond.Operator {
		case "is":
			op = warehouses.OpEqual
		case "is not":
			op = warehouses.OpNotEqual
		default:
			return nil, errors.New("invalid operator")
		}
		var value any
		switch column.Type.Kind() {
		case types.BooleanKind:
			value = false
			if cond.Value == "true" {
				value = true
			}
		case types.IntKind:
			value, _ = strconv.Atoi(cond.Value)
		case types.UintKind:
			v, _ := strconv.ParseUint(cond.Value, 10, 64)
			value = uint(v)
		case types.FloatKind:
			value, _ = strconv.ParseFloat(cond.Value, 64)
		case types.DecimalKind:
			value = decimal.RequireFromString(cond.Value)
		case types.DateTimeKind:
			value, _ = time.Parse(time.DateTime, cond.Value)
		case types.DateKind:
			value, _ = time.Parse(time.DateOnly, cond.Value)
		case types.TimeKind:
			value, _ = time.Parse("15:04:05.999999999", cond.Value)
		case types.YearKind:
			value, _ = strconv.Atoi(cond.Value)
		case types.UUIDKind, types.TextKind:
			value = cond.Value
		case types.JSONKind:
			value = json.RawMessage(cond.Value)
		case types.InetKind:
			value, _ = netip.ParseAddr(cond.Value)
		default:
			return nil, fmt.Errorf("unexpected type %s", column.Type)
		}
		exp.Operands[i] = warehouses.NewBaseExpr(column, op, value)
	}
	return exp, nil
}
