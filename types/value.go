//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package types

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/decimal"
	"github.com/meergo/meergo/json"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
)

// Decode reads JSON from r and decodes it, validating it against t. t must be a
// valid non-generic type, and T must be the Go type that corresponds to t.
//
// It returns a json.SyntaxError if the data being unmarshaled is not valid
// JSON, and it returns a *SchemaValidationError if an error occurs during
// schema validation. Specifically, if a required property is missing, it
// results in a schema validation error.
//
// The following are the expected JSON values for each type:
//
//   - text: a JSON String
//   - boolean: true or false
//   - int (8, 16, 24, and 32 bits): a JSON Number representing an integer
//   - int (64 bits): a JSON String representing an integer
//   - uint (8, 16, 24, and 32 bits): a JSON Number representing an integer
//   - uint (64 bits): a JSON String representing an integer
//   - float: a JSON Number, or one of "NaN", "Infinity" or "-Infinity"
//   - decimal: a JSON String representing a JSON Number
//   - datetime: a JSON String representing a time in the ISO8601 format
//   - date: a JSON String representing a date in the ISO8601 format, formatted
//     as the Go time format "2006-01-02"
//   - time: a JSON String representing a time in the ISO8601 format, formatted
//     as the Go time format "15:04:05.999999999"
//   - year: a JSON Number representing an integer
//   - uuid: a JSON String representing a UUID
//   - json: a JSON value; JSON null is always interpreted as Value("null")
//   - inet: a JSON String representing an IP number
//   - array: a JSON Array
//   - object: a JSON Object
//   - map: a JSON Object
func Decode[T any](r io.Reader, t Type) (T, error) {
	if r == nil {
		var zero T
		return zero, errors.New("r is nil")
	}
	if t.Generic() {
		var zero T
		return zero, errors.New("json: type is a generic type")
	}
	if t.Kind() == InvalidKind {
		var zero T
		return zero, errors.New("json: type is the invalid type")
	}
	v, err := decode(r, t)
	vt, ok := v.(T)
	if err == nil && !ok {
		err = fmt.Errorf("json: Decode[%T] called with type kind %s", vt, t.Kind())
	}
	return vt, err
}

// Marshal encodes the given data, based on the provided schema, into JSON, and
// returns it. schema must be a valid non-generic type.
//
// For json properties, both nil and json.Value("null") are marshaled as JSON
// null.
//
// Unlike Decode, this function does not validate the data. Its behavior is
// undefined if the value does not validate against the type.
func Marshal(data any, schema Type) (json.Value, error) {
	if schema.Generic() {
		return nil, errors.New("json: schema is a generic type")
	}
	if schema.Kind() == InvalidKind {
		return nil, errors.New("json: schema is the invalid type")
	}
	return marshal(nil, data, schema)
}

var (
	nan         = []byte("NaN")
	posInfinity = []byte("Infinity")
	negInfinity = []byte("-Infinity")
)

