// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by the MIT license
// that can be found in the LICENSE file.

package connectors

import (
	"math"
	"net/http"
	"strconv"
	"time"
)

// BackoffCap is the exponential backoff cap.
var BackoffCap = 5 * time.Second

var nowTestTime time.Time

// ConstantStrategy returns a retry strategy implementing a constant backoff.
func ConstantStrategy(reason FailureReason, waitTime time.Duration) RetryStrategy {
	return func(res *http.Response, retries int) (FailureReason, time.Duration) {
		return reason, max(0, waitTime)
	}
}

// ExponentialStrategy returns a retry strategy that applies exponential backoff
// with the specified base duration between attempts.
func ExponentialStrategy(reason FailureReason, base time.Duration) RetryStrategy {
	b := float64(max(base, 0))
	return func(res *http.Response, retries int) (FailureReason, time.Duration) {
		d := time.Duration(b * math.Pow(2, float64(retries))) // base * 2^retries
		d = min(d, BackoffCap)                                // limit to cap
		return reason, d
	}
}

// HeaderStrategy returns a retry strategy that determines the wait time from a
// specific response header. The parse function is used to extract the duration
// from the header value. If parse is nil, the strategy behaves like
// RetryAfterStrategy but uses the specified header instead of "Retry-After".
// If parse returns an error, the strategy disables retry attempts.
func HeaderStrategy(reason FailureReason, header string, parse func(s string) (time.Duration, error)) RetryStrategy {
	return func(res *http.Response, retries int) (FailureReason, time.Duration) {
		s := res.Header.Get(header)
		if parse == nil {
			// Some servers might return a decimal value instead of an integer value.
			if seconds, err := strconv.ParseFloat(s, 64); err == nil {
				d := max(time.Duration(seconds*float64(time.Second)), 0)
				return reason, d
			}
			date, err := time.Parse(time.RFC1123, s)
			if err != nil {
				return PermanentFailure, 0
			}
			now := time.Now().UTC()
			if !nowTestTime.IsZero() {
				now = nowTestTime
			}
			return reason, max(0, date.UTC().Sub(now))

		}
		d, err := parse(s)
		if err != nil {
			return PermanentFailure, 0
		}
		if d < 0 {
			d = 0
		}
		return reason, d
	}
}

// RetryAfterStrategy returns a retry strategy that determines the wait time
// from the "Retry-After" response header. Unlike the HTTP standard, it also
// accepts decimal values for seconds.
// https://httpwg.org/specs/rfc6585.html#status-429
// https://httpwg.org/specs/rfc9110.html#field.retry-after
func RetryAfterStrategy() RetryStrategy {
	return HeaderStrategy(RateLimited, "Retry-After", nil)
}
