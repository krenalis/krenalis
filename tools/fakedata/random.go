// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import "math"

type splitMix64 struct {
	state uint64
}

// BernoulliRatio returns a Bernoulli draw with probability numerator/denominator.
// It returns ErrInvalidProbability for an invalid ratio.
func (r *splitMix64) BernoulliRatio(numerator, denominator uint64) (bool, error) {

	if denominator == 0 || numerator > denominator {
		return false, ErrInvalidProbability
	}
	if numerator == 0 {
		return false, nil
	}
	if numerator == denominator {
		return true, nil
	}

	x, err := r.Uint64n(denominator)
	if err != nil {
		return false, err
	}

	return x < numerator, nil
}

// Uint64 returns the next SplitMix64 output.
func (r *splitMix64) Uint64() uint64 {

	r.state += 0x9e3779b97f4a7c15

	z := r.state
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	z ^= z >> 31

	return z
}

// Uint64n returns an unbiased draw in [0, n).
// It returns ErrZeroBound when n is zero.
func (r *splitMix64) Uint64n(n uint64) (uint64, error) {

	if n == 0 {
		return 0, ErrZeroBound
	}
	if n == 1 {
		return 0, nil
	}

	threshold := (uint64(0) - n) % n
	for {
		x := r.Uint64()
		if x >= threshold {
			return x % n, nil
		}
	}

}

// WeightedIndex returns the first weighted interval containing an unbiased draw.
// It returns ErrEmptyWeightedTable, ErrAllZeroWeights, or ErrTotalWeightOverflow as appropriate.
func (r *splitMix64) WeightedIndex(weights []uint64) (int, error) {

	if len(weights) == 0 {
		return 0, ErrEmptyWeightedTable
	}

	var total uint64
	for _, weight := range weights {
		if weight > math.MaxUint64-total {
			return 0, ErrTotalWeightOverflow
		}
		total += weight
	}
	if total == 0 {
		return 0, ErrAllZeroWeights
	}

	draw, err := r.Uint64n(total)
	if err != nil {
		return 0, err
	}
	var cumulative uint64
	for i, weight := range weights {
		cumulative += weight
		if draw < cumulative {
			return i, nil
		}
	}

	panic("weighted selection has no index")

}
