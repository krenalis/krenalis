// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"strconv"
	"sync"

	"github.com/meergo/meergo/tools/decimal"
	"github.com/meergo/meergo/tools/json/internal/json"
	"github.com/meergo/meergo/tools/json/internal/json/jsontext"
)

var (
	_ Marshaler   = Value{}
	_ Unmarshaler = (*Value)(nil)
)

// ErrRange is returned by the [Value.Decimal], [Value.Float], [Value.Int], and
// [Value.Uint] methods when the number is out of range.
var ErrRange = errors.New("out of range")

// NotExistError is returned by the Lookup method when the specified path does
// not exist.
type NotExistError struct {
	Index int
	Kind  Kind
}

func (err NotExistError) Error() string {
	return "path does not exist"
}

// Value is a JSON-encoded value.
//
// All methods of Value do not modify it. They also assume that it contains
// valid JSON, possibly with leading or trailing whitespace. If not, the
// behavior is undefined.
type Value []byte

// AppendUnquote writes the unquoted value of v into dst, and returns the
// extended buffer. v may contain leading and trailing JSON whitespace.
// It panics if v is not a string.
func (v Value) AppendUnquote(dst []byte) []byte {
	if !v.IsString() {
		panic("expected string")
	}
	dst, _ = jsontext.AppendUnquote(dst, v)
	return dst
}

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
// number or if v does not fit in the provided precision and scale (in which
// case it returns [ErrRange]).
//
// As a special case, if precision is 0, it only checks that the decimal's
// precision is in range [1, decimal.MaxPrecision], and the decimal's scale is
// in range [decimal.MinScale, decimal.MaxScale].
func (v Value) Decimal(precision, scale int) (decimal.Decimal, error) {
	n, err := decimal.Parse(TrimSpace(v), precision, scale)
	if err == decimal.ErrRange {
		err = ErrRange
	}
	return n, err
}

// Elements returns an iterator over the elements of an array.
// It panics if v is not an array.
func (v Value) Elements() iter.Seq2[int, Value] {
	if !v.IsArray() {
		panic("expected array")
	}
	return func(yield func(int, Value) bool) {
		dec := getDecoder(v)
		defer putDecoder(dec)
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
// bit size. It returns an error if bitSize is neither 32 nor 64, or if v is not
// a JSON number, or if it is out of range (in which case it returns
// [ErrRange]).
func (v Value) Float(bitSize int) (float64, error) {
	f, err := strconv.ParseFloat(string(TrimSpace(v)), bitSize)
	if err != nil {
		if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
			return 0, ErrRange
		}
		return 0, err
	}
	return f, nil
}

var formatText = []byte("json.Value{, }")

// Format implements the fmt.Formatter interface.
func (v Value) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('#') {
		_, _ = f.Write(formatText[0:11])
		for i, b := range v {
			if i > 0 {
				_, _ = f.Write(formatText[11:13])
			}
			_, _ = fmt.Fprintf(f, "%#02x", b)
		}
		_, _ = f.Write(formatText[13:14])
		return
	}
	s := fmt.FormatString(f, verb)
	_, _ = fmt.Fprintf(f, s, []byte(v))
}

// Get returns the value at the specified path in v and true, or nil and false
// if the path does not exist.
func (v Value) Get(path []string) (Value, bool) {
	v, err := v.Lookup(path)
	return v, err == nil
}

var dotZero = []byte(".0")

