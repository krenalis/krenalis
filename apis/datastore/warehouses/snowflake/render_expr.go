//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

// renderExpr renders the expression expr returning a fragment of a query
// representing a boolean expression.
func renderExpr(exp warehouses.Expr) (string, error) {

	s := strings.Builder{}

	// Handle MultiExpr expression.
	if multiExpr, ok := exp.(*warehouses.MultiExpr); ok {
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
	baseExpr := exp.(*warehouses.BaseExpr)
	c := baseExpr.Column

	// Validate the column name.
	if !warehouses.IsValidIdentifier(c.Name) {
		return "", fmt.Errorf("invalid property name %q", c.Name)
	}

	// Render the column identifier.
	s.WriteString(postgres.QuoteIdent(c.Name))
	s.WriteString(" ")

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

		switch k := c.Type.Kind(); k {
		case
			types.BooleanKind:
			v, ok := baseExpr.Value.(bool)
			if !ok {
				return "", fmt.Errorf("expecting value of type bool, got %T", baseExpr.Value)
			}
			if v {
				s.WriteString("TRUE")
			} else {
				s.WriteString("FALSE")
			}
		case
			types.FloatKind:
			v, ok := baseExpr.Value.(float64)
			if !ok {
				return "", fmt.Errorf("expecting value of type float64, got %T", baseExpr.Value)
			}
			s.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
		case types.DecimalKind:
			d, ok := baseExpr.Value.(decimal.Decimal)
			if !ok {
				return "", fmt.Errorf("expecting value of type decimal.Dec, got %T", baseExpr.Value)
			}
			s.WriteString(d.String())
		case types.DateTimeKind:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format("2006-01-02 15:04:05.999999999"))
			s.WriteByte('\'')
		case types.DateKind:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format(time.DateTime))
			s.WriteByte('\'')
		case types.TimeKind:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format("15:04:05.999999999"))
			s.WriteByte('\'')
		case types.JSONKind:
			s.WriteString("PARSE_JSON(")
			switch v := baseExpr.Value.(type) {
			case json.RawMessage:
				quoteBytes(&s, v)
			case json.Number:
				quoteString(&s, string(v))
			case bool, string, float64, map[string]any, []any:
				var b bytes.Buffer
				enc := json.NewEncoder(&b)
				enc.SetEscapeHTML(false)
				err := enc.Encode(v)
				if err != nil {
					return "", err
				}
				b.Truncate(b.Len() - 1) // remove the trailing new line.
				quoteBytes(&s, b.Bytes())
			default:
				return "", fmt.Errorf("expecting value of type json.RawMessage, json.Number, bool, string, float64, map[string]any, or []any but got %T", baseExpr.Value)
			}
			s.WriteByte(')')
		case types.TextKind:
			v, ok := baseExpr.Value.(string)
			if !ok {
				return "", fmt.Errorf("expecting value of type string, got %T", baseExpr.Value)
			}
			quoteString(&s, v)
		case types.ArrayKind:
			// Snowflake allows comparison between arrays, but we currently do not support it in Meergo.
			return "", errors.New("cannot apply operators on Array type")
		case types.MapKind:
			// Snowflake allows comparison between objects, but we currently do not support it in Meergo.
			return "", errors.New("cannot apply operators on Map type")
		default:
			return "", fmt.Errorf("unexpected column with type %q", k)
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
