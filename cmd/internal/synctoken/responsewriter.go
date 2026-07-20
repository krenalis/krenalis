// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package synctoken

import (
	"net/http"
)

// ResponseWriter sets the Sync-Token response header immediately before the
// response headers are sent.
type ResponseWriter struct {
	// ResponseWriter is the wrapped response writer.
	http.ResponseWriter

	// nonce is the nonce used to generate the Sync-Token.
	nonce [NonceSize]byte

	// stateVersion returns the latest state version.
	stateVersion func() int

	// codec encodes the Sync-Token.
	codec *Codec

	// wroteHeader reports whether the response headers have been sent.
	wroteHeader bool
}

// NewResponseWriter returns a ResponseWriter that wraps rw.
func NewResponseWriter(rw http.ResponseWriter, codec *Codec, nonce [NonceSize]byte, stateVersion func() int) *ResponseWriter {
	return &ResponseWriter{
		ResponseWriter: rw,
		codec:          codec,
		nonce:          nonce,
		stateVersion:   stateVersion,
	}
}

// Unwrap returns the wrapped response writer.
func (w *ResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// WriteHeader sets the Sync-Token header for a successful response and writes
// the HTTP status code.
func (w *ResponseWriter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	if code == http.StatusOK {
		w.setSyncToken()
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(code)
}

// Write sets the Sync-Token header, if necessary, and writes p to the response
// body.
func (w *ResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}

// Finish sets the Sync-Token header if the response headers have not yet been
// sent.
func (w *ResponseWriter) Finish() {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
}

// setSyncToken sets the Sync-Token response header.
func (w *ResponseWriter) setSyncToken() {
	token, err := w.codec.Encode(w.stateVersion(), w.nonce[:])
	if err != nil {
		panic("cannot encode Sync-Token: " + err.Error())
	}
	w.ResponseWriter.Header().Set("Sync-Token", token)
}
