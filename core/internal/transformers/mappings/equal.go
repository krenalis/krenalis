// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package mappings

import (
	"math"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
	"github.com/krenalis/krenalis/tools/types"
)

// equalDecimalText compares valid decimal number representations exactly.
// Exponents are kept separate from significant digits: even JSON exponents
// outside decimal.Decimal's range require space proportional only to the text.
func equalDecimalText(s0, s1 string) bool {

	var negative [2]bool
	var digits [2]string
	var exponent [2]big.Int
	for i, s := range [2]string{s0, s1} {

		if s[0] == '-' || s[0] == '+' {
			negative[i] = s[0] == '-'
			s = s[1:]
		}
		var e string
		if p := strings.IndexAny(s, "eE"); p != -1 {
			s, e = s[:p], s[p+1:]
		}

		// Normalize the coefficient and account for fractional and trailing digits.
		shift := 0
		if p := strings.IndexByte(s, '.'); p != -1 {
			shift = p + 1 - len(s)
			s = s[:p] + s[p+1:]
		}
		s = strings.TrimLeft(s, "0")
		digits[i] = strings.TrimRight(s, "0")
		if digits[i] == "" {
			negative[i] = false
			continue
		}
		shift += len(s) - len(digits[i])
		if e != "" {
			exponent[i].SetString(e, 10)
		}
		exponent[i].Add(&exponent[i], big.NewInt(int64(shift)))

	}

	return negative[0] == negative[1] && digits[0] == digits[1] && exponent[0].Cmp(&exponent[1]) == 0
}

// equalNumbers compares int, uint, float64, decimal.Decimal, and JSON numbers
// without rounding. A float equals an integer or decimal only if it represents
// that value exactly. All NaNs compare equal, as in PostgreSQL and Snowflake.
func equalNumbers(v0, v1 any) bool {

	// JSON numbers retain their text until comparison, including large exponents.
	if _, ok := v0.(json.Value); ok {
		v0, v1 = v1, v0
	}
	if j, ok := v1.(json.Value); ok {

		var s string
		switch n := v0.(type) {
		case int:
			s = strconv.FormatInt(int64(n), 10)
		case uint:
			s = strconv.FormatUint(uint64(n), 10)
		case decimal.Decimal:
			s = n.String()
		case float64:
			var f big.Rat
			if f.SetFloat64(n) == nil {
				return false
			}
			// This expansion is bounded by the precision of a binary float.
			s = f.FloatString(f.Denom().BitLen() - 1)
		case json.Value:
			s = string(json.TrimSpace(n))
		default:
			panic("unexpected numeric value type")
		}

		return equalDecimalText(s, string(json.TrimSpace(j)))

	}

	// Put a lone float second; two floats can be compared directly.
	if _, ok := v0.(float64); ok {
		v0, v1 = v1, v0
	}
	var n0 decimal.Decimal
	switch v0 := v0.(type) {
	case int:
		n0 = decimal.MustInt(v0)
	case uint:
		n0 = decimal.MustUint(v0)
	case decimal.Decimal:
		n0 = v0
	case float64:
		n1 := v1.(float64)
		return v0 == n1 || math.IsNaN(v0) && math.IsNaN(n1)
	default:
		panic("unexpected numeric value type")
	}

	switch v1 := v1.(type) {
	case int:
		return n0.Equal(decimal.MustInt(v1))
	case uint:
		return n0.Equal(decimal.MustUint(v1))
	case decimal.Decimal:
		return n0.Equal(v1)
	case float64:
		var f big.Rat
		if f.SetFloat64(v1) == nil {
			return false
		}
		// A binary float has denominator 2^scale, so this many decimal places
		// preserve its exact value, including for subnormal floats.
		scale := f.Denom().BitLen() - 1
		return n0.Equal(decimal.MustParse(f.FloatString(scale)))
	}

	panic("unexpected numeric value type")

}

