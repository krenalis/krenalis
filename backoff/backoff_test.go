//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2023 Open2b
//

package backoff

import (
	"context"
	"testing"
	"time"
)

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
