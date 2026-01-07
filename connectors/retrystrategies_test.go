// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"net/http"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {

	// Set a fake time.
	nowTestTime = time.Date(2024, 8, 20, 15, 49, 13, 387104382, time.UTC)

	tests := []struct {
		strategy RetryStrategy
		response *http.Response
		reason   FailureReason
		times    []time.Duration
	}{
		{
			strategy: ConstantStrategy(NetFailure, 300*time.Millisecond),
			reason:   NetFailure,
			times:    []time.Duration{300 * time.Millisecond, 300 * time.Millisecond},
		},
		{
			strategy: ConstantStrategy(Slowdown, -100*time.Millisecond),
			reason:   Slowdown,
			times:    []time.Duration{0, 0},
		},
		{
			strategy: ExponentialStrategy(Slowdown, 750*time.Millisecond),
			reason:   Slowdown,
			times:    []time.Duration{750 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second, BackoffCap, BackoffCap},
		},
		{
			strategy: ExponentialStrategy(Slowdown, 500*time.Microsecond),
			reason:   Slowdown,
			times:    []time.Duration{500 * time.Microsecond, time.Millisecond, 2 * time.Millisecond, 4 * time.Millisecond},
		},
		{
			strategy: ExponentialStrategy(Slowdown, 0),
			reason:   Slowdown,
			times:    []time.Duration{0, 0, 0},
		},
		{
			strategy: ExponentialStrategy(Slowdown, -100),
			reason:   Slowdown,
			times:    []time.Duration{0, 0, 0},
		},
		{
			strategy: ExponentialStrategy(Slowdown, BackoffCap+1),
			reason:   Slowdown,
			times:    []time.Duration{BackoffCap, BackoffCap, BackoffCap},
		},
		{
			strategy: HeaderStrategy(Slowdown, "After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"Tue, 20 Aug 2024 15:53:00 UTC"}}},
			reason:   Slowdown,
			times:    []time.Duration{226612895618},
		},
		{
			strategy: HeaderStrategy(Slowdown, "After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"Tue, 13 Aug 2024 09:23:51 UTC"}}},
			reason:   Slowdown,
			times:    []time.Duration{0},
		},
		{
			strategy: HeaderStrategy(Slowdown, "After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"5"}}},
			reason:   Slowdown,
			times:    []time.Duration{5 * time.Second},
		},
		{
			strategy: HeaderStrategy(Slowdown, "After", time.ParseDuration),
			response: &http.Response{Header: http.Header{"After": []string{"2s"}}},
			reason:   Slowdown,
			times:    []time.Duration{2 * time.Second},
		},
		{
			strategy: HeaderStrategy(Slowdown, "After", time.ParseDuration),
			response: &http.Response{Header: http.Header{"After": []string{""}}},
			reason:   PermanentFailure,
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"10"}}},
			reason:   RateLimited,
			times:    []time.Duration{10 * time.Second},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"2.25"}}},
			reason:   RateLimited,
			times:    []time.Duration{time.Duration(2.25 * float64(time.Second))},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"-3"}}},
			reason:   RateLimited,
			times:    []time.Duration{0},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"Tue, 20 Aug 2024 16:00:00 UTC"}}},
			reason:   RateLimited,
			times:    []time.Duration{646612895618},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{},
			reason:   PermanentFailure,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for retries := range max(len(test.times), 1) {
				reason, waitTime := test.strategy(test.response, retries)
				if reason != test.reason {
					t.Fatalf("expected reason %s, got %s", test.reason, reason)
				}
				if reason == PermanentFailure {
					continue
				}
				if expected := test.times[retries]; expected != waitTime {
					t.Fatalf("retry %d; expected wait time %s, got %s", retries, expected, waitTime)
				}
			}
		})
	}
}
