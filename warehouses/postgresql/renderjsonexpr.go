// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// renderJSONComparison compares JSON strings and numbers without coercing other
// kinds, using byte order for strings and exact decimal order for numbers.
func renderJSONComparison(b *strings.Builder, value, operator string, operand warehouses.JSONConditionValue) {
	isString := "jsonb_typeof(" + value + ") = 'string'"
	isNumber := "jsonb_typeof(" + value + ") = 'number'"
	text := "(" + value + " #>> '{}') COLLATE \"C\""
	fmt.Fprintf(b, "CASE WHEN %s THEN %s %s ", isString, text, operator)
	quoteString(b, operand.String)
	if operand.Number != nil {
		fmt.Fprintf(b, " WHEN %s THEN ", isNumber)
		renderJSONNumberComparison(b, value, operator, *operand.Number)
	}
	b.WriteString(" ELSE FALSE END")
}

// renderJSONExpr renders JSON filters with two-valued predicates, so negative
// operators include absent values, nulls, and non-matching JSON kinds.
func renderJSONExpr(b *strings.Builder, expr *warehouses.BaseExpr, value string) error {

	op := expr.Operator
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

	// Preserve structural equality for callers supplying an encoded JSON value.
	if count == 1 && op == warehouses.OpIs {
		if operand, ok := expr.Values[0].(json.Value); ok {
			if negative {
				b.WriteString("NOT ")
			}
			b.WriteString("COALESCE(")
			b.WriteString(value)
			b.WriteString(" = ")
			serializeValue(b, operand, types.JSON())
			b.WriteString(", FALSE)")
			return nil
		}
	}
	values := make([]warehouses.JSONConditionValue, count)
	for i, operand := range expr.Values {
		var ok bool
		values[i], ok = operand.(warehouses.JSONConditionValue)
		if !ok {
			return fmt.Errorf("expected a JSONConditionValue, got %T", operand)
		}
	}

	if op == warehouses.OpIsTrue || op == warehouses.OpIsFalse {
		b.WriteString(value)
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
		renderJSONComparison(b, value, operator, values[0])

	case warehouses.OpIsBetween:
		b.WriteByte('(')
		renderJSONComparison(b, value, ">=", values[0])
		b.WriteString(" AND ")
		renderJSONComparison(b, value, "<=", values[1])
		b.WriteByte(')')

	case warehouses.OpIsOneOf:
		// Group operands by kind to avoid repeating extraction and normalization
		// for every entry of a potentially long membership list.
		isString := "jsonb_typeof(" + value + ") = 'string'"
		isNumber := "jsonb_typeof(" + value + ") = 'number'"
		text := "(" + value + " #>> '{}') COLLATE \"C\""
		fmt.Fprintf(b, "CASE WHEN %s THEN %s IN (", isString, text)
		for i, operand := range values {
			if i > 0 {
				b.WriteString(", ")
			}
			quoteString(b, operand.String)
		}
		b.WriteByte(')')

		var numbers []string
		for _, operand := range values {
			if operand.Number == nil {
				continue
			}
			if operand.Number.Sign() == 0 {
				numbers = append(numbers, "0")
				continue
			}
			coefficient, exponent, _ := strings.Cut(fmt.Sprintf("%e", *operand.Number), "e")
			n, err := strconv.ParseInt(exponent, 10, 64)
			if err != nil {
				return fmt.Errorf("cannot render JSON number: %s", err)
			}
			sign := "+"
			if operand.Number.Sign() < 0 {
				sign = "-"
			}
			coefficient = strings.TrimRight(strings.ReplaceAll(strings.TrimPrefix(coefficient, "-"), ".", ""), "0")
			numbers = append(numbers, sign+strconv.FormatInt(n+1, 10)+":"+coefficient)
		}
		if len(numbers) > 0 {
			text := "(" + value + " #>> '{}')"
			mantissa := "SPLIT_PART(LOWER(" + text + "), 'e', 1)"
			digits := "LTRIM(REPLACE(" + mantissa + ", '.', ''), '-0')"
			magnitude := "(LENGTH(" + digits + ") - LENGTH(SPLIT_PART(" + mantissa + ", '.', 2))" +
				" + COALESCE(NULLIF(SPLIT_PART(LOWER(" + text + "), 'e', 2), '')::BIGINT, 0))"
			fmt.Fprintf(b, " WHEN %s THEN (CASE WHEN %s = '' THEN '0' ELSE CONCAT(", isNumber, digits)
			fmt.Fprintf(b, "CASE WHEN LEFT(%s, 1) = '-' THEN '-' ELSE '+' END, %s::VARCHAR, ':', "+
				"RTRIM(%s, '0')) END) COLLATE \"C\" IN (",
				text, magnitude, digits)
			for i, number := range numbers {
				if i > 0 {
					b.WriteString(", ")
				}
				quoteString(b, number)
			}
			b.WriteByte(')')
		}
		b.WriteString(" ELSE FALSE END")

	case warehouses.OpContains:
		b.WriteByte('(')
		if expr.Column.Type.Kind() == types.JSONKind {
			fmt.Fprintf(b, "(jsonb_typeof(%s) = 'string' AND POSITION(", value)
			quoteString(b, values[0].String)
			fmt.Fprintf(b, " IN (%s #>> '{}')) > 0)", value)
			b.WriteString(" OR ")
		}
		b.WriteString("EXISTS (SELECT 1 FROM ")
		if expr.Column.Type.Kind() == types.ArrayKind {
			fmt.Fprintf(b, "unnest(%s)", value)
		} else {
			fmt.Fprintf(b, "jsonb_array_elements(CASE WHEN jsonb_typeof(%s) = 'array' THEN %s ELSE '[]'::jsonb END)",
				value, value)
		}
		b.WriteString(" AS krenalis_json_element(value) WHERE ")
		renderJSONComparison(b, "krenalis_json_element.value", "=", values[0])
		b.WriteByte(')')
		b.WriteByte(')')

	case warehouses.OpStartsWith, warehouses.OpEndsWith:
		fmt.Fprintf(b, "(jsonb_typeof(%s) = 'string' AND ", value)
		if op == warehouses.OpStartsWith {
			b.WriteString("LEFT(")
		} else {
			b.WriteString("RIGHT(")
		}
		fmt.Fprintf(b, "(%s #>> '{}'), LENGTH(", value)
		quoteString(b, values[0].String)
		b.WriteString(")) COLLATE \"C\" = ")
		quoteString(b, values[0].String)
		b.WriteByte(')')

	case warehouses.OpIsEmpty:
		fmt.Fprintf(b, "(%s IS NULL OR %s IN ('null'::jsonb, '\"\"'::jsonb, '[]'::jsonb, '{}'::jsonb))",
			value, value)

	case warehouses.OpIsNull:
		fmt.Fprintf(b, "%s = 'null'::jsonb", value)

	case warehouses.OpExists:
		b.WriteString(value)
		b.WriteString(" IS NOT NULL")

	}
	b.WriteString(", FALSE)")

	return nil
}

