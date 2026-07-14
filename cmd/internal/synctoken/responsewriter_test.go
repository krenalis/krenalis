// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package synctoken

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/krenalis/krenalis/tools/base58"
)

// TestResponseWriter verifies when ResponseWriter emits Sync-Token and how it
// handles repeated writes.
func TestResponseWriter(t *testing.T) {
	t.Run("sets sync token before write", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		nonce := testNonce()
		w := NewResponseWriter(recorder, codec, nonce, func() int { return 12 })

		_, _ = w.Write([]byte("ok"))

		assertSyncTokenHeader(t, recorder, codec, 12, nonce)
	})

	t.Run("sets sync token before ok write header", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		nonce := testNonce()
		w := NewResponseWriter(recorder, codec, nonce, func() int { return 34 })

		w.WriteHeader(http.StatusOK)

		assertSyncTokenHeader(t, recorder, codec, 34, nonce)
	})

	t.Run("does not set sync token for non-ok write header", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		w := NewResponseWriter(recorder, codec, testNonce(), func() int { return 34 })

		w.WriteHeader(http.StatusNoContent)

		if got := recorder.Header().Get("Sync-Token"); got != "" {
			t.Fatalf("expected no Sync-Token header, got %q", got)
		}
	})

	t.Run("sets sync token before copy", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		nonce := testNonce()
		w := NewResponseWriter(recorder, codec, nonce, func() int { return 56 })

		_, _ = io.Copy(w, strings.NewReader("ok"))

		assertSyncTokenHeader(t, recorder, codec, 56, nonce)
	})

	t.Run("finish sets sync token when headers are pending", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		nonce := testNonce()
		w := NewResponseWriter(recorder, codec, nonce, func() int { return 78 })

		w.Finish()

		assertSyncTokenHeader(t, recorder, codec, 78, nonce)
	})

	t.Run("does not recompute after headers are written", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		codec := newTestCodec(t, 0x42)
		nonce := testNonce()
		version := 10
		w := NewResponseWriter(recorder, codec, nonce, func() int {
			version++
			return version
		})

		w.WriteHeader(http.StatusOK)
		first := recorder.Header().Get("Sync-Token")
		w.WriteHeader(http.StatusOK)
		second := recorder.Header().Get("Sync-Token")

		if second != first {
			t.Fatalf("expected memoized Sync-Token %q, got %q", first, second)
		}
		if version != 11 {
			t.Fatalf("expected state version callback count %d, got %d", 1, version-10)
		}
	})
}

// assertSyncTokenHeader verifies that recorder contains a Sync-Token with the
// expected version and nonce.
func assertSyncTokenHeader(t *testing.T, recorder *httptest.ResponseRecorder, codec *Codec, wantVersion int, wantNonce [NonceSize]byte) {
	t.Helper()

	token := recorder.Header().Get("Sync-Token")
	if token == "" {
		t.Fatalf("expected Sync-Token header, got empty")
	}

	version, err := codec.Decode(token)
	if err != nil {
		t.Fatalf("expected decoded Sync-Token, got %v", err)
	}
	if version != wantVersion {
		t.Fatalf("expected state version %d, got %d", wantVersion, version)
	}

	payload, err := base58.DecodeString(token)
	if err != nil {
		t.Fatalf("expected decoded Sync-Token payload, got %v", err)
	}
	if len(payload) != syncTokenSize() {
		t.Fatalf("expected Sync-Token binary length %d, got %d", syncTokenSize(), len(payload))
	}
	if !bytes.Equal(payload[:NonceSize], wantNonce[:]) {
		t.Fatalf("expected nonce prefix %v, got %v", wantNonce, payload[:NonceSize])
	}
}
