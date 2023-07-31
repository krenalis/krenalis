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

	"chichi/apis/datastore/warehouses"
	"chichi/connector/types"

	"github.com/open2b/nuts/decimal"
)

// renderExpr renders the expression expr returning a fragment of a query
// representing a boolean expression.
func renderExpr(expr warehouses.Expr) (string, error) {

	s := strings.Builder{}

	// Handle MultiExpr expression.
	if multiExpr, ok := expr.(*warehouses.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case warehouses.LogicalOperatorAnd:
			op = " AND "
		case warehouses.LogicalOperatorOr:
			op = " OR "
		default:
			return "", fmt.Errorf("invalid operator %q", multiExpr.Operator)
		}
		for i, operand := range multiExpr.Operands {
			if i > 0 {
				s.WriteString(op)
			}
			_, isMultiExpr := operand.(*warehouses.MultiExpr)
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
	baseExpr := expr.(*warehouses.BaseExpr)

	// Validate the column name.
	if !warehouses.IsValidIdentifier(baseExpr.Column.Name) {
		return "", fmt.Errorf("invalid column name %q", baseExpr.Column.Name)
	}

	// Render the column identifier.
	s.WriteByte('`')
	s.WriteString(baseExpr.Column.Name)
	s.WriteString("` ")

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
			s.WriteString("= ")
		case warehouses.OperatorNotEqual:
			s.WriteString("<> ")
		case warehouses.OperatorGreater:
			s.WriteString("> ")
		case warehouses.OperatorGreaterEqual:
			s.WriteString(">= ")
		case warehouses.OperatorLess:
			s.WriteString("< ")
		case warehouses.OperatorLessEqual:
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
			t, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.DateTime, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.DateTime))
		case types.PtDate:
			t, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type connector.Date, got %T", baseExpr.Value)
			}
			quoteValue(&s, t.Format(time.DateTime))
		case types.PtTime:
			t, ok := baseExpr.Value.(time.Time)
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
	case warehouses.OperatorIsNull:
		s.WriteString("IS NULL")
	case warehouses.OperatorIsNotNull:
		s.WriteString("IS NOT NULL")
	default:
		return "", fmt.Errorf("invalid operator %q", baseExpr.Operator)
	}

	return s.String(), nil

}
