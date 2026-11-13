// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

// TestUSANamesV2IndexEquivalence checks boundaries, state and rejection draws.
func TestUSANamesV2IndexEquivalence(t *testing.T) {

	tables := [][]uint64{
		{1}, {1, 1, 1}, {1, 999999999, 1}, {1, usaNamesMass - 2, 1}, {math.MaxUint64 - 2, 1, 1}, {1, 1 << 63},
	}
	for _, size := range []int{5000, 50000} {
		weights := make([]uint64, size)
		for i := range weights {
			weights[i] = uint64(i%31 + 1)
		}
		tables = append(tables, weights)
	}
	for table, weights := range tables {
		t.Run(strconv.Itoa(table), func(t *testing.T) {

			index := &usaNominalIndex{entries: make([]usaNameRow, len(weights)), cumulative: make([]uint64, len(weights))}
			for i, weight := range weights {
				index.total += weight
				index.cumulative[i] = index.total
				index.entries[i] = usaNameRow{display: strconv.Itoa(i), weight: weight}
			}
			check := func(seed uint64) {

				t.Helper()
				a, b := &splitMix64{state: seed}, &splitMix64{state: seed}
				position, err := a.WeightedIndex(weights)
				if err != nil {
					t.Fatalf("expected reference selection, got %v", err)
				}
				row, err := index.selectName(b)
				if err != nil {
					t.Fatalf("expected indexed selection, got %v", err)
				}
				if row != index.entries[position] || a.state != b.state || a.Uint64() != b.Uint64() {
					t.Fatalf("expected equal index and exact final RNG state at seed %x, got %+v/%x/%x", seed, row, a.state, b.state)
				}

			}
			for _, seed := range []uint64{0, 1, 2, 42, 2026, 1 << 63, math.MaxUint64} {
				for i := uint64(0); i < 128; i++ {
					check(seed + i*0x9e3779b97f4a7c15)
				}
			}
			threshold := (uint64(0) - index.total) % index.total
			for i, end := range index.cumulative {
				if len(weights) > 10 && i%997 != 0 && i != len(weights)-1 {
					continue
				}
				start := end - weights[i]
				for _, draw := range []uint64{start, end - 1, end % index.total} {
					output := draw
					if output < threshold {
						output += index.total
					}
					check(usaNamesV2SeedForOutput(output))
				}
			}
			if threshold > 0 {
				seed := usaNamesV2SeedForOutput(0)
				probe := &splitMix64{state: seed}
				_, err := probe.Uint64n(index.total)
				if err != nil {
					t.Fatalf("expected rejection reference, got %v", err)
				}
				if probe.state == seed+0x9e3779b97f4a7c15 {
					t.Fatal("expected exercised rejection draw, got one draw")
				}
				check(seed)
			}

		})
	}

}

// TestUSANamesV2IndexInvalid rejects invalid entries and nil samplers.
func TestUSANamesV2IndexInvalid(t *testing.T) {

	atom := "same-atom"
	entries := []internationalName{{"B", &atom, 1}, {"É", &atom, 2}, {"Ê", &atom, 3}}
	_, err := newUSANominalIndex(entries, 3, 6)
	if err != nil {
		t.Fatalf("expected Unicode16 entries and atom collisions admitted, got %v", err)
	}
	for _, display := range []string{"E\u0301", " é", "A  B", "\u0301A", "Ł", "😀", "", strings.Repeat("A", 81)} {
		_, err := newUSANominalIndex([]internationalName{{display, &atom, 1}}, 1, 1)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid display rejected, got %q", display)
	}
	for _, atom := range []string{"", "A", "-a", "a-", "a--b", "é", strings.Repeat("a", 81)} {
		_, err := newUSANominalIndex([]internationalName{{"A", &atom, 1}}, 1, 1)
		if err != nil {
			continue
		}
		t.Fatalf("expected invalid atom rejected, got %q", atom)
	}
	for _, entries := range [][]internationalName{
		{{"A", nil, 1}}, {{"A", &atom, 0}}, {{"B", &atom, 1}, {"A", &atom, 1}},
		{{"É", &atom, 1}, {"é", &atom, 1}}, {{"A", &atom, math.MaxUint64}, {"B", &atom, 1}},
	} {
		_, err := newUSANominalIndex(entries, len(entries), 2)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid entries rejected, got nil")
	}
	for _, count := range []int{-1, 0, 50001, 2} {
		_, err := newUSANominalIndex(entries, count, 6)
		if err != nil {
			continue
		}
		t.Fatal("expected invalid cardinality rejected, got nil")
	}
	t.Run("total", func(t *testing.T) {
		_, err := newUSANominalIndex(entries, 3, 7)
		if err != nil {
			return
		}
		t.Fatal("expected total mismatch rejected, got nil")
	})
	for _, index := range []*usaNominalIndex{nil, {}, {entries: []usaNameRow{{}}, cumulative: []uint64{0}}} {
		_, err := index.selectName(&splitMix64{})
		if err != nil {
			continue
		}
		t.Fatal("expected uninitialized index rejected, got nil")
	}
	for name, sample := range map[string]func() error{
		"surname-nil": func() error { _, err := (*usaSurname50KCandidate)(nil).selectSurname(&splitMix64{}); return err },
		"first-nil": func() error {
			_, err := (*usaFirstNamesV2Candidate)(nil).selectFirstName(&splitMix64{}, GenderFemale, "1950-1964")
			return err
		},
		"first-rng": func() error {
			_, err := (&usaFirstNamesV2Candidate{}).selectFirstName(nil, GenderFemale, "1950-1964")
			return err
		},
		"first-empty": func() error {
			_, err := (&usaFirstNamesV2Candidate{}).selectFirstName(&splitMix64{}, GenderFemale, "1950-1964")
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := sample()
			if err != nil {
				return
			}
			t.Fatal("expected invalid sampler rejected, got nil")
		})
	}

}

// usaNamesV2SeedForOutput inverts the frozen SplitMix64 permutation in tests.
func usaNamesV2SeedForOutput(output uint64) uint64 {
	output ^= output>>31 ^ output>>62
	output *= 0x319642b2d24d8ec3
	output ^= output>>27 ^ output>>54
	output *= 0x96de1b173f119089
	output ^= output>>30 ^ output>>60
	return output - 0x9e3779b97f4a7c15
}
