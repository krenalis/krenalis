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

	"github.com/open2b/chichi/apis/datastore/expr"
	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/apis/postgres"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// renderExpr renders the expression expr, which refers to the properties in
// schema, returning a fragment of a query representing a boolean expression.
//
// TODO(Gianluca): see the issue
// https://github.com/open2b/chichi/issues/727, where we revise the
// "where" expressions and the filters.
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
	if !warehouses.IsValidIdentifier(baseExpr.Column) {
		return "", fmt.Errorf("invalid property name %q", baseExpr.Column)
	}

	column, err := warehouses.PropertyPathToColumn(schema, baseExpr.Column)
	if err != nil {
		return "", fmt.Errorf("property %q not found", baseExpr.Column)
	}

	// Render the column identifier.
	s.WriteString(postgres.QuoteIdent(column.Name))
	s.WriteString(" ")

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

		property, err := schema.PropertyByPath(types.Path{baseExpr.Column})
		if err != nil {
			return "", fmt.Errorf("property %q not found", baseExpr.Column)
		}

		switch k := property.Type.Kind(); k {
		case types.BooleanKind:
			v, ok := baseExpr.Value.(bool)
			if !ok {
				return "", fmt.Errorf("expecting value of type bool, got %T", baseExpr.Value)
			}
			if v {
				s.WriteString("TRUE")
			} else {
				s.WriteString("FALSE")
			}
		case types.IntKind:
			v, ok := baseExpr.Value.(int)
			if !ok {
				return "", fmt.Errorf("expecting value of type int, got %T", baseExpr.Value)
			}
			s.WriteString(strconv.Itoa(v))
		case types.FloatKind:
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
			s.WriteString(v.Format("2006-01-02 15:04:05.999999"))
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
			s.WriteString(v.Format("15:04:05.999999"))
			s.WriteByte('\'')
		case types.UUIDKind, types.InetKind, types.TextKind:
			v, ok := baseExpr.Value.(string)
			if !ok {
				return "", fmt.Errorf("expecting value of type string, got %T", baseExpr.Value)
			}
			quoteString(&s, v)
		case types.JSONKind:
			return "", errors.New("cannot apply operators on JSON type")
		case types.ArrayKind:
			return "", errors.New("cannot apply operators on Array type")
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

// quoteString quotes s as a string and writes it into b.
//
// See the documentation at
// https://www.postgresql.org/docs/16/sql-syntax-lexical.html#SQL-SYNTAX-STRINGS
// (for the escaping of single quotes) and at
// https://www.postgresql.org/docs/13/datatype-character.html (for the character
// with code 0).
//
// NOTE: keep this function in sync with the one within the PostgreSQL
// connector.
func quoteString(b *strings.Builder, s string) {
	if s == "" {
		b.WriteString("''")
		return
	}
	b.WriteByte('\'')
	for {
		p := strings.IndexByte(s, '\'')
		if p == -1 {
			p = len(s)
		}
		b.WriteString(s[:p])
		if p == len(s) {
			break
		}
		b.WriteString("''")
		s = s[p+1:]
		if len(s) == 0 {
			break
		}
	}
	b.WriteByte('\'')
}
