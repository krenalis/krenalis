// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"

	"github.com/meergo/meergo/core/json/internal/json"
	"github.com/meergo/meergo/core/json/internal/json/jsontext"
)

// ErrInvalidJSON is returned when an argument is not valid JSON, or is not
// UTF-8 encoded.
var ErrInvalidJSON = errors.New("invalid JSON")

// ErrInvalidUTF8 is returned when an argument is not UTF-8 encoded.
var ErrInvalidUTF8 = errors.New("invalid UTF-8")

// A SyntaxError is a description of a JSON syntax error that occurred during
// the encoding or decoding of JSON, in accordance with the JSON grammar.
type SyntaxError struct {
	err    error
	offset int64
}

// NewSyntaxError returns a new SyntaxError with the provided error and offset.
func NewSyntaxError(err error, offset int64) *SyntaxError {
	return &SyntaxError{err: err, offset: offset}
}

func (err *SyntaxError) ByteOffset() int64 {
	if err, ok := err.err.(*jsontext.SyntacticError); ok {
		return err.ByteOffset
	}
	return err.offset
}

func (err *SyntaxError) Error() string {
	if err, ok := err.err.(*jsontext.SyntacticError); ok {
		return err.Err.Error()
	}
	return err.err.Error()
}

// Canonicalize returns a copy of data with canonical JSON formatting
// (RFC 8785). It normalizes numbers, removes whitespace, and sorts object keys.
// If data does not contain valid JSON, it returns nil and ErrInvalidJSON.
func Canonicalize(data []byte) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, ErrInvalidJSON
	}
	v := jsontext.Value(slices.Clone(data))
	err := v.Canonicalize()
	if err != nil {
		return nil, ErrInvalidJSON
	}
	return v, nil
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

// Decode deserializes JSON read from r into the value pointed to by out.
// It returns an error if out is nil or is not a pointer.
func Decode(r io.Reader, out any) error {
	err := json.UnmarshalRead(r, out)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
}

// Encode writes to out the JSON encoding of v.
func Encode(out io.Writer, v any) error {
	err := json.MarshalWrite(out, v, json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
}

// Indent returns a copy of data with the JSON code indented and object keys
// sorted. Each element in a JSON object or array begins on a new line with the
// specified prefix followed by copies of the indent string, added according to
// nesting depth. The returned data does not start or end with the prefix or any
// indentation.
//
// Example usage:
//
//	data, err = json.Indent(data, "", "\t")
//
// If data does not contain valid JSON, it returns nil and ErrInvalidJSON.
// It panics if prefix or indent strings do not contain only spaces or tabs
// (' ' or '\t').
func Indent(data []byte, prefix, indent string) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, ErrInvalidJSON
	}
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
	err := json.MarshalEncode(enc, v,
		json.Deterministic(true),
		json.FormatNilMapAsNull(true),
		json.FormatNilSliceAsNull(true),
	)
	if err != nil {
		return nil, err
	}
	buf.Truncate(buf.Len() - 1)
	return slices.Clone(buf.Bytes()), nil
}

// Marshal encodes the given data.
func Marshal(data any) (Value, error) {
	val, err := json.Marshal(data)
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

// Quote returns a double-quoted JSON string literal representing s. It returns
// ErrInvalidUTF8 if s is not valid UTF-8 encoded.
func Quote(s []byte) ([]byte, error) {
	d, err := jsontext.AppendQuote(nil, s)
	if err != nil {
		return nil, ErrInvalidUTF8
	}
	return d, nil
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

// TrimLeftSpace returns a subslice of data with all leading whitespace removed.
// data must contain valid JSON.
func TrimLeftSpace(data []byte) []byte {
	i := 0
	for ; lookupTable[data[i]] == 1; i++ {
	}
	return data[i:]
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

// Unmarshal decodes the JSON-encoded data into the value pointed to by out.
// It returns a [SyntaxError] if the input data is not valid JSON. It also
// returns an error if out is nil or is not a pointer, or if data cannot be
// decoded into out.
//
// Calling Unmarshal(data, out) is equivalent to calling
// Value(data).Unmarshal(out), with the key difference that the latter assumes
// data is valid JSON.
func Unmarshal(data []byte, out any) error {
	err := json.Unmarshal(data, out)
	if _, ok := err.(*jsontext.SyntacticError); ok {
		return &SyntaxError{err: err}
	}
	return err
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

// Valid reports whether data is a valid JSON encoding and properly encoded in
// UTF-8. For detailed error reporting, see the Validate function.
func Valid(data []byte) bool {
	return jsontext.Value(data).IsValid()
}

// Validate checks data and returns a *SyntaxError if it is not valid JSON or is
// not properly encoded in UTF-8. For a simple boolean check, see the Valid
// function.
func Validate(data []byte) error {
	dec := getDecoder(data)
	defer putDecoder(dec)
	err := dec.SkipValue()
	if err != nil {
		if err == io.EOF {
			return &SyntaxError{err: errors.New("content is empty")}
		}
		if _, ok := err.(*jsontext.SyntacticError); ok {
			return &SyntaxError{err: err}
		}
		return err
	}
	if dec.PeekKind() == 0 {
		return nil
	}
	offset := dec.InputOffset()
	tok, err := dec.ReadToken()
	if err != nil {
		return &SyntaxError{err: err, offset: offset}
	}
	err = &SyntaxError{
		err:    fmt.Errorf("invalid token '%s' after top-level value", tok),
		offset: offset,
	}
	return err
}
