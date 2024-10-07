//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"bytes"
	"encoding/json"
	"errors"
	"iter"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-json-experiment/json/jsontext"
	"github.com/shopspring/decimal"
	"github.com/tidwall/gjson"
)

var _ json.Marshaler = (*Value)(nil)
var _ json.Unmarshaler = (*Value)(nil)

// ErrInvalidJSON is returned when an argument is not valid JSON, or is not
// UTF-8 encoded.
var ErrInvalidJSON = errors.New("invalid JSON")

// Kind represents a specific kind of JSON value.
type Kind byte

const (
	Null   Kind = 'n'
	True   Kind = 't'
	False  Kind = 'f'
	String Kind = '"'
	Number Kind = '0'
	Object Kind = '{'
	Array  Kind = '['
)

// String returns the name of k.
func (k Kind) String() string {
	switch k {
	case Null:
		return "null"
	case True:
		return "true"
	case False:
		return "false"
	case String:
		return "string"
	case Number:
		return "number"
	case Object:
		return "object"
	case Array:
		return "array"
	}
	panic("unexpected kind")
}

// Value is a JSON-encoded value.
//
// All methods of Value assume that it contains valid JSON, potentially with
// leading and trailing JSON spaces; otherwise, the behavior is undefined.
type Value []byte

// Bytes returns a copy of v as []byte. If v is a string, it returns it
// unquoted. If v is an array or an object it returns nil.
func (v Value) Bytes() []byte {
	ts := Value(TrimSpace(v))
	switch ts.Kind() {
	case String:
		dst, _ := jsontext.AppendUnquote(nil, ts)
		return dst
	case Number, True, False, Null:
		return bytes.Clone(ts)
	}
	return nil
}

// Bool reports whether v is the boolean value true.
func (v Value) Bool() bool {
	return v.Kind() == True
}

// Decimal returns v as a decimal.Decimal. It returns an error if v is not a
// number or cannot be represented as a Decimal.
func (v Value) Decimal() (decimal.Decimal, error) {
	return decimal.NewFromString(string(TrimSpace(v)))
}

// Elements returns an iterator over the elements of an array.
// It panics if v is not an array.
func (v Value) Elements() iter.Seq2[int, Value] {
	if !v.IsArray() {
		panic("expected array")
	}
	return func(yield func(int, Value) bool) {
		dec := jsontext.NewDecoder(bytes.NewReader(v))
		_, _ = dec.ReadToken()
		for i := 0; ; i++ {
			v, err := dec.ReadValue()
			if err != nil {
				break
			}
			if !yield(i, Value(v)) {
				return
			}
		}
	}
}

// Float returns the floating-point value for a JSON number with the provided
// bit size. It returns an error if v is not a JSON number, is out of range, or
// bitSize is neither 32 nor 64.
func (v Value) Float(bitSize int) (float64, error) {
	return strconv.ParseFloat(string(TrimSpace(v)), bitSize)
}

// Properties returns an iterator over the key-value pairs of an object.
// It panics if v is not an object.
func (v Value) Properties() iter.Seq2[string, Value] {
	if !v.IsObject() {
		panic("expected object")
	}
	return func(yield func(string, Value) bool) {
		var b []byte
		dec := jsontext.NewDecoder(bytes.NewReader(v))
		_, _ = dec.ReadToken()
		for {
			k, err := dec.ReadValue()
			if err != nil {
				break
			}
			b, _ = jsontext.AppendUnquote(b, k)
			v, _ := dec.ReadValue()
			if !yield(string(b), Value(v)) {
				return
			}
			b = b[0:0]
		}
	}
}

// Kind returns the kind of v.
func (v Value) Kind() Kind {
	i := 0
	for {
		c := Kind(v[i])
		switch c {
		case Object, Array, Null, String, Number, True, False:
			return c
		default:
			if '1' <= c && c <= '9' || c == '-' {
				return Number
			}
		}
		i++
	}
}

var dotZero = []byte(".0")

