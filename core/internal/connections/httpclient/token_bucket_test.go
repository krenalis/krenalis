// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package httpclient

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Test that the token bucket refills and unblocks Take after depletion.
func TestTokenBucket_BasicUsage(t *testing.T) {
	capacity := 3
	refillCount := 2
	refillCh := make(chan struct{}, 1)
	defer close(refillCh)

	// Simple refill function: always returns refillCount when triggered
	refill := func() int {
		refillCh <- struct{}{}
		return refillCount
	}

	tb := newTokenBucket(capacity, refill)

	// Drain the bucket completely
	for i := 0; i < capacity; i++ {
		if err := tb.Take(context.Background()); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Now the bucket is empty; the next Take will block and refill will be triggered by the ticker (every 1s)
	done := make(chan struct{})
	go func() {
		err := tb.Take(context.Background())
		if err != nil {
			t.Errorf("Take failed: %v", err)
		}
		close(done)
	}()

	// Wait for the refill function to be called; allow enough time for the ticker to fire (up to 1.5s)
	select {
	case <-refillCh:
		// Refill was triggered as expected
	case <-time.After(1500 * time.Millisecond):
		t.Fatalf("refill not called in time (ticker interval is 1s)")
	}

	// Wait for Take to complete after refill
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("Take did not complete after refill")
	}
}

// Checks that the bucket never exceeds its capacity even if refill tries to
// overfill.
func TestTokenBucket_Capacity(t *testing.T) {
	capacity := 2
	refill := func() int { return 10 } // Tries to overfill the bucket

	tb := newTokenBucket(capacity, refill)

	// Consume all available tokens
	for i := 0; i < capacity; i++ {
		_ = tb.Take(context.Background())
	}

	// Trigger refill by calling Take in a goroutine
	done := make(chan struct{})
	go func() {
		_ = tb.Take(context.Background())
		close(done)
	}()

	// Wait for refill to happen
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("Take did not complete")
	}

	// Only (capacity-1) tokens should remain
	for i := 0; i < capacity-1; i++ {
		select {
		case <-tb.tokens:
		default:
			t.Fatalf("bucket should be at capacity, but is empty early")
		}
	}

	// No extra tokens should be present
	select {
	case <-tb.tokens:
		t.Fatalf("bucket overfilled, got extra token")
	default:
		// Correct
	}
}

// Test that Take blocks while the token bucket is paused and resumes only after
// the pause ends and a token is available.
func TestTokenBucket_PauseAndResume(t *testing.T) {

	tb := newTokenBucket(1, func() int { return 1 })

	// Consume initial token
	if err := tb.Take(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Pause the bucket for 200ms
	tb.Pause(200 * time.Millisecond)

	// Channel to notify when Take returns
	done := make(chan time.Time)

	// Start goroutine that will call Take (should block)
	go func() {
		err := tb.Take(context.Background())
		if err != nil {
			t.Errorf("Take failed: %v", err)
		}
		done <- time.Now()
	}()

	// Wait a bit and ensure Take is still blocked
	time.Sleep(100 * time.Millisecond)
	select {
	case <-done:
		t.Fatalf("Take should be blocked during pause")
	default:
	}

	// Wait until after pause duration and simulate token refill
	time.Sleep(150 * time.Millisecond) // total 250ms > 200ms pause

	tb.add(1) // Add a token to allow the blocked Take to proceed

	// Now the goroutine should finish soon
	select {
	case finish := <-done:
		elapsed := finish.Sub(time.Now().Add(-250 * time.Millisecond))
		if elapsed < 200*time.Millisecond {
			t.Errorf("pause was too short: %v", elapsed)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("Take did not resume after pause")
	}
}

// Tests that Take returns an error when the context is canceled and does not
// return too early.
func TestTokenBucket_TakeCanceled(t *testing.T) {
	tb := newTokenBucket(1, func() int { return 0 })

	// Consume token
	_ = tb.Take(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := tb.Take(ctx)
	if err == nil {
		t.Fatalf("expected error on canceled context")
	}
	elapsed := time.Since(start)
	// Allow some leeway for timing variations and goroutine scheduling.
	if elapsed < 90*time.Millisecond {
		t.Errorf("Take returned too quickly: got %v, want at least 90ms (context deadline)", elapsed)
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("Take took too long to return: got %v, want less than 300ms", elapsed)
	}
}

// TestTokenBucket_ConcurrentTake verifies that concurrent Take calls succeed as
// tokens are refilled automatically.
func TestTokenBucket_ConcurrentTake(t *testing.T) {
	tb := newTokenBucket(2, func() int { return 2 })

	var wg sync.WaitGroup
	const n = 10
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			err := tb.Take(context.Background())
			if err != nil {
				t.Errorf("Take failed: %v", err)
			}
		}()
	}
	// Wait enough for all goroutines to block on Take and for the ticker to refill tokens.
	time.Sleep(2 * time.Second)
	wg.Wait()
}
