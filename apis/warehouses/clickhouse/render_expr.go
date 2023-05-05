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

	wh "chichi/apis/warehouses"
	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/open2b/nuts/decimal"
)

// renderExpr renders the expression expr returning a fragment of a query
// representing a boolean expression.
func renderExpr(expr wh.Expr) (string, error) {

	s := strings.Builder{}

	// Handle MultiExpr expression.
	if multiExpr, ok := expr.(*wh.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case wh.LogicalOperatorAnd:
			op = " AND "
		case wh.LogicalOperatorOr:
			op = " OR "
		default:
			return "", fmt.Errorf("invalid operator %q", multiExpr.Operator)
		}
		for i, operand := range multiExpr.Operands {
			if i > 0 {
				s.WriteString(op)
			}
			_, isMultiExpr := operand.(*wh.MultiExpr)
			if isMultiExpr {
				s.WriteByte('(')
			}
			e, err := renderExpr(operand)
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
	baseExpr := expr.(*wh.BaseExpr)

	// Validate the column name.
	if !wh.IsValidIdentifier(baseExpr.Column.Name) {
		return "", fmt.Errorf("invalid column name %q", baseExpr.Column.Name)
	}

	// Render the column identifier.
	s.WriteByte('`')
	s.WriteString(baseExpr.Column.Name)
	s.WriteString("` ")

	// Render the operator and, if necessary, the value.
	switch baseExpr.Operator {
	case
		wh.OperatorEqual,
		wh.OperatorNotEqual,
		wh.OperatorGreater,
		wh.OperatorGreaterEqual,
		wh.OperatorLess,
		wh.OperatorLessEqual:

		switch baseExpr.Operator {
		case wh.OperatorEqual:
			s.WriteString("= ")
		case wh.OperatorNotEqual:
			s.WriteString("<> ")
		case wh.OperatorGreater:
			s.WriteString("> ")
		case wh.OperatorGreaterEqual:
			s.WriteString(">= ")
		case wh.OperatorLess:
			s.WriteString("< ")
		case wh.OperatorLessEqual:
			s.WriteString("<= ")
		}

		switch pt := baseExpr.Column.Type; pt {
		case
			types.PtBoolean:
			b, ok := baseExpr.Value.(bool)
			if !ok {
				return "", fmt.Errorf("expecting value of type bool, got %T", baseExpr.Value)
			}
			quoteValue(&s, b)
		case
			types.PtInt,
			types.PtInt8,
			types.PtInt16,
			types.PtInt24,
			types.PtInt64:
			i, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			quoteValue(&s, i)
		case
			types.PtUInt,
			types.PtUInt8,
			types.PtUInt16,
			types.PtUInt24,
			types.PtUInt64:
			u, ok := baseExpr.Value.(uint)
			if !ok {
				return "", fmt.Errorf("expecting value of type uint, got %T", baseExpr.Value)
			}
			quoteValue(&s, u)
		case
			types.PtFloat,
			types.PtFloat32:
			f, ok := baseExpr.Value.(float64)
			if !ok {
				return "", fmt.Errorf("expecting value of type float64, got %T", baseExpr.Value)
			}
			quoteValue(&s, f)
		case types.PtDecimal:
			d, ok := baseExpr.Value.(decimal.Dec)
			if !ok {
				return "", fmt.Errorf("expecting value of type decimal.Dec, got %T", baseExpr.Value)
			}
			s.WriteString(d.String())
		case types.PtDateTime:
			t, ok := baseExpr.Value.(_connector.DateTime)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.DateTime, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.DateTime))
		case types.PtDate:
			t, ok := baseExpr.Value.(_connector.Date)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.Date, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.String())
		case types.PtTime:
			t, ok := baseExpr.Value.(_connector.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.Time, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.TimeOnly))
		case types.PtYear:
			year, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			quoteValue(&s, year)
		case types.PtUUID, types.PtInet, types.PtText:
			u, ok := baseExpr.Value.(string)
			if !ok {
				return "", fmt.Errorf("expecting value of type uuid.UUID, got %T", baseExpr.Value)
			}
			quoteValue(&s, u)
		case types.PtJSON:
			j, ok := baseExpr.Value.(json.RawMessage)
			if !ok {
				return "", fmt.Errorf("expecting value of type json.RawMessage, got %T", baseExpr.Value)
			}
			quoteValue(&s, string(j))
		default:
			return "", fmt.Errorf("unexpected column with type %q", pt)
		}
	case wh.OperatorIsNull:
		s.WriteString("IS NULL")
	case wh.OperatorIsNotNull:
		s.WriteString("IS NOT NULL")
	default:
		return "", fmt.Errorf("invalid operator %q", baseExpr.Operator)
	}

	return s.String(), nil

}
