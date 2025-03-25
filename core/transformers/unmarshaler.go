//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package transformers

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

	"github.com/meergo/meergo/core/state"
	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/json"
	"github.com/meergo/meergo/types"

	"github.com/google/uuid"
)

var (
	nan         = []byte(`"NaN"`)
	posInfinity = []byte(`"Infinity"`)
	negInfinity = []byte(`"-Infinity"`)
)

// errSyntaxInvalid is the error returned by Unmarshal when the data being
// unmarshaled is not valid JSON, or does not conform to the expected structure.
var errSyntaxInvalid = errors.New("syntax is not valid")

// RecordValidationError represents an error that occurs when validating a
// transformed record.
type RecordValidationError struct {
	path string
	msg  string
}

func (err RecordValidationError) Error() string {
	if err.path == "" {
		return err.msg
	}
	return fmt.Sprintf(err.msg, err.path)
}

func (err RecordValidationError) appendIndexToPath(i int) RecordValidationError {
	err.path = "[" + strconv.Itoa(i) + "]." + err.path
	return err
}

func (err RecordValidationError) appendNameToPath(name string) RecordValidationError {
	if err.path == "" {
		err.path = name
	} else if err.path[0] == '[' {
		err.path = name + err.path
	} else {
		err.path = name + "." + err.path
	}
	return err
}

// errPropertyNotExist returns a new RecordValidationError for a nonexistent
// property.
func errPropertyNotExist(path string, terms map[string]string) error {
	return RecordValidationError{
		path: path,
		msg:  terms["property"] + " %q does not exist",
	}
}

// errMissingProperty returns a new RecordValidationError indicating a missing
// property.
func errMissingProperty(path string, terms map[string]string) error {
	return RecordValidationError{
		path: path,
		msg:  "required " + terms["property"] + " %q is missing",
	}
}

// errInvalidValue returns a new RecordValidationError indicating an invalid
// value.
func errInvalidValue(msg, path string, terms map[string]string) error {
	return RecordValidationError{
		path: path,
		msg:  terms["property"] + " %q " + strings.ReplaceAll(msg, "%", "%%"),
	}
}

// decoder implements a decoder for the JSON code returned by JavaScript or
// Python.
type decoder struct {
	dec  *json.Decoder
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
		"elements": "elements",
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
		"elements": "items",
		"object":   "dict",
		"property": "key",
	},
	int64AsString:  false,
	datetimeFormat: "2006-01-02 15:04:05.999999",
	dateFormat:     "2006-01-02",
	timeFormat:     "15:04:05.999999",
}

