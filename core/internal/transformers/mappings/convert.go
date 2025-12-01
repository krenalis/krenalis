// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

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

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/core/internal/util"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
)

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

var (
	errMaxByteLengthConversion = errors.New("invalid max byte length")
	errMaxLengthConversion     = errors.New("invalid max length")
	errEnumConversion          = errors.New("not a valid enum value")
	errInvalidConversion       = errors.New("cannot convert")
	errMaxConversion           = errors.New("too large")
	errMinConversion           = errors.New("too small")
	errParseConversion         = errors.New("cannot parse")
	errRangeConversion         = errors.New("out of range")
	errPatternConversion       = errors.New("pattern mismatch")
	errYearRangeConversion     = errors.New("year not in range [1,9999]")
)

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
//
// If the value cannot be converted, it returns v and one of the following:
//   - errMaxByteLengthConversion
//   - errMaxLengthConversion
//   - errEnumConversion
//   - errInvalidConversion
//   - errMaxConversion
//   - errMinConversion
//   - errParseConversion
//   - errRangeConversion
//   - errPatternConversion
//   - errYearRangeConversion
func convert(v any, st, dt types.Type, nullable, inPlace bool, layouts *state.TimeLayouts, purpose Purpose) (any, error) {
	sk := st.Kind()
	dk := dt.Kind()
	if nullable {
		switch {
		case v == nil:
			return nil, nil
		case sk == types.JSONKind && dk != types.JSONKind:
			if v := v.(json.Value); v.IsNull() {
				return nil, nil
			}
		case v == "":
			if dk != types.StringKind && dk != types.JSONKind {
				return nil, nil
			}
		}
	} else if v == nil {
		if dk == types.JSONKind {
			return json.Value("null"), nil
		}
		return v, errInvalidConversion
	} else if sk == types.JSONKind && dk != types.JSONKind {
		if v := v.(json.Value); v.IsNull() {
			return v, errInvalidConversion
		}
	}
	// Convert the unparsed cases, v is not nil.
	switch dk {
	case types.StringKind:
		var s string
		switch sk {
		case types.StringKind:
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
		case types.UUIDKind, types.IPKind:
			s = v.(string)
		case types.JSONKind:
			v := v.(json.Value)
			switch v.Kind() {
			case json.Array, json.Object:
				return v, errInvalidConversion
			}
			s = v.String()
		default:
			return v, errInvalidConversion
		}
		if values := dt.Values(); values != nil {
			if s == "" && nullable {
				return nil, nil
			}
			if slices.Contains(values, s) {
				return s, nil
			}
			return v, errEnumConversion
		} else if re := dt.Pattern(); re != nil {
			if !re.MatchString(s) {
				if s == "" && nullable {
					return nil, nil
				}
				return v, errPatternConversion
			}
		} else {
			if l, ok := dt.MaxByteLength(); ok && l < len(s) {
				return v, errMaxByteLengthConversion
			}
			if l, ok := dt.MaxLength(); ok {
				runes := len(s)
				if sk == types.JSONKind || sk == types.StringKind {
					runes = utf8.RuneCountInString(s)
				}
				if runes > l {
					return v, errMaxLengthConversion
				}
			}
		}
		return s, nil
	case types.BooleanKind:
		switch sk {
		case types.StringKind:
			switch v.(string) {
			case "false", "False", "FALSE", "no", "No", "NO":
				return false, nil
			case "true", "True", "TRUE", "yes", "Yes", "YES":
				return true, nil
			}
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
		case types.JSONKind:
			v := v.(json.Value)
			if v.IsBool() {
				return v.Bool(), nil
			}
		}
	case types.IntKind:
		var err error
		var n int
		switch sk {
		case types.StringKind:
			n, err = strconv.Atoi(v.(string))
			if err != nil {
				if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
					return v, errRangeConversion
				}
			}
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
				return v, errRangeConversion
			}
			n = int(u)
		case types.FloatKind:
			n, err = floatToInt(v.(float64))
			if err != nil {
				return v, err
			}
		case types.DecimalKind:
			n, err = decimalToInt(v.(decimal.Decimal))
			if err != nil {
				return v, err
			}
		case types.YearKind:
			n = v.(int)
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Int()
			if err == json.ErrRange {
				return v, errRangeConversion
			}
		default:
			err = errInvalidConversion
		}
		if err != nil {
			return v, errInvalidConversion
		}
		min, max := dt.IntRange()
		if int64(n) < min {
			return v, errMinConversion
		}
		if int64(n) > max {
			return v, errMaxConversion
		}
		return n, nil
	case types.UintKind:
		var err error
		var n uint
		switch sk {
		case types.StringKind:
			var u uint64
			u, err = strconv.ParseUint(v.(string), 10, 64)
			if err != nil {
				if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
					return v, errRangeConversion
				}
			}
			n = uint(u)
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
				return v, errRangeConversion
			}
			n = uint(i)
		case types.UintKind:
			n = v.(uint)
		case types.FloatKind:
			n, err = floatToUint(v.(float64))
			if err != nil {
				return v, err
			}
		case types.DecimalKind:
			n, err = decimalToUint(v.(decimal.Decimal))
			if err != nil {
				return v, err
			}
		case types.YearKind:
			n = uint(v.(int))
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Uint()
			if err == json.ErrRange {
				return v, errRangeConversion
			}
		default:
			return v, errInvalidConversion
		}
		if err != nil {
			return v, errInvalidConversion
		}
		min, max := dt.UintRange()
		if uint64(n) < min {
			return v, errMinConversion
		}
		if uint64(n) > max {
			return v, errMaxConversion
		}
		return n, nil
	case types.FloatKind:
		var err error
		var n float64
		switch sk {
		case types.StringKind:
			n, err = strconv.ParseFloat(v.(string), dt.BitSize())
			if err != nil {
				if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
					return v, errRangeConversion
				}
			}
		case types.FloatKind:
			n = v.(float64)
			if dt.IsReal() && !st.IsReal() && (math.IsNaN(n) || math.IsInf(n, 0)) {
				return v, errRangeConversion
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
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Float(dt.BitSize())
			if err == json.ErrRange {
				return v, errRangeConversion
			}
		default:
			return v, errInvalidConversion
		}
		if err != nil {
			return v, errInvalidConversion
		}
		min, max := dt.FloatRange()
		if n < min {
			return v, errMinConversion
		}
		if n > max {
			return v, errMaxConversion
		}
		return n, nil
	case types.DecimalKind:
		var err error
		var n decimal.Decimal
		switch sk {
		case types.StringKind:
			n, err = decimal.Parse(v.(string), dt.Precision(), dt.Scale())
		case types.DecimalKind:
			n, _ = v.(decimal.Decimal)
		case types.IntKind:
			n, err = decimal.Int(v.(int), dt.Precision(), dt.Scale())
			if err == decimal.ErrRange {
				return v, errRangeConversion
			}
		case types.UintKind:
			n, err = decimal.Uint(v.(uint), dt.Precision(), dt.Scale())
			if err == decimal.ErrRange {
				return v, errRangeConversion
			}
		case types.FloatKind:
			f := v.(float64)
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return v, errRangeConversion
			}
			n, err = decimal.Float64(f, dt.Precision(), dt.Scale())
			if err == decimal.ErrRange {
				return v, errRangeConversion
			}
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Decimal(dt.Precision(), dt.Scale())
			if err == json.ErrRange {
				return v, errRangeConversion
			}
		default:
			return v, errInvalidConversion
		}
		if err != nil {
			return v, errInvalidConversion
		}
		min, max := dt.DecimalRange()
		if n.Less(min) {
			return v, errMinConversion
		}
		if n.Greater(max) {
			return v, errMaxConversion
		}
		return n, nil
	case types.DateTimeKind:
		var t time.Time
		var err error
		switch sk {
		case types.StringKind:
			t, err = time.Parse(time.RFC3339Nano, v.(string))
			if err != nil {
				return v, errParseConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return v, errYearRangeConversion
			}
		case types.DateTimeKind, types.DateKind:
			t = v.(time.Time)
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return v, errInvalidConversion
			}
			t, err = iso8601.Parse(v.Bytes())
			if err != nil {
				return v, errParseConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return v, errYearRangeConversion
			}
		default:
			return v, errInvalidConversion
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
		switch sk {
		case types.StringKind:
			t, err = convertStringToDate(v.(string))
			if err != nil {
				return v, err
			}
		case types.DateKind:
			t = v.(time.Time)
		case types.DateTimeKind:
			t = v.(time.Time)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return v, errInvalidConversion
			}
			t, err = time.Parse(time.DateOnly, v.String())
			if err != nil {
				return v, errParseConversion
			}
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return v, errYearRangeConversion
			}
		default:
			return v, errInvalidConversion
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
		switch sk {
		case types.StringKind:
			var ok bool
			t, ok = util.ParseTime(v.(string))
			if !ok {
				return v, errParseConversion
			}
		case types.TimeKind:
			t = v.(time.Time)
		case types.DateTimeKind:
			t = v.(time.Time)
			t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return v, errInvalidConversion
			}
			var ok bool
			t, ok = util.ParseTime(v.Bytes())
			if !ok {
				return v, errParseConversion
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
		switch sk {
		case types.StringKind:
			s := v.(string)
			if l := len(s); l == 0 || l > 4 || s[0] == '+' || s[0] == '-' || s[0] == '0' {
				return v, errParseConversion
			}
			n, err = strconv.Atoi(s)
			if err != nil {
				if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
					return v, errYearRangeConversion
				}
				return v, errParseConversion
			}
		case types.YearKind:
			return v.(int), nil
		case types.IntKind:
			n = v.(int)
		case types.UintKind:
			u := v.(uint)
			if u > math.MaxInt64 {
				return v, errYearRangeConversion
			}
			n = int(u)
		case types.JSONKind:
			v := v.(json.Value)
			n, err = v.Int()
			if err != nil {
				if err == json.ErrRange {
					return v, errYearRangeConversion
				}
				return v, errInvalidConversion
			}
		default:
			return v, errInvalidConversion
		}
		if n < 1 || n > 9999 {
			return v, errYearRangeConversion
		}
		return n, nil
	case types.UUIDKind:
		switch sk {
		case types.StringKind:
			u, err := uuid.Parse(v.(string))
			if err != nil {
				return v, errParseConversion
			}
			return u.String(), nil
		case types.UUIDKind:
			return v.(string), nil
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return v, errInvalidConversion
			}
			u, err := uuid.ParseBytes(v.Bytes())
			if err != nil {
				return v, errParseConversion
			}
			return u.String(), nil
		}
	case types.JSONKind:
		if sk == types.JSONKind {
			return v, nil
		}
		// TODO(marco): time types are not correctly marshaled
		if encodeSorted {
			var b json.Buffer
			err := b.EncodeSorted(v)
			if err != nil {
				return v, errInvalidConversion
			}
			value, _ := b.Value()
			return value, nil
		}
		value, err := json.Marshal(v)
		if err != nil {
			return v, errInvalidConversion
		}
		return value, nil
	case types.IPKind:
		switch sk {
		case types.StringKind:
			ip, err := netip.ParseAddr(v.(string))
			if err != nil {
				return v, errParseConversion
			}
			return ip.String(), nil
		case types.IPKind:
			return v.(string), nil
		case types.JSONKind:
			v := v.(json.Value)
			if !v.IsString() {
				return v, errInvalidConversion
			}
			ip, err := netip.ParseAddr(v.String())
			if err != nil {
				return v, errParseConversion
			}
			return ip.String(), nil
		}
	case types.ArrayKind:
		switch sk {
		case types.JSONKind:
			s := v.(json.Value)
			if s.Kind() == json.Object {
				return v, errInvalidConversion
			}
			et := dt.Elem()
			min := dt.MinElements()
			max := dt.MaxElements()
			if s.Kind() != json.Array {
				if min > 1 || max == 0 {
					return v, errInvalidConversion
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
					return v, errInvalidConversion
				}
				e, err := convert(elem, types.JSON(), et, false, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
				d = append(d, e)
			}
			if len(d) < min {
				return v, errInvalidConversion
			}
			if dt.Unique() {
				for i, it := range d {
					for _, it2 := range d[i:] {
						if it == it2 {
							return v, errInvalidConversion
						}
					}
				}
			}
			return d, nil
		case types.ArrayKind:
			s := v.([]any)
			d := s
			if len(s) < dt.MinElements() || len(s) > dt.MaxElements() {
				return v, errInvalidConversion
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
							return v, errInvalidConversion
						}
					}
				}
			}
			return d, nil
		}
	case types.ObjectKind:
		var d map[string]any
		dProperties := dt.Properties()
		switch sk {
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
			sProperties := st.Properties()
			for name, value := range s {
				dp, ok := dProperties.ByName(name)
				if !ok {
					if inPlace {
						delete(d, name)
					}
					continue
				}
				if value == nil {
					if !dp.Nullable {
						return v, errInvalidConversion
					}
					if !inPlace {
						d[name] = nil
					}
					continue
				}
				sp, ok := sProperties.ByName(name)
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
			for _, p := range dProperties.All() {
				if value, ok := s[p.Name]; ok {
					d[p.Name], err = convert(value, vt, p.Type, p.Nullable, inPlace, layouts, purpose)
					if err != nil {
						return nil, err
					}
					continue
				}
				if purpose == Create && p.CreateRequired || purpose == Update && p.UpdateRequired {
					return v, errInvalidConversion
				}
			}
			return d, nil
		case types.JSONKind:
			v := v.(json.Value)
			if v.Kind() != json.Object {
				return v, errInvalidConversion
			}
			var err error
			d = make(map[string]any)
			for name, value := range v.Properties() {
				if !types.IsValidPropertyName(name) {
					continue
				}
				p, ok := dProperties.ByName(name)
				if !ok {
					continue
				}
				d[name], err = convert(value, types.JSON(), p.Type, p.Nullable, inPlace, layouts, purpose)
				if err != nil {
					return nil, err
				}
			}
		default:
			return v, errInvalidConversion
		}
		switch purpose {
		case Create:
			for _, p := range dProperties.All() {
				if !p.CreateRequired {
					continue
				}
				if _, ok := d[p.Name]; !ok {
					return v, errInvalidConversion
				}
			}
		case Update:
			for _, p := range dProperties.All() {
				if !p.UpdateRequired {
					continue
				}
				if _, ok := d[p.Name]; !ok {
					return v, errInvalidConversion
				}
			}
		}
		return d, nil
	case types.MapKind:
		switch sk {
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
			for _, p := range st.Properties().All() {
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
				return v, errInvalidConversion
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
	return v, errInvalidConversion
}

func convertStringToDate(s string) (time.Time, error) {
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
			return time.Time{}, errParseConversion
		}
		if days == 60 {
			// 1900-02-29 does not exist. Excel returns it for compatibility with Lotus 1-2-3.
			return time.Time{}, errParseConversion
		}
		if days > 60 {
			days--
		}
		t := excelEpoch.Add(time.Duration(days) * 24 * time.Hour)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	}
	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, errParseConversion
	}
	if year < 1 || year > 9999 {
		return time.Time{}, errYearRangeConversion
	}
	t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if t.Year() != year || int(t.Month()) != month || t.Day() != day {
		return time.Time{}, errParseConversion
	}
	return t, nil
}

func decimalToInt(n decimal.Decimal) (int, error) {
	i, err := n.Int64()
	if err != nil {
		if err == decimal.ErrRange {
			return 0, errRangeConversion
		}
		return 0, errInvalidConversion
	}
	return int(i), nil
}

func decimalToUint(n decimal.Decimal) (uint, error) {
	i, err := n.Uint64()
	if err != nil {
		if err == decimal.ErrRange {
			return 0, errRangeConversion
		}
		return 0, errInvalidConversion
	}
	return uint(i), nil
}

func floatToInt(n float64) (int, error) {
	if math.IsNaN(n) || n < minFloatConvertibleToInt64 || n > maxFloatConvertibleToInt64 {
		return 0, errRangeConversion
	}
	return int(math.Round(n)), nil
}

func floatToUint(n float64) (uint, error) {
	if math.IsNaN(n) || n < 0 || n > maxFloatConvertibleToUint64 {
		return 0, errRangeConversion
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
