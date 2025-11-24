// Copyright 2025 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"io"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// Verifies reads succeed and the timeout does not fire when calls stay within
// the window.
func TestTimeoutReaderReadBeforeTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 50 * time.Millisecond
		timeoutHit := make(chan struct{}, 1) // buffered to avoid blocking the callback

		r := newTimeoutReader(strings.NewReader("hello"), timeout, func() {
			timeoutHit <- struct{}{} // record that the timeout was reached
		})
		defer r.Close()

		buf := make([]byte, 2)
		n, err := r.Read(buf)
		if err != nil {
			t.Fatalf("unexpected error on first read: %v", err)
		}
		if got := string(buf[:n]); got != "he" {
			t.Fatalf("unexpected data from first read: %q", got)
		}

		time.Sleep(timeout / 4) // advances the fake clock without waiting real time

		buf = make([]byte, 3)
		n, err = r.Read(buf)
		if err != nil {
			t.Fatalf("unexpected error on second read: %v", err)
		}
		if n != 3 {
			t.Fatalf("second read length mismatch: got %d, want 3", n)
		}
		if got := string(buf[:n]); got != "llo" {
			t.Fatalf("second read content mismatch: got %q, want %q", got, "llo")
		}

		n, err = r.Read(buf[:1])
		if n != 0 {
			t.Fatalf("expected EOF after consuming reader, got n=%d err=%v", n, err)
		}
		if err != io.EOF {
			t.Fatalf("expected EOF after consuming reader, got err=%v", err)
		}

		select {
		case <-timeoutHit:
			t.Fatalf("timeout callback fired during timely reads, timer should reset on each call")
		default:
		}
	})
}

// Verifies that after an initial read, idling past the timeout fires the
// callback and subsequent reads fail.
func TestTimeoutReaderTimeoutAfterRead(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 20 * time.Millisecond
		timeoutHit := make(chan struct{}, 1)

		r := newTimeoutReader(strings.NewReader("abc"), timeout, func() {
			timeoutHit <- struct{}{}
		})
		defer r.Close()

		buf := make([]byte, 1)
		if n, err := r.Read(buf); n != 1 || err != nil {
			t.Fatalf("first read failed: n=%d err=%v", n, err)
		}

		time.Sleep(timeout + timeout/2)
		synctest.Wait()

		select {
		case <-timeoutHit:
		default:
			t.Fatalf("timeout callback not fired after idle period")
		}

		n, err := r.Read(buf)
		if n != 0 {
			t.Fatalf("expected timed-out reader error after idle, got n=%d err=%v", n, err)
		}
		if err == nil {
			t.Fatalf("expected timed-out reader error after idle, got nil")
		}
		if err.Error() != "read on a timed-out reader" {
			t.Fatalf("expected timed-out reader error after idle, got %v", err)
		}
	})
}

// Verifies the timeout callback fires after inactivity and subsequent reads
// fail.
func TestTimeoutReaderTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 20 * time.Millisecond
		timeoutHit := make(chan struct{}, 1)

		r := newTimeoutReader(strings.NewReader("data"), timeout, func() {
			timeoutHit <- struct{}{}
		})
		defer r.Close()

		time.Sleep(timeout + timeout/2) // let the timer elapse in simulated time
		synctest.Wait()                 // ensure the AfterFunc goroutine runs

		select {
		case <-timeoutHit:
		default:
			t.Fatalf("timeout callback not fired after inactivity")
		}

		buf := make([]byte, 1)
		n, err := r.Read(buf)
		if n != 0 {
			t.Fatalf("unexpected data length after timeout: got %d, want 0", n)
		}
		if err == nil {
			t.Fatalf("expected timeout error after inactivity, got nil")
		}
		if err.Error() != "read on a timed-out reader" {
			t.Fatalf("unexpected error after timeout: got %q", err)
		}
	})
}

// Verifies closing stops the timer and prevents further reads.
func TestTimeoutReaderCloseStopsTimer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const timeout = 30 * time.Millisecond
		timeoutHit := make(chan struct{}, 1)

		r := newTimeoutReader(strings.NewReader("data"), timeout, func() {
			timeoutHit <- struct{}{}
		})

		if err := r.Close(); err != nil {
			t.Fatalf("close returned error: %v", err)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("second close should be idempotent, got error: %v", err)
		}

		time.Sleep(timeout + timeout/2)
		synctest.Wait() // allow any outstanding timers to fire if they were not stopped

		select {
		case <-timeoutHit:
			t.Fatalf("timeout callback fired after close")
		default:
		}

		buf := make([]byte, 1)
		n, err := r.Read(buf)
		if n != 0 {
			t.Fatalf("expected closed reader error, got n=%d err=%v", n, err)
		}
		if err == nil {
			t.Fatalf("expected closed reader error, got nil")
		}
		if err.Error() != "read on a closed reader" {
			t.Fatalf("expected closed reader error, got %v", err)
		}
	})
}
