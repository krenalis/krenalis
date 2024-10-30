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
	"iter"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/meergo/meergo/decimal"
	"github.com/meergo/meergo/types"

	"github.com/meergo/meergo/json/internal/json"
	"github.com/meergo/meergo/json/internal/json/jsontext"
)

var (
	_ Marshaler   = Value{}
	_ Unmarshaler = (*Value)(nil)
)

// ErrInvalidJSON is returned when an argument is not valid JSON, or is not
// UTF-8 encoded.
var ErrInvalidJSON = errors.New("invalid JSON")

// NotExistError is returned by the Lookup method when the specified path does
// not exist.
type NotExistError struct {
	Index int
	Kind  Kind
}

func (err NotExistError) Error() string {
	return "path does not exist"
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

// Decimal returns v as a decimal.Decimal. If v is not a number, it returns
// decimal.ErrSyntax. If v does not fit in the provided precision and scale, it
// returns decimal.ErrOutOfRange.
//
// As a special case, if precision is 0, it only checks that the decimal's
// precision is in range [1, decimal.MaxPrecision], and the decimal's scale is
// in range [decimal.MinScale, decimal.MaxScale].
func (v Value) Decimal(precision, scale int) (decimal.Decimal, error) {
	return decimal.Parse(TrimSpace(v), precision, scale)
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
// bit size. It returns an error if v is not a JSON number, is out of range, or
// bitSize is neither 32 nor 64.
func (v Value) Float(bitSize int) (float64, error) {
	return strconv.ParseFloat(string(TrimSpace(v)), bitSize)
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
// If any part of the path does not exist, it returns a NotFoundError. The error
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

// Unmarshal unmarshals v into the value pointed by out.
// It returns an error if out is nil or is not a pointer.
func (v Value) Unmarshal(out any) error {
	err := json.Unmarshal(v, out)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
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
