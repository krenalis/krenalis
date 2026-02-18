// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package datastore

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

// testRow is a minimal row type used in flusher tests.
type testRow struct {
	id int
}

// startOperationStub is a no-op StartOperation used in tests.
func startOperationStub(ctx context.Context, _ allowedMode) (context.Context, func(), error) {
	return ctx, func() {}, nil
}

// baseOptions returns the default flusherOptions for tests.
func baseOptions() flusherOptions {
	return flusherOptions{
		QueueSize:       10,
		BatchSize:       10,
		MaxBatchSize:    10,
		IdleFlushDelay:  time.Hour,
		MaxFlushLatency: time.Hour,
		RateAlpha:       0.5,
	}
}

// newTestFlusher constructs a flusher with the test StartOperation stub.
func newTestFlusher(opts flusherOptions, flush flushFunc[testRow]) *flusher[testRow] {
	return newFlusher[testRow](opts, startOperationStub, flush)
}

// waitChannelClosed waits until the flusher input channel is closed.
func waitChannelClosed[T any](t *testing.T, ch <-chan flusherRow[T]) {
	t.Helper()
	timeout := time.After(1 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatalf("expected channel closed, got open")
		}
	}
}

// receiveNow reads a value from ch without blocking.
func receiveNow[T any](t *testing.T, ch <-chan T, expected string) (v T) {
	t.Helper()
	select {
	case v = <-ch:
		return v
	default:
	}
	t.Fatalf("expected %s, got none", expected)
	return v
}

// receiveWithin reads a value from ch within the provided timeout.
func receiveWithin[T any](t *testing.T, ch <-chan T, d time.Duration, expected string) (v T) {
	t.Helper()
	select {
	case v = <-ch:
		return v
	case <-time.After(d):
	}
	t.Fatalf("expected %s, got none", expected)
	return v
}

// TestFlusherFlushOnMaxBatchSize verifies flushing when MaxBatchSize is
// reached.
func TestFlusherFlushOnMaxBatchSize(t *testing.T) {
	opts := baseOptions()
	opts.BatchSize = 2
	opts.MaxBatchSize = 3
	opts.MinFlushInterval = time.Hour
	flushCh := make(chan int, 1)
	var flushCount atomic.Int32
	flushFn := func(ctx context.Context, rows []testRow) error {
		flushCount.Add(1)
		select {
		case flushCh <- len(rows):
		default:
		}
		return nil
	}
	f := newTestFlusher(opts, flushFn)
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 2}}
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 3}}

	got := receiveWithin(t, flushCh, 1*time.Second, "flush")
	if got != opts.MaxBatchSize {
		t.Fatalf("expected %d, got %d", opts.MaxBatchSize, got)
	}

	if err := f.Stop(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if got := flushCount.Load(); got != 1 {
		t.Fatalf("expected %d, got %d", 1, got)
	}
	waitChannelClosed(t, f.rows)
}

// TestFlusherIdleFlush verifies idle-time flushing.
func TestFlusherIdleFlush(t *testing.T) {
	// synctest keeps timer-driven idle flush deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 5
		opts.MaxBatchSize = 5
		opts.IdleFlushDelay = 2 * time.Second
		opts.MaxFlushLatency = 10 * time.Second
		flushCh := make(chan []testRow, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- append([]testRow(nil), rows...)
			return nil
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}

		time.Sleep(opts.IdleFlushDelay)
		synctest.Wait()

		got := receiveNow(t, flushCh, "flush")
		if len(got) != 1 {
			t.Fatalf("expected %d, got %d", 1, len(got))
		}

		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
	})
}

// TestFlusherMaxFlushLatency verifies the max-latency deadline is enforced.
func TestFlusherMaxFlushLatency(t *testing.T) {
	// synctest keeps max-latency timers deterministic under continuous arrivals.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 100
		opts.MaxBatchSize = 100
		opts.MaxFlushLatency = 5 * time.Second
		flushCh := make(chan time.Time, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- time.Now()
			return nil
		}
		f := newTestFlusher(opts, flushFn)

		start := time.Now()
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 0}}

		for i := 1; ; i++ {
			select {
			case flushTime := <-flushCh:
				if flushTime.Sub(start) > opts.MaxFlushLatency {
					t.Fatalf("expected <= %v, got %v", opts.MaxFlushLatency, flushTime.Sub(start))
				}
				if err := f.Stop(context.Background()); err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				waitChannelClosed(t, f.rows)
				return
			default:
			}
			time.Sleep(1 * time.Second)
			f.Ch() <- flusherRow[testRow]{row: testRow{id: i}}
		}
	})
}

