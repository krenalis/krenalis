// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package cmd

import (
	"context"
	"net/http"

	"github.com/krenalis/krenalis/cmd/internal/requestid"
)

// requestIDContextKey is the context key for the RequestID associated with an
// API request.
type requestIDContextKey struct{}

// requestIDResponseWriter sets the Request-Id header before the response is sent.
type requestIDResponseWriter struct {
	// ResponseWriter is the wrapped writer that receives the final response.
	http.ResponseWriter

	// requestID is the value associated with the current request.
	requestID *requestid.RequestID

	// stateVersion returns the state version to include in Request-Id.
	stateVersion func() int

	// wrote reports whether this writer has already set Request-Id.
	wrote bool
}

// contextWithRequestID returns a context carrying requestID.
func contextWithRequestID(ctx context.Context, requestID *requestid.RequestID) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// requestIDOf returns the request ID associated with r.
func requestIDOf(r *http.Request) *requestid.RequestID {
	return requestIDFromContext(r.Context())
}

// WriteHeader sets the Request-Id header and then writes the status code.
func (w *requestIDResponseWriter) WriteHeader(code int) {
	w.writeRequestID()
	w.ResponseWriter.WriteHeader(code)
}

// Write sets the Request-Id header and then writes the response body.
func (w *requestIDResponseWriter) Write(p []byte) (int, error) {
	w.writeRequestID()
	return w.ResponseWriter.Write(p)
}

// finish sets the Request-Id header if it has not already been set.
func (w *requestIDResponseWriter) finish() {
	w.writeRequestID()
}

// writeRequestID updates and sets the Request-Id header.
func (w *requestIDResponseWriter) writeRequestID() {
	if w.wrote {
		return
	}
	w.requestID.SetStateVersion(w.stateVersion())
	w.Header().Set("Request-Id", w.requestID.String())
	w.wrote = true
}

// requestIDFromContext returns the request ID stored in ctx.
func requestIDFromContext(ctx context.Context) *requestid.RequestID {
	requestID, ok := ctx.Value(requestIDContextKey{}).(*requestid.RequestID)
	if !ok || requestID == nil {
		return nil
	}
	return requestID
}
