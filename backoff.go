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

// BackoffPolicy defines a mapping between HTTP status codes and their
// corresponding backoff strategies. Each key in the map is a string containing
// one or more space-separated HTTP status codes. The associated value is the
// strategy to apply when an HTTP response returns one of the specified status
// codes.
//
// For example:
//
//	BackoffPolicy{
//	    "429":     meergo.RetryAfterStrategy(),
//	    "500 503": meergo.ExponentialStrategy(1 * time.Second),
//	}
type BackoffPolicy map[string]BackoffStrategy

// BackoffStrategy represents a backoff strategy. It returns the duration to
// wait before the next attempt, based on the response from the previous attempt
// and the number of retries made. retries starts at 0 before the first retry
// and increments by 1 after each retry.
//
// If the returned duration is negative, it is considered zero.
// It returns NoRetry if the request should not be retried.
type BackoffStrategy func(res *http.Response, retries int) (time.Duration, error)

// ConstantStrategy returns a backoff strategy implementing a constant backoff.
func ConstantStrategy(waitTime time.Duration) BackoffStrategy {
	return func(res *http.Response, retries int) (time.Duration, error) {
		return max(0, waitTime), nil
	}
}

// ExponentialStrategy returns an exponential backoff strategy based on the
// provided base duration.
func ExponentialStrategy(base time.Duration) BackoffStrategy {
	b := float64(base.Milliseconds())
	return func(res *http.Response, retries int) (time.Duration, error) {
		d := time.Duration(b*math.Pow(2, float64(retries))) * time.Millisecond // base * 2^retries
		d = min(d, BackoffCap)                                                 // limit to cap
		return d + jitter(d), nil
	}
}

// HeaderStrategy returns a backoff strategy that determines the wait time from
// the provided response header. parse is used to parse the header's value. If
// parse is nil, it behaves like RetryAfterStrategy but uses the specified
// header instead of "Retry-After". If parse returns an error, HeaderStrategy
// will return a NoRetry error to indicate no retry should be attempted.
func HeaderStrategy(header string, parse func(s string) (time.Duration, error)) BackoffStrategy {
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

// RetryAfterStrategy returns a backoff strategy that determines the wait time
// from the Retry-After header. Unlike the standard, it also accepts seconds
// expressed with decimal values.
// https://httpwg.org/specs/rfc6585.html#status-429
// https://httpwg.org/specs/rfc9110.html#field.retry-after
func RetryAfterStrategy() BackoffStrategy {
	return HeaderStrategy("Retry-After", nil)
}

// jitter returns a random duration in the range [0, d/2). d cannot be negative.
// It is called by backoff strategy to add variability to the delay.
// If BackoffJitterEnabled is false, it returns 0.
func jitter(d time.Duration) time.Duration {
	if BackoffJitterEnabled {
		return time.Duration(float64(d) * rand.Float64() * 0.5)
	}
	return 0
}
