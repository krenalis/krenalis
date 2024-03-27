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

	"github.com/open2b/chichi/apis/errors"
	"github.com/open2b/chichi/apis/state"
	"github.com/open2b/chichi/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
)

var (
	arrayType  = reflect.TypeOf(([]any)(nil))
	objectType = reflect.TypeOf((map[string]any)(nil))
	mapType    = objectType
)

// normalizeAppProperty normalizes a property value returned by an app
// connector, and returns its normalized value. If the value is not valid
// it returns an error.
func normalizeAppProperty(name string, typ types.Type, src any, nullable bool, layouts *state.Layouts) (any, error) {
	if src == nil {
		if !nullable {
			return nil, newValidationErrorf(name, "has value null but it is not nullable")
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
				return nil, newValidationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = int(v)
		}
	case types.UintKind:
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
			min, max := typ.UintRange()
			if v < min || v > max {

				return nil, newValidationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = uint(v)
		}
	case types.FloatKind:
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
					return nil, newValidationErrorf(name, "has a value of NaN, which is not allowed")
				}
			} else {
				min, max := typ.FloatRange()
				if v < min || v > max {
					return nil, newValidationErrorf(name, "has a value %f that is not in the range [%f, %f]", v, min, max)
				}
			}
			value = v
		}
	case types.DecimalKind:
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
				return nil, newValidationErrorf(name, "has a value %s that is not in range [%s, %s]", v, min, max)
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
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, newValidationErrorf(name, "has date and time %q with a year not in range [1, 9999]", src)
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
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, newValidationErrorf(name, "has date %q with a year not in range [1, 9999]", src)
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
		}
	case types.YearKind:
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
	case types.UUIDKind:
		if s, ok := src.(string); ok {
			if v, err := uuid.Parse(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.JSONKind:
		if !validJSON(src) {
			return nil, fmt.Errorf("app has returned an invalid JSON for property %q", name)
		}
		value = src
		valid = true
	case types.InetKind:
		if s, ok := src.(string); ok {
			if v, err := netip.ParseAddr(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.TextKind:
		var v string
		v, valid = src.(string)
		if valid {
			if !utf8.ValidString(v) {
				return nil, fmt.Errorf("app has returned a text with invalid UTF-8 characters for property %q", name)
			}
			if values := typ.Values(); values != nil {
				if !slices.Contains(values, v) {
					return nil, newValidationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else if rx := typ.Regexp(); rx != nil {
				if !rx.MatchString(v) {
					return nil, newValidationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else {
				if l, ok := typ.ByteLen(); ok && len(v) > l {
					return nil, newValidationErrorf(name, "has value %q that is longer than %d bytes", errors.Abbreviate(v, 20), l)
				}
				if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
					return nil, newValidationErrorf(name, "has value %q that is longer than %d characters", errors.Abbreviate(v, 20), l)
				}
			}
			value = v
		}
	case types.ArrayKind:
		rv := reflect.ValueOf(src)
		if rv.Type() == arrayType {
			var err error
			n := rv.Len()
			if n < typ.MinItems() || n > typ.MaxItems() {
				return nil, newValidationErrorf(name, "is an array with %d items, but they must be in range [%d, %d]", n, typ.MinItems(), typ.MaxItems())
			}
			a := make([]any, n)
			t := typ.Elem()
			for i := 0; i < n; i++ {
				v := rv.Index(i).Interface()
				a[i], err = normalizeAppProperty(name, t, v, false, layouts)
				if err != nil {
					return nil, err
				}
			}
			value = a
			valid = true
		}
	case types.ObjectKind:
		if src, ok := src.(map[string]any); ok {
			properties := typ.Properties()
			var err error
			for _, p := range properties {
				value, ok := src[p.Name]
				if !ok {
					return nil, fmt.Errorf(`app did not return a value for the "%s.%s" property`, name, p.Name)
				}
				src[p.Name], err = normalizeAppProperty(name, p.Type, value, p.Nullable, layouts)
				if err != nil {
					return nil, err
				}
			}
			if len(src) != len(properties) {
			SRC:
				for name := range src {
					for _, p := range properties {
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
				m[k], err = normalizeAppProperty(name, t, v, false, layouts)
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
			src, name, typ.Kind())
	}
	return value, nil
}

// normalizeDatabaseFileProperty normalizes a property value returned by a
// database connector or a file connector and returns its normalized value.
// If the value is not valid it returns an error.
func normalizeDatabaseFileProperty(name string, typ types.Type, src any, nullable bool) (any, error) {
	if src == nil {
		if !nullable {
			return nil, newValidationErrorf(name, "has value null but it is not nullable")
		}
		return nil, nil
	}
	var value any
	var valid bool
	switch typ.Kind() {
	case types.BooleanKind:
		if _, ok := src.(bool); ok {
			value = src
			valid = true
		}
	case types.IntKind:
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
				return nil, newValidationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = int(v)
		}
	case types.UintKind:
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
			min, max := typ.UintRange()
			if v < min || v > max {
				return nil, newValidationErrorf(name, "has value %d which is not in the range [%d, %d]", v, min, max)
			}
			value = uint(v)
		}
	case types.FloatKind:
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
			v, err = strconv.ParseFloat(string(src), typ.BitSize())
			valid = err == nil
		}
		if valid {
			if math.IsNaN(v) {
				if typ.IsReal() {
					return nil, newValidationErrorf(name, "has a value of NaN, which is not allowed")
				}
			} else {
				min, max := typ.FloatRange()
				if v < min || v > max {
					return nil, newValidationErrorf(name, "has a value %f that is not in the range [%f, %f]", v, min, max)
				}
			}
			value = v
		}
	case types.DecimalKind:
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
				return nil, newValidationErrorf(name, "has a value %s that is not in range [%s, %s]", v, min, max)
			}
			value = v
		}
	case types.DateTimeKind:
		if t, ok := src.(time.Time); ok {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, newValidationErrorf(name, "has date and time %q with a year not in range [1, 9999]", src)
			}
			value = t
			valid = true
		}
	case types.DateKind:
		if t, ok := src.(time.Time); ok {
			t = t.UTC()
			if y := t.Year(); y < 1 || y > 9999 {
				return nil, newValidationErrorf(name, "has date %q with a year not in range [1, 9999]", src)
			}
			value = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			valid = true
		}
	case types.TimeKind:
		switch src := src.(type) {
		case time.Time:
			value = time.Date(1970, 1, 1, src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), time.UTC)
			valid = true
		case []byte:
			value, valid = parseTime(src)
		case string:
			value, valid = parseTime(src)
		}
	case types.YearKind:
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
	case types.UUIDKind:
		if s, ok := src.(string); ok {
			if v, err := uuid.Parse(s); err == nil {
				value = v.String()
				valid = true
			}
		}
	case types.JSONKind:
		if v, ok := src.([]byte); ok {
			src = json.RawMessage(v)
		}
		if !validJSON(src) {
			return nil, fmt.Errorf("database returned an invalid JSON for property %s", name)
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
				return nil, fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
					errors.Abbreviate(v, 20), name)
			}
			if values := typ.Values(); values != nil {
				if !slices.Contains(values, v) {
					return nil, newValidationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else if rx := typ.Regexp(); rx != nil {
				if !rx.MatchString(v) {
					return nil, newValidationErrorf(name, "has a not allowed value of %q", errors.Abbreviate(v, 20))
				}
			} else {
				if l, ok := typ.ByteLen(); ok && len(v) > l {
					return nil, newValidationErrorf(name, "has value %q that is longer than %d bytes", errors.Abbreviate(v, 20), l)
				}
				if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
					return nil, newValidationErrorf(name, "has value %q that is longer than %d characters", errors.Abbreviate(v, 20), l)
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
				if n < typ.MinItems() || n > typ.MaxItems() {
					return nil, newValidationErrorf(name, "is an array with %d items, but they must be in range [%d, %d]", n, typ.MinItems(), typ.MaxItems())
				}
				a := make([]any, n)
				t := typ.Elem()
				for i := 0; i < n; i++ {
					v := rv.Index(i).Interface()
					a[i], err = normalizeDatabaseFileProperty(name, t, v, false)
					if err != nil {
						return nil, err
					}
				}
				value = a
				valid = true
			}
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
					m[k], err = normalizeDatabaseFileProperty(name, t, v, false)
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
			src, name, typ.Kind())
	}
	return value, nil
}

// validateStringProperty validates a string property like
// normalizeDatabaseFileProperty does.
func validateStringProperty(p types.Property, s string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("database returned a value of %q for column %s, which does not contain valid UTF-8 characters",
			errors.Abbreviate(s, 20), p.Name)
	}
	if l, ok := p.Type.ByteLen(); ok && len(s) > l {
		return newValidationErrorf(p.Name, "has value %q that is longer than %d bytes", errors.Abbreviate(s, 20), l)
	}
	if l, ok := p.Type.CharLen(); ok && utf8.RuneCountInString(s) > l {
		return newValidationErrorf(p.Name, "has value %q that is longer than %d characters", errors.Abbreviate(s, 20), l)
	}
	return nil
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
