// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package filters

import (
	"strings"
	"time"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/json"
)

// Applies reports whether where matches the given attributes.
func Applies(where *state.Where, attributes map[string]any) bool {
	if where == nil {
		return true
	}
	for _, cond := range where.Conditions {
		op := cond.Operator
		v, exists := readAttributeFrom(attributes, cond.Property)
		var applies bool
		switch op {
		case state.OpIs:
			applies = opIs(v, cond.Values)
		case state.OpIsNot:
			applies = !opIs(v, cond.Values)
		case state.OpIsLessThan:
			applies = opIsLessThan(v, cond.Values)
		case state.OpIsLessThanOrEqualTo:
			applies = opIsLessThanOrEqualTo(v, cond.Values)
		case state.OpIsGreaterThan:
			applies = opIsGreaterThan(v, cond.Values)
		case state.OpIsGreaterThanOrEqualTo:
			applies = opIsGreaterThanOrEqualTo(v, cond.Values)
		case state.OpIsBetween:
			applies = opIsBetween(v, cond.Values)
		case state.OpIsNotBetween:
			applies = !opIsBetween(v, cond.Values)
		case state.OpContains:
			applies = opContains(v, cond.Values)
		case state.OpDoesNotContain:
			applies = !opContains(v, cond.Values)
		case state.OpIsOneOf:
			applies = opIsIn(v, cond.Values)
		case state.OpIsNotOneOf:
			applies = !opIsIn(v, cond.Values)
		case state.OpStartsWith:
			applies = opStartsWith(v, cond.Values)
		case state.OpEndsWith:
			applies = opEndsWith(v, cond.Values)
		case state.OpIsBefore:
			applies = opIsBefore(v, cond.Values)
		case state.OpIsOnOrBefore:
			applies = opIsOnOrBefore(v, cond.Values)
		case state.OpIsAfter:
			applies = opIsAfter(v, cond.Values)
		case state.OpIsOnOrAfter:
			applies = opIsOnOrAfter(v, cond.Values)
		case state.OpIsTrue:
			applies = opIsTrue(v)
		case state.OpIsFalse:
			applies = opIsFalse(v)
		case state.OpIsEmpty:
			applies = opIsEmpty(v)
		case state.OpIsNotEmpty:
			applies = !opIsEmpty(v)
		case state.OpIsNull:
			applies = exists && opIsNull(v)
		case state.OpIsNotNull:
			applies = !(exists && opIsNull(v))
		case state.OpExists:
			applies = exists
		case state.OpDoesNotExist:
			applies = !exists
		}
		if applies && where.Logical == state.OpOr {
			return true
		}
		if !applies && where.Logical == state.OpAnd {
			return false
		}
	}
	if where.Logical == state.OpOr {
		return false // none of the conditions applied.
	}
	// All the conditions applied.
	return true
}

func opIs(v any, values []any) bool {
	v0 := values[0]
	switch v := v.(type) {
	case decimal.Decimal:
		return v.Equal(v0.(decimal.Decimal))
	case time.Time:
		return v.Equal(v0.(time.Time))
	case json.Value:
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.Equal(*v0.Number)
			}
		case json.String:
			return v.String() == v0.String
		}
		return false
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
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.Less(*v0.Number)
			}
		case json.String:
			return v.String() < v0.String
		}
	}
	return false
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
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.LessEqual(*v0.Number)
			}
		case json.String:
			return v.String() <= v0.String
		}
	}
	return false
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
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.Greater(*v0.Number)
			}
		case json.String:
			return v.String() > v0.String
		}
	}
	return false
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
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.GreaterEqual(*v0.Number)
			}
		case json.String:
			return v.String() >= v0.String
		}
	}
	return false
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
		v0 := v0.(state.JSONConditionValue)
		v1 := v1.(state.JSONConditionValue)
		switch v.Kind() {
		case json.Number:
			if v0.Number != nil {
				v, err := v.Decimal(0, 0)
				return err == nil && v.GreaterEqual(*v0.Number) && v.LessEqual(*v1.Number)
			}
		case json.String:
			v := v.String()
			return v0.String <= v && v <= v1.String
		}
	}
	return false
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
		v0 := v0.(state.JSONConditionValue)
		switch v.Kind() {
		case json.String:
			return strings.Contains(v.String(), v0.String)
		case json.Array:
			for _, ev := range v.Elements() {
				if opIs(ev, values) {
					return true
				}
			}
		}
	case []any:
		for _, ev := range v {
			if opIs(ev, values) {
				return true
			}
		}
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
		switch v.Kind() {
		case json.Number:
			v, err := v.Decimal(0, 0)
			if err == nil {
				for _, vi := range values {
					vi := vi.(state.JSONConditionValue)
					if vi.Number != nil && v.Equal(*vi.Number) {
						return true
					}
				}
			}
		case json.String:
			v := v.String()
			for _, vi := range values {
				vi := vi.(state.JSONConditionValue)
				if v == vi.String {
					return true
				}
			}
		}
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
			v0 := v0.(state.JSONConditionValue)
			return strings.HasPrefix(v.String(), v0.String)
		}
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
			v0 := v0.(state.JSONConditionValue)
			return strings.HasSuffix(v.String(), v0.String)
		}
	}
	return false
}

func opIsBefore(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	return v.(time.Time).Before(v0.(time.Time))
}

func opIsOnOrBefore(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0].(time.Time)
	vv := v.(time.Time)
	return vv.Equal(v0) || vv.Before(v0)
}

func opIsAfter(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	return v.(time.Time).After(v0.(time.Time))
}

func opIsOnOrAfter(v any, values []any) bool {
	if v == nil {
		return false
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
		return v.Bool()
	}
	return false
}

func opIsFalse(v any) bool {
	switch v := v.(type) {
	case bool:
		return !v
	case json.Value:
		return !v.Bool()
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
