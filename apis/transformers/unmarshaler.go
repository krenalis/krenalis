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

// ErrSyntaxInvalid is the error returned by Unmarshal when the data being
// unmarshaled is not valid JSON, or does not conform to the expected structure.
var ErrSyntaxInvalid = errors.New("syntax is not valid")

// SchemaValidationError represents a validation error related to the output
// schema. It can be returned by Unmarshal for each single result in the
// Result.Error field.
type SchemaValidationError struct {
	kind  schemaValidationKind
	msg   string
	path  string
	terms map[string]string
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
		return fmt.Sprintf("%s %q does not exist", err.terms["property"], err.path)
	case missingProperty:
		return fmt.Sprintf("required %s %q is missing", err.terms["property"], err.path)
	case invalidValue:
		return fmt.Sprintf("%s %q %s", err.terms["property"], err.path, err.msg)
	}
	panic("invalid SchemaValidationError's kind")
}

func (err *SchemaValidationError) appendIndexToPath(i int) {
	err.path = "[" + strconv.Itoa(i) + "]." + err.path
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

// newErrPropertyNotExist returns a new SchemaValidationError with kind
// propertyNotExist.
func newErrPropertyNotExist(path string, terms map[string]string) error {
	return &SchemaValidationError{kind: propertyNotExist, path: path, terms: terms}
}

// newErrMissingProperty returns a new SchemaValidationError with kind
// missingProperty.
func newErrMissingProperty(path string, terms map[string]string) error {
	return &SchemaValidationError{kind: missingProperty, path: path, terms: terms}
}

// newErrInvalidValue returns a new SchemaValidationError with kind
// invalidValue.
func newErrInvalidValue(msg, path string, terms map[string]string) error {
	return &SchemaValidationError{kind: invalidValue, msg: msg, path: path, terms: terms}
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
// according to the schema of its elements, which must be an Object.
//
// It returns the error ErrSyntaxInvalid if the data being unmarshaled is not
// valid JSON or does not conform to the expected structure.
//
// If a single result returned has an error, it can be a *ExecutionError or a
// *SchemaValidationError.
//
// For JavaScript, it assumes that the JSON code has been marshaled by a
// JavaScript JSON.stringify(v) call after executing:
//
//	BigInt.prototype.toJSON = function() { return this.toString(); }
//
// The following are the expected JSON for each schema type:
//   - Boolean: true or false
//   - Int, Int8, Int16, Int24, and Int32: a Number representing an integer
//   - Int64: a String representing an integer
//   - UInt, UInt8, UInt16, UInt24, and UInt32: a Number representing an integer
//   - UInt64: a String representing an integer
//   - Float and Float32: a Number
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
//   - Int, Int8, Int16, Int24, Int32, and Int64: a Number representing an integer
//   - UInt, UInt8, UInt16, UInt24, UInt32, and Uint64: a Number representing an integer
//   - Float and Float32: a Number
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
	if !schema.Valid() {
		return nil, errors.New("apis/transformers: schema is not valid")
	}
	if schema.PhysicalType() != types.PtObject {
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
			return nil, ErrSyntaxInvalid
		}
		return nil, err
	}
	if tok.Kind() != '[' {
		return nil, ErrSyntaxInvalid
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
			return nil, ErrSyntaxInvalid
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
				result.Error = err
			} else {
				result.Value = value.(map[string]any)
			}
		case "error":
			tok, err := d.readToken()
			if err != nil {
				return nil, err
			}
			if tok.Kind() != '"' {
				return nil, ErrSyntaxInvalid
			}
			result.Error = NewExecutionError(tok.String())
		default:
			return nil, ErrSyntaxInvalid
		}
		results = append(results, result)
		tok, err = d.readToken()
		if err != nil {
			return nil, err
		}
		if tok.Kind() != '}' {
			return nil, ErrSyntaxInvalid
		}
	}
	return results, nil
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
		defer func() {
			// Consume the remaining items.
			if _, ok := err.(*SchemaValidationError); ok {
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
		if t.PhysicalType() != types.PtArray {
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
				if err, ok := err.(*SchemaValidationError); ok {
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
			if _, ok := err.(*SchemaValidationError); ok {
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
		switch t.PhysicalType() {
		case types.PtObject:
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
						if err, ok := err.(*SchemaValidationError); ok {
							err.appendNameToPath(name)
						}
						return nil, err
					}
				}
				o[name] = value
			}
			for _, p := range t.Properties() {
				if p.Required {
					if _, ok := o[p.Name]; !ok {
						return nil, newErrMissingProperty(p.Name, d.opts.terms)
					}
				}
			}
			_, err = d.readToken()
			if err != nil {
				return nil, err
			}
			return o, nil
		case types.PtMap:
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

// value returns the marshalled value of v according to t.
func (d decoder) value(v jsontext.Value, t types.Type) (any, error) {
	pt := t.PhysicalType()
	switch pt {
	case types.PtBoolean:
		if v.Kind() == 'f' {
			return false, nil
		} else if v.Kind() == 't' {
			return true, nil
		}
	case types.PtInt, types.PtInt8, types.PtInt16, types.PtInt24, types.PtInt64:
		var s string
		switch v.Kind() {
		case '0':
			if pt != types.PtInt64 || !d.opts.int64AsString {
				s = string(v)
			}
		case '"':
			if pt == types.PtInt64 && d.opts.int64AsString {
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
	case types.PtUInt, types.PtUInt8, types.PtUInt16, types.PtUInt24, types.PtUInt64:
		var s string
		switch v.Kind() {
		case '0':
			if pt != types.PtUInt64 || !d.opts.int64AsString {
				s = string(v)
			}
		case '"':
			if pt == types.PtUInt64 && d.opts.int64AsString {
				s = d.unquoteString(v)
			}
		}
		if s != "" {
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				if min, max := t.UIntRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "", d.opts.terms)
				}
				return uint(n), nil
			}
		}
	case types.PtFloat, types.PtFloat32:
		if v.Kind() == '0' {
			bs := 64
			if pt == types.PtFloat32 {
				bs = 32
			}
			if n, err := strconv.ParseFloat(string(v), bs); err == nil {
				if min, max := t.FloatRange(); n < min || n > max {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%g, %g]: %g", min, max, n), "", d.opts.terms)
				}
				return n, nil
			}
		}
	case types.PtDecimal:
		if v.Kind() == '"' {
			if n, err := decimal.NewFromString(d.unquoteString(v)); err == nil {
				if min, max := t.DecimalRange(); n.LessThan(min) || n.GreaterThan(max) {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "", d.opts.terms)
				}
				return n, nil
			}
		}
	case types.PtDateTime:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.datetimeFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return t, nil
				}
			}
		}
	case types.PtDate:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.dateFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				if y := t.Year(); 1 <= y && y <= 9999 {
					return time.Date(y, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), nil
				}
			}
		}
	case types.PtTime:
		if v.Kind() == '"' {
			if t, err := time.Parse(d.opts.timeFormat, d.unquoteString(v)); err == nil {
				t = t.UTC()
				return time.Date(1970, 1, 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC), nil
			}
		}
	case types.PtYear:
		if v.Kind() == '0' {
			y, err := strconv.ParseInt(string(v), 10, 64)
			if err == nil {
				if y < types.MinYear || y > types.MaxYear {
					return nil, newErrInvalidValue(fmt.Sprintf("is out of range [1, 9999]: %d", y), "", d.opts.terms)
				}
				return int(y), nil
			}
		}
	case types.PtUUID:
		if v.Kind() == '"' {
			if u, err := uuid.ParseBytes(d.unquoteBytes(v)); err == nil {
				return u.String(), nil
			}
		}
	case types.PtJSON:
		if v.Kind() == '"' {
			data := d.unquoteBytes(v)
			if !json.Valid(data) {
				return nil, newErrInvalidValue(fmt.Sprintf("does not contain valid JSON: %s", data), "", d.opts.terms)
			}
			return json.RawMessage(data), nil
		}
	case types.PtInet:
		if v.Kind() == '"' {
			if ip, err := netip.ParseAddr(d.unquoteString(v)); err == nil {
				return ip.String(), nil
			}
		}
	case types.PtText:
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

// unquoteBytes unquote a JSON string.
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
