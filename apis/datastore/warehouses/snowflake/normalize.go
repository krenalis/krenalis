//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package snowflake

import (
	"fmt"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"
)

// Normalize normalizes a value v returned by the Query method.
func (warehouse *Snowflake) Normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
	if v == nil {
		if !nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but Snowflake returned a NULL value", name)
		}
		return nil, nil
	}
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.FloatKind:
		if v, ok := v.(float64); ok {
			return warehouses.ValidateFloat(name, typ, v)
		}
	case types.DecimalKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateDecimalString(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDate(name, v)
		}
	case types.TimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateTime(v)
		}
	case types.JSONKind:
		return warehouses.ValidateJSON(name, v)
	case types.TextKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		// The driver returns the value as a JSON array.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		if v == "" {
			return nil, fmt.Errorf("data warehouse returned an empty string for column %s which is an Array type", name)
		}
		// Snowflake only supports JSON as the item type.
		if typ.Elem().Kind() != types.JSONKind {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		ev := json.Value(v)
		if json.Valid(ev) {
			return nil, fmt.Errorf("data warehouse returned a string with invalid JSON for column %s", name)
		}
		if !ev.IsArray() {
			return nil, fmt.Errorf("data warehouse returned a JSON %s for column %s which is an Array type", ev.Kind(), name)
		}
		min := typ.MinElements()
		max := typ.MaxElements()
		arr := []any{}
		for i, elem := range ev.Elements() {
			if i == max {
				return nil, fmt.Errorf("data warehouse returned an array with more than %d elements for column %s", max, name)
			}
			arr = append(arr, elem)
		}
		if len(arr) < min {
			return nil, fmt.Errorf("data warehouse returned an array with less than %d elements for column %s", min, name)
		}
		return arr, nil
	case types.MapKind:
		// The driver returns the value as a JSON object.
		v, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		if v == "" {
			return nil, fmt.Errorf("data warehouse returned an empty string for column %s which is an Array type", name)
		}
		// Snowflake only supports JSON as the item type.
		if typ.Elem().Kind() != types.JSONKind {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Map type", v, name)
		}
		ev := json.Value(v)
		if json.Valid(ev) {
			return nil, fmt.Errorf("data warehouse returned a string with invalid JSON for column %s", name)
		}
		if !ev.IsObject() {
			return nil, fmt.Errorf("data warehouse returned a JSON %s for column %s which is a Map type", ev.Kind(), name)
		}
		m := map[string]any{}
		for k, v := range ev.Properties() {
			m[k] = v
		}
		return m, nil
	}
	return nil, fmt.Errorf("Snowflake has returned an unsupported type %T for column %s", v, name)
}
