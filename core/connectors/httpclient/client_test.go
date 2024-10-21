//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package httpclient

import (
	"net/http"
	"testing"
	"time"

	"github.com/meergo/meergo"
)

func Test_waitTime(t *testing.T) {

	// Disable jitter in backoff.
	backoffJitterEnabled = false

	var exponentialTimes = []time.Duration{backoffBase, backoffBase * 2, backoffBase * 4, backoffBase * 8, backoffBase * 16, backoffBase * 32, meergo.BackoffCap, meergo.BackoffCap}

	tests := []struct {
		policy   meergo.BackoffPolicy
		response *http.Response
		times    []time.Duration
		err      error
	}{
		{
			response: &http.Response{Status: "404 Not Found", StatusCode: 404},
			err:      meergo.NoRetry,
		},
		// Default backoff: 429 with an integer Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"2"}}},
			times:    []time.Duration{2 * time.Second},
		},
		// Default backoff: 429 with a decimal Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"0.5"}}},
			times:    []time.Duration{time.Duration(0.5 * float64(time.Second))},
		},
		// Default backoff: 429 without Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429},
			times:    exponentialTimes,
		},
		// Default backoff: 429 with an invalid Retry-After header.
		{
			response: &http.Response{Status: "429 Too Many Requests", StatusCode: 429, Header: http.Header{"Retry-After": []string{"3s"}}},
			times:    exponentialTimes,
		},
		// Default backoff: 500.
		{
			response: &http.Response{Status: "500 Internal Server Error", StatusCode: 500},
			times:    exponentialTimes,
		},
		// Custom backoff: 404.
		{
			policy: meergo.BackoffPolicy{"500": func(res *http.Response, retries int) (time.Duration, error) {
				return time.Duration(retries), nil
			}},
			response: &http.Response{Status: "404 Not Found", StatusCode: 404},
			err:      meergo.NoRetry,
		},
		// Custom backoff: 500.
		{
			policy: meergo.BackoffPolicy{"500": func(res *http.Response, retries int) (time.Duration, error) {
				return time.Duration(retries), nil
			}},
			response: &http.Response{Status: "500 Internal Server Error", StatusCode: 500},
			times:    []time.Duration{0, 1, 2, 3, 4, 5},
		},
	}

	c := &Client{}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			c.backoffPolicy = test.policy
			for retries := range max(len(test.times), 1) {
				got, err := c.waitTime(test.response, retries)
				if err != nil {
					if test.err == nil {
						t.Fatalf("expected no error, got error %q (type %T)", err, err)
					}
					if err.Error() != test.err.Error() {
						t.Fatalf("expected error %q (type %T), got error %q (type %T)", test.err, test.err, err, err)
					}
					return
				}
				if expected := test.times[retries]; expected != got {
					t.Fatalf("retry %d; expected wait time %s, got %s", retries, expected, got)
				}
			}
		})
	}

}

func Test_jitter(t *testing.T) {
	c := &Client{
		backoffPolicy: meergo.BackoffPolicy{
			"500": func(res *http.Response, replies int) (time.Duration, error) {
				return 0, nil
			}},
	}
	res := &http.Response{Status: "500 Internal Server Error", StatusCode: 500}
	tests := []time.Duration{0, 1, 10 * time.Millisecond, 250 * time.Millisecond, 5 * time.Second}
	for _, d := range tests {
		t.Run(d.String(), func(t *testing.T) {
			for range 10 {
				got, err := c.waitTime(res, 0)
				if err != nil {
					t.Fatalf("unexpected error %s", err)
				}
				if got < 0 || d != 0 && got >= d {
					t.Fatalf("expected a value in range [0, %s), got %s", d/2, got)
				}
			}
		})
	}
	t.Run("backoffJitterEnabled = false", func(t *testing.T) {
		backoffJitterEnabled = false
		if got, err := c.waitTime(res, 0); got != 0 {
			if err != nil {
				t.Fatalf("unexpected error %s", err)
			}
			t.Fatalf("expected 0, got %s", got)
		}
	})
}
