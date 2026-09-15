// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/hex"
	"reflect"
	"slices"
	"testing"
)

// TestLayer1DatasetDependencyIsolation verifies each data identity is confined
// to its intended generated field family.
func TestLayer1DatasetDependencyIsolation(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	baseline, err := world.Person(42)
	if err != nil {
		t.Fatalf("expected baseline Person, got %v", err)
	}

	ageWorld := layer1WorldWithDependency(t, world, ageDatasetComponentName, func(data *italyData, component Component) {
		data.ageDependency = component
	})
	agePerson, err := ageWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-age-dependency Person, got %v", err)
	}
	if agePerson.ID != baseline.ID || agePerson.Gender != baseline.Gender || agePerson.LastName != baseline.LastName ||
		agePerson.Phone != baseline.Phone || agePerson.SiteURL != baseline.SiteURL || agePerson.Address != baseline.Address {
		t.Fatalf("expected age dependency isolation, got %#v and %#v", baseline, agePerson)
	}

	firstWorld := layer1WorldWithDependency(t, world, firstNameDatasetComponentName, func(data *italyData, component Component) {
		data.firstNameDependency = component
	})
	firstPerson, err := firstWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-first-name-dependency Person, got %v", err)
	}
	firstPerson.FirstName = baseline.FirstName
	firstPerson.Email = baseline.Email
	if firstPerson != baseline {
		t.Fatalf("expected first-name dependency isolation, got %#v and %#v", baseline, firstPerson)
	}

	lastWorld := layer1WorldWithDependency(t, world, lastNameDatasetComponentName, func(data *italyData, component Component) {
		data.lastNameDependency = component
	})
	lastPerson, err := lastWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-last-name-dependency Person, got %v", err)
	}
	lastPerson.LastName = baseline.LastName
	lastPerson.Email = baseline.Email
	if lastPerson != baseline {
		t.Fatalf("expected last-name dependency isolation, got %#v and %#v", baseline, lastPerson)
	}

	placesWorld := layer1WorldWithDependency(t, world, placesDatasetComponentName, func(data *italyData, component Component) {
		data.placesDependency = component
	})
	placesPerson, err := placesWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-places-dependency Person, got %v", err)
	}
	placesPerson.Address = baseline.Address
	if placesPerson != baseline {
		t.Fatalf("expected places dependency isolation, got %#v and %#v", baseline, placesPerson)
	}

	postalWorld := layer1WorldWithDependency(t, world, postalDatasetComponentName, func(data *italyData, component Component) {
		data.postalDependency = component
	})
	postalPerson, err := postalWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-postal-dependency Person, got %v", err)
	}
	postalPerson.Address.PostalCode = baseline.Address.PostalCode
	if postalPerson != baseline {
		t.Fatalf("expected postal dependency isolation, got %#v and %#v", baseline, postalPerson)
	}

	streetWorld := layer1WorldWithDependency(t, world, streetNameDatasetComponentName, func(data *italyData, component Component) {
		data.streetNameDependency = component
	})
	streetPerson, err := streetWorld.Person(42)
	if err != nil {
		t.Fatalf("expected changed-street-dependency Person, got %v", err)
	}
	streetPerson.Address.Street = baseline.Address.Street
	if streetPerson != baseline {
		t.Fatalf("expected street dependency isolation, got %#v and %#v", baseline, streetPerson)
	}

}

// TestLayer1DependencyComponents verifies the exact centralized component
// names and raw frozen SHA-256 bytes.
func TestLayer1DependencyComponents(t *testing.T) {
	data, err := loadLayer1ItalyData()
	if err != nil {
		t.Fatalf("expected frozen Italy data, got %v", err)
	}
	tests := []struct {
		component Component
		name      string
		digest    string
	}{
		{data.ageDependency, ageDatasetComponentName, ageWeightsSHA256},
		{data.firstNameDependency, firstNameDatasetComponentName, firstNamesSHA256},
		{data.lastNameDependency, lastNameDatasetComponentName, lastNamesSHA256},
		{data.placesDependency, placesDatasetComponentName, placesSHA256},
		{data.postalDependency, postalDatasetComponentName, postalCodesSHA256},
		{data.streetNameDependency, streetNameDatasetComponentName, streetNamesSHA256},
	}
	for _, test := range tests {
		if test.component.name != test.name || test.component.kind != componentKindBytes {
			t.Fatalf("expected bytes dependency %q, got %#v", test.name, test.component)
		}
		if got := hex.EncodeToString(test.component.value); got != test.digest {
			t.Fatalf("expected dependency %q SHA-256 %s, got %s", test.name, test.digest, got)
		}
	}
}

