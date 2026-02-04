// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

// Package backoff implements an exponential backoff algorithm with jitter.
//
// For the backoff New(base), the first waiting time is zero, and subsequent
// waiting times are calculated in milliseconds as a random value within the
// range [1, 1 + (base * 2^attempt)] with attempt > 0.
//
// How to use:
//
//	bo := backoff.New(base)
//	bo.SetNextWaitTime(10 * time.Millisecond) // waits 10ms before the first attempt.
//	for bo.Next(ctx) {
//		err := doSomething()
//		if err != nil {
//			if bo.WaitTime() > 0 {
//				log.Printf("try attempt %d after %s", bo.Attempt(), bo.WaitTime())
//			}
//			continue
//		}
//		break
//	}
//
// Use AfterFunc for non-blocking execution. The provided function will run in a
// separate goroutine after the wait time or upon context cancellation.
//
//	bo.AfterFunc(ctx, func(ctx context.Context) {
//		// do something.
//	})
package backoff

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

const defaultCap = 15 * time.Minute

var randFloat64 = rand.Float64

// Backoff implements an exponential backoff algorithm with jitter.
type Backoff struct {
	attempts int
	base     float64
	cap      time.Duration
	attempt  int
	waitTime time.Duration
	timer    *time.Timer
}

// New returns a new Backoff with the given base, with unlimited attempts and
// a default cap of 15 minutes. It panics if base < 0.
func New(base int) *Backoff {
	if base < 0 {
		panic("backoff: base is negative")
	}
	return &Backoff{base: float64(base)}
}

// AfterFunc calls f, if another attempt can be made, in its own goroutine after
// the bo.WaitTime() duration, and returns true. If no other attempts can be
// made, it returns false. It calls f even if the context is canceled and does
// so as soon as possible after cancellation.
func (bo *Backoff) AfterFunc(ctx context.Context, f func(ctx context.Context)) bool {
	if bo.attempt > 0 {
		if bo.attempt == bo.attempts {
			return false
		}
		if bo.waitTime == 0 {
			bo.setWaitTime()
		}
	}
	if bo.attempt < math.MaxInt {
		bo.attempt++
	}
	if bo.waitTime == 0 {
		go f(ctx)
		return true
	}
	if bo.timer == nil {
		bo.timer = time.NewTimer(bo.waitTime)
	} else {
		bo.timer.Reset(bo.waitTime)
	}
	bo.waitTime = 0
	go func() {
		select {
		case <-ctx.Done():
		case <-bo.timer.C:
		}
		f(ctx)
	}()
	return true
}

// Attempt returns the current attempt within the range [0, maxInt]. It returns
// 0 if Next has not been called yet and returns maxInt if the current attempt
// count is maxInt or greater.
func (bo *Backoff) Attempt() int {
	return bo.attempt
}

// Next waits for bo.WaitTime() and returns true if another attempt can be
// made, otherwise, it returns false.
func (bo *Backoff) Next(ctx context.Context) bool {
	if bo.attempt > 0 {
		if bo.attempt == bo.attempts {
			return false
		}
		if bo.waitTime == 0 {
			bo.setWaitTime()
		}
	}
	if bo.waitTime > 0 {
		if bo.timer == nil {
			bo.timer = time.NewTimer(bo.waitTime)
		} else {
			bo.timer.Reset(bo.waitTime)
		}
		bo.waitTime = 0
		select {
		case <-ctx.Done():
			return false
		case <-bo.timer.C:
		}
	}
	if bo.attempt < math.MaxInt {
		bo.attempt++
	}
	return true
}

// SetAttempts sets the attempts. Use backoff.NoLimit for unlimited attempts.
// It panics if attempts is zero or negative.
func (bo *Backoff) SetAttempts(attempts int) {
	if attempts <= 0 {
		panic("backoff: attempts is zero or negative")
	}
	bo.attempts = attempts
}

// SetBase sets the base. It panics if base is negative.
func (bo *Backoff) SetBase(base int) {
	if base < 0 {
		panic("backoff: base is negative")
	}
	bo.base = float64(base)
}

// SetCap sets the cap. It panics if cap is less than 1ms.
func (bo *Backoff) SetCap(cap time.Duration) {
	if cap < time.Millisecond {
		panic("backoff: cap is less than 1ms")
	}
	bo.cap = cap
}

// SetNextWaitTime sets the wait time for the next attempt.
// It panics if d is zero or negative.
func (bo *Backoff) SetNextWaitTime(d time.Duration) {
	if d == 0 {
		panic("backoff: wait time is zero")
	}
	if d < 0 {
		panic("backoff: wait time is negative")
	}
	bo.waitTime = d
}

// Stop prevents the execution of the function passed to AfterFunc.
// It returns true if a scheduled function was successfully stopped from
// executing, and returns false if there was no scheduled function, either
// because AfterFunc was not called or the function has already been executed.
func (bo *Backoff) Stop() bool {
	if bo.timer == nil {
		return false
	}
	return bo.timer.Stop()
}

// WaitTime returns the wait time for the next retry attempt in the range
// [min, max), where min is 1ms and max is 1 + base * 2^attempt milliseconds,
// but never greater than the cap (defaults to 15 minutes if not set).
// As a special case, it returns 0 if the Next and AfterFunc methods have not
// already been called or if there are no other retry attempts.
func (bo *Backoff) WaitTime() time.Duration {
	if bo.attempt > 0 {
		if bo.attempt == bo.attempts {
			return 0
		}
		if bo.waitTime == 0 {
			bo.setWaitTime()
		}
	}
	return bo.waitTime
}

// setWaitTime sets the wait time.
func (bo *Backoff) setWaitTime() {
	capDuration := bo.cap
	if capDuration == 0 {
		capDuration = defaultCap
	}

	// Base 0 degenerates to a fixed 1ms wait.
	if bo.base == 0 {
		bo.waitTime = time.Millisecond
		if bo.waitTime > capDuration {
			bo.waitTime = capDuration
		}
		return
	}

	capMs := float64(capDuration / time.Millisecond)
	if capMs < 1 {
		capMs = 1
	}

	upperMs := bo.base
	if upperMs < 1 {
		upperMs = 1
	}
	for i := 0; i < bo.attempt && upperMs < capMs; i++ {
		upperMs *= 2
	}

	maxMs := 1 + upperMs
	if maxMs > capMs {
		maxMs = capMs
	}

	bo.waitTime = time.Duration(1+randFloat64()*(maxMs-1)) * time.Millisecond
	if bo.waitTime > capDuration {
		bo.waitTime = capDuration
	}
}