// TestFlusherAdaptiveFlushRespectsMinInterval verifies adaptive scheduling
// respects limits.
func TestFlusherAdaptiveFlushRespectsMinInterval(t *testing.T) {
	// synctest keeps adaptive scheduling deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 10
		opts.MaxBatchSize = 20
		opts.MinFlushInterval = 500 * time.Millisecond
		opts.IdleFlushDelay = 2 * time.Second
		opts.MaxFlushLatency = 3 * time.Second
		flushCh := make(chan time.Time, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- time.Now()
			return nil
		}
		f := newTestFlusher(opts, flushFn)

		start := time.Now()
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}
		for i := 2; ; i++ {
			select {
			case flushTime := <-flushCh:
				elapsed := flushTime.Sub(start)
				if elapsed < opts.MinFlushInterval {
					t.Fatalf("expected >= %v, got %v", opts.MinFlushInterval, elapsed)
				}
				if elapsed > opts.MaxFlushLatency {
					t.Fatalf("expected <= %v, got %v", opts.MaxFlushLatency, elapsed)
				}
				if elapsed >= opts.IdleFlushDelay {
					t.Fatalf("expected < %v, got %v", opts.IdleFlushDelay, elapsed)
				}
				if err := f.Stop(context.Background()); err != nil {
					t.Fatalf("expected nil, got %v", err)
				}
				waitChannelClosed(t, f.rows)
				return
			default:
			}
			time.Sleep(100 * time.Millisecond)
			f.Ch() <- flusherRow[testRow]{row: testRow{id: i}}
		}
	})
}

// TestFlusherDedupSemantics verifies deduplication rules.
func TestFlusherDedupSemantics(t *testing.T) {
	opts := baseOptions()

	cases := []struct {
		name string
		key1 any
		key2 any
		want []testRow
	}{
		{
			name: "dedup-enabled",
			key1: "key",
			key2: "key",
			want: []testRow{{id: 2}},
		},
		{
			name: "dedup-disabled",
			key1: nil,
			key2: nil,
			want: []testRow{{id: 1}, {id: 2}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flushCh := make(chan []testRow, 1)
			flushFn := func(ctx context.Context, rows []testRow) error {
				flushCh <- append([]testRow(nil), rows...)
				return nil
			}
			f := newTestFlusher(opts, flushFn)
			f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, key: tc.key1}
			f.Ch() <- flusherRow[testRow]{row: testRow{id: 2}, key: tc.key2}

			if err := f.Close(context.Background()); err != nil {
				t.Fatalf("expected nil, got %v", err)
			}
			waitChannelClosed(t, f.rows)

			got := receiveNow(t, flushCh, "flush")
			if len(got) != len(tc.want) {
				t.Fatalf("expected %d, got %d", len(tc.want), len(got))
			}
			for i := range got {
				if got[i].id != tc.want[i].id {
					t.Fatalf("expected %v, got %v", tc.want, got)
				}
			}
		})
	}
}

// TestFlusherAckAfterRetry verifies acks only after a successful retry.
func TestFlusherAckAfterRetry(t *testing.T) {
	// synctest keeps backoff retries deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 2
		opts.MaxBatchSize = 2
		var ackCount atomic.Int32
		acked := make(chan struct{}, 1)
		ack := func() {
			ackCount.Add(1)
			acked <- struct{}{}
		}

		thirdStart := make(chan struct{}, 1)
		allowSuccess := make(chan struct{})
		var mu sync.Mutex
		attempt := 0
		flushFn := func(ctx context.Context, rows []testRow) error {
			mu.Lock()
			attempt++
			n := attempt
			mu.Unlock()
			if n <= 2 {
				return errors.New("retry")
			}
			thirdStart <- struct{}{}
			<-allowSuccess
			return nil
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, ack: nil}
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 2}, ack: ack}

		<-thirdStart
		if got := ackCount.Load(); got != 0 {
			t.Fatalf("expected %d, got %d", 0, got)
		}
		close(allowSuccess)
		synctest.Wait()

		receiveNow(t, acked, "ack")
		if got := ackCount.Load(); got != 1 {
			t.Fatalf("expected %d, got %d", 1, got)
		}

		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
	})
}

