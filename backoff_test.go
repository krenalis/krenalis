//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package meergo

import (
	"net/http"
	"testing"
	"time"
)

func Test_Backoff(t *testing.T) {

	// Set a fake time.
	nowTestTime = time.Date(2024, 8, 20, 15, 49, 13, 387104382, time.UTC)

	tests := []struct {
		strategy BackoffStrategy
		response *http.Response
		times    []time.Duration
		err      error
	}{
		{
			strategy: ConstantStrategy(300 * time.Millisecond),
			times:    []time.Duration{300 * time.Millisecond, 300 * time.Millisecond},
		},
		{
			strategy: ConstantStrategy(-100 * time.Millisecond),
			times:    []time.Duration{0, 0},
		},
		{
			strategy: ExponentialStrategy(750 * time.Millisecond),
			times:    []time.Duration{750 * time.Millisecond, 1500 * time.Millisecond, 3 * time.Second, BackoffCap, BackoffCap},
		},
		{
			strategy: ExponentialStrategy(0),
			times:    []time.Duration{0, 0, 0},
		},
		{
			strategy: ExponentialStrategy(-100),
			times:    []time.Duration{0, 0, 0},
		},
		{
			strategy: ExponentialStrategy(BackoffCap + 1),
			times:    []time.Duration{BackoffCap, BackoffCap, BackoffCap},
		},
		{
			strategy: HeaderStrategy("After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"Tue, 20 Aug 2024 15:53:00 UTC"}}},
			times:    []time.Duration{226612895618},
		},
		{
			strategy: HeaderStrategy("After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"Tue, 13 Aug 2024 09:23:51 UTC"}}},
			times:    []time.Duration{0},
		},
		{
			strategy: HeaderStrategy("After", nil),
			response: &http.Response{Header: http.Header{"After": []string{"5"}}},
			times:    []time.Duration{5 * time.Second},
		},
		{
			strategy: HeaderStrategy("After", time.ParseDuration),
			response: &http.Response{Header: http.Header{"After": []string{"2s"}}},
			times:    []time.Duration{2 * time.Second},
		},
		{
			strategy: HeaderStrategy("After", time.ParseDuration),
			response: &http.Response{Header: http.Header{"After": []string{""}}},
			err:      NoRetry,
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"10"}}},
			times:    []time.Duration{10 * time.Second},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"2.25"}}},
			times:    []time.Duration{time.Duration(2.25 * float64(time.Second))},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"-3"}}},
			times:    []time.Duration{0},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{Header: http.Header{"Retry-After": []string{"Tue, 20 Aug 2024 16:00:00 UTC"}}},
			times:    []time.Duration{646612895618},
		},
		{
			strategy: RetryAfterStrategy(),
			response: &http.Response{},
			err:      NoRetry,
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			for retries := range max(len(test.times), 1) {
				got, err := test.strategy(test.response, retries)
				if err != nil {
					if test.err == nil {
						t.Fatalf("expected no error, got error %q (type %T)", err, err)
					}
					if err != NoRetry {
						t.Fatalf("expected NoRetry error, got error %q (type %T)", err, err)
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
