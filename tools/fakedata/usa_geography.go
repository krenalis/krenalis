// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
	"slices"
)

// usaGeographyCandidate owns admitted localities and their effective masses.
// It deliberately has no public World, registry or approval entry point.
type usaGeographyCandidate struct {
	locations      []usaLocation
	weights        []uint64
	byID           map[string]int
	divisionMasses [9]uint64
	total          uint64
	identity       [sha256.Size]byte
}

func (d *usaGeographyCandidate) resolve(id string) (usaLocation, error) {

	if d == nil || len(d.locations) == 0 {
		return usaLocation{}, fmt.Errorf("uninitialized USA geography candidate")
	}
	position, ok := d.byID[id]
	if !ok {
		return usaLocation{}, fmt.Errorf("unknown USA locality")
	}
	location := d.locations[position]
	location.zips = slices.Clone(location.zips)

	return location, nil
}

// selectLocation makes one proportional locality draw; its state supplies the
// L2 Division. Future address/Person composition must retain this same value.
// The caller owns the deterministic RNG; no Italian mixture stream is used.
func (d *usaGeographyCandidate) selectLocation(rng *splitMix64) (usaLocation, error) {

	if d == nil || len(d.locations) == 0 || rng == nil {
		return usaLocation{}, fmt.Errorf("initialized USA geography and RNG are required")
	}
	position, err := rng.WeightedIndex(d.weights)
	if err != nil {
		return usaLocation{}, err
	}

	return d.resolve(d.locations[position].id)
}

// usaLocation retains postal locality identity independently of municipality
// boundaries; its weight is editorial and must never be called population.
type usaLocation struct {
	id     string
	name   string
	state  usaState
	weight uint64
	zips   []string
}
