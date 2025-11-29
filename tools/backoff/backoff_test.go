// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package backoff

import (
	"context"
	"math"
	"testing"
	"time"
)

// Test_AfterFunc ensures the provided function is executed the expected number
// of times depending on the attempts cap.
func Test_AfterFunc(t *testing.T) {

	c := make(chan struct{})
	f := func(_ context.Context) { c <- struct{}{} }

	// Test NoLimit attempts.
	cap := 10 * time.Millisecond
	bo := New(1)
	bo.SetCap(cap)
	i := 0
	const attemptsCap = 10
	for {
		ok := bo.AfterFunc(context.Background(), f)
		if !ok {
			t.Fatalf("AfterFunc: expected true, got false")
		}
		select {
		case <-c:
		case <-time.NewTimer(cap * 10).C:
			t.Fatalf("function has not been called")
		}
		i++
		if i == attemptsCap {
			break
		}
	}

	// Tests from 1 to 5 attempts.
	cap = time.Second
	for attempts := 1; attempts < 5; attempts++ {
		bo := New(1)
		bo.SetAttempts(attempts)
		bo.SetCap(cap)
		ctx := context.Background()
		i := 0
		for bo.AfterFunc(ctx, f) {
			select {
			case <-c:
			case <-time.NewTimer(cap * 2).C:
				t.Fatalf("function has not been called")
			}
			i++
		}
		if i != attempts {
			t.Fatalf("expected %d attempts, got %d", attempts, i)
		}
	}

}

// Test_AfterFunc_Context verifies that AfterFunc stops when the context is
// canceled or times out.
func Test_AfterFunc_Context(t *testing.T) {

	c := make(chan struct{})

	cap := 2 * time.Second
	for i := 0; i < 10; i++ {
		bo := New(1)
		bo.SetCap(cap)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Duration(i)*time.Millisecond)
		defer cancel()
		var done bool
		f := func(ctx context.Context) {
			done = ctx.Err() != nil
			c <- struct{}{}
		}
		for !done && bo.AfterFunc(ctx, f) {
			select {
			case <-c:
			case <-time.NewTimer(cap * 2).C:
				t.Fatalf("function has not been called")
			}
		}
		err := ctx.Err()
		if err == nil {
			t.Fatal("backoff exited but the context has not been canceled")
		}
	}

}

// Test_Attempt checks that Attempt returns the correct attempt counter when
// calling Next.
func Test_Attempt(t *testing.T) {
	bo := New(1)
	bo.SetAttempts(5)
	bo.SetCap(2 * time.Second)
	i := 0
	for bo.Next(context.Background()) {
		i++
		if got := bo.Attempt(); i != got {
			t.Fatalf("expected attempt %d, got %d", i, got)
		}
	}
}

// Ensure attempts saturate at math.MaxInt
func Test_AttemptSaturates(t *testing.T) {
	bo := New(1)
	bo.attempt = math.MaxInt - 1
	bo.Next(context.Background()) // increment to MaxInt
	bo.Next(context.Background()) // should remain MaxInt
	if bo.Attempt() != math.MaxInt {
		t.Fatalf("expected attempt %d, got %d", math.MaxInt, bo.Attempt())
	}
}

// Test constructors and setters for panic on invalid input and proper state update
func Test_InvalidInputPanics(t *testing.T) {
	type panicFunc func()
	tests := []panicFunc{
		func() { New(-1) },
		func() { New(1).SetBase(-1) },
		func() { New(1).SetAttempts(0) },
		func() { New(1).SetAttempts(-2) },
		func() { New(1).SetCap(0) },
		func() { New(1).SetCap(-time.Second) },
		func() { New(1).SetNextWaitTime(0) },
		func() { New(1).SetNextWaitTime(-time.Second) },
	}
	for i, f := range tests {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatalf("test %d: expected panic", i)
				}
			}()
			f()
		}()
	}
}

// Test_Next checks the return value of Next and that it respects the attempts
// limit when set.
func Test_Next(t *testing.T) {

	// Test NoLimit attempts.
	bo := New(1)
	bo.SetCap(10 * time.Millisecond)
	i := 0
	const attemptsCap = 10
	for bo.Next(context.Background()) {
		i++
		if i == attemptsCap {
			break
		}
	}
	if i != attemptsCap {
		t.Fatalf("expected no limit, got %d attempt limit", i)
	}

	// Tests from 1 to 5 attempts.
	for attempts := 1; attempts < 5; attempts++ {
		bo := New(1)
		bo.SetAttempts(attempts)
		bo.SetCap(time.Second)
		ctx := context.Background()
		i := 0
		for bo.Next(ctx) {
			i++
		}
		if i != attempts {
			t.Fatalf("expected %d attempts, got %d", attempts, i)
		}
	}

}

