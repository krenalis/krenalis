//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// renderExpr renders the expression expr, which refers to the properties in
// schema, returning a fragment of a query representing a boolean expression.
func renderExpr(schema types.Type, exp expr.Expr) (string, error) {

	s := strings.Builder{}

	// Handle MultiExpr expression.
	if multiExpr, ok := exp.(*expr.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case expr.LogicalOperatorAnd:
			op = " AND "
		case expr.LogicalOperatorOr:
			op = " OR "
		default:
			return "", fmt.Errorf("invalid operator %q", multiExpr.Operator)
		}
		for i, operand := range multiExpr.Operands {
			if i > 0 {
				s.WriteString(op)
			}
			_, isMultiExpr := operand.(*expr.MultiExpr)
			if isMultiExpr {
				s.WriteByte('(')
			}
			e, err := renderExpr(schema, operand)
			if err != nil {
				return "", err
			}
			s.WriteString(e)
			if isMultiExpr {
				s.WriteByte(')')
			}
		}
		return s.String(), nil
	}

	// Handle BaseExpr expressions.
	baseExpr := exp.(*expr.BaseExpr)

	// Validate the column name.
	if !warehouses.IsValidIdentifier(baseExpr.Property) {
		return "", fmt.Errorf("invalid property name %q", baseExpr.Property)
	}

	column, err := warehouses.PropertyPathToColumn(schema, baseExpr.Property)
	if err != nil {
		return "", fmt.Errorf("property %q not found", baseExpr.Property)
	}

	// Render the column identifier.
	s.WriteByte('`')
	s.WriteString(column.Name)
	s.WriteString("` ")

	// Render the operator and, if necessary, the value.
	switch baseExpr.Operator {
	case
		expr.OperatorEqual,
		expr.OperatorNotEqual,
		expr.OperatorGreater,
		expr.OperatorGreaterEqual,
		expr.OperatorLess,
		expr.OperatorLessEqual:

		switch baseExpr.Operator {
		case expr.OperatorEqual:
			s.WriteString("= ")
		case expr.OperatorNotEqual:
			s.WriteString("<> ")
		case expr.OperatorGreater:
			s.WriteString("> ")
		case expr.OperatorGreaterEqual:
			s.WriteString(">= ")
		case expr.OperatorLess:
			s.WriteString("< ")
		case expr.OperatorLessEqual:
			s.WriteString("<= ")
		}

		property, err := schema.PropertyByPath(types.Path{baseExpr.Property})
		if err != nil {
			return "", fmt.Errorf("property %q not found", baseExpr.Property)
		}

		switch k := property.Type.Kind(); k {
		case types.BooleanKind:
			b, ok := baseExpr.Value.(bool)
			if !ok {
				return "", fmt.Errorf("expecting value of type bool, got %T", baseExpr.Value)
			}
			quoteValue(&s, b)
		case types.IntKind:
			i, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			quoteValue(&s, i)
		case types.UintKind:
			u, ok := baseExpr.Value.(uint)
			if !ok {
				return "", fmt.Errorf("expecting value of type uint, got %T", baseExpr.Value)
			}
			quoteValue(&s, u)
		case types.FloatKind:
			f, ok := baseExpr.Value.(float64)
			if !ok {
				return "", fmt.Errorf("expecting value of type float64, got %T", baseExpr.Value)
			}
			quoteValue(&s, f)
		case types.DecimalKind:
			d, ok := baseExpr.Value.(decimal.Decimal)
			if !ok {
				return "", fmt.Errorf("expecting value of type decimal.Dec, got %T", baseExpr.Value)
			}
			s.WriteString(d.String())
		case types.DateTimeKind:
			t, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.DateTime, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.DateTime))
		case types.DateKind:
			t, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.Date, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.DateTime))
		case types.TimeKind:
			t, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.Time, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.TimeOnly))
		case types.YearKind:
			year, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			quoteValue(&s, year)
		case types.UUIDKind, types.InetKind, types.TextKind:
			u, ok := baseExpr.Value.(string)
			if !ok {
				return "", fmt.Errorf("expecting value of type uuid.UUID, got %T", baseExpr.Value)
			}
			quoteValue(&s, u)
		case types.JSONKind:
			j, ok := baseExpr.Value.(json.RawMessage)
			if !ok {
				return "", fmt.Errorf("expecting value of type json.RawMessage, got %T", baseExpr.Value)
			}
			quoteValue(&s, string(j))
		default:
			return "", fmt.Errorf("unexpected column with type %q", k)
		}
	case expr.OperatorIsNull:
		s.WriteString("IS NULL")
	case expr.OperatorIsNotNull:
		s.WriteString("IS NOT NULL")
	default:
		return "", fmt.Errorf("invalid operator %q", baseExpr.Operator)
	}

	return s.String(), nil

}