// renderJSONNumberComparison orders decimals by sign, decimal magnitude, and
// significant digits, avoiding database casts that round or overflow operands.
func renderJSONNumberComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {

	text := "(" + value + " #>> '{}')"
	mantissa := "SPLIT_PART(LOWER(" + text + "), 'e', 1)"
	digits := "LTRIM(REPLACE(" + mantissa + ", '.', ''), '-0')"
	sign := "(CASE WHEN " + digits + " = '' THEN 0 WHEN LEFT(" + text + ", 1) = '-' THEN -1 ELSE 1 END)"
	if number.Sign() == 0 {
		fmt.Fprintf(b, "%s %s 0", sign, operator)
		return
	}
	magnitude := "(LENGTH(" + digits + ") - LENGTH(SPLIT_PART(" + mantissa + ", '.', 2))" +
		" + COALESCE(NULLIF(SPLIT_PART(LOWER(" + text + "), 'e', 2), '')::BIGINT, 0))"
	significand := "RTRIM(" + digits + ", '0') COLLATE \"C\""

	// Scientific formatting preserves all significant digits without expanding
	// a potentially very large exponent.
	coefficient, exponent, _ := strings.Cut(fmt.Sprintf("%e", number), "e")
	n, err := strconv.ParseInt(exponent, 10, 64)
	if err != nil {
		panic(err)
	}
	coefficient = strings.TrimRight(strings.ReplaceAll(strings.TrimPrefix(coefficient, "-"), ".", ""), "0")
	s := number.Sign()

	fmt.Fprintf(b, "(CASE WHEN %s < %d THEN -1 WHEN %s > %d THEN 1", sign, s, sign, s)
	fmt.Fprintf(b, " WHEN %s < %d THEN %d WHEN %s > %d THEN %d", magnitude, n+1, -s, magnitude, n+1, s)
	fmt.Fprintf(b, " WHEN %s < '%s' THEN %d WHEN %s > '%s' THEN %d ELSE 0 END) %s 0",
		significand, coefficient, -s, significand, coefficient, s, operator)

}
