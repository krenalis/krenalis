// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package filters

import (
	"bytes"
	"strings"
	"time"

	"github.com/krenalis/krenalis/core/internal/state"
	"github.com/krenalis/krenalis/tools/decimal"
	"github.com/krenalis/krenalis/tools/json"
)

// Applies reports whether where matches the given attributes.
func Applies(where *state.Where, attributes map[string]any) bool {
	if where == nil {
		return true
	}
	for _, rule := range where.Rules {
		var applies bool
		switch rule := rule.(type) {
		case *state.Where:
			applies = Applies(rule, attributes)
		case *state.WhereCondition:
			applies = appliesCondition(rule, attributes)
		}
		if applies && where.Operator == state.OpOr {
			return true
		}
		if !applies && where.Operator == state.OpAnd {
			return false
		}
	}
	if where.Operator == state.OpOr {
		return false // none of the conditions applied.
	}
	// All the conditions applied.
	return true
}

// appliesCondition reports whether condition matches the given attributes.
func appliesCondition(condition *state.WhereCondition, attributes map[string]any) bool {

	op := condition.Operator
	v, exists := readAttributeFrom(attributes, condition.Property)
	switch op {
	case state.OpIs:
		return opIs(v, condition.Values)
	case state.OpIsNot:
		return !opIs(v, condition.Values)
	case state.OpIsLessThan:
		return opIsLessThan(v, condition.Values)
	case state.OpIsLessThanOrEqualTo:
		return opIsLessThanOrEqualTo(v, condition.Values)
	case state.OpIsGreaterThan:
		return opIsGreaterThan(v, condition.Values)
	case state.OpIsGreaterThanOrEqualTo:
		return opIsGreaterThanOrEqualTo(v, condition.Values)
	case state.OpIsBetween:
		return opIsBetween(v, condition.Values)
	case state.OpIsNotBetween:
		return !opIsBetween(v, condition.Values)
	case state.OpContains:
		return opContains(v, condition.Values)
	case state.OpDoesNotContain:
		return !opContains(v, condition.Values)
	case state.OpIsOneOf:
		return exists && opIsIn(v, condition.Values)
	case state.OpIsNotOneOf:
		return !(exists && opIsIn(v, condition.Values))
	case state.OpStartsWith:
		return opStartsWith(v, condition.Values)
	case state.OpEndsWith:
		return opEndsWith(v, condition.Values)
	case state.OpIsBefore:
		return opIsBefore(v, condition.Values)
	case state.OpIsOnOrBefore:
		return opIsOnOrBefore(v, condition.Values)
	case state.OpIsAfter:
		return opIsAfter(v, condition.Values)
	case state.OpIsOnOrAfter:
		return opIsOnOrAfter(v, condition.Values)
	case state.OpIsTrue:
		return opIsTrue(v)
	case state.OpIsFalse:
		return opIsFalse(v)
	case state.OpIsEmpty:
		return opIsEmpty(v)
	case state.OpIsNotEmpty:
		return !opIsEmpty(v)
	case state.OpIsNull:
		return exists && opIsNull(v)
	case state.OpIsNotNull:
		return !(exists && opIsNull(v))
	case state.OpExists:
		return exists
	case state.OpDoesNotExist:
		return !exists
	}

	return false
}

// compareJSON compares JSON strings or numbers of the same kind.
// Other kinds and unreadable numbers are not ordered.
func compareJSON(value, operand json.Value) (int, bool) {
	if value.Kind() != operand.Kind() {
		return 0, false
	}
	switch value.Kind() {
	case json.String:
		return strings.Compare(value.String(), operand.String()), true
	case json.Number:
		number, err := value.Decimal(0, 0)
		if err != nil {
			return 0, false
		}
		bound, err := operand.Decimal(0, 0)
		if err != nil {
			return 0, false
		}
		return number.Cmp(bound), true
	default:
		return 0, false
	}
}

func opIs(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case decimal.Decimal:
		return v.Equal(v0.(decimal.Decimal))
	case time.Time:
		return v.Equal(v0.(time.Time))
	case json.Value:
		comparison, ok := compareJSON(v, v0.(json.Value))
		return ok && comparison == 0
	default:
		return v == v0
	}
}

