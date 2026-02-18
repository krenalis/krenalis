// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package bufferpool

import (
	"math/rand"
	"testing"
)

// This file contains benchmarks comparing buffer allocation with reuse.
//
// The benchmarks simulate concurrent clients that prepare HTTP request bodies
// by allocating or reusing byte slices of various sizes
// (from 4 KiB to 256 KiB), and writing into a portion of them before sending.
//
// To run these benchmarks:
//
//	go test -bench=WithUse -benchmem -benchtime=10s
//
// The output includes per-operation time, memory usage, and allocation count.
// A buffer pool is significantly faster and more memory-efficient under load.

var sink []byte

// sampleSize returns a buffer size based on a fixed distribution.
func sampleSize(i int) int {
	switch i % 20 {
	case 0:
		return 256 * 1024
	case 1, 2:
		return 64 * 1024
	case 3, 4, 5:
		return 16 * 1024
	case 6, 7, 8, 9:
		return 8 * 1024
	default:
		return 4 * 1024
	}
}

// fillBuffer appends bytes to the buffer up to a percentage of its capacity.
func fillBuffer(b []byte, percent int) []byte {
	n := min(cap(b)*percent/100, cap(b))
	for i := 0; i < n; i += 64 {
		b = append(b, byte(i%256))
	}
	return b
}

// BenchmarkAllocWithUse measures the cost of allocating and filling new buffers.
func BenchmarkAllocWithUse(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		i := rand.Int()
		for pb.Next() {
			size := sampleSize(i)
			buf := make([]byte, 0, size)
			buf = fillBuffer(buf, 50+(i%51))
			sink = buf
			i++
		}
	})
	_ = sink
}

// BenchmarkPoolWithUse measures the performance of reusing buffers from the pool.
func BenchmarkPoolWithUse(b *testing.B) {
	for range 10000 {
		Put(Get(4096))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := rand.Int()
		for pb.Next() {
			size := sampleSize(i)
			buf := Get(size)
			buf = fillBuffer(buf, 50+(i%51))
			sink = buf
			Put(buf)
			i++
		}
	})
	_ = sink
}
