//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"time"
	"unicode/utf8"

	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

var errInvalidConversion = errors.New("cannot convert")

const (
	// Range of Float values convertible to Int64: [-9223372036854776000.0, 9223372036854775000.0].
	// These are the same limits that PostgreSQL uses when converting a double precision value to a bigint.
	minFloatConvertibleToInt64 = -9223372036854776000.0 // converted to -9223372036854775808
	maxFloatConvertibleToInt64 = 9223372036854775000.0  // converted to 9223372036854774784

	// Range of Float values convertible to UInt64: [0, 18446744073709550000].
	maxFloatConvertibleToUInt64 = 18446744073709550000.0 // converted to 18446744073709549568
)

var (
	minIntDecimal  = decimal.NewFromInt(math.MinInt64)
	maxIntDecimal  = decimal.NewFromInt(math.MaxInt64)
	maxUIntDecimal = decimal.RequireFromString("18446744073709551615")
)

// convert converts v from type t1 to type t2 and returns the converted value.
// nullable reports whether nil is allowed as return value.
// For Array, Object, and Map values, it can modify the argument v.
//
// It returns an error if v cannot be converted, and panics if v is nil.
func convert(v any, t1, t2 types.Type, nullable bool) (any, error) {
	pt1 := t1.PhysicalType()
	pt2 := t2.PhysicalType()
	if nullable {
		switch {
		case pt1 == types.PtJSON && pt2 != types.PtJSON:
			if v, ok := v.(json.RawMessage); ok && v[0] == 'n' {
				return nil, nil
			}
		case v == "" && pt2 != types.PtText:
			return nil, nil
		}
	}
	switch pt2 {
	case types.PtBoolean:
		switch pt1 {
		case types.PtBoolean:
			return v.(bool), nil
		case types.PtText:
			switch v.(string) {
			case "false", "False", "FALSE", "no", "No", "NO":
				return false, nil
			case "true", "True", "TRUE", "yes", "Yes", "YES":
				return true, nil
			}
		case types.PtJSON:
			return jsonToBoolean(v)
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var err error
		var n int
		switch pt1 {
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			n = v.(int)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			u := v.(uint)
			if u > math.MaxInt64 {
				err = errInvalidConversion
			}
			n = int(u)
		case types.PtFloat, types.PtFloat32:
			n, err = floatToInt(v.(float64))
		case types.PtDecimal:
			n, err = decimalToInt(v.(decimal.Decimal))
		case types.PtYear:
			n = v.(int)
		case types.PtText:
			n, err = strconv.Atoi(v.(string))
		case types.PtJSON:
			n, err = jsonToInt(v)
		default:
			err = errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := t2.IntRange(); int64(n) < min || int64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var err error
		var n uint
		switch pt1 {
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			i := v.(int)
			if i < 0 {
				return nil, errInvalidConversion
			}
			n = uint(i)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			n = v.(uint)
		case types.PtFloat, types.PtFloat32:
			n, err = floatToUInt(v.(float64))
		case types.PtDecimal:
			n, err = decimalToUInt(v.(decimal.Decimal))
		case types.PtYear:
			n = uint(v.(int))
		case types.PtText:
			var u uint64
			u, err = strconv.ParseUint(v.(string), 10, 64)
			n = uint(u)
		case types.PtJSON:
			n, err = jsonToUInt(v)
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		min, max := t2.UIntRange()
		if uint64(n) < min || uint64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtFloat, types.PtFloat32:
		var err error
		var n float64
		switch pt1 {
		case types.PtFloat:
			n = v.(float64)
			if pt2 == types.PtFloat32 {
				n = float64(float32(n))
			}
		case types.PtFloat32:
			n = v.(float64)
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			n = float64(v.(int))
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			n = float64(v.(uint))
		case types.PtDecimal:
			n, _ = v.(decimal.Decimal).Float64()
			if pt2 == types.PtFloat32 {
				n = float64(float32(n))
			}
		case types.PtText:
			bits := 64
			if pt2 == types.PtFloat32 {
				bits = 32
			}
			n, err = strconv.ParseFloat(v.(string), bits)
		case types.PtJSON:
			bits := 64
			if pt2 == types.PtFloat32 {
				bits = 32
			}
			n, err = jsonToFloat(v, bits)
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := t2.FloatRange(); n < min || n > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtDecimal:
		var err error
		var n decimal.Decimal
		switch pt1 {
		case types.PtDecimal:
			n, _ = v.(decimal.Decimal)
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			n = decimal.New(int64(v.(int)), 0)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			n, _ = decimal.NewFromString(strconv.FormatUint(uint64(v.(uint)), 10))
		case types.PtFloat, types.PtFloat32:
			f := v.(float64)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, errInvalidConversion
			}
			n = decimal.NewFromFloat(f)
		case types.PtText:
			n, err = decimal.NewFromString(v.(string))
		case types.PtJSON:
			n, err = jsonToDecimal(v)
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := t2.DecimalRange(); n.LessThan(min) || n.GreaterThan(max) {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtDateTime:
		switch pt1 {
		case types.PtDateTime, types.PtDate:
			return v.(time.Time), nil
		case types.PtText:
			t, err := time.Parse(time.RFC3339Nano, v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return t.UTC(), nil
		case types.PtJSON:
			return jsonToDateTime(v)
		}
	case types.PtDate:
		switch pt1 {
		case types.PtDate:
			return v.(time.Time), nil
		case types.PtDateTime:
			t := v.(time.Time)
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
		case types.PtText:
			d, ok := convertTextToDate(v.(string))
			if !ok {
				return nil, errInvalidConversion
			}
			return d, nil
		case types.PtJSON:
			return jsonToDate(v)
		}
	case types.PtTime:
		switch pt1 {
		case types.PtTime:
			return v.(time.Time), nil
		case types.PtDateTime:
			t := v.(time.Time)
			return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
		case types.PtText:
			if t, ok := parseTime(v.(string)); ok {
				return t, nil
			}
		case types.PtJSON:
			return jsonToTime(v)
		}
	case types.PtYear:
		var err error
		var n int
		switch pt1 {
		case types.PtYear:
			return v.(int), nil
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			n = v.(int)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			u := v.(uint)
			if u > math.MaxInt64 {
				return nil, errInvalidConversion
			}
			n = int(u)
		case types.PtText:
			s := v.(string)
			if l := len(s); l == 0 || l > 4 || s[0] == '+' || s[0] == '-' || s[0] == '0' {
				return nil, errInvalidConversion
			}
			n, err = strconv.Atoi(s)
		case types.PtJSON:
			n, err = jsonToYear(v)
		default:
			return nil, errInvalidConversion
		}
		if err == nil && 1 <= n && n <= 9999 {
			return n, nil
		}
	case types.PtUUID:
		switch pt1 {
		case types.PtUUID:
			return v.(string), nil
		case types.PtText:
			u, err := uuid.Parse(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return u.String(), nil
		case types.PtJSON:
			return jsonToUUID(v)
		}
	case types.PtJSON:
		switch v := v.(type) {
		case json.RawMessage:
			return v, nil
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			return json.RawMessage(b), nil
		}
	case types.PtInet:
		switch pt1 {
		case types.PtInet:
			return v.(string), nil
		case types.PtText:
			ip, err := netip.ParseAddr(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return ip.String(), nil
		case types.PtJSON:
			return jsonToInet(v)
		}
	case types.PtText:
		var s string
		switch pt1 {
		case types.PtText:
			s = v.(string)
		case types.PtBoolean:
			s = "false"
			if v.(bool) {
				s = "true"
			}
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			s = strconv.FormatInt(int64(v.(int)), 10)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			s = strconv.FormatUint(uint64(v.(uint)), 10)
		case types.PtFloat, types.PtFloat32:
			bits := 64
			if pt1 == types.PtFloat32 {
				bits = 32
			}
			s = strconv.FormatFloat(v.(float64), 'g', -1, bits)
		case types.PtDecimal:
			s = v.(decimal.Decimal).String()
		case types.PtDateTime:
			s = v.(time.Time).Format(time.RFC3339Nano)
		case types.PtDate:
			s = v.(time.Time).Format(time.DateOnly)
		case types.PtTime:
			s = v.(time.Time).Format("15:04:05.999999999")
		case types.PtYear:
			s = strconv.Itoa(v.(int))
		case types.PtUUID, types.PtInet:
			s = v.(string)
		case types.PtJSON:
			var err error
			s, err = jsonToText(v)
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if enum := t2.Enum(); enum != nil {
			if s == "" && nullable {
				return nil, nil
			}
			if slices.Contains(enum, s) {
				return s, nil
			}
			return nil, errInvalidConversion
		}
		if re := t2.Regexp(); re != nil {
			if !re.MatchString(s) {
				if s == "" && nullable {
					return nil, nil
				}
				return nil, errInvalidConversion
			}
		}
		if l, ok := t2.ByteLen(); ok && l < len(s) {
			return nil, errInvalidConversion
		}
		if l, ok := t2.CharLen(); ok {
			runes := len(s)
			if pt1 == types.PtJSON || pt1 == types.PtText {
				runes = utf8.RuneCountInString(s)
			}
			if runes > l {
				return nil, errInvalidConversion
			}
		}
		return s, nil
	case types.PtArray:
		switch pt1 {
		case types.PtArray:
			s := v.([]any)
			if len(s) < t2.MinItems() || len(s) > t2.MaxItems() {
				return nil, errInvalidConversion
			}
			it1 := t1.ItemType()
			it2 := t2.ItemType()
			if it1.EqualTo(it2) {
				return s, nil
			}
			var err error
			for i, item := range s {
				s[i], err = convert(item, it1, it2, false)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		case types.PtJSON:
			s, err := jsonToArray(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			if len(s) < t2.MinItems() || len(s) > t2.MaxItems() {
				return nil, errInvalidConversion
			}
			it2 := t2.ItemType()
			for i, item := range s {
				s[i], err = convert(item, types.JSON(), it2, false)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		}
	case types.PtObject:
		switch pt1 {
		case types.PtObject:
			obj := v.(map[string]any)
			if t1.EqualTo(t2) {
				return obj, nil
			}
			for name, value := range obj {
				p2, ok := t2.Property(name)
				if !ok {
					delete(obj, name)
					continue
				}
				if value == nil {
					if !p2.Nullable {
						return nil, errInvalidConversion
					}
					continue
				}
				var err error
				p1, ok := t1.Property(name)
				if !ok {
					panic(fmt.Sprintf("unknown property %s", name))
				}
				obj[name], err = convert(value, p1.Type, p2.Type, p2.Nullable)
				if err != nil {
					return nil, err
				}
			}
			return obj, nil
		case types.PtJSON:
			s, err := jsonToMap(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			for name, value := range s {
				p2, ok := t2.Property(name)
				if !ok {
					delete(s, name)
					continue
				}
				if value == nil {
					if !p2.Nullable {
						return nil, errInvalidConversion
					}
					continue
				}
				var err error
				s[name], err = convert(value, types.JSON(), p2.Type, p2.Nullable)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		}
	case types.PtMap:
		switch pt1 {
		case types.PtMap:
			vt1 := t1.ValueType()
			vt2 := t2.ValueType()
			m := v.(map[string]any)
			if vt1.EqualTo(vt2) {
				return m, nil
			}
			var err error
			for key, value := range m {
				m[key], err = convert(value, vt1, vt2, false)
				if err != nil {
					return nil, err
				}
			}
			return m, nil
		case types.PtJSON:
			s, err := jsonToMap(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			vt2 := t2.ValueType()
			for key, value := range s {
				s[key], err = convert(value, types.JSON(), vt2, false)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		}
	}
	return nil, errInvalidConversion
}

// jsonToBoolean converts v of type JSON to Boolean.
func jsonToBoolean(v any) (bool, error) {
	switch v := v.(type) {
	case bool:
		return v, nil
	case json.RawMessage:
		if v[0] == 'f' {
			return false, nil
		}
		if v[0] == 't' {
			return true, nil
		}
	}
	return false, errInvalidConversion
}

// jsonToInt converts v of type JSON to Int.
func jsonToInt(v any) (int, error) {
	switch v := v.(type) {
	case float64:
		if v < minFloatConvertibleToInt64 || v > maxFloatConvertibleToInt64 {
			return 0, errInvalidConversion
		}
		return int(math.Round(v)), nil
	case json.Number:
		return strconv.Atoi(string(v))
	case json.RawMessage:
		return strconv.Atoi(string(v))
	}
	return 0, errInvalidConversion
}

// jsonToUInt converts v of type JSON to UInt.
func jsonToUInt(v any) (uint, error) {
	switch v := v.(type) {
	case float64:
		if v < 0 || v > maxFloatConvertibleToUInt64 {
			return 0, errInvalidConversion
		}
		return uint(math.Round(v)), nil
	case json.Number:
		n, err := strconv.ParseUint(string(v), 10, 64)
		if err == nil {
			return uint(n), nil
		}
		var f float64
		f, err = strconv.ParseFloat(string(v), 64)
		if err == nil {
			return uint(f), nil
		}
		var d decimal.Decimal
		d, err = decimal.NewFromString(string(v))
		if err == nil {
			f, _ = d.Float64()
			return uint(f), nil
		}
	case json.RawMessage:
		return jsonToUInt(json.Number(v))
	}
	return 0, errInvalidConversion
}

// jsonToFloat converts v of type JSON to Float or Float64 depending on bitSize
// bits (32 for Float32, 64 for Float).
func jsonToFloat(v any, bitSize int) (float64, error) {
	switch v := v.(type) {
	case float64:
		if bitSize == 32 {
			return float64(float32(v)), nil
		}
		return v, nil
	case json.Number:
		n, err := strconv.ParseFloat(string(v), bitSize)
		if err == nil {
			return n, nil
		}
		d, err := decimal.NewFromString(string(v))
		if err == nil {
			n, _ = d.Float64()
			if bitSize == 32 {
				n = float64(float32(n))
			}
			return n, nil
		}
	case json.RawMessage:
		return jsonToFloat(json.Number(v), bitSize)
	}
	return 0, errInvalidConversion
}

// jsonToDecimal converts v of type JSON to Decimal.
func jsonToDecimal(v any) (decimal.Decimal, error) {
	switch v := v.(type) {
	case float64:
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			return decimal.NewFromFloat(v), nil
		}
	case json.Number:
		return decimal.NewFromString(string(v))
	case json.RawMessage:
		return decimal.NewFromString(string(v))
	}
	return decimal.Decimal{}, errInvalidConversion
}

// jsonToDateTime converts v of type JSON to DateTime.
func jsonToDateTime(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		t, err := iso8601.ParseString(v)
		if err == nil {
			return t.UTC(), nil
		}
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		var s any
		err := enc.Decode(&s)
		if err == nil {
			return jsonToDateTime(s)
		}
	}
	return time.Time{}, errInvalidConversion
}

// jsonToDate converts v of type JSON to Date.
func jsonToDate(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		t, err := time.Parse(time.DateOnly, v)
		if err == nil {
			return t.UTC(), nil
		}
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		var s any
		err := enc.Decode(&s)
		if err == nil {
			return jsonToDate(s)
		}
	}
	return time.Time{}, errInvalidConversion
}

// jsonToTime converts v of type JSON to Time.
func jsonToTime(v any) (time.Time, error) {
	switch v := v.(type) {
	case string:
		t, ok := parseTime(v)
		if ok {
			return t, nil
		}
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		var s any
		err := enc.Decode(&s)
		if err == nil {
			return jsonToTime(s)
		}
	}
	return time.Time{}, errInvalidConversion
}

// jsonToYear converts v of type JSON to Year.
func jsonToYear(v any) (int, error) {
	switch v := v.(type) {
	case float64:
		if 1 <= v && v <= 9999 {
			return int(math.Round(v)), nil
		}
	case json.Number:
		n, err := strconv.Atoi(string(v))
		if err == nil {
			return n, nil
		}
		f, err := strconv.ParseFloat(string(v), 64)
		if err == nil && 1 <= f && f <= 9999 {
			return int(math.Round(f)), nil
		}
	case json.RawMessage:
		return jsonToYear(json.Number(v))
	}
	return 0, errInvalidConversion
}

// jsonToText converts v of type JSON to Text.
func jsonToText(v any) (string, error) {
	switch v := v.(type) {
	case string:
		return v, nil
	case float64:
		return strconv.FormatFloat(v, 'g', -1, 64), nil
	case json.Number:
		return string(v), nil
	case bool:
		if v {
			return "true", nil
		}
		return "false", nil
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		enc.UseNumber()
		var s any
		err := enc.Decode(&s)
		if err == nil {
			return jsonToText(s)
		}
	}
	return "", errInvalidConversion
}

// jsonToUUID converts v of type JSON to UUID.
func jsonToUUID(v any) (string, error) {
	switch v := v.(type) {
	case string:
		if u, err := uuid.Parse(v); err == nil {
			return u.String(), nil
		}
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		var s string
		err := enc.Decode(&s)
		if err == nil {
			return jsonToUUID(s)
		}
	}
	return "", errInvalidConversion
}

// jsonToInet converts v of type JSON to Inet.
func jsonToInet(v any) (string, error) {
	switch v := v.(type) {
	case string:
		if ip, err := netip.ParseAddr(v); err == nil {
			return ip.String(), nil
		}
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		var s string
		err := enc.Decode(&s)
		if err == nil {
			return jsonToInet(s)
		}
	}
	return "", errInvalidConversion
}

// jsonToArray converts v of type JSON to Array.
func jsonToArray(v any) ([]any, error) {
	switch v := v.(type) {
	case []any:
		return v, nil
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		enc.UseNumber()
		var s []any
		err := enc.Decode(&s)
		if err == nil {
			return s, nil
		}
	}
	return nil, errInvalidConversion
}

// jsonToMap converts v of type JSON to Object/Map.
func jsonToMap(v any) (map[string]any, error) {
	switch v := v.(type) {
	case map[string]any:
		return v, nil
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		enc.UseNumber()
		var s map[string]any
		err := enc.Decode(&s)
		if err == nil {
			return s, nil
		}
	}
	return nil, errInvalidConversion
}

func decimalToInt(n decimal.Decimal) (int, error) {
	if !n.IsInteger() {
		return 0, errInvalidConversion
	}
	if n.LessThan(minIntDecimal) {
		return 0, errInvalidConversion
	}
	if n.GreaterThan(maxIntDecimal) {
		return 0, errInvalidConversion
	}
	return int(n.IntPart()), nil
}

func decimalToUInt(n decimal.Decimal) (uint, error) {
	if !n.IsInteger() || n.IsNegative() || n.GreaterThan(maxUIntDecimal) {
		return 0, errInvalidConversion
	}
	if n.LessThanOrEqual(maxIntDecimal) {
		return uint(n.IntPart()), nil
	}
	u, err := strconv.ParseUint(n.String(), 10, 64)
	if err != nil {
		return 0, errInvalidConversion
	}
	return uint(u), nil
}

func floatToInt(n float64) (int, error) {
	if math.IsNaN(n) || n < minFloatConvertibleToInt64 || n > maxFloatConvertibleToInt64 {
		return 0, errInvalidConversion
	}
	return int(math.Round(n)), nil
}

func floatToUInt(n float64) (uint, error) {
	if math.IsNaN(n) || n < 0 || n > maxFloatConvertibleToUInt64 {
		return 0, errInvalidConversion
	}
	return uint(math.Round(n)), nil
}

func parseUint(s string) int {
	var n int
	for _, c := range []byte(s) {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
		if n < 0 {
			return -1 // overflow
		}
	}
	return n
}

func isSimpleFloat(s string) bool {
	if len(s) < 3 {
		return false
	}
	var dot bool
	for i, c := range []byte(s) {
		if c == '.' {
			if dot || i == 0 || i == len(s)-1 {
				return false
			}
			dot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func convertTextToDate(s string) (t time.Time, ok bool) {
	month, day, year := -1, -1, -1
	if len(s) == 10 {
		if s[4] == '-' && s[7] == '-' {
			year, month, day = parseUint(s[0:4]), parseUint(s[5:7]), parseUint(s[8:10]) // yyyy-mm-dd
		} else if s[2] == '/' && s[5] == '/' || s[2] == '.' && s[5] == '.' {
			month, day, year = parseUint(s[0:2]), parseUint(s[3:5]), parseUint(s[6:10]) // mm/dd/yyyy, mm.dd.yyyy
		}
	} else if len(s) == 8 {
		if s[2] == '-' && s[5] == '-' {
			year, month, day = parseUint(s[0:2]), parseUint(s[3:5]), parseUint(s[5:8]) // yy-mm-dd
		} else if s[2] == '/' && s[5] == '/' || s[2] == '.' && s[5] == '.' {
			month, day, year = parseUint(s[0:2]), parseUint(s[3:5]), parseUint(s[6:10]) // mm/dd/yy, mm.dd.yy
		}
	} else if isSimpleFloat(s) {
		// Parse as Excel serial date-time.
		// https://support.microsoft.com/en-us/office/datevalue-function-df8b07d4-7761-4a93-bc33-b7471bbff252
		days, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return
		}
		if days == 60 {
			return // 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
		}
		if days > 60 {
			days--
		}
		t = excelEpoch.Add(time.Duration(days) * 24 * time.Hour)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), true
	}
	if year < 0 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
		return
	}
	t = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return
	}
	return t, true
}

// parseTime parses a time formatted as "hh:nn:ss.nnnnnnnnn" and returns it as
// the time on January 1, 1970 UTC. The sub-second part can contain from 1 to 9
// digits or can be missing. The hour must be in range [0, 23], minute and second
// must be in range [0, 59], and any trailing characters are discarded.
// The boolean return value indicates whether the time was successfully parsed.
//
// Keep in sync with the parseTime function in the normalization package.
func parseTime[bytes []byte | string](p bytes) (t time.Time, ok bool) {
	if len(p) < 8 {
		return
	}
	parse := func(n bytes) int {
		if n[0] < '0' || n[0] > '9' || n[1] < '0' || n[1] > '9' {
			return -1
		}
		return int(n[0]-'0')*10 + int(n[1]-'0')
	}
	h, m, s := parse(p[0:2]), parse(p[3:5]), parse(p[6:8])
	if h < 0 || h > 23 || p[2] != ':' || m < 0 || m > 59 || p[5] != ':' || s < 0 || s > 59 {
		return
	}
	p = p[8:]
	var ns int
	if len(p) > 0 && p[0] == '.' {
		p = p[1:]
		var i int
		for ; i < 9 && i < len(p) && '0' <= p[i] && p[i] <= '9'; i++ {
			ns = ns*10 + int(p[i]-'0')
		}
		if i == 0 {
			return
		}
		for ; i < 9; i++ {
			ns *= 10
		}
	}
	return time.Date(1970, 1, 1, h, m, s, ns, time.UTC), true
}
