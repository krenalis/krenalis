// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package transformers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/meergo/meergo/core/internal/state"
	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json"
	"github.com/meergo/meergo/tools/types"

	"github.com/google/uuid"
)

var (
	nan         = []byte(`"NaN"`)
	posInfinity = []byte(`"Infinity"`)
	negInfinity = []byte(`"-Infinity"`)
)

// errInvalidResponseFormat is returned when the response JSON does not conform
// to the expected format.
var errInvalidResponseFormat = errors.New("invalid response format")

// RecordValidationError represents an error that occurs when validating a
// transformed record.
type RecordValidationError struct {
	path string
	msg  string
}

// newRecordValidationError returns a new RecordValidationError.
func newRecordValidationError(path, msg string) error {
	return RecordValidationError{
		path: path,
		msg:  msg,
	}
}

func (err RecordValidationError) Error() string {
	return err.msg
}

func (err RecordValidationError) addIndexToPath(i int) RecordValidationError {
	err.path = "[" + strconv.Itoa(i) + "]." + err.path
	return err
}

func (err RecordValidationError) addNameToPath(name string) RecordValidationError {
	if err.path == "" {
		err.path = name
	} else if err.path[0] == '[' {
		err.path = name + err.path
	} else {
		err.path = name + "." + err.path
	}
	return err
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
	terms          terms
	int64AsString  bool
	datetimeFormat string
	dateFormat     string
	timeFormat     string
}

type terms struct {
	Null     string
	Elements string
	Type     func(t types.Type) string
}

// javaScriptDecoderOptions are the JavaScript's options used by the decoder.
var javaScriptDecoderOptions = decoderOptions{
	terms: terms{
		Null:     "null",
		Elements: "elements",
		Type:     toJavascriptType,
	},
	int64AsString:  true,
	datetimeFormat: "2006-01-02T15:04:05.000Z07:00",
	dateFormat:     "2006-01-02T15:04:05.000Z07:00",
	timeFormat:     "2006-01-02T15:04:05.000Z07:00",
}

// pythonDecoderOptions are the Python's options used by the decoder.
var pythonDecoderOptions = decoderOptions{
	terms: terms{
		Null:     "None",
		Elements: "items",
		Type:     toPythonType,
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
//   - ip: a String representing an IP number
//   - string: a String
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
//   - ip: a String representing an IP number
//   - json: if preserveJSON is false: true, false, a Number, a String, an
//     Array, or an Object; Otherwise a String representing a JSON value
//   - string: a String
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
//
// If the response JSON does not conform to the expected format, it returns the
// errInvalidResponseFormat error.
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
			return errInvalidResponseFormat
		}
		return err
	}
	if tok.Kind() != '{' {
		return errInvalidResponseFormat
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
			return errInvalidResponseFormat
		}
		msg := tok.String()
		if tok, err = d.readToken(); err != nil {
			return err
		}
		if tok.Kind() != '}' {
			return errInvalidResponseFormat
		}
		if _, err := d.readToken(); err != io.EOF {
			return errInvalidResponseFormat
		}
		return FunctionExecError{msg: msg}
	}
	if key != "records" {
		return errInvalidResponseFormat
	}
	// Parse the records.
	if tok, err = d.readToken(); err != nil {
		return err
	}
	if tok.Kind() != '[' {
		return errInvalidResponseFormat
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
			return errInvalidResponseFormat
		}
		// Read the key:
		tok, err = d.readToken()
		if err != nil {
			return err
		}
		if i == len(records) {
			return fmt.Errorf("core/transformers: expected %d results got more", len(records))
		}
		switch tok.String() {
		case "value":
			attributes, err := d.unmarshal(schema, preserveJSON, records[i].Purpose)
			if err != nil {
				if err == errInvalidResponseFormat {
					return err
				}
				if e, ok := err.(RecordValidationError); ok {
					e.msg = `property «` + e.path + `» ` + e.msg
					err = e
				}
				records[i].Attributes = nil
				records[i].Err = err
			} else {
				records[i].Attributes = attributes.(map[string]any)
			}
		case "error":
			tok, err := d.readToken()
			if err != nil {
				return err
			}
			if tok.Kind() != '"' {
				return errInvalidResponseFormat
			}
			records[i].Attributes = nil
			records[i].Err = RecordTransformationError{msg: fmt.Sprintf("%s: %s", language, tok.String())}
		default:
			return errInvalidResponseFormat
		}
		tok, err = d.readToken()
		if err != nil {
			return err
		}
		if tok.Kind() != '}' {
			return errInvalidResponseFormat
		}
		i++
	}
	if tok, err = d.readToken(); err != nil {
		return err
	}
	if tok.Kind() != '}' {
		return errInvalidResponseFormat
	}
	if _, err := d.readToken(); err != io.EOF {
		return errInvalidResponseFormat
	}
	if i < len(records) {
		return fmt.Errorf("core/transformers: expected %d results got %d", len(records), i)
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
		err = errInvalidResponseFormat
	} else if _, ok := err.(*json.SyntaxError); ok {
		err = errInvalidResponseFormat
	}
	return tok, err
}

