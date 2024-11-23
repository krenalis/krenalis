//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package json

import (
	"bytes"

	"github.com/meergo/meergo/json/internal/json"
	"github.com/meergo/meergo/json/internal/json/jsontext"
)

// Buffer wraps a bytes.Buffer, implementing all its methods, and additionally
// provides the [Buffer.Encode] and [Buffer.EncodeQuoted] methods for appending
// JSON-encoded values to the buffer.
// The zero value of Buffer is an empty buffer, ready for use.
type Buffer struct {
	buffer
	enc         jsontext.Encoder
	text        textMarshaler
	initialized bool
}

// NewBuffer returns a new buffer.
func NewBuffer() *Buffer {
	return &Buffer{}
}

// Encode appends the JSON encoding of in to the buffer.
// It returns an error if the value cannot be encoded as JSON.
func (b *Buffer) Encode(in any) error {
	if !b.initialized {
		b.enc.Reset(&b.buffer)
		b.initialized = true
	}
	err := json.MarshalEncode(&b.enc, in)
	if err != nil {
		return err
	}
	b.Truncate(b.Len() - 1)
	return nil
}

// EncodeQuoted appends the JSON encoding of in, further quoted as a JSON
// string, to the buffer.
// It returns an error if the value cannot be encoded as JSON.
func (b *Buffer) EncodeQuoted(in any) error {
	if !b.initialized {
		b.enc.Reset(&b.buffer)
		b.initialized = true
	}
	n1 := b.Len()
	err := json.MarshalEncode(&b.enc, in)
	if err != nil {
		return err
	}
	n2 := b.Len() - 1
	b.Truncate(n1)
	p := b.AvailableBuffer()
	b.text = append(b.text, p[:n2-n1]...)
	_ = json.MarshalEncode(&b.enc, b.text)
	b.Truncate(b.Len() - 1)
	b.text = b.text[:0]
	return nil
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
