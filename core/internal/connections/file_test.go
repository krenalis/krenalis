// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package connections

import (
	"errors"
	"io"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

// TestMaterializeReadSeeker verifies that a streaming reader is materialized as
// a read seeker with offset-read support.
func TestMaterializeReadSeeker(t *testing.T) {

	original := io.ReadCloser(io.NopCloser(strings.NewReader("abcdef")))
	rc, rs, err := materializeReadSeeker(&original)
	if err != nil {
		t.Fatalf("materializeReadSeeker failed: %v", err)
	}
	defer rc.Close()
	if _, ok := original.(closedReadCloser); !ok {
		t.Fatalf("expected original reader type closedReadCloser, got %T", original)
	}

	if _, ok := any(rs).(io.ReaderAt); !ok {
		t.Fatal("expected materialized reader to implement io.ReaderAt")
	}
	if _, ok := any(rs).(io.Closer); ok {
		t.Fatal("materialized reader exposed io.Closer to the connector")
	}
	if _, ok := any(rs).(interface{ Name() string }); ok {
		t.Fatal("materialized reader exposed Name to the connector")
	}

	buf := make([]byte, 2)
	if n, err := rs.Read(buf); err != nil || n != 2 || string(buf) != "ab" {
		t.Fatalf("expected Read to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "ab", n, buf, err)
	}
	if _, err := rs.Seek(4, io.SeekStart); err != nil {
		t.Fatalf("expected Seek to succeed, got %v", err)
	}
	if n, err := rs.Read(buf); err != nil || n != 2 || string(buf) != "ef" {
		t.Fatalf("expected Read after Seek to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "ef", n, buf, err)
	}
	if n, err := any(rs).(io.ReaderAt).ReadAt(buf, 2); err != nil || n != 2 || string(buf) != "cd" {
		t.Fatalf("expected ReadAt to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "cd", n, buf, err)
	}

}

// TestMaterializeReadSeekerKeepsOriginalReaderOnCloseError verifies that a
// close error leaves the original reader available to the caller.
func TestMaterializeReadSeekerKeepsOriginalReaderOnCloseError(t *testing.T) {

	closeErr := errors.New("close failed")
	originalReader := &errorCloseReader{
		Reader: strings.NewReader("abcdef"),
		err:    closeErr,
	}
	original := io.ReadCloser(originalReader)

	rc, rs, err := materializeReadSeeker(&original)
	if err == nil {
		if rc != nil {
			_ = rc.Close()
		}
		t.Fatal("expected materializeReadSeeker to fail")
	}
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected close error, got %v", err)
	}
	if rs != nil {
		t.Fatal("expected no read seeker after close error")
	}
	if original != originalReader {
		t.Fatalf("expected original reader %p, got %p", originalReader, original)
	}

}

// TestMaterializeReadSeekerReusesReadSeekAt verifies that a reader that already
// supports random access is reused.
func TestMaterializeReadSeekerReusesReadSeekAt(t *testing.T) {

	originalReader := &readSeekAtCloser{Reader: strings.NewReader("abcdef")}
	original := io.ReadCloser(originalReader)

	rc, rs, err := materializeReadSeeker(&original)
	if err != nil {
		t.Fatalf("materializeReadSeeker failed: %v", err)
	}
	if _, ok := original.(closedReadCloser); !ok {
		t.Fatalf("expected original reader type closedReadCloser, got %T", original)
	}

	buf := make([]byte, 2)
	if n, err := rs.Read(buf); err != nil || n != 2 || string(buf) != "ab" {
		t.Fatalf("expected Read to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "ab", n, buf, err)
	}
	if _, err := rs.Seek(4, io.SeekStart); err != nil {
		t.Fatalf("expected Seek to succeed, got %v", err)
	}
	if n, err := rs.Read(buf); err != nil || n != 2 || string(buf) != "ef" {
		t.Fatalf("expected Read after Seek to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "ef", n, buf, err)
	}
	if n, err := any(rs).(io.ReaderAt).ReadAt(buf, 2); err != nil || n != 2 || string(buf) != "cd" {
		t.Fatalf("expected ReadAt to return n=2 buf=%q err=<nil>, got n=%d buf=%q err=%v", "cd", n, buf, err)
	}
	if originalReader.closed {
		t.Fatal("expected reused reader to stay open until returned closer is closed")
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if !originalReader.closed {
		t.Fatal("expected returned closer to close reused reader")
	}

}

// errorCloseReader is a reader whose Close method fails.
type errorCloseReader struct {
	io.Reader
	err error
}

// Close returns the configured error.
func (r *errorCloseReader) Close() error {
	return r.err
}

// readSeekAtCloser is a random-access reader with observable close state.
type readSeekAtCloser struct {
	*strings.Reader
	closed bool
}

// Close marks r as closed.
func (r *readSeekAtCloser) Close() error {
	r.closed = true
	return nil
}

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
