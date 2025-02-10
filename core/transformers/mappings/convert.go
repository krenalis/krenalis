//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
	"errors"
	"fmt"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
)

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

var errInvalidConversion = errors.New("cannot convert")

const (
	// Range of float values convertible to int(64): [-9223372036854776000.0, 9223372036854775000.0].
	// These are the same limits that PostgreSQL uses when converting a double precision value to a bigint.
	minFloatConvertibleToInt64 = -9223372036854776000.0 // converted to -9223372036854775808
	maxFloatConvertibleToInt64 = 9223372036854775000.0  // converted to 9223372036854774784

	// Range of float values convertible to uint(64): [0, 18446744073709550000].
	maxFloatConvertibleToUint64 = 18446744073709550000.0 // converted to 18446744073709549568
)

var (
	minIntDecimal  = decimal.MustInt(math.MinInt64)
	maxIntDecimal  = decimal.MustInt(math.MaxInt64)
	maxUintDecimal = decimal.MustUint(18446744073709551615)
)

// convert converts v from type st to type dt and returns the converted value.
// nullable reports whether nil is allowed as return value. If v is nil and
// nullable is true, it returns nil.
//
// If inPlace is true, the conversion is permitted to modify array, object, and
// map values directly within the value being converted.
//
// layouts represents, if not nil, the layouts used to format datetime, date,
// and time values as strings.
//
// purpose specifies the reason for the transformation. If Create or Update,
// then all the properties required for creation or the update must be present
// in the returned value.
func convert(v any, st, dt types.Type, nullable, inPlace bool, layouts *state.TimeLayouts, purpose Purpose) (any, error) {
	spt := st.Kind()
	dpt := dt.Kind()
	if nullable {
		switch {
		case v == nil:
			return nil, nil
		case spt == types.JSONKind && dpt != types.JSONKind:
			if v := v.(json.Value); v.IsNull() {
				return nil, nil
			}
		case v == "":
			if dpt != types.TextKind && dpt != types.JSONKind {
				return nil, nil
			}
		}
	} else if v == nil {
		if dpt == types.JSONKind {
			return json.Value("null"), nil
		}
		return nil, errInvalidConversion
	} else if spt == types.JSONKind && dpt != types.JSONKind {
		if v := v.(json.Value); v.IsNull() {
			return nil, errInvalidConversion
		}
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
			v := v.(json.Value)
			if v.IsBool() {
				return v.Bool(), nil
			}
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
			v := v.(json.Value)
			n, err = v.Int()
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
			v := v.(json.Value)
			n, err = v.Uint()
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
			if dt.IsReal() && !st.IsReal() && (math.IsNaN(n) || math.IsInf(n, 0)) {
				return nil, errInvalidConversion
			}
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
			v := v.(json.Value)
			n, err = v.Float(dt.BitSize())
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
			n, err = decimal.Int(v.(int), dt.Precision(), dt.Scale())
		case types.UintKind:
			n, err = decimal.Uint(v.(uint), dt.Precision(), dt.Scale())
		case types.FloatKind:
			f := v.(float64)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return nil, errInvalidConversion
			}
			n, err = decimal.Float64(f, dt.Precision(), dt.Scale())
		case types.TextKind:
			n, err = decimal.Parse(v.(string), dt.Precision(), dt.Scale())
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Decimal(dt.Precision(), dt.Scale())
		default:
			return nil, errInvalidConversion
		}
		if err != nil {
			return nil, errInvalidConversion
		}
		if min, max := dt.DecimalRange(); n.Less(min) || n.Greater(max) {
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
			v := v.(json.Value)
			if !v.IsString() {
				return nil, errInvalidConversion
			}
			t, err = iso8601.Parse(v.Bytes())
			if err != nil {
				return nil, errInvalidConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
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
			v := v.(json.Value)
			if !v.IsString() {
				return nil, errInvalidConversion
			}
			t, err = time.Parse(time.DateOnly, v.String())
			if err != nil {
				return nil, errInvalidConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, errInvalidConversion
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
		switch spt {
		case types.TimeKind:
			t = v.(time.Time)
		case types.DateTimeKind:
			t = v.(time.Time)
			t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		case types.TextKind:
			var ok bool
			t, ok = util.ParseTime(v.(string))
			if !ok {
				return nil, errInvalidConversion
			}
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return nil, errInvalidConversion
			}
			var ok bool
			t, ok = util.ParseTime(v.Bytes())
			if !ok {
				return nil, errInvalidConversion
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
			v := v.(json.Value)
			n, err = v.Int()
			if err != nil {
				return nil, errInvalidConversion
			}
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
			v := v.(json.Value)
			u, err := uuid.ParseBytes(v.Bytes())
			if err != nil {
				return nil, errInvalidConversion
			}
			return u.String(), nil
		}
	case types.JSONKind:
		if spt == types.JSONKind {
			return v, nil
		}
		// TODO(marco): time types are not correctly marshaled
		if encodeSorted {
			var b json.Buffer
			err := b.EncodeSorted(v)
			if err != nil {
				return nil, errInvalidConversion
			}
			return b.Value()
		}
		value, err := json.Marshal(v)
		if err != nil {
			return nil, errInvalidConversion
		}
		return value, nil
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
			v := v.(json.Value)
			if !v.IsString() {
				return nil, errInvalidConversion
			}
			ip, err := netip.ParseAddr(v.String())
			if err != nil {
				return nil, errInvalidConversion
			}
			return ip.String(), nil
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
			v := v.(json.Value)
			switch v.Kind() {
			case json.Array, json.Object:
				return nil, errInvalidConversion
			}
			s = v.String()
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
		case types.JSONKind:
			s := v.(json.Value)
			if s.Kind() == json.Object {
				return nil, errInvalidConversion
			}
			et := dt.Elem()
			min := dt.MinElements()
			max := dt.MaxElements()
			if s.Kind() != json.Array {
				if min > 1 || max == 0 {
					return nil, errInvalidConversion
				}
				elem, err := convert(s, types.JSON(), et, false, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
				return []any{elem}, nil
			}
			d := make([]any, 0)
			for i, elem := range s.Elements() {
				if i == max {
					return nil, errInvalidConversion
				}
				e, err := convert(elem, types.JSON(), et, false, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
				d = append(d, e)
			}
			if len(d) < min {
				return nil, errInvalidConversion
			}
			if dt.Unique() {
				for i, it := range d {
					for _, it2 := range d[i:] {
						if it == it2 {
							return nil, errInvalidConversion
						}
					}
				}
			}
			return d, nil
		case types.ArrayKind:
			s := v.([]any)
			d := s
			if len(s) < dt.MinElements() || len(s) > dt.MaxElements() {
				return nil, errInvalidConversion
			}
			it1 := st.Elem()
			it2 := dt.Elem()
			if !types.Equal(it1, it2) {
				if !inPlace {
					d = make([]any, len(s))
				}
				var err error
				for i, item := range s {
					d[i], err = convert(item, it1, it2, false, inPlace, layouts, purpose)
					if err != nil {
						return nil, err
					}
				}
			}
			if !st.Unique() && dt.Unique() {
				for i, item := range d {
					for _, item2 := range d[i:] {
						if item == item2 {
							return nil, errInvalidConversion
						}
					}
				}
			}
			return d, nil
		}
	case types.ObjectKind:
		var d map[string]any
		switch spt {
		case types.ObjectKind:
			if types.Equal(st, dt) {
				return v, nil
			}
			s := v.(map[string]any)
			d = s
			if !inPlace {
				d = make(map[string]any)
			}
			var err error
			for name, value := range s {
				dp, ok := dt.Property(name)
				if !ok {
					if inPlace {
						delete(d, name)
					}
					continue
				}
				if value == nil {
					if !dp.Nullable {
						return nil, errInvalidConversion
					}
					if !inPlace {
						d[name] = nil
					}
					continue
				}
				sp, ok := st.Property(name)
				if !ok {
					panic(fmt.Sprintf("unknown property %s", name))
				}
				d[name], err = convert(value, sp.Type, dp.Type, dp.Nullable, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
			}
		case types.MapKind:
			s := v.(map[string]any)
			d := s
			if !inPlace {
				d = make(map[string]any)
			}
			vt := st.Elem()
			var err error
			for _, p := range dt.Properties() {
				if value, ok := s[p.Name]; ok {
					d[p.Name], err = convert(value, vt, p.Type, p.Nullable, inPlace, layouts, purpose)
					if err != nil {
						return nil, err
					}
					continue
				}
				if purpose == Create && p.CreateRequired || purpose == Update && p.UpdateRequired {
					return nil, errInvalidConversion
				}
			}
			return d, nil
		case types.JSONKind:
			v := v.(json.Value)
			if v.Kind() != json.Object {
				return nil, errInvalidConversion
			}
			var err error
			d = make(map[string]any)
			for name, value := range v.Properties() {
				if !types.IsValidPropertyName(name) {
					continue
				}
				p, ok := dt.Property(name)
				if !ok {
					continue
				}
				d[name], err = convert(value, types.JSON(), p.Type, p.Nullable, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
			}
		}
		switch purpose {
		case Create:
			for _, p := range dt.Properties() {
				if !p.CreateRequired {
					continue
				}
				if _, ok := d[p.Name]; !ok {
					return nil, errInvalidConversion
				}
			}
		case Update:
			for _, p := range dt.Properties() {
				if !p.UpdateRequired {
					continue
				}
				if _, ok := d[p.Name]; !ok {
					return nil, errInvalidConversion
				}
			}
		}
		return d, nil
	case types.MapKind:
		switch spt {
		case types.MapKind:
			vt1 := st.Elem()
			vt2 := dt.Elem()
			if types.Equal(vt1, vt2) {
				return v, nil
			}
			s := v.(map[string]any)
			d := s
			if !inPlace {
				d = make(map[string]any, len(s))
			}
			var err error
			for key, value := range s {
				d[key], err = convert(value, vt1, vt2, false, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
			}
			return d, nil
		case types.ObjectKind:
			s := v.(map[string]any)
			d := s
			if !inPlace {
				d = make(map[string]any, len(s))
			}
			vt := dt.Elem()
			var err error
			for _, p := range st.Properties() {
				if value, ok := s[p.Name]; ok {
					d[p.Name], err = convert(value, p.Type, vt, true, inPlace, layouts, purpose)
					if err != nil {
						return nil, err
					}
				}
			}
			return d, nil
		case types.JSONKind:
			s := v.(json.Value)
			if s.Kind() != json.Object {
				return nil, errInvalidConversion
			}
			vt := dt.Elem()
			d := make(map[string]any)
			var err error
			for name, value := range s.Properties() {
				d[name], err = convert(value, types.JSON(), vt, false, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
			}
			return d, nil
		}
	}
	return nil, errInvalidConversion
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

func decimalToInt(n decimal.Decimal) (int, error) {
	i, err := n.Int64()
	if err != nil {
		return 0, errInvalidConversion
	}
	return int(i), nil
}

func decimalToUint(n decimal.Decimal) (uint, error) {
	i, err := n.Uint64()
	if err != nil {
		return 0, errInvalidConversion
	}
	return uint(i), nil
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
