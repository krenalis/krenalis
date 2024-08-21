//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2024 Open2b
//

package meergo

import (
	"errors"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"
)

// BackoffCap is the exponential backoff cap.
var BackoffCap = 5 * time.Second

// BackoffJitterEnabled controls whether jitter is applied in backoff.
var BackoffJitterEnabled = true

// NoRetry is the error returned by a Backoff function when the request
// should not be retried.
var NoRetry = errors.New("no retry")

var nowTestTime time.Time

// Backoff represents a backoff strategy. It returns the duration to wait before
// the next attempt, based on the response from the previous attempt and the
// number of retries made. retries starts at 0 before the first retry and
// increments by 1 after each retry.
//
// If the returned duration is negative, it is considered zero.
// It returns NoRetry if the request should not be retried.
type Backoff func(res *http.Response, retries int) (time.Duration, error)

// ConstantBackoff returns a backoff function implementing a constant backoff.
func ConstantBackoff(waitTime time.Duration) Backoff {
	return func(res *http.Response, retries int) (time.Duration, error) {
		return max(0, waitTime), nil
	}
}

// ExponentialBackoff returns a Backoff that implements an exponential backoff
// strategy based on the provided base duration.
func ExponentialBackoff(base time.Duration) Backoff {
	b := float64(base.Milliseconds())
	return func(res *http.Response, retries int) (time.Duration, error) {
		d := time.Duration(b*math.Pow(2, float64(retries))) * time.Millisecond // base * 2^retries
		d = min(d, BackoffCap)                                                 // limit to cap
		return d + jitter(d), nil
	}
}

// HeaderBackoff returns a Backoff that determines the wait time from the
// provided response header. parse is used to parse the header's value. If parse
// is nil, it behaves like RetryAfterBackoff but uses the specified header
// instead of "Retry-After". If parse returns an error, HeaderBackoff will
// return a NoRetry error to indicate no retry should be attempted.
func HeaderBackoff(header string, parse func(s string) (time.Duration, error)) Backoff {
	return func(res *http.Response, retries int) (time.Duration, error) {
		s := res.Header.Get(header)
		if parse == nil {
			// Some servers might return a decimal value instead of an integer value.
			if seconds, err := strconv.ParseFloat(s, 64); err == nil {
				d := time.Duration(seconds * float64(time.Second))
				if d < 0 {
					d = 0
				}
				return d, nil
			}
			if date, err := time.Parse(time.RFC1123, s); err == nil {
				now := time.Now().UTC()
				if !nowTestTime.IsZero() {
					now = nowTestTime
				}
				d := date.UTC().Sub(now)
				if d < 0 {
					d = 0
				}
				return d + jitter(d), nil
			}
			return 0, NoRetry
		}
		d, err := parse(s)
		if err != nil {
			return 0, NoRetry
		}
		if d < 0 {
			d = 0
		}
		return d + jitter(d), nil
	}
}

// RetryAfterBackoff returns a Backoff that determines the wait time from the
// Retry-After header. Unlike the standard, it also accepts seconds expressed
// with decimal values.
// https://httpwg.org/specs/rfc6585.html#status-429
// https://httpwg.org/specs/rfc9110.html#field.retry-after
func RetryAfterBackoff() Backoff {
	return HeaderBackoff("Retry-After", nil)
}

// jitter returns a random duration in the range [0, d/2). d cannot be negative.
// It is called by Backoff functions to add variability to the delay.
// If BackoffJitterEnabled is false, it returns 0.
func jitter(d time.Duration) time.Duration {
	if BackoffJitterEnabled {
		return time.Duration(float64(d) * rand.Float64() * 0.5)
	}
	return 0
}
