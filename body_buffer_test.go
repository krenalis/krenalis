//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package meergo

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"testing"
)

// TestBodyBuffer exercises the behavior of BodyBuffer under different content encodings,
// including NoEncoding and Gzip, testing request creation, content length, headers,
// body reading, flushing, closing, and panic cases.
func TestBodyBuffer(t *testing.T) {

	ctx := context.Background()

	// Verify that an empty payload with no encoding produces a valid request.
	t.Run("NoEncodingEmptyPayload", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		req, err := bb.NewRequest(ctx, http.MethodPost, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating request: %v", err)
		}
		if req.ContentLength != 0 {
			t.Errorf("ContentLength = %d, want 0", req.ContentLength)
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type header = %q, want \"application/json\"", ct)
		}
		if a := req.Header.Get("Accept"); a != "application/json" {
			t.Errorf("Accept header = %q, want \"application/json\"", a)
		}
		if h, ok := req.Header["Content-Encoding"]; ok {
			t.Errorf("unexpected Content-Encoding header: %#v", h)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		if len(body) != 0 {
			t.Errorf("body length = %d, want 0", len(body))
		}
		err = req.Body.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
	})

	// Write simple payload and verify request body and headers.
	t.Run("NoEncodingSimplePayload", func(t *testing.T) {
		payload := []byte("hello world")
		bb := GetBodyBuffer(NoEncoding, 128)
		if _, err := bb.Write(payload); err != nil {
			t.Errorf("unexpected error writing payload: %v", err)
		}
		req, err := bb.NewRequest(ctx, http.MethodPost, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating request: %v", err)
		}
		if req.ContentLength != int64(len(payload)) {
			t.Errorf("ContentLength = %d, want %d", req.ContentLength, len(payload))
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type header = %q, want \"application/json\"", ct)
		}
		if a := req.Header.Get("Accept"); a != "application/json" {
			t.Errorf("Accept header = %q, want \"application/json\"", a)
		}
		got, _ := io.ReadAll(req.Body)
		if !bytes.Equal(got, payload) {
			t.Errorf("body mismatch: got %q, want %q", got, payload)
		}

		// GetBody before Close must work.
		rc, err := req.GetBody()
		if err != nil {
			t.Errorf("unexpected error calling GetBody: %v", err)
		}
		copy1, err := io.ReadAll(rc)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		err = rc.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		if !bytes.Equal(copy1, payload) {
			t.Errorf("GetBody mismatch")
		}

		// Reading after NewRequest must return an error.
		if _, err := req.Body.Read(make([]byte, 1)); err == nil {
			t.Error("expected read error after NewRequest")
		}

		// GetBody after Close must return an error.
		err = req.Body.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		bb.Close()
		if _, err := req.GetBody(); err == nil {
			t.Error("expected error from GetBody after Close")
		}

	})

	// Write payload and call NewRequest without explicit Flush.
	// Verify Gzip encoding and proper decompression.
	t.Run("GzipSimplePayload", func(t *testing.T) {
		payload := []byte("compress me!")
		bb := GetBodyBuffer(Gzip, 0)
		bb.Write(payload)
		req, err := bb.NewRequest(ctx, http.MethodPut, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating a request: %v", err)
		}
		if ct := req.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type header = %q, want \"application/json\"", ct)
		}
		if a := req.Header.Get("Accept"); a != "application/json" {
			t.Errorf("Accept header = %q, want \"application/json\"", a)
		}
		if enc := req.Header.Get("Content-Encoding"); enc != "gzip" {
			t.Errorf("Content-Encoding header = %q, want \"gzip\"", enc)
		}

		encoded, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		if req.ContentLength != int64(len(encoded)) {
			t.Errorf("ContentLength mismatch: got %d, want %d", req.ContentLength, len(encoded))
		}

		gr, err := gzip.NewReader(bytes.NewReader(encoded))
		if err != nil {
			t.Errorf("unexpected error calling gzip.NewReader: %v", err)
		}
		decoded, err := io.ReadAll(gr)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		err = gr.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		if !bytes.Equal(decoded, payload) {
			t.Errorf("decoded payload mismatch")
		}

		// GetBody must return a valid new gzip stream.
		rc, err := req.GetBody()
		if err != nil {
			t.Errorf("unexpected error calling GetBody: %v", err)
		}
		enc2, err := io.ReadAll(rc)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		err = rc.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		gr2, err := gzip.NewReader(bytes.NewReader(enc2))
		if err != nil {
			t.Errorf("unexpected error decompressing: %v", err)
		}
		dec2, err := io.ReadAll(gr2)
		if err != nil {
			t.Errorf("unexpected error decompressing: %v", err)
		}
		err = gr2.Close()
		if err != nil {
			t.Errorf("unexpected error closing: %v", err)
		}
		if !bytes.Equal(dec2, payload) {
			t.Errorf("GetBody decoded mismatch")
		}
	})

	// Flush must clear the internal buffer.
	t.Run("FlushNoEncoding", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		if l := bb.Len(); l != 0 {
			t.Errorf("expected length 0, got %d", l)
		}
		_, err := bb.Write([]byte("abc"))
		if err != nil {
			t.Errorf("unexpected error writing: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		bb.Close()
	})

	// Flush must clear the internal buffer.
	t.Run("FlushGzipWithData", func(t *testing.T) {
		bb := GetBodyBuffer(Gzip, 0)
		if l := bb.Len(); l != 0 {
			t.Errorf("expected length 0, got %d", l)
		}
		_, err := bb.Write([]byte("abc"))
		if err != nil {
			t.Errorf("unexpected error writing: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		if l := bb.Len(); l != 3 {
			t.Errorf("expected length 3, got %d", l)
		}
		bb.Close()
	})

	// Flush should succeed even if no data was written.
	t.Run("FlushNoEncodingNoData", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		bb.Close()
	})

	// Flush should succeed even if no data was written.
	t.Run("FlushGzipNoData", func(t *testing.T) {
		bb := GetBodyBuffer(Gzip, 0)
		if err := bb.Flush(); err != nil {
			t.Errorf("unexpected error flushing body: %v", err)
		}
		bb.Close()
	})

	// Calling Close multiple times must not panic.
	t.Run("CloseIdempotent", func(t *testing.T) {
		bb1 := GetBodyBuffer(NoEncoding, 0)
		bb1.Close()
		bb1.Close()

		bb2 := GetBodyBuffer(Gzip, 0)
		bb2.Close()
		bb2.Close()
	})

	// Writes after NewRequest must return an error.
	t.Run("UseAfterNewRequestFails", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		bb.Write([]byte("x"))
		if _, err := bb.NewRequest(ctx, http.MethodGet, "http://example.com"); err != nil {
			t.Errorf("unexpected error creating request: %v", err)
		}
		_, err := bb.Write([]byte("y"))
		if err == nil {
			t.Error("expected error on write after NewRequest")
		}
	})

	// Reading after Body is closed must return an error.
	t.Run("BodyReaderReadAfterClose", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		bb.Write([]byte("x"))
		req, err := bb.NewRequest(ctx, http.MethodPost, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating a request: %v", err)
		}
		err = req.Body.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		if _, err := req.Body.Read(make([]byte, 1)); err == nil {
			t.Error("expected error reading after Close")
		}
	})

	// GetBody must return an error after Body is closed.
	t.Run("GetBodyAfterClose", func(t *testing.T) {
		bb := GetBodyBuffer(NoEncoding, 0)
		bb.Write([]byte("x"))
		req, err := bb.NewRequest(ctx, http.MethodPost, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating a request: %v", err)
		}
		err = req.Body.Close()
		if err != nil {
			t.Errorf("unexpected error closing body: %v", err)
		}
		bb.Close()
		if _, err := req.GetBody(); err == nil {
			t.Error("expected error from GetBody after Close")
		}
	})

	// Write large payload and verify correct gzip compression and decompression.
	t.Run("LargePayloadGzip", func(t *testing.T) {
		payload := bytes.Repeat([]byte("a"), 1<<20)
		bb := GetBodyBuffer(Gzip, 0)
		if _, err := bb.Write(payload); err != nil {
			t.Errorf("unexpected error writing payload: %v", err)
		}
		req, err := bb.NewRequest(ctx, http.MethodPost, "http://example.com")
		if err != nil {
			t.Errorf("unexpected error creating request: %v", err)
		}
		encoded, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("unexpected error reading body: %v", err)
		}
		gr, err := gzip.NewReader(bytes.NewReader(encoded))
		if err != nil {
			t.Errorf("unexpected error decompressing body: %v", err)
		}
		decoded, err := io.ReadAll(gr)
		if err != nil {
			t.Errorf("unexpected error reading the decompressed body: %v", err)
		}
		err = gr.Close()
		if err != nil {
			t.Errorf("unexpected error closing: %v", err)
		}
		if !bytes.Equal(decoded, payload) {
			t.Errorf("decoded large payload mismatch")
		}
	})

	// Truncate must correctly truncate with any encoding.
	t.Run("LargePayloadGzip", func(t *testing.T) {

		payload := []byte("sentence to truncate")

		// With NoEncoding.
		bb1 := GetBodyBuffer(NoEncoding, 0)
		defer bb1.Close()
		if _, err := bb1.Write(payload); err != nil {
			t.Errorf("unexpected error writing payload: %v", err)
		}
		bb1.Truncate(len(payload))
		if plain := bb1.plain.Bytes(); !bytes.Equal(payload, plain) {
			t.Errorf("expected %q, got %q", payload, plain)
		}
		bb1.Truncate(len("sentence"))
		if plain := string(bb1.plain.Bytes()); plain != "sentence" {
			t.Errorf("expected %q, got %q", "sentence", plain)
		}
		err := bb1.Flush()
		if err != nil {
			t.Errorf("expected error flushing body: %s", err)
		}
		_, err = bb1.WriteString(", new sentence")
		if err != nil {
			t.Errorf("expected error writing body: %s", err)
		}
		bb1.Truncate(len(", new"))
		if plain := string(bb1.plain.Bytes()); plain != "sentence, new" {
			t.Errorf("expected %q, got %q", "sentence, new", plain)
		}

		// With Gzip.
		bb2 := GetBodyBuffer(Gzip, 0)
		defer bb2.Close()
		if _, err := bb2.Write(payload); err != nil {
			t.Errorf("unexpected error writing payload: %v", err)
		}
		bb2.Truncate(len(payload))
		if plain := bb2.plain.Bytes(); !bytes.Equal(payload, plain) {
			t.Errorf("expected %q, got %q", payload, plain)
		}
		bb2.Truncate(len("sentence"))
		if plain := string(bb2.plain.Bytes()); plain != "sentence" {
			t.Errorf("expected %q, got %q", "sentence", plain)
		}
		err = bb2.Flush()
		if err != nil {
			t.Errorf("expected error flushing body: %s", err)
		}
		_, err = bb2.WriteString(", new sentence")
		if err != nil {
			t.Errorf("expected error writing body: %s", err)
		}
		bb2.Truncate(len(", new"))
		if plain := string(bb2.plain.Bytes()); plain != ", new" {
			t.Errorf("expected %q, got %q", ", new", plain)
		}

		// Truncate panics if n < 0.
		defer func() {
			if r := recover(); r == nil {
				panic("expected panic on underflow truncate, got no panic")
			}
			// Truncate panics if n > len(plain).
			defer func() {
				if r := recover(); r == nil {
					panic("expected panic on overflow truncate, got no panic")
				}
			}()
			bb2.Truncate(len(bb2.plain.Bytes()) + 1)
		}()
		bb2.Truncate(-1)

	})

}

