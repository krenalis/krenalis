// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// renderExpr renders the expression expr returning a fragment of a query
// representing a boolean expression.
func renderExpr(b *strings.Builder, exp warehouses.Expr) error {

	// Handle MultiExpr expression.
	if multiExpr, ok := exp.(*warehouses.MultiExpr); ok {
		var op string
		switch multiExpr.Operator {
		case warehouses.OpAnd:
			op = " AND "
		case warehouses.OpOr:
			op = " OR "
		default:
			return fmt.Errorf("invalid operator %d", multiExpr.Operator)
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

	qname := quoteIdent(c.Name)
	if len(baseExpr.Keys) > 0 && c.Type.Kind() != types.JSONKind {
		return fmt.Errorf("JSON keys cannot be used with type %q", c.Type)
	}
	if c.Type.Kind() == types.JSONKind {
		if len(baseExpr.Keys) > 0 {
			var value strings.Builder
			value.WriteByte('(')
			value.WriteString(qname)
			for _, key := range baseExpr.Keys {
				value.WriteString(" -> ")
				quoteString(&value, key)
			}
			value.WriteByte(')')
			qname = value.String()
		}
		return renderJSONExpr(b, baseExpr, qname)
	}
	if (c.Type.Kind() == types.ArrayKind || c.Type.Kind() == types.MapKind) && c.Type.Elem().Kind() == types.JSONKind {
		switch baseExpr.Operator {
		case warehouses.OpIsEmpty, warehouses.OpIsNotEmpty, warehouses.OpIsNull, warehouses.OpIsNotNull,
			warehouses.OpExists, warehouses.OpDoesNotExist:
		case warehouses.OpContains, warehouses.OpDoesNotContain:
			if c.Type.Kind() == types.ArrayKind {
				return renderJSONExpr(b, baseExpr, qname)
			}
			return fmt.Errorf("operator %d is not supported for type %q", baseExpr.Operator, c.Type)
		default:
			return fmt.Errorf("operator %d is not supported for type %q", baseExpr.Operator, c.Type)
		}
	}

	// Render the column identifier, the operator, and, if necessary, the values.
	switch op := baseExpr.Operator; op {
	case
		warehouses.OpIs,
		warehouses.OpIsNot,
		warehouses.OpIsLessThan,
		warehouses.OpIsLessThanOrEqualTo,
		warehouses.OpIsGreaterThan,
		warehouses.OpIsGreaterThanOrEqualTo,
		warehouses.OpIsBefore,
		warehouses.OpIsOnOrBefore,
		warehouses.OpIsAfter,
		warehouses.OpIsOnOrAfter:

		b.WriteString(qname)

		switch op {
		case warehouses.OpIs:
			b.WriteString(" = ")
		case warehouses.OpIsNot:
			b.WriteString(" <> ")
		case warehouses.OpIsLessThan, warehouses.OpIsBefore:
			b.WriteString(" < ")
		case warehouses.OpIsLessThanOrEqualTo, warehouses.OpIsOnOrBefore:
			b.WriteString(" <= ")
		case warehouses.OpIsGreaterThan, warehouses.OpIsAfter:
			b.WriteString(" > ")
		case warehouses.OpIsGreaterThanOrEqualTo, warehouses.OpIsOnOrAfter:
			b.WriteString(" >= ")
		}
		serializeValue(b, baseExpr.Values[0], c.Type)

	case warehouses.OpIsBetween, warehouses.OpIsNotBetween:
		b.WriteString(qname)
		if op == warehouses.OpIsNotBetween {
			b.WriteString(" NOT")
		}
		b.WriteString(" BETWEEN ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(" AND ")
		serializeValue(b, baseExpr.Values[1], c.Type)

	case warehouses.OpContains, warehouses.OpDoesNotContain:
		switch c.Type.Kind() {
		case types.StringKind:
			b.WriteString("POSITION(")
			serializeValue(b, baseExpr.Values[0], c.Type)
			b.WriteString(" IN ")
			b.WriteString(qname)
			if op == warehouses.OpContains {
				b.WriteString(") > 0")
			} else {
				b.WriteString(") = 0")
			}
		case types.ArrayKind:
			if op == warehouses.OpDoesNotContain {
				b.WriteString("NOT (")
			}
			serializeValue(b, baseExpr.Values[0], c.Type.Elem())
			b.WriteString(" = ANY(")
			b.WriteString(qname)
			b.WriteByte(')')
			if op == warehouses.OpDoesNotContain {
				b.WriteByte(')')
			}
		}

	case warehouses.OpIsOneOf, warehouses.OpIsNotOneOf:
		b.WriteString(qname)
		if op == warehouses.OpIsOneOf {
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

	case warehouses.OpStartsWith, warehouses.OpEndsWith:
		if op == warehouses.OpStartsWith {
			b.WriteString("LEFT(")
		} else {
			b.WriteString("RIGHT(")
		}
		b.WriteString(qname)
		b.WriteString(", LENGTH(")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")) = ")
		serializeValue(b, baseExpr.Values[0], c.Type)

	case warehouses.OpIsTrue, warehouses.OpIsFalse:
		if c.Type.Kind() != types.BooleanKind {
			return fmt.Errorf("operator %d is not supported for type %q", op, c.Type)
		}
		if op == warehouses.OpIsFalse {
			b.WriteString("NOT ")
		}
		b.WriteString(qname)

	case warehouses.OpIsEmpty:
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
			case types.StringKind:
				s = " = ''"
			case types.MapKind:
				s = " = '{}'::jsonb"
			}
			b.WriteString(s)
			if c.Nullable {
				b.WriteByte(')')
			}
		}

	case warehouses.OpIsNotEmpty:
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
			case types.StringKind:
				s = " <> ''"
			case types.MapKind:
				s = " <> '{}'::jsonb"
			}
			b.WriteString(s)
			if c.Nullable {
				b.WriteByte(')')
			}
		}

	case warehouses.OpIsNull, warehouses.OpDoesNotExist:
		b.WriteString(qname)
		b.WriteString(" IS NULL")

	case warehouses.OpIsNotNull, warehouses.OpExists:
		b.WriteString(qname)
		b.WriteString(" IS NOT NULL")

	default:
		panic(fmt.Sprintf("unexpected operator %d", op))
	}

	return nil
}

// serializeValue serializes v with type t into b.
// As special case, v can have type warehouse.Column.
func serializeValue(b *strings.Builder, v any, t types.Type) {
	switch v := v.(type) {
	case nil:
		b.WriteString("NULL")
	case warehouses.Column:
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
	case json.Value:
		quoteString(b, string(v))
		b.WriteString("::jsonb")
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}
