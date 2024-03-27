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

	"github.com/open2b/chichi/apis/datastore/warehouses"
	"github.com/open2b/chichi/types"

	"github.com/shopspring/decimal"
)

// scanValue implements the sql.Scanner interface to read the database values.
type scanValue struct {
	column      types.Property
	rows        *[][]any
	columnIndex int
	columnCount int
}

// newScanValues returns a slice containing scan values to be used to scan rows.
func newScanValues(columns []types.Property, rows *[][]any) []any {
	values := make([]any, len(columns))
	for i, c := range columns {
		values[i] = scanValue{
			column:      c,
			rows:        rows,
			columnIndex: i,
			columnCount: len(columns),
		}
	}
	return values
}

func (sv scanValue) Scan(src any) error {
	c := sv.column
	value, err := normalize(c.Name, c.Type, src, c.Nullable)
	if err != nil {
		return err
	}
	var row []any
	if sv.columnIndex == 0 {
		row = make([]any, sv.columnCount)
		*sv.rows = append(*sv.rows, row)
	} else {
		row = (*sv.rows)[len(*sv.rows)-1]
	}
	row[sv.columnIndex] = value
	return nil
}

// normalize normalizes a value returned by Clickhouse and returns its
// normalized form. If the value is not valid it returns an error.
func normalize(name string, typ types.Type, v any, nullable bool) (any, error) {
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
			a[i], err = normalize(name, t, e, false)
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
			m[k], err = normalize(name, t, v, false)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	}
	return nil, fmt.Errorf("Clickhouse has returned an unsupported type %T for column %s", v, name)
}
