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
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
)

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

var errInvalidConversion = errors.New("cannot convert")

const (
	// Range of Float values convertible to Int(64): [-9223372036854776000.0, 9223372036854775000.0].
	// These are the same limits that PostgreSQL uses when converting a double precision value to a bigint.
	minFloatConvertibleToInt64 = -9223372036854776000.0 // converted to -9223372036854775808
	maxFloatConvertibleToInt64 = 9223372036854775000.0  // converted to 9223372036854774784

	// Range of Float values convertible to Uint(64): [0, 18446744073709550000].
	maxFloatConvertibleToUint64 = 18446744073709550000.0 // converted to 18446744073709549568
)

var (
	minIntDecimal  = decimal.NewFromInt(math.MinInt64)
	maxIntDecimal  = decimal.NewFromInt(math.MaxInt64)
	maxUintDecimal = decimal.RequireFromString("18446744073709551615")
)

// convert converts v from type st to type dt and returns the converted value.
// nullable reports whether nil is allowed as return value. If v is nil and
// nullable is true, it returns nil.
//
// layouts represents, if not null, the layouts used to format DateTime, Date,
// and Time values as strings.
//
// For Array, Object, and Map values, it can modify the argument v. It returns
// an error if v cannot be converted.
func convert(v any, st, dt types.Type, nullable bool, layouts *state.TimeLayouts) (any, error) {
	spt := st.Kind()
	dpt := dt.Kind()
	// Convert between nil and other values.
	if nullable {
		switch {
		case v == nil:
			return nil, nil
		case spt == types.JSONKind && dpt != types.JSONKind:
			if v, ok := v.(json.RawMessage); ok && v[0] == 'n' {
				return nil, nil
			}
		case v == "" && dpt != types.TextKind:
			return nil, nil
		}
	} else if v == nil {
		switch dpt {
		case types.TextKind:
			return "", nil
		case types.JSONKind:
			return json.RawMessage("null"), nil
		}
		return nil, errInvalidConversion
	}
	// Convert the unparsed cases, v is not nil.
	switch dpt {
	case types.BooleanKind:
		switch spt {
		case types.BooleanKind:
			return v.(bool), nil
		case types.IntKind:
			if st.BitSize() == 8 {
				return v.(int) != 0, nil
			}
		case types.UintKind:
			if st.BitSize() == 8 {
				return v.(uint) > 0, nil
			}
		case types.TextKind:
			switch v.(string) {
			case "false", "False", "FALSE", "no", "No", "NO":
				return false, nil
			case "true", "True", "TRUE", "yes", "Yes", "YES":
				return true, nil
			}
		case types.JSONKind:
			return jsonToBoolean(v)
		}
	case types.IntKind:
		var err error
		var n int
		switch spt {
		case types.BooleanKind:
			if v.(bool) {
				n = 1
			}
			if dt.BitSize() != 8 {
				err = errInvalidConversion
			}
		case types.IntKind:
			n = v.(int)
		case types.UintKind:
			u := v.(uint)
			if u > math.MaxInt64 {
				err = errInvalidConversion
			}
			n = int(u)
		case types.FloatKind:
			n, err = floatToInt(v.(float64))
		case types.DecimalKind:
			n, err = decimalToInt(v.(decimal.Decimal))
		case types.YearKind:
			n = v.(int)
		case types.TextKind:
			n, err = strconv.Atoi(v.(string))
		case types.JSONKind:
			n, err = jsonToInt(v)
		default:
			err = errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := dt.IntRange(); int64(n) < min || int64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.UintKind:
		var err error
		var n uint
		switch spt {
		case types.BooleanKind:
			if v.(bool) {
				n = 1
			}
			if dt.BitSize() != 8 {
				err = errInvalidConversion
			}
		case types.IntKind:
			i := v.(int)
			if i < 0 {
				return nil, errInvalidConversion
			}
			n = uint(i)
		case types.UintKind:
			n = v.(uint)
		case types.FloatKind:
			n, err = floatToUint(v.(float64))
		case types.DecimalKind:
			n, err = decimalToUint(v.(decimal.Decimal))
		case types.YearKind:
			n = uint(v.(int))
		case types.TextKind:
			var u uint64
			u, err = strconv.ParseUint(v.(string), 10, 64)
			n = uint(u)
		case types.JSONKind:
			n, err = jsonToUint(v)
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		min, max := dt.UintRange()
		if uint64(n) < min || uint64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.FloatKind:
		var err error
		var n float64
		switch spt {
		case types.FloatKind:
			n = v.(float64)
			if dt.BitSize() == 32 && st.BitSize() != 32 {
				n = float64(float32(n))
			}
		case types.IntKind:
			n = float64(v.(int))
		case types.UintKind:
			n = float64(v.(uint))
		case types.DecimalKind:
			n, _ = v.(decimal.Decimal).Float64()
			if dt.BitSize() == 32 {
				n = float64(float32(n))
			}
		case types.TextKind:
			n, err = strconv.ParseFloat(v.(string), dt.BitSize())
		case types.JSONKind:
			n, err = jsonToFloat(v, dt.BitSize())
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := dt.FloatRange(); n < min || n > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.DecimalKind:
		var err error
		var n decimal.Decimal
		switch spt {
		case types.DecimalKind:
			n, _ = v.(decimal.Decimal)
		case types.IntKind:
			n = decimal.New(int64(v.(int)), 0)
		case types.UintKind:
			n, _ = decimal.NewFromString(strconv.FormatUint(uint64(v.(uint)), 10))
		case types.FloatKind:
			f := v.(float64)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, errInvalidConversion
			}
			n = decimal.NewFromFloat(f)
		case types.TextKind:
			n, err = decimal.NewFromString(v.(string))
		case types.JSONKind:
			n, err = jsonToDecimal(v)
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := dt.DecimalRange(); n.LessThan(min) || n.GreaterThan(max) {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.DateTimeKind:
		var t time.Time
		var err error
		switch spt {
		case types.DateTimeKind, types.DateKind:
			t = v.(time.Time)
		case types.TextKind:
			t, err = time.Parse(time.RFC3339Nano, v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, errInvalidConversion
			}
		case types.JSONKind:
			t, err = jsonToDateTime(v)
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if layouts != nil {
			switch layouts.DateTime {
			case "unix":
				return t.Unix(), nil
			case "unixmilli":
				return t.UnixMilli(), nil
			case "unixmicro":
				return t.UnixMicro(), nil
			case "unixnano":
				return t.UnixNano(), nil
			default:
				layout := layouts.DateTime
				if layout == "" {
					layout = "2006-01-02T15:04:05.999Z"
				}
				return t.Format(layout), nil
			}
		}
		return t, nil
	case types.DateKind:
		var t time.Time
		var err error
		switch spt {
		case types.DateKind:
			t = v.(time.Time)
		case types.DateTimeKind:
			t = v.(time.Time)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		case types.TextKind:
			t, err = convertTextToDate(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
		case types.JSONKind:
			t, err = jsonToDate(v)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errInvalidConversion
		}
		if layouts != nil {
			layout := layouts.Date
			if layout == "" {
				layout = "2006-01-02"
			}
			return t.Format(layout), nil
		}
		return t, nil
	case types.TimeKind:
		var t time.Time
		var err error
		switch spt {
		case types.TimeKind:
			t = v.(time.Time)
		case types.DateTimeKind:
			t = v.(time.Time)
			t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		case types.TextKind:
			var ok bool
			t, ok = parseTime(v.(string))
			if !ok {
				return nil, errInvalidConversion
			}
		case types.JSONKind:
			t, err = jsonToTime(v)
			if err != nil {
				return nil, err
			}
		}
		if layouts != nil {
			layout := layouts.Time
			if layout == "" {
				layout = "15:04:05.999Z"
			}
			return t.Format(layout), nil
		}
		return t, nil
	case types.YearKind:
		var err error
		var n int
		switch spt {
		case types.YearKind:
			return v.(int), nil
		case types.IntKind:
			n = v.(int)
		case types.UintKind:
			u := v.(uint)
			if u > math.MaxInt64 {
				return nil, errInvalidConversion
			}
			n = int(u)
		case types.TextKind:
			s := v.(string)
			if l := len(s); l == 0 || l > 4 || s[0] == '+' || s[0] == '-' || s[0] == '0' {
				return nil, errInvalidConversion
			}
			n, err = strconv.Atoi(s)
		case types.JSONKind:
			n, err = jsonToYear(v)
		default:
			return nil, errInvalidConversion
		}
		if err == nil && 1 <= n && n <= 9999 {
			return n, nil
		}
	case types.UUIDKind:
		switch spt {
		case types.UUIDKind:
			return v.(string), nil
		case types.TextKind:
			u, err := uuid.Parse(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return u.String(), nil
		case types.JSONKind:
			return jsonToUUID(v)
		}
	case types.JSONKind:
		switch v := v.(type) {
		case json.RawMessage:
			return v, nil
		case decimal.Decimal:
			return json.RawMessage(v.String()), nil
		default:
			if v == "" {
				return json.RawMessage("null"), nil
			}
			b, err := json.Marshal(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			return json.RawMessage(b), nil
		}
	case types.InetKind:
		switch spt {
		case types.InetKind:
			return v.(string), nil
		case types.TextKind:
			ip, err := netip.ParseAddr(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return ip.String(), nil
		case types.JSONKind:
			return jsonToInet(v)
		}
	case types.TextKind:
		var s string
		switch spt {
		case types.TextKind:
			s = v.(string)
		case types.BooleanKind:
			s = "false"
			if v.(bool) {
				s = "true"
			}
		case types.IntKind:
			s = strconv.FormatInt(int64(v.(int)), 10)
		case types.UintKind:
			s = strconv.FormatUint(uint64(v.(uint)), 10)
		case types.FloatKind:
			s = strconv.FormatFloat(v.(float64), 'g', -1, st.BitSize())
		case types.DecimalKind:
			s = v.(decimal.Decimal).String()
		case types.DateTimeKind:
			s = v.(time.Time).Format(time.RFC3339Nano)
		case types.DateKind:
			s = v.(time.Time).Format(time.DateOnly)
		case types.TimeKind:
			s = v.(time.Time).Format("15:04:05.999999999")
		case types.YearKind:
			s = strconv.Itoa(v.(int))
		case types.UUIDKind, types.InetKind:
			s = v.(string)
		case types.JSONKind:
			var err error
			s, err = jsonToText(v)
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if values := dt.Values(); values != nil {
			if s == "" && nullable {
				return nil, nil
			}
			if slices.Contains(values, s) {
				return s, nil
			}
			return nil, errInvalidConversion
		} else if re := dt.Regexp(); re != nil {
			if !re.MatchString(s) {
				if s == "" && nullable {
					return nil, nil
				}
				return nil, errInvalidConversion
			}
		} else {
			if l, ok := dt.ByteLen(); ok && l < len(s) {
				return nil, errInvalidConversion
			}
			if l, ok := dt.CharLen(); ok {
				runes := len(s)
				if spt == types.JSONKind || spt == types.TextKind {
					runes = utf8.RuneCountInString(s)
				}
				if runes > l {
					return nil, errInvalidConversion
				}
			}
		}
		return s, nil
	case types.ArrayKind:
		switch spt {
		default:
			if dt.MinItems() > 1 {
				return nil, errInvalidConversion
			}
			it := dt.Elem()
			if !types.Equal(st, it) {
				var err error
				v, err = convert(v, st, it, false, layouts)
				if err != nil {
					return nil, err
				}
			}
			return []any{v}, nil
		case types.JSONKind:
			s, err := jsonToArray(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			if len(s) < dt.MinItems() || len(s) > dt.MaxItems() {
				return nil, errInvalidConversion
			}
			it2 := dt.Elem()
			for i, item := range s {
				s[i], err = convert(item, types.JSON(), it2, false, layouts)
				if err != nil {
					return nil, err
				}
			}
			if dt.Unique() {
				for i, item := range s {
					for _, item2 := range s[i:] {
						if item == item2 {
							return nil, errInvalidConversion
						}
					}
				}
			}
			return s, nil
		case types.ArrayKind:
			s := v.([]any)
			if len(s) < dt.MinItems() || len(s) > dt.MaxItems() {
				return nil, errInvalidConversion
			}
			it1 := st.Elem()
			it2 := dt.Elem()
			if !types.Equal(it1, it2) {
				var err error
				for i, item := range s {
					s[i], err = convert(item, it1, it2, false, layouts)
					if err != nil {
						return nil, err
					}
				}
			}
			if !st.Unique() && dt.Unique() {
				for i, item := range s {
					for _, item2 := range s[i:] {
						if item == item2 {
							return nil, errInvalidConversion
						}
					}
				}
			}
			return s, nil
		case types.ObjectKind, types.MapKind:

		}
	case types.ObjectKind:
		switch spt {
		case types.ObjectKind:
			obj := v.(map[string]any)
			if types.Equal(st, dt) {
				return obj, nil
			}
			for name, value := range obj {
				p2, ok := dt.Property(name)
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
				p1, ok := st.Property(name)
				if !ok {
					panic(fmt.Sprintf("unknown property %s", name))
				}
				obj[name], err = convert(value, p1.Type, p2.Type, p2.Nullable, layouts)
				if err != nil {
					return nil, err
				}
			}
			return obj, nil
		case types.JSONKind:
			s, err := jsonToMap(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			for name, value := range s {
				p2, ok := dt.Property(name)
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
				s[name], err = convert(value, types.JSON(), p2.Type, p2.Nullable, layouts)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		}
	case types.MapKind:
		switch spt {
		case types.MapKind:
			vt1 := st.Elem()
			vt2 := dt.Elem()
			m := v.(map[string]any)
			if types.Equal(vt1, vt2) {
				return m, nil
			}
			var err error
			for key, value := range m {
				m[key], err = convert(value, vt1, vt2, false, layouts)
				if err != nil {
					return nil, err
				}
			}
			return m, nil
		case types.JSONKind:
			s, err := jsonToMap(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			vt2 := dt.Elem()
			for key, value := range s {
				s[key], err = convert(value, types.JSON(), vt2, false, layouts)
				if err != nil {
					return nil, err
				}
			}
			return s, nil
		}
	}
	return nil, errInvalidConversion
}

// appendAsString appends v to b after converting it to a string.
// Calling appendAsString(b, v, t) is the same of calling
// convert(v, t, types.Text(), false, false) and appending the result to b.
func appendAsString(b []byte, v any, t types.Type) ([]byte, error) {
	if v == nil {
		return b, nil
	}
	if s, ok := v.(string); ok {
		return append(b, s...), nil
	}
	switch t.Kind() {
	case types.BooleanKind:
		strconv.AppendBool(b, v.(bool))
	case types.IntKind, types.YearKind:
		return strconv.AppendInt(b, int64(v.(int)), 10), nil
	case types.UintKind:
		return strconv.AppendUint(b, uint64(v.(uint)), 10), nil
	case types.FloatKind:
		return strconv.AppendFloat(b, v.(float64), 'g', -1, t.BitSize()), nil
	case types.DecimalKind:
		return append(b, v.(decimal.Decimal).String()...), nil
	case types.DateTimeKind:
		return v.(time.Time).AppendFormat(b, time.RFC3339Nano), nil
	case types.DateKind:
		return v.(time.Time).AppendFormat(b, time.DateOnly), nil
	case types.TimeKind:
		return v.(time.Time).AppendFormat(b, "15:04:05.999999999"), nil
	case types.JSONKind:
		switch v := v.(type) {
		case float64:
			return strconv.AppendFloat(b, v, 'g', -1, 64), nil
		case json.Number:
			return append(b, v...), nil
		case bool:
			strconv.AppendBool(b, v)
		case json.RawMessage:
			s, err := jsonToText(v)
			if err == nil {
				return append(b, s...), nil
			}
		}
	}
	return b, errInvalidConversion
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

// jsonToUint converts v of type JSON to Uint.
func jsonToUint(v any) (uint, error) {
	switch v := v.(type) {
	case float64:
		if v < 0 || v > maxFloatConvertibleToUint64 {
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
		return jsonToUint(json.Number(v))
	}
	return 0, errInvalidConversion
}

// jsonToFloat converts v of type JSON to Float with the provided bit size that
// can be 32 or 64.
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
			t = t.UTC()
			if y := t.Year(); 1 <= y && y <= 9999 {
				return t, nil
			}
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
			t = t.UTC()
			if y := t.Year(); 1 <= y && y <= 9999 {
				return t, nil
			}
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
			if s == nil {
				return "", nil
			}
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
	case bool, string, float64, json.Number:
		return []any{v}, nil
	case []any:
		return v, nil
	case json.RawMessage:
		enc := json.NewDecoder(bytes.NewReader(v))
		enc.UseNumber()
		var s any
		err := enc.Decode(&s)
		if err == nil {
			return jsonToArray(s)
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

func decimalToUint(n decimal.Decimal) (uint, error) {
	if !n.IsInteger() || n.IsNegative() || n.GreaterThan(maxUintDecimal) {
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

func floatToUint(n float64) (uint, error) {
	if math.IsNaN(n) || n < 0 || n > maxFloatConvertibleToUint64 {
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
	// NOTE: keep in sync with the function within 'apis/connectors'.
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

func convertTextToDate(s string) (time.Time, error) {
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
			return time.Time{}, errInvalidConversion
		}
		if days == 60 {
			// 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
			return time.Time{}, errInvalidConversion
		}
		if days > 60 {
			days--
		}
		t := excelEpoch.Add(time.Duration(days) * 24 * time.Hour)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	if year < 0 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, errInvalidConversion
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, errInvalidConversion
	}
	return t, nil
}

// parseTime parses a time formatted as "hh:nn:ss.nnnnnnnnn" and returns it as
// the time on January 1, 1970 UTC. The sub-second part can contain from 1 to 9
// digits or can be missing. The hour must be in range [0, 23], minute and second
// must be in range [0, 59], and any trailing characters are discarded.
// The boolean return value indicates whether the time was successfully parsed.
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
