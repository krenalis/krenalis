//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package apis

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"reflect"
	"strconv"
	"time"
	"unicode/utf8"

	_connector "chichi/connector"
	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	arrayType  = reflect.TypeOf(([]any)(nil))
	objectType = reflect.TypeOf((map[string]any)(nil))
	mapType    = objectType
)

// normalizeAppPropertyValue normalizes a property value returned by an app
// connector, and returns its normalized value. If the value is not valid
// it returns an error.
func normalizeAppPropertyValue(name string, nullable bool, typ types.Type, src any) (any, error) {
	if src == nil {
		if !nullable {
			return nil, fmt.Errorf("property %s is non-nullable, but the app returned a nil value", name)
		}
		return nil, nil
	}
	var value any
	var valid bool
	switch pt := typ.PhysicalType(); pt {
	case types.PtBoolean:
		if _, ok := src.(bool); ok {
			value = src
			valid = true
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var v int64
		switch src := src.(type) {
		case int:
			v = int64(src)
			valid = true
		case float64:
			v = int64(src)
			valid = true
		case json.Number:
			var err error
			v, err = src.Int64()
			if err != nil {
				var f float64
				f, err = src.Float64()
				v = int64(f)
			}
			valid = err == nil
		}
		if valid {
			min, max := typ.IntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("app returnd a value of %d for property %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = int(v)
		}
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var v uint64
		switch src := src.(type) {
		case uint:
			v = uint64(src)
			valid = true
		case float64:
			v = uint64(src)
			valid = true
		case json.Number:
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			if err != nil {
				var f float64
				f, err = src.Float64()
				v = uint64(f)
			}
			valid = err == nil
		}
		if valid {
			min, max := typ.UIntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("app returnd a value of %d for property %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = uint(v)
		}
	case types.PtFloat, types.PtFloat32:
		var v float64
		switch src := src.(type) {
		case float64:
			v = src
			valid = true
		case json.Number:
			var err error
			v, err = src.Float64()
			valid = err == nil
		}
		if valid {
			min, max := typ.FloatRange()
			if v < min || v > max {
				return nil, fmt.Errorf("app returnd a value of %f for property %s which is not within the expected range of [%f, %f]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDecimal:
		var v decimal.Decimal
		switch src := src.(type) {
		case decimal.Decimal:
			v = src
			valid = true
		case float64:
			v = decimal.NewFromFloat(src)
			valid = true
		case string:
			var err error
			v, err = decimal.NewFromString(src)
			valid = err == nil
		case json.Number:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		}
		if valid {
			min, max := typ.DecimalRange()
			if v.LessThan(min) || v.GreaterThan(max) {
				return nil, fmt.Errorf("app returnd a value of %s for property %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDateTime:
		var err error
		switch src := src.(type) {
		case _connector.DateTime:
			value = src
			valid = true
		case float64:
			switch typ.Layout() {
			case types.Nanoseconds, types.Microseconds, types.Milliseconds, types.Seconds:
				value, err = normalizeDateTimeFloat(typ.Layout(), src)
				valid = err == nil
			}
		case string:
			layout := typ.Layout()
			if layout == "" {
				layout = time.DateTime
			}
			switch layout {
			case types.Nanoseconds, types.Microseconds, types.Milliseconds, types.Seconds:
				n, err := strconv.ParseInt(src, 10, 64)
				if err == nil {
					value, err = normalizeDateTimeInt(layout, n)
					valid = err == nil
				}
			default:
				if t, err := time.Parse(layout, src); err == nil {
					value, err = _connector.AsDateTime(t)
					valid = err == nil
				}
			}
		case json.Number:
			if n, err := src.Int64(); err == nil {
				value, err = normalizeDateTimeInt(typ.Layout(), n)
				valid = err == nil
			} else if f, err := src.Float64(); err == nil {
				value, err = normalizeDateTimeFloat(typ.Layout(), f)
				valid = err == nil
			}
		}
	case types.PtDate:
		switch src := src.(type) {
		case _connector.Date:
			value = src
			valid = true
		case string:
			layout := typ.Layout()
			if layout == "" {
				layout = time.DateOnly
			}
			if t, err := time.Parse(layout, src); err == nil {
				value, err = _connector.AsDate(t)
				valid = err == nil
			}
		}
	case types.PtTime:
		switch src := src.(type) {
		case _connector.Time:
			value = src
			valid = true
		case float64:
			switch layout := typ.Layout(); layout {
			case types.Nanoseconds, types.Microseconds, types.Milliseconds, types.Seconds:
				var err error
				value, err = normalizeTimeFloat(layout, src)
				valid = err == nil
			}
		case string:
			layout := typ.Layout()
			if layout == "" {
				layout = "15:04:05.999999999"
			}
			switch layout {
			case types.Nanoseconds, types.Microseconds, types.Milliseconds, types.Seconds:
				n, err := strconv.ParseInt(src, 10, 64)
				if err == nil {
					value, err = normalizeTimeInt(layout, n)
					valid = err == nil
				}
			default:
				if t, err := time.Parse(layout, src); err == nil {
					t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
					value = _connector.Time(t.UnixNano())
					valid = true
				}
			}
		case json.Number:
			if n, err := src.Int64(); err == nil {
				value, err = normalizeTimeInt(typ.Layout(), n)
				valid = err == nil
			} else if f, err := src.Float64(); err == nil {
				value, err = normalizeTimeFloat(typ.Layout(), f)
				valid = err == nil
			}
		}
	case types.PtYear:
		var v int64
		switch src := src.(type) {
		case int:
			v = int64(src)
			valid = true
		case float64:
			v = int64(src)
			valid = true
		case json.Number:
			var err error
			v, err = src.Int64()
			valid = err == nil
		}
		value = int(v)
		valid = valid && types.MinYear <= v && v <= types.MaxYear
	case types.PtUUID:
		if s, ok := src.(string); ok {
			if v, err := uuid.Parse(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtJSON:
		switch src := src.(type) {
		case json.RawMessage:
			value = src
			valid = json.Valid(src)
		case string:
			v := json.RawMessage(src)
			value = v
			valid = json.Valid(v)
		}
	case types.PtInet:
		if s, ok := src.(string); ok {
			if v, err := netip.ParseAddr(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtText:
		var v string
		v, valid = src.(string)
		if valid {
			if !utf8.ValidString(v) {
				return nil, fmt.Errorf("app returned a value of %q for property %s, which does not contain valid UTF-8 characters",
					abbreviate(v, 20), name)
			}
			if l, ok := typ.ByteLen(); ok && len(v) > l {
				return nil, fmt.Errorf("app returned a value of %q for property %s, which is longer than %d bytes",
					abbreviate(v, 20), name, l)
			}
			if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
				return nil, fmt.Errorf("app returned a value of %q for property %s, which is longer than %d characters",
					abbreviate(v, 20), name, l)
			}
			value = v
		}
	case types.PtArray:
		rv := reflect.ValueOf(src)
		if rv.Type() == arrayType {
			var err error
			n := rv.Len()
			if n < typ.MinItems() || n > typ.MaxItems() {
				return nil, fmt.Errorf("app returned an array with %d items for property %s, which is not within the expected range of [%d, %d]",
					n, name, typ.MinItems(), typ.MaxItems())
			}
			a := make([]any, n)
			t := typ.ItemType()
			for i := 0; i < n; i++ {
				v := rv.Index(i).Interface()
				a[i], err = normalizeAppPropertyValue(name, false, t, v)
				if err != nil {
					return nil, err
				}
			}
			value = a
			valid = true
		}
	case types.PtObject:
		rv := reflect.ValueOf(src)
		if rv.Type() == objectType {
			var err error
			properties := typ.Properties()
			propertyByName := make(map[string]types.Property, len(properties))
			for _, p := range properties {
				propertyByName[p.Name] = p
			}
			n := rv.Len()
			obj := make(map[string]any, n)
			iter := rv.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				v := iter.Value().Interface()
				p, ok := propertyByName[k]
				if !ok {
					return nil, fmt.Errorf("app returned a non-existent property %s for for object property %s", k, name)
				}
				obj[k], err = normalizeAppPropertyValue(name, p.Nullable, p.Type, v)
				if err != nil {
					return nil, err
				}
			}
			value = obj
			valid = true
		}
	case types.PtMap:
		rv := reflect.ValueOf(src)
		if rv.Type() == mapType {
			var err error
			n := rv.Len()
			m := make(map[string]any, n)
			t := typ.ValueType()
			iter := rv.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				v := iter.Value().Interface()
				m[k], err = normalizeAppPropertyValue(name, false, t, v)
				if err != nil {
					return nil, err
				}
			}
			value = m
			valid = true
		}
	}
	if !valid {
		return nil, fmt.Errorf("app returned a value of %v for property %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	return value, nil
}

// normalizeDatabaseFilePropertyValue normalizes a property value returned by a
// database or file connector, and returns its normalized value. If the value is
// not valid it returns an error.
func normalizeDatabaseFilePropertyValue(property types.Property, src any) (any, error) {
	name := property.Name
	if src == nil {
		if !property.Nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but the database returned a NULL value", name)
		}
		return nil, nil
	}
	typ := property.Type
	var value any
	var valid bool
	switch typ.PhysicalType() {
	case types.PtBoolean:
		if _, ok := src.(bool); ok {
			value = src
			valid = true
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var v int64
		switch src := src.(type) {
		case int8:
			v = int64(src)
			valid = true
		case int16:
			v = int64(src)
			valid = true
		case int32:
			v = int64(src)
			valid = true
		case int64:
			v = src
			valid = true
		case []byte:
			var err error
			v, err = strconv.ParseInt(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.IntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = int(v)
		}
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var v uint64
		switch src := src.(type) {
		case uint8:
			v = uint64(src)
			valid = true
		case uint16:
			v = uint64(src)
			valid = true
		case uint32:
			v = uint64(src)
			valid = true
		case uint64:
			v = src
			valid = true
		case []byte:
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.UIntRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %d for column %s which is not within the expected range of [%d, %d]",
					v, name, min, max)
			}
			value = uint(v)
		}
	case types.PtFloat, types.PtFloat32:
		var v float64
		switch src := src.(type) {
		case float32:
			v = float64(src)
			valid = true
		case float64:
			v = src
			valid = true
		case []byte:
			var err error
			size := 64
			if typ.PhysicalType() == types.PtFloat32 {
				size = 32
			}
			v, err = strconv.ParseFloat(string(src), size)
			valid = err == nil
		}
		if valid {
			min, max := typ.FloatRange()
			if v < min || v > max {
				return nil, fmt.Errorf("database returnd a value of %f for column %s which is not within the expected range of [%f, %f]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDecimal:
		var v decimal.Decimal
		switch src := src.(type) {
		case string:
			var err error
			v, err = decimal.NewFromString(src)
			valid = err == nil
		case []byte:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		case int32:
			v = decimal.NewFromInt32(src)
			valid = true
		case int64:
			v = decimal.NewFromInt(src)
			valid = true
		case decimal.Decimal:
			v = src
			valid = true
		}
		if valid {
			min, max := typ.DecimalRange()
			if v.LessThan(min) || v.GreaterThan(max) {
				return nil, fmt.Errorf("database returnd a value of %s for column %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDateTime:
		if t, ok := src.(time.Time); ok {
			var err error
			value, err = _connector.AsDateTime(t)
			valid = err == nil
		}
	case types.PtDate:
		if t, ok := src.(time.Time); ok {
			var err error
			value, err = _connector.AsDate(t)
			valid = err == nil
		}
	case types.PtTime:
		if s, ok := src.([]byte); ok {
			var err error
			value, err = _connector.ParseTime(string(s))
			valid = err == nil
		}
	case types.PtYear:
		switch y := src.(type) {
		case int64:
			if valid = types.MinYear <= y && y <= types.MaxYear; valid {
				value = int(y)
			}
		case []byte:
			year, err := strconv.Atoi(string(y))
			value = year
			valid = err == nil && types.MinYear <= year && year <= types.MaxYear
		}
	case types.PtUUID:
		if s, ok := src.(string); ok {
			if v, err := uuid.Parse(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtJSON:
		if s, ok := src.([]byte); ok {
			if valid = json.Valid(s); valid {
				value = json.RawMessage(s)
			}
		}
	case types.PtInet:
		if s, ok := src.(string); ok {
			if v, err := netip.ParseAddr(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.PtText:
		var v string
		switch s := src.(type) {
		case string:
			v = s
			valid = true
		case []byte:
			v = string(s)
			valid = true
		}
		if valid {
			if !utf8.ValidString(v) {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
					abbreviate(v, 20), name)
			}
			if l, ok := typ.ByteLen(); ok && len(v) > l {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
					abbreviate(v, 20), name, l)
			}
			if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
				return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
					abbreviate(v, 20), name, l)
			}
			value = v
		}
	}
	if !valid {
		return nil, fmt.Errorf("database returned a value of %v for column %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	return value, nil
}

// validateStringProperty validates a string property like
// normalizeDatabaseFilePropertyValue does.
func validateStringProperty(p types.Property, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
			abbreviate(s, 20), p.Name)
	}
	if l, ok := p.Type.ByteLen(); ok && len(s) > l {
		return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
			abbreviate(s, 20), p.Name, l)
	}
	if l, ok := p.Type.CharLen(); ok && utf8.RuneCountInString(s) > l {
		return fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
			abbreviate(s, 20), p.Name, l)
	}
	return nil
}

func normalizeDateTimeInt(layout string, src int64) (_connector.DateTime, error) {
	switch layout {
	case types.Nanoseconds:
		return _connector.AsDateTime(time.Unix(0, src))
	case types.Microseconds:
		return _connector.AsDateTime(time.UnixMicro(src).UTC())
	case types.Milliseconds:
		return _connector.AsDateTime(time.UnixMilli(src).UTC())
	case types.Seconds:
		return _connector.AsDateTime(time.Unix(src, 0).UTC())
	}
	panic(errors.New("invalid layout"))
}

func normalizeDateTimeFloat(layout string, src float64) (_connector.DateTime, error) {
	switch layout {
	case types.Nanoseconds:
		return _connector.AsDateTime(time.Unix(0, int64(src)))
	case types.Microseconds:
		return _connector.AsDateTime(time.UnixMicro(int64(src)))
	case types.Milliseconds:
		return _connector.AsDateTime(time.UnixMilli(int64(src)))
	case types.Seconds:
		sec := int64(src)
		nsec := int64((src - float64(sec)) * 1e9)
		return _connector.AsDateTime(time.Unix(sec, nsec))
	}
	panic(errors.New("invalid layout"))
}

func normalizeTimeInt(layout string, src int64) (_connector.Time, error) {
	if src < 0 {
		return 0, errors.New("time overflow")
	}
	var t _connector.Time
	switch layout {
	case types.Nanoseconds:
		t = _connector.Time(src)
	case types.Microseconds:
		t = _connector.Time(src * int64(time.Microsecond))
	case types.Milliseconds:
		t = _connector.Time(src * int64(time.Millisecond))
	case types.Seconds:
		t = _connector.Time(src * int64(time.Second))
	default:
		panic(errors.New("invalid layout"))
	}
	if t < 0 || t > _connector.MaxTime {
		return 0, errors.New("time overflow")
	}
	return t, nil
}

func normalizeTimeFloat(layout string, src float64) (_connector.Time, error) {
	if src < 0 {
		return 0, errors.New("time overflow")
	}
	var t _connector.Time
	switch layout {
	case types.Nanoseconds:
		t = _connector.Time(int64(src))
	case types.Microseconds:
		t = _connector.Time(int64(src * float64(time.Microsecond)))
	case types.Milliseconds:
		t = _connector.Time(int64(src * float64(time.Millisecond)))
	case types.Seconds:
		t = _connector.Time(int64(src * float64(time.Second)))
	default:
		panic(errors.New("invalid layout"))
	}
	if t < 0 || t > _connector.MaxTime {
		return 0, errors.New("time overflow")
	}
	return t, nil
}
