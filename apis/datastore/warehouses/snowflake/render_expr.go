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
	b.WriteString(postgres.QuoteIdent(c.Name))
	b.WriteString(" ")

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

		if c.Type.Kind() == types.JSONKind {
			b.WriteString("PARSE_JSON(")
			switch v := baseExpr.Value.(type) {
			case json.RawMessage:
				quoteBytes(b, v)
			case json.Number:
				quoteString(b, string(v))
			case bool, string, float64, map[string]any, []any:
				var s bytes.Buffer
				enc := json.NewEncoder(&s)
				enc.SetEscapeHTML(false)
				err := enc.Encode(v)
				if err != nil {
					return err
				}
				s.Truncate(s.Len() - 1) // remove the trailing new line.
				quoteBytes(b, s.Bytes())
			default:
				return fmt.Errorf("unexpected value of type %T for a JSON type", baseExpr.Value)
			}
			b.WriteByte(')')
			return nil
		}

		switch v := baseExpr.Value.(type) {
		case warehouses.Column:
			b.WriteByte('"')
			b.WriteString(v.Name)
			b.WriteByte('"')
		case bool:
			if t := c.Type; t.Kind() != types.BooleanKind {
				return unexpectedTypeErr(v, t)
			}
			if v {
				b.WriteString("TRUE")
			} else {
				b.WriteString("FALSE")
			}
		case float64:
			if t := c.Type; t.Kind() != types.FloatKind {
				return unexpectedTypeErr(v, t)
			}
			b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
		case decimal.Decimal:
			if t := c.Type; t.Kind() != types.DecimalKind {
				return unexpectedTypeErr(v, t)
			}
			b.WriteString(v.String())
		case time.Time:
			b.WriteByte('\'')
			switch t := c.Type; t.Kind() {
			default:
				return unexpectedTypeErr(v, t)
			case types.DateTimeKind:
				b.WriteString(v.Format("2006-01-02 15:04:05.999999999"))
			case types.DateKind:
				b.WriteString(v.Format(time.DateTime))
			case types.TimeKind:
				b.WriteString(v.Format("15:04:05.999999999"))
			}
			b.WriteByte('\'')
		case string:
			if c.Type.Kind() != types.TextKind {
				return unexpectedTypeErr(v, c.Type)
			}
			quoteString(b, v)
		default:
			return unexpectedTypeErr(v, c.Type)
		}

	case warehouses.OperatorIsNull:
		b.WriteString("IS NULL")

	case warehouses.OperatorIsNotNull:
		b.WriteString("IS NOT NULL")

	default:
		return fmt.Errorf("invalid operator %q", baseExpr.Operator)
	}

	return nil

}

func unexpectedTypeErr(v any, t types.Type) error {
	switch t.Kind() {
	case types.ArrayKind:
		return errors.New("cannot apply operators on Array type")
	case types.MapKind:
		return errors.New("cannot apply operators on Map type")
	}
	return fmt.Errorf("unexpected value %v (type %T) for the %s type", v, v, t)
}
