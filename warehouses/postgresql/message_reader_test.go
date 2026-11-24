// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package postgresql

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5/pgproto3"
)

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
	for _, end := range []int{2, 5, 65539} {
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

// TestMessageReaderProtocolLimit checks the actual wire length boundary
// without allocating a maximum-size message.
func TestMessageReaderProtocolLimit(t *testing.T) {

	const maxBody = 1<<30 - 2
	for _, tc := range []struct {
		body   int32
		prefix int
	}{
		{0, 0},
		{16 << 20, 0},
		{(16 << 20) + 1, 5 + (16 << 20)},
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

	var r messageReader
	err := r.startMessage(maxBody + 5)
	if err != nil {
		limitErr, ok := errors.AsType[*pgproto3.ExceededMaxBodyLenErr](err)
		if !ok || limitErr.MaxExpectedBodyLen != maxBody || limitErr.ActualBodyLen != maxBody+1 {
			t.Fatalf("expected a %d-byte body to exceed the %d-byte limit, got %v", maxBody+1, maxBody, err)
		}
		return
	}
	t.Fatal("expected ExceededMaxBodyLenErr, got nil")

}