func opIsLessThan(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return v < v0.(string)
	case decimal.Decimal:
		return v.Less(v0.(decimal.Decimal))
	case int:
		return v < v0.(int)
	case uint:
		return v < v0.(uint)
	case float64:
		return v < v0.(float64)
	case json.Value:
		comparison, ok := compareJSON(v, v0.(json.Value))
		return ok && comparison < 0
	default:
		return false
	}
}

func opIsLessThanOrEqualTo(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return v <= v0.(string)
	case decimal.Decimal:
		return v.LessEqual(v0.(decimal.Decimal))
	case int:
		return v <= v0.(int)
	case uint:
		return v <= v0.(uint)
	case float64:
		return v <= v0.(float64)
	case json.Value:
		comparison, ok := compareJSON(v, v0.(json.Value))
		return ok && comparison <= 0
	default:
		return false
	}
}

func opIsGreaterThan(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return v > v0.(string)
	case decimal.Decimal:
		return v.Greater(v0.(decimal.Decimal))
	case int:
		return v > v0.(int)
	case uint:
		return v > v0.(uint)
	case float64:
		return v > v0.(float64)
	case json.Value:
		comparison, ok := compareJSON(v, v0.(json.Value))
		return ok && comparison > 0
	default:
		return false
	}
}

func opIsGreaterThanOrEqualTo(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return v >= v0.(string)
	case decimal.Decimal:
		return v.GreaterEqual(v0.(decimal.Decimal))
	case int:
		return v >= v0.(int)
	case uint:
		return v >= v0.(uint)
	case float64:
		return v >= v0.(float64)
	case json.Value:
		comparison, ok := compareJSON(v, v0.(json.Value))
		return ok && comparison >= 0
	default:
		return false
	}
}

func opIsBetween(v any, values []any) bool {
	v0 := values[0]
	v1 := values[1]
	switch v := v.(type) {
	case string:
		return v0.(string) <= v && v <= v1.(string)
	case decimal.Decimal:
		return v.GreaterEqual(v0.(decimal.Decimal)) && v.LessEqual(v1.(decimal.Decimal))
	case time.Time:
		v0 := v0.(time.Time)
		v1 := v1.(time.Time)
		return v.After(v0) && v.Before(v1) || v.Equal(v0) || v.Equal(v1)
	case int:
		return v0.(int) <= v && v <= v1.(int)
	case uint:
		return v0.(uint) <= v && v <= v1.(uint)
	case float64:
		return v0.(float64) <= v && v <= v1.(float64)
	case json.Value:
		lower, lowerOK := compareJSON(v, v0.(json.Value))
		upper, upperOK := compareJSON(v, v1.(json.Value))
		return lowerOK && upperOK && lower >= 0 && upper <= 0
	default:
		return false
	}
}

func opContains(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return strings.Contains(v, v0.(string))
	case json.Value:
		v0 := v0.(json.Value)
		switch v.Kind() {
		case json.String:
			return v0.IsString() && strings.Contains(v.String(), v0.String())
		case json.Array:
			for _, ev := range v.Elements() {
				if opIs(ev, values) {
					return true
				}
			}
		default:
			// Other JSON kinds cannot contain a scalar operand.
		}
	case []any:
		for _, ev := range v {
			if opIs(ev, values) {
				return true
			}
		}
	default:
		// Other record types cannot contain a scalar operand.
	}
	return false
}