// Unmarshal decodes a JSON array of objects read from r into records,
// validating it according to the schema of its elements, which must be an
// object or invalid. An invalid schema is treated as an object with no
// properties.
//
// For JavaScript, apart from null, the following are the expected JSON for each
// schema type:
//   - boolean: true or false
//   - int(8, 16, 24, and 32 bits): a Number representing an integer
//   - int(64 bits): a String representing an integer
//   - uint (8, 16, 24, and 32 bits): a Number representing an integer
//   - uint (64 bits): a String representing an integer
//   - float: a Number or one of "NaN", "Infinity", and "-Infinity"
//   - decimal: a String representing a number
//   - datetime, date, and time: a String representing a time formatted as
//     "2006-01-02T15:04:05.000Z07:00"
//   - year: a Number representing an integer
//   - uuid: a String representing a UUID
//   - json: if preserveJSON is false: true, false, a Number, a String, an
//     Array, or an Object; Otherwise a String representing a JSON value
//   - inet: a String representing an IP number
//   - text: a String
//   - array: an array
//   - object: an object
//   - map: an object
//
// For Python, apart from null, the following are the expected JSON for each
// schema type:
//   - boolean: true or false
//   - int a Number representing an integer
//   - uint: a Number representing an integer
//   - float: a Number or one of "NaN", "Infinity", and "-Infinity"
//   - decimal: a String representing a number
//   - datetime: a String representing a time formatted as "2006-01-02
//     15:04:05.999999"
//   - date: a String representing a date formatted as "2006-01-02"
//   - time: a String representing a time formatted as "15:04:05.999999"
//   - year: a Number representing an integer
//   - uuid: a String representing a UUID
//   - inet: a String representing an IP number
//   - json: if preserveJSON is false: true, false, a Number, a String, an
//     Array, or an Object; Otherwise a String representing a JSON value
//   - text: a String
//   - array: an array
//   - object: an object
//   - map: an object
//
// If an error occurred during the transformation of a single record, either a
// RecordTransformationError or RecordValidationError is stored in the Err field
// of the corresponding record.
//
// If an error occurred during the transformation function execution, it returns
// a FunctionExecError.
func Unmarshal(r io.Reader, records []Record, schema types.Type, language state.Language, preserveJSON bool) error {
	if r == nil {
		return errors.New("core/transformers: r is nil")
	}
	if schema.Valid() && schema.Kind() != types.ObjectKind {
		return errors.New("core/transformers: schema is not an object")
	}
	d := decoder{dec: json.NewDecoder(r)}
	switch language {
	case state.JavaScript:
		d.opts = &javaScriptDecoderOptions
	case state.Python:
		d.opts = &pythonDecoderOptions
	default:
		return errors.New("core/transformers: language is not valid")
	}
	tok, err := d.readToken()
	if err != nil {
		if err == io.EOF {
			return errSyntaxInvalid
		}
		return err
	}
	if tok.Kind() != '{' {
		return errSyntaxInvalid
	}
	tok, err = d.readToken()
	if err != nil {
		return err
	}
	key := tok.String()
	if key == "error" {
		// Parse and return a FunctionExecError error.
		if tok, err = d.readToken(); err != nil {
			return err
		}
		if tok.Kind() != '"' {
			return errSyntaxInvalid
		}
		msg := tok.String()
		if tok, err = d.readToken(); err != nil {
			return err
		}
		if tok.Kind() != '}' {
			return errSyntaxInvalid
		}
		if _, err := d.readToken(); err != io.EOF {
			return errSyntaxInvalid
		}
		return FunctionExecError{msg: msg}
	}
	if key != "records" {
		return errSyntaxInvalid
	}
	// Parse the records.
	if tok, err = d.readToken(); err != nil {
		return err
	}
	if tok.Kind() != '[' {
		return errSyntaxInvalid
	}
	i := 0
	for {
		tok, err := d.readToken()
		if err != nil {
			return err
		}
		if tok.Kind() == ']' {
			break
		}
		if tok.Kind() != '{' {
			return errSyntaxInvalid
		}
		// Read the key:
		tok, err = d.readToken()
		if err != nil {
			return err
		}
		if i == len(records) {
			return fmt.Errorf("transformers/lambda: expected %d results got more", len(records))
		}
		switch tok.String() {
		case "value":
			properties, err := d.unmarshal(schema, preserveJSON, records[i].Purpose)
			if err != nil {
				if err == errSyntaxInvalid {
					return err
				}
				records[i].Properties = nil
				records[i].Err = err
			} else {
				records[i].Properties = properties.(map[string]any)
			}
		case "error":
			tok, err := d.readToken()
			if err != nil {
				return err
			}
			if tok.Kind() != '"' {
				return errSyntaxInvalid
			}
			records[i].Properties = nil
			records[i].Err = errors.New(tok.String())
		default:
			return errSyntaxInvalid
		}
		tok, err = d.readToken()
		if err != nil {
			return err
		}
		if tok.Kind() != '}' {
			return errSyntaxInvalid
		}
		i++
	}
	if tok, err = d.readToken(); err != nil {
		return err
	}
	if tok.Kind() != '}' {
		return errSyntaxInvalid
	}
	if _, err := d.readToken(); err != io.EOF {
		return errSyntaxInvalid
	}
	if i < len(records) {
		return fmt.Errorf("transformers/lambda: expected %d results got %d", len(records), i)
	}
	return nil
}

// peekKind peeks the next token kind.
func (d decoder) peekKind() json.Kind {
	return d.dec.PeekKind()
}

// readToken reads a token.
// It returns the errSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readToken() (json.Token, error) {
	tok, err := d.dec.ReadToken()
	if err == io.ErrUnexpectedEOF {
		err = errSyntaxInvalid
	} else if _, ok := err.(*json.SyntaxError); ok {
		err = errSyntaxInvalid
	}
	return tok, err
}

