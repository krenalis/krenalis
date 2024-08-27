//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"

	"github.com/shopspring/decimal"
)

// renderExpr renders the expression expr, which refers to the properties in
// schema, returning a fragment of a query representing a boolean expression.
func renderExpr(b *strings.Builder, exp warehouses.Expr) error {

	// Handle MultiExpr expression.
	if multiExpr, ok := exp.(*warehouses.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case warehouses.LogicalOperatorAnd:
			op = " AND "
		case warehouses.LogicalOperatorOr:
			op = " OR "
		default:
			return fmt.Errorf("invalid operator %q", multiExpr.Operator)
		}
		for i, operand := range multiExpr.Operands {
			if i > 0 {
				b.WriteString(op)
			}
			_, isMultiExpr := operand.(*warehouses.MultiExpr)
			if isMultiExpr {
				b.WriteByte('(')
			}
			err := renderExpr(b, operand)
			if err != nil {
				return err
			}
			if isMultiExpr {
				b.WriteByte(')')
			}
		}
		return nil
	}

	// Handle BaseExpr expressions.
	baseExpr := exp.(*warehouses.BaseExpr)
	c := baseExpr.Column

	// Validate the column name.
	if !warehouses.IsValidIdentifier(c.Name) {
		return fmt.Errorf("invalid property name %q", c.Name)
	}

	// Render the column identifier.
	b.WriteByte('`')
	b.WriteString(c.Name)
	b.WriteString("` ")

	// Render the operator and, if necessary, the value.
	switch baseExpr.Operator {
	case
		warehouses.OperatorEqual,
		warehouses.OperatorNotEqual,
		warehouses.OperatorGreater,
		warehouses.OperatorGreaterEqual,
		warehouses.OperatorLess,
		warehouses.OperatorLessEqual:

		switch baseExpr.Operator {
		case warehouses.OperatorEqual:
			b.WriteString("= ")
		case warehouses.OperatorNotEqual:
			b.WriteString("<> ")
		case warehouses.OperatorGreater:
			b.WriteString("> ")
		case warehouses.OperatorGreaterEqual:
			b.WriteString(">= ")
		case warehouses.OperatorLess:
			b.WriteString("< ")
		case warehouses.OperatorLessEqual:
			b.WriteString("<= ")
		}
		serializeValue(b, baseExpr.Value)

	case warehouses.OperatorIsNull:
		b.WriteString("IS NULL")

	case warehouses.OperatorIsNotNull:
		b.WriteString("IS NOT NULL")

	case warehouses.OperatorNotIn:
		b.WriteString(" NOT ")
		fallthrough

	case warehouses.OperatorIn:
		b.WriteString("IN (")
		for i, v := range baseExpr.Value.([]any) {
			if i > 0 {
				b.WriteByte(',')
			}
			serializeValue(b, v)
		}
		b.WriteString(")")

	}

	return nil
}

// serializeValue serializes v into b.
// As special case, v can have type warehouse.Column.
func serializeValue(b *strings.Builder, v any) {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case warehouses.Column:
		b.WriteByte('`')
		b.WriteString(v.Name)
		b.WriteByte('`')
	case bool:
		if v {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int16:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		b.WriteString(strconv.FormatInt(v, 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		b.WriteString(strconv.FormatUint(v, 10))
	case float32:
		b.WriteString(strconv.FormatFloat(float64(v), 'G', -1, 32))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case decimal.Decimal:
		b.WriteString(v.String())
	case netip.Addr:
		quoteString(b, v.String())
	case string:
		quoteString(b, v)
	case time.Time:
		b.WriteByte('\'')
		b.WriteString(v.Format("2006-01-02 15:04:05"))
		b.WriteByte('\'')
	}
}
