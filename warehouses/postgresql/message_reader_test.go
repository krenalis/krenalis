// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
)

// TestMessageReaderDataRowHeader checks that the reader releases the header
// of a 33 MiB DataRow as soon as half its body has arrived.
func TestMessageReaderDataRowHeader(t *testing.T) {

	const bodyBytes = 33 << 20
	// A one-column DataRow has a two-byte field count and a four-byte field length.
	row := pgproto3.DataRow{Values: [][]byte{make([]byte, bodyBytes-6)}}
	frame, err := row.Encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	var header [5]byte
	if got := len(frame) - len(header); got != bodyBytes {
		t.Fatalf("expected a %d-byte DataRow body, got %d bytes", bodyBytes, got)
	}

	source := &io.LimitedReader{R: bytes.NewReader(frame), N: int64(len(header) + bodyBytes/2 - 1)}
	r := messageReader{src: bufio.NewReader(source)}
	n, err := r.Read(header[:])
	if n != 0 || err == nil {
		t.Fatalf("expected no header one byte before half the body, got %d bytes and %v", n, err)
	}
	source.N = 1
	_, err = io.ReadFull(&r, header[:])
	if err != nil {
		t.Fatalf("expected the header after half the body, got %v", err)
	}
	if !bytes.Equal(header[:], frame[:len(header)]) {
		t.Fatalf("expected header %x, got %x", frame[:len(header)], header)
	}

}

// TestMessageReaderPartialReads checks that an interrupted read retains the
// prefix and withholds the header without reserving the declared body size.
func TestMessageReaderPartialReads(t *testing.T) {

	data := bytes.Repeat([]byte("abc"), maxEagerBodyLen/3+1)
	frame, err := (&pgproto3.CopyData{Data: data}).Encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	var source bytes.Buffer
	r := &messageReader{src: bufio.NewReader(&source)}
	var header [5]byte
	start := 0
	for _, end := range []int{2, 5, 6, 65539} {
		source.Write(frame[start:end])
		start = end
		n, err := r.Read(header[:])
		if n != 0 {
			t.Fatalf("expected the header to be withheld, got %d bytes", n)
		}
		if err != nil {
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				t.Fatalf("expected end of input, got %v", err)
			}
			capacity := 0
			for _, chunk := range r.prefix {
				capacity += cap(chunk)
			}
			if capacity > end+65536 {
				t.Fatalf("expected at most %d bytes of prefix storage, got %d", end+65536, capacity)
			}
			continue
		}
		t.Fatal("expected end of input, got nil")
	}

	source.Write(frame[start:])
	next := &pgproto3.ReadyForQuery{TxStatus: 'I'}
	trailer, err := next.Encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	source.Write(trailer)
	source.Write(frame[:len(header)])
	frontend := pgproto3.NewFrontend(r, io.Discard)
	msg, err := frontend.Receive()
	if err != nil {
		t.Fatal(err)
	}
	got, ok := msg.(*pgproto3.CopyData)
	if !ok || !bytes.Equal(got.Data, data) {
		t.Fatal("expected the original CopyData payload, got different data")
	}
	msg, err = frontend.Receive()
	if err != nil {
		t.Fatal(err)
	}
	ready, ok := msg.(*pgproto3.ReadyForQuery)
	if !ok {
		t.Fatalf("expected ReadyForQuery, got %T", msg)
	}
	if *ready != *next {
		t.Fatalf("expected %v, got %v", next, ready)
	}

	if n := frontend.ReadBufferLen(); n != 0 {
		t.Fatalf("expected no bytes of the next header in pgx, got %d", n)
	}

	// Deliver all but the last byte required to release the next header.
	const prefixEnd = len(header) + maxEagerBodyLen
	source.Write(frame[len(header) : prefixEnd-1])
	n, err := r.Read(header[:])
	if n != 0 {
		t.Fatalf("expected the next header to be withheld, got %d bytes", n)
	}
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			t.Fatalf("expected end of input, got %v", err)
		}
		source.Write(frame[prefixEnd-1 : prefixEnd])
		_, err = io.ReadFull(r, header[:])
		if err != nil {
			t.Fatalf("expected the next header once the prefix arrived, got %v", err)
		}
		if !bytes.Equal(header[:], frame[:len(header)]) {
			t.Fatalf("expected header %x, got %x", frame[:len(header)], header)
		}
		return
	}
	t.Fatal("expected end of input, got nil")

}

// TestMessageReaderProtocolLimit checks both wire length boundaries
// without allocating a maximum-size message.
func TestMessageReaderProtocolLimit(t *testing.T) {

	t.Run("wire length below four", func(t *testing.T) {
		header := []byte{'1', 0, 0, 0, 3}
		r := messageReader{src: bufio.NewReader(bytes.NewReader(header))}
		n, err := r.Read(make([]byte, len(header)))
		if n != 0 || err == nil {
			t.Fatalf("expected zero bytes and an error for wire length 3, got %d bytes and %v", n, err)
		}
	})

	const maxBody = 1<<30 - 2
	for _, tc := range []struct {
		body   int32
		prefix int
	}{
		{0, 0},
		{16 << 20, 0},
		{(16 << 20) + 1, 5 + (16 << 20)},
		{(32 << 20) + 1, 5 + (16 << 20) + 1},
		{maxBody, 5 + maxBody/2},
	} {
		var r messageReader
		err := r.startMessage(tc.body + 4)
		if err != nil {
			t.Fatalf("expected a %d-byte body to be accepted, got %v", tc.body, err)
		}
		if r.remaining != int(tc.body)+5 || r.toBuffer != tc.prefix {
			t.Fatalf("expected %d frame bytes and %d prefix bytes, got %d and %d",
				int(tc.body)+5, tc.prefix, r.remaining, r.toBuffer)
		}
	}

	const oversizedBody = maxBody + 1
	header := binary.BigEndian.AppendUint32([]byte{'D'}, oversizedBody+4)
	r := messageReader{src: bufio.NewReader(bytes.NewReader(header))}
	n, err := r.Read(make([]byte, len(header)))
	if n != 0 {
		t.Fatalf("expected zero bytes for an oversized body, got %d", n)
	}
	requireMessageBodyLimitError(t, err, maxBody, oversizedBody)

}

// requireMessageBodyLimitError checks both lengths reported for a rejected body.
func requireMessageBodyLimitError(t *testing.T, err error, limit, bodyLen int) {
	t.Helper()
	if err != nil {
		limitErr, ok := errors.AsType[*pgproto3.ExceededMaxBodyLenErr](err)
		if !ok || limitErr.MaxExpectedBodyLen != limit || limitErr.ActualBodyLen != bodyLen {
			t.Fatalf("expected ExceededMaxBodyLenErr for %d bytes with limit %d, got %T (%v)",
				bodyLen, limit, err, err)
		}
		return
	}
	t.Fatal("expected ExceededMaxBodyLenErr, got nil")
}
