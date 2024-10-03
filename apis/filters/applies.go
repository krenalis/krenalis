//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package filters

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/meergo/meergo/apis/state"

	"github.com/shopspring/decimal"
)

// Applies determines whether where applies to the provided properties. Returns
// an error if any property in the where is not found in the properties map.
func Applies(where *state.Where, properties map[string]any) bool {
	if where == nil {
		return true
	}
	for _, cond := range where.Conditions {
		op := cond.Operator
		v, exists := readPropertyFrom(properties, cond.Property)
		if raw, ok := v.(json.RawMessage); ok {
			if c := raw[0]; c != '{' {
				v = nil
				_ = json.Unmarshal(raw, &v)
			}
		}
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
			applies = opIsNotBetween(v, cond.Values)
		case state.OpContains:
			applies = opContains(v, cond.Values)
		case state.OpDoesNotContain:
			applies = opDoesNotContain(v, cond.Values)
		case state.OpIsOneOf:
			applies = opIsIn(v, cond.Values)
		case state.OpIsNotOneOf:
			applies = opIsNotIn(v, cond.Values)
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
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				return decimal.NewFromFloat(v).Equal(*v0.Number)
			}
		case json.Number:
			if v0.Number != nil {
				return decimal.RequireFromString(string(v)).Equal(*v0.Number)
			}
		case string:
			return v == v0.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.Equal(v0.(decimal.Decimal))
	case time.Time:
		return v.Equal(v0.(time.Time))
	default:
		return v == v0
	}
}

func opIsLessThan(v any, values []any) bool {
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				return decimal.NewFromFloat(v).LessThan(*v0.Number)
			}
		case json.Number:
			if v0.Number != nil {
				return decimal.RequireFromString(string(v)).LessThan(*v0.Number)
			}
		case string:
			return v < v0.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.LessThan(v0.(decimal.Decimal))
	case int:
		return v < v0.(int)
	case uint:
		return v < v0.(uint)
	case float64:
		return v < v0.(float64)
	case string:
		return v < v0.(string)
	}
	return false
}

func opIsLessThanOrEqualTo(v any, values []any) bool {
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				return decimal.NewFromFloat(v).LessThanOrEqual(*v0.Number)
			}
		case json.Number:
			if v0.Number != nil {
				return decimal.RequireFromString(string(v)).LessThanOrEqual(*v0.Number)
			}
		case string:
			return v <= v0.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.LessThanOrEqual(v0.(decimal.Decimal))
	case int:
		return v <= v0.(int)
	case uint:
		return v <= v0.(uint)
	case float64:
		return v <= v0.(float64)
	case string:
		return v <= v0.(string)
	}
	return false
}

func opIsGreaterThan(v any, values []any) bool {
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				return decimal.NewFromFloat(v).GreaterThan(*v0.Number)
			}
		case json.Number:
			if v0.Number != nil {
				return decimal.RequireFromString(string(v)).GreaterThan(*v0.Number)
			}
		case string:
			return v > v0.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.GreaterThan(v0.(decimal.Decimal))
	case int:
		return v > v0.(int)
	case uint:
		return v > v0.(uint)
	case float64:
		return v > v0.(float64)
	case string:
		return v > v0.(string)
	}
	return false
}

func opIsGreaterThanOrEqualTo(v any, values []any) bool {
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				return decimal.NewFromFloat(v).GreaterThanOrEqual(*v0.Number)
			}
		case json.Number:
			if v0.Number != nil {
				return decimal.RequireFromString(string(v)).GreaterThanOrEqual(*v0.Number)
			}
		case string:
			return v >= v0.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.GreaterThanOrEqual(v0.(decimal.Decimal))
	case int:
		return v >= v0.(int)
	case uint:
		return v >= v0.(uint)
	case float64:
		return v >= v0.(float64)
	case string:
		return v >= v0.(string)
	}
	return false
}

func opIsBetween(v any, values []any) bool {
	v0 := values[0]
	v1 := values[1]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		v1 := v1.(*state.JSONConditionValue)
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				v := decimal.NewFromFloat(v)
				return v.GreaterThanOrEqual(*v0.Number) && v.LessThanOrEqual(*v1.Number)
			}
		case json.Number:
			if v0.Number != nil {
				v := decimal.RequireFromString(string(v))
				return v.GreaterThanOrEqual(*v0.Number) && v.LessThanOrEqual(*v1.Number)
			}
		case string:
			return v0.String <= v && v <= v1.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.GreaterThanOrEqual(v0.(decimal.Decimal)) && v.LessThanOrEqual(v1.(decimal.Decimal))
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
	case string:
		return v0.(string) <= v && v <= v1.(string)
	}
	return false
}

