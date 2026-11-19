// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

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

const decfloatPrecision = 38 // maximum significant digits in DECFLOAT

// jsonFixedKey renders a signed decimal string with a point and no trailing
// fractional zeros from the fixed-point TO_JSON representation.
func jsonFixedKey(text string) string {
	return "COLLATE(SPLIT_PART(" + text + ", '.', 1) || '.' || RTRIM(SPLIT_PART(" + text + ", '.', 2), '0'), 'utf8')"
}

// jsonFixedLiteral returns the operand in the same form as jsonFixedKey.
func jsonFixedLiteral(number decimal.Decimal) string {
	if number.Sign() == 0 {
		return "0."
	}
	_, scale := number.PrecisionScale()
	text := fmt.Sprintf("%.*f", scale, number)
	if scale == 0 {
		return text + "."
	}
	return text
}

// jsonNumberParts returns the significant digits and decimal magnitude of a
// nonzero number within the expression operand limits.
func jsonNumberParts(number decimal.Decimal) (string, int) {
	_, scale := number.PrecisionScale()
	text := strings.TrimPrefix(fmt.Sprintf("%.*f", scale, number), "-")
	digits := strings.TrimLeft(strings.ReplaceAll(text, ".", ""), "0")
	return strings.TrimRight(digits, "0"), len(digits) - scale
}

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
		var number *decimal.Decimal
		switch value := value.(type) {
		case decimal.Decimal:
			number = &value
		case warehouses.JSONConditionValue:
			number = value.Number
		}
		if number == nil {
			continue
		}
		precision, scale := number.PrecisionScale()
		if precision > types.MaxDecimalPrecision || scale > types.MaxDecimalScale {
			return fmt.Errorf("decimal operand exceeds precision %d or scale %d",
				types.MaxDecimalPrecision, types.MaxDecimalScale)
		}
	}

	jsonArrayFilter := c.Type.IsArray() && c.Type.Elem().IsJSON()
	if jsonArrayFilter && len(baseExpr.Values) == 1 &&
		(baseExpr.Operator == warehouses.OpContains || baseExpr.Operator == warehouses.OpDoesNotContain) {
		// Native array containment compares encoded JSON elements structurally.
		_, encoded := baseExpr.Values[0].(json.Value)
		jsonArrayFilter = !encoded
	}
	if c.Type.IsJSON() || jsonArrayFilter || c.Type.IsMap() && c.Type.Elem().IsJSON() {
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
		if c.Type.IsArray() {
			elem := c.Type.Elem()
			if op == warehouses.OpDoesNotContain {
				b.WriteString("NOT ")
			}
			switch elem.Kind() {
			case types.DecimalKind, types.FloatKind, types.DateTimeKind, types.DateKind, types.TimeKind:
				fmt.Fprintf(b, "(ARRAY_SIZE(FILTER(%s, krenalis_element -> ", qname)
				switch elem.Kind() {
				case types.DecimalKind:
					number, ok := baseExpr.Values[0].(decimal.Decimal)
					if !ok {
						return fmt.Errorf("warehouses/snowflake: unexpected operand type %T", baseExpr.Values[0])
					}
					_, scale := number.PrecisionScale()
					numberText := "0"
					if number.Sign() != 0 {
						numberText = fmt.Sprintf("%.*f", scale, number)
					}
					digits := strings.ReplaceAll(numberText, ".", "")
					digits = strings.Trim(strings.TrimPrefix(digits, "-"), "0")
					// INTEGER elements can exceed DECFLOAT's 38-digit precision.
					b.WriteString("IFF(TYPEOF(krenalis_element) = 'INTEGER', ")
					if scale == 0 {
						b.WriteString("TO_JSON(krenalis_element) = ")
						quoteString(b, numberText)
					} else {
						b.WriteString("FALSE")
					}
					b.WriteString(", ")
					if len(digits) > 38 {
						b.WriteString("FALSE")
					} else {
						b.WriteString("TRY_TO_DECFLOAT(TO_JSON(krenalis_element)) = TO_DECFLOAT(")
						quoteString(b, number.String())
						b.WriteByte(')')
					}
					b.WriteByte(')')
				case types.FloatKind:
					b.WriteString("TO_DOUBLE(krenalis_element) = ")
					if value, ok := baseExpr.Values[0].(float64); ok && elem.IsFloat(32) {
						// Match the element precision used by types.Marshal in Merge.
						b.WriteString(strconv.FormatFloat(value, 'G', -1, 32))
					} else {
						serializeValue(b, baseExpr.Values[0], elem)
					}
				case types.DateTimeKind:
					instant, ok := baseExpr.Values[0].(time.Time)
					if !ok {
						return fmt.Errorf("warehouses/snowflake: unexpected operand type %T", baseExpr.Values[0])
					}
					b.WriteString("CONVERT_TIMEZONE('UTC', " +
						"TO_TIMESTAMP_TZ(AS_VARCHAR(krenalis_element)))::TIMESTAMP_NTZ = ")
					serializeValue(b, instant.UTC(), elem)
				case types.DateKind:
					b.WriteString("TO_DATE(AS_VARCHAR(krenalis_element)) = ")
					serializeValue(b, baseExpr.Values[0], elem)
				case types.TimeKind:
					b.WriteString("TO_TIME(AS_VARCHAR(krenalis_element)) = ")
					serializeValue(b, baseExpr.Values[0], elem)
				default:
					return fmt.Errorf("warehouses/snowflake: unexpected array element type %s", elem)
				}
				b.WriteString(")) > 0)")
			default:
				// Match the JSON encoding used when array elements are stored.
				value, err := types.Marshal(baseExpr.Values[0], elem)
				if err != nil {
					return err
				}
				b.WriteString("ARRAY_CONTAINS(")
				serializeValue(b, value, types.JSON())
				b.WriteString(", ")
				b.WriteString(qname)
				b.WriteByte(')')
			}
			return nil
		}
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

	case warehouses.OpIsTrue, warehouses.OpIsFalse:
		if !c.Type.IsBoolean() {
			return fmt.Errorf("operator %d is not supported for type %q", op, c.Type)
		}
		if op == warehouses.OpIsFalse {
			b.WriteString("NOT ")
		}
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
		case types.StringKind:
			s = " = ''"
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
		case types.StringKind:
			s = " <> ''"
		case types.ArrayKind:
			s = " <> ARRAY_CONSTRUCT()"
		case types.MapKind:
			s = " <> OBJECT_CONSTRUCT()"
		}
		b.WriteString(s)
		if c.Nullable {
			b.WriteByte(')')
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

// renderJSONComparison compares JSON strings and numbers without coercing other
// types, using byte order for strings and exact decimal ordering for numbers.
func renderJSONComparison(b *strings.Builder, target, operator string, value warehouses.JSONConditionValue) {
	isString := "IS_VARCHAR(" + target + ")"
	text := "COLLATE(AS_VARCHAR(" + target + "), 'utf8')"
	fmt.Fprintf(b, "CASE WHEN %s THEN %s %s ", isString, text, operator)
	quoteString(b, value.String)
	if value.Number != nil {
		fmt.Fprintf(b, " WHEN IS_DECIMAL(%s) THEN ", target)
		renderJSONFixedComparison(b, target, operator, *value.Number)
		fmt.Fprintf(b, " WHEN IS_DOUBLE(%s) THEN ", target)
		renderJSONSerializedDoubleComparison(b, target, operator, *value.Number)
	}
	b.WriteString(" ELSE FALSE END")
}

// renderJSONExpr renders filters on JSON columns and containers of JSON values.
// Predicates on JSON values are two-valued, so negative operators include
// absent values, nulls, and non-matching JSON kinds.
func renderJSONExpr(b *strings.Builder, expr *warehouses.BaseExpr) error {

	c := expr.Column
	target := quoteIdent(c.Name)

	op := expr.Operator

	if !c.Type.IsJSON() && len(expr.Keys) > 0 {
		return fmt.Errorf("JSON keys cannot be used with type %q", c.Type)
	}

	if (c.Type.IsArray() || c.Type.IsMap()) && op != warehouses.OpContains && op != warehouses.OpDoesNotContain {
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
			if c.Type.IsArray() {
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

	// Preserve structural equality when callers provide an encoded JSON value.
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
		// Group operands by kind to avoid repeated extraction and conversion
		// for each entry in a potentially long membership list.
		isString := "IS_VARCHAR(" + target + ")"
		text := "COLLATE(AS_VARCHAR(" + target + "), 'utf8')"
		// Snowflake supports explicit collation in a simple CASE expression.
		fmt.Fprintf(b, "CASE WHEN %s THEN CASE %s", isString, text)
		for _, operand := range values {
			b.WriteString(" WHEN ")
			quoteString(b, operand.String)
			b.WriteString(" THEN TRUE")
		}
		b.WriteString(" ELSE FALSE END")

		var fixed strings.Builder
		var doubles []string
		for _, operand := range values {
			if operand.Number == nil {
				continue
			}
			fmt.Fprintf(&fixed, " WHEN '%s' THEN TRUE", jsonFixedLiteral(*operand.Number))
			if operand.Number.Sign() == 0 {
				doubles = append(doubles, "TO_DECFLOAT('0')")
				continue
			}
			coefficient, magnitude := jsonNumberParts(*operand.Number)
			if len(coefficient) > decfloatPrecision {
				// A serialized DOUBLE cannot equal an operand with more than 38 significant digits.
				continue
			}
			sign := ""
			if operand.Number.Sign() < 0 {
				sign = "-"
			}
			doubles = append(doubles, "TO_DECFLOAT('"+sign+coefficient+"e"+
				strconv.Itoa(magnitude-len(coefficient))+"')")
		}
		if fixed.Len() > 0 {
			fmt.Fprintf(b, " WHEN IS_DECIMAL(%s) THEN CASE %s%s ELSE FALSE END", target,
				jsonFixedKey("TO_JSON("+target+")"), fixed.String())
		}
		if len(doubles) > 0 {
			fmt.Fprintf(b, " WHEN IS_DOUBLE(%s) THEN TRY_TO_DECFLOAT(TO_JSON(%s)) IN (%s)",
				target, target, strings.Join(doubles, ", "))
		}
		b.WriteString(" ELSE FALSE END")

	case warehouses.OpContains:
		if c.Type.IsMap() {
			return fmt.Errorf("operator %d is not supported for type %q", expr.Operator, c.Type)
		}
		b.WriteByte('(')
		if expr.Column.Type.IsJSON() {
			fmt.Fprintf(b, "CONTAINS(AS_VARCHAR(%s), ", target)
			quoteString(b, values[0].String)
			b.WriteByte(')')
			b.WriteString(" OR ")
		}
		b.WriteString("ARRAY_SIZE(FILTER(")
		if expr.Column.Type.IsArray() {
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

// renderJSONFixedComparison compares the exact digits of a fixed-point VARIANT.
// INTEGER values can exceed the precision of both NUMBER and DECFLOAT.
func renderJSONFixedComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {

	text := "TO_JSON(" + value + ")"
	literal := jsonFixedLiteral(number)
	if operator == "=" {
		fmt.Fprintf(b, "%s = '%s'", jsonFixedKey(text), literal)
		return
	}

	// Opposite signs determine the result. For two negatives reverse the
	// magnitude comparison.
	less := operator[0] == '<'
	signTest := "="
	if number.Sign() < 0 {
		signTest = "<>"
		less = !less
		literal = literal[1:]
		text = "LTRIM(" + text + ", '-')"
	}
	strict := ">"
	if less {
		strict = "<"
	}
	fmt.Fprintf(b, "CASE WHEN LEFT(TO_JSON(%s), 1) %s '-' THEN %t ELSE ", value, signTest, less)

	// Compare integer digit counts first. Within one magnitude, byte order
	// on the normalized digits also orders the fractional parts exactly.
	whole, _, _ := strings.Cut(literal, ".")
	length := "LENGTH(SPLIT_PART(" + text + ", '.', 1))"
	fmt.Fprintf(b, "(%s %s %d OR (%s = %d AND %s %s '%s')) END",
		length, strict, len(whole), length, len(whole), jsonFixedKey(text), strict+operator[1:], literal)

}

// renderJSONSerializedDoubleComparison compares the JSON text of a DOUBLE
// numerically against an exact operand.
func renderJSONSerializedDoubleComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {
	if number.Sign() == 0 {
		fmt.Fprintf(b, "TRY_TO_DECFLOAT(TO_JSON(%s)) %s TO_DECFLOAT('0')", value, operator)
		return
	}
	coefficient, magnitude := jsonNumberParts(number)
	// The serialized DOUBLE fits in 38 significant digits. If an operand
	// has a nonzero remainder beyond that precision, no serialized DOUBLE lies
	// strictly between the operand and its truncation to 38 significant digits.
	// Equality with the operand is therefore impossible. Ordering comparisons
	// use the truncated bound, adjusting whether it is inclusive based on the
	// sign of the discarded remainder.
	if len(coefficient) > decfloatPrecision {
		if operator == "=" {
			b.WriteString("FALSE")
			return
		}
		switch operator {
		case "<", "<=":
			operator = "<"
			if number.Sign() > 0 {
				operator = "<="
			}
		case ">", ">=":
			operator = ">"
			if number.Sign() < 0 {
				operator = ">="
			}
		default:
			panic(fmt.Sprintf("warehouses/snowflake: unexpected comparison operator %q", operator))
		}
		coefficient = coefficient[:decfloatPrecision]
	}
	sign := ""
	if number.Sign() < 0 {
		sign = "-"
	}
	fmt.Fprintf(b, "TRY_TO_DECFLOAT(TO_JSON(%s)) %s TO_DECFLOAT('%s%se%d')",
		value, operator, sign, coefficient, magnitude-len(coefficient))
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
