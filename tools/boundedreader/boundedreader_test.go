// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package boundedreader

import (
	"errors"
	"io"
	"strings"
	"testing"
)

// Reads less than the limit.
func TestReadLessThanLimit(t *testing.T) {
	br := New(strings.NewReader("abc"), 5)

	b, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("expected %q, got %q", "abc", b)
	}
}

// Reads exactly up to the limit.
func TestReadExactlyLimit(t *testing.T) {
	br := New(strings.NewReader("abc"), 3)

	b, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("expected %q, got %q", "abc", b)
	}
}

// Reports ErrTooLarge when the limit is exceeded.
func TestReadBeyondLimit(t *testing.T) {
	br := New(strings.NewReader("abcd"), 3)

	b, err := io.ReadAll(br)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("expected %q, got %q", "abc", b)
	}
}

// Detects overflow across multiple reads.
func TestReadBeyondLimitWithSmallBuffer(t *testing.T) {
	br := New(strings.NewReader("abcdef"), 5)

	var out []byte
	buf := make([]byte, 2)

	for {
		n, err := br.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			if !errors.Is(err, ErrTooLarge) {
				t.Fatalf("expected ErrTooLarge, got %v", err)
			}
			break
		}
	}

	if string(out) != "abcde" {
		t.Fatalf("expected %q, got %q", "abcde", out)
	}
}

// Handles an empty input with a zero limit.
func TestReadZeroLimitEmptyInput(t *testing.T) {
	br := New(strings.NewReader(""), 0)

	b, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty output, got %q", b)
	}
}

// Reports ErrTooLarge when the limit is zero.
func TestReadZeroLimitNonEmptyInput(t *testing.T) {
	br := New(strings.NewReader("a"), 0)

	b, err := io.ReadAll(br)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
	if len(b) != 0 {
		t.Fatalf("expected empty output, got %q", b)
	}
}

// Does not consume data on an empty read.
func TestReadEmptyBuffer(t *testing.T) {
	br := New(strings.NewReader("abc"), 3)

	n, err := br.Read(nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if n != 0 {
		t.Fatalf("expected n == 0, got %d", n)
	}

	b, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("expected %q, got %q", "abc", b)
	}
}

// Uses the configured error function.
func TestSetErrorFunc(t *testing.T) {
	want := errors.New("custom error")

	br := New(strings.NewReader("abcd"), 3)
	br.SetErrorFunc(func() error {
		return want
	})

	_, err := io.ReadAll(br)
	if !errors.Is(err, want) {
		t.Fatalf("expected custom error, got %v", err)
	}
}

// Falls back to ErrTooLarge when the error function is nil.
func TestSetErrorFuncNil(t *testing.T) {
	br := New(strings.NewReader("abcd"), 3)
	br.SetErrorFunc(nil)

	_, err := io.ReadAll(br)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("expected ErrTooLarge, got %v", err)
	}
}

// Calls the error function only once.
func TestErrorFuncCalledOnce(t *testing.T) {
	want := errors.New("custom error")
	calls := 0

	br := New(strings.NewReader("abcd"), 3)
	br.SetErrorFunc(func() error {
		calls++
		return want
	})

	_, err := io.ReadAll(br)
	if !errors.Is(err, want) {
		t.Fatalf("expected custom error, got %v", err)
	}

	_, err = br.Read(make([]byte, 1))
	if !errors.Is(err, want) {
		t.Fatalf("expected custom error, got %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected error function to be called once, got %d", calls)
	}
}

// Propagates errors from the underlying reader.
func TestUnderlyingReaderError(t *testing.T) {
	want := errors.New("read error")
	br := New(errReader{err: want}, 3)

	n, err := br.Read(make([]byte, 1))
	if n != 0 {
		t.Fatalf("expected n == 0, got %d", n)
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected underlying error, got %v", err)
	}
}

// Panics when the limit is negative.
func TestNewNegativeLimit(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic, got nil")
		}
	}()

	New(strings.NewReader(""), -1)
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}
