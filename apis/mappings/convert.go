//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package mappings

import (
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
	"github.com/shopspring/decimal"
)

var excelEpoch = time.Date(1899, 12, 31, 0, 0, 0, 0, time.UTC)

var errInvalidConversion = errors.New("cannot convert")

// convert converts v from type t1 to type t2 and returns the converted value.
// For Array, Object and Map values it can change the argument v.
// It returns an error if v cannot be converted.
// It panics if v is nil.
func convert(v any, t1, t2 types.Type) (any, error) {
	pt1 := t1.PhysicalType()
	pt2 := t2.PhysicalType()
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
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var n int
		switch pt1 {
		case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
			n = v.(int)
		case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
			u := v.(uint)
			if u > math.MaxInt64 {
				return nil, errInvalidConversion
			}
			n = int(u)
		case types.PtFloat, types.PtFloat32:
			f := v.(float64)
			switch {
			case math.IsNaN(f):
				return nil, errInvalidConversion
			case math.IsInf(f, 0):
				min, max := t2.IntRange()
				if f < 0 {
					n = int(min)
				} else {
					n = int(max)
				}
			default:
				n = int(f)
			}
		case types.PtDecimal:
			n = int(v.(decimal.Decimal).Round(0).IntPart())
		case types.PtYear:
			n = v.(int)
		case types.PtText:
			var err error
			n, err = strconv.Atoi(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if min, max := t2.IntRange(); int64(n) < min || int64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
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
			f := v.(float64)
			switch {
			case math.IsNaN(f):
				return nil, errInvalidConversion
			case math.IsInf(f, 0):
				min, max := t2.IntRange()
				if f < 0 {
					n = uint(min)
				} else {
					n = uint(max)
				}
			default:
				n = uint(f)
			}
		case types.PtDecimal:
			i := v.(decimal.Decimal).Round(0).IntPart()
			if i < 0 {
				return nil, errInvalidConversion
			}
			n = uint(i)
		case types.PtYear:
			n = uint(v.(int))
		case types.PtText:
			u, err := strconv.ParseUint(v.(string), 10, 64)
			if err != nil {
				return nil, errInvalidConversion
			}
			n = uint(u)
		default:
			return nil, errInvalidConversion
		}
		if min, max := t2.UIntRange(); uint64(n) < min || uint64(n) > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtFloat, types.PtFloat32:
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
			bitSize := 64
			if pt2 == types.PtFloat32 {
				bitSize = 32
			}
			var err error
			n, err = strconv.ParseFloat(v.(string), bitSize)
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if min, max := t2.FloatRange(); n < min || n > max {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtDecimal:
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
			if math.IsNaN(f) {
				return nil, errInvalidConversion
			}
			if math.IsInf(f, 0) {
				neg := f < 0
				f = math.MaxFloat64
				if pt1 == types.PtFloat32 {
					f = math.MaxFloat32
				}
				if neg {
					f = -f
				}
			}
			n = decimal.NewFromFloat(f)
		case types.PtText:
			var err error
			n, err = decimal.NewFromString(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
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
			t, err := time.Parse(time.DateTime, v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return t.UTC(), nil
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
		}
	case types.PtTime:
		switch pt1 {
		case types.PtTime:
			return v.(time.Time), nil
		case types.PtDateTime:
			t := v.(time.Time)
			return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
		case types.PtText:
			t, ok := parseTime(v.(string))
			if !ok {
				return nil, errInvalidConversion
			}
			return t, nil
		}
	case types.PtYear:
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
			if len(s) == 0 || s[0] < '0' || s[1] < '9' {
				return nil, errInvalidConversion
			}
			var err error
			n, err = strconv.Atoi(s)
			if err != nil {
				return nil, errInvalidConversion
			}
		default:
			return nil, errInvalidConversion
		}
		if n < 1 || n > 9999 {
			return nil, errInvalidConversion
		}
		return n, nil
	case types.PtUUID:
		switch pt1 {
		case types.PtUUID:
			return v.(string), nil
		case types.PtText:
			u, err := uuid.Parse(v.(string))
			if err != nil {
				return nil, errInvalidConversion
			}
			return u, nil
		}
	case types.PtJSON:
		if pt1 == types.PtText {
			s := json.RawMessage(v.(string))
			if !json.Valid(s) {
				return nil, errInvalidConversion
			}
			if l, ok := t2.CharLen(); ok && utf8.RuneCount(s) > l {
				return nil, errInvalidConversion
			}
			return s, nil
		}
		s, err := json.Marshal(v)
		if err != nil {
			return nil, errInvalidConversion
		}
		if l, ok := t2.CharLen(); ok && utf8.RuneCount(s) > l {
			return nil, errInvalidConversion
		}
		return s, nil
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
			bitSize := 64
			if pt1 == types.PtFloat32 {
				bitSize = 32
			}
			s = strconv.FormatFloat(v.(float64), 'e', -1, bitSize)
		case types.PtDecimal:
			s = v.(decimal.Decimal).String()
		case types.PtDateTime:
			s = v.(time.Time).Format(time.DateTime)
		case types.PtDate:
			s = v.(time.Time).Format(time.DateOnly)
		case types.PtTime:
			s = v.(time.Time).Format("15:04:05.999999999")
		case types.PtYear:
			s = strconv.Itoa(v.(int))
		case types.PtUUID, types.PtInet:
			s = v.(string)
		case types.PtJSON:
			s = string(v.(json.RawMessage))
		default:
			return nil, errInvalidConversion
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
		if pt1 != types.PtArray {
			return nil, errInvalidConversion
		}
		s := v.([]any)
		if len(s) < t2.MinItems() {
			return nil, errInvalidConversion
		}
		it1 := t1.ItemType()
		it2 := t2.ItemType()
		if it1.EqualTo(it2) {
			return s, nil
		}
		var err error
		for i, item := range s {
			s[i], err = convert(item, it1, it2)
			if err != nil {
				return nil, err
			}
		}
		return s, nil
	case types.PtObject:
		if pt1 != types.PtObject {
			return nil, errInvalidConversion
		}
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
			obj[name], err = convert(value, p1.Type, p2.Type)
			if err != nil {
				return nil, err
			}
		}
		return obj, nil
	case types.PtMap:
		if pt1 != types.PtMap {
			return nil, errInvalidConversion
		}
		vt1 := t1.ValueType()
		vt2 := t2.ValueType()
		m := v.(map[string]any)
		if vt1.EqualTo(vt2) {
			return m, nil
		}
		var err error
		for key, value := range m {
			m[key], err = convert(value, vt1, vt2)
			if err != nil {
				return nil, err
			}
		}
		return m, nil
	}
	return nil, errInvalidConversion
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
