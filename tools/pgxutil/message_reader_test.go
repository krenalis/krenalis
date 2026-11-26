// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package pgxutil

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math/rand/v2"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
)

const testMaxMessageBodyLen = 1<<30 - 2

// TestMessageReaderBufferingThresholds checks how many bytes the reader plans
// to buffer at key body sizes without reading or allocating the body.
func TestMessageReaderBufferingThresholds(t *testing.T) {
	for _, tc := range []struct {
		name   string
		body   int32
		prefix int
	}{
		{"empty body", 0, 0},
		{"16 MiB", 16 << 20, 0},
		{"16 MiB plus one", (16 << 20) + 1, 5 + (16 << 20)},
		{"32 MiB", 32 << 20, 5 + (16 << 20)},
		{"32 MiB plus one", (32 << 20) + 1, 5 + (16 << 20) + 1},
		{"maximum body", testMaxMessageBodyLen, 5 + testMaxMessageBodyLen/2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r messageReader
			err := r.startMessage(tc.body + 4)
			if err != nil {
				t.Fatalf("expected a %d-byte body to be accepted, got %v", tc.body, err)
			}
			if r.remaining != int(tc.body)+5 || r.toBuffer != tc.prefix {
				t.Fatalf("expected %d frame bytes and %d prefix bytes, got %d and %d",
					int(tc.body)+5, tc.prefix, r.remaining, r.toBuffer)
			}
		})
	}
}

// TestMessageReaderDataRowHeader checks that the reader withholds the header
// of a DataRow with a 33 MiB body until it has read half the body.
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

	const supplied = len(header) + bodyBytes/2 - 1
	source := &io.LimitedReader{R: bytes.NewReader(frame), N: int64(supplied)}
	r := messageReader{src: bufio.NewReader(source)}
	n, err := r.Read(header[:])
	if n != 0 {
		t.Fatalf("expected no header one byte before half the body, got %d bytes", n)
	}
	requireEndOfInput(t, err, supplied)
	if source.N != 0 {
		t.Fatalf("expected the available prefix to be consumed, got %d bytes remaining", source.N)
	}
	source.N = 1
	_, err = io.ReadFull(&r, header[:])
	if err != nil {
		t.Fatalf("expected the header after half the body, got %v", err)
	}
	if source.N != 0 {
		t.Fatalf("expected the last prefix byte to be consumed, got %d bytes remaining", source.N)
	}
	if !bytes.Equal(header[:], frame[:len(header)]) {
		t.Fatalf("expected header %x, got %x", frame[:len(header)], header)
	}

}

// TestMessageReaderPartialReads checks that the reader limits prefix storage
// during partial reads, withholds each large message's header until it has
// read the required body prefix, and preserves message contents and boundaries.
func TestMessageReaderPartialReads(t *testing.T) {

	const chunkBytes = 64 << 10
	data := make([]byte, maxEagerBodyLen+2*chunkBytes+1)
	rand.NewChaCha8([32]byte{}).Read(data)
	frame, err := (&pgproto3.CopyData{Data: data}).Encode(nil)
	if err != nil {
		t.Fatal(err)
	}
	var source bytes.Buffer
	r := &messageReader{src: bufio.NewReader(&source)}
	var header [5]byte

	checkHeaderWithheld := func(supplied int) {
		t.Helper()
		n, err := r.Read(header[:])
		if n != 0 {
			t.Fatalf("expected no header after %d bytes, got %d bytes", supplied, n)
		}
		capacity := 0
		for _, chunk := range r.prefix {
			capacity += cap(chunk)
		}
		if capacity > supplied+chunkBytes {
			t.Fatalf("expected at most %d bytes of prefix storage after %d bytes, got %d",
				supplied+chunkBytes, supplied, capacity)
		}
		requireEndOfInput(t, err, supplied)
	}

	start := 0
	// The first buffered chunk includes the header.
	for _, end := range []int{2, len(header), len(header) + 1, chunkBytes - 1, chunkBytes, chunkBytes + 1} {
		source.Write(frame[start:end])
		start = end
		checkHeaderWithheld(end)
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
	if !ok {
		t.Fatalf("expected CopyData, got %T", msg)
	}
	if !bytes.Equal(got.Data, data) {
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
	checkHeaderWithheld(prefixEnd - 1)
	source.Write(frame[prefixEnd-1 : prefixEnd])
	_, err = io.ReadFull(r, header[:])
	if err != nil {
		t.Fatalf("expected the next header once the prefix arrived, got %v", err)
	}
	if !bytes.Equal(header[:], frame[:len(header)]) {
		t.Fatalf("expected header %x, got %x", frame[:len(header)], header)
	}

	source.Write(frame[prefixEnd:])
	body := make([]byte, len(data))
	_, err = io.ReadFull(r, body)
	if err != nil {
		t.Fatalf("expected the next CopyData body, got %v", err)
	}
	if !bytes.Equal(body, data) {
		t.Fatal("expected the original payload in the next CopyData body, got different data")
	}

}

// TestMessageReaderProtocolLimits checks the minimum and maximum message
// lengths the reader accepts without allocating large messages.
func TestMessageReaderProtocolLimits(t *testing.T) {

	t.Run("below minimum length", func(t *testing.T) {
		header := binary.BigEndian.AppendUint32([]byte{'D'}, 3)
		r := messageReader{src: bufio.NewReader(bytes.NewReader(header))}
		n, err := r.Read(make([]byte, len(header)))
		if n != 0 || err == nil {
			t.Fatalf("expected zero bytes and an error for wire length 3, got %d bytes and %v", n, err)
		}
	})

	for _, tc := range []struct {
		name string
		body int32
	}{
		{"minimum valid length", 0},
		{"maximum valid length", testMaxMessageBodyLen},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r messageReader
			err := r.startMessage(tc.body + 4)
			if err != nil {
				t.Fatalf("expected a %d-byte body to be accepted, got %v", tc.body, err)
			}
		})
	}

	t.Run("above maximum length", func(t *testing.T) {
		const oversizedBody = testMaxMessageBodyLen + 1
		header := binary.BigEndian.AppendUint32([]byte{'D'}, oversizedBody+4)
		r := messageReader{src: bufio.NewReader(bytes.NewReader(header))}
		n, err := r.Read(make([]byte, len(header)))
		if n != 0 {
			t.Fatalf("expected zero bytes for an oversized body, got %d", n)
		}
		if err != nil {
			limitErr, ok := errors.AsType[*pgproto3.ExceededMaxBodyLenErr](err)
			if !ok {
				t.Fatalf("expected ExceededMaxBodyLenErr, got %T (%v)", err, err)
			}
			if limitErr.MaxExpectedBodyLen != testMaxMessageBodyLen || limitErr.ActualBodyLen != oversizedBody {
				t.Fatalf("expected body length %d and limit %d, got %d and %d",
					oversizedBody, testMaxMessageBodyLen, limitErr.ActualBodyLen, limitErr.MaxExpectedBodyLen)
			}
			return
		}
		t.Fatal("expected ExceededMaxBodyLenErr, got nil")
	})

}

// requireEndOfInput checks that err is either io.EOF or io.ErrUnexpectedEOF.
func requireEndOfInput(t *testing.T, err error, supplied int) {
	t.Helper()
	if err != nil {
		if err != io.EOF && err != io.ErrUnexpectedEOF {
			t.Fatalf("expected end of input after %d bytes, got %v", supplied, err)
		}
		return
	}
	t.Fatalf("expected end of input after %d bytes, got nil", supplied)
}
