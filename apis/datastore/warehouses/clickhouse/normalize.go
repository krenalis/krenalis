//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package clickhouse

import (
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/meergo/meergo/apis/datastore/warehouses"
	"github.com/meergo/meergo/types"

	"github.com/shopspring/decimal"
)

// Normalize normalizes a value v returned by the Query method.
func (warehouse *ClickHouse) Normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
	if v == nil {
		if !nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but Clickhouse returned a NULL value", name)
		}
		return nil, nil
	}
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := v.(bool); ok {
			return v, nil
		}
	case types.IntKind:
		var n int
		switch v := v.(type) {
		case int8:
			n = int(v)
		case int16:
			n = int(v)
		case int32:
			n = int(v)
		case int64:
			n = int(v)
		default:
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Int type", v, name)
		}
		return warehouses.ValidateInt(name, typ, n)
	case types.UintKind:
		var n uint
		switch v := v.(type) {
		case uint8:
			n = uint(v)
		case uint16:
			n = uint(v)
		case uint32:
			n = uint(v)
		case uint64:
			n = uint(v)
		default:
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Uint type", v, name)
		}
		return warehouses.ValidateUint(name, typ, n)
	case types.FloatKind:
		var n float64
		switch v := v.(type) {
		case float32:
			n = float64(v)
		case float64:
			n = v
		default:
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Float type", v, name)
		}
		return warehouses.ValidateFloat(name, typ, n)
	case types.DecimalKind:
		if v, ok := v.(decimal.Decimal); ok {
			return warehouses.ValidateDecimal(name, typ, v)
		}
	case types.DateTimeKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDateTime(name, v)
		}
	case types.DateKind:
		if v, ok := v.(time.Time); ok {
			return warehouses.ValidateDate(name, v)
		}
	case types.UUIDKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateUUID(name, v)
		}
	case types.InetKind:
		if v, ok := v.(net.IP); ok {
			return warehouses.ValidateInet(name, v.String())
		}
	case types.TextKind:
		if v, ok := v.(string); ok {
			return warehouses.ValidateText(name, typ, v)
		}
	case types.ArrayKind:
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Slice {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is an Array type", v, name)
		}
		var err error
		n := rv.Len()
		if n < typ.MinItems() || n > typ.MaxItems() {
			return nil, fmt.Errorf("data warehouse returned an array with %d items for column %s, which is not within the expected range of [%d, %d]",
				n, name, typ.MinItems(), typ.MaxItems())
		}
		a := make([]any, n)
		t := typ.Elem()
		for i := 0; i < n; i++ {
			e := rv.Index(i).Interface()
			a[i], err = warehouse.Normalize(name, t, e, false)
			if err != nil {
				return nil, err
			}
		}
		return a, nil
	case types.MapKind:
		rv := reflect.ValueOf(v)
		if t := rv.Type(); t.Kind() != reflect.Map || t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("data warehouse returned a value of type %T for column %s which is a Map type", v, name)
		}
		n := rv.Len()
		m := make(map[string]any, n)
		t := typ.Elem()
		iter := rv.MapRange()
		var err error
		for iter.Next() {
			k := iter.Key().String()
			v := iter.Value().Interface()
			m[k], err = warehouse.Normalize(name, t, v, false)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	}
	return nil, fmt.Errorf("Clickhouse has returned an unsupported type %T for column %s", v, name)
}
