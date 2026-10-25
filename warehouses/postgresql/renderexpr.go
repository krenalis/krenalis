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

	for _, value := range baseExpr.Values {
		if number, ok := value.(decimal.Decimal); ok && !types.IsDecimal(number) {
			return fmt.Errorf("decimal operand exceeds precision %d or scale %d",
				types.MaxDecimalPrecision, types.MaxDecimalScale)
		}
	}

	if c.Type.IsJSON() || (c.Type.IsArray() || c.Type.IsMap()) && c.Type.Elem().IsJSON() {
		return renderJSONExpr(b, baseExpr)
	}

	if len(baseExpr.Keys) > 0 {
		return fmt.Errorf("JSON keys cannot be used with type %q", c.Type)
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
		if !c.Type.IsBoolean() {
			return fmt.Errorf("operator %d is not supported for type %q", op, c.Type)
		}
		if op == warehouses.OpIsFalse {
			b.WriteString("NOT ")
		}
		b.WriteString(qname)

	case warehouses.OpIsEmpty:
		if c.Type.IsArray() {
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
			switch c.Type.Kind() {
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
		if c.Type.IsArray() {
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
			switch c.Type.Kind() {
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

// renderJSONComparison compares a validated JSON String or Number operand.
// Strings use byte order and numbers use exact decimal ordering.
func renderJSONComparison(b *strings.Builder, target, operator string, value json.Value) {
	if value.IsString() {
		fmt.Fprintf(b, "CASE WHEN jsonb_typeof(%s) = 'string' THEN (%s #>> '{}') COLLATE \"C\" %s ",
			target, target, operator)
		quoteString(b, value.String())
	} else {
		number, err := value.Decimal(0, 0)
		if err != nil {
			panic(fmt.Sprintf("warehouses/postgresql: invalid prepared JSON operand: %v", err))
		}
		_, scale := number.PrecisionScale()
		fmt.Fprintf(b, "CASE WHEN jsonb_typeof(%s) = 'number' THEN (%s)::numeric %s %.*f",
			target, target, operator, scale, number)
	}
	b.WriteString(" ELSE FALSE END")
}

// renderJSONExpr renders filters on JSON columns and containers of JSON values.
// Predicates on JSON values are two-valued, so negative operators include
// absent values, nulls, and non-matching JSON kinds.
func renderJSONExpr(b *strings.Builder, expr *warehouses.BaseExpr) error {

	if expr.JSONComparison != warehouses.JSONStructuralComparison &&
		expr.JSONComparison != warehouses.JSONTypedComparison {
		return fmt.Errorf("invalid JSON comparison mode %d", expr.JSONComparison)
	}

	c := expr.Column
	target := quoteIdent(c.Name)

	op := expr.Operator

	if !c.Type.IsJSON() && len(expr.Keys) > 0 {
		return fmt.Errorf("JSON keys cannot be used with type %q", c.Type)
	}

	if (c.Type.IsArray() || c.Type.IsMap()) && op != warehouses.OpContains && op != warehouses.OpDoesNotContain {
		switch expr.Operator {
		case warehouses.OpIsEmpty, warehouses.OpIsNotEmpty:
			switch c.Type.Kind() {
			case types.ArrayKind:
				b.WriteString("array_length(")
				b.WriteString(target)
				if expr.Operator == warehouses.OpIsEmpty {
					b.WriteString(", 1) IS NULL")
				} else {
					b.WriteString(", 1) IS NOT NULL")
				}
			case types.MapKind:
				if c.Nullable {
					b.WriteByte('(')
					b.WriteString(target)
					if expr.Operator == warehouses.OpIsEmpty {
						b.WriteString(" IS NULL OR ")
					} else {
						b.WriteString(" IS NOT NULL AND ")
					}
				}
				b.WriteString(target)
				if expr.Operator == warehouses.OpIsEmpty {
					b.WriteString(" = '{}'::jsonb")
				} else {
					b.WriteString(" <> '{}'::jsonb")
				}
				if c.Nullable {
					b.WriteByte(')')
				}
			}
			return nil
		case warehouses.OpIsNull, warehouses.OpDoesNotExist:
			b.WriteString(target)
			b.WriteString(" IS NULL")
			return nil
		case warehouses.OpIsNotNull, warehouses.OpExists:
			b.WriteString(target)
			b.WriteString(" IS NOT NULL")
			return nil
		}
		return fmt.Errorf("operator %d is not supported for type %q", expr.Operator, c.Type)
	}

	if len(expr.Keys) > 0 {
		var path strings.Builder
		path.WriteByte('(')
		path.WriteString(target)
		for _, key := range expr.Keys {
			path.WriteString(" -> ")
			quoteString(&path, key)
		}
		path.WriteByte(')')
		target = path.String()
	}

	negative := false
	switch op {
	case warehouses.OpIsNot:
		op, negative = warehouses.OpIs, true
	case warehouses.OpIsNotBetween:
		op, negative = warehouses.OpIsBetween, true
	case warehouses.OpDoesNotContain:
		op, negative = warehouses.OpContains, true
	case warehouses.OpIsNotOneOf:
		op, negative = warehouses.OpIsOneOf, true
	case warehouses.OpIsNotEmpty:
		op, negative = warehouses.OpIsEmpty, true
	case warehouses.OpIsNotNull:
		op, negative = warehouses.OpIsNull, true
	case warehouses.OpDoesNotExist:
		op, negative = warehouses.OpExists, true
	}

	var count int
	switch op {
	case warehouses.OpIs, warehouses.OpIsLessThan, warehouses.OpIsLessThanOrEqualTo,
		warehouses.OpIsGreaterThan, warehouses.OpIsGreaterThanOrEqualTo,
		warehouses.OpContains, warehouses.OpStartsWith, warehouses.OpEndsWith:
		count = 1
	case warehouses.OpIsBetween:
		count = 2
	case warehouses.OpIsOneOf:
		count = len(expr.Values)
		if count == 0 {
			return fmt.Errorf("JSON membership requires at least one value")
		}
	case warehouses.OpIsTrue, warehouses.OpIsFalse, warehouses.OpIsEmpty, warehouses.OpIsNull, warehouses.OpExists:
	default:
		return fmt.Errorf("operator %d is not supported for type %q", expr.Operator, expr.Column.Type)
	}
	if len(expr.Values) != count {
		return fmt.Errorf("operator %d requires %d values, got %d", expr.Operator, count, len(expr.Values))
	}

	// Preserve structural equality when callers provide an encoded JSON value.
	if count == 1 && op == warehouses.OpIs && expr.JSONComparison == warehouses.JSONStructuralComparison {
		if operand, ok := expr.Values[0].(json.Value); ok {
			if len(operand) == 0 || !json.Valid(operand) {
				return fmt.Errorf("structural JSON operand must contain valid JSON")
			}
			if negative {
				b.WriteString("NOT ")
			}
			b.WriteString("COALESCE(")
			b.WriteString(target)
			b.WriteString(" = ")
			serializeValue(b, operand, types.JSON())
			b.WriteString(", FALSE)")
			return nil
		}
	}
	if count > 0 && expr.JSONComparison != warehouses.JSONTypedComparison {
		return fmt.Errorf("JSON filter operands require typed comparison mode")
	}
	values := make([]json.Value, count)
	for i, operand := range expr.Values {
		value, ok := operand.(json.Value)
		if !ok || len(value) == 0 || !json.Valid(value) {
			return fmt.Errorf("JSON filter operand must contain valid encoded JSON")
		}
		value = json.TrimSpace(value)
		switch value.Kind() {
		case json.String:
			// String operands are valid for every supported comparison.
		case json.Number:
			if op == warehouses.OpStartsWith || op == warehouses.OpEndsWith {
				return fmt.Errorf("JSON prefix and suffix operands must be strings")
			}
			number, err := value.Decimal(0, 0)
			if err != nil {
				return fmt.Errorf("decimal operand exceeds precision 76 or scale 37")
			}
			if !types.IsDecimal(number) {
				return fmt.Errorf("decimal operand exceeds precision 76 or scale 37")
			}
		case json.True, json.False, json.Null:
			if op != warehouses.OpIsOneOf {
				return fmt.Errorf("JSON boolean and null operands require membership")
			}
		default:
			return fmt.Errorf("JSON filter operand has an incompatible kind")
		}
		values[i] = value
	}
	if op == warehouses.OpIsBetween && values[0].Kind() != values[1].Kind() {
		return fmt.Errorf("JSON interval bounds must have the same kind")
	}

	if op == warehouses.OpIsTrue || op == warehouses.OpIsFalse {
		b.WriteString(target)
		b.WriteString(" IS NOT DISTINCT FROM ")
		if op == warehouses.OpIsTrue {
			b.WriteString("'true'::jsonb")
		} else {
			b.WriteString("'false'::jsonb")
		}
		return nil
	}

	if negative {
		b.WriteString("NOT ")
	}
	b.WriteString("COALESCE(")
	switch op {

	case warehouses.OpIs, warehouses.OpIsLessThan, warehouses.OpIsLessThanOrEqualTo,
		warehouses.OpIsGreaterThan, warehouses.OpIsGreaterThanOrEqualTo:
		operator := map[warehouses.Operator]string{
			warehouses.OpIs: "=", warehouses.OpIsLessThan: "<", warehouses.OpIsLessThanOrEqualTo: "<=",
			warehouses.OpIsGreaterThan: ">", warehouses.OpIsGreaterThanOrEqualTo: ">=",
		}[op]
		renderJSONComparison(b, target, operator, values[0])

	case warehouses.OpIsBetween:
		b.WriteByte('(')
		renderJSONComparison(b, target, ">=", values[0])
		b.WriteString(" AND ")
		renderJSONComparison(b, target, "<=", values[1])
		b.WriteByte(')')

	case warehouses.OpIsOneOf:
		// Partition by kind without coercing a value into another domain.
		var texts, numbers, booleans []string
		hasNull := false
		for _, operand := range values {
			switch operand.Kind() {
			case json.String:
				var literal strings.Builder
				quoteString(&literal, operand.String())
				texts = append(texts, literal.String())
			case json.Number:
				number, _ := operand.Decimal(0, 0) // Validated above.
				_, scale := number.PrecisionScale()
				numbers = append(numbers, fmt.Sprintf("%.*f", scale, number))
			case json.True, json.False:
				booleans = append(booleans, "'"+string(operand)+"'::jsonb")
			case json.Null:
				hasNull = true
			default:
				return fmt.Errorf("warehouses/postgresql: unexpected JSON operand kind %v", operand.Kind())
			}
		}
		b.WriteString("CASE")
		if len(texts) > 0 {
			fmt.Fprintf(b, " WHEN jsonb_typeof(%s) = 'string' THEN (%s #>> '{}') COLLATE \"C\" IN (%s)",
				target, target, strings.Join(texts, ", "))
		}
		if len(numbers) > 0 {
			fmt.Fprintf(b, " WHEN jsonb_typeof(%s) = 'number' THEN (%s)::numeric IN (%s)",
				target, target, strings.Join(numbers, ", "))
		}
		if len(booleans) > 0 {
			fmt.Fprintf(b, " WHEN jsonb_typeof(%s) = 'boolean' THEN %s IN (%s)",
				target, target, strings.Join(booleans, ", "))
		}
		if hasNull {
			fmt.Fprintf(b, " WHEN jsonb_typeof(%s) = 'null' THEN TRUE", target)
		}
		b.WriteString(" ELSE FALSE END")

	case warehouses.OpContains:
		if c.Type.IsMap() {
			return fmt.Errorf("operator %d is not supported for type %q", expr.Operator, c.Type)
		}
		b.WriteByte('(')
		if expr.Column.Type.IsJSON() && values[0].IsString() {
			fmt.Fprintf(b, "(jsonb_typeof(%s) = 'string' AND POSITION(", target)
			quoteString(b, values[0].String())
			fmt.Fprintf(b, " IN (%s #>> '{}')) > 0)", target)
			b.WriteString(" OR ")
		}
		b.WriteString("EXISTS (SELECT 1 FROM ")
		if expr.Column.Type.IsArray() {
			fmt.Fprintf(b, "unnest(%s)", target)
		} else {
			fmt.Fprintf(b, "jsonb_array_elements(CASE WHEN jsonb_typeof(%s) = 'array' THEN %s ELSE '[]'::jsonb END)",
				target, target)
		}
		b.WriteString(" AS krenalis_json_element(value) WHERE ")
		renderJSONComparison(b, "krenalis_json_element.value", "=", values[0])
		b.WriteByte(')')
		b.WriteByte(')')

	case warehouses.OpStartsWith, warehouses.OpEndsWith:
		fmt.Fprintf(b, "(jsonb_typeof(%s) = 'string' AND ", target)
		if op == warehouses.OpStartsWith {
			b.WriteString("LEFT(")
		} else {
			b.WriteString("RIGHT(")
		}
		fmt.Fprintf(b, "(%s #>> '{}'), LENGTH(", target)
		quoteString(b, values[0].String())
		b.WriteString(")) COLLATE \"C\" = ")
		quoteString(b, values[0].String())
		b.WriteByte(')')

	case warehouses.OpIsEmpty:
		fmt.Fprintf(b, "(%s IS NULL OR %s IN ('null'::jsonb, '\"\"'::jsonb, '[]'::jsonb, '{}'::jsonb))",
			target, target)

	case warehouses.OpIsNull:
		fmt.Fprintf(b, "%s = 'null'::jsonb", target)

	case warehouses.OpExists:
		b.WriteString(target)
		b.WriteString(" IS NOT NULL")

	}
	b.WriteString(", FALSE)")

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
