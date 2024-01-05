//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"chichi/apis/state"
	"chichi/connector/types"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// errSyntaxInvalid is the error returned by Unmarshal when the data being
// unmarshaled is not valid JSON, or does not conform to the expected structure.
var errSyntaxInvalid = errors.New("syntax is not valid")

// functionValidationError represents a validation error related to the output
// schema. It can be returned by Unmarshal for each single result in the
// Result.Err field. It implements the ValidationError interface of apis.
type functionValidationError struct {
	path string
	msg  string
}

func (err *functionValidationError) Error() string {
	return fmt.Sprintf(err.msg, err.path)
}

func (err *functionValidationError) PropertyPath() string {
	return err.path
}

func (err *functionValidationError) appendIndexToPath(i int) {
	err.path = "[" + strconv.Itoa(i) + "]." + err.path
}

func (err *functionValidationError) appendNameToPath(name string) {
	if err.path == "" {
		err.path = name
	} else if err.path[0] == '[' {
		err.path = name + err.path
	} else {
		err.path = name + "." + err.path
	}
}

// newErrPropertyNotExist returns a new functionValidationError for a
// nonexistent property.
func newErrPropertyNotExist(path string, terms map[string]string) error {
	return &functionValidationError{
		path: path,
		msg:  terms["property"] + " %q does not exist",
	}
}

// newErrMissingProperty returns a new functionValidationError indicating a
// missing property.
func newErrMissingProperty(path string, terms map[string]string) error {
	return &functionValidationError{
		path: path,
		msg:  "required " + terms["property"] + " %q is missing",
	}
}

// newErrInvalidValue returns a new functionValidationError indicating an
// invalid value.
func newErrInvalidValue(msg, path string, terms map[string]string) error {
	return &functionValidationError{
		path: path,
		msg:  terms["property"] + " %q " + strings.ReplaceAll(msg, "%", "%%"),
	}
}

// decoder implements a decoder for the JSON code returned by JavaScript or
// Python.
type decoder struct {
	dec  *jsontext.Decoder
	opts *decoderOptions
}

// decoderOptions are the language-specific options used internally by the
// decoder.
type decoderOptions struct {
	terms          map[string]string
	int64AsString  bool
	datetimeFormat string
	dateFormat     string
	timeFormat     string
}

// javaScriptDecoderOptions are the JavaScript's options used by the decoder.
var javaScriptDecoderOptions = decoderOptions{
	terms: map[string]string{
		"null":     "null",
		"false":    "false",
		"true":     "true",
		"array":    "array",
		"items":    "elements",
		"object":   "object",
		"property": "property",
	},
	int64AsString:  true,
	datetimeFormat: "2006-01-02T15:04:05.000Z07:00",
	dateFormat:     "2006-01-02T15:04:05.000Z07:00",
	timeFormat:     "2006-01-02T15:04:05.000Z07:00",
}

// pythonDecoderOptions are the Python's options used by the decoder.
var pythonDecoderOptions = decoderOptions{
	terms: map[string]string{
		"null":     "None",
		"false":    "False",
		"true":     "True",
		"array":    "list",
		"items":    "items",
		"object":   "dict",
		"property": "key",
	},
	int64AsString:  false,
	datetimeFormat: "2006-01-02 15:04:05.999999",
	dateFormat:     "2006-01-02",
	timeFormat:     "15:04:05.999999",
}

