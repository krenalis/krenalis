// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"math"
	"sort"
)

// usaNominalIndex owns admitted scalar entries and immutable cumulative ends.
type usaNominalIndex struct {
	entries    []usaNameRow
	cumulative []uint64
	total      uint64
}

// newUSANominalIndex validates and copies a bounded NameEntry collection.
func newUSANominalIndex(entries []internationalName, count int, expectedTotal uint64) (*usaNominalIndex, error) {

	if count <= 0 || count > 50000 || len(entries) != count || expectedTotal == 0 {
		return nil, fmt.Errorf("invalid USA nominal cardinality or total")
	}
	prepared := &usaNominalIndex{
		entries: make([]usaNameRow, count), cumulative: make([]uint64, count),
	}
	previous := ""
	for i, entry := range entries {

		if len(entry.Display) > 80 {
			return nil, fmt.Errorf("USA nominal display exceeds byte limit at position %d", i)
		}
		key, err := italianDisplayKey(entry.Display)
		if err != nil {
			return nil, fmt.Errorf("USA nominal display at position %d: %w", i, err)
		}
		if key <= previous || entry.ASCIIAtom == nil || len(*entry.ASCIIAtom) > 80 ||
			!internationalASCIIAtom(*entry.ASCIIAtom) || entry.Weight == 0 || entry.Weight > math.MaxUint64-prepared.total {
			return nil, fmt.Errorf("invalid USA nominal key, atom or weight at position %d", i)
		}

		previous = key
		prepared.total += entry.Weight
		prepared.entries[i] = usaNameRow{display: entry.Display, asciiAtom: *entry.ASCIIAtom, weight: entry.Weight}
		prepared.cumulative[i] = prepared.total

	}
	if prepared.total != expectedTotal {
		return nil, fmt.Errorf("USA nominal total does not reconcile")
	}

	return prepared, nil
}

// selectName uses WeightedIndex's exact bounded draw and half-open intervals.
// Ownership at admission makes the cumulative ends immutable during sampling.
func (d *usaNominalIndex) selectName(rng *splitMix64) (usaNameRow, error) {

	if d == nil || rng == nil || len(d.entries) == 0 || len(d.cumulative) != len(d.entries) ||
		d.total == 0 || d.cumulative[len(d.cumulative)-1] != d.total {
		return usaNameRow{}, fmt.Errorf("uninitialized USA nominal index or random stream")
	}
	draw, err := rng.Uint64n(d.total)
	if err != nil {
		return usaNameRow{}, err
	}
	position := sort.Search(len(d.cumulative), func(i int) bool { return draw < d.cumulative[i] })

	return d.entries[position], nil
}