func opIsIn(v any, values []any) bool {
	switch v := v.(type) {
	case string:
		for _, vi := range values {
			if v == vi.(string) {
				return true
			}
		}
	case decimal.Decimal:
		for _, vi := range values {
			if v.Equal(vi.(decimal.Decimal)) {
				return true
			}
		}
	case time.Time:
		for _, vi := range values {
			if v.Equal(vi.(time.Time)) {
				return true
			}
		}
	case int:
		for _, vi := range values {
			if v == vi.(int) {
				return true
			}
		}
	case uint:
		for _, vi := range values {
			if v == vi.(uint) {
				return true
			}
		}
	case float64:
		for _, vi := range values {
			if v == vi.(float64) {
				return true
			}
		}
	case json.Value:
		switch kind := v.Kind(); kind {
		case json.String:
			var value []byte
			var decoded bool
			var buffer []byte
			for _, operand := range values {
				operand := operand.(json.Value)
				if !operand.IsString() {
					continue
				}
				if !decoded {
					value = json.TrimSpace(v)
					if bytes.IndexByte(value, '\\') < 0 {
						value = value[1 : len(value)-1]
					} else {
						value = v.Bytes()
					}
					decoded = true
				}
				operand = json.TrimSpace(operand)
				// Unescaped JSON strings already contain their decoded UTF-8 bytes.
				if bytes.IndexByte(operand, '\\') < 0 {
					if bytes.Equal(value, operand[1:len(operand)-1]) {
						return true
					}
					continue
				}
				buffer = operand.AppendUnquote(buffer[:0])
				if bytes.Equal(value, buffer) {
					return true
				}
			}
		case json.Number:
			var number decimal.Decimal
			var decoded bool
			for _, operand := range values {
				operand := operand.(json.Value)
				if !operand.IsNumber() {
					continue
				}
				if !decoded {
					var err error
					number, err = v.Decimal(0, 0)
					if err != nil {
						return false
					}
					decoded = true
				}
				bound, err := operand.Decimal(0, 0)
				if err != nil {
					continue
				}
				if number.Equal(bound) {
					return true
				}
			}
		case json.True, json.False, json.Null:
			for _, operand := range values {
				if kind == operand.(json.Value).Kind() {
					return true
				}
			}
		default:
			// Objects and arrays do not satisfy scalar membership.
		}
	case nil:
		for _, operand := range values {
			if operand, ok := operand.(json.Value); ok && operand.IsNull() {
				return true
			}
		}
	default:
		// Other record types do not satisfy scalar membership.
	}
	return false
}

func opStartsWith(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return strings.HasPrefix(v, v0.(string))
	case json.Value:
		if v.Kind() == json.String {
			v0 := v0.(json.Value)
			return v0.IsString() && strings.HasPrefix(v.String(), v0.String())
		}
	default:
		// Other record types cannot have a string prefix.
	}
	return false
}

func opEndsWith(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	switch v := v.(type) {
	case string:
		return strings.HasSuffix(v, v0.(string))
	case json.Value:
		if v.Kind() == json.String {
			v0 := v0.(json.Value)
			return v0.IsString() && strings.HasSuffix(v.String(), v0.String())
		}
	default:
		// Other record types cannot have a string suffix.
	}
	return false
}

func opIsBefore(v any, values []any) bool {
	if v == nil {
		return false
	}
	if v, ok := v.(int); ok {
		return v < values[0].(int)
	}
	v0 := values[0]
	return v.(time.Time).Before(v0.(time.Time))
}

func opIsOnOrBefore(v any, values []any) bool {
	if v == nil {
		return false
	}
	if v, ok := v.(int); ok {
		return v <= values[0].(int)
	}
	v0 := values[0].(time.Time)
	vv := v.(time.Time)
	return vv.Equal(v0) || vv.Before(v0)
}

func opIsAfter(v any, values []any) bool {
	if v == nil {
		return false
	}
	if v, ok := v.(int); ok {
		return v > values[0].(int)
	}
	v0 := values[0]
	return v.(time.Time).After(v0.(time.Time))
}

func opIsOnOrAfter(v any, values []any) bool {
	if v == nil {
		return false
	}
	if v, ok := v.(int); ok {
		return v >= values[0].(int)
	}
	v0 := values[0].(time.Time)
	vv := v.(time.Time)
	return vv.Equal(v0) || vv.After(v0)
}

func opIsTrue(v any) bool {
	switch v := v.(type) {
	case bool:
		return v
	case json.Value:
		return v.IsTrue()
	}
	return false
}

func opIsFalse(v any) bool {
	switch v := v.(type) {
	case bool:
		return !v
	case json.Value:
		return v.IsFalse()
	}
	return false
}

func opIsNull(v any) bool {
	if v, ok := v.(json.Value); ok {
		return v.IsNull()
	}
	return v == nil
}

func opIsEmpty(v any) bool {
	switch v := v.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case json.Value:
		return v.IsNull() || v.IsEmpty()
	case []any:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	}
	return false
}

// readAttributeFrom reads the property with the given path from m, returning
// its value (if found, otherwise nil) and a boolean indicating if the property
// path corresponds to a value in m or not.
func readAttributeFrom(m map[string]any, path []string) (any, bool) {
	last := len(path) - 1
	for i, name := range path {
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if i == last {
			return v, true
		}
		switch v := v.(type) {
		case map[string]any:
			m = v
		case json.Value:
			if v, ok := v.Get(path[i+1:]); ok {
				return v, true
			}
			return nil, false
		default:
			return nil, false
		}
	}
	panic("unreachable code")
}
