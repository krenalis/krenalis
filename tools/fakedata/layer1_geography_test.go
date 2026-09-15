// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
	"testing"
)

// TestLayer1AddressSemantics verifies municipality/CAP linkage, exact civic
// and suffix rules, canonical formatting, and mechanical option reachability.
func TestLayer1AddressSemantics(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	streetFragments := make(map[string]struct{}, len(world.data.streetNames))
	for _, row := range world.data.streetNames {
		streetFragments[row.display] = struct{}{}
	}
	wantTypes := map[string]struct{}{"Via": {}, "Viale": {}, "Piazza": {}, "Corso": {}, "Largo": {}, "Vicolo": {}}

	seenTypes := map[string]struct{}{}
	seenFragmentWeights := map[uint64]struct{}{}
	seenSuffixes := map[string]struct{}{}
	seenBranches := map[bool]struct{}{}
	for index := PersonIndex(1); index <= 10_000; index++ {

		address, err := world.address(index)
		if err != nil {
			t.Fatalf("expected address for %d, got %v", index, err)
		}
		if address.Country != "IT" {
			t.Fatalf("expected country IT for %d, got %q", index, address.Country)
		}

		mixture, err := world.personStream(index, streamGeographyPlaceMixture)
		if err != nil {
			t.Fatalf("expected mixture stream for %d, got %v", index, err)
		}
		populationWeighted, err := mixture.BernoulliRatio(95, 100)
		if err != nil {
			t.Fatalf("expected mixture draw for %d, got %v", index, err)
		}
		seenBranches[populationWeighted] = struct{}{}
		place, err := world.municipality(index, populationWeighted)
		if err != nil {
			t.Fatalf("expected municipality for %d, got %v", index, err)
		}
		if address.Municipality != place.municipality || address.StateProv != place.stateProv {
			t.Fatalf("expected municipality/state %s/%s, got %s/%s", place.municipality, place.stateProv, address.Municipality, address.StateProv)
		}
		if !slices.Contains(world.data.postalCodes[place.istatCode], address.PostalCode) {
			t.Fatalf("expected CAP %s for municipality %s, got none", address.PostalCode, place.municipality)
		}

		streetType, err := world.weightedString(index, streamGeographyAddressType, streetTypes)
		if err != nil {
			t.Fatalf("expected street type for %d, got %v", index, err)
		}
		seenTypes[streetType] = struct{}{}
		if _, exists := wantTypes[streetType]; !exists {
			t.Fatalf("expected one of six street types, got %q", streetType)
		}
		fragmentRNG, err := world.personStream(index, streamGeographyAddressFragment, world.data.streetNameDependency)
		if err != nil {
			t.Fatalf("expected fragment stream for %d, got %v", index, err)
		}
		fragmentPosition, err := fragmentRNG.WeightedIndex(world.data.streetNameWeights)
		if err != nil {
			t.Fatalf("expected fragment for %d, got %v", index, err)
		}
		fragment := world.data.streetNames[fragmentPosition].display
		seenFragmentWeights[world.data.streetNames[fragmentPosition].weight] = struct{}{}
		if _, exists := streetFragments[fragment]; !exists {
			t.Fatalf("expected frozen street fragment, got %q", fragment)
		}

		civicA := addressCivicDraw(t, world, index, streamGeographyAddressCivicA)
		civicB := addressCivicDraw(t, world, index, streamGeographyAddressCivicB)
		civic := 1 + min(civicA, civicB)
		if civic < 1 || civic > 250 {
			t.Fatalf("expected civic number in 1..250, got %d", civic)
		}

		suffixPresence, err := world.personStream(index, streamGeographyAddressSuffixPresence)
		if err != nil {
			t.Fatalf("expected suffix presence stream for %d, got %v", index, err)
		}
		hasSuffix, err := suffixPresence.BernoulliRatio(7, 100)
		if err != nil {
			t.Fatalf("expected suffix presence for %d, got %v", index, err)
		}
		suffix := ""
		if hasSuffix {
			suffixRNG, err := world.personStream(index, streamGeographyAddressSuffixChoice)
			if err != nil {
				t.Fatalf("expected suffix choice stream for %d, got %v", index, err)
			}
			position, err := suffixRNG.Uint64n(uint64(len(streetSuffixes)))
			if err != nil {
				t.Fatalf("expected suffix choice for %d, got %v", index, err)
			}
			suffix = streetSuffixes[position]
		}
		seenSuffixes[suffix] = struct{}{}

		wantStreet := fmt.Sprintf("%s %s %d", streetType, fragment, civic)
		if suffix == "bis" {
			wantStreet += " bis"
		} else {
			wantStreet += suffix
		}
		if address.Street != wantStreet {
			t.Fatalf("expected canonical street %q, got %q", wantStreet, address.Street)
		}

	}

	if len(seenTypes) != len(wantTypes) {
		t.Fatalf("expected all six street types reachable, got %v", seenTypes)
	}
	if len(seenFragmentWeights) != 3 {
		t.Fatalf("expected all street-fragment weight tiers reachable, got %v", seenFragmentWeights)
	}
	if len(seenBranches) != 2 {
		t.Fatalf("expected both municipality branches reachable, got %v", seenBranches)
	}
	wantSeenSuffixes := map[string]struct{}{"": {}, "/A": {}, "/B": {}, "/C": {}, "bis": {}}
	if !sameSet(seenSuffixes, wantSeenSuffixes) {
		t.Fatalf("expected only all four suffixes and absence, got %v", seenSuffixes)
	}

}

