//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/apis/postgres"
	"github.com/meergo/meergo/json"
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

		b.WriteString(postgres.QuoteIdent(c.Name))

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
		b.WriteString(postgres.QuoteIdent(c.Name))
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
		b.WriteString(postgres.QuoteIdent(c.Name))
		b.WriteString(", ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")")

	case warehouses.OpIsOneOf, warehouses.OpIsNotOneOf:
		b.WriteString(postgres.QuoteIdent(c.Name))
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
		b.WriteString(postgres.QuoteIdent(c.Name))
		b.WriteString(", ")
		serializeValue(b, baseExpr.Values[0], c.Type)
		b.WriteString(")")

	case warehouses.OpIsTrue:
		b.WriteString(postgres.QuoteIdent(c.Name))

	case warehouses.OpIsFalse:
		b.WriteString("NOT ")
		b.WriteString(postgres.QuoteIdent(c.Name))

	case warehouses.OpIsNull:
		b.WriteString(postgres.QuoteIdent(c.Name))
		b.WriteString(" IS NULL")

	case warehouses.OpIsNotNull:
		b.WriteString(postgres.QuoteIdent(c.Name))
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
		b.WriteByte('"')
		b.WriteString(v.Name)
		b.WriteByte('"')
	case bool:
		if v {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
		}
	case float64:
		b.WriteString(strconv.FormatFloat(v, 'G', -1, 64))
	case decimal.Decimal:
		b.WriteString(v.String())
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
	case string:
		quoteString(b, v)
	default:
		panic(fmt.Sprintf("unexpected type %T", v))
	}
}
