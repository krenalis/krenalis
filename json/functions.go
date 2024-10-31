//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

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

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json/internal/json"
	"github.com/meergo/meergo/json/internal/json/jsontext"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
)

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

// A SyntaxError is a description of a JSON syntax error that occurred during
// the encoding or decoding of JSON, in accordance with the JSON grammar.
type SyntaxError struct {
	err    error
	offset int64
}

func (err *SyntaxError) ByteOffset() int64 {
	if err, ok := err.err.(*jsontext.SyntacticError); ok {
		return err.ByteOffset
	}
	return err.offset
}

func (err *SyntaxError) Error() string {
	str := err.err.Error()
	if _, ok := err.err.(*jsontext.SyntacticError); ok {
		if strings.HasPrefix(str, "jsontext: ") {
			str = str[len("jsontext: "):]
		}
	}
	return str
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

// AppendUnquote writes the unquoted value of v into dst, and returns the
// extended buffer. v may contain leading and trailing JSON whitespace.
// It returns an error if v is not of String kind.
func (v Value) AppendUnquote(dst []byte) ([]byte, error) {
	return jsontext.AppendUnquote(dst, TrimSpace(v))
}

// Compact returns a copy of data with all insignificant whitespace removed. If
// data is already compact, it returns the original data unchanged. If data does
// not contain valid JSON, it returns nil and ErrInvalidJSON.
func Compact(data []byte) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, ErrInvalidJSON
	}
	v := jsontext.Value(slices.Clone(data))
	err := v.Compact()
	if err != nil {
		return nil, ErrInvalidJSON
	}
	return v, nil
}

// Decode deserialize JSON read from r into the value pointed by out.
// It returns an error if out is nil or is not a pointer.
func Decode(r io.Reader, out any) error {
	err := json.UnmarshalRead(r, out)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
}

// DecodeBySchema decodes JSON read from r, validating it according to the
// provided schema, which cannot be the invalid type. If a property is missing
// and it is not optional for reading, it returns a *SchemaValidationError
// error.
//
// It returns the error ErrSyntaxInvalid if the data being unmarshaled is not
// valid JSON and returns a *SchemaValidationError value if an error occurs
// during schema validation.
//
// The following are the expected JSON values for each schema type:
//
//   - Boolean: true or false
//   - Int (8, 16, 24, and 32 bits): a JSON Number representing an integer
//   - Int (64 bits): a JSON String representing an integer
//   - Uint (8, 16, 24, and 32 bits): a JSON Number representing an integer
//   - Uint (64 bits): a JSON String representing an integer
//   - Float: a JSON Number, or one of "NaN", "Infinity" or "-Infinity"
//   - Decimal: a JSON String representing a JSON Number
//   - DateTime: a JSON String representing a time in the ISO8601 format
//   - Date: a JSON String representing a date in the ISO8601 format, formatted
//     as the Go time format "2006-01-02"
//   - Time: a JSON String representing a time in the ISO8601 format, formatted
//     as the Go time format "15:04:05.999999999"
//   - Year: a JSON Number representing an integer
//   - UUID: a JSON String representing a UUID
//   - JSON: a JSON String representing a JSON value
//   - Inet: a JSON String representing an IP number
//   - Text: a JSON String
//   - Array: a JSON Array
//   - Object: a JSON Object
//   - Map: a JSON Object
func DecodeBySchema(r io.Reader, schema types.Type) (map[string]any, error) {
	return decodeBySchema(r, schema)
}

// Encode writes to out the JSON encoding of v.
func Encode(out io.Writer, v any) error {
	err := json.MarshalWrite(out, v)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
}

// Marshal encodes the given value.
func Marshal(v any) (Value, error) {
	val, err := json.Marshal(v)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return Value{}, &SyntaxError{err: err}
	}
	return val, nil
}

// Marshaler is the interface implemented by types that can marshal themselves
// into valid JSON.
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

// MarshalBySchema encodes the given value, based on the provided schema, into
// JSON, and returns it. schema cannot be the invalid type.
//
// Unlike DecodeBySchema, this function does not validate the value. Its
// behavior is undefined if the value does not validate against the type.
func MarshalBySchema(v any, schema types.Type) (Value, error) {
	return marshalBySchema(nil, v, schema)
}

var zeroByte = []byte(`\u0000`)

