// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
	"github.com/krenalis/krenalis/warehouses"
)

// jsonNumberParts returns the significant digits and decimal magnitude of a
// nonzero number within the expression operand limits.
func jsonNumberParts(number decimal.Decimal) (string, int) {
	_, scale := number.PrecisionScale()
	text := strings.TrimPrefix(fmt.Sprintf("%.*f", scale, number), "-")
	digits := strings.TrimLeft(strings.ReplaceAll(text, ".", ""), "0")
	return strings.TrimRight(digits, "0"), len(digits) - scale
}

// renderJSONComparison compares JSON strings and numbers without coercing other
// kinds, using byte order for strings and exact decimal order for numbers.
func renderJSONComparison(b *strings.Builder, value, operator string, operand warehouses.JSONConditionValue) {
	isString := "IS_VARCHAR(" + value + ")"
	isNumber := "(IS_DECIMAL(" + value + ") OR IS_DOUBLE(" + value + "))"
	text := "COLLATE(AS_VARCHAR(" + value + "), 'utf8')"
	fmt.Fprintf(b, "CASE WHEN %s THEN %s %s ", isString, text, operator)
	quoteString(b, operand.String)
	if operand.Number != nil {
		fmt.Fprintf(b, " WHEN %s THEN ", isNumber)
		renderJSONNumberComparison(b, value, operator, *operand.Number)
	}
	b.WriteString(" ELSE FALSE END")
}

