//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

package httpclient

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// tokenBucket manages available tokens and goroutines waiting for tokens.
// Implements a token bucket algorithm with a blocking Take method.
type tokenBucket struct {
	tokens    chan struct{} // tokens currently available
	ticker    *time.Ticker  // ticker to trigger periodic refills if Take is not called
	refill    func() int    // function returning the number of tokens to refill
	refilling atomic.Bool   // true if a refill is currently in progress
	pause     struct {
		atomic.Bool             // true if the bucket is currently paused
		sync.Mutex              // mutex acquired when entering or leaving the paused state
		until       time.Time   // time until which the bucket remains paused
		timer       *time.Timer // timer to automatically end the pause
	}
}

// newTokenBucket returns a new tokenBucket with the given capacity and
// refilling function. It panics if capacity <= 0 or refill is nil.
func newTokenBucket(capacity int, refill func() int) *tokenBucket {
	if capacity <= 0 {
		panic("core/connectors/httpclient: capacity must be > 0")
	}
	if refill == nil {
		panic("core/connectors/httpclient: refill cannot be nil")
	}
	b := &tokenBucket{
		tokens: make(chan struct{}, capacity),
		ticker: time.NewTicker(time.Second),
		refill: refill,
	}
	for range capacity {
		b.tokens <- struct{}{}
	}
	return b
}

// Pause pauses the token bucket for the duration d.
// If it is already paused, it extends the duration if appropriate.
func (b *tokenBucket) Pause(d time.Duration) {
	b.pause.Lock()
	defer b.pause.Unlock()
	b.pause.Store(true)
	until := time.Now().Add(d)
	if !until.After(b.pause.until) {
		return
	}
	b.pause.until = until
	if b.pause.timer != nil {
		b.pause.timer.Reset(d)
		return
	}
	b.pause.timer = time.AfterFunc(d, func() {
		b.pause.Lock()
		if time.Since(b.pause.until) >= 0 {
			b.pause.Store(false)
		}
		b.pause.Unlock()
	})
}

// Take acquires one token from the bucket. If no token is available,
// Take blocks until a token is released or the context is canceled.
// Take is safe for concurrent use by multiple goroutines.
func (b *tokenBucket) Take(ctx context.Context) error {
	if !b.pause.Load() {
		select {
		case <-b.tokens:
			return nil
		default:
		}
	}
	for {
		if !b.pause.Load() && b.refilling.CompareAndSwap(false, true) {
			b.add(b.refill())
			b.refilling.Store(false)
		}
		select {
		case <-b.tokens:
			if !b.pause.Load() {
				return nil
			}
		case <-b.ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// add adds n tokens to the bucket.
func (b *tokenBucket) add(n int) {
	if n < 0 {
		panic("core/connections/httpclient: invalid n")
	}
	for range n {
		select {
		case b.tokens <- struct{}{}:
			continue
		default:
			return
		}
	}
}