// TestLayer1ItalyDataRuntimeValidation verifies all immutable lookup and weight
// structures built by the one-time runtime loader.
func TestLayer1ItalyDataRuntimeValidation(t *testing.T) {

	data, err := loadLayer1ItalyData()
	if err != nil {
		t.Fatalf("expected frozen Italy data, got %v", err)
	}
	if len(data.ages) != 55 || len(data.ageWeights) != 55 {
		t.Fatalf("expected 55 age rows/weights, got %d/%d", len(data.ages), len(data.ageWeights))
	}
	for position, row := range data.ages {
		if row.age != position+21 || row.weight == 0 || data.ageWeights[position] != row.weight {
			t.Fatalf("expected age row %d with positive matching weight, got %#v/%d", position+21, row, data.ageWeights[position])
		}
	}

	if len(data.firstNames) != 10 || len(data.firstNameWeights) != 10 {
		t.Fatalf("expected 10 first-name groups, got %d/%d", len(data.firstNames), len(data.firstNameWeights))
	}
	for group, rows := range data.firstNames {
		if len(rows) != 80 || len(data.firstNameWeights[group]) != 80 {
			t.Fatalf("expected 80 first names for %#v, got %d/%d", group, len(rows), len(data.firstNameWeights[group]))
		}
		if !validTierQuota(data.firstNameWeights[group], 20, 40, 20) {
			t.Fatalf("expected 20/40/20 first-name tier quota for %#v, got %v", group, data.firstNameWeights[group])
		}
		for position, row := range rows {
			if !validDisplayName(row.display) || !validASCIIAtom(row.asciiAtom) || row.weight != data.firstNameWeights[group][position] {
				t.Fatalf("expected valid first-name row for %#v at %d, got %#v", group, position, row)
			}
		}
	}

	if len(data.lastNames) != 600 || !validTierQuota(data.lastNameWeights, 240, 240, 120) {
		t.Fatalf("expected 600 last names with 240/240/120 quota, got %d/%v", len(data.lastNames), data.lastNameWeights)
	}
	if len(data.streetNames) != 400 || !validTierQuota(data.streetNameWeights, 160, 160, 80) {
		t.Fatalf("expected 400 street names with 160/160/80 quota, got %d/%v", len(data.streetNames), data.streetNameWeights)
	}

	if len(data.places) != 7896 || len(data.postalCodes) != 7896 {
		t.Fatalf("expected 7896 places/postal sets, got %d/%d", len(data.places), len(data.postalCodes))
	}
	known := make(map[string]struct{}, len(data.places))
	for position, place := range data.places {
		if !validFixedDigits(place.istatCode, 6) || !validDisplayName(place.municipality) ||
			!validStateProv(place.stateProv) || !validDisplayName(place.region) || place.population == 0 {
			t.Fatalf("expected valid place at %d, got %#v", position, place)
		}
		if _, exists := known[place.istatCode]; exists {
			t.Fatalf("expected unique municipality key, got duplicate %q", place.istatCode)
		}
		known[place.istatCode] = struct{}{}
		postalCodes, exists := data.postalCodes[place.istatCode]
		if !exists || len(postalCodes) == 0 {
			t.Fatalf("expected postal codes for %q, got none", place.istatCode)
		}
		for postalPosition, postalCode := range postalCodes {
			if !validFixedDigits(postalCode, 5) || postalPosition > 0 && postalCode == postalCodes[postalPosition-1] {
				t.Fatalf("expected valid unique CAP for %q, got %q", place.istatCode, postalCode)
			}
		}
		if !slices.IsSorted(postalCodes) {
			t.Fatalf("expected sorted CAP set for %q, got %v", place.istatCode, postalCodes)
		}
	}
	for code := range data.postalCodes {
		if _, exists := known[code]; !exists {
			t.Fatalf("expected postal municipality %q in places, got none", code)
		}
	}

}

// TestLayer1StreamPaths protects the complete Synthetic World v1 path list.
func TestLayer1StreamPaths(t *testing.T) {
	got := []string{
		streamIdentityGender,
		streamIdentityBirthAge,
		streamIdentityBirthDate,
		streamIdentityNameFirst,
		streamIdentityNameLast,
		streamIdentityEmailProvider,
		streamIdentityEmailPattern,
		streamIdentityEmailToken,
		streamIdentitySiteDomain,
		streamIdentitySiteToken,
		streamIdentityPhone,
		streamGeographyPlaceMixture,
		streamGeographyPlacePopulation,
		streamGeographyPlaceUniform,
		streamGeographyPlacePostal,
		streamGeographyAddressType,
		streamGeographyAddressFragment,
		streamGeographyAddressCivicA,
		streamGeographyAddressCivicB,
		streamGeographyAddressSuffixPresence,
		streamGeographyAddressSuffixChoice,
		streamPhotoSelect,
	}
	want := []string{
		"identity/gender",
		"identity/birth/age",
		"identity/birth/date",
		"identity/name/first",
		"identity/name/last",
		"identity/email/provider",
		"identity/email/pattern",
		"identity/email/token",
		"identity/site/domain",
		"identity/site/token",
		"identity/phone",
		"geography/place/mixture",
		"geography/place/population",
		"geography/place/uniform",
		"geography/place/postal",
		"geography/address/type",
		"geography/address/fragment",
		"geography/address/civic-a",
		"geography/address/civic-b",
		"geography/address/suffix-presence",
		"geography/address/suffix-choice",
		"photo/select",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected stream paths %v, got %v", want, got)
	}
}

func layer1WorldWithDependency(t *testing.T, world *World, name string, set func(*italyData, Component)) *World {
	t.Helper()
	component, err := BytesComponent(name, bytes.Repeat([]byte{0xa5}, 32))
	if err != nil {
		t.Fatalf("expected changed dependency %q, got %v", name, err)
	}
	data := *world.data
	set(&data, component)
	clone := *world
	clone.data = &data
	return &clone
}
