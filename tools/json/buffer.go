// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package json

import (
	"bytes"
	"slices"

	"github.com/krenalis/krenalis/tools/json/internal/json"
	"github.com/krenalis/krenalis/tools/json/internal/json/jsontext"
)

// Buffer embeds a bytes.Buffer, providing all its methods along with
// additional functionality. It includes [Buffer.Encode], [Buffer.EncodeIndent],
// [Buffer.EncodeKeyValue], [Buffer.EncodeQuoted], and [Buffer.EncodeSorted] for
// appending JSON-encoded values to the buffer. Additionally, the [Buffer.Value]
// method allows copying the unread portion of the buffer as a Value.
//
// The zero value of Buffer represents an empty buffer, ready for use.
type Buffer struct {
	buffer
	enc         jsontext.Encoder
	text        textMarshaler
	kvOff       int
	initialized bool
	indent      bool
	_           noCopy
}

// NewBuffer returns a new buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Encode appends the JSON encoding of value to the buffer.
// It returns an error if the value cannot be encoded as JSON.
func (b *Buffer) Encode(value any) error {
	if !b.initialized || b.indent {
		b.enc.Reset(&b.buffer)
		b.initialized = true
		b.indent = false
	}
	err := json.MarshalEncode(&b.enc, value, json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
	if err != nil {
		return err
	}
	b.buffer.Truncate(b.Len() - 1)
	return nil
}

// EncodeIndent is like [Encode], but writes the resulting JSON with
// indentation. It also sorts the object keys. Each element in a JSON object or
// array begins on a new line with the specified prefix, followed by copies of
// the indent string, added according to the nesting depth. The returned JSON
// does not start or end with the prefix or any indentation.
//
// Example usage:
//
//	err = buf.EncodeIndent(in, "", "\t")
//
// It panics if the prefix or indent strings contain characters other than
// spaces or tabs (' ' or '\t').
func (b *Buffer) EncodeIndent(value any, prefix, indent string) error {
	if !b.initialized || !b.indent {
		b.enc.Reset(&b.buffer, jsontext.WithIndentPrefix(prefix), jsontext.WithIndent(indent))
		b.initialized = true
		b.indent = true
	}
	err := json.MarshalEncode(&b.enc, value,
		json.FormatNilMapAsNull(true),
		json.FormatNilSliceAsNull(true),
		json.Deterministic(true))
	if err != nil {
		return err
	}
	b.buffer.Truncate(b.Len() - 1)
	return nil
}

// EncodeKeyValue appends one JSON object entry (`"key": value`) to the buffer.
// Comma rule:
//   - no manual comma is needed between consecutive EncodeKeyValue calls
//   - a comma is auto-inserted only when the previous write was EncodeKeyValue
//   - if you write anything else between calls, you must manage commas manually
//
// Example usage:
//
//	b.WriteByte('{')
//	_ = b.EncodeKeyValue("name", name)
//	_ = b.EncodeKeyValue("age", age)
//	b.WriteByte('}')
func (b *Buffer) EncodeKeyValue(key string, value any) error {
	if !b.initialized || b.indent {
		b.enc.Reset(&b.buffer)
		b.initialized = true
		b.indent = false
	}
	if b.kvOff == b.Cap()-b.Available() {
		b.WriteByte(',')
	}
	err := json.MarshalEncode(&b.enc, key)
	if err != nil {
		return err
	}
	b.buffer.Truncate(b.Len() - 1)
	b.buffer.WriteByte(':')
	if value == nil {
		b.WriteString("null")
	} else {
		err = json.MarshalEncode(&b.enc, value, json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
		if err != nil {
			return err
		}
		b.buffer.Truncate(b.Len() - 1)
	}
	b.kvOff = b.Cap() - b.Available()
	return nil
}

// EncodeQuoted is like [Encode] but wraps the resulting JSON in quotes as a
// JSON string.
func (b *Buffer) EncodeQuoted(value any) error {
	if !b.initialized || b.indent {
		b.enc.Reset(&b.buffer)
		b.initialized = true
		b.indent = false
	}
	n1 := b.Len()
	err := json.MarshalEncode(&b.enc, value, json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
	if err != nil {
		return err
	}
	n2 := b.Len() - 1
	b.buffer.Truncate(n1)
	p := b.AvailableBuffer()
	b.text = append(b.text, p[:n2-n1]...)
	_ = json.MarshalEncode(&b.enc, b.text)
	b.buffer.Truncate(b.Len() - 1)
	b.text = b.text[:0]
	return nil
}

// EncodeSorted is like [Encode] but sorts object keys.
func (b *Buffer) EncodeSorted(v any) error {
	if !b.initialized || b.indent {
		b.enc.Reset(&b.buffer)
		b.initialized = true
		b.indent = false
	}
	err := json.MarshalEncode(&b.enc, v, json.Deterministic(true), json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
	if err != nil {
		return err
	}
	b.buffer.Truncate(b.Len() - 1)
	return nil
}

// Reset resets the buffer to write to s.
// To keep the current buffer, call [Buffer.Truncate] with n == 0 instead.
func (b *Buffer) Reset(s []byte) {
	b.buffer.Buffer = *bytes.NewBuffer(s)
	b.enc.Reset(&b.buffer)
	b.initialized = true
	b.indent = false
	b.kvOff = 0
}

// Truncate truncates the buffer to the first n unread bytes, retaining the
// underlying allocated storage for future use.
// It panics if n is negative or greater than the current length of the buffer.
func (b *Buffer) Truncate(n int) {
	b.buffer.Truncate(n)
	b.indent = false
	b.kvOff = 0
}

// Value returns a copy of the unread portion of the buffer as a Value.
// It returns a [*SyntaxError] if the unread portion is not valid JSON.
func (b *Buffer) Value() (Value, error) {
	v := b.buffer.Bytes()
	if err := Validate(v); err != nil {
		return nil, err
	}
	return slices.Clone(v), nil
}

type buffer struct {
	bytes.Buffer
}

type noCopy struct{}

func (*noCopy) Lock()   {}
func (*noCopy) Unlock() {}

// textMarshaler implements the encoding.TextMarshaler interface for a []byte
// value. When used with the internal json.Marshal method, the []byte value is
// encoded as a JSON string.
type textMarshaler []byte

func (text textMarshaler) MarshalText() ([]byte, error) {
	return text, nil
}
