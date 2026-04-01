// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/krenalis/krenalis/connectors"
)

// Test the main state transitions and behaviors of the rate limiter.
func TestRateLimiter_BasicStates(t *testing.T) {

	l := newRateLimiter(10, 3, 0)

	// Initial state should be normal.
	l.mu.Lock()
	if l.state != normal {
		t.Errorf("initial state: got %v, want normal", l.state)
	}
	l.mu.Unlock()

	// OnSuccess in normal state should not change the state.
	l.OnSuccess(0)
	l.mu.Lock()
	if l.state != normal {
		t.Errorf("OnSuccess in normal: got %v, want normal", l.state)
	}
	l.mu.Unlock()

	// OnFailure with NetFailure and low error rate should not change state.
	l.OnFailure(0, connectors.NetFailure, 0)
	l.mu.Lock()
	if l.state != normal {
		t.Errorf("OnFailure at low errorRate: got %v, want normal", l.state)
	}
	l.mu.Unlock()

	// OnFailure with Slowdown should transition to slowdown state and set min error rate.
	l.OnFailure(0, connectors.Slowdown, 0)
	l.mu.Lock()
	if l.state != slowdown {
		t.Errorf("OnFailure with Slowdown: got %v, want slowdown", l.state)
	}
	if l.errorRate.rate < minSlowdownErrorRate {
		t.Errorf("OnSlowdown did not set min errorRate: got %f, want >=%f", l.errorRate.rate, minSlowdownErrorRate)
	}
	l.mu.Unlock()

	// Recovery: force a return to normal by directly resetting the error rate.
	l.mu.Lock()
	l.errorRate.Set(0)
	l.mu.Unlock()
	l.OnSuccess(0)
	l.mu.Lock()
	if l.state != normal {
		t.Errorf("Did not return to normal after forced recovery, got %v", l.state)
	}
	l.mu.Unlock()
}

// Test that rate limiter correctly enters rateLimited and pause is respected.
func TestRateLimiter_RateLimitPause(t *testing.T) {

	l := newRateLimiter(10, 3, 0)

	// Pause for a long time to guarantee the pause is in effect during Wait
	l.OnFailure(0, connectors.RateLimited, 1*time.Second)
	l.mu.Lock()
	if l.state != rateLimited {
		t.Errorf("RateLimit: got %v, want rateLimited", l.state)
	}
	l.mu.Unlock()

	// Use a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := l.Wait(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Wait error = %v, want context deadline exceeded", err)
	}
}

// Test that the minimum refill rate never drops to zero, even with maximum errorRate.
func TestRateLimiter_MinRateNeverZero(t *testing.T) {

	l := newRateLimiter(10, 3, 0)

	// Force errorRate to 1 (100% failures)
	l.mu.Lock()
	l.errorRate.Set(1)
	// Set last refill time to now
	l.refilling.last = time.Now()
	l.mu.Unlock()

	// Calculate the minimum allowed rate (should never be less than minScale * max)
	minRate := l.refilling.max * minScale

	// Wait enough time to earn at least 1 token at minRate (time = 1 token / minRate)
	wait := time.Duration(float64(time.Second) / minRate)
	time.Sleep(wait)

	tokens := l.refill()
	if tokens == 0 {
		t.Errorf("refill returned zero tokens at max errorRate, want >0 (minRate=%v, waited=%v)", minRate, wait)
	}
}

// Test that the refill scaling matches expectations for given error rates.
func TestRateLimiter_RateScaling(t *testing.T) {
	l := newRateLimiter(100, 10, 0)
	steps := []struct {
		errorRate float64
		wantScale float64
	}{
		{0, 1},
		{0.5, 1 - 0.5*(1-minScale)},
		{1, minScale},
	}
	for _, step := range steps {
		l.mu.Lock()
		l.errorRate.Set(step.errorRate)
		// Set last refill to exactly 1 second ago.
		base := time.Now().Add(-1 * time.Second)
		l.refilling.last = base
		l.mu.Unlock()

		// Call refill and calculate tokens.
		tokens := l.refill()

		// Compute how many seconds actually passed using the same reference.
		l.mu.Lock()
		seconds := l.refilling.last.Sub(base).Seconds()
		l.mu.Unlock()

		want := int(l.refilling.max * step.wantScale * seconds)
		// Allow up to 1 token of rounding difference.
		if diff := tokens - want; diff < -1 || diff > 1 {
			t.Errorf("errorRate=%v: got %d tokens, want %d (diff %d)", step.errorRate, tokens, want, diff)
		}
	}
}

// Checks thread safety with concurrent successful requests.
func TestRateLimiter_ConcurrentAccess_SuccessOnly(t *testing.T) {

	l := newRateLimiter(1000, 500, 0)
	var wg sync.WaitGroup
	errCh := make(chan error, 4*100)

	ctx := context.Background()

	for range 4 {
		wg.Go(func() {
			for range 100 {
				err := l.Wait(ctx)
				if err != nil {
					errCh <- fmt.Errorf("unexpected error: %s", err)
					return
				}
				l.OnSuccess(0)
			}
		})
	}
	wg.Wait()

	close(errCh)
	for err := range errCh {
		t.Errorf("%v", err)
	}
}

