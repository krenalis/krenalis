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

	"github.com/meergo/meergo/core/internal/state"
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

// InputValidationError represents an error that occurs when validating a
// property value.
type InputValidationError struct {
	path string
	msg  string
}

// inputValidationErrorf returns an InputValidationError based on a format
// specifier. The error message can report the invalid value and should complete
// the sentence "property foo ".
func inputValidationErrorf(path string, format string, a ...any) InputValidationError {
	return InputValidationError{
		path: path,
		msg:  fmt.Sprintf(format, a...),
	}
}

func (err InputValidationError) Error() string {
	return fmt.Sprintf("property '%s' ", err.path) + err.msg
}

func (err InputValidationError) appendKey(key string) InputValidationError {
	err.path += "[" + strconv.Quote(key) + "]"
	return err
}

func (err InputValidationError) prependPath(path string) InputValidationError {
	err.path = path + "." + err.path
	return err
}

func invalidType(name string, src any, typ types.Type) InputValidationError {
	return inputValidationErrorf(name, "has type %T that is not allowed for type %s type", src, typ)
}

// normalize normalizes a property value, and returns its normalized value. If
// the value is not valid it returns an InputValidationError.
func normalize(name string, typ types.Type, src any, nullable bool, layouts *state.TimeLayouts) (any, error) {
	if src == nil {
		if !nullable {
			return nil, inputValidationErrorf(name, "has value null but it is not nullable")
		}
		return nil, nil
	}
	switch k := typ.Kind(); k {
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
		case []byte:
			if s == nil && nullable {
				return nil, nil
			}
			v = string(s)
		default:
			return nil, invalidType(name, src, typ)
		}
		if !utf8.ValidString(v) {
			return nil, inputValidationErrorf(name, "does not contain valid UTF-8 characters")
		}
		if values := typ.Values(); values != nil {
			if !slices.Contains(values, v) {
				return nil, inputValidationErrorf(name, "contains an unsupported value")
			}
		} else if rx := typ.Regexp(); rx != nil {
			if !rx.MatchString(v) {
				return nil, inputValidationErrorf(name, "contains an unsupported value")
			}
		} else {
			if l, ok := typ.ByteLen(); ok && len(v) > l {
				return nil, inputValidationErrorf(name, "has a value longer than %d bytes", l)
			}
			if l, ok := typ.CharLen(); ok && utf8.RuneCountInString(v) > l {
				return nil, inputValidationErrorf(name, "has a value longer than %d characters", l)
			}
		}
		return v, nil
	case types.BooleanKind:
		switch src.(type) {
		case bool:
			return src, nil
		case string:
			switch src {
			case "true":
				return true, nil
			case "false":
				return false, nil
			default:
				return nil, inputValidationErrorf(name, "has a string value but it is not 'true' or 'false'")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
	case types.IntKind:
		var v int64
		var err error
		// Keep in sync with the case 'types.YearKind'.
		switch src := src.(type) {
		case int:
			v = int64(src)
		case int8:
			v = int64(src)
		case int16:
			v = int64(src)
		case int32:
			v = int64(src)
		case int64:
			v = src
		case uint:
			if src > math.MaxInt64 {
				min, max := typ.IntRange()
				return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", min, max)
			}
			v = int64(src)
		case uint8:
			v = int64(src)
		case uint16:
			v = int64(src)
		case uint32:
			v = int64(src)
		case uint64:
			if src > math.MaxInt64 {
				min, max := typ.IntRange()
				return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", min, max)
			}
			v = int64(src)
		case float32:
			f := float64(src)
			if math.IsInf(f, 0) || f != math.Trunc(f) {
				return nil, inputValidationErrorf(name, "has a float32 value that cannot represent an int(%d) value", typ.BitSize())
			}
			v = int64(f)
		case float64:
			if math.IsInf(src, 0) || src != math.Trunc(src) {
				return nil, inputValidationErrorf(name, "has a float64 value that cannot represent an int(%d) value", typ.BitSize())
			}
			v = int64(src)
		case decimal.Decimal:
			v, err = src.Int64()
			if err != nil {
				return nil, inputValidationErrorf(name, "has a decimal.decimal value that cannot represent an int value")
			}
		case string:
			v, err = strconv.ParseInt(src, 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that does not represent an int value")
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = strconv.ParseInt(string(src), 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent an int value")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if min, max := typ.IntRange(); v < min || v > max {
			return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", min, max)
		}
		return int(v), nil
	case types.UintKind:
		var v uint64
		var err error
		switch src := src.(type) {
		case int:
			if src < 0 {
				return nil, inputValidationErrorf(name, "has a negative int value that cannot represent an uint(%d) value", typ.BitSize())
			}
			v = uint64(src)
		case int8:
			v = uint64(src)
		case int16:
			v = uint64(src)
		case int32:
			v = uint64(src)
		case int64:
			if src < 0 {
				return nil, inputValidationErrorf(name, "has a negative int64 value that cannot represent an uint(%d) value", typ.BitSize())
			}
			v = uint64(src)
		case uint:
			v = uint64(src)
		case uint8:
			v = uint64(src)
		case uint16:
			v = uint64(src)
		case uint32:
			v = uint64(src)
		case uint64:
			v = src
		case float32:
			f := float64(src)
			if src < 0 || math.IsInf(f, 1) || f != math.Trunc(f) {
				return nil, inputValidationErrorf(name, "has a float32 value that cannot represent an uint(%d) value", typ.BitSize())
			}
			v = uint64(src)
		case float64:
			if src < 0 || math.IsInf(src, 1) || src != math.Trunc(src) {
				return nil, inputValidationErrorf(name, "has a float64 value that cannot represent an uint(%d) value", typ.BitSize())
			}
			v = uint64(src)
		case decimal.Decimal:
			v, err = src.Uint64()
			if err != nil {
				return nil, inputValidationErrorf(name, "has a decimal.decimal value that cannot represent an uint value")
			}
		case string:
			v, err = strconv.ParseUint(src, 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent an uint value")
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = strconv.ParseUint(string(src), 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent an uint value")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if min, max := typ.UintRange(); v < min || v > max {
			return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", min, max)
		}
		return uint(v), nil
	case types.FloatKind:
		var v float64
		var err error
		switch src := src.(type) {
		case int:
			min, max := minIntRepresentableAsFloat64, maxIntRepresentableAsFloat64
			if typ.BitSize() == 32 {
				min, max = minIntRepresentableAsFloat32, maxIntRepresentableAsFloat32
			}
			if src < min || src > max {
				return nil, inputValidationErrorf(name, "has an int value that cannot represent a float(%d) value", typ.BitSize())
			}
			v = float64(src)
		case int8:
			v = float64(src)
		case int16:
			v = float64(src)
		case int32:
			v = float64(src)
		case int64:
			min, max := int64(minIntRepresentableAsFloat64), int64(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				min, max = minIntRepresentableAsFloat32, maxIntRepresentableAsFloat32
			}
			if src < min || src > max {
				return nil, inputValidationErrorf(name, "has an int64 value that cannot represent a float(%d) value", typ.BitSize())
			}
			v = float64(src)
		case uint:
			max := uint(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				max = uint(maxIntRepresentableAsFloat32)
			}
			if src > max {
				return nil, inputValidationErrorf(name, "has an uint value that cannot represent a float(%d) value", typ.BitSize())
			}
			v = float64(src)
		case uint8:
			v = float64(src)
		case uint16:
			v = float64(src)
		case uint32:
			v = float64(src)
		case uint64:
			max := uint64(maxIntRepresentableAsFloat64)
			if typ.BitSize() == 32 {
				max = uint64(maxIntRepresentableAsFloat32)
			}
			if src > max {
				return nil, inputValidationErrorf(name, "has an uint64 value that cannot represent a float(%d) value", typ.BitSize())
			}
			v = float64(src)
		case float32:
			v = float64(src)
		case float64:
			if typ.BitSize() == 32 && !math.IsNaN(src) && float64(float32(src)) != src {
				return nil, inputValidationErrorf(name, "has a float64 value that cannot represent a float(%d) value", typ.BitSize())
			}
			v = src
		case decimal.Decimal:
			var ok bool
			v, ok = src.Float64()
			if !ok {
				return nil, inputValidationErrorf(name, "has a decimal.Decimal value that cannot represent a float(%d) value", typ.BitSize())
			}
		case string:
			v, err = strconv.ParseFloat(src, typ.BitSize())
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent a float(%d) value", typ.BitSize())
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = strconv.ParseFloat(string(src), typ.BitSize())
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent a float(%d) value", typ.BitSize())
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if math.IsNaN(v) {
			if typ.IsReal() {
				return nil, inputValidationErrorf(name, "has a value of NaN, which is not allowed")
			}
		} else {
			min, max := typ.FloatRange()
			if v < min || v > max {
				return nil, inputValidationErrorf(name, "has a value %f that is not in the range [%f, %f]", v, min, max)
			}
		}
		return v, nil
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
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that does not represent a decimal(%d,%d)) value", p, s)
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = decimal.Parse(src, p, s)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that does not represent a decimal(%d,%d)) value", p, s)
			}
		case fmt.Stringer:
			v, err = decimal.Parse(src.String(), p, s)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a fmt.Stringer value that does not represent a decimal(%d,%d)) value", p, s)
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if min, max := typ.DecimalRange(); v.Less(min) || v.Greater(max) {
			return nil, inputValidationErrorf(name, "has a value that is not in range [%s, %s]", min, max)
		}
		return v, nil
	case types.DateTimeKind:
		var t time.Time
		var err error
		switch src := src.(type) {
		case time.Time:
			t = src
		case float64:
			var ok bool
			t, ok = dateTimeFromUnixFloat(src, layouts.DateTime)
			if !ok {
				return nil, inputValidationErrorf(name, "has a float64 value that cannot represent a datetime value")
			}
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			switch layouts.DateTime {
			case "":
				t, err = iso8601.ParseString(src)
				if err != nil {
					return nil, inputValidationErrorf(name, "has a string value that cannot represent a datetime value")
				}
			case "unix", "unixmilli", "unixmicro", "unixnano":
				n, err := strconv.ParseInt(src, 10, 64)
				if err != nil {
					return nil, inputValidationErrorf(name, "has a string value that cannot represent a datetime value")
				}
				var ok bool
				t, ok = dateTimeFromUnixInt(n, layouts.DateTime)
				if !ok {
					return nil, inputValidationErrorf(name, "has a string value that cannot represent a datetime value")
				}
			default:
				t, err = time.Parse(layouts.DateTime, src)
				if err != nil {
					return nil, inputValidationErrorf(name, "has a string value that cannot represent a datetime value")
				}
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		t = t.UTC()
		if y := t.Year(); y < types.MinYear || y > types.MaxYear {
			return nil, inputValidationErrorf(name, "has date and time with a year not in range [1, 9999]")
		}
		return t, nil
	case types.DateKind:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = src
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
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent a date value")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		t = t.UTC()
		if y := t.Year(); y < types.MinYear || y > types.MaxYear {
			return nil, inputValidationErrorf(name, "has date with a year not in range [1, 9999]")
		}
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
	case types.TimeKind:
		var t time.Time
		switch src := src.(type) {
		case time.Time:
			t = time.Date(1970, 1, 1, src.Hour(), src.Minute(), src.Second(), src.Nanosecond(), time.UTC)
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			var err error
			if layouts.Time == "" {
				t, err = iso8601.ParseString(src)
			} else {
				t, err = time.Parse(layouts.Time, src)
			}
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that does not represent a time value")
			}
			t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			var err error
			if layouts.Time == "" {
				t, err = iso8601.Parse(src)
			} else {
				t, err = time.Parse(layouts.Time, string(src))
			}
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent a time value")
			}
			t = time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
		}
		return t, nil
	case types.YearKind:
		var v int64
		var err error
		// Keep in sync with the case 'types.IntKind'.
		switch src := src.(type) {
		case int:
			v = int64(src)
		case int8:
			v = int64(src)
		case int16:
			v = int64(src)
		case int32:
			v = int64(src)
		case int64:
			v = src
		case uint:
			if src > math.MaxInt64 {
				return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", types.MinYear, types.MaxYear)
			}
			v = int64(src)
		case uint8:
			v = int64(src)
		case uint16:
			v = int64(src)
		case uint32:
			v = int64(src)
		case uint64:
			if src > math.MaxInt64 {
				return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", types.MinYear, types.MaxYear)
			}
			v = int64(src)
		case float32:
			f := float64(src)
			if math.IsInf(f, 0) || f != math.Trunc(f) {
				return nil, inputValidationErrorf(name, "has a float32 value that cannot represent a year value")
			}
			v = int64(f)
		case float64:
			v = int64(src)
			if math.IsInf(src, 0) || src != math.Trunc(src) {
				return nil, inputValidationErrorf(name, "has a float64 value that cannot represent a year value")
			}
		case decimal.Decimal:
			v, err = src.Int64()
			if err != nil {
				return nil, inputValidationErrorf(name, "has a deciaml.Decimal value that cannot represent a year value")
			}
		case string:
			v, err = strconv.ParseInt(src, 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent a year value")
			}
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, err = strconv.ParseInt(string(src), 10, 64)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent a year value")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if v < types.MinYear || v > types.MaxYear {
			return nil, inputValidationErrorf(name, "has value which is not in the range [%d, %d]", types.MinYear, types.MaxYear)
		}
		return int(v), nil
	case types.UUIDKind:
		switch src := src.(type) {
		case string:
			if src == "" && nullable {
				return nil, nil
			}
			v, ok := types.ParseUUID(src)
			if !ok {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent a uuid value")
			}
			return v, nil
		case []byte:
			if src == nil && nullable {
				return nil, nil
			}
			v, ok := types.DecodeUUID(src)
			if !ok {
				return nil, inputValidationErrorf(name, "has a []byte value that cannot represent a uuid value")
			}
			return v, nil
		default:
			return nil, invalidType(name, src, typ)
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
				return nil, inputValidationErrorf(name, "cannot be unmarshalled")
			}
		default:
			return nil, invalidType(name, src, typ)
		}
		if data == nil {
			return nil, inputValidationErrorf(name, "has a nil value")
		}
		if !json.Valid(data) {
			return nil, inputValidationErrorf(name, "is not valid JSON")
		}
		return json.Value(data), nil
	case types.InetKind:
		var addr netip.Addr
		switch ip := src.(type) {
		case string:
			if ip == "" && nullable {
				return nil, nil
			}
			// Remove the number of bits in the netmask, if any.
			if i := strings.IndexByte(ip, '/'); i > 0 {
				ip = ip[:i]
			}
			var err error
			addr, err = netip.ParseAddr(ip)
			if err != nil {
				return nil, inputValidationErrorf(name, "has a string value that cannot represent a valid inet value")
			}
		case net.IP:
			if ip == nil && nullable {
				return nil, nil
			}
			var ok bool
			addr, ok = netip.AddrFromSlice(ip)
			if !ok {
				return nil, inputValidationErrorf(name, "has a net.IP value that cannot represent a valid inet value")
			}
			// Unmap an IPv6-mapped IPv4 address as the net.IP.String method does.
			if addr.Is4In6() {
				addr = addr.Unmap()
			}
		case netip.Addr:
			addr = ip
		default:
			return nil, invalidType(name, src, typ)
		}
		if !addr.IsValid() {
			return nil, inputValidationErrorf(name, "is not a valid IP address")
		}
		return addr.WithZone("").String(), nil
	case types.ArrayKind:
		if s, ok := src.(string); ok {
			// Snowflake only supports json as the item type. The driver returns the value as a JSON array.
			if s == "" || s[0] != '[' || typ.Elem().Kind() != types.JSONKind {
				return nil, inputValidationErrorf(name, "has a string value but does not contain a JSON array")
			}
			v := json.Value(s)
			if !json.Valid(v) {
				return nil, inputValidationErrorf(name, "has a string value but is not valid JSON")
			}
			min := typ.MinElements()
			max := typ.MaxElements()
			arr := []any{}
			for i, element := range v.Elements() {
				if i == max {
					return nil, inputValidationErrorf(name, "is an array with more than %d elements; they must be in range [%d, %d]", max, min, max)
				}
				arr = append(arr, element)
			}
			if len(arr) < min {
				return nil, inputValidationErrorf(name, "is an array with less than %d elements; they must be in range [%d, %d]", min, min, max)
			}
			return arr, nil
		}
		rv := reflect.ValueOf(src)
		if rv.Kind() != reflect.Slice {
			return nil, invalidType(name, src, typ)
		}
		if rv.IsNil() && nullable {
			return nil, nil
		}
		var err error
		n := rv.Len()
		if n < typ.MinElements() || n > typ.MaxElements() {
			return nil, inputValidationErrorf(name, "is an array with %d elements, but they must be in range [%d, %d]", n, typ.MinElements(), typ.MaxElements())
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
						return nil, inputValidationErrorf(name, "contains the duplicated value %v", e)
					}
				}
			}
		}
		return a, nil
	case types.ObjectKind:
		src, ok := src.(map[string]any)
		if !ok {
			return nil, invalidType(name, src, typ)
		}
		if src == nil && nullable {
			return nil, nil
		}
		var err error
		for _, p := range typ.Properties() {
			value, ok := src[p.Name]
			if !ok {
				if !p.ReadOptional {
					err := inputValidationErrorf(p.Name, "does not have a value, but the property is not optional for reading")
					return nil, err.prependPath(name)
				}
				continue
			}
			src[p.Name], err = normalize(p.Name, p.Type, value, p.Nullable, layouts)
			if err != nil {
				if err, ok := err.(InputValidationError); ok {
					return nil, err.prependPath(name)
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
		return src, nil
	case types.MapKind:
		if s, ok := src.(string); ok {
			// Snowflake only supports json as the value type. The driver returns the value as a JSON object.
			if s == "" || s[0] != '{' || typ.Elem().Kind() != types.JSONKind {
				return nil, inputValidationErrorf(name, "has a string value but does not contain a JSON object")
			}
			v := json.Value(s)
			if !json.Valid(v) {
				return nil, inputValidationErrorf(name, "has a string value but is not valid JSON")
			}
			m := map[string]any{}
			for k, v := range v.Properties() {
				m[k] = v
			}
			return m, nil
		}
		rv := reflect.ValueOf(src)
		if rv.Kind() != reflect.Map {
			return nil, invalidType(name, src, typ)
		}
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
				if err, ok := err.(InputValidationError); ok {
					return nil, err.appendKey(k)
				}
				return nil, err
			}
		}
		return m, nil
	}
	return nil, invalidType(name, src, typ)
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