// StripZeroBytes removes all zero bytes ('\u0000') from the provided data,
// which may contain valid JSON code, and modifies the original slice in place.
// It returns the modified slice.
func StripZeroBytes(data []byte) []byte {
	p := data
	for {
		i := bytes.Index(p, zeroByte)
		if i == -1 {
			break
		}
		// Check if it is preceded by an even number or zero of backslashes.
		even := true
		for j := i - 1; j >= 0 && p[j] == '\\'; j-- {
			even = !even
		}
		if even {
			copy(p[i:], p[i+6:])
			p = p[:len(p)-6]
			data = data[:len(data)-6]
		} else {
			p = p[i+6:]
		}
	}
	return data
}

// TrimSpace returns a subslice of data with all leading and trailing whitespace
// removed. data must contain valid JSON.
func TrimSpace(data []byte) []byte {
	i, j := 0, len(data)-1
	for ; lookupTable[data[i]] == 1; i++ {
	}
	for ; lookupTable[data[j]] == 1; j-- {
	}
	return data[i : j+1]
}

// Unmarshaler is the interface for types that can unmarshal a JSON
// representation of themselves. The input is assumed to be a valid JSON value.
// UnmarshalJSON must copy the JSON data if it needs to retain it after
// returning.
//
// By convention, to mimic the behavior of [Unmarshal], Unmarshalers
// should implement UnmarshalJSON([]byte("null")) as a no-op.
type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

// Unquote removes the quotes from a JSON-encoded string and returns the
// unquoted data. If data is not valid JSON string it returns nil and
// ErrInvalidJSON.
func Unquote(data []byte) ([]byte, error) {
	d, err := jsontext.AppendUnquote(nil, TrimSpace(data))
	if err != nil {
		return nil, ErrInvalidJSON
	}
	return d, err
}

// Valid reports whether data is a valid JSON encoding and properly encoded in
// UTF-8.
func Valid(data []byte) bool {
	return jsontext.Value(data).IsValid()
}

var (
	nan         = []byte("NaN")
	posInfinity = []byte("Infinity")
	negInfinity = []byte("-Infinity")
)

func decodeBySchema(r io.Reader, schema types.Type) (map[string]any, error) {
	if r == nil {
		return nil, errors.New("r is nil")
	}
	if schema.Kind() == types.InvalidKind {
		return nil, errors.New("json: schema is the invalid type")
	}
	d := decoderBySchema{dec: NewDecoder(r)}
	value, err := d.unmarshal(schema)
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
		return nil, &SyntaxError{err: err}
	}
	return value.(map[string]any), nil
}

// decoderBySchema implements a decoder for JSON.
type decoderBySchema struct {
	dec *Decoder
}

// consumeTokens consume the remaining tokens returning the ErrSyntaxInvalid
// error if the JSON source is not valid.
func (d decoderBySchema) consumeTokens() error {
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
func (d decoderBySchema) peek() Kind {
	return d.dec.Peek()
}

// readToken reads a token.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoderBySchema) readToken() (Token, error) {
	tok, err := d.dec.ReadToken()
	if err == io.ErrUnexpectedEOF {
		err = &SyntaxError{err: errors.New("invalid JSON")}
	}
	return tok, err
}

