// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/meergo/meergo/connectors"
)

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
