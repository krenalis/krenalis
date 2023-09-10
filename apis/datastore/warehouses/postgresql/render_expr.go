//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"chichi/apis/datastore/warehouses"
	"chichi/apis/postgres"
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
	s.WriteString(postgres.QuoteIdent(baseExpr.Column.Name))
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

		switch pt := baseExpr.Column.Type; pt {
		case
			types.PtBoolean:
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
			types.PtInt,
			types.PtInt16,
			types.PtInt64:
			v, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			s.WriteString(strconv.Itoa(v))
		case
			types.PtFloat,
			types.PtFloat32:
			v, ok := baseExpr.Value.(float64)
			if !ok {
				return "", fmt.Errorf("expecting value of type float64, got %T", baseExpr.Value)
			}
			s.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
		case types.PtDecimal:
			d, ok := baseExpr.Value.(decimal.Dec)
			if !ok {
				return "", fmt.Errorf("expecting value of type decimal.Dec, got %T", baseExpr.Value)
			}
			s.WriteString(d.String())
		case types.PtDateTime:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format("2006-01-02 15:04:05.999999"))
			s.WriteByte('\'')
		case types.PtDate:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format(time.DateTime))
			s.WriteByte('\'')
		case types.PtTime:
			v, ok := baseExpr.Value.(time.Time)
			if !ok {
				return "", fmt.Errorf("expecting value of type time.Time, got %T", baseExpr.Value)
			}
			s.WriteByte('\'')
			s.WriteString(v.Format("15:04:05.999999"))
			s.WriteByte('\'')
		case types.PtUUID, types.PtInet, types.PtText:
			v, ok := baseExpr.Value.(string)
			if !ok {
				return "", fmt.Errorf("expecting value of type string, got %T", baseExpr.Value)
			}
			quoteString(&s, v)
		case types.PtJSON:
			return "", errors.New("cannot apply operators on JSON type")
		case types.PtArray:
			return "", errors.New("cannot apply operators on Array type")
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

// quoteString quotes s as a string and writes it into b.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexAny(s, "\x00'")
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		if s[p] == '\'' {
			b.WriteByte('\'')
		}
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}