// Int returns the integer value for a JSON number. It returns an error if v is
// not a valid JSON number, or if it does not represent an integer, or if it is
// out of range (in which case it returns [ErrRange]).
//
// As a special case, an integer followed by ".0" is considered to represent an
// integer; for instance, "1" and "1.0" are both valid.
func (v Value) Int() (int, error) {
	n, err := strconv.ParseInt(string(bytes.TrimSuffix(TrimSpace(v), dotZero)), 10, 64)
	if err != nil {
		if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
			return 0, ErrRange
		}
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

// IsEmpty reports whether v is an empty string, array, or object.
func (v Value) IsEmpty() bool {
	ts := Value(TrimLeftSpace(v))
	switch ts[0] {
	case '"':
		return ts[1] == '"'
	case '[':
		return TrimLeftSpace(ts[1:])[0] == ']'
	case '{':
		return TrimLeftSpace(ts[1:])[0] == '}'
	}
	return false
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

// lookupTable is used by both the Lookup method and the TrimSpace function.
var lookupTable = [256]uint8{'\t': 1, '\n': 1, '\r': 1, ' ': 1, ':': 2}

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

// Lookup returns the value at the specified path in v as a sub-slice of v.
//
// If any part of the path does not exist, it returns a NotExistError. The error
// contains the Index, which indicates the position in the path where the lookup
// failed, and Kind, representing the kind of the JSON value where the property
// was expected but not found.
func (v Value) Lookup(path []string) (Value, error) {
	dec := getDecoder(v)
	defer putDecoder(dec)
	var tok jsontext.Token
	for i, name := range path {
		if tok, _ = dec.ReadToken(); tok.Kind() != '{' {
			return nil, NotExistError{Index: i, Kind: Kind(tok.Kind())}
		}
		for {
			tok, _ = dec.ReadToken()
			if tok.Kind() == '}' {
				return nil, NotExistError{Index: i, Kind: Object}
			}
			if tok.String() == name {
				break
			}
			_ = dec.SkipValue()
		}
	}
	start := dec.InputOffset() + 1
	for lookupTable[v[start]] != 0 {
		start++
	}
	_ = dec.SkipValue()
	end := dec.InputOffset()
	return v[start:end], nil
}

// MarshalJSON returns v as the JSON encoding of v.
func (v Value) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	return v, nil
}

// NumElement returns the number of elements in an array.
// It panics if v is not an array.
func (v Value) NumElement() int {
	if !v.IsArray() {
		panic("expected array")
	}
	dec := getDecoder(v)
	defer putDecoder(dec)
	_, _ = dec.ReadToken()
	for i := 0; ; i++ {
		err := dec.SkipValue()
		if err != nil {
			return i
		}
	}
}

// NumProperty returns the number of properties in an object.
// It panics if v is not an object.
func (v Value) NumProperty() int {
	if !v.IsObject() {
		panic("expected object")
	}
	dec := getDecoder(v)
	defer putDecoder(dec)
	_, _ = dec.ReadToken()
	for i := 0; ; i++ {
		err := dec.SkipValue()
		if err != nil {
			return i / 2
		}
	}
}

// Properties returns an iterator over the key-value pairs of an object.
// It panics if v is not an object.
func (v Value) Properties() iter.Seq2[string, Value] {
	if !v.IsObject() {
		panic("expected object")
	}
	return func(yield func(string, Value) bool) {
		var b []byte
		dec := getDecoder(v)
		defer putDecoder(dec)
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
// error if v is not a valid JSON number, or if it does not represent an
// integer, or if it is out of range (in which case it returns [ErrRange]).
//
// As a special case, an integer followed by ".0" is considered to represent an
// integer; for instance, "1" and "1.0" are both valid.
func (v Value) Uint() (uint, error) {
	n, err := strconv.ParseUint(string(bytes.TrimSuffix(TrimSpace(v), dotZero)), 10, 64)
	if err != nil {
		if e := err.(*strconv.NumError); e.Err == strconv.ErrRange {
			return 0, ErrRange
		}
		return 0, err
	}
	return uint(n), nil
}

// Unmarshal unmarshals v into the value pointed to by out.
// It returns an error if out is nil or is not a pointer, or if v cannot be
// decoded into out.
func (v Value) Unmarshal(out any) error {
	err := json.Unmarshal(v, out)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}

	}
	return err
}

// UnmarshalJSON sets *v to a copy of the data, excluding leading and trailing
// whitespace. If data does not contain valid JSON, it does nothing and returns
// ErrInvalidJSON. v must not be nil, and *v must not be empty.
func (v *Value) UnmarshalJSON(data []byte) error {
	if v == nil {
		return errors.New("UnmarshalJSON on nil pointer")
	}
	if len(*v) > 0 {
		return errors.New("UnmarshalJSON on non-empty value")
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

// valueDecoder is used by the Elements, Properties, and Lookup methods.
// It combines a byte buffer and a JSON decoder to facilitate decoding
// of JSON-encoded data from a Value.
type valueDecoder struct {
	b bytes.Buffer
	jsontext.Decoder
}

// decPool is a pool of *valueDecoder values.
var decPool sync.Pool

func init() {
	decPool.New = func() any {
		return &valueDecoder{}
	}
}

// getDecoder retrieves a valueDecoder from the pool and initializes it to
// decode the provided Value.
func getDecoder(v Value) *valueDecoder {
	dec := decPool.Get().(*valueDecoder)
	dec.b = *bytes.NewBuffer(v)
	dec.Reset(&dec.b)
	return dec
}

// putDecoder returns the provided decoder to the pool for future reuse.
func putDecoder(dec *valueDecoder) {
	dec.b = bytes.Buffer{}
	decPool.Put(dec)
}
