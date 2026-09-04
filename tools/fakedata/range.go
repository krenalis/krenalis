// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import "math"

// PersonRange is an opaque contiguous range of one-based person indices.
// Its zero value is the valid empty range starting at PersonIndex(1).
type PersonRange struct {
	start PersonIndex
	count uint64
}

// NewPersonRange validates and returns a range beginning at start with count indices.
// It returns ErrInvalidIndex or ErrRangeOverflow for an invalid range.
func NewPersonRange(start PersonIndex, count uint64) (PersonRange, error) {

	if start == 0 {
		return PersonRange{}, ErrInvalidIndex
	}
	if count > 0 && count-1 > math.MaxUint64-uint64(start) {
		return PersonRange{}, ErrRangeOverflow
	}

	return PersonRange{start: start, count: count}, nil
}

// Count returns the number of indices in the range.
func (r PersonRange) Count() uint64 {
	return r.count
}

// Empty reports whether the range contains no indices.
func (r PersonRange) Empty() bool {
	return r.count == 0
}

// Last returns the final index and reports whether the range is non-empty.
func (r PersonRange) Last() (PersonIndex, bool) {

	if r.Empty() {
		return 0, false
	}

	return r.Start() + PersonIndex(r.count-1), true
}

// Start returns the first index of the range.
func (r PersonRange) Start() PersonIndex {

	if r.start == 0 {
		return 1
	}

	return r.start
}