// Test_Next_Context verifies Next exits when the context is done.
func Test_Next_Context(t *testing.T) {

	for i := 0; i < 10; i++ {
		bo := New(1)
		bo.SetCap(2 * time.Second)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Duration(i)*time.Millisecond)
		defer cancel()
		for bo.Next(ctx) {
		}
		err := ctx.Err()
		if err == nil {
			t.Fatal("backoff exited but the context has not been canceled")
		}
	}

}

// Test_Next_Cap asserts WaitTime never exceeds the configured cap.
func Test_Next_Cap(t *testing.T) {
	for i := 1; i < 10; i++ {
		cap := time.Duration(i) * time.Millisecond
		bo := New(1)
		bo.SetAttempts(10)
		bo.SetCap(cap)
		for bo.Next(context.Background()) {
			if got := bo.WaitTime(); got > cap {
				t.Fatalf("wait time (%s) is greater than cap (%s)", got, cap)
			}
		}
	}
}

// Test_SetNextWaitTime validates that SetNextWaitTime overrides the next
// calculated wait time.
func Test_SetNextWaitTime(t *testing.T) {
	bo := New(1)
	bo.SetAttempts(5)
	bo.SetCap(time.Second)
	for {
		wt := 1 + time.Duration(bo.Attempt())*time.Millisecond
		bo.SetNextWaitTime(wt)
		if got := bo.WaitTime(); wt != got {
			t.Fatalf("expected waiting time %s, got %s", wt, got)
		}
		if !bo.Next(context.Background()) || bo.WaitTime() == 0 {
			break
		}
	}
}

// Test setters actually modify the internal state
func Test_SettersUpdateFields(t *testing.T) {
	bo := New(1)
	bo.SetBase(5)
	if bo.base != 5 {
		t.Fatalf("expected base 5, got %v", bo.base)
	}
	bo.SetAttempts(3)
	if bo.attempts != 3 {
		t.Fatalf("expected attempts 3, got %d", bo.attempts)
	}
	bo.SetCap(10 * time.Millisecond)
	if bo.cap != 10*time.Millisecond {
		t.Fatalf("expected cap 10ms, got %s", bo.cap)
	}
}

// Test_Stop checks that Stop prevents future executions and returns true only
// on the first call after stopping.
func Test_Stop(t *testing.T) {
	bo := New(10000)
	if bo.Stop() {
		t.Fatalf("expected false, got true")
	}
	ch := make(chan struct{})
	bo.AfterFunc(context.Background(), func(context.Context) {
		if bo.Stop() {
			t.Fatalf("expected false, got true")
		}
		ch <- struct{}{}
	})
	<-ch
	bo.SetNextWaitTime(1)
	bo.AfterFunc(context.Background(), func(context.Context) {
		t.Fatalf("unexpected call after the backoff has been stopped")
	})
	if !bo.Stop() {
		t.Fatalf("expected true, got false")
	}
	// Wait a bit to ensure no unexpected goroutines run after Stop.
	time.Sleep(10 * time.Millisecond)
}

// Test_WaitTime ensures WaitTime changes across attempts and resets to zero at
// the expected points.
func Test_WaitTime(t *testing.T) {
	ctx := context.Background()
	bo := New(1)
	if got := bo.WaitTime(); got != 0 {
		t.Fatalf("expected waiting time 0, got %s", got)
	}
	bo.SetAttempts(2)
	bo.Next(ctx)
	if got := bo.WaitTime(); got < time.Millisecond || got >= 3*time.Millisecond {
		t.Fatalf("expected waiting time in range [1,3), got %s", got)
	}
	bo.Next(ctx)
	if got := bo.WaitTime(); got != 0 {
		t.Fatalf("expected waiting time 0, got %s", got)
	}
}

// WaitTime with base 0 should always be 1ms after the first attempt
func Test_WaitTime_BaseZero(t *testing.T) {
	bo := New(0)
	bo.SetAttempts(2)
	if !bo.Next(context.Background()) {
		t.Fatal("Next returned false on first call")
	}
	wt := bo.WaitTime()
	if wt != time.Millisecond {
		t.Fatalf("expected wait time 1ms, got %s", wt)
	}
}
