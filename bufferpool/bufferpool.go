//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package bufferpool provides a pool for reusing buffers.
package bufferpool

import (
	"math"
	"math/bits"
	"sync"
)

const (
	// maxLength is the largest buffer size managed by the pool.
	maxLength = math.MaxInt32

	// minPoolSize is the minimum capacity of a buffer to be pooled.
	// Buffers smaller than this are always allocated directly.
	minPoolSize = 1 << 10 // 1024

	// minBucket and maxBucket define the inclusive power-of-two range
	// of capacities that are pooled. Buckets go from 2^10 to 2^31.
	minBucket    = 10
	maxBucket    = 31
	totalBuckets = maxBucket - minBucket + 1
)

// globalPool is the internal shared buffer pool instance.
// All calls to Get and Put use this pool.
var globalPool = newBufferPool()

// bufferPool holds a fixed array of sync.Pool buckets and a separate pool
// for buffer wrappers. Each buffer is grouped by capacity using power-of-two
// buckets between minBucket and maxBucket.
type bufferPool struct {
	pools [totalBuckets]sync.Pool
	ptrs  sync.Pool
}

// buffer is a wrapper used to store byte slices inside the pool.
// It avoids escaping allocations by boxing the slice.
type buffer struct {
	data []byte
}

// newBufferPool creates and returns a new bufferPool instance.
// Only one instance is used globally; this is not exported.
func newBufferPool() *bufferPool {
	p := &bufferPool{}
	p.ptrs = sync.Pool{
		New: func() interface{} { return &buffer{} },
	}
	return p
}

// get returns a non-nil zero-length []byte slice with capacity >= size. It
// reuses memory from the pool when possible, using size-based buckets. The
// returned buffer is always zeroed.
//
// It falls back to allocation when no reusable buffer is found in the expected
// bucket or the next two larger buckets.
func (p *bufferPool) get(size int) []byte {

	switch {
	case size == 0:
		return []byte{}
	case size < 0:
		// Let make panic as expected for negative lengths.
		return make([]byte, size)
	case size > maxLength:
		buf := make([]byte, size)
		clear(buf)
		return buf
	case size < minPoolSize:
		buf := make([]byte, size)
		clear(buf)
		return buf[:0]
	}

	idx := int(nextLog2(uint32(size)))
	bucket := idx - minBucket

	// Try up to 3 buckets: exact, +1, +2
	for i := bucket; i <= bucket+2 && i < totalBuckets; i++ {
		if v := p.pools[i].Get(); v != nil {
			b := v.(*buffer)
			buf := b.data
			b.data = nil
			clear(buf)
			return buf[:0]
		}
	}

	// No buffer found, allocate new one.
	buf := make([]byte, 1<<idx)

	return buf[:0]
}

// put returns the given buffer to the pool, if it is within the managed size
// range. The buffer is boxed into a wrapper and inserted into the appropriate
// bucket. Buffers with capacity < minPoolSize or > maxLength are discarded.
func (p *bufferPool) put(buf []byte) {
	if buf == nil {
		return
	}
	c := cap(buf)
	if c < minPoolSize || c > maxLength {
		return
	}
	idx := int(prevLog2(uint32(c)))
	if idx < minBucket || idx > maxBucket {
		return
	}
	bucket := idx - minBucket
	b := p.ptrs.Get().(*buffer)
	b.data = buf
	p.pools[bucket].Put(b)
}

// Get returns a non-nil zero-length []byte with capacity ≥ size.
// Only buffers with capacity between 1 KiB and 1 GiB are pooled.
//
// If a previously used buffer is returned, elements up to its previous length
// are cleared to zero before returning. To clear the entire buffer, reset its
// length before putting it back: bufferpool.Put(b[:cap(b)]).
func Get(size int) []byte {
	return globalPool.get(size)
}

// Put returns a previously acquired buffer to the pool, if it is within range.
// Only buffers with capacity between 1 KiB and 1 GiB are pooled.
func Put(buf []byte) {
	globalPool.put(buf)
}

// nextLog2 returns the ceiling of log2(v), assuming v > 0.
func nextLog2(v uint32) uint32 {
	return uint32(bits.Len32(v - 1))
}

// prevLog2 returns the floor of log2(v), assuming v > 0.
func prevLog2(v uint32) uint32 {
	return uint32(bits.Len32(v)) - 1
}
