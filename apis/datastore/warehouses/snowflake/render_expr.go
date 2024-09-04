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

	// Render the column identifier.
	b.WriteString(postgres.QuoteIdent(c.Name))
	b.WriteString(" ")

	// Render the operator and, if necessary, the value.
	switch baseExpr.Operator {
	case
		warehouses.OpEqual,
		warehouses.OpNotEqual,
		warehouses.OpGreater,
		warehouses.OpGreaterEqual,
		warehouses.OpLess,
		warehouses.OpLessEqual:

		switch baseExpr.Operator {
		case warehouses.OpEqual:
			b.WriteString("= ")
		case warehouses.OpNotEqual:
			b.WriteString("<> ")
		case warehouses.OpGreater:
			b.WriteString("> ")
		case warehouses.OpGreaterEqual:
			b.WriteString(">= ")
		case warehouses.OpLess:
			b.WriteString("< ")
		case warehouses.OpLessEqual:
			b.WriteString("<= ")
		}
		serializeValue(b, baseExpr.Value, c.Type)

	case warehouses.OpIsNull:
		b.WriteString("IS NULL")

	case warehouses.OpIsNotNull:
		b.WriteString("IS NOT NULL")

	case warehouses.OpNotIn:
		b.WriteString(" NOT ")
		fallthrough

	case warehouses.OpIn:
		b.WriteString("IN (")
		for i, v := range baseExpr.Value.([]any) {
			if i > 0 {
				b.WriteByte(',')
			}
			serializeValue(b, v, c.Type)
		}
		b.WriteString(")")
	}

	return nil
}

// serializeValue serializes v with type t into b.
// As special case, v can have type warehouse.Column.
func serializeValue(b *strings.Builder, v any, t types.Type) {

	if t.Kind() == types.JSONKind {
		b.WriteString("PARSE_JSON(")
		switch v := v.(type) {
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
				panic(err)
			}
			s.Truncate(s.Len() - 1) // remove the trailing new line.
			quoteBytes(b, s.Bytes())
		}
		b.WriteByte(')')
		return
	}

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
			b.WriteString(v.Format(time.DateTime))
		case types.TimeKind:
			b.WriteString(v.Format("15:04:05.999999999"))
		}
		b.WriteByte('\'')
	case string:
		quoteString(b, v)
	}

}