// readValue reads a value.
// It returns the errSyntaxInvalid error if the JSON source is not valid.
func (d decoder) readValue() (json.Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = errSyntaxInvalid
	} else if _, ok := err.(*json.SyntaxError); ok {
		err = errSyntaxInvalid
	}
	return v, err
}

// unmarshal unmarshals a JSON value.
func (d decoder) unmarshal(t types.Type, preserveJSON bool, purpose Purpose) (_ any, err error) {
	if t.Kind() == types.JSONKind && !preserveJSON {
		v, err := d.readValue()
		if err != nil {
			return nil, err
		}
		return json.Value(bytes.Clone(v)), nil
	}
	switch d.peekKind() {
	case '[':
		// Unmarshal an array.
		defer func() {
			// Consume the remaining items.
			if _, ok := err.(RecordValidationError); ok {
				if e := d.skipOut(); e != nil {
					err = e
				}
			}
		}()
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if t.Kind() != types.ArrayKind {
			return nil, errInvalidValue("cannot be an "+d.opts.terms["array"], "", d.opts.terms)
		}
		min := t.MinElements()
		max := t.MaxElements()
		arr := []any{}
		for i := 0; d.peekKind() != ']'; i++ {
			if i == max {
				return nil, errInvalidValue(fmt.Sprintf("contains more than %d %s", max, d.opts.terms["elements"]), "", d.opts.terms)
			}
			elem, err := d.unmarshal(t.Elem(), preserveJSON, purpose)
			if err != nil {
				if e, ok := err.(RecordValidationError); ok {
					err = e.appendIndexToPath(i)
				}
				return nil, err
			}
			arr = append(arr, elem)
			i++
		}
		if _, err := d.readToken(); err != nil {
			return nil, err
		}
		if len(arr) < min {
			return nil, errInvalidValue(fmt.Sprintf("contains less than %d %s", min, d.opts.terms["elements"]), "", d.opts.terms)
		}
		if t.Unique() {
			for i, elem := range arr {
				for _, item2 := range arr[i+1:] {
					if elem == item2 {
						return nil, errInvalidValue(fmt.Sprintf("contains a duplicated value: %v", elem), "", d.opts.terms)
					}
				}
			}
		}
		return arr, nil
	case '{':
		// Unmarshal an object.
		defer func() {
			// Consume the remaining key/value pairs.
			if _, ok := err.(RecordValidationError); ok {
				if e := d.skipOut(); e != nil {
					err = e
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
				if !types.IsValidPropertyName(name) {
					return nil, errPropertyNotExist(name, d.opts.terms)
				}
				if !t.Valid() {
					return nil, errPropertyNotExist(name, d.opts.terms)
				}
				p, ok := t.Property(name)
				if !ok {
					return nil, errPropertyNotExist(name, d.opts.terms)
				}
				// Read the property's value.
				var value any
				if d.peekKind() == 'n' {
					if _, err := d.readToken(); err != nil {
						return nil, err
					}
					if !p.Nullable {
						if p.Type.Kind() != types.JSONKind || preserveJSON {
							return nil, errInvalidValue("cannot be "+d.opts.terms["null"], p.Name, d.opts.terms)
						}
						value = json.Value("null")
					}
				} else {
					value, err = d.unmarshal(p.Type, preserveJSON, purpose)
					if err != nil {
						if e, ok := err.(RecordValidationError); ok {
							err = e.appendNameToPath(name)
						}
						return nil, err
					}
				}
				o[name] = value
			}
			if t.Valid() {
				switch purpose {
				case Create:
					for _, p := range t.Properties() {
						if !p.CreateRequired {
							continue
						}
						if _, ok := o[p.Name]; !ok {
							return nil, errMissingProperty(p.Name, d.opts.terms)
						}
					}
				case Update:
					for _, p := range t.Properties() {
						if !p.UpdateRequired {
							continue
						}
						if _, ok := o[p.Name]; !ok {
							return nil, errMissingProperty(p.Name, d.opts.terms)
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
				value, err := d.unmarshal(t.Elem(), preserveJSON, purpose)
				if err != nil {
					return nil, err
				}
				m[name] = value
			}
			return m, nil
		}
		msg := "cannot be a"
		if term := d.opts.terms["object"]; term == "object" {
			msg += "n object"
		} else {
			msg += " " + term
		}
		return nil, errInvalidValue(msg, "", d.opts.terms)
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

// SkipOut skips out of the current object or array and advances the read offset
// to the byte immediately after. If not in an object or array, it advances to
// the end of the input and returns io.EOF.
func (d decoder) skipOut() error {
	return d.dec.SkipOut()
}

// value returns the unmarshalled value of v according to t.
func (d decoder) value(v json.Value, t types.Type) (any, error) {
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
					return nil, errInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "", d.opts.terms)
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
					return nil, errInvalidValue(fmt.Sprintf("is out of range [%d, %d]: %d", min, max, n), "", d.opts.terms)
				}
				return uint(n), nil
			}
		}
	case types.FloatKind:
		switch v.Kind() {
		case '0':
			if n, err := strconv.ParseFloat(string(v), t.BitSize()); err == nil {
				if min, max := t.FloatRange(); n < min || n > max {
					return nil, errInvalidValue(fmt.Sprintf("is out of range [%g, %g]: %g", min, max, n), "", d.opts.terms)
				}
				return n, nil
			}
		case '"':
			if t.IsReal() {
				return nil, errInvalidValue(fmt.Sprintf("is not a real number: %s", string(v)), "", d.opts.terms)
			}
			if bytes.Equal(v, nan) {
				return math.NaN(), nil
			}
			if bytes.Equal(v, posInfinity) {
				return math.Inf(1), nil
			}
			if bytes.Equal(v, negInfinity) {
				return math.Inf(-1), nil
			}
		}
	case types.DecimalKind:
		if v.Kind() == '"' {
			if n, err := decimal.Parse(d.unquoteString(v), t.Precision(), t.Scale()); err == nil {
				if min, max := t.DecimalRange(); n.Less(min) || n.Greater(max) {
					return nil, errInvalidValue(fmt.Sprintf("is out of range [%s, %s]: %s", min, max, n), "", d.opts.terms)
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
					return nil, errInvalidValue(fmt.Sprintf("is out of range [1, 9999]: %d", y), "", d.opts.terms)
				}
				return int(y), nil
			}
		}
	case types.UUIDKind:
		if v.Kind() == '"' {
			if u, err := uuid.ParseBytes(v.AppendUnquote(nil)); err == nil {
				return u.String(), nil
			}
		}
	case types.JSONKind:
		if v.Kind() == '"' {
			data := v.AppendUnquote(nil)
			if !json.Valid(data) {
				return nil, errInvalidValue(fmt.Sprintf("does not contain valid JSON: %s", v), "", d.opts.terms)
			}
			return json.Value(data), nil
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
					return nil, errInvalidValue(fmt.Sprintf("has an invalid value: %s; valid values are %s",
						d.formatString(v), formatValues(values)), "", d.opts.terms)
				}
				return s, nil
			} else if rx := t.Regexp(); rx != nil {
				if !rx.MatchString(s) {
					return nil, errInvalidValue(fmt.Sprintf("has an invalid value: %s; it does not match the property's regular expression",
						d.formatString(v)), "", d.opts.terms)
				}
				return s, nil
			} else {
				if n, ok := t.CharLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, errInvalidValue(fmt.Sprintf("is longer than %d characters: %s", n, d.formatString(v)), "", d.opts.terms)
				}
				if n, ok := t.ByteLen(); ok && utf8.RuneCountInString(s) > n {
					return nil, errInvalidValue(fmt.Sprintf("is longer than %d bytes: %s", n, d.formatString(v)), "", d.opts.terms)
				}
				return s, nil
			}
		}
	default:
		return nil, fmt.Errorf("core/transformers: unexpected %s type", t)
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
		return nil, fmt.Errorf("core/tranformations: unxpected kind '%s'", string(v.Kind()))
	}
	return nil, errInvalidValue("does not have a valid value: "+value, "", d.opts.terms)
}

// unquoteString unquote a JSON string.
func (d decoder) unquoteString(v []byte) string {
	return string(json.Value(v).AppendUnquote(nil))
}

// formatString formats a JSON string into a formatted string.
func (d decoder) formatString(v []byte) string {
	b := json.Value(v).AppendUnquote(nil)
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
