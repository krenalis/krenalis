// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"
	"github.com/meergo/meergo/warehouses"
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

	qname := quoteIdent(c.Name)

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
		if op == warehouses.OpDoesNotContain {
			b.WriteString("NOT ")
		}
		b.WriteString("CONTAINS(")
		b.WriteString(qname)
		b.WriteString(", ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")")

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
			b.WriteString("STARTSWITH(")
		} else {
			b.WriteString("ENDSWITH(")
		}
		b.WriteString(qname)
		b.WriteString(", ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")")

	case warehouses.OpIsTrue:
		b.WriteString(qname)

	case warehouses.OpIsFalse:
		b.WriteString("NOT ")
		b.WriteString(qname)

	case warehouses.OpIsEmpty:
		if c.Nullable {
			b.WriteByte('(')
			b.WriteString(qname)
			b.WriteString(" IS NULL OR ")
		}
		b.WriteString(qname)
		var s string
		switch c.Type.Kind() {
		case types.TextKind:
			s = " = ''"
		// See issue https://github.com/meergo/meergo/issues/1804.
		// case types.JSONKind:
		//	s = " IN (OBJECT_CONSTRUCT(),ARRAY_CONSTRUCT(),'',PARSE_JSON('null'))"
		case types.ArrayKind:
			s = " = ARRAY_CONSTRUCT()"
		case types.MapKind:
			s = " = OBJECT_CONSTRUCT()"
		}
		b.WriteString(s)
		if c.Nullable {
			b.WriteByte(')')
		}

	case warehouses.OpIsNotEmpty:
		if c.Nullable {
			b.WriteByte('(')
			b.WriteString(qname)
			b.WriteString(" IS NOT NULL AND ")
		}
		b.WriteString(qname)
		var s string
		switch c.Type.Kind() {
		case types.TextKind:
			s = " <> ''"
		// See issue https://github.com/meergo/meergo/issues/1804.
		// case types.JSONKind:
		//	s = " NOT IN (OBJECT_CONSTRUCT(),ARRAY_CONSTRUCT(),'',PARSE_JSON('null'))"
		case types.ArrayKind:
			s = " <> ARRAY_CONSTRUCT()"
		case types.MapKind:
			s = " <> OBJECT_CONSTRUCT()"
		}
		b.WriteString(s)
		if c.Nullable {
			b.WriteByte(')')
		}

	case warehouses.OpIsNull:
		b.WriteString(qname)
		b.WriteString(" IS NULL")

	case warehouses.OpIsNotNull:
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
			b.WriteString(v.Format("2006-01-02 15:04:05.999999999"))
		case types.DateKind:
			b.WriteString(v.Format(time.DateOnly))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999999"))
		}
		b.WriteByte('\'')
	case json.Value:
		b.WriteString("PARSE_JSON(")
		quoteBytes(b, v)
		b.WriteByte(')')
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}
