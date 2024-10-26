//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package json

import (
	"bytes"
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

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json/internal/json/jsontext"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
	"github.com/relvacode/iso8601"
)

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
	d := decoder{dec: NewDecoder(r)}
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
func (d decoder) peek() Kind {
	return d.dec.Peek()
}

// readToken reads a token.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readToken() (Token, error) {
	tok, err := d.dec.ReadToken()
	if err == io.ErrUnexpectedEOF {
		err = &SyntaxError{err: errors.New("invalid JSON")}
	}
	return tok, err
}

// readValue reads a value.
// It returns the ErrSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readValue() (Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = &SyntaxError{err: errors.New("invalid JSON")}
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoder) unmarshal(t types.Type) (_ any, err error) {
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
func (d decoder) unquoteString(v []byte) []byte {
	b, _ := Unquote(v)
	return b
}

// formatString formats a JSON string into a formatted string.
func (d decoder) formatString(v []byte) string {
	b, _ := jsontext.AppendUnquote(nil, v)
	return `"` + strings.ReplaceAll(strings.ReplaceAll(string(b), `\`, `\\`), `"`, `\"`) + `"`
}

// value returns the unmarshalled value of v according to t.
func (d decoder) value(v Value, t types.Type) (any, error) {
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

// decoder implements a decoder for JSON.
type decoder struct {
	dec *Decoder
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
