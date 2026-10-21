// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/connectors"
	"github.com/krenalis/krenalis/tools/errors"
)

// responseBody tracks how many times it is closed.
type responseBody struct {
	io.Reader
	closes int
}

// Close increments the close count.
func (b *responseBody) Close() error {
	b.closes++
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f with req.
func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// TestClientDoResponseBody checks that discarded response bodies are closed and
// the returned response body remains open.
func TestClientDoResponseBody(t *testing.T) {

	tests := []struct {
		name              string
		method            string
		idempotencyHeader string
		status            int
		header            http.Header
		accessToken       string
		retrieveToken     bool
		cancelDuringWait  bool
		wantRetry         bool
	}{
		{name: "ok", status: 200},
		{name: "created", status: 201},
		{name: "no content", status: 204},
		{name: "permanent failure", status: 404},
		{name: "slowdown", status: 429, wantRetry: true},
		{name: "rate limited", status: 429, header: http.Header{"Retry-After": {"1"}}, wantRetry: true},
		{name: "internal server error", status: 500, wantRetry: true},
		{name: "POST slowdown", method: http.MethodPost, status: 429},
		{name: "POST rate limited", method: http.MethodPost, status: 429, header: http.Header{"Retry-After": {"1"}}},
		{name: "POST internal server error", method: http.MethodPost, status: 500},
		{name: "idempotent POST slowdown", method: http.MethodPost, idempotencyHeader: "Idempotency-Key", status: 429, wantRetry: true},
		{name: "idempotent POST rate limited", method: http.MethodPost, idempotencyHeader: "Idempotency-Key", status: 429, header: http.Header{"Retry-After": {"1"}}, wantRetry: true},
		{name: "idempotent POST internal server error", method: http.MethodPost, idempotencyHeader: "Idempotency-Key", status: 500, wantRetry: true},
		{name: "idempotent POST with X-Idempotency-Key", method: http.MethodPost, idempotencyHeader: "X-Idempotency-Key", status: 500, wantRetry: true},
		{name: "unauthorized without OAuth", status: 401},
		{name: "unauthorized with OAuth", status: 401, accessToken: "token", wantRetry: true},
		{name: "unauthorized POST with OAuth", method: http.MethodPost, status: 401, accessToken: "token", wantRetry: true},
		{name: "unauthorized token retrieval", status: 401, accessToken: "token", retrieveToken: true},
		{name: "canceled during rate limit", status: 429, header: http.Header{"Retry-After": {"1"}}, cancelDuringWait: true, wantRetry: true},
		{name: "canceled during backoff", status: 500, cancelDuringWait: true, wantRetry: true},
		{name: "POST rate limited with deadline", method: http.MethodPost, status: 429, header: http.Header{"Retry-After": {"1"}}, cancelDuringWait: true},
		{name: "POST internal server error with deadline", method: http.MethodPost, status: 500, cancelDuringWait: true},
	}

	for _, test := range tests {
		for _, bodyKind := range []string{"nil", "empty", "replayable", "non-replayable"} {
			t.Run(test.name+"/"+bodyKind, func(t *testing.T) {
				// TODO: remove when Do no longer retries requests that are not
				// idempotent or whose body cannot be read again.
				isIdempotentRequest := test.method != http.MethodPost || test.idempotencyHeader != ""
				isRetriable := isIdempotentRequest && bodyKind != "non-replayable"
				if (test.status == 429 || test.status == 500) && !isRetriable {
					t.Skip("Do retries requests that are not retriable")
				}
				synctest.Test(t, func(t *testing.T) {

					firstContent := "first response"
					if test.status == http.StatusNoContent {
						firstContent = ""
					}
					firstBody := &responseBody{Reader: strings.NewReader(firstContent)}
					first := &http.Response{StatusCode: test.status, Header: test.header, Body: firstBody}
					lastBody := &responseBody{Reader: strings.NewReader("last response")}
					last := &http.Response{StatusCode: 200, Body: lastBody}
					wantRequestContent := ""
					if bodyKind == "replayable" || bodyKind == "non-replayable" {
						wantRequestContent = "request"
					}
					calls := 0
					c := &Client{
						accessToken: test.accessToken,
						transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {

							wantMethod := test.method
							if wantMethod == "" {
								wantMethod = http.MethodGet
							}
							if req.Method != wantMethod {
								t.Errorf("expected method %q, got %q", wantMethod, req.Method)
							}
							if test.idempotencyHeader != "" {
								if got := req.Header.Get(test.idempotencyHeader); got != "key" {
									t.Errorf("expected idempotency key %q, got %q", "key", got)
								}
							}
							var data []byte
							if req.Body != nil {
								defer req.Body.Close()
								var err error
								data, err = io.ReadAll(req.Body)
								if err != nil {
									t.Fatalf("expected request body read to succeed, got %v", err)
								}
							}
							if string(data) != wantRequestContent {
								t.Errorf("expected request body %q, got %q", wantRequestContent, data)
							}

							calls++
							if calls == 1 {
								return first, nil
							}
							if firstBody.closes != 1 {
								t.Errorf("expected first response body closed once before retry, got %d", firstBody.closes)
							}

							return last, nil
						}),
					}
					c.endpointGroups.mux = http.NewServeMux()
					c.endpointGroups.mux.Handle("/", noOpHandle)
					c.endpointGroups.byPattern = map[string]endpointGroup{
						"/": {requireOAuth: true, rateLimiter: newRateLimiter(100, 10, 0)},
					}

					ctx := t.Context()
					if test.cancelDuringWait {
						var cancel context.CancelFunc
						ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
						defer cancel()
					}
					var body io.Reader
					switch bodyKind {
					case "nil":
					case "empty":
						body = http.NoBody
					case "replayable":
						body = strings.NewReader("request")
					case "non-replayable":
						body = io.NopCloser(strings.NewReader("request"))
					default:
						t.Fatalf("expected a known request body kind, got %q", bodyKind)
					}
					req, err := http.NewRequestWithContext(ctx, test.method, "https://example.com/", body)
					if err != nil {
						t.Fatalf("expected request creation to succeed, got %v", err)
					}
					if test.idempotencyHeader != "" {
						req.Header.Set(test.idempotencyHeader, "key")
					}

					wantRetry := test.wantRetry && bodyKind != "non-replayable"
					wantCancellation := test.cancelDuringWait && wantRetry
					res, err := c.do(req, test.retrieveToken)
					if err != nil {
						if !wantCancellation || !errors.Is(err, context.DeadlineExceeded) {
							t.Fatalf("expected only a deadline error during the retry wait, got %v", err)
						}
						if res != nil {
							t.Errorf("expected no response on cancellation, got %v", res)
						}
						if firstBody.closes != 1 {
							t.Errorf("expected first response body closed once on cancellation, got %d", firstBody.closes)
						}
						if calls != 1 {
							t.Errorf("expected 1 request before cancellation, got %d", calls)
						}
						return
					}
					if wantCancellation {
						t.Fatal("expected deadline error during the retry wait, got nil")
					}
					if res == nil {
						t.Fatal("expected a response, got nil")
					}
					defer res.Body.Close()
					wantResponse, wantCalls, wantCloses := first, 1, 0
					wantContent := firstContent
					if wantRetry {
						wantResponse, wantCalls, wantCloses = last, 2, 1
						wantContent = "last response"
					}
					if res != wantResponse {
						t.Errorf("expected response %p, got %p", wantResponse, res)
					}
					if calls != wantCalls {
						t.Errorf("expected %d requests, got %d", wantCalls, calls)
					}
					if firstBody.closes != wantCloses {
						t.Errorf("expected first response body closed %d times, got %d", wantCloses, firstBody.closes)
					}
					if lastBody.closes != 0 {
						t.Errorf("expected final response body to remain open, got %d closes", lastBody.closes)
					}
					data, err := io.ReadAll(res.Body)
					if err != nil {
						t.Fatalf("expected response body read to succeed, got %v", err)
					}
					if string(data) != wantContent {
						t.Errorf("expected response body %q, got %q", wantContent, data)
					}

				})
			})
		}
	}

}