// renderJSONExpr renders filters on JSON columns and containers of JSON values.
// Predicates on JSON values are two-valued, so negative operators include
// absent values, nulls, and non-matching JSON kinds.
func renderJSONExpr(b *strings.Builder, expr *warehouses.BaseExpr) error {

	c := expr.Column
	target := quoteIdent(c.Name)

	k := c.Type.Kind()
	op := expr.Operator

	if k != types.JSONKind && len(expr.Keys) > 0 {
		return fmt.Errorf("JSON keys cannot be used with type %q", c.Type)
	}

	if (k == types.ArrayKind || k == types.MapKind) && op != warehouses.OpContains && op != warehouses.OpDoesNotContain {
		switch expr.Operator {
		case warehouses.OpIsEmpty, warehouses.OpIsNotEmpty:
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
				b.WriteString(" = ")
			} else {
				b.WriteString(" <> ")
			}
			if k == types.ArrayKind {
				b.WriteString("ARRAY_CONSTRUCT()")
			} else {
				b.WriteString("OBJECT_CONSTRUCT()")
			}
			if c.Nullable {
				b.WriteByte(')')
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
		for range expr.Keys {
			path.WriteString("GET(")
		}
		path.WriteString(target)
		for _, key := range expr.Keys {
			path.WriteString(", ")
			quoteString(&path, key)
			path.WriteByte(')')
		}
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

	// Preserve structural equality for callers supplying an encoded JSON value.
	if count == 1 && op == warehouses.OpIs {
		if operand, ok := expr.Values[0].(json.Value); ok {
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
	values := make([]warehouses.JSONConditionValue, count)
	for i, operand := range expr.Values {
		var ok bool
		values[i], ok = operand.(warehouses.JSONConditionValue)
		if !ok {
			return fmt.Errorf("expected a JSONConditionValue, got %T", operand)
		}
	}

	if op == warehouses.OpIsTrue || op == warehouses.OpIsFalse {
		fmt.Fprintf(b, "AS_BOOLEAN(%s) IS NOT DISTINCT FROM ", target)
		if op == warehouses.OpIsTrue {
			b.WriteString("TRUE")
		} else {
			b.WriteString("FALSE")
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
		// Group operands by kind to avoid repeating extraction and normalization
		// for every entry of a potentially long membership list.
		isString := "IS_VARCHAR(" + target + ")"
		isNumber := "(IS_DECIMAL(" + target + ") OR IS_DOUBLE(" + target + "))"
		text := "COLLATE(AS_VARCHAR(" + target + "), 'utf8')"
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
			coefficient, magnitude := jsonNumberParts(*operand.Number)
			sign := "+"
			if operand.Number.Sign() < 0 {
				sign = "-"
			}
			numbers = append(numbers, sign+strconv.Itoa(magnitude)+":"+coefficient)
		}
		if len(numbers) > 0 {
			text := "TO_JSON(" + target + ")"
			mantissa := "SPLIT_PART(LOWER(" + text + "), 'e', 1)"
			digits := "LTRIM(REPLACE(" + mantissa + ", '.', ''), '-0')"
			magnitude := "(LENGTH(" + digits + ") - LENGTH(SPLIT_PART(" + mantissa + ", '.', 2))" +
				" + COALESCE(NULLIF(SPLIT_PART(LOWER(" + text + "), 'e', 2), '')::BIGINT, 0))"
			fmt.Fprintf(b, " WHEN %s THEN (CASE WHEN %s = '' THEN '0' ELSE CONCAT(", isNumber, digits)
			fmt.Fprintf(b, "CASE WHEN LEFT(%s, 1) = '-' THEN '-' ELSE '+' END, %s::VARCHAR, ':', "+
				"RTRIM(%s, '0')) END) COLLATE 'utf8' IN (",
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
		if k == types.MapKind {
			return fmt.Errorf("operator %d is not supported for type %q", expr.Operator, c.Type)
		}
		b.WriteByte('(')
		if expr.Column.Type.Kind() == types.JSONKind {
			fmt.Fprintf(b, "CONTAINS(AS_VARCHAR(%s), ", target)
			quoteString(b, values[0].String)
			b.WriteByte(')')
			b.WriteString(" OR ")
		}
		b.WriteString("ARRAY_SIZE(FILTER(")
		if expr.Column.Type.Kind() == types.ArrayKind {
			b.WriteString(target)
		} else {
			fmt.Fprintf(b, "AS_ARRAY(%s)", target)
		}
		b.WriteString(", krenalis_json_element -> ")
		renderJSONComparison(b, "krenalis_json_element", "=", values[0])
		b.WriteString(")) > 0")
		b.WriteByte(')')

	case warehouses.OpStartsWith, warehouses.OpEndsWith:
		if op == warehouses.OpStartsWith {
			b.WriteString("STARTSWITH(")
		} else {
			b.WriteString("ENDSWITH(")
		}
		fmt.Fprintf(b, "AS_VARCHAR(%s), ", target)
		quoteString(b, values[0].String)
		b.WriteByte(')')

	case warehouses.OpIsEmpty:
		fmt.Fprintf(b, "(%s IS NULL OR IS_NULL_VALUE(%s) OR AS_VARCHAR(%s) = '' OR "+
			"ARRAY_SIZE(AS_ARRAY(%s)) = 0 OR ARRAY_SIZE(OBJECT_KEYS(AS_OBJECT(%s))) = 0)",
			target, target, target, target, target)

	case warehouses.OpIsNull:
		fmt.Fprintf(b, "IS_NULL_VALUE(%s)", target)

	case warehouses.OpExists:
		b.WriteString(target)
		b.WriteString(" IS NOT NULL")

	}
	b.WriteString(", FALSE)")

	return nil
}

// renderJSONNumberComparison orders decimals by sign, decimal magnitude, and
// significant digits, avoiding database casts that round or overflow operands.
func renderJSONNumberComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {

	text := "TO_JSON(" + value + ")"
	mantissa := "SPLIT_PART(LOWER(" + text + "), 'e', 1)"
	digits := "LTRIM(REPLACE(" + mantissa + ", '.', ''), '-0')"
	sign := "(CASE WHEN " + digits + " = '' THEN 0 WHEN LEFT(" + text + ", 1) = '-' THEN -1 ELSE 1 END)"
	if number.Sign() == 0 {
		fmt.Fprintf(b, "%s %s 0", sign, operator)
		return
	}
	magnitude := "(LENGTH(" + digits + ") - LENGTH(SPLIT_PART(" + mantissa + ", '.', 2))" +
		" + COALESCE(NULLIF(SPLIT_PART(LOWER(" + text + "), 'e', 2), '')::BIGINT, 0))"
	significand := "COLLATE(RTRIM(" + digits + ", '0'), 'utf8')"

	coefficient, n := jsonNumberParts(number)
	s := number.Sign()

	fmt.Fprintf(b, "(CASE WHEN %s < %d THEN -1 WHEN %s > %d THEN 1", sign, s, sign, s)
	fmt.Fprintf(b, " WHEN %s < %d THEN %d WHEN %s > %d THEN %d", magnitude, n, -s, magnitude, n, s)
	fmt.Fprintf(b, " WHEN %s < '%s' THEN %d WHEN %s > '%s' THEN %d ELSE 0 END) %s 0",
		significand, coefficient, -s, significand, coefficient, s, operator)

}