func decode(r io.Reader, t Type) (any, error) {
	d := decoder{dec: json.NewDecoder(r)}
	value, err := d.unmarshal(t)
	if err != nil {
		if _, ok := err.(*SchemaValidationError); ok {
			// Consume the remaining tokens to return a ErrSyntaxInvalid error
			// in case of a syntax error, instead of the validation error.
			if err := d.consumeTokens(); err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	if _, err := d.readToken(); err != io.EOF {
		return nil, json.NewSyntaxError(err, 0)
	}
	return value, nil
}

// decoder implements a decoder for JSON.
type decoder struct {
	dec *json.Decoder
}

// consumeTokens consume the remaining tokens returning the ErrSyntaxInvalid
// error if the JSON source is not valid.
func (d decoder) consumeTokens() error {
	var err error
	for err == nil {
		_, err = d.readToken()
	}
	if err == io.EOF {
		return nil
	}
	return err
}

// peek peeks the next token kind.
func (d decoder) peek() json.Kind {
	return d.dec.PeekKind()
}

// readToken reads a token.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readToken() (json.Token, error) {
	return d.dec.ReadToken()
}

// readValue reads a value.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readValue() (json.Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = json.NewSyntaxError(errors.New("invalid JSON"), 0)
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoder) unmarshal(t Type) (_ any, err error) {
	if t.Kind() == JSONKind {
		v, err := d.readValue()
		if err != nil {
			return nil, err
		}
		return slices.Clone(v), nil
	}
	switch d.peek() {
	case '[':
		// DecodeBySchema an array.
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if t.Kind() != ArrayKind {
			return nil, newErrInvalidValue("cannot be an array", "")
		}
		minElements, maxElements := t.MinElements(), t.MaxElements()
		elements := make([]any, 0, minElements)
		for i := 0; d.peek() != ']'; i++ {
			if i == maxElements {
				return nil, newErrInvalidValue(fmt.Sprintf("contains more than %d elements", maxElements), "")
			}
			elem, err := d.unmarshal(t.Elem())
			if err != nil {
				if err, ok := err.(*SchemaValidationError); ok {
					err.appendIndexToPath(i)
				}
				return nil, err
			}
			elements = append(elements, elem)
			i++
		}
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if len(elements) < minElements {
			return nil, newErrInvalidValue(fmt.Sprintf("contains less than %d elements", minElements), "")
		}
		return elements, nil
	case '{':
		// DecodeBySchema an object.
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		switch t.Kind() {
		case ObjectKind:
			o := map[string]any{}
			for {
				if d.peek() == '}' {
					break
				}
				// Read the property's name.
				tok, err := d.readToken()
				if err != nil {
					return nil, err
				}
				name := tok.String()
				if !IsValidPropertyName(name) {
					return nil, json.NewSyntaxError(errors.New("property name is not valid"), 0)
				}
				p, ok := t.Property(name)
				if !ok {
					return nil, newErrPropertyNotExist(name)
				}
				// Read the property's value.
				var value any
				if d.peek() == 'n' {
					if _, err := d.readToken(); err != nil {
						return nil, err
					}
					if p.Type.Kind() == JSONKind {
						value = json.Value("null")
					} else {
						if !p.Nullable {
							return nil, newErrInvalidValue("cannot be null", p.Name)
						}
					}
				} else {
					value, err = d.unmarshal(p.Type)
					if err != nil {
						if err, ok := err.(*SchemaValidationError); ok {
							err.appendNameToPath(name)
						}
						return nil, err
					}
				}
				o[name] = value
			}
			for _, p := range t.Properties() {
				if p.ReadOptional {
					continue
				}
				if _, ok := o[p.Name]; ok {
					continue
				}
				return nil, newErrMissingProperty(p.Name)
			}
			_, err = d.readToken()
			if err != nil {
				return nil, err
			}
			return o, nil
		case MapKind:
			m := map[string]any{}
			for {
				// Read the property's name or the end of the map.
				tok, err := d.readToken()
				if err != nil {
					return nil, err
				}
				if tok.Kind() == '}' {
					break
				}
				name := tok.String()
				// Read the property's value.
				value, err := d.unmarshal(t.Elem())
				if err != nil {
					return nil, err
				}
				m[name] = value
			}
			return m, nil
		}
		return nil, newErrInvalidValue("cannot be an object ", "")
	default:
		value, err := d.readValue()
		if err != nil {
			return nil, err
		}
		return d.value(value, t)
	case 0:
		_, err := d.readToken()
		if err == io.EOF {
			err = json.NewSyntaxError(err, 0)
		}
		return nil, err
	}
}

// unquoteString unquote a JSON string.
func (d decoder) unquoteString(v []byte) []byte {
	return json.Value(v).AppendUnquote(nil)
}

