//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package normalization

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/connector/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	arrayType  = reflect.TypeOf(([]any)(nil))
	objectType = reflect.TypeOf((map[string]any)(nil))
	mapType    = objectType
)

// NormalizeAppProperty normalizes a property value returned by an app
// connector, and returns its normalized value. If the value is not valid
// it returns an error.
func NormalizeAppProperty(name string, typ types.Type, src any, nullable bool) (any, error) {
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
				return nil, fmt.Errorf("app returned a value of %d for property %s which is not within the expected range of [%d, %d]",
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
				return nil, fmt.Errorf("app returned a value of %d for property %s which is not within the expected range of [%d, %d]",
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
			if math.IsNaN(v) {
				if typ.IsReal() {
					return nil, fmt.Errorf("app returned NaN for property %s but its type does not allow it", name)
				}
			} else {
				min, max := typ.FloatRange()
				if v < min || v > max {
					return nil, fmt.Errorf("app returned a value of %f for property %s which is not within the expected range of [%f, %f]",
						v, name, min, max)
				}
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
				return nil, fmt.Errorf("app returned a value of %s for property %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDateTime:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = src
			valid = true
		case float64:
			t, valid = dateTimeFromUnixFloat(src, typ.Layout())
		case string:
			layout := typ.Layout()
			switch layout {
			case types.Seconds, types.Milliseconds, types.Microseconds, types.Nanoseconds:
				n, err := strconv.ParseInt(src, 10, 64)
				if err == nil {
					t, valid = dateTimeFromUnixInt(n, layout)
				}
			default:
				if layout == "" {
					layout = time.DateTime
				}
				var err error
				t, err = time.Parse(layout, src)
				valid = err == nil
			}
		case json.Number:
			if n, err := src.Int64(); err == nil {
				t, valid = dateTimeFromUnixInt(n, typ.Layout())
			} else if f, err := src.Float64(); err == nil {
				t, valid = dateTimeFromUnixFloat(f, typ.Layout())
			}
		}
		if valid {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, fmt.Errorf("app returned a value of %q for property %s, with year %d not in range [1, 9999]", src, name, y)
			}
			value = t
		}
	case types.PtDate:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = src
			valid = true
		case string:
			layout := typ.Layout()
			if layout == "" {
				layout = time.DateOnly
			}
			var err error
			t, err = time.Parse(layout, src)
			valid = err == nil
		}
		if valid {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, fmt.Errorf("app returned a value of %q for property %s, with year %d not in range [1, 9999]", src, name, y)
			}
			value = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	case types.PtTime:
		switch src := src.(type) {
		case time.Time:
			value = time.Date(1970, 1, 1, src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), time.UTC)
			valid = true
		case string:
			layout := typ.Layout()
			if layout == "" {
				value, valid = parseTime(src)
			} else {
				t, err := time.Parse(layout, src)
				if valid = err == nil; valid {
					value = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
				}
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
		if !validJSON(src) {
			return nil, fmt.Errorf("app returned an invalid JSON for property %s", name)
		}
		value = src
		valid = true
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
			if enum := typ.Enum(); enum != nil {
				if !slices.Contains(enum, v) {
					return nil, fmt.Errorf("app returned a value of %q for property %s, which is not valid", v, name)
				}
			} else if rx := typ.Regexp(); rx != nil {
				if !rx.MatchString(v) {
					return nil, fmt.Errorf("app returned a value of %q for property %s, which is not valid", v, name)
				}
			} else {
				if l, ok := typ.ByteLen(); ok && len(v) > l {
					return nil, fmt.Errorf("app returned a value of %q for property %s, which is longer than %d bytes",
						abbreviate(v, 20), name, l)
				}
				if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
					return nil, fmt.Errorf("app returned a value of %q for property %s, which is longer than %d characters",
						abbreviate(v, 20), name, l)
				}
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
			t := typ.Elem()
			for i := 0; i < n; i++ {
				v := rv.Index(i).Interface()
				a[i], err = NormalizeAppProperty(name, t, v, false)
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
					return nil, fmt.Errorf("app returned a non-existent property %s for object property %s", k, name)
				}
				obj[k], err = NormalizeAppProperty(name, p.Type, v, p.Nullable)
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
			t := typ.Elem()
			iter := rv.MapRange()
			for iter.Next() {
				k := iter.Key().String()
				v := iter.Value().Interface()
				m[k], err = NormalizeAppProperty(name, t, v, false)
				if err != nil {
					return nil, err
				}
			}
			value = m
			valid = true
		}
	}
	if !valid {
		return nil, fmt.Errorf("app returned a value of '%v' for property %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	return value, nil
}

// NormalizeDatabaseFileProperty normalizes a property value returned by a
// database connector, a file connector, or a data warehouse and returns its
// normalized value. If the value is not valid it returns an error.
func NormalizeDatabaseFileProperty(name string, typ types.Type, src any, nullable bool) (any, error) {
	if src == nil {
		if !nullable {
			return nil, fmt.Errorf("column %s is non-nullable, but the database returned a NULL value", name)
		}
		return nil, nil
	}
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
		case int:
			v = int64(src)
			valid = true
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
				return nil, fmt.Errorf("database returned a value of %d for column %s which is not within the expected range of [%d, %d]",
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
				return nil, fmt.Errorf("database returned a value of %d for column %s which is not within the expected range of [%d, %d]",
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
			if math.IsNaN(v) {
				if typ.IsReal() {
					return nil, fmt.Errorf("database returned NaN for property %s but its type does not allow it", name)
				}
			} else {
				min, max := typ.FloatRange()
				if v < min || v > max {
					return nil, fmt.Errorf("database returned a value of %f for column %s which is not within the expected range of [%f, %f]",
						v, name, min, max)
				}
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
				return nil, fmt.Errorf("database returned a value of %s for column %s which is not within the expected range of [%s, %s]",
					v, name, min, max)
			}
			value = v
		}
	case types.PtDateTime:
		if t, ok := src.(time.Time); ok {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, fmt.Errorf("database returned a value of %q for property %s, with year %d not in range [1, 9999]", src, name, y)
			}
			value = t
			valid = true
		}
	case types.PtDate:
		if t, ok := src.(time.Time); ok {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, fmt.Errorf("database returned a value of %q for property %s, with year %d not in range [1, 9999]", src, name, y)
			}
			value = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			valid = true
		}
	case types.PtTime:
		switch src := src.(type) {
		case time.Time:
			value = time.Date(1970, 1, 1, src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), time.UTC)
			valid = true
		case []byte:
			value, valid = parseTime(src)
		case string:
			value, valid = parseTime(src)
		}
	case types.PtYear:
		switch y := src.(type) {
		case int:
			if valid = types.MinYear <= y && y <= types.MaxYear; valid {
				value = y
			}
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
		if v, ok := src.([]byte); ok {
			src = json.RawMessage(v)
		}
		if !validJSON(src) {
			return nil, fmt.Errorf("database returned an invalid JSON for property %s", name)
		}
		value = src
		valid = true
	case types.PtInet:
		switch src := src.(type) {
		case string:
			if v, err := netip.ParseAddr(src); err == nil {
				value = v.String()
				valid = true
			}
		case net.IP:
			value = src.String()
			valid = true
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
			if enum := typ.Enum(); enum != nil {
				if !slices.Contains(enum, v) {
					return nil, fmt.Errorf("database returned a value of %q for property %s, which is not valid", v, name)
				}
			} else if rx := typ.Regexp(); rx != nil {
				if !rx.MatchString(v) {
					return nil, fmt.Errorf("database returned a value of %q for property %s, which is not valid", v, name)
				}
			} else {
				if l, ok := typ.ByteLen(); ok && len(v) > l {
					return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d bytes",
						abbreviate(v, 20), name, l)
				}
				if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
					return nil, fmt.Errorf("database returned a value of %q for column %s, which is longer than %d characters",
						abbreviate(v, 20), name, l)
				}
			}
			value = v
		}
	case types.PtArray:
		if s, ok := src.(string); ok {
			// Snowflake only supports JSON as the item type. The driver returns the value as a JSON array.
			if s != "" && s[0] == '[' && typ.Elem().PhysicalType() == types.PtJSON {
				dec := json.NewDecoder(strings.NewReader(s))
				dec.UseNumber()
				err := dec.Decode(&value)
				valid = err == nil
			}
		} else {
			rv := reflect.ValueOf(src)
			if rv.Kind() == reflect.Slice {
				var err error
				n := rv.Len()
				if n < typ.MinItems() || n > typ.MaxItems() {
					return nil, fmt.Errorf("database returned an array with %d items for property %s, which is not within the expected range of [%d, %d]",
						n, name, typ.MinItems(), typ.MaxItems())
				}
				a := make([]any, n)
				t := typ.Elem()
				for i := 0; i < n; i++ {
					v := rv.Index(i).Interface()
					a[i], err = NormalizeDatabaseFileProperty(name, t, v, false)
					if err != nil {
						return nil, err
					}
				}
				value = a
				valid = true
			}
		}
	case types.PtMap:
		if s, ok := src.(string); ok {
			// Snowflake only supports JSON as the value type. The driver returns the value as a JSON object.
			if s != "" && s[0] == '{' && typ.Elem().PhysicalType() == types.PtJSON {
				dec := json.NewDecoder(strings.NewReader(s))
				dec.UseNumber()
				err := dec.Decode(&value)
				valid = err == nil
			}
		} else {
			rv := reflect.ValueOf(src)
			if rv.Kind() == reflect.Map {
				var err error
				n := rv.Len()
				m := make(map[string]any, n)
				t := typ.Elem()
				iter := rv.MapRange()
				for iter.Next() {
					k := iter.Key().String()
					v := iter.Value().Interface()
					m[k], err = NormalizeDatabaseFileProperty(name, t, v, false)
					if err != nil {
						return nil, err
					}
				}
				value = m
				valid = true
			}
		}
	}
	if !valid {
		return nil, fmt.Errorf("database returned a value of '%v' for column %s, but it cannot be converted to the %s type",
			src, name, typ.PhysicalType())
	}
	return value, nil
}

