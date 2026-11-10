// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package snowflake

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/krenalis/krenalis/tools/decimal"
)

// jsonDoubleBounds encloses an exact decimal between adjacent binary64 values.
// Prepared operands are finite and within the global decimal operand limits.
func jsonDoubleBounds(number decimal.Decimal) (float64, float64) {
	value, ok := new(big.Rat).SetString(number.String())
	if !ok {
		panic("warehouses/snowflake: invalid prepared decimal operand")
	}
	lower, _ := new(big.Float).SetPrec(53).SetMode(big.ToNegativeInf).SetRat(value).Float64()
	upper, _ := new(big.Float).SetPrec(53).SetMode(big.ToPositiveInf).SetRat(value).Float64()
	return lower, upper
}

func renderJSONDoubleComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {
	lower, upper := jsonDoubleBounds(number)
	bound := lower
	if lower != upper {
		switch operator {
		case "=":
			b.WriteString("FALSE")
			return
		case "<", "<=":
			operator = "<="
		case ">", ">=":
			operator, bound = ">=", upper
		default:
			panic(fmt.Sprintf("warehouses/snowflake: unexpected comparison operator %q", operator))
		}
	}
	fmt.Fprintf(b, "AS_DOUBLE(%s) %s TO_DOUBLE('%s')", value, operator, strconv.FormatFloat(bound, 'g', -1, 64))
}

// renderJSONFixedComparison orders fixed-point VARIANT values without casting
// them to NUMBER or DECFLOAT, which can lose an INTEGER's 39th digit.
func renderJSONFixedComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {

	text := "TO_JSON(" + value + ")"
	negative := number.Sign() < 0
	signTest := "='-'"
	if negative {
		signTest = "<>'-'"
	}
	fmt.Fprintf(b, "CASE WHEN LEFT(%s,1)%s THEN ", text, signTest)
	var differentSign bool
	switch operator {
	case "=":
		// Opposite signs cannot be equal; fixed-point zero has no sign.
	case "<", "<=":
		differentSign = !negative
	case ">", ">=":
		differentSign = negative
	default:
		panic(fmt.Sprintf("warehouses/snowflake: unexpected comparison operator %q", operator))
	}
	if differentSign {
		b.WriteString("TRUE ELSE ")
	} else {
		b.WriteString("FALSE ELSE ")
	}
	if negative {
		switch operator {
		case "=":
			// Equality does not depend on the sign.
		case "<":
			operator = ">"
		case "<=":
			operator = ">="
		case ">":
			operator = "<"
		case ">=":
			operator = "<="
		default:
			panic(fmt.Sprintf("warehouses/snowflake: unexpected comparison operator %q", operator))
		}
	}

	_, scale := number.PrecisionScale()
	literal := strings.TrimPrefix(fmt.Sprintf("%.*f", scale, number), "-")
	whole, fraction, _ := strings.Cut(literal, ".")
	literal = whole + "." + strings.TrimRight(fraction, "0")
	absolute := "LTRIM(" + text + ",'-')"
	integer := "SPLIT_PART(" + absolute + ",'.',1)"
	// With equally long integer parts, canonical digit strings order exactly.
	// Always retain the decimal point and remove only fractional trailing zeros.
	key := integer + "||'.'||RTRIM(SPLIT_PART(" + absolute + ",'.',2),'0')"
	if operator == "=" {
		fmt.Fprintf(b, "COLLATE(%s,'utf8') = '%s'", key, literal)
	} else {
		strict := string(operator[0])
		fmt.Fprintf(b, "(LENGTH(%s) %s %d OR (LENGTH(%s) = %d AND COLLATE(%s,'utf8') %s '%s'))",
			integer, strict, len(whole), integer, len(whole), key, operator, literal)
	}
	b.WriteString(" END")

}

// renderJSONNumberComparison compares the stored numeric value, preserving
// fixed-point digits and the exact binary value of DOUBLEs separately.
func renderJSONNumberComparison(b *strings.Builder, value, operator string, number decimal.Decimal) {
	fmt.Fprintf(b, "CASE WHEN TYPEOF(%s)='DOUBLE' THEN ", value)
	renderJSONDoubleComparison(b, value, operator, number)
	b.WriteString(" ELSE ")
	renderJSONFixedComparison(b, value, operator, number)
	b.WriteString(" END")
}
