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
	bo := New(NoLimit, 1, cap)
	i := 0
	const attemptsCap = 10
	for {
		ok := bo.AfterFunc(context.Background(), f)
		if !ok {
			t.Fatalf("AfterFunc: expecting true, got false")
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
		bo := New(attempts, 1, cap)
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
		bo := New(NoLimit, 1, cap)
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
	bo := New(5, 1, 2*time.Second)
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
	bo := New(NoLimit, 1, 10*time.Millisecond)
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
		bo := New(attempts, 1, time.Second)
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
		bo := New(NoLimit, 1, 2*time.Second)
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
		bo := New(10, 1, cap)
		for bo.Next(context.Background()) {
			if got := bo.WaitTime(); got > cap {
				t.Fatalf("wait time (%s) is greater than cap (%s)", got, cap)
			}
		}
	}
}

func Test_SetNextWaitTime(t *testing.T) {
	bo := New(5, 1, time.Second)
	for {
		wt := time.Duration(bo.Attempt()) * time.Millisecond
		bo.SetNextWaitTime(wt)
		if got := bo.WaitTime(); wt != got {
			t.Fatalf("expected waiting time %s, got %s", wt, got)
		}
		if !bo.Next(context.Background()) || bo.WaitTime() == 0 {
			break
		}
	}
}
