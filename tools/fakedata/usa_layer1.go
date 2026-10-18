// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"time"
)

// usaLayer1Harness composes internal candidates only. It neither constructs a
// public World nor loads photographs, socioeconomic data or Source Simulation.
type usaLayer1Harness struct {
	namespace IdentityNamespace
	streams   streamFactory
	base      *usaBaseCandidate
	phones    *usaNPACandidate
	contacts  *usaContacts
}

// newUSALayer1Harness binds prepared USA inputs to the existing World factory
// without exposing a public World or loading socioeconomic data.
func newUSALayer1Harness(namespace IdentityNamespace, seed uint64, base *usaBaseCandidate, phones *usaNPACandidate) (*usaLayer1Harness, error) {

	err := namespace.validate()
	if err != nil {
		return nil, err
	}
	if base == nil || base.digest == ([32]byte{}) || phones == nil || len(phones.npas) != 368 {
		return nil, fmt.Errorf("prepared USA base and phone rule are required")
	}
	// Reuse the World factory procedure; national paths provide separation.
	model, err := StringComponent(worldModelVersionComponentName, WorldModelVersion)
	if err != nil {
		return nil, err
	}
	worldSeed, err := Uint64Component(worldSeedComponentName, seed)
	if err != nil {
		return nil, err
	}
	factory, err := newStreamFactory(model, worldSeed)
	if err != nil {
		return nil, err
	}
	contacts, err := newUSAContacts(factory)
	if err != nil {
		return nil, err
	}

	return &usaLayer1Harness{
		namespace: namespace, streams: factory, base: base, phones: phones, contacts: contacts,
	}, nil
}

// address retains the single selected locality so the caller carries exactly
// its state-derived Division forward into any future Layer 2 computation.
func (w *usaLayer1Harness) address(index PersonIndex) (Address, usaLocation, error) {

	if w == nil || w.base == nil || w.base.geography == nil || len(w.base.streets) != 256 {
		return Address{}, usaLocation{}, fmt.Errorf("initialized USA address inputs are required")
	}
	placeRNG, err := w.stream(index, streamGeographyPlacePopulation, w.base.dependencies[3])
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	place, err := w.base.geography.selectLocation(&placeRNG)
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	postalRNG, err := w.stream(index, streamGeographyPlacePostal, w.base.dependencies[4])
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	postal, err := postalRNG.Uint64n(uint64(len(place.zips)))
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	streetRNG, err := w.stream(index, streamGeographyAddressFragment, w.base.dependencies[5])
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	street, err := streetRNG.Uint64n(uint64(len(w.base.streets)))
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	civicRNG, err := w.stream(index, streamGeographyAddressCivicA)
	if err != nil {
		return Address{}, usaLocation{}, err
	}
	civic, err := civicRNG.Uint64n(9999)
	if err != nil {
		return Address{}, usaLocation{}, err
	}

	return Address{
		Street: fmt.Sprintf("%d %s", civic+1, w.base.streets[street]), Municipality: place.name,
		StateProv: place.state.USPS, PostalCode: place.zips[postal], Country: "US",
	}, place, nil
}

// person returns internal candidate ground truth without a PhotoID; photo
// qualification and reuse rights remain a separate prerequisite.
func (w *usaLayer1Harness) person(index PersonIndex) (usaLayer1Person, error) {

	if index == 0 {
		return usaLayer1Person{}, ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return usaLayer1Person{}, ErrLayer1IndexOutOfRange
	}
	if w == nil || w.base == nil || w.base.digest == ([32]byte{}) || w.contacts == nil || w.phones == nil {
		return usaLayer1Person{}, fmt.Errorf("initialized USA Layer 1 harness is required")
	}
	id, err := entityIDForIndex(w.namespace, personEntityKind, uint64(index))
	if err != nil {
		return usaLayer1Person{}, err
	}
	genderRNG, err := w.stream(index, streamIdentityGender)
	if err != nil {
		return usaLayer1Person{}, err
	}
	genderDraw, err := genderRNG.Uint64n(2)
	if err != nil {
		return usaLayer1Person{}, err
	}
	gender := GenderFemale
	if genderDraw == 1 {
		gender = GenderMale
	}

	ageRNG, err := w.stream(index, streamIdentityBirthAge, w.base.dependencies[0])
	if err != nil {
		return usaLayer1Person{}, err
	}
	position, err := ageRNG.WeightedIndex(w.base.ageWeights)
	if err != nil {
		return usaLayer1Person{}, err
	}
	anchorAge := int(w.base.ages[position].Age)
	anchor := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	start, end := anchor.AddDate(-anchorAge-1, 0, 1), anchor.AddDate(-anchorAge, 0, 0)
	dateRNG, err := w.stream(index, streamIdentityBirthDate)
	if err != nil {
		return usaLayer1Person{}, err
	}
	offset, err := dateRNG.Uint64n(uint64(end.Sub(start)/(24*time.Hour)) + 1)
	if err != nil {
		return usaLayer1Person{}, err
	}
	birth := start.AddDate(0, 0, int(offset))
	cohort, err := cohortForYear(birth.Year())
	if err != nil {
		return usaLayer1Person{}, err
	}

	firstRNG, err := w.stream(index, streamIdentityNameFirst, w.base.dependencies[1])
	if err != nil {
		return usaLayer1Person{}, err
	}
	first, err := w.base.firstNames.selectFirstName(&firstRNG, gender, cohort)
	if err != nil {
		return usaLayer1Person{}, err
	}
	lastRNG, err := w.stream(index, streamIdentityNameLast, w.base.dependencies[2])
	if err != nil {
		return usaLayer1Person{}, err
	}
	last, err := w.base.lastNames.selectSurname(&lastRNG)
	if err != nil {
		return usaLayer1Person{}, err
	}
	email, site, err := w.contacts.compose(index, first.asciiAtom, last.asciiAtom)
	if err != nil {
		return usaLayer1Person{}, err
	}
	phone, err := w.phones.phone(index)
	if err != nil {
		return usaLayer1Person{}, err
	}
	address, location, err := w.address(index)
	if err != nil {
		return usaLayer1Person{}, err
	}

	return usaLayer1Person{
		person: Person{
			ID: id, Gender: gender, BirthDate: birth.Format(time.DateOnly), CurrentAge: ageAt(birth, anchor),
			FirstName: first.display, LastName: last.display, Email: email, SiteURL: site, Phone: phone, Address: address,
		},
		anchorAge: anchorAge, cohort: cohort, localityID: location.id,
		division: location.state.Division, divisionOrdinal: location.state.DivisionOrdinal,
		addressLine: fmt.Sprintf("%s, %s, %s %s",
			address.Street, address.Municipality, address.StateProv, address.PostalCode),
	}, nil
}

// stream derives a national person stream after checking its coordinate domain.
func (w *usaLayer1Harness) stream(index PersonIndex, path string, dependencies ...Component) (splitMix64, error) {
	if index == 0 {
		return splitMix64{}, ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return splitMix64{}, ErrLayer1IndexOutOfRange
	}
	return w.streams.entityStream(personEntityKind, uint64(index), usaLayer1StreamPrefix+path, dependencies...)
}

type usaLayer1Person struct {
	person          Person
	addressLine     string
	anchorAge       int
	cohort          string
	localityID      string
	division        string
	divisionOrdinal uint32
}