// readValue reads a value.
// It returns the errInvalidResponseFormat error if the JSON source is not
// valid.
func (d decoder) readValue() (json.Value, error) {
	v, err := d.dec.ReadValue()
	if err == io.ErrUnexpectedEOF {
		err = errInvalidResponseFormat
	} else if _, ok := err.(*json.SyntaxError); ok {
		err = errInvalidResponseFormat
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
			return nil, newRecordValidationError("", fmt.Sprintf("has a value that is not of type «%s»", d.opts.terms.Type(t)))
		}
		min := t.MinElements()
		max := t.MaxElements()
		arr := []any{}
		for i := 0; d.peekKind() != ']'; i++ {
			if i == max {
				return nil, newRecordValidationError("", fmt.Sprintf("contains more than %d %s", max, d.opts.terms.Elements))
			}
			elem, err := d.unmarshal(t.Elem(), preserveJSON, purpose)
			if err != nil {
				if e, ok := err.(RecordValidationError); ok {
					err = e.addIndexToPath(i)
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
			return nil, newRecordValidationError("", fmt.Sprintf("contains less than %d %s", min, d.opts.terms.Elements))
		}
		if t.Unique() {
			for i, elem := range arr {
				for _, item2 := range arr[i+1:] {
					if elem == item2 {
						return nil, newRecordValidationError("", "contains a duplicated value")
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
			var tProperties types.Properties
			if t.Valid() {
				tProperties = t.Properties()
			}
			for {
				if d.peekKind() == '}' {
					break
				}
				// Read the property's name.
				tok, err := d.readToken()
				if err != nil {
					return nil, err
				}
				const errNotInSchema = "is not part of output schema; rename or remove it in the transformation function"
				name := tok.String()
				if !t.Valid() {
					if !types.IsValidPropertyName(name) {
						return nil, newRecordValidationError(name, errNotInSchema)
					}
					return nil, newRecordValidationError(name, errNotInSchema)
				}
				p, ok := tProperties.ByName(name)
				if !ok {
					if !types.IsValidPropertyName(name) {
						return nil, newRecordValidationError(name, errNotInSchema)
					}
					return nil, newRecordValidationError(name, errNotInSchema)
				}
				// Read the property's value.
				var value any
				if d.peekKind() == 'n' {
					if _, err := d.readToken(); err != nil {
						return nil, err
					}
					if !p.Nullable {
						if p.Type.Kind() != types.JSONKind || preserveJSON {
							if purpose == Import {
								continue
							}
							return nil, newRecordValidationError(p.Name, fmt.Sprintf("cannot be «%s», but it is set to «%s»", d.opts.terms.Null, d.opts.terms.Null))
						}
						value = json.Value("null")
					}
				} else {
					value, err = d.unmarshal(p.Type, preserveJSON, purpose)
					if err != nil {
						if e, ok := err.(RecordValidationError); ok {
							err = e.addNameToPath(name)
						}
						return nil, err
					}
				}
				o[name] = value
			}
			if t.Valid() {
				switch purpose {
				case Create:
					for _, p := range tProperties.All() {
						if !p.CreateRequired {
							continue
						}
						if _, ok := o[p.Name]; !ok {
							return nil, newRecordValidationError(p.Name, "is missing but it is required for creation")
						}
					}
				case Update:
					for _, p := range tProperties.All() {
						if !p.UpdateRequired {
							continue
						}
						if _, ok := o[p.Name]; !ok {
							return nil, newRecordValidationError(p.Name, "is missing but it is required for update")
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
		return nil, newRecordValidationError("", fmt.Sprintf("has a value that is not of type «%s»", d.opts.terms.Type(t)))
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
	case types.StringKind:
		if v.Kind() == '"' {
			s := d.unquoteString(v)
			if values := t.Values(); values != nil {
				if !slices.Contains(values, s) {
					return nil, newRecordValidationError("", "is not one of the allowed values")
				}
				return s, nil
			} else if re := t.Pattern(); re != nil {
				if !re.MatchString(s) {
					return nil, newRecordValidationError("", fmt.Sprintf("does not match «/%s/»", quoteRegExpr(re)))
				}
				return s, nil
			} else {
				if n, ok := t.MaxLength(); ok && utf8.RuneCountInString(s) > n {
					return nil, newRecordValidationError("", fmt.Sprintf("exceeds the %d-char limit", n))
				}
				if n, ok := t.MaxBytes(); ok && utf8.RuneCountInString(s) > n {
					return nil, newRecordValidationError("", fmt.Sprintf("exceeds the %d-byte limit", n))
				}
				return s, nil
			}
		}
	case types.BooleanKind:
		if v.Kind() == 'f' {
			return false, nil
		} else if v.Kind() == 't' {
			return true, nil
		}
	case types.IntKind:
		if t.IsUnsigned() {
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
					min, max := t.UnsignedRange()
					if n < min {
						return nil, newRecordValidationError("", fmt.Sprintf("is less than %d", min))
					}
					if n > max {
						return nil, newRecordValidationError("", fmt.Sprintf("is greater than %d", max))
					}
					return uint(n), nil
				}
			}
		} else {
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
					min, max := t.IntRange()
					if n < min {
						return nil, newRecordValidationError("", fmt.Sprintf("is less than %d", min))
					}
					if n > max {
						return nil, newRecordValidationError("", fmt.Sprintf("is greater than %d", max))
					}
					return int(n), nil
				}
			}
		}
	case types.FloatKind:
		switch v.Kind() {
		case '0':
			if n, err := strconv.ParseFloat(string(v), t.BitSize()); err == nil {
				min, max := t.FloatRange()
				if n < min {
					return nil, newRecordValidationError("", fmt.Sprintf("is less than %g", min))
				}
				if n > max {
					return nil, newRecordValidationError("", fmt.Sprintf("is greater than %g", max))
				}
				return n, nil
			}
		case '"':
			if t.IsReal() {
				return nil, newRecordValidationError("", fmt.Sprintf("is %s, which is not a valid real number", string(v)))
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
				min, max := t.DecimalRange()
				if n.Less(min) {
					return nil, newRecordValidationError("", fmt.Sprintf("is less than %s", min))
				}
				if n.Greater(max) {
					return nil, newRecordValidationError("", fmt.Sprintf("is greater than %s", max))
				}
				return n, nil
			} else if err == decimal.ErrRange {
				return nil, newRecordValidationError("", fmt.Sprintf("has a value which is not in range of «decimal(%d, %d)»", t.Precision(), t.Scale()))
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
					return nil, newRecordValidationError("", "year is not in range [1,9999]")
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
				return nil, newRecordValidationError("", "does not contain valid JSON")
			}
			return json.Value(data), nil
		}
	case types.IPKind:
		if v.Kind() == '"' {
			if ip, err := netip.ParseAddr(d.unquoteString(v)); err == nil {
				return ip.String(), nil
			}
		}
	}
	return nil, newRecordValidationError("", fmt.Sprintf("has a value that is not of type «%s»", d.opts.terms.Type(t)))
}

// unquoteString unquote a JSON string.
func (d decoder) unquoteString(v []byte) string {
	return string(json.Value(v).AppendUnquote(nil))
}

// quoteRegExpr quotes a regular expression between "/" and "/".
func quoteRegExpr(re *regexp.Regexp) string {
	return strings.ReplaceAll(strings.ReplaceAll(re.String(), `/`, `\/`), `»`, `≫`)
}

// toJavascriptType returns the JavaScript type corresponding to the given type
// t, to be used in error messages to represent the type in JavaScript.
func toJavascriptType(t types.Type) string {
	switch t.Kind() {
	case types.StringKind:
		return "string"
	case types.BooleanKind:
		return "boolean"
	case types.IntKind:
		if t.BitSize() == 64 {
			return "bigint"
		}
		return "number"
	case types.FloatKind:
		return "number"
	case types.DecimalKind:
		return "string"
	case types.DateTimeKind, types.DateKind, types.TimeKind:
		return "Date"
	case types.YearKind:
		return "number"
	case types.UUIDKind, types.JSONKind, types.IPKind:
		return "string"
	case types.ArrayKind:
		et := toJavascriptType(t.Elem())
		return "array of " + et
	case types.ObjectKind:
		return "object"
	case types.MapKind:
		et := toJavascriptType(t.Elem())
		return "object with " + et + " values"
	default:
		panic("schema contains unknown property kind " + t.Kind().String())
	}
}

// toPythonType returns the Python type corresponding to the given type t, to be
// used in error messages to represent the type in Python.
func toPythonType(t types.Type) string {
	switch t.Kind() {
	case types.StringKind:
		return "str"
	case types.BooleanKind:
		return "bool"
	case types.IntKind:
		return "int"
	case types.FloatKind:
		return "float"
	case types.DecimalKind:
		return "decimal.Decimal"
	case types.DateTimeKind:
		return "datetime.datetime"
	case types.DateKind:
		return "datetime.date"
	case types.TimeKind:
		return "datetime.time"
	case types.YearKind:
		return "int"
	case types.UUIDKind:
		return "uuid.UUID"
	case types.JSONKind, types.IPKind:
		return "str"
	case types.ArrayKind:
		et := toPythonType(t.Elem())
		return "list[" + et + "]"
	case types.ObjectKind:
		return "dict"
	case types.MapKind:
		et := toPythonType(t.Elem())
		return "dict[str, " + et + "]"
	default:
		panic("schema contains unknown property kind " + t.Kind().String())
	}
}