// formatString formats a JSON string into a formatted string.
func (d decoder) formatString(v json.Value) string {
	b := v.AppendUnquote(nil)
	return `"` + strings.ReplaceAll(strings.ReplaceAll(string(b), `\`, `\\`), `"`, `\"`) + `"`
}

// value returns the unmarshalled value of v according to t.
func (d decoder) value(v json.Value, t Type) (any, error) {
	switch t.Kind() {
	case TextKind:
		if v.Kind() == '"' {
			s := string(d.unquoteString(v))
			if values := t.Values(); values != nil {
				if !slices.Contains(values, s) {
					return nil, newErrInvalidValue(fmt.Sprintf("has an invalid value: %s; valid values are %s",
						d.formatString(v), formatValues(values)), "")
				}
				return s, nil
			} else if rx := t.Regexp(); rx != nil {
				if !rx.MatchString(s) {
					return nil, newErrInvalidValue(fmt.Sprintf("has an invalid value: %s; it does not match the property's regular expression",
						d.formatString(v)), "")
				}
				return s, nil
			} else {
				if n, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, newErrInvalidValue(fmt.Sprintf("is longer than %d characters: %s", n, d.formatString(v)), "")
				}
				if n, ok := t.ByteLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, newErrInvalidValue(fmt.Sprintf("is longer than %d bytes: %s", n, d.formatString(v)), "")
				}
				return s, nil
			}
		}
	case BooleanKind:
		if v.Kind() == 'f' {
			return false, nil
		} else if v.Kind() == 't' {
			return true, nil
		}
	case IntKind:
		var s string
		switch v.Kind() {
		case '0':
			if t.BitSize() != 64 {
				s = string(v)
			}
		case '"':
			if t.BitSize() == 64 {
				s = string(d.unquoteString(v))
			}
		}
		if s != "" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				if min, max := t.IntRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "")
				}
				return int(n), nil
			}
		}
	case UintKind:
		var s string
		switch v.Kind() {
		case '0':
			if t.BitSize() != 64 {
				s = string(v)
			}
		case '"':
			if t.BitSize() == 64 {
				s = string(d.unquoteString(v))
			}
		}
		if s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				if min, max := t.UintRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "")
				}
				return uint(n), nil
			}
		}
	case FloatKind:
		switch v.Kind() {
		case '0':
			if n, err := strconv.ParseFloat(string(v), t.BitSize()); err == nil {
				if min, max := t.FloatRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%g, %g]: %g", min, max, n), "")
				}
				return n, nil
			}
		case '"':
			if bytes.Equal(v, nan) || bytes.Equal(v, posInfinity) || bytes.Equal(v, negInfinity) {
				if t.IsReal() {
					return nil, newErrInvalidValue(fmt.Sprintf("is not a real: %s", string(v)), "")
				}
				var n float64
				if bytes.Equal(v, nan) {
					n = math.NaN()
				} else if v[0] == 'p' {
					n = math.Inf(1)
				} else {
					n = math.Inf(-1)
				}
				return n, nil
			}
		}
	case DecimalKind:
		if v.Kind() == '0' {
			if n, err := decimal.Parse(v, t.Precision(), t.Scale()); err == nil {
				if min, max := t.DecimalRange(); n.Less(min) || n.Greater(max) {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "")
				}
				return n, nil
			}
		}
	case DateTimeKind:
		if v.Kind() == '"' {
			if t, err := iso8601.Parse(v.AppendUnquote(nil)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return t, nil
				}
			}
		}
	case DateKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("2006-01-02", string(d.unquoteString(v))); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
				}
			}
		}
	case TimeKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("15:04:05.999999999", string(d.unquoteString(v))); err == nil {
				t = t.UTC()
				return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
			}
		}
	case YearKind:
		if v.Kind() == '0' {
			y, err := strconv.ParseInt(string(v), 10, 64)
			if err == nil {
				if y < MinYear || y > MaxYear {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [1, 9999]: %d", y), "")
				}
				return int(y), nil
			}
		}
	case UUIDKind:
		if v.Kind() == '"' {
			if u, err := uuid.ParseBytes(v.AppendUnquote(nil)); err == nil {
				return u.String(), nil
			}
		}
	case JSONKind:
		if v.Kind() == '"' {
			return v.AppendUnquote(nil), nil
		}
	case InetKind:
		if v.Kind() == '"' {
			if ip, err := netip.ParseAddr(string(d.unquoteString(v))); err == nil {
				return ip.String(), nil
			}
		}
	case ArrayKind, ObjectKind, MapKind:
	default:
		return nil, fmt.Errorf("json: unexpected %s type", t)
	}
	// Return an invalid value error.
	var value string
	switch v.Kind() {
	case 'f':
		value = "false"
	case 't':
		value = "true"
	case '"':
		value = d.formatString(v)
	case '0':
		value = v.String()
	default:
		return nil, fmt.Errorf("json: unxpected kind '%s'", string(v.Kind()))
	}
	return nil, newErrInvalidValue("does not have a valid value: "+value, "")
}

// formatValues formats values to be used in an error message.
func formatValues(values []string) string {
	var b []byte
	last := len(values) - 1
	for i, value := range values {
		if i == 10 {
			b = append(b, "..."...)
			break
		}
		if i == last {
			b = append(b, ", and "...)
		} else if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, '"')
		b = append(b, strings.ReplaceAll(strings.ReplaceAll(value, `\`, `\\`), `"`, `\"`)...)
		b = append(b, '"')
	}
	return string(b)
}

// marshal marshals v as a JSON value and appends it to b.
func marshal(b []byte, data any, t Type) (json.Value, error) {
	if data == nil {
		return append(b, "null"...), nil
	}
	switch v := data.(type) {
	case string:
		quoted, _ := json.Quote([]byte(v))
		b = append(b, quoted...)
	case bool:
		if v {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case int:
		quoted := t.Kind() != YearKind && t.BitSize() == 64
		if quoted {
			b = append(b, '"')
		}
		b = strconv.AppendInt(b, int64(v), 10)
		if quoted {
			b = append(b, '"')
		}
	case uint:
		if t.BitSize() == 64 {
			b = append(b, '"')
		}
		b = strconv.AppendUint(b, uint64(v), 10)
		if t.BitSize() == 64 {
			b = append(b, '"')
		}
	case float64:
		if math.IsNaN(v) {
			b = append(b, `"NaN"`...)
		} else if math.IsInf(v, 0) {
			if v > 0 {
				b = append(b, `"Infinity"`...)
			} else {
				b = append(b, `"-Infinity"`...)
			}
		} else {
			b = strconv.AppendFloat(b, v, 'g', -1, t.BitSize())
		}
	case decimal.Decimal:
		b = append(b, v.String()...)
	case time.Time:
		b = append(b, '"')
		switch t.Kind() {
		case DateTimeKind:
			layout := "2006-01-02T15:04:05.000000000Z07:00"
			if ns := v.Nanosecond(); ns == 0 {
				layout = "2006-01-02T15:04:05Z07:00"
			} else if ns%1e6 == 0 {
				layout = "2006-01-02T15:04:05.000Z07:00"
			} else if ns%1e3 == 0 {
				layout = "2006-01-02T15:04:05.000000Z07:00"
			}
			b = v.AppendFormat(b, layout)
		case DateKind:
			b = v.AppendFormat(b, time.DateOnly)
		case TimeKind:
			layout := "15:04:05.000000000"
			if ns := v.Nanosecond(); ns == 0 {
				layout = "15:04:05"
			} else if ns%1e6 == 0 {
				layout = "15:04:05.000"
			} else if ns%1e3 == 0 {
				layout = "15:04:05.000000"
			}
			b = v.AppendFormat(b, layout)
		}
		b = append(b, '"')
	case json.Value:
		value, _ := json.Compact(v)
		b = append(b, value...)
	default:
		rv := reflect.ValueOf(v)
		switch t.Kind() {
		case ArrayKind:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				var err error
				b, err = marshal(b, item, t.Elem())
				if err != nil {
					return nil, err
				}
			}
			b = append(b, ']')
		case ObjectKind:
			b = append(b, '{')
			var err error
			i := 0
			for _, p := range t.Properties() {
				rv := rv.MapIndex(reflect.ValueOf(p.Name))
				if !rv.IsValid() {
					continue
				}
				if i > 0 {
					b = append(b, ',')
				}
				b = append(b, '"')
				b = append(b, p.Name...)
				b = append(b, '"', ':')
				b, err = marshal(b, rv.Interface(), p.Type)
				if err != nil {
					return nil, err
				}
				i++
			}
			b = append(b, '}')
		case MapKind:
			type entry struct {
				k string
				v any
			}
			s := make([]entry, rv.Len())
			iter := rv.MapRange()
			i := 0
			for iter.Next() {
				s[i].k = iter.Key().String()
				s[i].v = iter.Value().Interface()
				i++
			}
			slices.SortFunc(s, func(a, b entry) int {
				return strings.Compare(a.k, b.k)
			})
			var err error
			vt := t.Elem()
			b = append(b, '{')
			for i, e := range s {
				if i > 0 {
					b = append(b, ',')
				}
				quoted, _ := json.Quote([]byte(e.k))
				b = append(b, quoted...)
				b = append(b, ':')
				b, err = marshal(b, e.v, vt)
				if err != nil {
					return nil, err
				}
			}
			b = append(b, '}')
		default:
			return nil, fmt.Errorf("json: unexpected type %s", t)
		}
	}
	return b, nil
}

// SchemaValidationError represents a validation error related to the output
// schema. It can be returned by DecodeBySchema for each single result in the
// Result.Error field.
type SchemaValidationError struct {
	kind schemaValidationKind
	msg  string
	path string
}

type schemaValidationKind int

const (
	propertyNotExist schemaValidationKind = iota
	missingProperty
	invalidValue
)

func (err *SchemaValidationError) Error() string {
	switch err.kind {
	case propertyNotExist:
		return fmt.Sprintf("property %q does not exist", err.path)
	case missingProperty:
		return fmt.Sprintf("non-optional property %q is missing", err.path)
	case invalidValue:
		if err.path != "" && err.path[len(err.path)-1] == ']' {
			return fmt.Sprintf("%q %s", err.path, err.msg)
		}
		return fmt.Sprintf("property %q %s", err.path, err.msg)
	}
	panic("invalid SchemaValidationError's kind")
}

func (err *SchemaValidationError) appendIndexToPath(i int) {
	path := err.path
	err.path = "[" + strconv.Itoa(i) + "]"
	if path != "" {
		err.path += "." + path
	}
}

func (err *SchemaValidationError) appendNameToPath(name string) {
	if err.path == "" {
		err.path = name
	} else if err.path[0] == '[' {
		err.path = name + err.path
	} else {
		err.path = name + "." + err.path
	}
}

// newErrInvalidValue returns a new SchemaValidationError with kind
// invalidValue.
func newErrInvalidValue(msg, path string) error {
	return &SchemaValidationError{kind: invalidValue, msg: msg, path: path}
}

// newErrMissingProperty returns a new SchemaValidationError with kind
// missingProperty.
func newErrMissingProperty(path string) error {
	return &SchemaValidationError{kind: missingProperty, path: path}
}

// newErrPropertyNotExist returns a new SchemaValidationError with kind
// propertyNotExist.
func newErrPropertyNotExist(path string) error {
	return &SchemaValidationError{kind: propertyNotExist, path: path}
}