func opIsNotBetween(v any, values []any) bool {
	v0 := values[0]
	v1 := values[1]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		v1 := v1.(*state.JSONConditionValue)
		switch v := v.(type) {
		case float64:
			if v0.Number != nil {
				v := decimal.NewFromFloat(v)
				return v.LessThan(*v0.Number) || v.GreaterThan(*v1.Number)
			}
		case json.Number:
			if v0.Number != nil {
				v := decimal.RequireFromString(string(v))
				return v.LessThan(*v0.Number) || v.GreaterThan(*v1.Number)
			}
		case string:
			return v < v0.String || v > v1.String
		}
		return false
	}
	switch v := v.(type) {
	case decimal.Decimal:
		return v.LessThan(v0.(decimal.Decimal)) || v.GreaterThan(v1.(decimal.Decimal))
	case time.Time:
		v0 := v0.(time.Time)
		v1 := v1.(time.Time)
		return v.Before(v0) || v.After(v1)
	case int:
		return v < v0.(int) || v > v1.(int)
	case uint:
		return v < v0.(uint) || v > v1.(uint)
	case float64:
		return v < v0.(float64) || v > v1.(float64)
	case string:
		return v < v0.(string) || v > v1.(string)
	}
	return false
}

func opContains(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case string:
			return strings.Contains(v, v0.String)
		case []any:
			for _, ev := range v {
				if opIs(ev, values) {
					return true
				}
			}
		}
		return false
	}
	switch v := v.(type) {
	case string:
		return strings.Contains(v, v0.(string))
	case []any:
		for _, ev := range v {
			if opIs(ev, values) {
				return true
			}
		}
	}
	return false
}

func opDoesNotContain(v any, values []any) bool {
	if v == nil {
		return true
	}
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case string:
			return !strings.Contains(v, v0.String)
		case []any:
			for _, ev := range v {
				if opIs(ev, values) {
					return false
				}
			}
		}
		return true
	}
	switch v := v.(type) {
	case string:
		return !strings.Contains(v, v0.(string))
	case []any:
		for _, ev := range v {
			if opIs(ev, values) {
				return false
			}
		}
	}
	return true
}

func opIsIn(v any, values []any) bool {
	if _, ok := values[0].(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			vv := decimal.NewFromFloat(v)
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); vi.Number != nil && vv.Equal(*vi.Number) {
					return true
				}
			}
		case json.Number:
			vv := decimal.RequireFromString(string(v))
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); vi.Number != nil && vv.Equal(*vi.Number) {
					return true
				}
			}
		case string:
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); v == vi.String {
					return true
				}
			}
		}
		return false
	}
	switch v := v.(type) {
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
	case string:
		for _, vi := range values {
			if v == vi.(string) {
				return true
			}
		}
	}
	return false
}

func opIsNotIn(v any, values []any) bool {
	if _, ok := values[0].(*state.JSONConditionValue); ok {
		switch v := v.(type) {
		case float64:
			vv := decimal.NewFromFloat(v)
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); vi.Number != nil && vv.Equal(*vi.Number) {
					return false
				}
			}
		case json.Number:
			vv := decimal.RequireFromString(string(v))
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); vi.Number != nil && vv.Equal(*vi.Number) {
					return false
				}
			}
		case string:
			for _, vi := range values {
				if vi := vi.(*state.JSONConditionValue); v == vi.String {
					return false
				}
			}
		}
		return true
	}
	switch v := v.(type) {
	case decimal.Decimal:
		for _, vi := range values {
			if v.Equal(vi.(decimal.Decimal)) {
				return false
			}
		}
	case time.Time:
		for _, vi := range values {
			if v.Equal(vi.(time.Time)) {
				return false
			}
		}
	case int:
		for _, vi := range values {
			if v == vi.(int) {
				return false
			}
		}
	case uint:
		for _, vi := range values {
			if v == vi.(uint) {
				return false
			}
		}
	case float64:
		for _, vi := range values {
			if v == vi.(float64) {
				return false
			}
		}
	case string:
		for _, vi := range values {
			if v == vi.(string) {
				return false
			}
		}
	}
	return true
}

func opStartsWith(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		if v, ok := v.(string); ok {
			return strings.HasPrefix(v, v0.String)
		}
		return false
	}
	return strings.HasPrefix(v.(string), v0.(string))
}

func opEndsWith(v any, values []any) bool {
	if v == nil {
		return false
	}
	v0 := values[0]
	if v0, ok := v0.(*state.JSONConditionValue); ok {
		if v, ok := v.(string); ok {
			return strings.HasSuffix(v, v0.String)
		}
		return false
	}
	return strings.HasSuffix(v.(string), v0.(string))
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
	if v == nil {
		return false
	}
	vv, _ := v.(bool)
	return vv
}

func opIsFalse(v any) bool {
	if v == nil {
		return false
	}
	vv, ok := v.(bool)
	return ok && !vv
}

func opIsNull(v any) bool {
	return v == nil
}

// readPropertyFrom reads the property with the given path from m, returning its
// value (if found, otherwise nil) and a boolean indicating if the property path
// corresponds to a value in m or not.
func readPropertyFrom(m map[string]any, path string) (any, bool) {
	var name string
	for {
		name, path, _ = strings.Cut(path, ".")
		v, ok := m[name]
		if !ok {
			return nil, false
		}
		if path == "" {
			return v, true
		}
		m, ok = v.(map[string]any)
		if !ok {
			return nil, false
		}
	}
}
