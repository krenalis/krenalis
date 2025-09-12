//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"io"

	"github.com/meergo/meergo/core/json/internal/json"
	"github.com/meergo/meergo/core/json/internal/json/jsontext"
)

type Decoder struct {
	dec jsontext.Decoder
}

// NewDecoder returns a new decoder reading from r.
//
// If r is a [bytes.Buffer], the decoder reads directly from it without
// allocating a new buffer. The decoder does not modify the buffer, and writes to
// the buffer must not occur while the decoder is in use.
func NewDecoder(r io.Reader) *Decoder {
	var d Decoder
	d.dec.Reset(r)
	return &d
}

// ByteOffset returns the byte offset of the next byte after the last returned
// token or value. Due to internal buffering, the actual bytes read from the
// underlying [io.Reader] may exceed this offset.
func (d *Decoder) ByteOffset() int64 {
	return d.dec.InputOffset()
}

// Decode reads the next value, unmarshals it into the value pointed to by out,
// and advances the read offset. If the read value cannot be stored into out, it
// returns an error.
func (d *Decoder) Decode(out any) error {
	return json.UnmarshalDecode(&d.dec, out)
}

// IsKey returns true if the last token read is a key in an object.
func (d *Decoder) IsKey() bool {
	k, n := d.dec.StackIndex(d.dec.StackDepth())
	return k == '{' && n%2 == 1
}

// PeekKind returns the type of the next token without advancing the read
// offset. It returns Invalid if there are no more tokens.
func (d *Decoder) PeekKind() Kind {
	return Kind(d.dec.PeekKind())
}

// ReadToken reads and returns the next [Token], advancing the read offset. The
// returned token remains valid only until the next call to Peek, Read, or Skip.
// Returns [io.EOF] if there are no more tokens.
func (d *Decoder) ReadToken() (Token, error) {
	tok, err := d.dec.ReadToken()
	if err != nil {
		if _, ok := err.(*jsontext.SyntacticError); ok {
			err = &SyntaxError{err: err}
		}
		return Token{}, err
	}
	return Token{tok}, nil
}

// ReadValue reads and returns the next value, advancing the read offset. The
// value is stripped of leading and trailing whitespace and contains the exact
// bytes of the input.
//
// The returned value remains valid only until the next call to Peek, Read, or
// Skip, and must not be mutated while the Decoder is in use. Returns [io.EOF]
// if there are no more values.
func (d *Decoder) ReadValue() (Value, error) {
	v, err := d.dec.ReadValue()
	if err != nil {
		if _, ok := err.(*jsontext.SyntacticError); ok {
			err = &SyntaxError{err: err}
		}
		return Value{}, err

	}
	return Value(v), nil
}

// Reset resets the Decoder, allowing it to read from the specified io.Reader
// as a fresh input source.
func (d *Decoder) Reset(r io.Reader) {
	d.dec.Reset(r)
}

// SkipOut skips out of the current object or array and advances the read offset
// to the byte immediately after. If not in an object or array, it advances to
// the end of the input and returns io.EOF.
func (d *Decoder) SkipOut() error {
	var err error
	for err == nil {
		err = d.dec.SkipValue()
	}
	tok, _ := d.dec.ReadToken()
	if tok.Kind() == 0 {
		if _, ok := err.(*jsontext.SyntacticError); ok {
			err = &SyntaxError{err: err}
		}
		return err
	}
	return nil
}

// SkipToken discards the next token and advances the read offset.
func (d *Decoder) SkipToken() error {
	_, err := d.dec.ReadToken()
	if _, ok := err.(*jsontext.SyntacticError); ok {
		err = &SyntaxError{err: err}
	}
	return err
}

// SkipValue discards the next value and advances the read offset. It is more
// efficient than calling [Decoder.ReadValue] when the value is not needed.
func (d *Decoder) SkipValue() error {
	err := d.dec.SkipValue()
	if _, ok := err.(*jsontext.SyntacticError); ok {
		err = &SyntaxError{err: err}
	}
	return err
}
