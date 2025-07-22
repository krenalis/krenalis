//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/meergo/meergo/bytespool"
	"github.com/meergo/meergo/json"
)

// ContentEncoding represents supported HTTP body encodings.
type ContentEncoding int

const (
	NoEncoding ContentEncoding = iota // No encoding (identity); body is uncompressed
	Gzip                              // Gzip encoding; conforms to RFC 1952
)

// BodyBuffer provides a reusable buffer for constructing the body of an HTTP
// request. Encode methods write JSON, but the buffer supports any content type,
// including binary.
//
// Call Flush whenever a portion of the written content is complete and should
// no longer be modified. For Gzip encoding, each call to Flush triggers
// compression of the buffered data up to that point.
//
// Once the body has been fully written, the NewRequest method returns an
// *http.Request with the buffered content set as the request body. If encoding
// is Gzip, the request body is compressed using Gzip.
//
// BodyBuffer internally uses a memory pool to reduce allocations and improve
// performance across multiple uses. Always call Close when you're done with the
// BodyBuffer to release its resources.
type BodyBuffer struct {
	mu sync.Mutex // protects access to bodyBuffer during Close and in request.GetBody
	*bodyBuffer
}

// bodyBuffer is the internal, pooled type used by BodyBuffer.
type bodyBuffer struct {
	enc     ContentEncoding // content encoding
	plain   json.Buffer     // plain data
	gzipW   gzip.Writer     // gzip writer
	flushed int             // size of the flushed plain data
	body    struct {
		// WaitGroup tracks active body readers.
		// Add(1) is called by NewRequest and request.GetBody.
		// Done is called by bodyReader.Close.
		// Close blocks until all readers have called Done.
		sync.WaitGroup
		// bodyWriter.buf holds the request body.
		// It is written to by the gzip writer and NewRequest, and read from by body readers.
		bodyWriter
	}
}

// GetBodyBuffer returns a BodyBuffer configured with the specified content
// encoding. If the encoding is Gzip, the body will be automatically compressed
// using Gzip. length defines the minimum initial length for the internal
// buffer.
//
// After writing to the buffer, call NewRequest to obtain an *http.Request with
// the body set. Once finished, Close must be called to release the resources
// associated with the BodyBuffer.
func GetBodyBuffer(enc ContentEncoding, length int) *BodyBuffer {
	bb := bodyBufPool.Get()
	bb.enc = enc
	switch enc {
	case NoEncoding:
		b := bytespool.Get(length)
		bb.plain.Reset(b)
	case Gzip:
		b := bytespool.Get(1024)
		bb.plain.Reset(b)
		bb.body.buf = bytespool.Get(length)
		bb.gzipW.Reset(&bb.body)
	default:
		panic(fmt.Sprintf("meergo: invalid encoding %d", enc))
	}
	return &BodyBuffer{bodyBuffer: bb}
}

// Close releases the resources associated with the BodyBuffer and must always
// be called when the buffer is no longer needed.
//
// If NewReader has been called, Close waits for both Reader.Body and any bodies
// returned by Reader.GetBody to be closed before returning.
func (bb *BodyBuffer) Close() {
	// Return if it was already closed.
	if bb.bodyBuffer == nil {
		return
	}
	// Returns the plain buffer to the pool.
	if plain := bb.plain.Bytes(); plain != nil {
		clear(plain)
		bytespool.Put(plain)
		bb.plain.Reset(nil)
	}
	bb.mu.Lock()
	// Returns the body buffer to the pool.
	if bb.body.buf != nil {
		// Wait for all readers to be closed.
		bb.body.Wait()
		clear(bb.body.buf)
		bytespool.Put(bb.body.buf)
		bb.body.buf = nil
	}
	// Return the bodyBuffer to the pool.
	bodyBufPool.Put(bb.bodyBuffer)
	bb.bodyBuffer = nil
	bb.mu.Unlock()
}

var errPostReqWrite = errors.New("cannot write after the request has been created")

