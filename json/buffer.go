//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"bytes"
	"slices"

	"github.com/meergo/meergo/json/internal/json"
	"github.com/meergo/meergo/json/internal/json/jsontext"
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
	err := json.MarshalEncode(&b.enc, value, json.Deterministic(true), json.FormatNilMapAsNull(true), json.FormatNilSliceAsNull(true))
	if err != nil {
		return err
	}
	b.buffer.Truncate(b.Len() - 1)
	return nil
}

// EncodeKeyValue appends the JSON encoding of a key-value pair to the buffer.
// If the previous write to the buffer was made by EncodeKeyValue, a comma is
// appended before the key-value pair.
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

// EncodeQuoted is like [Encode] but wraps the resulting JSON in quotes as
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

// Reset resets the buffer, making it empty, while retaining the underlying
// storage for future writes.
// It is functionally equivalent to calling [Buffer.Truncate](0).
func (b *Buffer) Reset() {
	b.buffer.Reset()
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
// It returns a *SyntaxError if the unread portion is not valid JSON.
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

// textMarshaler implements the encoding.TextMarshaler interface for a []byte
// value. When used with the internal json.Marshal method, the []byte value is
// encoded as a JSON string.
type textMarshaler []byte

func (text textMarshaler) MarshalText() ([]byte, error) {
	return text, nil
}
