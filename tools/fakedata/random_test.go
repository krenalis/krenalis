// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"errors"
	"math"
	"reflect"
	"testing"
)

// TestBernoulliRatioGolden verifies exact draws, boundaries, validation, and consumption.
func TestBernoulliRatioGolden(t *testing.T) {

	rng := splitMix64{}
	got := make([]bool, 5)
	for i := range got {
		value, err := rng.BernoulliRatio(25, 100)
		if err != nil {
			t.Fatalf("expected nil error at draw %d, got %v", i, err)
		}
		got[i] = value
	}
	if want := []bool{false, true, false, false, false}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

	const state = 0x123456789abcdef0
	boundaries := []struct {
		numerator   uint64
		denominator uint64
		want        bool
		wantErr     error
	}{
		{0, 100, false, nil},
		{100, 100, true, nil},
		{0, 0, false, ErrInvalidProbability},
		{101, 100, false, ErrInvalidProbability},
	}
	for _, test := range boundaries {
		rng = splitMix64{state: state}
		value, err := rng.BernoulliRatio(test.numerator, test.denominator)
		if err != nil {
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("ratio %d/%d: expected error %v, got %v", test.numerator, test.denominator, test.wantErr, err)
			}
		}
		if test.wantErr != nil && err == nil {
			t.Fatalf("ratio %d/%d: expected error %v, got nil", test.numerator, test.denominator, test.wantErr)
		}
		if value != test.want {
			t.Fatalf("ratio %d/%d: expected %v, got %v", test.numerator, test.denominator, test.want, value)
		}
		if rng.state != state {
			t.Fatalf("ratio %d/%d consumed RNG: expected %016x, got %016x", test.numerator, test.denominator, state, rng.state)
		}
	}

}

// TestSplitMix64Golden verifies each authoritative sequence.
func TestSplitMix64Golden(t *testing.T) {

	tests := []struct {
		state uint64
		want  []uint64
	}{
		{
			state: 0,
			want: []uint64{
				0xe220a8397b1dcdaf,
				0x6e789e6aa1b965f4,
				0x06c45d188009454f,
				0xf88bb8a8724c81ec,
				0x1b39896a51a8749b,
			},
		},
		{
			state: 1,
			want: []uint64{
				0x910a2dec89025cc1,
				0xbeeb8da1658eec67,
				0xf893a2eefb32555e,
				0x71c18690ee42c90b,
				0x71bb54d8d101b5b9,
			},
		},
		{
			state: 0x89a10f64d4a955ca,
			want: []uint64{
				0xd9926b84dbfd233e,
				0x08c61e2295738cc8,
				0x7d26fff6c15c1920,
				0x1d1fb5fd4c085e59,
				0x2d65f0dae6dcc95a,
			},
		},
	}

	for _, test := range tests {
		rng := splitMix64{state: test.state}
		for i, want := range test.want {
			if got := rng.Uint64(); got != want {
				t.Fatalf("state %016x output %d: expected %016x, got %016x", test.state, i, want, got)
			}
		}
	}

}