// TestBodyBufferPool verifies the reuse behavior of BodyBuffer instances and
// internal buffers. It ensures that:
//   - bodyBufferState structs are correctly pooled and reused for both
//     NoEncoding and Gzip encodings.
//   - The internal plain byte slice is reused via the pool when released
//     correctly.
//   - Buffers are not reused if Close or NewRequest is not called, preventing
//     premature reuse.
func TestBodyBufferPool(t *testing.T) {

	// BodyBuffer instance reuse (NoEncoding)
	t.Run("PoolReuseBodyBufferNoEncoding", func(t *testing.T) {
		// First allocation
		bb1 := GetBodyBuffer(NoEncoding, 0)
		ptr1 := bb1.bodyBufferState
		bb1.Close()

		// Second allocation – should reuse same object.
		bb2 := GetBodyBuffer(NoEncoding, 0)
		ptr2 := bb2.bodyBufferState
		if ptr1 != ptr2 {
			t.Fatalf("bodyBufferState not reused: %p vs %p", ptr1, ptr2)
		}
		bb2.Close()
	})

	// BodyBuffer instance reuse (Gzip).
	t.Run("PoolReuseBodyBufferGzip", func(t *testing.T) {
		bb1 := GetBodyBuffer(Gzip, 0)
		ptr1 := bb1.bodyBufferState
		bb1.Close()

		bb2 := GetBodyBuffer(Gzip, 0)
		ptr2 := bb2.bodyBufferState
		if ptr1 != ptr2 {
			t.Fatalf("bodyBufferState not reused for Gzip: %p vs %p", ptr1, ptr2)
		}
		bb2.Close()
	})

	// Pool not reused when Close/NewRequest omitted.
	t.Run("PoolNotReusedWithoutClose", func(t *testing.T) {
		bb1 := GetBodyBuffer(NoEncoding, 0)
		ptr1 := bb1.bodyBufferState // NOT closed here

		bb2 := GetBodyBuffer(NoEncoding, 0)
		ptr2 := bb2.bodyBufferState
		if ptr1 == ptr2 {
			t.Fatalf("expected distinct bodyBufferState when first was not closed")
		}

		// Clean up to avoid leaking buffers into the pool between subtests.
		bb1.Close()
		bb2.Close()
	})
}