// TODO(Gianluca): correctly implement this function; currently it just calls
// 'NormalizeAppProperty'.
func NormalizeTransformationProperty(name string, typ types.Type, src any, nullable, formatTime bool) (any, error) {
	return NormalizeAppProperty(name, typ, src, nullable)
}

// ValidateStringProperty validates a string property like
// NormalizeDatabaseFileProperty does.
func ValidateStringProperty(p types.Property, s string) error {
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

// dateTimeFromUnixInt returns the local Time corresponding to the given Unix
// time. Unix time is expressed in seconds, milliseconds, microseconds or
// nanoseconds according to layout.
// The second return value reports whether the layout is appropriate.
func dateTimeFromUnixInt(n int64, layout string) (time.Time, bool) {
	switch layout {
	case types.Seconds:
		return time.Unix(n, 0), true
	case types.Milliseconds:
		return time.UnixMilli(n), true
	case types.Microseconds:
		return time.UnixMicro(n), true
	case types.Nanoseconds:
		return time.Unix(0, n), true
	}
	return time.Time{}, false
}

// dateTimeFromUnixFloat returns the local Time corresponding to the given Unix
// time. Unix time is expressed in seconds, milliseconds, microseconds or
// nanoseconds according to layout.
// The second return value reports whether the layout is appropriate.
func dateTimeFromUnixFloat(n float64, layout string) (time.Time, bool) {
	switch layout {
	case types.Seconds:
		sec := int64(n)
		nsec := int64((n - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), true
	case types.Milliseconds:
		return time.UnixMilli(int64(n)), true
	case types.Microseconds:
		return time.UnixMicro(int64(n)), true
	case types.Nanoseconds:
		return time.Unix(0, int64(n)), true
	}
	return time.Time{}, false
}

// parseTime parses a time formatted as "hh:nn:ss.nnnnnnnnn" and returns it as
// the time on January 1, 1970 UTC. The sub-second part can contain from 1 to 9
// digits or can be missing. The hour must be in range [0, 23], minute and second
// must be in range [0, 59], and any trailing characters are discarded.
// The boolean return value indicates whether the time was successfully parsed.
//
// Keep in sync with the parseTime function in the mappings package.
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

// abbreviate abbreviates s to almost n runes. If s is longer than n runes,
// the abbreviated string terminates with "...".
func abbreviate(s string, n int) string {
	const spaces = " \n\r\t\f" // https://infra.spec.whatwg.org/#ascii-whitespace
	s = strings.TrimRight(s, spaces)
	if len(s) <= n {
		return s
	}
	if n < 3 {
		return ""
	}
	p := 0
	n2 := 0
	for i := range s {
		switch p {
		case n - 2:
			n2 = i
		case n:
			break
		}
		p++
	}
	if p < n {
		return s
	}
	if p = strings.LastIndexAny(s[:n2], spaces); p > 0 {
		s = strings.TrimRight(s[:p], spaces)
	} else {
		s = ""
	}
	if l := len(s) - 1; l >= 0 && (s[l] == '.' || s[l] == ',') {
		s = s[:l]
	}
	return s + "..."
}

// validJSON reports whether src is a valid JSON value as returned by a
// connector.
func validJSON(src any) bool {
	switch src := src.(type) {
	case string:
		return utf8.ValidString(src)
	case bool:
		return true
	case float64:
		return !math.IsNaN(src) && !math.IsInf(src, 0)
	case []any:
		for _, v := range src {
			if ok := validJSON(v); !ok {
				return false
			}
		}
		return true
	case map[string]any:
		for _, v := range src {
			if ok := validJSON(v); !ok {
				return false
			}
		}
		return true
	case json.Number:
		return src != "" && (src[0] == '-' || src[0] >= '0' && src[0] <= '9') && json.Valid([]byte(src))
	case json.RawMessage:
		return json.Valid(src)
	}
	return false
}
