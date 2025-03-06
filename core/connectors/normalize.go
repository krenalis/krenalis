//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

import (
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

	"github.com/meergo/meergo/core/errors"
	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/core/util"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/relvacode/iso8601"
)

const (
	minIntRepresentableAsFloat64 = -9007199254740991
	maxIntRepresentableAsFloat64 = 9007199254740991

	minIntRepresentableAsFloat32 = -16777216
	maxIntRepresentableAsFloat32 = 16777216
)

// normalizationError represents an error occurred normalizing a property. It
// implements the ValidationError interface of core.
type normalizationError struct {
	path string
	msg  string
}

// newNormalizationErrorf returns a *normalizationError error based on a format
// specifier. The error message can report the invalid value and should complete
// the sentence "property foo ".
func newNormalizationErrorf(path string, format string, a ...any) *normalizationError {
	return &normalizationError{
		path: path,
		msg:  fmt.Sprintf(format, a...),
	}
}

func (err *normalizationError) Error() string {
	return fmt.Sprintf("property %q ", err.path) + err.msg
}

func (err *normalizationError) PropertyPath() string {
	return err.path
}

func (err *normalizationError) prependPath(path string) {
	err.path = path + "." + err.path
}

// normalize normalizes a property value, and returns its normalized value. If
// the value is not valid it returns an error.
func normalize(name string, typ types.Type, src any, nullable bool, layouts *state.TimeLayouts) (any, error) {
	if src == nil {
		if !nullable {
			return nil, newNormalizationErrorf(name, "has value null but it is not nullable")
		}
		return nil, nil
	}
	var value any
	var valid bool
	switch k := typ.Kind(); k {
	case types.BooleanKind:
		switch src.(type) {
		case bool:
			value = src
			valid = true
		case string:
			switch src {
			case "true":
				value = true
				valid = true
			case "false":
				value = false
				valid = true
			}
		}
	case types.IntKind:
		var v int64
		// Keep in sync with the case 'types.YearKind'.
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
		case uint:
			v = int64(src)
			valid = src <= math.MaxInt64
		case uint8:
			v = int64(src)
			valid = true
		case uint16:
			v = int64(src)
			valid = true
		case uint32:
			v = int64(src)
			valid = true
		case uint64:
			v = int64(src)
			valid = src <= math.MaxInt64
		case float32:
			f := float64(src)
			v = int64(f)
			valid = !math.IsInf(f, 0) && f == math.Trunc(f)
		case float64:
			v = int64(src)
			valid = !math.IsInf(src, 0) && src == math.Trunc(src)
		case decimal.Decimal:
			i, err := src.Int64()
			v = i
			valid = err == nil
		case string:
			value, err := strconv.ParseInt(src, 10, 64)
			v = value
			valid = err == nil
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			value, err = strconv.ParseInt(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.IntRange()
			if v < min || v > max {
				return nil, newNormalizationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = int(v)
		}
	case types.UintKind:
		var v uint64
		switch src := src.(type) {
		case int:
			if src >= 0 {
				v = uint64(src)
				valid = true
			}
		case int8:
			v = uint64(src)
			valid = true
		case int16:
			v = uint64(src)
			valid = true
		case int32:
			v = uint64(src)
			valid = true
		case int64:
			if src >= 0 {
				v = uint64(src)
				valid = true
			}
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
		case float32:
			f := float64(src)
			if src >= 0 && !math.IsInf(f, 1) && f == math.Trunc(f) {
				v = uint64(src)
				valid = true
			}
		case float64:
			if src >= 0 && !math.IsInf(src, 1) && src == math.Trunc(src) {
				v = uint64(src)
				valid = true
			}
		case decimal.Decimal:
			var err error
			v, err = src.Uint64()
			valid = err == nil
		case string:
			var err error
			v, err = strconv.ParseUint(src, 10, 64)
			valid = err == nil
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			valid = err == nil
		}
		if valid {
			min, max := typ.UintRange()
			if v < min || v > max {
				return nil, newNormalizationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = uint(v)
		}
	case types.FloatKind:
		var v float64
		switch src := src.(type) {
		case int:
			min, max := minIntRepresentableAsFloat64, maxIntRepresentableAsFloat64
			if typ.BitSize() == 32 {
				min, max = minIntRepresentableAsFloat32, maxIntRepresentableAsFloat32
			}
			if min <= src && src <= max {
				v = float64(src)
				valid = true
			}
		case int8:
			v = float64(src)
			valid = true
		case int16:
			v = float64(src)
			valid = true
		case int32:
			v = float64(src)
			valid = true
		case int64:
			min, max := int64(minIntRepresentableAsFloat64), int64(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				min, max = minIntRepresentableAsFloat32, maxIntRepresentableAsFloat32
			}
			if min <= src && src <= max {
				v = float64(src)
				valid = true
			}
		case uint:
			max := uint(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				max = uint(maxIntRepresentableAsFloat32)
			}
			if src <= max {
				v = float64(src)
				valid = true
			}
		case uint8:
			v = float64(src)
			valid = true
		case uint16:
			v = float64(src)
			valid = true
		case uint32:
			v = float64(src)
			valid = true
		case uint64:
			max := uint64(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				max = uint64(maxIntRepresentableAsFloat32)
			}
			if src <= max {
				v = float64(src)
				valid = true
			}
		case float32:
			v = float64(src)
			valid = true
		case float64:
			if typ.BitSize() == 32 && !math.IsNaN(src) {
				valid = float64(float32(src)) == src
			} else {
				valid = true
			}
			v = src
		case decimal.Decimal:
			v, valid = src.Float64()
		case string:
			var err error
			v, err = strconv.ParseFloat(src, typ.BitSize())
			valid = err == nil
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			v, err = strconv.ParseFloat(string(src), typ.BitSize())
			valid = err == nil
		}
		if valid {
			if math.IsNaN(v) {
				if typ.IsReal() {
					return nil, newNormalizationErrorf(name, "has a value of NaN, which is not allowed")
				}
			} else {
				min, max := typ.FloatRange()
				if v < min || v > max {
					return nil, newNormalizationErrorf(name, "has a value %f that is not in the range [%f, %f]", v, min, max)
				}
			}
			value = v
		}
	case types.DecimalKind:
		var v decimal.Decimal
		p, s := typ.Precision(), typ.Scale()
		var err error
		switch src := src.(type) {
		case int:
			v, err = decimal.Int(src, p, s)
		case int8:
			v, err = decimal.Int(int(src), p, s)
		case int16:
			v, err = decimal.Int(int(src), p, s)
		case int32:
			v, err = decimal.Int(int(src), p, s)
		case int64:
			v, err = decimal.Int(int(src), p, s)
		case uint:
			v, err = decimal.Uint(src, p, s)
		case uint8:
			v, err = decimal.Uint(uint(src), p, s)
		case uint16:
			v, err = decimal.Uint(uint(src), p, s)
		case uint32:
			v, err = decimal.Uint(uint(src), p, s)
		case uint64:
			v, err = decimal.Uint(uint(src), p, s)
		case decimal.Decimal:
			v, err = decimal.Parse(src.String(), p, s)
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			v, err = decimal.Parse(src, p, s)
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = decimal.Parse(src, p, s)
		case fmt.Stringer:
			v, err = decimal.Parse(src.String(), p, s)
		}
		if err == nil {
			min, max := typ.DecimalRange()
			if v.Less(min) || v.Greater(max) {
				return nil, newNormalizationErrorf(name, "has a value %s that is not in range [%s, %s]", v, min, max)
			}
			value = v
			valid = true
		}
	case types.DateTimeKind:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = src
			valid = true
		case float64:
			t, valid = dateTimeFromUnixFloat(src, layouts.DateTime)
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			switch layouts.DateTime {
			case "":
				var err error
				t, err = iso8601.ParseString(src)
				valid = err == nil
			case "unix", "unixmilli", "unixmicro", "unixnano":
				n, err := strconv.ParseInt(src, 10, 64)
				if err == nil {
					t, valid = dateTimeFromUnixInt(n, layouts.DateTime)
				}
			default:
				var err error
				t, err = time.Parse(layouts.DateTime, src)
				valid = err == nil
			}
		}
		if valid {
			t = t.UTC()
			if y := t.Year(); y < types.MinYear || y > types.MaxYear {
				return nil, newNormalizationErrorf(name, "has date and time %q with a year not in range [1, 9999]", src)
			}
			value = t
		}
	case types.DateKind:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = src
			valid = true
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			var err error
			if layouts.Date == "" {
				t, err = iso8601.ParseString(src)
			} else {
				t, err = time.Parse(layouts.Date, src)
			}
			valid = err == nil
		}
		if valid {
			t = t.UTC()
			if y := t.Year(); y < types.MinYear || y > types.MaxYear {
				return nil, newNormalizationErrorf(name, "has date %q with a year not in range [1, 9999]", src)
			}
			value = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		}
	case types.TimeKind:
		switch src := src.(type) {
		case time.Time:
			value = time.Date(1970, 1, 1, src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), time.UTC)
			valid = true
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			var t time.Time
			var err error
			if layouts.Time == "" {
				t, err = iso8601.ParseString(src)
			} else {
				t, err = time.Parse(layouts.Time, src)
			}
			valid = err == nil
			if valid {
				value = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var t time.Time
			var err error
			if layouts.Time == "" {
				t, err = iso8601.Parse(src)
			} else {
				t, err = time.Parse(layouts.Time, string(src))
			}
			valid = err == nil
			if valid {
				value = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
			}
		}
	case types.YearKind:
		var v int64
		// Keep in sync with the case 'types.IntKind'.
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
		case uint:
			v = int64(src)
			valid = src <= math.MaxInt64
		case uint8:
			v = int64(src)
			valid = true
		case uint16:
			v = int64(src)
			valid = true
		case uint32:
			v = int64(src)
			valid = true
		case uint64:
			v = int64(src)
			valid = src <= math.MaxInt64
		case float32:
			f := float64(src)
			v = int64(f)
			valid = !math.IsInf(f, 0) && f == math.Trunc(f)
		case float64:
			v = int64(src)
			valid = !math.IsInf(src, 0) && src == math.Trunc(src)
		case decimal.Decimal:
			i, err := src.Int64()
			v = i
			valid = err == nil
		case string:
			value, err := strconv.ParseInt(src, 10, 64)
			v = value
			valid = err == nil
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			value, err = strconv.ParseInt(string(src), 10, 64)
			valid = err == nil
		}
		value = int(v)
		valid = valid && types.MinYear <= v && v <= types.MaxYear
	case types.UUIDKind:
		switch src := src.(type) {
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			value, valid = util.ParseUUID(src)
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			value, valid = util.UUIDFromBytes(src)
		}
	case types.JSONKind:
		var data []byte
		switch src := src.(type) {
		case json.Value:
			if src == nil && nullable {
				return nil, nil
			}
			data = src
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			data = src
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			data = []byte(src)
		case json.Marshaler:
			var err error
			data, err = src.MarshalJSON()
			if err != nil {
				return nil, newNormalizationErrorf(name, "cannot be unmarshalled; MarshalJSON returned an error: %s", err)
			}
		}
		if data != nil {
			if !json.Valid(data) {
				return nil, newNormalizationErrorf(name, "is not valid JSON")
			}
			value = json.Value(data)
			valid = true
		}
	case types.InetKind:
		switch ip := src.(type) {
		case string:
			if ip == "" && nullable {
				return nil, nil
			}
			// Remove the number of bits in the netmask, if any.
			if i := strings.IndexByte(ip, '/'); i > 0 {
				ip = ip[:i]
			}
			src, _ = netip.ParseAddr(ip)
		case net.IP:
			if ip == nil && nullable {
				return nil, nil
			}
			if addr, ok := netip.AddrFromSlice(ip); ok {
				// Unmap an IPv6-mapped IPv4 address as the net.IP.String method does.
				if addr.Is4In6() {
					addr = addr.Unmap()
				}
				src = addr
			}
		}
		if addr, ok := src.(netip.Addr); ok && addr.IsValid() {
			value = addr.WithZone("").String()
			valid = true
		}
	case types.TextKind:
		var v string
		switch s := src.(type) {
		case string:
			if s == "" {
				if values := typ.Values(); values != nil && !slices.Contains(values, "") {
					return nil, nil
				}
			}
			v = s
			valid = true
		case []byte:
			if s == nil && nullable {
				return nil, nil
			}
			v = string(s)
			valid = true
		}
		if valid {
			if !utf8.ValidString(v) {
				return nil, newNormalizationErrorf(name, "does not contain valid UTF-8 characters")
			}
			if values := typ.Values(); values != nil {
				if !slices.Contains(values, v) {
					return nil, newNormalizationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else if rx := typ.Regexp(); rx != nil {
				if !rx.MatchString(v) {
					return nil, newNormalizationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else {
				if l, ok := typ.ByteLen(); ok && len(v) > l {
					return nil, newNormalizationErrorf(name, "has value %q that is longer than %d bytes", errors.Abbreviate(v, 20), l)
				}
				if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
					return nil, newNormalizationErrorf(name, "has value %q that is longer than %d characters", errors.Abbreviate(v, 20), l)
				}
			}
			value = v
		}
	case types.ArrayKind:
		if s, ok := src.(string); ok {
			// Snowflake only supports json as the item type. The driver returns the value as a JSON array.
			if s != "" && s[0] == '[' && typ.Elem().Kind() == types.JSONKind {
				v := json.Value(s)
				if !json.Valid(v) {
					return nil, newNormalizationErrorf(name, "is not valid JSON")
				}
				min := typ.MinElements()
				max := typ.MaxElements()
				arr := []any{}
				for i, element := range v.Elements() {
					if i == max {
						return nil, newNormalizationErrorf(name, "is an array with more than %d elements; they must be in range [%d, %d]", max, min, max)
					}
					arr = append(arr, element)
				}
				if len(arr) < min {
					return nil, newNormalizationErrorf(name, "is an array with less than %d elements; they must be in range [%d, %d]", min, min, max)
				}
				value = arr
				valid = true
			}
		} else {
			rv := reflect.ValueOf(src)
			if rv.Kind() == reflect.Slice {
				if rv.IsNil() && nullable {
					return nil, nil
				}
				var err error
				n := rv.Len()
				if n < typ.MinElements() || n > typ.MaxElements() {
					return nil, newNormalizationErrorf(name, "is an array with %d elements, but they must be in range [%d, %d]", n, typ.MinElements(), typ.MaxElements())
				}
				a := make([]any, n)
				t := typ.Elem()
				for i := 0; i < n; i++ {
					v := rv.Index(i).Interface()
					a[i], err = normalize(name, t, v, false, layouts)
					if err != nil {
						return nil, err
					}
				}
				if typ.Unique() {
					for i, e := range a {
						for _, e2 := range a[i:] {
							if e == e2 {
								return nil, newNormalizationErrorf(name, "contains the duplicated value %v", e)
							}
						}
					}
				}
				value = a
				valid = true
			}
		}
	case types.ObjectKind:
		if src, ok := src.(map[string]any); ok {
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			for _, p := range typ.Properties() {
				value, ok := src[p.Name]
				if !ok {
					if !p.ReadOptional {
						err := newNormalizationErrorf(p.Name, "does not have a value, but the property is not optional for reading")
						err.prependPath(name)
						return nil, err
					}
					continue
				}
				src[p.Name], err = normalize(p.Name, p.Type, value, p.Nullable, layouts)
				if err != nil {
					if err, ok := err.(*normalizationError); ok {
						err.prependPath(name)
					}
					return nil, err
				}
			}
			if len(src) != types.NumProperties(typ) {
			SRC:
				for name := range src {
					for _, p := range typ.Properties() {
						if p.Name == name {
							continue SRC
						}
					}
					delete(src, name)
				}
			}
			value = src
			valid = true
		}
	case types.MapKind:
		if s, ok := src.(string); ok {
			// Snowflake only supports json as the value type. The driver returns the value as a JSON object.
			if s != "" && s[0] == '{' && typ.Elem().Kind() == types.JSONKind {
				v := json.Value(s)
				if json.Valid(v) {
					m := map[string]any{}
					for k, v := range v.Properties() {
						m[k] = v
					}
					value = m
					valid = true
				}
			}
		} else {
			rv := reflect.ValueOf(src)
			if rv.Kind() == reflect.Map {
				if rv.IsNil() && nullable {
					return nil, nil
				}
				var err error
				n := rv.Len()
				m := make(map[string]any, n)
				t := typ.Elem()
				iter := rv.MapRange()
				for iter.Next() {
					k := iter.Key().String()
					v := iter.Value().Interface()
					m[k], err = normalize(name, t, v, false, layouts)
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
		return nil, newNormalizationErrorf(name, "has value %#v and type %T that cannot be represented as the %s type", src, src, typ)
	}
	return value, nil
}

// dateTimeFromUnixInt returns the local Time corresponding to the provided Unix
// time. Unix time is expressed in seconds, milliseconds, microseconds or
// nanoseconds according to layout.
// The second return value reports whether the layout is appropriate.
func dateTimeFromUnixInt(n int64, layout string) (time.Time, bool) {
	switch layout {
	case "unix":
		return time.Unix(n, 0), true
	case "unixmilli":
		return time.UnixMilli(n), true
	case "unixmicro":
		return time.UnixMicro(n), true
	case "unixnano":
		return time.Unix(0, n), true
	}
	return time.Time{}, false
}

// dateTimeFromUnixFloat returns the local Time corresponding to the provided
// Unix time. Unix time is expressed in seconds, milliseconds, microseconds or
// nanoseconds according to layout.
// The second return value reports whether the layout is appropriate.
func dateTimeFromUnixFloat(n float64, layout string) (time.Time, bool) {
	switch layout {
	case "unix":
		sec := int64(n)
		nsec := int64((n - float64(sec)) * 1e9)
		return time.Unix(sec, nsec), true
	case "unixmilli":
		return time.UnixMilli(int64(n)), true
	case "unixmicro":
		return time.UnixMicro(int64(n)), true
	case "unixnano":
		return time.Unix(0, int64(n)), true
	}
	return time.Time{}, false
}