// Unmarshal decodes a JSON array of objects read from r, validating it
// according to the schema of its elements, which must be an Object or invalid.
// An invalid schema is treated as an object with no properties.
//
// For JavaScript, it assumes that the JSON code has been marshaled by a
// JavaScript JSON.stringify(v) call after executing:
//
//	BigInt.prototype.toJSON = function() { return this.toString(); }
//
// The following are the expected JSON for each schema type:
//   - Boolean: true or false
//   - Int (8, 16, 24, and 32 bits): a Number representing an integer
//   - Int (64 bits): a String representing an integer
//   - Uint (8, 16, 24, and 32 bits): a Number representing an integer
//   - Uint (64 bits): a String representing an integer
//   - Float: a Number
//   - Decimal: a String representing a number
//   - DateTime, Date, and Time: a String representing a time formatted as
//     "2006-01-02T15:04:05.000Z07:00"
//   - Year: a Number representing an integer
//   - UUID: a String representing a UUID
//   - JSON: a String representing a JSON value
//   - Inet: a String representing an IP number
//   - Text: a String
//   - Array: an array
//   - Object: an object
//   - Map: an object
//
// For Python, it assumes that the JSON code has been marshaled by a Python
// json.dumps(v, default=str) call.
//
// The following are the expected JSON for each schema type:
//   - Boolean: true or false
//   - Int: a Number representing an integer
//   - Uint: a Number representing an integer
//   - Float: a Number
//   - Decimal: a String representing a number
//   - DateTime: a String representing a time formatted as "2006-01-02
//     15:04:05.999999"
//   - Date: a String representing a date formatted as "2006-01-02"
//   - Time: a String representing a time formatted as "15:04:05.999999"
//   - Year: a Number representing an integer
//   - UUID: a String representing a UUID
//   - JSON: a String representing a JSON value
//   - Inet: a String representing an IP number
//   - Text: a String
//   - Array: an array
//   - Object: an object
//   - Map: an object
func Unmarshal(r io.Reader, schema types.Type, language state.Language) ([]Result, error) {
	if r == nil {
		return nil, errors.New("apis/transformers: r is nil")
	}
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return nil, errors.New("apis/transformers: schema is not an object")
	}
	d := decoder{dec: jsontext.NewDecoder(r)}
	switch language {
	case state.JavaScript:
		d.opts = &javaScriptDecoderOptions
	case state.Python:
		d.opts = &pythonDecoderOptions
	default:
		return nil, errors.New("apis/transformers: language is not valid")
	}
	tok, err := d.readToken()
	if err != nil {
		if err == io.EOF {
			return nil, errSyntaxInvalid
		}
		return nil, err
	}
	if tok.Kind() != '[' {
		return nil, errSyntaxInvalid
	}
	var results []Result
	for {
		tok, err := d.readToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind() == ']' {
			break
		}
		if tok.Kind() != '{' {
			return nil, errSyntaxInvalid
		}
		// Read the key:
		tok, err = d.readToken()
		if err != nil {
			return nil, err
		}
		var result Result
		switch tok.String() {
		case "value":
			value, err := d.unmarshal(schema)
			if err != nil {
				result.Err = err
			} else {
				result.Value = value.(map[string]any)
			}
		case "error":
			tok, err := d.readToken()
			if err != nil {
				return nil, err
			}
			if tok.Kind() != '"' {
				return nil, errSyntaxInvalid
			}
			result.Err = FunctionExecutionError(tok.String())
		default:
			return nil, errSyntaxInvalid
		}
		results = append(results, result)
		tok, err = d.readToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind() != '}' {
			return nil, errSyntaxInvalid
		}
	}
	if _, err := d.readToken(); err != io.EOF {
		return nil, errSyntaxInvalid
	}
	return results, nil
}

// peekKind peeks the next token kind.
func (d decoder) peekKind() jsontext.Kind {
	return d.dec.PeekKind()
}

// readToken reads a token.
// It returns the errSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readToken() (jsontext.Token, error) {
	tok, err := d.dec.ReadToken()
	if err == io.ErrUnexpectedEOF {
		err = errSyntaxInvalid
	} else if _, ok := err.(*jsontext.SyntacticError); ok {
		err = errSyntaxInvalid
	}
	return tok, err
}