// equalValues compares containers recursively and numbers exactly before any
// conversion can round their values or project away object properties. JSON
// containers follow the same rules as native containers. Native strings are
// parsed in the other non-JSON scalar's type. Other scalar coercions must agree
// in both directions, so a lossy conversion alone cannot establish equality.
// Values must conform to their types.
func equalValues(value0, value1 any, typ0, typ1 types.Type) bool {

	// Decode one level of JSON, preserving numbers and nested JSON as encoded values.
	values := [2]any{value0, value1}
	valueTypes := [2]types.Type{typ0, typ1}
	for i, value := range values {

		j, ok := value.(json.Value)
		if !ok {
			continue
		}

		switch j.Kind() {
		case json.Null:
			values[i] = nil
		case json.True, json.False:
			values[i], valueTypes[i] = j.Bool(), types.Boolean()
		case json.String:
			values[i], valueTypes[i] = j.String(), types.String()
		case json.Array:
			items := []any{}
			for _, item := range j.Elements() {
				items = append(items, item)
			}
			values[i], valueTypes[i] = items, types.Array(types.JSON())
		case json.Object:
			properties := map[string]any{}
			for name, value := range j.Properties() {
				properties[name] = value
			}
			values[i], valueTypes[i] = properties, types.Map(types.JSON())
		}

	}
	v0, v1 := values[0], values[1]
	t0, t1 := valueTypes[0], valueTypes[1]
	if v0 == nil || v1 == nil {
		return v0 == nil && v1 == nil
	}

	// Containers are compared structurally, without converting either container.
	k0, k1 := t0.Kind(), t1.Kind()
	if k0 == types.ArrayKind || k1 == types.ArrayKind {

		if k0 != types.ArrayKind || k1 != types.ArrayKind {
			return false
		}

		a0, a1 := v0.([]any), v1.([]any)
		if len(a0) != len(a1) {
			return false
		}
		for i, value := range a0 {
			if !equalValues(value, a1[i], t0.Elem(), t1.Elem()) {
				return false
			}
		}
		return true

	}

	object0 := k0 == types.ObjectKind || k0 == types.MapKind
	object1 := k1 == types.ObjectKind || k1 == types.MapKind
	if object0 || object1 {

		if !object0 || !object1 {
			return false
		}

		o0, o1 := v0.(map[string]any), v1.(map[string]any)
		if len(o0) != len(o1) {
			return false
		}
		for name, value0 := range o0 {
			value1, ok := o1[name]
			if !ok {
				return false
			}
			var et0, et1 types.Type
			if k0 == types.ObjectKind {
				p, _ := t0.Properties().ByName(name)
				et0 = p.Type
			} else {
				et0 = t0.Elem()
			}
			if k1 == types.ObjectKind {
				p, _ := t1.Properties().ByName(name)
				et1 = p.Type
			} else {
				et1 = t1.Elem()
			}
			if !equalValues(value0, value1, et0, et1) {
				return false
			}
		}
		return true

	}

	// Any remaining JSON values are numbers; their original precision is intact.
	numeric0 := k0 == types.IntKind || k0 == types.FloatKind || k0 == types.DecimalKind || k0 == types.JSONKind
	numeric1 := k1 == types.IntKind || k1 == types.FloatKind || k1 == types.DecimalKind || k1 == types.JSONKind
	if numeric0 && numeric1 {
		return equalNumbers(v0, v1)
	}
	if k0 != k1 && (typ0.Kind() == types.JSONKind || typ1.Kind() == types.JSONKind) {
		// Scalar coercion must still distinguish JSON strings from numbers and booleans.
		v0, v1, t0, t1 = value0, value1, typ0, typ1
	}
	if !types.Equal(t0, t1) {

		// Parse a native string in the other type, independently of argument order.
		if t1.Kind() == types.StringKind && t0.Kind() != types.StringKind {
			v0, v1, t0, t1 = v1, v0, t1, t0
		}
		converted, err := convert(v0, t0, t1, true, false, nil, None)
		if err != nil {
			return false
		}
		if !equalValues(converted, v1, t1, t1) {
			return false
		}
		if t0.Kind() == types.StringKind && t1.Kind() != types.JSONKind {
			return true
		}

		// Check the reverse conversion too: truncating a datetime or converting
		// any nonzero int8 to true must not make different values equal.
		converted, err = convert(v1, t1, t0, true, false, nil, None)
		if err != nil {
			return false
		}

		return equalValues(v0, converted, t0, t0)
	}

	// Compare instants, calendar dates, and clock times independently of time.Time metadata.
	switch t0.Kind() {
	case types.DateTimeKind:
		return v0.(time.Time).UTC().Equal(v1.(time.Time).UTC())
	case types.DateKind:
		y0, m0, d0 := v0.(time.Time).Date()
		y1, m1, d1 := v1.(time.Time).Date()
		return y0 == y1 && m0 == m1 && d0 == d1
	case types.TimeKind:
		a, b := v0.(time.Time), v1.(time.Time)
		h0, m0, s0 := a.Clock()
		h1, m1, s1 := b.Clock()
		return h0 == h1 && m0 == m1 && s0 == s1 && a.Nanosecond() == b.Nanosecond()
	}

	return reflect.DeepEqual(v0, v1)
}