// TestFlusherStartOperationRetry verifies StartOperation errors are retried.
func TestFlusherStartOperationRetry(t *testing.T) {
	// synctest keeps backoff retries deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 1
		opts.MaxBatchSize = 1

		var attempts atomic.Int32
		startOp := func(ctx context.Context, _ allowedMode) (context.Context, func(), error) {
			if attempts.Add(1) <= 2 {
				return nil, nil, errors.New("start operation failed")
			}
			return ctx, func() {}, nil
		}

		flushCh := make(chan struct{}, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- struct{}{}
			return nil
		}
		acked := make(chan struct{}, 1)
		f := newFlusher[testRow](opts, startOp, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, ack: func() { acked <- struct{}{} }}

		receiveWithin(t, flushCh, 30*time.Second, "flush")
		receiveWithin(t, acked, 30*time.Second, "ack")

		if got := attempts.Load(); got != 3 {
			t.Fatalf("expected %d, got %d", 3, got)
		}
		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
	})
}

// TestFlusherStartOperationCanceled verifies Stop/Close aborts persistent
// StartOperation failures.
func TestFlusherStartOperationCanceled(t *testing.T) {
	// synctest keeps backoff retries deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 1
		opts.MaxBatchSize = 1

		started := make(chan struct{})
		var attempts atomic.Int32
		startOp := func(ctx context.Context, _ allowedMode) (context.Context, func(), error) {
			if attempts.Add(1) == 1 {
				close(started)
			}
			return nil, nil, errors.New("start operation failed")
		}

		var flushCount atomic.Int32
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCount.Add(1)
			return nil
		}
		acked := make(chan struct{}, 1)
		f := newFlusher[testRow](opts, startOp, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, ack: func() { acked <- struct{}{} }}

		<-started
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := f.Stop(ctx)
		if err == nil {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}

		if got := attempts.Load(); got == 0 {
			t.Fatalf("expected > %d, got %d", 0, got)
		}
		if got := flushCount.Load(); got != 0 {
			t.Fatalf("expected %d, got %d", 0, got)
		}
		select {
		case <-acked:
			t.Fatalf("expected no ack, got ack")
		default:
		}
		waitChannelClosed(t, f.rows)
	})
}

// TestFlusherMetricsAggregation verifies per-pipeline aggregation and
// finalization.
func TestFlusherMetricsAggregation(t *testing.T) {
	finalizeCh := make(chan struct{}, 2)
	var mu sync.Mutex
	finalized := make(map[int]int)
	opts := baseOptions()
	opts.BatchSize = 4
	opts.MaxBatchSize = 4
	opts.MetricsFinalizer = func(pipeline, count int) {
		mu.Lock()
		finalized[pipeline] += count
		mu.Unlock()
		finalizeCh <- struct{}{}
	}
	flushCh := make(chan struct{}, 1)
	flushFn := func(ctx context.Context, rows []testRow) error {
		flushCh <- struct{}{}
		return nil
	}
	f := newTestFlusher(opts, flushFn)
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, pipeline: 1}
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 2}, pipeline: 1}
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 3}, pipeline: 2}
	f.Ch() <- flusherRow[testRow]{row: testRow{id: 4}, pipeline: 0}

	receiveWithin(t, flushCh, 1*time.Second, "flush")

	for range 2 {
		receiveWithin(t, finalizeCh, 1*time.Second, "metrics finalize")
	}

	mu.Lock()
	gotPipeline1 := finalized[1]
	gotPipeline2 := finalized[2]
	_, hasZero := finalized[0]
	mu.Unlock()

	if gotPipeline1 != 2 {
		t.Fatalf("expected %d, got %d", 2, gotPipeline1)
	}
	if gotPipeline2 != 1 {
		t.Fatalf("expected %d, got %d", 1, gotPipeline2)
	}
	if hasZero {
		t.Fatalf("expected %v, got %v", false, true)
	}

	if err := f.Stop(context.Background()); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	waitChannelClosed(t, f.rows)
}

