//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package httpclient

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/meergo/meergo"
)

// rateLimiterState represents the rate limiter's state.
type rateLimiterState int

const (
	normal      rateLimiterState = iota // operating at full rate
	slowdown                            // throttling due to increased errors
	rateLimited                         // paused due to server rate limiting
)

const minScale = 0.2             // minimum rate scale factor during slowdown
const minSlowdownErrorRate = 0.3 // minimum error rate to trigger slowdown

// rateLimiter implements an adaptive rate limiter with error-rate-based
// scaling. The allowed request rate is dynamically reduced as recent failures
// increase, never dropping below minScale * refilling.max. Optionally, it can
// also limit the maximum number of concurrent (in-flight) requests.
//
// It is the caller's responsibility to ensure that every call to Wait is always
// followed by either OnSuccess or OnFailure, regardless of request outcome.
type rateLimiter struct {

	// inFlight controls the maximum number of concurrent (in-flight) requests.
	// If nil, there is no concurrency limit.
	inFlight chan struct{}

	tokens *tokenBucket

	mu        sync.Mutex       // protects state, errorRate, and refilling.
	state     rateLimiterState // state; protected by mu
	errorRate errorRate        // error rate; protected by mu
	refilling struct {
		max  float64   // max allowed requests per second in normal state; protected by mu
		last time.Time // last refill timestamp; protected by mu
	}
}

// newRateLimiter returns a rate limiter configured with the specified max rate
// (requests per second), capacity (burst size), and maxConcurrency (maximum
// number of concurrent requests). If maxConcurrency is zero, there is no limit
// on concurrent requests.
// It panics if rate <= 0, capacity <= 0, or maxConcurrency < 0.
func newRateLimiter(rate float64, capacity, maxConcurrency int) *rateLimiter {
	if rate <= 0 {
		panic("core/connectors/httpclient: rate must be > 0")
	}
	if maxConcurrency < 0 {
		panic("core/connectors/httpclient: maxConcurrency must be >= 0")
	}
	b := &rateLimiter{}
	if maxConcurrency != 0 {
		b.inFlight = make(chan struct{}, maxConcurrency)
	}
	b.refilling.max = rate
	b.refilling.last = time.Now()
	b.tokens = newTokenBucket(capacity, b.refill)
	return b
}

// OnFailure must be called after a request completes with an error or is
// cancelled. For every call to Wait, either OnSuccess or OnFailure must be
// called to avoid leaking concurrency slots.
func (b *rateLimiter) OnFailure(reason meergo.FailureReason, waitTime time.Duration) {
	if b.inFlight != nil {
		<-b.inFlight
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch reason {
	case meergo.PermanentFailure:
	case meergo.NetFailure:
		b.onNetFailure()
	case meergo.Slowdown:
		b.onSlowdown()
	case meergo.RateLimited:
		b.onRateLimited(waitTime)
	default:
		panic(fmt.Errorf("core/connectors/httpclient: unexpected FailureReason %d", reason))
	}
}

// OnSuccess must be called after a request completes successfully. For every
// call to Wait, either OnSuccess or OnFailure must be called to avoid leaking
// concurrency slots.
func (b *rateLimiter) OnSuccess() {
	if b.inFlight != nil {
		<-b.inFlight
	}
	b.mu.Lock()
	switch b.state {
	case normal:
		b.errorRate.Success()
	case slowdown:
		b.errorRate.Success()
		if b.errorRate.rate < 0.1 {
			// slowdown -> normal
			b.state = normal
		}
	case rateLimited:
	}
	b.mu.Unlock()
}

// Wait blocks until a token and (if enabled) a concurrency slot are available.
// For every call to Wait, you must later call either OnSuccess or OnFailure to
// release the concurrency slot. Failing to do so may cause the limiter to block
// indefinitely due to exhausted concurrency slots.
func (b *rateLimiter) Wait(ctx context.Context) error {
	if b.inFlight != nil {
		select {
		case b.inFlight <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return b.tokens.Take(ctx)
}

func (b *rateLimiter) onNetFailure() {
	switch b.state {
	case normal, slowdown:
		b.errorRate.Failure()
	case rateLimited:
	}
}

func (b *rateLimiter) onRateLimited(wt time.Duration) {
	switch b.state {
	case rateLimited:
	default:
		// normal/slowdown -> rateLimited
		b.state = rateLimited
	}
	b.tokens.Pause(wt)
}

func (b *rateLimiter) onSlowdown() {
	switch b.state {
	case normal:
		// normal -> slowdown
		b.state = slowdown
		if b.errorRate.rate < minSlowdownErrorRate {
			b.errorRate.Set(minSlowdownErrorRate) // set to mid-error to force partial slowdown
		} else {
			b.errorRate.Failure()
		}
	case slowdown:
		b.errorRate.Failure()
	case rateLimited:
	}
}

func (b *rateLimiter) refill() int {
	b.mu.Lock()
	// Compute the rate.
	scale := 1 - b.errorRate.rate*(1-minScale)
	rate := b.refilling.max * scale
	// Compute the tokens.
	seconds := time.Since(b.refilling.last).Seconds()
	tokens := int(seconds * rate)
	if tokens > 0 {
		delta := time.Duration(float64(tokens) / rate * float64(time.Second))
		b.refilling.last = b.refilling.last.Add(delta)
	}
	b.mu.Unlock()
	return tokens
}