// Encode appends the JSON encoding of value to the buffer. It returns an error
// if the value cannot be encoded as JSON, or if called after NewRequest.
func (bb *BodyBuffer) Encode(value any) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.Encode(value)
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
func (bb *BodyBuffer) EncodeIndent(value any, prefix, indent string) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.EncodeIndent(value, prefix, indent)
}

// EncodeKeyValue appends the JSON encoding of a key-value pair to the buffer.
// If the previous write to the buffer was made by EncodeKeyValue, a comma is
// appended before the key-value pair.
//
// Example usage:
//
//	bb.WriteByte('{')
//	_ = bb.EncodeKeyValue("name", name)
//	_ = bb.EncodeKeyValue("age", age)
//	bb.WriteByte('}')
//
// It returns an error if the value cannot be encoded as JSON, or if called
// after NewRequest.
func (bb *BodyBuffer) EncodeKeyValue(key string, value any) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.EncodeKeyValue(key, value)
}

// EncodeQuoted is like [Encode] but wraps the resulting JSON in quotes as a
// JSON string.
func (bb *BodyBuffer) EncodeQuoted(value any) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.EncodeQuoted(value)
}

// EncodeSorted is like [Encode] but sorts object keys.
func (bb *BodyBuffer) EncodeSorted(v any) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.EncodeSorted(v)
}

// Flush flushes data appending to the body. It returns an error if the value
// cannot be encoded as JSON, or if called after NewRequest. If an error occurs,
// it closes the buffer and returns the error.
func (bb *BodyBuffer) Flush() error {
	plain := bb.plain.Bytes()
	if plain == nil {
		return errPostReqWrite
	}
	if len(plain) == 0 {
		return nil
	}
	switch bb.enc {
	case NoEncoding:
		bb.flushed += len(plain)
	case Gzip:
		n, err := bb.gzipW.Write(plain)
		bb.flushed += n
		bb.plain.Truncate(len(plain) - n)
		if err != nil {
			bb.Close()
			return err
		}
	}
	return nil
}

// Len returns the number of bytes written, including unflushed data.
// The result may differ from the request body length if Gzip encoding is use.
func (bb *BodyBuffer) Len() int {
	return bb.plain.Len() + bb.flushed
}

