// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package requestid provides helpers for storing and retrieving request IDs in
// contexts.
//
// It is used by the core package and its subpackages to associate log messages
// with the API request being handled.
package requestid

import "context"

// contextKey is the context key type used by this package.
type contextKey struct{}

// requestIDKey is the key used to store a request ID in a context.
var requestIDKey contextKey

// RequestID returns the request ID associated with ctx.
// It returns an empty string if the context does not contain one.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDKey).(string)
	return requestID
}

// WithRequestID stores requestID in ctx and returns the resulting context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}
