//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package encoding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/types"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
	"github.com/shopspring/decimal"
)

var (
	nan         = []byte("NaN")
	posInfinity = []byte("Infinity")
	negInfinity = []byte("-Infinity")
)

// ErrSyntaxInvalid is the error returned by Unmarshal when the data being
// unmarshaled is not valid JSON, or does not conform to the expected structure.
var ErrSyntaxInvalid = errors.New("syntax is not valid")

// SchemaValidationError represents a validation error related to the output
// schema. It can be returned by Unmarshal for each single result in the
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

// newErrPropertyNotExist returns a new SchemaValidationError with kind
// propertyNotExist.
func newErrPropertyNotExist(path string) error {
	return &SchemaValidationError{kind: propertyNotExist, path: path}
}

// newErrMissingProperty returns a new SchemaValidationError with kind
// missingProperty.
func newErrMissingProperty(path string) error {
	return &SchemaValidationError{kind: missingProperty, path: path}
}

// newErrInvalidValue returns a new SchemaValidationError with kind
// invalidValue.
func newErrInvalidValue(msg, path string) error {
	return &SchemaValidationError{kind: invalidValue, msg: msg, path: path}
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

// decoder implements a decoder for JSON.
type decoder struct {
	dec *jsontext.Decoder
}

// Unmarshal decodes a JSON object read from r, validating it according to the
// provided input schema, which must be an Object. If a property is missing and
// it is not optional for reading, it returns a *SchemaValidationError error.
//
// name is used in error messages as the name of the unmarshaled object.
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
func Unmarshal(r io.Reader, name string, schema types.Type) (map[string]any, error) {
	if r == nil {
		return nil, errors.New("r is nil")
	}
	if k := schema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("apis/encoding: schema is the invalid schema")
		}
		return nil, errors.New("apis/encoding:schema is not an object")
	}
	d := decoder{dec: jsontext.NewDecoder(r)}
	value, err := d.unmarshal(schema)
	if err != nil {
		if err, ok := err.(*SchemaValidationError); ok {
			// Consume the remaining tokens to return a ErrSyntaxInvalid error
			// in case of a syntax error, instead of the validation error.
			if err := d.consumeTokens(); err != nil {
				return nil, err
			}
			err.appendNameToPath(name)
		}
		return nil, err
	}
	if _, err := d.readToken(); err != io.EOF {
		return nil, ErrSyntaxInvalid
	}
	return value.(map[string]any), nil
}

// UnmarshalSlice is like Unmarshal but expects a JSON array of objects.
// The schema is the schema of the elements of the array.
func UnmarshalSlice(r io.Reader, name string, schema types.Type) ([]map[string]any, error) {
	if r == nil {
		return nil, errors.New("r is nil")
	}
	if k := schema.Kind(); k != types.ObjectKind {
		if k == types.InvalidKind {
			return nil, errors.New("apis/encoding: schema is the invalid schema")
		}
		return nil, errors.New("apis/encoding:schema is not an object")
	}
	d := decoder{dec: jsontext.NewDecoder(r)}
	tok, err := d.readToken()
	if err != nil {
		if err == io.EOF {
			return nil, ErrSyntaxInvalid
		}
		return nil, err
	}
	if tok.Kind() != '[' {
		return nil, ErrSyntaxInvalid
	}
	values := make([]map[string]any, 0)
	for i := 0; d.peekKind() != ']'; i++ {
		value, err := d.unmarshal(schema)
		if err != nil {
			if err, ok := err.(*SchemaValidationError); ok {
				// Consume the remaining tokens to return a ErrSyntaxInvalid error
				// in case of a syntax error, instead of the validation error.
				if err := d.consumeTokens(); err != nil {
					return nil, err
				}
				err.appendIndexToPath(i)
				err.appendNameToPath(name)
			}
			return nil, err
		}
		values = append(values, value.(map[string]any))
	}
	_, _ = d.readToken() // skip ']'
	if _, err := d.readToken(); err != io.EOF {
		return nil, ErrSyntaxInvalid
	}
	return values, nil
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

// peekKind peeks the next token kind.
func (d decoder) peekKind() jsontext.Kind {
	return d.dec.PeekKind()
}

// readToken reads a token.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readToken() (jsontext.Token, error) {
	tok, err := d.dec.ReadToken()
	if err == io.ErrUnexpectedEOF {
		err = ErrSyntaxInvalid
	} else if _, ok := err.(*jsontext.SyntacticError); ok {
		err = ErrSyntaxInvalid
	}
	return tok, err
}

