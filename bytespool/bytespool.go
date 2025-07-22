//
// SPDX-License-Identifier: Elastic-2.0
//
//
// Copyright (c) 2025 Open2b
//

// Package bytespool provides a pool for reusing byte slices.
package bytespool

import (
	pool "github.com/libp2p/go-buffer-pool"
)

// Get returns a byte slice with len == 0 and cap <= capacity. Its contents are
// unspecified and may retain data from previous use.
func Get(capacity int) []byte {
	if capacity == 0 {
		return []byte{}
	}
	return pool.Get(capacity)[:0]
}

// Put returns b to the pool after clearing its contents up to len(b).
//
// To clear the entire buffer, use Put(b[:cap(b)]); to skip clearing,
// use Put(b[:0]).
func Put(b []byte) {
	clear(b)
	pool.Put(b)
}
