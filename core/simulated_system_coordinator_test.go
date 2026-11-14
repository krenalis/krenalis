// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// TestSimulatedSystemCoordinatorClose waits for an executor already stopping
// and prevents a requested restart from escaping shutdown.
func TestSimulatedSystemCoordinatorClose(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		release := make(chan struct{})
		passes := 0
		c := newSimulatedSystemCoordinator(t.Context(), func(ctx context.Context) bool {
			passes++
			<-ctx.Done()
			<-release
			return true
		})
		defer func() {
			select {
			case <-release:
			default:
				close(release)
			}
			c.close()
		}()
		c.setLeader(true)
		synctest.Wait()
		c.setLeader(false)
		c.setLeader(true)
		closed := make(chan struct{})
		go func() {
			c.close()
			close(closed)
		}()
		synctest.Wait()
		select {
		case <-closed:
			t.Fatal("expected shutdown waiting for the executor, got completed shutdown")
		default:
		}

		close(release)
		<-closed
		c.setLeader(true)
		c.wakeUp()
		synctest.Wait()
		if passes != 1 {
			t.Fatalf("expected one pass before shutdown, got %d", passes)
		}

	})
}

// TestSimulatedSystemCoordinatorLeadership checks startup recovery, idle
// behavior and repeated election handling without periodic scans.
func TestSimulatedSystemCoordinatorLeadership(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		passes := 0
		c := newSimulatedSystemCoordinator(t.Context(), func(context.Context) bool {
			passes++
			return false
		})
		defer c.close()
		c.setLeader(false)
		c.wakeUp()
		time.Sleep(time.Minute)
		synctest.Wait()
		if passes != 0 {
			t.Fatalf("expected no follower passes, got %d", passes)
		}

		c.setLeader(true)
		synctest.Wait()
		if passes != 1 {
			t.Fatalf("expected one startup recovery pass, got %d", passes)
		}
		c.setLeader(true)
		time.Sleep(time.Minute)
		synctest.Wait()
		if passes != 1 {
			t.Fatalf("expected no idle or duplicate executor passes, got %d", passes)
		}

		c.wakeUp()
		synctest.Wait()
		if passes != 2 {
			t.Fatalf("expected two passes after a notification, got %d", passes)
		}
		c.setLeader(false)
		synctest.Wait()
		c.wakeUp()
		time.Sleep(time.Minute)
		synctest.Wait()
		if passes != 2 {
			t.Fatalf("expected no passes after losing leadership, got %d", passes)
		}

	})
}

// TestSimulatedSystemCoordinatorRapidElection preserves a requested restart
// while the canceled executor is still finishing its operation.
func TestSimulatedSystemCoordinatorRapidElection(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		release := make(chan struct{})
		var contexts []context.Context
		c := newSimulatedSystemCoordinator(t.Context(), func(ctx context.Context) bool {
			contexts = append(contexts, ctx)
			if len(contexts) == 1 {
				<-ctx.Done()
				<-release
			}
			return false
		})
		defer func() {
			select {
			case <-release:
			default:
				close(release)
			}
			c.close()
		}()
		c.setLeader(true)
		synctest.Wait()
		c.setLeader(false)
		c.setLeader(true)
		c.wakeUp()
		synctest.Wait()
		if len(contexts) != 1 || contexts[0].Err() == nil {
			t.Fatalf("expected one canceled executor awaiting exit, got %d contexts", len(contexts))
		}

		close(release)
		synctest.Wait()
		if len(contexts) != 2 || contexts[1].Err() != nil || contexts[0] == contexts[1] {
			t.Fatalf("expected a fresh executor after the previous exit, got %d contexts", len(contexts))
		}

	})
}

// TestSimulatedSystemCoordinatorRetry checks conditional retry, including
// notifications received before a retry deadline.
func TestSimulatedSystemCoordinatorRetry(t *testing.T) {
	for _, test := range []struct {
		name       string
		notify     bool
		completeAt int
	}{
		{"timer", false, 2},
		{"notification before deadline", true, 3},
		{"notification clears retry", true, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {

				var passes atomic.Int64
				c := newSimulatedSystemCoordinator(t.Context(), func(context.Context) bool {
					return passes.Add(1) < int64(test.completeAt)
				})
				defer c.close()
				c.setLeader(true)
				synctest.Wait()
				time.Sleep(2 * time.Second)
				if test.notify {
					c.wakeUp()
					synctest.Wait()
				}
				before := passes.Load()
				time.Sleep(3*time.Second - time.Nanosecond)
				synctest.Wait()
				if actual := passes.Load(); actual != before {
					t.Fatalf("expected %d passes before the retry deadline, got %d", before, actual)
				}
				time.Sleep(time.Nanosecond)
				synctest.Wait()
				if actual := passes.Load(); actual != int64(test.completeAt) {
					t.Fatalf("expected %d passes at the retry deadline, got %d", test.completeAt, actual)
				}
				time.Sleep(time.Minute)
				synctest.Wait()
				if actual := passes.Load(); actual != int64(test.completeAt) {
					t.Fatalf("expected no retry after successful processing, got %d passes", actual)
				}

			})
		})
	}
}

// TestSimulatedSystemCoordinatorSignals retains notifications received
// during a pass without requiring a periodic scan.
func TestSimulatedSystemCoordinatorSignals(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		release := make(chan struct{})
		passes := 0
		c := newSimulatedSystemCoordinator(t.Context(), func(context.Context) bool {
			passes++
			if passes == 1 {
				<-release
			}
			return false
		})
		defer c.close()
		c.setLeader(true)
		synctest.Wait()
		for range 100 {
			c.wakeUp()
		}
		close(release)
		synctest.Wait()
		if passes != 2 {
			t.Fatalf("expected two passes for coalesced notifications, got %d", passes)
		}

	})
}
