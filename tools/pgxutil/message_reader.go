// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package pgxutil

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"
)

const (
	maxMessageBodyLen = 1<<30 - 2 // largest body accepted by pgproto3 by default
	maxEagerBodyLen   = 16 << 20  // maximum body pgx may reserve from the header alone
)

// messageReader delays large message headers until at least half the body has
// arrived. It forwards one message at a time so read-ahead cannot expose the
// next header before its body has been checked.
type messageReader struct {
	src       *bufio.Reader
	remaining int         // bytes still to deliver in this message, including the header
	toBuffer  int         // bytes still needed before the header can be forwarded
	prefix    net.Buffers // blocks to deliver once the prefix is complete
	pending   []byte      // unfilled suffix of the last prefix block
}

// Read returns bytes from the PostgreSQL message stream. It rejects wire
// lengths below four bytes and returns *pgproto3.ExceededMaxBodyLenErr for
// bodies larger than maxMessageBodyLen.
func (r *messageReader) Read(p []byte) (int, error) {

	if len(p) == 0 {
		return 0, nil
	}
	if r.remaining == 0 {
		header, err := r.src.Peek(5)
		if err != nil {
			return 0, err
		}
		err = r.startMessage(int32(binary.BigEndian.Uint32(header[1:])))
		if err != nil {
			return 0, err
		}
	}

	for r.toBuffer > 0 {
		const chunkSize = 64 << 10
		if r.pending == nil {
			r.pending = make([]byte, min(chunkSize, r.toBuffer))
			r.prefix = append(r.prefix, r.pending)
		}
		n, err := io.ReadFull(r.src, r.pending)
		r.toBuffer -= n
		if err != nil {
			r.pending = r.pending[n:]
			return 0, err
		}
		r.pending = nil
	}

	if len(r.prefix) > 0 {
		n, _ := r.prefix.Read(p) // EOF here only marks the end of the buffered prefix.
		if len(r.prefix) == 0 {
			r.prefix = nil
		}
		r.remaining -= n
		return n, nil
	}
	n, err := r.src.Read(p[:min(len(p), r.remaining)])
	r.remaining -= n

	return n, err
}

// startMessage validates the wire length, which includes the four length bytes
// but excludes the type byte, and sets how much to read before forwarding it.
// It rejects lengths below four and returns *pgproto3.ExceededMaxBodyLenErr
// for bodies larger than maxMessageBodyLen.
func (r *messageReader) startMessage(length int32) error {
	if length < 4 {
		return fmt.Errorf("invalid PostgreSQL message length: %d", length)
	}
	bodyLen := int(length - 4)
	if bodyLen > maxMessageBodyLen {
		return &pgproto3.ExceededMaxBodyLenErr{MaxExpectedBodyLen: maxMessageBodyLen, ActualBodyLen: bodyLen}
	}
	r.remaining = 5 + bodyLen
	if bodyLen > maxEagerBodyLen {
		// pgx rounds allocations up to a power of two through 32 MiB.
		// Wait for enough data to cover at least half the allocation.
		r.toBuffer = 5 + max(maxEagerBodyLen, (bodyLen+1)/2)
	}
	return nil
}

// ConfigureMessageReader limits how much memory pgx can reserve from a header
// alone, starting with the first response and preserving support for wide rows.
// config must not be nil.
func ConfigureMessageReader(config *pgconn.Config) {
	if config == nil {
		panic("tools/pgxutil: nil config")
	}
	config.MaxProtocolMessageBodyLen = maxMessageBodyLen
	config.BuildFrontend = func(r io.Reader, w io.Writer) *pgproto3.Frontend {
		return pgproto3.NewFrontend(&messageReader{src: bufio.NewReaderSize(r, 8192)}, w)
	}
}
