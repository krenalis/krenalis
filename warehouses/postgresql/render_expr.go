//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package postgresql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"
)

// renderExpr renders the expression expr returning a fragment of a query
// representing a boolean expression.
func renderExpr(b *strings.Builder, exp meergo.Expr) error {

	// Handle MultiExpr expression.
	if multiExpr, ok := exp.(*meergo.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case meergo.OpAnd:
			op = " AND "
		case meergo.OpOr:
			op = " OR "
		default:
			return fmt.Errorf("invalid operator %q", multiExpr.Operator)
		}
		for i, operand := range multiExpr.Operands {
			if i > 0 {
				b.WriteString(op)
			}
			_, isMultiExpr := operand.(*meergo.MultiExpr)
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
	baseExpr := exp.(*meergo.BaseExpr)
	c := baseExpr.Column

	// Validate the column name.
	if !meergo.IsValidIdentifier(c.Name) {
		return fmt.Errorf("invalid property name %q", c.Name)
	}

	qname := quoteIdent(c.Name)

	// Render the column identifier, the operator, and, if necessary, the values.
	switch op := baseExpr.Operator; op {
	case
		meergo.OpIs,
		meergo.OpIsNot,
		meergo.OpIsLessThan,
		meergo.OpIsLessThanOrEqualTo,
		meergo.OpIsGreaterThan,
		meergo.OpIsGreaterThanOrEqualTo,
		meergo.OpIsBefore,
		meergo.OpIsOnOrBefore,
		meergo.OpIsAfter,
		meergo.OpIsOnOrAfter:

		b.WriteString(qname)

		switch op {
		case meergo.OpIs:
			b.WriteString(" = ")
		case meergo.OpIsNot:
			b.WriteString(" <> ")
		case meergo.OpIsLessThan, meergo.OpIsBefore:
			b.WriteString(" < ")
		case meergo.OpIsLessThanOrEqualTo, meergo.OpIsOnOrBefore:
			b.WriteString(" <= ")
		case meergo.OpIsGreaterThan, meergo.OpIsAfter:
			b.WriteString(" > ")
		case meergo.OpIsGreaterThanOrEqualTo, meergo.OpIsOnOrAfter:
			b.WriteString(" >= ")
		}
		serializeValue(b, baseExpr.Values[0], c.Type)

	case meergo.OpIsBetween, meergo.OpIsNotBetween:
		b.WriteString(qname)
		if op == meergo.OpIsNotBetween {
			b.WriteString(" NOT")
		}
		b.WriteString(" BETWEEN ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(" AND ")
		serializeValue(b, baseExpr.Values[1], c.Type)

	case meergo.OpContains, meergo.OpDoesNotContain:
		switch c.Type.Kind() {
		case types.TextKind:
			b.WriteString("POSITION(")
			serializeValue(b, baseExpr.Values[0], c.Type)
			b.WriteString(" IN ")
			b.WriteString(qname)
			if op == meergo.OpContains {
				b.WriteString(") > 0")
			} else {
				b.WriteString(") = 0")
			}
		case types.ArrayKind:
			if op == meergo.OpDoesNotContain {
				b.WriteString("NOT (")
			}
			serializeValue(b, baseExpr.Values[0], c.Type.Elem())
			b.WriteString(" = ANY(")
			b.WriteString(qname)
			b.WriteByte(')')
			if op == meergo.OpDoesNotContain {
				b.WriteByte(')')
			}
		}

	case meergo.OpIsOneOf, meergo.OpIsNotOneOf:
		b.WriteString(qname)
		if op == meergo.OpIsOneOf {
			b.WriteString(" IN (")
		} else {
			b.WriteString(" NOT IN (")
		}
		for i, v := range baseExpr.Values {
			if i > 0 {
				b.WriteByte(',')
			}
			serializeValue(b, v, c.Type)
		}
		b.WriteString(")")

	case meergo.OpStartsWith, meergo.OpEndsWith:
		if op == meergo.OpStartsWith {
			b.WriteString("LEFT(")
		} else {
			b.WriteString("RIGHT(")
		}
		b.WriteString(qname)
		b.WriteString(", LENGTH(")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")) = ")
		serializeValue(b, baseExpr.Values[0], c.Type)

	case meergo.OpIsTrue:
		b.WriteString(qname)

	case meergo.OpIsFalse:
		b.WriteString("NOT ")
		b.WriteString(qname)

	case meergo.OpIsEmpty:
		k := c.Type.Kind()
		if k == types.ArrayKind {
			b.WriteString("array_length(")
			b.WriteString(qname)
			b.WriteString(", 1) IS NULL")
		} else {
			if c.Nullable {
				b.WriteByte('(')
				b.WriteString(qname)
				b.WriteString(" IS NULL OR ")
			}
			b.WriteString(qname)
			var s string
			switch k {
			case types.TextKind:
				s = " = ''"
			// See issue https://github.com/meergo/meergo/issues/1804.
			// case types.JSONKind:
			//	s = ` IN ('{}'::jsonb,'[]'::jsonb,'""'::jsonb,'null'::jsonb)`
			case types.MapKind:
				s = " = '{}'::jsonb"
			}
			b.WriteString(s)
			if c.Nullable {
				b.WriteByte(')')
			}
		}

	case meergo.OpIsNotEmpty:
		k := c.Type.Kind()
		if k == types.ArrayKind {
			b.WriteString("array_length(")
			b.WriteString(qname)
			b.WriteString(", 1) IS NOT NULL")
		} else {
			if c.Nullable {
				b.WriteByte('(')
				b.WriteString(qname)
				b.WriteString(" IS NOT NULL AND ")
			}
			b.WriteString(qname)
			var s string
			switch k {
			case types.TextKind:
				s = " <> ''"
			// See issue https://github.com/meergo/meergo/issues/1804.
			// case types.JSONKind:
			//	s = ` NOT IN ('{}'::jsonb,'[]'::jsonb,'""'::jsonb,'null'::jsonb)`
			case types.MapKind:
				s = " <> '{}'::jsonb"
			}
			b.WriteString(s)
			if c.Nullable {
				b.WriteByte(')')
			}
		}

	case meergo.OpIsNull:
		b.WriteString(qname)
		b.WriteString(" IS NULL")

	case meergo.OpIsNotNull:
		b.WriteString(qname)
		b.WriteString(" IS NOT NULL")

	default:
		panic(fmt.Sprintf("unexpected operator %q", op))
	}

	return nil
}

// serializeValue serializes v with type t into b.
// As special case, v can have type warehouse.Column.
func serializeValue(b *strings.Builder, v any, t types.Type) {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case meergo.Column:
		b.WriteString(quoteIdent(v.Name))
	case string:
		quoteString(b, v)
	case bool:
		if v {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	case int:
		b.WriteString(strconv.FormatInt(int64(v), 10))
	case uint:
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case decimal.Decimal:
		v.WriteTo(b)
	case time.Time:
		b.WriteByte('\'')
		switch t.Kind() {
		case types.DateTimeKind:
			b.WriteString(v.Format("2006-01-02 15:04:05.999999"))
		case types.DateKind:
			b.WriteString(v.Format(time.DateOnly))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999"))
		}
		b.WriteByte('\'')
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}