func Test_retryStrategy(t *testing.T) {

	// Disable jitter in backoff.
	backoffJitterEnabled = false

	var exponentialTimes = []time.Duration{backoffBase, backoffBase * 2, backoffBase * 4, backoffBase * 8, backoffBase * 16, backoffBase * 32, connectors.BackoffCap, connectors.BackoffCap}

	tests := []struct {
		policy   connectors.RetryPolicy
		response *http.Response
		reason   connectors.FailureReason
		times    []time.Duration
	}{
		{
			response: &http.Response{Status: "404 Not Found", StatusCode: 404},
			reason:   connectors.PermanentFailure,
		},
		// Default backoff: 429 with an integer Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"2"}}},
			times:    []time.Duration{2 * time.Second},
			reason:   connectors.RateLimited,
		},
		// Default backoff: 429 with a decimal Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"0.5"}}},
			times:    []time.Duration{time.Duration(0.5 * float64(time.Second))},
			reason:   connectors.RateLimited,
		},
		// Default backoff: 429 without Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429},
			times:    exponentialTimes,
			reason:   connectors.Slowdown,
		},
		// Default backoff: 429 with an invalid Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"3s"}}},
			times:    exponentialTimes,
			reason:   connectors.Slowdown,
		},
		// Default backoff: 500.
		{
			response: &http.Response{Status: "500 Internal Server Error", StatusCode: 500},
			times:    exponentialTimes,
			reason:   connectors.NetFailure,
		},
		// Custom backoff: 404.
		{
			policy: connectors.RetryPolicy{"500": func(res *http.Response, retries int) (connectors.FailureReason, time.Duration) {
				return connectors.NetFailure, time.Duration(retries)
			}},
			response: &http.Response{Status: "404 Not Found", StatusCode: 404},
			reason:   connectors.PermanentFailure,
		},
		// Custom backoff: 500.
		{
			policy: connectors.RetryPolicy{"500": func(res *http.Response, retries int) (connectors.FailureReason, time.Duration) {
				return connectors.NetFailure, time.Duration(retries * 2)
			}},
			response: &http.Response{Status: "500 Internal Server Error", StatusCode: 500},
			reason:   connectors.NetFailure,
			times:    []time.Duration{0, 2, 4, 6, 8, 10},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for retries := range len(test.times) {
				reason, wt := retryStrategy(test.policy, test.response, retries)
				if reason != test.reason {
					t.Fatalf("expected reason %s, got %s", test.reason, reason)
				}
				if reason != connectors.PermanentFailure {
					if expected := test.times[retries]; expected != wt {
						t.Fatalf("retry %d; expected wait time %s, got %s", retries, expected, wt)
					}
				}
			}
		})
	}

}

func Test_jitter(t *testing.T) {
	policy := connectors.RetryPolicy{
		"500": func(res *http.Response, replies int) (connectors.FailureReason, time.Duration) {
			return connectors.NetFailure, 0
		}}
	res := &http.Response{Status: "500 Internal Server Error", StatusCode: 500}
	tests := []time.Duration{0, 1, 10 * time.Millisecond, 250 * time.Millisecond, 5 * time.Second}
	for _, d := range tests {
		t.Run(d.String(), func(t *testing.T) {
			for range 10 {
				reason, wt := retryStrategy(policy, res, 0)
				if reason != connectors.NetFailure {
					t.Fatalf("unexpected reason %s", reason)
				}
				if wt < 0 || d != 0 && wt >= d {
					t.Fatalf("expected a value in range [0, %s), got %s", d/2, wt)
				}
			}
		})
	}
	t.Run("backoffJitterEnabled = false", func(t *testing.T) {
		backoffJitterEnabled = false
		reason, wt := retryStrategy(policy, res, 0)
		if reason != connectors.NetFailure {
			t.Fatalf("unexpected reason %s", reason)
		}
		if wt != 0 {
			t.Fatalf("expected 0, got %s", wt)
		}
	})
}