// TestUint64nErrorsDoNotConsume verifies zero-bound and unit-bound consumption semantics.
func TestUint64nErrorsDoNotConsume(t *testing.T) {

	const state = 0x123456789abcdef0
	rng := splitMix64{state: state}
	_, err := rng.Uint64n(0)
	if err != nil {
		if !errors.Is(err, ErrZeroBound) {
			t.Fatalf("expected zero bound, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected zero bound, got nil")
	}
	if rng.state != state {
		t.Fatalf("expected unchanged state %016x, got %016x", state, rng.state)
	}
	value, err := rng.Uint64n(1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if value != 0 {
		t.Fatalf("expected zero, got %d", value)
	}
	if rng.state != state {
		t.Fatalf("expected unchanged state %016x, got %016x", state, rng.state)
	}

	for n := uint64(2); n < 1_000; n++ {
		value, err = rng.Uint64n(n)
		if err != nil {
			t.Fatalf("expected nil error for %d, got %v", n, err)
		}
		if value >= n {
			t.Fatalf("expected value below %d, got %d", n, value)
		}
	}

}

// TestUint64nGolden verifies boundaries, sequences, and the normative rejection algorithm.
func TestUint64nGolden(t *testing.T) {

	tests := []struct {
		n          uint64
		want       uint64
		finalState uint64
	}{
		{1, 0, 0x0000000000000000},
		{2, 1, 0x9e3779b97f4a7c15},
		{10, 5, 0x9e3779b97f4a7c15},
		{uint64(1) << 63, 0x6220a8397b1dcdaf, 0x9e3779b97f4a7c15},
		{math.MaxUint64, 0xe220a8397b1dcdaf, 0x9e3779b97f4a7c15},
	}
	for _, test := range tests {
		rng := splitMix64{}
		got, err := rng.Uint64n(test.n)
		if err != nil {
			t.Fatalf("expected nil error for n=%d, got %v", test.n, err)
		}
		if got != test.want || rng.state != test.finalState {
			t.Fatalf("n=%d: expected %d/state %016x, got %d/state %016x", test.n, test.want, test.finalState, got, rng.state)
		}
	}

	rng := splitMix64{}
	sequence := make([]uint64, 10)
	for i := range sequence {
		value, err := rng.Uint64n(10)
		if err != nil {
			t.Fatalf("expected nil error at draw %d, got %v", i, err)
		}
		sequence[i] = value
	}
	if want := []uint64{5, 0, 9, 4, 7, 0, 3, 0, 9, 0}; !reflect.DeepEqual(sequence, want) {
		t.Fatalf("expected %v, got %v", want, sequence)
	}

	candidates := splitMix64{state: 3}
	if got := candidates.Uint64(); got != 0x1d0b14e4db018fed {
		t.Fatalf("expected rejected candidate 1d0b14e4db018fed, got %016x", got)
	}
	if got := candidates.Uint64(); got != 0xb3466f8a7b81a989 {
		t.Fatalf("expected accepted candidate b3466f8a7b81a989, got %016x", got)
	}

	const rejectionBound = uint64(9223372036854775809)
	n := rejectionBound
	if threshold := (uint64(0) - n) % n; threshold != 9223372036854775807 {
		t.Fatalf("expected threshold 9223372036854775807, got %d", threshold)
	}
	rejection := splitMix64{state: 3}
	value, err := rejection.Uint64n(rejectionBound)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if value != 3694763184872335752 || rejection.state != 0x3c6ef372fe94f82d {
		t.Fatalf("expected rejection result 3694763184872335752/state 3c6ef372fe94f82d, got %d/state %016x",
			value, rejection.state)
	}

}

// TestWeightedIndexErrorsDoNotConsume verifies complete validation before random consumption.
func TestWeightedIndexErrorsDoNotConsume(t *testing.T) {

	const state = 0x123456789abcdef0
	tests := []struct {
		name    string
		weights []uint64
		want    error
	}{
		{"nil", nil, ErrEmptyWeightedTable},
		{"empty", []uint64{}, ErrEmptyWeightedTable},
		{"all zero", []uint64{0, 0, 0}, ErrAllZeroWeights},
		{"overflow", []uint64{math.MaxUint64, 1}, ErrTotalWeightOverflow},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			rng := splitMix64{state: state}
			_, err := rng.WeightedIndex(test.weights)
			if err != nil {
				if !errors.Is(err, test.want) {
					t.Fatalf("expected %v, got %v", test.want, err)
				}
			}
			if err == nil {
				t.Fatalf("expected %v, got nil", test.want)
			}
			if rng.state != state {
				t.Fatalf("expected unchanged state %016x, got %016x", state, rng.state)
			}

		})

	}

	rng := splitMix64{state: state}
	index, err := rng.WeightedIndex([]uint64{0, 1, 0})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if index != 1 {
		t.Fatalf("expected index 1, got %d", index)
	}
	if rng.state != state {
		t.Fatalf("expected total-one table not to consume RNG: %016x != %016x", rng.state, state)
	}

}

// TestWeightedIndexGolden verifies the authoritative ordered weighted sequence.
func TestWeightedIndexGolden(t *testing.T) {

	rng := splitMix64{}
	got := make([]int, 10)
	for i := range got {
		index, err := rng.WeightedIndex([]uint64{0, 3, 1, 6})
		if err != nil {
			t.Fatalf("expected nil error at draw %d, got %v", i, err)
		}
		got[i] = index
	}
	if want := []int{3, 1, 3, 3, 3, 1, 2, 1, 3, 1}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}

}