// readValue reads a value.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoderBySchema) readValue() (Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = &SyntaxError{err: errors.New("invalid JSON")}
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoderBySchema) unmarshal(t types.Type) (_ any, err error) {
	switch d.peek() {
	case '[':
		// DecodeBySchema an array.
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if t.Kind() != types.ArrayKind {
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
		case types.ObjectKind:
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
				if !types.IsValidPropertyName(name) {
					return nil, &SyntaxError{err: errors.New("property name is not valid")}
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
					if !p.Nullable {
						return nil, newErrInvalidValue("cannot be null", p.Name)
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
		case types.MapKind:
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
			err = &SyntaxError{err: err}
		}
		return nil, err
	}
}

// unquoteString unquote a JSON string.
func (d decoderBySchema) unquoteString(v []byte) []byte {
	b, _ := Unquote(v)
	return b
}

// formatString formats a JSON string into a formatted string.
func (d decoderBySchema) formatString(v []byte) string {
	b, _ := jsontext.AppendUnquote(nil, v)
	return `"` + strings.ReplaceAll(strings.ReplaceAll(string(b), `\`, `\\`), `"`, `\"`) + `"`
}

// value returns the unmarshalled value of v according to t.
func (d decoderBySchema) value(v Value, t types.Type) (any, error) {
	switch t.Kind() {
	case types.BooleanKind:
		if v.Kind() == 'f' {
			return false, nil
		} else if v.Kind() == 't' {
			return true, nil
		}
	case types.IntKind:
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
	case types.UintKind:
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
	case types.FloatKind:
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
	case types.DecimalKind:
		if v.Kind() == '"' {
			if n, err := decimal.Parse(d.unquoteString(v), t.Precision(), t.Scale()); err == nil {
				if min, max := t.DecimalRange(); n.Less(min) || n.Greater(max) {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "")
				}
				return n, nil
			}
		}
	case types.DateTimeKind:
		if v.Kind() == '"' {
			if v, err := Unquote(v); err == nil {
				if t, err := iso8601.Parse(v); err == nil {
					t = t.UTC()
					if y := t.Year(); 1 <= y && y <= 9999 {
						return t, nil
					}
				}
			}
		}
	case types.DateKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("2006-01-02", string(d.unquoteString(v))); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
				}
			}
		}
	case types.TimeKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("15:04:05.999999999", string(d.unquoteString(v))); err == nil {
				t = t.UTC()
				return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
			}
		}
	case types.YearKind:
		if v.Kind() == '0' {
			y, err := strconv.ParseInt(string(v), 10, 64)
			if err == nil {
				if y < types.MinYear || y > types.MaxYear {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [1, 9999]: %d", y), "")
				}
				return int(y), nil
			}
		}
	case types.UUIDKind:
		if v.Kind() == '"' {
			if v, err := Unquote(v); err == nil {
				if u, err := uuid.ParseBytes(v); err == nil {
					return u.String(), nil
				}
			}
		}
	case types.JSONKind:
		if v.Kind() == '"' {
			s, err := Unquote(v)
			if err != nil {
				return nil, newErrInvalidValue(fmt.Sprint("contains invalid JSON"), "")
			}
			return Value(s), nil
		}
	case types.InetKind:
		if v.Kind() == '"' {
			if ip, err := netip.ParseAddr(string(d.unquoteString(v))); err == nil {
				return ip.String(), nil
			}
		}
	case types.TextKind:
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
	case types.ArrayKind, types.ObjectKind, types.MapKind:
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

// marshalBySchema marshals v as a JSON value and appends it to b.
func marshalBySchema(b []byte, v any, t types.Type) (Value, error) {
	if v == nil {
		return append(b, "null"...), nil
	}
	switch v := v.(type) {
	case bool:
		if v {
			b = append(b, "true"...)
		} else {
			b = append(b, "false"...)
		}
	case int:
		quoted := t.Kind() != types.YearKind && t.BitSize() == 64
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
		b = append(b, '"')
		b = append(b, v.String()...)
		b = append(b, '"')
	case time.Time:
		b = append(b, '"')
		switch t.Kind() {
		case types.DateTimeKind:
			b = v.AppendFormat(b, time.RFC3339Nano)
		case types.DateKind:
			b = v.AppendFormat(b, time.DateOnly)
		case types.TimeKind:
			b = v.AppendFormat(b, "15:04:05.999999999")
		}
		b = append(b, '"')
	case Value:
		b, _ = jsontext.AppendQuote(b, v)
	case string:
		var err error
		b, err = jsontext.AppendQuote(b, v)
		if err != nil {
			return nil, err
		}
	default:
		rv := reflect.ValueOf(v)
		switch t.Kind() {
		case types.ArrayKind:
			b = append(b, '[')
			n := rv.Len()
			for i := 0; i < n; i++ {
				if i > 0 {
					b = append(b, ',')
				}
				item := rv.Index(i).Interface()
				var err error
				b, err = marshalBySchema(b, item, t.Elem())
				if err != nil {
					return nil, err
				}
			}
			b = append(b, ']')
		case types.ObjectKind:
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
				b, err = marshalBySchema(b, rv.Interface(), p.Type)
				if err != nil {
					return nil, err
				}
				i++
			}
			b = append(b, '}')
		case types.MapKind:
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
				b, err = jsontext.AppendQuote(b, e.k)
				if err != nil {
					return nil, err
				}
				b = append(b, ':')
				b, err = marshalBySchema(b, e.v, vt)
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