// readValue reads a value.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readValue() (jsontext.Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = ErrSyntaxInvalid
	} else if _, ok := err.(*jsontext.SyntacticError); ok {
		err = ErrSyntaxInvalid
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoder) unmarshal(t types.Type) (_ any, err error) {
	switch d.peekKind() {
	case '[':
		// Unmarshal an array.
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if t.Kind() != types.ArrayKind {
			return nil, newErrInvalidValue("cannot be an array", "")
		}
		minElements, maxElements := t.MinElements(), t.MaxElements()
		elements := make([]any, 0, minElements)
		for i := 0; d.peekKind() != ']'; i++ {
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
		// Unmarshal an object.
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		switch t.Kind() {
		case types.ObjectKind:
			o := map[string]any{}
			for {
				if d.peekKind() == '}' {
					break
				}
				// Read the property's name.
				tok, err := d.readToken()
				if err != nil {
					return nil, err
				}
				name := tok.String()
				if !types.IsValidPropertyName(name) {
					return nil, ErrSyntaxInvalid
				}
				p, ok := t.Property(name)
				if !ok {
					return nil, newErrPropertyNotExist(name)
				}
				// Read the property's value.
				var value any
				if d.peekKind() == 'n' {
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
			err = ErrSyntaxInvalid
		}
		return nil, err
	}
}

// unquoteBytes unquote a JSON string.
func (d decoder) unquoteBytes(v []byte) []byte {
	b, _ := jsontext.AppendUnquote(nil, v)
	return b
}

// unquoteString unquote a JSON string.
func (d decoder) unquoteString(v []byte) string {
	b, _ := jsontext.AppendUnquote(nil, v)
	return string(b)
}

// formatString formats a JSON string into a formatted string.
func (d decoder) formatString(v []byte) string {
	b, _ := jsontext.AppendUnquote(nil, v)
	return `"` + strings.ReplaceAll(strings.ReplaceAll(string(b), `\`, `\\`), `"`, `\"`) + `"`
}

// value returns the unmarshalled value of v according to t.
func (d decoder) value(v jsontext.Value, t types.Type) (any, error) {
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
				s = d.unquoteString(v)
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
				s = d.unquoteString(v)
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
			if n, err := decimal.NewFromString(d.unquoteString(v)); err == nil {
				if min, max := t.DecimalRange(); n.LessThan(min) || n.GreaterThan(max) {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "")
				}
				return n, nil
			}
		}
	case types.DateTimeKind:
		if v.Kind() == '"' {
			if t, err := iso8601.Parse(d.unquoteBytes(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return t, nil
				}
			}
		}
	case types.DateKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("2006-01-02", d.unquoteString(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
				}
			}
		}
	case types.TimeKind:
		if v.Kind() == '"' {
			if t, err := time.Parse("15:04:05.999999999", d.unquoteString(v)); err == nil {
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
			if u, err := uuid.ParseBytes(d.unquoteBytes(v)); err == nil {
				return u.String(), nil
			}
		}
	case types.JSONKind:
		if v.Kind() == '"' {
			data := d.unquoteBytes(v)
			if !json.Valid(data) {
				return nil, newErrInvalidValue(fmt.Sprintf("does not contain valid JSON: %s", data), "")
			}
			return json.RawMessage(data), nil
		}
	case types.InetKind:
		if v.Kind() == '"' {
			if ip, err := netip.ParseAddr(d.unquoteString(v)); err == nil {
				return ip.String(), nil
			}
		}
	case types.TextKind:
		if v.Kind() == '"' {
			s := d.unquoteString(v)
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
		return nil, fmt.Errorf("apis/encoding: unexpected %s type", t)
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
		return nil, fmt.Errorf("apis/encoding: unxpected kind '%s'", string(v.Kind()))
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