// TestFlusherRetryLogErrorDedup verifies logError de-duplication across
// retries.
func TestFlusherRetryLogErrorDedup(t *testing.T) {
	// synctest keeps backoff retries deterministic without real sleeps.
	synctest.Test(t, func(t *testing.T) {
		finalizeCh := make(chan struct{}, 1)
		var mu sync.Mutex
		finalized := make(map[int]int)
		var logMu sync.Mutex
		var logErrors []string
		logFn := func(err error) {
			logMu.Lock()
			logErrors = append(logErrors, err.Error())
			logMu.Unlock()
		}
		opts := baseOptions()
		opts.BatchSize = 1
		opts.MaxBatchSize = 1
		opts.MetricsFinalizer = func(pipeline, count int) {
			mu.Lock()
			finalized[pipeline] += count
			mu.Unlock()
			finalizeCh <- struct{}{}
		}
		opts.LogError = logFn
		attempt := 0
		errorsSeq := []error{
			errors.New("alpha"),
			errors.New("alpha"),
			errors.New("beta"),
		}
		ackCh := make(chan struct{}, 1)
		ack := func() {
			ackCh <- struct{}{}
		}
		flushFn := func(ctx context.Context, rows []testRow) error {
			if attempt < len(errorsSeq) {
				err := errorsSeq[attempt]
				attempt++
				return err
			}
			return nil
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}, pipeline: 1, ack: ack}

		receiveWithin(t, ackCh, 30*time.Second, "ack")
		receiveWithin(t, finalizeCh, 30*time.Second, "metrics finalize")

		logMu.Lock()
		gotErrors := append([]string(nil), logErrors...)
		logMu.Unlock()
		if len(gotErrors) != 2 {
			t.Fatalf("expected %d, got %d", 2, len(gotErrors))
		}
		if gotErrors[0] != "alpha" {
			t.Fatalf("expected %v, got %v", "alpha", gotErrors[0])
		}
		if gotErrors[1] != "beta" {
			t.Fatalf("expected %v, got %v", "beta", gotErrors[1])
		}
		mu.Lock()
		gotFinalized := finalized[1]
		mu.Unlock()
		if gotFinalized != 1 {
			t.Fatalf("expected %d, got %d", 1, gotFinalized)
		}

		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
	})
}

// TestFlusherCloseStopSemantics verifies Close and Stop behaviors.
func TestFlusherCloseStopSemantics(t *testing.T) {
	opts := baseOptions()

	t.Run("close-drains", func(t *testing.T) {
		flushCh := make(chan struct{}, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- struct{}{}
			return nil
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}
		if err := f.Close(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
		receiveNow(t, flushCh, "flush")
		if err := f.Close(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})

	t.Run("stop-does-not-drain", func(t *testing.T) {
		flushCh := make(chan struct{}, 1)
		flushFn := func(ctx context.Context, rows []testRow) error {
			flushCh <- struct{}{}
			return nil
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}
		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
		waitChannelClosed(t, f.rows)
		select {
		case <-flushCh:
			t.Fatalf("expected no flush, got flush")
		default:
		}
		if err := f.Stop(context.Background()); err != nil {
			t.Fatalf("expected nil, got %v", err)
		}
	})
}

// TestFlusherInFlightCancellationStop verifies Stop cancels in-flight flushes
// on timeout.
func TestFlusherInFlightCancellationStop(t *testing.T) {
	runInFlightCancellation(t, func(ctx context.Context, f *flusher[testRow]) error { return f.Stop(ctx) })
}

// TestFlusherInFlightCancellationClose verifies Close cancels in-flight flushes
// on timeout.
func TestFlusherInFlightCancellationClose(t *testing.T) {
	runInFlightCancellation(t, func(ctx context.Context, f *flusher[testRow]) error { return f.Close(ctx) })
}

func runInFlightCancellation(t *testing.T, stopFn func(ctx context.Context, f *flusher[testRow]) error) {
	// synctest keeps context timeouts deterministic when stopping a blocked flush.
	synctest.Test(t, func(t *testing.T) {
		opts := baseOptions()
		opts.BatchSize = 1
		opts.MaxBatchSize = 1

		started := make(chan struct{})
		flushFn := func(ctx context.Context, rows []testRow) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		}
		f := newTestFlusher(opts, flushFn)
		f.Ch() <- flusherRow[testRow]{row: testRow{id: 1}}
		<-started

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		err := stopFn(ctx, f)
		if err == nil {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected %v, got %v", context.DeadlineExceeded, err)
		}
		waitChannelClosed(t, f.rows)
	})
}