// Checks rate scaling down on frequent errors.
func TestRateLimiter_ErrorRateScaling(t *testing.T) {

	l := newRateLimiter(10, 3, 0)
	n := 50

	// Simulate a burst of failures to increase error rate quickly.
	for range n {
		l.errorRate.Failure()
	}

	// Calculate current allowed rate based on error rate.
	l.mu.Lock()
	currRate := l.refilling.max * (1 - l.errorRate.rate*(1-minScale))
	l.mu.Unlock()

	// Check that the current rate is reduced from the initial value.
	if currRate >= l.refilling.max {
		t.Errorf("rate did not decrease as expected: got %v", currRate)
	}

	// Log the value if it's not exactly minScale due to float precision.
	if math.Abs(currRate-10*minScale) > 1e-6 {
		t.Logf("current rate after many failures: %v (expected around %v)", currRate, 10*minScale)
	}
}

// Checks that maxConcurrency is never exceeded.
func TestRateLimiter_MaxConcurrency(t *testing.T) {
	l := newRateLimiter(100, 100, 3) // maximum 3 concurrent
	start := make(chan struct{})
	var running int32
	var maxRunning int32
	var wg sync.WaitGroup

	for range 10 {
		wg.Go(func() {
			<-start // synchronize start
			err := l.Wait(context.Background())
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			n := atomic.AddInt32(&running, 1)
			// Atomically update the maximum ever seen.
			for {
				old := atomic.LoadInt32(&maxRunning)
				if n <= old {
					break
				}
				if atomic.CompareAndSwapInt32(&maxRunning, old, n) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&running, -1)
			l.OnSuccess(0)
		})
	}
	close(start)
	wg.Wait()
	if maxRunning > 3 {
		t.Errorf("concurrent requests exceeded maxConcurrency: got %d, want <= 3", maxRunning)
	}
}

// Checks token bucket refills as expected.
func TestRateLimiter_TokenRefill(t *testing.T) {
	l := newRateLimiter(2, 2, 0)
	ctx := context.Background()
	for range 2 {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		l.OnSuccess(0)
	}
	// Now the bucket is empty: the next call should block for ~0.5s.
	start := time.Now()
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	elapsed := time.Since(start)
	l.OnSuccess(0)
	if elapsed < 400*time.Millisecond {
		t.Errorf("token bucket did not wait as expected, waited %v", elapsed)
	}
}

// Checks state transitions on different error types.
func TestRateLimiter_StateTransitions(t *testing.T) {

	l := newRateLimiter(10, 3, 0)
	ctx := context.Background()

	// Bring to slowdown state
	for range 10 {
		if err := l.Wait(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		l.OnFailure(0, connectors.Slowdown, 0)
	}
	l.mu.Lock()
	state := l.state
	l.mu.Unlock()
	if state != slowdown {
		t.Errorf("expected state=slowdown, got %v", state)
	}

	// Bring to rateLimited state
	if err := l.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l.OnFailure(0, connectors.RateLimited, time.Millisecond)
	l.mu.Lock()
	state = l.state
	l.mu.Unlock()
	if state != rateLimited {
		t.Errorf("expected state=rateLimited, got %v", state)
	}
}

// Test the WaitTime method under various conditions.
func TestRateLimiter_WaitTime(t *testing.T) {
	l := newRateLimiter(1, 1, 0)

	// With an available token and no pause, WaitTime should be zero.
	if d := l.WaitTime(); d != 0 {
		t.Fatalf("initial WaitTime = %v, want 0", d)
	}

	// Consume the only token so the bucket becomes empty.
	if err := l.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	l.OnSuccess(0)

	// Immediately after consuming, WaitTime should be close to 1 second.
	if d := l.WaitTime(); d < 900*time.Millisecond || d > time.Second {
		t.Fatalf("waitTime without tokens = %v, want about 1s", d)
	}

	// Enter rate limit state with a short pause.
	l.OnFailure(0, connectors.RateLimited, 200*time.Millisecond)
	if d := l.WaitTime(); d < 180*time.Millisecond {
		t.Fatalf("waitTime during pause = %v, want >= 180ms", d)
	}

	// WaitTime should also account for concurrent requests in flight.
	l2 := newRateLimiter(1000, 2, 1)
	ctx := context.Background()

	// Record an average latency of about 100ms.
	if err := l2.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	l2.OnSuccess(100 * time.Millisecond)

	// Start another request without completing it to fill the concurrency slot.
	if err := l2.Wait(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// With the slot occupied, WaitTime should reflect the observed latency.
	if d := l2.WaitTime(); d < 80*time.Millisecond {
		t.Fatalf("waitTime with full concurrency = %v, want >= 80ms", d)
	}

	l2.OnSuccess(100 * time.Millisecond)
}
