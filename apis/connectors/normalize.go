//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package connectors

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

	"github.com/meergo/meergo/apis/errors"
	"github.com/meergo/meergo/apis/state"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
)

const (
	minIntRepresentableAsFloat64 = -9007199254740991
	maxIntRepresentableAsFloat64 = 9007199254740991

	minIntRepresentableAsFloat32 = -16777216
	maxIntRepresentableAsFloat32 = 16777216
)

var (
	minIntDecimal  = decimal.NewFromInt(math.MinInt64)
	maxIntDecimal  = decimal.NewFromInt(math.MaxInt64)
	maxUintDecimal = decimal.RequireFromString("18446744073709551615")
)

// normalizationError represents an error occurred normalizing a property. It
// implements the ValidationError interface of apis.
type normalizationError struct {
	path string
	msg  string
}

// newNormalizationErrorf returns a *normalizationError error based on a format
// specifier. The error message can report the invalid value and should complete
// the sentence "property foo ".
func newNormalizationErrorf(path string, format string, a ...any) error {
	return &normalizationError{
		path: path,
		msg:  fmt.Sprintf("property %q ", path) + fmt.Sprintf(format, a...),
	}
}

func (err *normalizationError) Error() string {
	return err.msg
}

func (err *normalizationError) PropertyPath() string {
	return err.path
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
		v, valid = asInt64(src)
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
			if src.IsInteger() && !src.IsNegative() && src.LessThanOrEqual(maxUintDecimal) {
				var err error
				v, err = strconv.ParseUint(src.String(), 10, 64)
				value = err != nil
			}
		case json.Number:
			var err error
			v, err = strconv.ParseUint(string(src), 10, 64)
			if err != nil {
				var f float64
				f, err = src.Float64()
				v = uint64(f)
			}
			valid = err == nil
		case string:
			var err error
			v, err = strconv.ParseUint(src, 10, 64)
			valid = err == nil
		case []byte:
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
			if valid && typ.BitSize() == 32 {
				valid = float64(float32(v)) == v
			}
		case json.Number:
			var err error
			v, err = strconv.ParseFloat(string(src), typ.BitSize())
			valid = err == nil
		case string:
			var err error
			v, err = strconv.ParseFloat(src, typ.BitSize())
			valid = err == nil
		case []byte:
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
		switch src := src.(type) {
		case int:
			v = decimal.NewFromInt(int64(src))
			valid = true
		case int8:
			v = decimal.NewFromInt(int64(src))
			valid = true
		case int16:
			v = decimal.NewFromInt(int64(src))
			valid = true
		case int32:
			v = decimal.NewFromInt(int64(src))
			valid = true
		case int64:
			v = decimal.NewFromInt(src)
			valid = true
		case uint:
			v = decimal.NewFromUint64(uint64(src))
			valid = true
		case uint8:
			v = decimal.NewFromUint64(uint64(src))
			valid = true
		case uint16:
			v = decimal.NewFromUint64(uint64(src))
			valid = true
		case uint32:
			v = decimal.NewFromUint64(uint64(src))
			valid = true
		case uint64:
			v = decimal.NewFromUint64(src)
			valid = true
		case float32:
			v = decimal.NewFromFloat32(src)
			valid = true
		case float64:
			v = decimal.NewFromFloat(src)
			valid = true
		case decimal.Decimal:
			v = src
			valid = true
		case json.Number:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		case string:
			var err error
			v, err = decimal.NewFromString(src)
			valid = err == nil
		case []byte:
			var err error
			v, err = decimal.NewFromString(string(src))
			valid = err == nil
		}
		if valid {
			min, max := typ.DecimalRange()
			if v.LessThan(min) || v.GreaterThan(max) {
				return nil, newNormalizationErrorf(name, "has a value %s that is not in range [%s, %s]", v, min, max)
			}
			value = v
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
		case json.Number:
			if n, err := src.Int64(); err == nil {
				t, valid = dateTimeFromUnixInt(n, layouts.DateTime)
			} else if f, err := src.Float64(); err == nil {
				t, valid = dateTimeFromUnixFloat(f, layouts.DateTime)
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
		v, valid = asInt64(src)
		value = int(v)
		valid = valid && types.MinYear <= v && v <= types.MaxYear
	case types.UUIDKind:
		if s, ok := src.(string); ok {
			value, valid = parseUUID(s)
		}
	case types.JSONKind:
		if !validJSON(src) {
			return nil, fmt.Errorf("app has returned an invalid JSON for property %q", name)
		}
		value = src
		valid = true
	case types.InetKind:
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
	case types.TextKind:
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
				return nil, fmt.Errorf("value of property %s does not contain valid UTF-8 characters: %q ",
					errors.Abbreviate(v, 20), name)
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
			// Snowflake only supports JSON as the item type. The driver returns the value as a JSON array.
			if s != "" && s[0] == '[' && typ.Elem().Kind() == types.JSONKind {
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
			var err error
			for _, p := range typ.Properties() {
				value, ok := src[p.Name]
				if !ok {
					return nil, fmt.Errorf(`there is not a value for the "%s.%s" property`, name, p.Name)
				}
				src[p.Name], err = normalize(name, p.Type, value, p.Nullable, layouts)
				if err != nil {
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
			// Snowflake only supports JSON as the value type. The driver returns the value as a JSON object.
			if s != "" && s[0] == '{' && typ.Elem().Kind() == types.JSONKind {
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
		return nil, newNormalizationErrorf(name, "has value %#v that cannot be represented as the %s type", src, typ)
	}
	return value, nil
}

func asInt64(v any) (int64, bool) {
	switch v := v.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), v <= math.MaxInt64
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		return int64(v), v <= math.MaxInt64
	case float32:
		f := float64(v)
		return int64(f), !math.IsInf(f, 0) && f == math.Trunc(f)
	case float64:
		return int64(v), !math.IsInf(v, 0) && v == math.Trunc(v)
	case decimal.Decimal:
		return v.IntPart(), v.IsInteger() && v.GreaterThanOrEqual(minIntDecimal) && v.LessThanOrEqual(maxIntDecimal)
	case json.Number:
		value, err := v.Int64()
		if err != nil {
			if d, err := decimal.NewFromString(string(v)); err == nil {
				return d.IntPart(), d.IsInteger() && d.GreaterThanOrEqual(minIntDecimal) && d.LessThanOrEqual(maxIntDecimal)
			}
		}
		return value, err == nil
	case string:
		value, err := strconv.ParseInt(v, 10, 64)
		return value, err == nil
	case []byte:
		value, err := strconv.ParseInt(string(v), 10, 64)
		return value, err == nil
	}
	return 0, false
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

// parseUUID parses s as a UUID in the standard form xxxx-xxxx-xxxx-xxxxxxxxxxxx
// and returns it in the canonical form without uppercase letters. The boolean
// return value reports whether s is a UUID in the standard form.
func parseUUID(s string) (string, bool) {
	if len(s) != 36 {
		return "", false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return "", false
	}
	return id.String(), true
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
			if v != nil {
				if ok := validJSON(v); !ok {
					return false
				}
			}
		}
		return true
	case map[string]any:
		for _, v := range src {
			if v != nil {
				if ok := validJSON(v); !ok {
					return false
				}
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
