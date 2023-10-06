//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

// Package backoff implements an exponential backoff algorithm with jitter.
//
// For the backoff New(attempts, base, cap), the waiting time is calculated in
// milliseconds as a random value within the range [0, min(base * 2^attempt, cap)]
// where attempt varies in the range [1, attempts].
//
// How to use:
//
//	bo := backoff.New(attempts, 10, 2*time.Second) // use backoff.NoLimit for unlimited attempts
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
package backoff

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// NoLimit indicates that there are no attempt limits.
const NoLimit = 0

// Backoff implements an exponential backoff algorithm with jitter.
type Backoff struct {
	attempts int
	base     float64
	cap      time.Duration
	attempt  int
	waitTime time.Duration
}

// New returns a new Backoff with the given attempts, base and cap.
// If attempts is 0 (NoLimit), the attempts are not limited.
// It panics if attempts < 0 or base < 0.
func New(attempts, base int, cap time.Duration) *Backoff {
	if attempts < 0 || base < 0 {
		panic("backoff: invalid argument")
	}
	return &Backoff{attempts, float64(base), cap, 0, 0}
}

// Attempt returns the current attempt within the range [0, maxInt]. It returns
// 0 if Step has not been called yet and returns maxInt if the current attempt
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
		ctx2, cancel := context.WithTimeout(ctx, bo.waitTime)
		defer cancel()
		bo.waitTime = 0
		select {
		case <-ctx2.Done():
		}
		if ctx.Err() != nil {
			return false
		}
	}
	if bo.attempt < math.MaxInt {
		bo.attempt++
	}
	return true
}

// SetNextWaitTime sets the wait time for the next attempt.
func (bo *Backoff) SetNextWaitTime(d time.Duration) {
	bo.waitTime = d
}

// WaitTime returns the wait time for the next retry attempt. If there are no
// other retry attempts, it returns 0.
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
	// waitTime = min(random_between(0, base * 2^attempt), cap)
	bo.waitTime = min(time.Duration(1+rand.Float64()*bo.base*math.Pow(2, float64(bo.attempt)))*time.Millisecond, bo.cap)
}