// TestLayer1MunicipalityPostalIsolation verifies that postal data cannot feed
// back into either municipality selector.
func TestLayer1MunicipalityPostalIsolation(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	changedData := *world.data
	changedData.postalCodes = make(map[string][]string, len(world.data.postalCodes))
	for code := range world.data.postalCodes {
		changedData.postalCodes[code] = []string{"99999"}
	}
	changedWorld := *world
	changedWorld.data = &changedData

	for _, populationWeighted := range []bool{false, true} {
		for index := PersonIndex(1); index <= 1_000; index++ {
			before, err := world.municipality(index, populationWeighted)
			if err != nil {
				t.Fatalf("expected baseline municipality for %d/%t, got %v", index, populationWeighted, err)
			}
			after, err := changedWorld.municipality(index, populationWeighted)
			if err != nil {
				t.Fatalf("expected changed-postal municipality for %d/%t, got %v", index, populationWeighted, err)
			}
			if before != after {
				t.Fatalf("expected postal-independent municipality %#v, got %#v", before, after)
			}
		}
	}

	before, err := world.address(42)
	if err != nil {
		t.Fatalf("expected baseline address, got %v", err)
	}
	after, err := changedWorld.address(42)
	if err != nil {
		t.Fatalf("expected changed-postal address, got %v", err)
	}
	if before.Municipality != after.Municipality || before.StateProv != after.StateProv {
		t.Fatalf("expected postal-independent municipality/state %#v, got %#v", before, after)
	}
	if after.PostalCode != "99999" {
		t.Fatalf("expected isolated changed CAP 99999, got %q", after.PostalCode)
	}

}

func addressCivicDraw(t *testing.T, world *World, index PersonIndex, path string) uint64 {
	t.Helper()
	rng, err := world.personStream(index, path)
	if err != nil {
		t.Fatalf("expected civic stream %s for %d, got %v", path, index, err)
	}
	value, err := rng.Uint64n(250)
	if err != nil {
		t.Fatalf("expected civic draw %s for %d, got %v", path, index, err)
	}
	return value
}

func sameSet(got, want map[string]struct{}) bool {
	if len(got) != len(want) {
		return false
	}
	for value := range got {
		if _, exists := want[value]; !exists {
			return false
		}
	}
	return true
}