// readValue reads a value.
// It returns the errSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readValue() (jsontext.Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = errSyntaxInvalid
	} else if _, ok := err.(*jsontext.SyntacticError); ok {
		err = errSyntaxInvalid
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoder) unmarshal(t types.Type) (_ any, err error) {
	switch d.peekKind() {
	case '[':
		// Unmarshal an array.
		defer func() {
			// Consume the remaining items.
			if _, ok := err.(*functionValidationError); ok {
				for {
					if d.peekKind() == ']' {
						_, err2 := d.readToken()
						if err2 != nil {
							err = err2
							break
						}
					}
					_, err2 := d.readValue()
					if err2 != nil {
						err = err2
						break
					}
				}
			}
		}()
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if t.Kind() != types.ArrayKind {
			return nil, newErrInvalidValue("cannot be an "+d.opts.terms["array"], "", d.opts.terms)
		}
		minItems, maxItems := t.MinItems(), t.MaxItems()
		items := make([]any, 0, minItems)
		for i := 0; d.peekKind() != ']'; i++ {
			if i == maxItems {
				return nil, newErrInvalidValue(fmt.Sprintf("contains more than %d %s", maxItems, d.opts.terms["items"]), "", d.opts.terms)
			}
			item, err := d.unmarshal(t.Elem())
			if err != nil {
				if err, ok := err.(*functionValidationError); ok {
					err.appendIndexToPath(i)
				}
				return nil, err
			}
			items = append(items, item)
			i++
		}
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if len(items) < minItems {
			return nil, newErrInvalidValue(fmt.Sprintf("contains less than %d %s", minItems, d.opts.terms["items"]), "", d.opts.terms)
		}
		return items, nil
	case '{':
		// Unmarshal an object.
		defer func() {
			// Consume the remaining key/value pairs.
			if _, ok := err.(*functionValidationError); ok {
				var er error
				for er == nil && d.peekKind() != '}' {
					_, er = d.readValue()
				}
				if er == nil {
					_, er = d.readToken()
				}
				if er != nil {
					err = er
				}
			}
		}()
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		switch t.Kind() {
		case types.ObjectKind, types.InvalidKind:
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
				if !types.IsValidPropertyPath(name) {
					return nil, newErrPropertyNotExist(name, d.opts.terms)
				}
				if !t.Valid() {
					return nil, newErrPropertyNotExist(name, d.opts.terms)
				}
				p, ok := t.Property(name)
				if !ok {
					return nil, newErrPropertyNotExist(name, d.opts.terms)
				}
				// Read the property's value.
				var value any
				if d.peekKind() == 'n' {
					if _, err := d.readToken(); err != nil {
						return nil, err
					}
					if !p.Nullable {
						return nil, newErrInvalidValue("cannot be "+d.opts.terms["null"], p.Name, d.opts.terms)
					}
				} else {
					value, err = d.unmarshal(p.Type)
					if err != nil {
						if err, ok := err.(*functionValidationError); ok {
							err.appendNameToPath(name)
						}
						return nil, err
					}
				}
				o[name] = value
			}
			if t.Valid() {
				for _, p := range t.Properties() {
					if p.Required {
						if _, ok := o[p.Name]; !ok {
							return nil, newErrMissingProperty(p.Name, d.opts.terms)
						}
					}
				}
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
		return nil, newErrInvalidValue("cannot be a "+d.opts.terms["object"], "", d.opts.terms)
	default:
		value, err := d.readValue()
		if err != nil {
			return nil, err
		}
		return d.value(value, t)
	case 0:
		_, err := d.readToken()
		return nil, err
	}
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
			if t.BitSize() != 64 || !d.opts.int64AsString {
				s = string(v)
			}
		case '"':
			if t.BitSize() == 64 && d.opts.int64AsString {
				s = d.unquoteString(v)
			}
		}
		if s != "" {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				if min, max := t.IntRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "", d.opts.terms)
				}
				return int(n), nil
			}
		}
	case types.UintKind:
		var s string
		switch v.Kind() {
		case '0':
			if t.BitSize() != 64 || !d.opts.int64AsString {
				s = string(v)
			}
		case '"':
			if t.BitSize() == 64 && d.opts.int64AsString {
				s = d.unquoteString(v)
			}
		}
		if s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				if min, max := t.UintRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "", d.opts.terms)
				}
				return uint(n), nil
			}
		}
	case types.FloatKind:
		if v.Kind() == '0' {
			if n, err := strconv.ParseFloat(string(v), t.BitSize()); err == nil {
				if min, max := t.FloatRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%g, %g]: %g", min, max, n), "", d.opts.terms)
				}
				return n, nil
			}
		}
	case types.DecimalKind:
		if v.Kind() == '"' {
			if n, err := decimal.NewFromString(d.unquoteString(v)); err == nil {
				if min, max := t.DecimalRange(); n.LessThan(min) || n.GreaterThan(max) {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "", d.opts.terms)
				}
				return n, nil
			}
		}
	case types.DateTimeKind:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.datetimeFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return t, nil
				}
			}
		}
	case types.DateKind:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.dateFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
				}
			}
		}
	case types.TimeKind:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.timeFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
			}
		}
	case types.YearKind:
		if v.Kind() == '0' {
			y, err := strconv.ParseInt(string(v), 10, 64)
			if err == nil {
				if y < types.MinYear || y > types.MaxYear {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [1, 9999]: %d", y), "", d.opts.terms)
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
				return nil, newErrInvalidValue(fmt.Sprintf("does not contain valid JSON: %s", data), "", d.opts.terms)
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
						d.formatString(v), formatValues(values)), "", d.opts.terms)
				}
				return s, nil
			} else if rx := t.Regexp(); rx != nil {
				if !rx.MatchString(s) {
					return nil, newErrInvalidValue(fmt.Sprintf("has an invalid value: %s; it does not match the property's regular expression",
						d.formatString(v)), "", d.opts.terms)
				}
				return s, nil
			} else {
				if n, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, newErrInvalidValue(fmt.Sprintf("is longer than %d characters: %s", n, d.formatString(v)), "", d.opts.terms)
				}
				if n, ok := t.ByteLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, newErrInvalidValue(fmt.Sprintf("is longer than %d bytes: %s", n, d.formatString(v)), "", d.opts.terms)
				}
				return s, nil
			}
		}
	default:
		return nil, fmt.Errorf("apis/transformers: unexpected %s type", t)
	}
	// Return an invalid value error.
	var value string
	switch v.Kind() {
	case 'f':
		value = d.opts.terms["false"]
	case 't':
		value = d.opts.terms["true"]
	case '"':
		value = d.formatString(v)
	case '0':
		value = v.String()
	default:
		return nil, fmt.Errorf("apis/tranformations: unxpected kind '%s'", string(v.Kind()))
	}
	return nil, newErrInvalidValue("does not have a valid value: "+value, "", d.opts.terms)
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
