// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package bufferpool

import (
	"reflect"
	"sync"
	"testing"
)

// TestGetZero verifies that Get(0) returns an empty non-nil slice.
func TestGetZero(t *testing.T) {
	b := Get(0)
	if b == nil {
		t.Errorf("expected non-nil slice, got nil")
	}
	if len(b) != 0 {
		t.Errorf("expected len 0, got %d", len(b))
	}
}

// TestGetNegativePanics checks that Get with negative input panics.
func TestGetNegativePanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic on Get(-1), got none")
		}
	}()
	_ = Get(-1)
}

// TestGetSmall checks behavior for sizes below the pooling threshold.
func TestGetSmall(t *testing.T) {
	for _, size := range []int{1, 512, 1023} {
		b := Get(size)
		if b == nil {
			t.Errorf("expected non-nil slice for size %d, got nil", size)
			continue
		}
		if len(b) != 0 {
			t.Errorf("expected len 0 for size %d, got %d", size, len(b))
		}
		if cap(b) != size {
			t.Errorf("expected cap %d, got %d", size, cap(b))
		}
	}
}

// TestGetMinPoolSize checks allocation at the exact pooling threshold.
func TestGetMinPoolSize(t *testing.T) {
	b := Get(minPoolSize)
	if len(b) != 0 {
		t.Errorf("expected len 0, got %d", len(b))
	}
	if cap(b) != minPoolSize {
		t.Errorf("expected cap %d, got %d", minPoolSize, cap(b))
	}
}

// TestGetAboveThreshold checks allocation and rounding above threshold.
func TestGetAboveThreshold(t *testing.T) {
	size := 1500
	b := Get(size)
	want := 1 << nextLog2(uint32(size))
	if cap(b) != want {
		t.Errorf("expected cap %d, got %d", want, cap(b))
	}
	if len(b) != 0 {
		t.Errorf("expected len 0, got %d", len(b))
	}
}

// TestGetPowerOfTwoCap checks behavior on power-of-two sizes.
func TestGetPowerOfTwoCap(t *testing.T) {
	for _, size := range []int{1024, 2048, 4096, 1 << 20} {
		b := Get(size)
		if cap(b) != size {
			t.Errorf("expected cap %d, got %d", size, cap(b))
		}
	}
}

// TestPutNil ensures that Put(nil) does not panic.
func TestPutNil(t *testing.T) {
	Put(nil) // should not panic
}

// TestPutTooSmall ensures small buffers are not pooled or reused.
func TestPutTooSmall(t *testing.T) {
	buf := make([]byte, minPoolSize-1)
	Put(buf)
	b := Get(minPoolSize - 1)
	if cap(b) != minPoolSize-1 {
		t.Errorf("expected cap %d, got %d", minPoolSize-1, cap(b))
	}
}

// TestPutAndGetReuse checks buffer reuse for the same bucket.
func TestPutAndGetReuse(t *testing.T) {
	size := 2048
	b1 := Get(size)
	ptr1 := reflect.ValueOf(b1).Pointer()
	Put(b1)
	b2 := Get(size)
	ptr2 := reflect.ValueOf(b2).Pointer()
	if ptr1 != ptr2 {
		t.Errorf("expected reused buffer (pointer %v), got new buffer (pointer %v)", ptr1, ptr2)
	}
}

// TestFallbackOne checks fallback to next bucket if exact is empty.
func TestFallbackOne(t *testing.T) {
	size := 1500
	nextCap := 1 << (nextLog2(uint32(size)) + 1)
	Put(make([]byte, nextCap))
	b := Get(size)
	if cap(b) != nextCap {
		t.Errorf("expected cap %d, got %d", nextCap, cap(b))
	}
}

// TestFallbackTwo checks fallback to two buckets up if needed.
func TestFallbackTwo(t *testing.T) {
	size := 1500
	twoUpCap := 1 << (nextLog2(uint32(size)) + 2)
	Put(make([]byte, twoUpCap))
	b := Get(size)
	if cap(b) != twoUpCap {
		t.Errorf("expected cap %d, got %d", twoUpCap, cap(b))
	}
}

// TestNoFallbackBeyondTwo ensures fallback does not go beyond two buckets.
func TestNoFallbackBeyondTwo(t *testing.T) {
	size := 1500
	tooBigCap := 1 << (nextLog2(uint32(size)) + 3)
	Put(make([]byte, tooBigCap))
	b := Get(size)
	want := 1 << nextLog2(uint32(size))
	if cap(b) != want {
		t.Errorf("expected cap %d, got %d", want, cap(b))
	}
}

// TestPutLarge checks that large pooled buffers are reused correctly.
func TestPutLarge(t *testing.T) {
	size := minPoolSize * 4
	buf := Get(size)
	buf = buf[:size]
	Put(buf)
	b := Get(size)
	if cap(b) < size {
		t.Errorf("expected cap >= %d, got %d", size, cap(b))
	}
}

// TestGetAboveMaxLength checks that requests beyond the managed range still
// return an empty slice whose capacity matches the requested size.
func TestGetAboveMaxLength(t *testing.T) {
	length := maxLength + 1
	b := Get(length)
	if len(b) != 0 {
		t.Errorf("expected len 0, got %d", len(b))
	}
	if cap(b) != length {
		t.Errorf("expected cap %d, got %d", length, cap(b))
	}
}

// TestClearReturnedBuffer ensures returned buffers are zeroed out.
func TestClearReturnedBuffer(t *testing.T) {
	size := 1500
	fullCap := 1 << nextLog2(uint32(size))
	buf := make([]byte, fullCap)
	for i := range buf {
		buf[i] = 0xFF
	}
	Put(buf)
	b := Get(size)
	for i, v := range b[:cap(b)] {
		if v != 0 {
			t.Errorf("expected zeroed buffer, got non-zero at index %d (value %d)", i, v)
			break
		}
	}
}

// TestConcurrentUsage checks that Get and Put are safe under concurrency.
func TestConcurrentUsage(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			size := minPoolSize + (i % 4096)
			b := Get(size)
			Put(b)
		}(i)
	}
	wg.Wait()
}

func TestPutAboveMaxLengthIgnored(t *testing.T) {
	buf := make([]byte, maxLength+1)
	Put(buf)
	b := Get(maxLength + 1)
	if cap(b) != maxLength+1 {
		t.Errorf("expected cap %d, got %d", maxLength+1, cap(b))
	}
}