// Int returns the integer value for a JSON number. It returns an error if v is
// not a valid JSON number, does not represent an integer, or is out of range.
// As a special case, an integer followed by ".0" is considered valid;
// for instance, "1" and "1.0" are both valid.
func (v Value) Int() (int, error) {
	n, err := strconv.ParseInt(string(bytes.TrimSuffix(TrimSpace(v), dotZero)), 10, 64)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// IsArray reports whether v represents a JSON array.
func (v Value) IsArray() bool {
	return v.Kind() == Array
}

// IsBool reports whether v represents a JSON bool.
func (v Value) IsBool() bool {
	k := v.Kind()
	return k == False || k == True
}

// IsFalse reports whether v represents a JSON false.
func (v Value) IsFalse() bool {
	return v.Kind() == False
}

// IsNull reports whether v represents a JSON null.
func (v Value) IsNull() bool {
	return v.Kind() == Null
}

// IsNumber reports whether v represents a JSON number.
func (v Value) IsNumber() bool {
	return v.Kind() == Number
}

// IsObject reports whether v represents a JSON object.
func (v Value) IsObject() bool {
	return v.Kind() == Object
}

// IsString reports whether v represents a JSON string.
func (v Value) IsString() bool {
	return v.Kind() == String
}

// IsTrue reports whether v represents a JSON true.
func (v Value) IsTrue() bool {
	return v.Kind() == True
}

// Lookup retrieves the JSON value for the specified path. The path uses dot
// syntax, such as "address.city". If the value exists in v, it returns the
// value and a boolean indicating true; otherwise it returns nil and false.
func (v Value) Lookup(path string) (Value, bool) {
	if !v.IsObject() {
		panic("expected object")
	}
	var b strings.Builder
	for _, c := range path {
		if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' || c == '.' {
			b.WriteRune(c)
		} else {
			b.WriteByte('\\')
			b.WriteRune(c)
		}
	}
	path = b.String()
	r := gjson.GetBytes(v, path)
	if !r.Exists() {
		return nil, false
	}
	return []byte(r.Raw), true
}

// MarshalJSON returns v as the JSON encoding of v.
func (v Value) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return v, nil
}

// String returns a string representation of v. If v is a string, it returns it
// unquoted. If v is an array or an object it returns an empty string.
func (v Value) String() string {
	ts := Value(TrimSpace(v))
	switch ts.Kind() {
	case String:
		dst, _ := jsontext.AppendUnquote(nil, ts)
		return string(dst)
	case Number, True, False, Null:
		return string(ts)
	}
	return ""
}

// Uint returns the unsigned integer value for a JSON number. It returns an
// error if v is not a valid JSON number, does not represent an integer, or is
// out of range. As a special case, an integer followed by ".0" is considered
// valid; for instance, "1" and "1.0" are both valid.
func (v Value) Uint() (uint, error) {
	n, err := strconv.ParseUint(string(bytes.TrimSuffix(TrimSpace(v), dotZero)), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(n), nil
}

// UnmarshalJSON sets *v to a copy of the data, excluding leading and trailing
// whitespace. If data does not contain valid JSON, it does nothing and returns
// ErrInvalidJSON.
func (v *Value) UnmarshalJSON(data []byte) error {
	if v == nil {
		return errors.New("UnmarshalJSON on nil pointer")
	}
	if !Valid(data) {
		return ErrInvalidJSON
	}
	data = TrimSpace(data)
	b := make([]byte, len(data))
	copy(b, data)
	*v = b
	return nil
}

// Compact returns a copy of data with all insignificant whitespace removed. If
// data is already compact, it returns the original data unchanged. If data does
// not contain valid JSON, it returns nil and ErrInvalidJSON.
func Compact(data []byte) ([]byte, error) {
	v := jsontext.Value(data)
	if !utf8.Valid(data) {
		return nil, ErrInvalidJSON
	}
	err := v.Compact()
	if err != nil {
		return nil, ErrInvalidJSON
	}
	return v, nil
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

var space = [256]uint8{'\t': 1, '\n': 1, '\r': 1, ' ': 1}

// TrimSpace returns a subslice of data with all leading and trailing whitespace
// removed. data must contain valid JSON.
func TrimSpace(data []byte) []byte {
	i, j := 0, len(data)-1
	for ; space[data[i]] == 1; i++ {
	}
	for ; space[data[j]] == 1; j-- {
	}
	return data[i : j+1]
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