// NewRequest creates a new http.Request using the given method and URL.
// The request body is populated with the data written to bb.
// Any buffered data is flushed before creating the request.
//
// The returned request includes the following headers:
//   - Content-Type: application/json
//   - Accept: application/json
//   - Content-Encoding: gzip (only if the selected encoding is Gzip)
//
// The ContentLength and GetBody fields of the request are also set.
//
// If called on a nil value, the request is created without a body, and only the
// "Accept: application/json" header is included.
//
// After calling NewRequest, only the Close method can be called.
func (bb *BodyBuffer) NewRequest(ctx context.Context, method, url string) (*http.Request, error) {

	if bb == nil {
		req, err := http.NewRequestWithContext(ctx, method, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		return req, nil
	}

	if bb.plain.Bytes() == nil {
		return nil, errors.New("NewRequest can only be called once")
	}

	plain := bb.plain.Bytes()
	switch bb.enc {
	case NoEncoding:
		bb.flushed += bb.plain.Len()
		bb.body.buf = plain
	case Gzip:
		// Flush the plain buffer.
		bb.flushed += bb.plain.Len()
		_, err := bb.plain.WriteTo(&bb.gzipW)
		if err != nil {
			return nil, err
		}
		// Put the plain buffer into the pool.
		clear(plain)
		bytespool.Put(plain)
		// Flushes the gzip buffer into the body buffer.
		err = bb.gzipW.Close()
		if err != nil {
			return nil, err
		}
	default:
		panic("unexpected encoding")
	}
	bb.plain.Reset(nil)

	bb.body.Add(1) // marks creation of a reader
	body := newBodyReader(bb.body.buf, bb.body.Done)
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	req.ContentLength = int64(len(bb.body.buf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if bb.enc == Gzip {
		req.Header.Set("Content-Encoding", "gzip")
	}
	req.GetBody = func() (io.ReadCloser, error) {
		bb.mu.Lock()
		defer bb.mu.Unlock()
		if bb.bodyBuffer == nil {
			return nil, errors.New("body has been already released")
		}
		bb.body.Add(1) // marks creation of a reader
		return newBodyReader(bb.body.buf, bb.body.Done), nil
	}

	return req, nil
}

// Truncate discards unflushed bytes after the first n.
// It panics if n is negative, greater than the number of unflushed bytes,
// or if Truncate is called after NewRequest.
func (bb *BodyBuffer) Truncate(n int) {
	plain := bb.plain.Bytes()
	if plain == nil {
		panic(errPostReqWrite.Error())
	}
	if n < 0 || n > len(plain)-bb.flushed {
		panic("meergo: truncation out of range")
	}
	switch bb.enc {
	case NoEncoding:
		bb.plain.Truncate(bb.flushed + n)
	case Gzip:
		bb.plain.Truncate(n)
	}
}

// Value returns a copy of the unflushed portion of the buffer as a
// [json.Value]. It returns a *[json.SyntaxError] if the unflushed portion is
// not valid JSON.
func (bb *BodyBuffer) Value() (json.Value, error) {
	return bb.plain.Value()
}

// Write appends p to the buffer and returns the length of p and nil.
// It returns an error if called after NewRequest.
func (bb *BodyBuffer) Write(p []byte) (n int, err error) {
	if bb.plain.Bytes() == nil {
		return 0, errPostReqWrite
	}
	return bb.plain.Write(p)
}

// WriteByte appends c to the buffer. The returned error is always nil.
// It returns an error if called after NewRequest.
func (bb *BodyBuffer) WriteByte(c byte) error {
	if bb.plain.Bytes() == nil {
		return errPostReqWrite
	}
	return bb.plain.WriteByte(c)
}

// WriteString appends s to the buffer. It returns the length of s and nil.
// It returns an error if called after NewRequest.
func (bb *BodyBuffer) WriteString(s string) (int, error) {
	if bb.plain.Bytes() == nil {
		return 0, errPostReqWrite
	}
	return bb.plain.WriteString(s)
}

// bodyReader implements io.Writer for request bodies.
type bodyWriter struct {
	buf []byte
}

func (w *bodyWriter) Write(p []byte) (int, error) {
	if n := len(w.buf) + len(p); n > cap(w.buf) {
		old := w.buf
		w.buf = bytespool.Get(n)
		w.buf = w.buf[0:n]
		copy(w.buf, old)
		copy(w.buf[len(old):], p)
		clear(old)
		bytespool.Put(old)
	} else {
		w.buf = append(w.buf, p...)
	}
	return len(p), nil
}

// bodyReader implements io.ReadCloser for request bodies.
type bodyReader struct {
	buf    []byte
	closed func() // closed is called when it is closed.
}

// newBodyReader returns an  io.ReadCloser that reads from body and calls the
// function closed when closed.
func newBodyReader(buf []byte, closed func()) io.ReadCloser {
	return &bodyReader{buf: buf, closed: closed}
}

var errClosed = errors.New("reader is closed")

func (r *bodyReader) Read(p []byte) (int, error) {
	if r.buf == nil {
		return 0, errClosed
	}
	if len(r.buf) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.buf)
	r.buf = r.buf[n:]
	return n, nil
}

func (r *bodyReader) Close() error {
	if r.buf != nil {
		r.buf = nil
		r.closed()
	}
	return nil
}

// bodyBufferPool is a pool of *bodyBuffer.
type bodyBufferPool struct {
	sync.Pool
}

// Get returns a *bodyBuffer from the pool.
func (p *bodyBufferPool) Get() *bodyBuffer {
	return p.Pool.Get().(*bodyBuffer)
}

// Put returns bb to the pool.
func (p *bodyBufferPool) Put(bb *bodyBuffer) {
	p.Pool.Put(bb)
}

// bodyBufPool is a pool of reusable *bodyBuffer instances to reduce
// allocations.
var bodyBufPool = &bodyBufferPool{
	Pool: sync.Pool{
		New: func() any {
			bb := &bodyBuffer{}
			bb.gzipW = *gzip.NewWriter(nil)
			return bb
		},
	},
}
