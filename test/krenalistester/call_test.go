// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package krenalistester

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// TestTryCallRetriesRateLimit verifies that TryCall retries a rate-limited
// request and replays its body.
func TestTryCallRetriesRateLimit(t *testing.T) {
	var (
		attempts int
		bodies   []string
	)
	k := &Krenalis{
		t: t,
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				return nil, err
			}
			attempts++
			bodies = append(bodies, string(body))
			if attempts == 1 {
				return testHTTPResponse(request, http.StatusTooManyRequests, http.Header{"Retry-After": {"1"}}, "rate limited"), nil
			}
			return testHTTPResponse(request, http.StatusOK, nil, `{"id":"retried"}`), nil
		})},
	}

	var response struct {
		ID string `json:"id"`
	}
	if err := k.TryCall(http.MethodPost, "/v1/test", nil, map[string]string{"name": "retry"}, &response); err != nil {
		t.Fatalf("TryCall() error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if len(bodies) != 2 || bodies[0] != bodies[1] {
		t.Fatalf("expected the request body to be replayed, got %#v", bodies)
	}
	if response.ID != "retried" {
		t.Fatalf("expected retried response, got %#v", response)
	}
}

// TestTryCallWithoutRetryReturnsFirstResponse verifies that
// TryCallWithoutRetry returns a rate-limited response without retrying it.
func TestTryCallWithoutRetryReturnsFirstResponse(t *testing.T) {
	attempts := 0
	k := &Krenalis{
		t: t,
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			attempts++
			return testHTTPResponse(request, http.StatusTooManyRequests, http.Header{"Retry-After": {"1"}}, "rate limited"), nil
		})},
	}

	err := k.TryCallWithoutRetry(http.MethodGet, "/v1/test", nil, nil, nil)
	statusErr, ok := err.(*StatusCodeError)
	if !ok {
		t.Fatalf("expected *StatusCodeError, got %T: %v", err, err)
	}
	if statusErr.Response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected HTTP status %d, got %d", http.StatusTooManyRequests, statusErr.Response.Code)
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func testHTTPResponse(request *http.Request, status int, header http.Header, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    request,
	}
}

// TestParseRetryAfter verifies that parseRetryAfter accepts only the canonical
// positive integer seconds emitted by Krenalis.
func TestParseRetryAfter(t *testing.T) {
	for _, test := range []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{name: "seconds", value: "2", want: 2 * time.Second},
		{name: "maximum", value: "1200", want: 20 * time.Minute},
		{name: "above maximum", value: "1201", wantErr: true},
		{name: "missing", value: "", wantErr: true},
		{name: "invalid", value: "later", wantErr: true},
		{name: "zero", value: "0", wantErr: true},
		{name: "negative", value: "-1", wantErr: true},
		{name: "positive sign", value: "+1", wantErr: true},
		{name: "leading zero", value: "01", wantErr: true},
		{name: "leading whitespace", value: " 1", wantErr: true},
		{name: "trailing whitespace", value: "1 ", wantErr: true},
		{name: "http date", value: "Mon, 04 Aug 2026 12:00:00 GMT", wantErr: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := parseRetryAfter(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got duration %s", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRetryAfter() error: %v", err)
			}
			if got != test.want {
				t.Fatalf("parseRetryAfter() = %s, want %s", got, test.want)
			}
		})
	}
}
