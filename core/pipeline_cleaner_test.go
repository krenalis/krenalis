// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/krenalis/krenalis/core/internal/metrics"
	"github.com/krenalis/krenalis/tools/errors"
)

// TestPipelineCleanerCloseCancelsMetricsWait checks that a shutdown timeout
// cancels a pending metrics wait.
func TestPipelineCleanerCloseCancelsMetricsWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		c := newTestPipelineCleaner(t)
		m := metrics.New(nil, nil, nil)
		defer m.Close(t.Context())
		result := make(chan error, 1)
		c.start(func() {
			_, err := c.complete(func() error { return m.Pipelines.WaitStore(c.close.ctx) })
			result <- err
		})
		synctest.Wait()
		select {
		case err := <-result:
			t.Fatalf("expected pending metrics wait before Close, got %v", err)
		default:
		}

		const timeout = 100 * time.Millisecond
		ctx, cancel := context.WithTimeout(t.Context(), timeout)
		defer cancel()
		start := time.Now()
		c.Close(ctx)
		if elapsed := time.Since(start); elapsed != timeout {
			t.Fatalf("expected Close to wait %s for shutdown cancellation, got %s", timeout, elapsed)
		}
		err := <-result
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context.Canceled, got %v", err)
			}
			return
		}

		t.Fatal("expected context.Canceled, got nil")

	})
}

// TestPipelineCleanerCloseRejectsConcurrentWork verifies that new work is
// rejected during Close while an existing operation is still running.
func TestPipelineCleanerCloseRejectsConcurrentWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		c := newTestPipelineCleaner(t)
		release := make(chan struct{})
		c.start(func() {
			ran, err := c.complete(func() error {
				<-release
				return nil
			})
			if err != nil {
				t.Errorf("expected successful initial operation, got %v", err)
			}
			if !ran {
				t.Error("expected initial operation admitted, got rejected operation")
			}
		})
		synctest.Wait()
		closed := make(chan struct{})
		go func() {
			c.Close(t.Context())
			close(closed)
		}()
		synctest.Wait()

		var callers sync.WaitGroup
		for range 32 {
			callers.Go(func() {
				for range 100 {
					ran, err := c.complete(func() error {
						t.Error("expected operation rejected during Close, got executed operation")
						return nil
					})
					if err != nil {
						t.Errorf("expected nil for rejected operation, got %v", err)
					}
					if ran {
						t.Error("expected operation rejected during Close, got admitted operation")
					}
					c.start(func() { t.Error("expected goroutine rejected during Close, got started goroutine") })
				}
			})
		}
		callers.Wait()
		synctest.Wait()
		select {
		case <-closed:
			t.Error("expected Close waiting for initial operation, got completed Close")
		default:
		}

		close(release)
		<-closed
		c.Close(t.Context())

	})
}

// TestPipelineCleanerCloseStopsPeriodicTasks verifies that Close stops
// both periodic tasks before returning.
func TestPipelineCleanerCloseStopsPeriodicTasks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		c := newTestPipelineCleaner(t)
		functionsExited := make(chan struct{})
		orphanedRunsExited := make(chan struct{})
		c.start(func() {
			c.deleteDiscontinuedFunctions()
			close(functionsExited)
		})
		c.start(func() {
			c.terminateOrphanedRuns()
			close(orphanedRunsExited)
		})
		synctest.Wait()
		select {
		case <-functionsExited:
			t.Fatal("expected deleteDiscontinuedFunctions running before Close, got exited goroutine")
		default:
		}
		select {
		case <-orphanedRunsExited:
			t.Fatal("expected terminateOrphanedRuns running before Close, got exited goroutine")
		default:
		}

		c.Close(t.Context())
		select {
		case <-functionsExited:
		default:
			t.Error("expected deleteDiscontinuedFunctions exited after Close, got active goroutine")
		}
		select {
		case <-orphanedRunsExited:
		default:
			t.Error("expected terminateOrphanedRuns exited after Close, got active goroutine")
		}
		select {
		case <-c.close.ctx.Done():
		default:
			t.Error("expected canceled internal context, got active context")
		}

	})
}

// TestPipelineCleanerCloseWaitsForOperationsAndGoroutines verifies that Close
// waits for ongoing operations and goroutines to finish.
func TestPipelineCleanerCloseWaitsForOperationsAndGoroutines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {

		c := newTestPipelineCleaner(t)
		releaseOperation := make(chan struct{})
		operationDone := make(chan struct{})
		releaseGoroutine := make(chan struct{})
		goroutineDone := make(chan struct{})
		c.start(func() {
			_, err := c.complete(func() error {
				select {
				case <-releaseOperation:
					close(operationDone)
					return nil
				case <-c.close.ctx.Done():
					return c.close.ctx.Err()
				}
			})
			if err != nil {
				t.Errorf("expected completed operation, got %v", err)
			}
			<-c.close.ctx.Done()
			<-releaseGoroutine
			close(goroutineDone)
		})
		synctest.Wait()

		closed := make(chan struct{})
		go func() {
			c.Close(t.Context())
			close(closed)
		}()
		synctest.Wait()
		select {
		case <-closed:
			t.Error("expected Close waiting for operation, got completed Close")
		default:
		}
		if err := c.close.ctx.Err(); err != nil {
			t.Errorf("expected active operation context, got %v", err)
		}

		close(releaseOperation)
		synctest.Wait()
		select {
		case <-operationDone:
		default:
			t.Error("expected completed operation, got pending operation")
		}
		select {
		case <-c.close.ctx.Done():
		default:
			t.Error("expected canceled internal context after operation completion, got active context")
		}
		select {
		case <-closed:
			t.Error("expected Close waiting for goroutine, got completed Close")
		default:
		}

		close(releaseGoroutine)
		<-closed
		select {
		case <-goroutineDone:
		default:
			t.Error("expected exited goroutine after Close, got active goroutine")
		}

	})
}

// TestPipelineCleanerCompleteReturnsOperationError verifies that complete
// propagates the error returned by the operation.
func TestPipelineCleanerCompleteReturnsOperationError(t *testing.T) {
	c := newTestPipelineCleaner(t)
	defer c.Close(t.Context())
	expected := errors.New("operation failed")
	ran, err := c.complete(func() error { return expected })
	if !ran {
		t.Fatal("expected operation admitted, got rejected operation")
	}
	if err != nil {
		if !errors.Is(err, expected) {
			t.Fatalf("expected operation error %v, got %v", expected, err)
		}
		return
	}
	t.Fatal("expected operation error, got nil")
}

func newTestPipelineCleaner(t *testing.T) *pipelineCleaner {
	t.Helper()
	c := &pipelineCleaner{}
	c.close.ctx, c.close.cancel = context.WithCancel(t.Context())
	t.Cleanup(c.close.cancel)
	return c
}
