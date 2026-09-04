// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"errors"
	"math"
	"math/bits"
	"testing"
)

// TestPersonRangeExtendedArithmetic verifies overflow classification against carry-aware unsigned addition.
func TestPersonRangeExtendedArithmetic(t *testing.T) {

	start := uint64(0x243f6a8885a308d3)
	count := uint64(0x13198a2e03707344)
	for i := 0; i < 100_000; i++ {

		start = start*6364136223846793005 + 1442695040888963407
		count = count*2862933555777941757 + 3037000493
		if start == 0 {
			start = 1
		}

		r, err := NewPersonRange(PersonIndex(start), count)
		if count == 0 {
			if err != nil {
				t.Fatalf("expected nil error for empty range starting at %d, got %v", start, err)
			}
			if !r.Empty() {
				t.Fatalf("expected empty range for start %d, got %#v", start, r)
			}
			continue
		}
		last, carry := bits.Add64(start, count-1, 0)
		if carry != 0 {
			if err != nil {
				if !errors.Is(err, ErrRangeOverflow) {
					t.Fatalf("expected overflow for start/count %d/%d, got %v", start, count, err)
				}
				continue
			}
			t.Fatalf("expected overflow for start/count %d/%d, got nil", start, count)
		}
		if err != nil {
			t.Fatalf("expected valid range for start/count %d/%d, got %v", start, count, err)
		}
		got, ok := r.Last()
		if !ok || uint64(got) != last {
			t.Fatalf("expected last %d/true for start/count %d/%d, got %d/%v", last, start, count, got, ok)
		}

	}

}

// TestPersonRangeGolden verifies every authoritative boundary and zero-value behavior.
func TestPersonRangeGolden(t *testing.T) {

	tests := []struct {
		name      string
		start     PersonIndex
		count     uint64
		wantStart PersonIndex
		wantLast  PersonIndex
		wantOK    bool
		wantErr   error
	}{
		{"zero empty", 0, 0, 0, 0, false, ErrInvalidIndex},
		{"zero nonempty", 0, 1, 0, 0, false, ErrInvalidIndex},
		{"empty", 1, 0, 1, 0, false, nil},
		{"single", 1, 1, 1, 1, true, nil},
		{"full", 1, math.MaxUint64, 1, math.MaxUint64, true, nil},
		{"upper three", math.MaxUint64 - 2, 3, math.MaxUint64 - 2, math.MaxUint64, true, nil},
		{"upper single", math.MaxUint64, 1, math.MaxUint64, math.MaxUint64, true, nil},
		{"overflow", math.MaxUint64, 2, 0, 0, false, ErrRangeOverflow},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {

			r, err := NewPersonRange(test.start, test.count)
			if err != nil {
				if !errors.Is(err, test.wantErr) {
					t.Fatalf("expected error %v, got %v", test.wantErr, err)
				}
				return
			}
			if test.wantErr != nil {
				t.Fatalf("expected error %v, got nil", test.wantErr)
			}
			if r.Start() != test.wantStart || r.Count() != test.count || r.Empty() != (test.count == 0) {
				t.Fatalf("expected start/count/empty %d/%d/%v, got %d/%d/%v",
					test.wantStart, test.count, test.count == 0, r.Start(), r.Count(), r.Empty())
			}
			last, ok := r.Last()
			if last != test.wantLast || ok != test.wantOK {
				t.Fatalf("expected last %d/%v, got %d/%v", test.wantLast, test.wantOK, last, ok)
			}

		})

	}

	var zero PersonRange
	if zero.Start() != 1 || zero.Count() != 0 || !zero.Empty() {
		t.Fatalf("expected zero range start/count/empty 1/0/true, got %d/%d/%v", zero.Start(), zero.Count(), zero.Empty())
	}
	if last, ok := zero.Last(); last != 0 || ok {
		t.Fatalf("expected zero range last 0/false, got %d/%v", last, ok)
	}

}
