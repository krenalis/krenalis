// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

const (
	layer1TestReferenceDate = "2026-01-01"
	layer1TestSeed          = uint64(726381)
	goldenLayer1WorldID     = WorldID("world_v1_e51e7922e0763f75ac47a1f3973253bd20ea3a1772158d5963f38d546304ef9b")
	goldenLayer1SnapshotID  = SnapshotID("snapshot_v1_35163bcc47f0cf94a2ac55a20b6c8f86bc0612ccf064e2a6eff08819d320e71a")
)

var loadLayer1TestCatalog = sync.OnceValues(func() (*FaceCatalog, error) {
	return loadSelectedFaceCatalogFixture(
		context.Background(), fakefacegenFixtureDirectory, []string{"face-000013", "face-000037"},
	)
})

type stablePersonGolden struct {
	index        PersonIndex
	id           string
	gender       Gender
	birthDate    string
	firstName    string
	lastName     string
	email        string
	phone        string
	siteURL      string
	municipality string
	stateProv    string
	postalCode   string
	street       string
	photoID      string
}

// TestLayer1AgeAndBirthDate verifies exact anchor ages, Gregorian boundaries,
// leap-day anniversaries, and the complete ReferenceDate range.
func TestLayer1AgeAndBirthDate(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	for index := PersonIndex(1); index <= 10_000; index++ {
		anchorAge, _, birthDate, err := world.birth(index)
		if err != nil {
			t.Fatalf("expected birth data for index %d, got %v", index, err)
		}
		anchor, err := time.Parse(time.DateOnly, PopulationAnchorDate)
		if err != nil {
			t.Fatalf("expected valid population anchor, got %v", err)
		}
		if got := ageAt(birthDate, anchor); got != anchorAge {
			t.Fatalf("expected anchor age %d for %s, got %d", anchorAge, birthDate.Format(time.DateOnly), got)
		}
	}

	tests := []struct {
		name      string
		birthDate string
		reference string
		want      int
	}{
		{"day before birthday", "1980-09-16", "2026-09-15", 45},
		{"birthday", "1980-09-15", "2026-09-15", 46},
		{"day after birthday", "1980-09-14", "2026-09-15", 46},
		{"leap day before non-leap anniversary", "2000-02-29", "2026-02-27", 25},
		{"leap day non-leap anniversary", "2000-02-29", "2026-02-28", 26},
		{"leap day before leap anniversary", "2000-02-29", "2028-02-28", 27},
		{"leap day leap anniversary", "2000-02-29", "2028-02-29", 28},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			birthDate, err := time.Parse(time.DateOnly, test.birthDate)
			if err != nil {
				t.Fatalf("expected valid birth date, got %v", err)
			}
			reference, err := time.Parse(time.DateOnly, test.reference)
			if err != nil {
				t.Fatalf("expected valid reference date, got %v", err)
			}
			if got := ageAt(birthDate, reference); got != test.want {
				t.Fatalf("expected age %d, got %d", test.want, got)
			}
		})
	}

	minimum := mustLayer1World(t, "2026-01-01", layer1TestSeed)
	maximum := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	foundAge80 := false
	for index := PersonIndex(1); index <= 10_000; index++ {
		atMinimum, err := minimum.Person(index)
		if err != nil {
			t.Fatalf("expected minimum-date Person %d, got %v", index, err)
		}
		atMaximum, err := maximum.Person(index)
		if err != nil {
			t.Fatalf("expected maximum-date Person %d, got %v", index, err)
		}
		if stablePerson(atMinimum) != stablePerson(atMaximum) {
			t.Fatalf("expected stable ground truth across ReferenceDate for index %d, got %#v and %#v", index, atMinimum, atMaximum)
		}
		if atMaximum.CurrentAge == 80 {
			foundAge80 = true
			break
		}
	}
	if !foundAge80 {
		t.Fatal("expected a generated Person reaching age 80, got none")
	}

}

// TestLayer1Determinism verifies random access, ordering, concurrency, seed,
// namespace, and ReferenceDate isolation.
func TestLayer1Determinism(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	indices := []PersonIndex{1, 2, 42, 1537291, Layer1MaxPersonIndex}
	want := make(map[PersonIndex]Person, len(indices))
	for _, index := range indices {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected Person %d, got %v", index, err)
		}
		want[index] = person
	}
	for position := len(indices) - 1; position >= 0; position-- {
		index := indices[position]
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected repeated Person %d, got %v", index, err)
		}
		if person != want[index] {
			t.Fatalf("expected repeated Person %#v, got %#v", want[index], person)
		}
	}

	concurrent := make([]Person, len(indices))
	var wait sync.WaitGroup
	for position := range indices {
		wait.Add(1)
		go func(position int) {
			defer wait.Done()
			person, err := world.Person(indices[position])
			if err != nil {
				t.Errorf("expected concurrent Person %d, got %v", indices[position], err)
				return
			}
			concurrent[position] = person
		}(position)
	}
	wait.Wait()
	for position, index := range indices {
		if concurrent[position] != want[index] {
			t.Fatalf("expected concurrent Person %#v, got %#v", want[index], concurrent[position])
		}
	}

	differentSeed := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed+1)
	baseline, err := world.Person(42)
	if err != nil {
		t.Fatalf("expected baseline Person, got %v", err)
	}
	changed, err := differentSeed.Person(42)
	if err != nil {
		t.Fatalf("expected changed-seed Person, got %v", err)
	}
	if baseline.ID != changed.ID {
		t.Fatalf("expected seed-independent ID %q, got %q", baseline.ID, changed.ID)
	}
	if stablePersonWithoutID(baseline) == stablePersonWithoutID(changed) {
		t.Fatalf("expected seed-dependent synthetic fields to change, got %#v", changed)
	}

	catalog := mustLayer1Catalog(t)
	namespace, err := NewIdentityNamespace("another-demo", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	differentNamespace, err := NewWorld(WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         layer1TestSeed,
		ReferenceDate:     layer1TestReferenceDate,
		FaceCatalog:       catalog,
	})
	if err != nil {
		t.Fatalf("expected different-namespace World, got %v", err)
	}
	namespacePerson, err := differentNamespace.Person(42)
	if err != nil {
		t.Fatalf("expected different-namespace Person, got %v", err)
	}
	if namespacePerson.ID == baseline.ID {
		t.Fatalf("expected namespace-dependent ID, got %q", namespacePerson.ID)
	}
	if stablePersonWithoutID(namespacePerson) != stablePersonWithoutID(baseline) {
		t.Fatalf("expected namespace-independent synthetic fields, got %#v and %#v", baseline, namespacePerson)
	}

}

// TestLayer1InvalidInputs verifies construction and operational index errors.
func TestLayer1InvalidInputs(t *testing.T) {

	catalog := mustLayer1Catalog(t)
	namespace, err := NewIdentityNamespace("krenalis-demo", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	for _, referenceDate := range []string{"", "2025-12-31", "2031-01-01", "2026-02-29", "2026-1-01", "2026-01-01T00:00:00Z"} {
		_, err := NewWorld(WorldConfig{
			IdentityNamespace: namespace,
			WorldSeed:         layer1TestSeed,
			ReferenceDate:     referenceDate,
			FaceCatalog:       catalog,
		})
		if err != nil {
			if !errors.Is(err, ErrInvalidReferenceDate) {
				t.Fatalf("expected invalid reference date for %q, got %v", referenceDate, err)
			}
			continue
		}
		t.Fatalf("expected invalid reference date for %q, got nil", referenceDate)
	}
	_, err = NewWorld(WorldConfig{
		IdentityNamespace: IdentityNamespace{},
		WorldSeed:         layer1TestSeed,
		ReferenceDate:     layer1TestReferenceDate,
		FaceCatalog:       catalog,
	})
	if err != nil {
		if !errors.Is(err, ErrInvalidIdentityNamespace) {
			t.Fatalf("expected invalid identity namespace, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid identity namespace, got nil")
	}

	_, err = NewWorld(WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         layer1TestSeed,
		ReferenceDate:     layer1TestReferenceDate,
	})
	if err != nil {
		if !errors.Is(err, ErrInvalidFaceCatalog) {
			t.Fatalf("expected invalid face catalog, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid face catalog, got nil")
	}

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	_, err = world.Person(0)
	if err != nil {
		if !errors.Is(err, ErrInvalidIndex) {
			t.Fatalf("expected invalid Layer 0 index, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected invalid Layer 0 index, got nil")
	}
	_, err = world.Person(Layer1MaxPersonIndex + 1)
	if err != nil {
		if !errors.Is(err, ErrLayer1IndexOutOfRange) {
			t.Fatalf("expected Layer 1 index error, got %v", err)
		}
	}
	if err == nil {
		t.Fatal("expected Layer 1 index error, got nil")
	}

}

// TestLayer1Names verifies gender/cohort partitioning and frozen atom use.
func TestLayer1Names(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	seenFirstWeights := map[uint64]struct{}{}
	seenLastWeights := map[uint64]struct{}{}
	seenGenders := map[Gender]struct{}{}
	for index := PersonIndex(1); index <= 10_000; index++ {
		gender, err := world.gender(index)
		if err != nil {
			t.Fatalf("expected gender for index %d, got %v", index, err)
		}
		_, _, birthDate, err := world.birth(index)
		if err != nil {
			t.Fatalf("expected birth for index %d, got %v", index, err)
		}
		firstName, firstASCII, err := world.firstName(index, gender, birthDate.Year())
		if err != nil {
			t.Fatalf("expected first name for index %d, got %v", index, err)
		}
		cohort, err := cohortForYear(birthDate.Year())
		if err != nil {
			t.Fatalf("expected cohort for index %d, got %v", index, err)
		}
		if !containsFirstName(world.data.firstNames[firstNameGroup{gender: gender, cohort: cohort}], firstName, firstASCII) {
			t.Fatalf("expected exact %s/%s name %q/%q, got no matching frozen row", gender, cohort, firstName, firstASCII)
		}
		seenGenders[gender] = struct{}{}
		for _, row := range world.data.firstNames[firstNameGroup{gender: gender, cohort: cohort}] {
			if row.display == firstName && row.asciiAtom == firstASCII {
				seenFirstWeights[row.weight] = struct{}{}
				break
			}
		}
		lastName, lastASCII, err := world.lastName(index)
		if err != nil {
			t.Fatalf("expected last name for index %d, got %v", index, err)
		}
		if !containsName(world.data.lastNames, lastName, lastASCII) {
			t.Fatalf("expected exact last name %q/%q, got no matching frozen row", lastName, lastASCII)
		}
		for _, row := range world.data.lastNames {
			if row.display == lastName && row.asciiAtom == lastASCII {
				seenLastWeights[row.weight] = struct{}{}
				break
			}
		}
	}
	if len(seenGenders) != 2 {
		t.Fatalf("expected both genders reachable, got %v", seenGenders)
	}
	if len(seenFirstWeights) != 3 {
		t.Fatalf("expected all first-name weight tiers reachable, got %v", seenFirstWeights)
	}
	if len(seenLastWeights) != 3 {
		t.Fatalf("expected all last-name weight tiers reachable, got %v", seenLastWeights)
	}

	for group, weights := range world.data.firstNameWeights {
		if !validTierQuota(weights, 20, 40, 20) {
			t.Fatalf("expected all weight tiers in group %#v, got %v", group, weights)
		}
	}
	if !validTierQuota(world.data.lastNameWeights, 240, 240, 120) {
		t.Fatalf("expected all last-name weight tiers, got %v", world.data.lastNameWeights)
	}
	if !containsFirstName(
		world.data.firstNames[firstNameGroup{gender: GenderMale, cohort: "1990-1999"}], "Niccolò", "niccolo",
	) {
		t.Fatal("expected frozen display/ASCII atom Niccolò/niccolo, got none")
	}
	if !containsName(world.data.lastNames, "Dalla Costa", "dalla-costa") {
		t.Fatal("expected frozen display/ASCII atom Dalla Costa/dalla-costa, got none")
	}

}

// TestLayer1PersonGolden protects stable Layer 1 v1 Person outputs for the
// documented namespace krenalis-demo generation 1, seed 726381, ReferenceDate
// 2026-01-01, and authentic selected fakefacegen fixture.
func TestLayer1PersonGolden(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	goldens := []stablePersonGolden{
		{
			index: 1, id: "person_v1_krenalis-demo_00000001_1z4yz7emgwe48", gender: GenderMale,
			birthDate: "1989-07-04", firstName: "Andrea", lastName: "Zangari",
			email: "andrea.zangari.ujfxo@mailia.test", phone: "+39000934416313",
			siteURL: "https://pagina.test/anwpg", municipality: "Prato", stateProv: "PO", postalCode: "59100",
			street: "Via Santa Lucia 59", photoID: "face-000037",
		},
		{
			index: 2, id: "person_v1_krenalis-demo_00000001_3hhcxdgc68vsb", gender: GenderMale,
			birthDate: "2000-10-05", firstName: "Achille", lastName: "Riva",
			email: "achille.qmrnp@mailia.test", phone: "+39000619422990",
			siteURL: "https://profilo.test/sxcvl", municipality: "Torre de' Roveri", stateProv: "BG", postalCode: "24060",
			street: "Via del Prato 19", photoID: "face-000037",
		},
		{
			index: 42, id: "person_v1_krenalis-demo_00000001_4r0n8d46pgtwb", gender: GenderMale,
			birthDate: "1951-01-14", firstName: "Luigi", lastName: "Bocchi",
			email: "luigi.ebskt@postafacile.test", phone: "+39000019689950",
			siteURL: "https://spaziopersonale.test/3fbp5", municipality: "Ponte Buggianese", stateProv: "PT",
			postalCode: "51019", street: "Via della Cura 168", photoID: "face-000037",
		},
		{
			index: 1537291, id: "person_v1_krenalis-demo_00000001_c5m6wxatk5862", gender: GenderFemale,
			birthDate: "1978-02-08", firstName: "Lucia", lastName: "Barbera",
			email: "lucia.barbera.nml46@casellablu.test", phone: "+39000844059283",
			siteURL: "https://spaziopersonale.test/qbqly", municipality: "Tuglie", stateProv: "LE", postalCode: "73058",
			street: "Via Sardegna 25", photoID: "face-000013",
		},
		{
			index: Layer1MaxPersonIndex, id: "person_v1_krenalis-demo_00000001_5ww7tsc2sdg6v", gender: GenderFemale,
			birthDate: "2004-01-28", firstName: "Rebecca", lastName: "Grassi",
			email: "rebecca.grassi.yg47n@mailia.test", phone: "+39000351602378",
			siteURL: "https://spaziopersonale.test/segjb", municipality: "San Marco in Lamis", stateProv: "FG",
			postalCode: "71014", street: "Vicolo degli Ulivi 178", photoID: "face-000013",
		},
	}

	for _, golden := range goldens {

		person, err := world.Person(golden.index)
		if err != nil {
			t.Fatalf("expected golden Person %d, got %v", golden.index, err)
		}
		got := stablePersonGolden{
			index: golden.index, id: person.ID, gender: person.Gender, birthDate: person.BirthDate,
			firstName: person.FirstName, lastName: person.LastName, email: person.Email, phone: person.Phone,
			siteURL: person.SiteURL, municipality: person.Address.Municipality, stateProv: person.Address.StateProv,
			postalCode: person.Address.PostalCode, street: person.Address.Street, photoID: person.PhotoID,
		}
		if got != golden {
			t.Fatalf("expected golden %#v, got %#v", golden, got)
		}

		id, err := PersonIDForIndex(world.namespace, golden.index)
		if err != nil {
			t.Fatalf("expected Layer 0 ID for %d, got %v", golden.index, err)
		}
		if person.ID != id {
			t.Fatalf("expected Layer 0 ID %q, got %q", id, person.ID)
		}
		assertPersonAssetConsistency(t, world, golden.index, person)

	}

}

// TestLayer1WorldAndSnapshotIDs verifies exact component integration and that
// composed IDs are not stream roots.
func TestLayer1WorldAndSnapshotIDs(t *testing.T) {

	world := mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed)
	if world.WorldID() != goldenLayer1WorldID {
		t.Fatalf("expected WorldID %s, got %s", goldenLayer1WorldID, world.WorldID())
	}
	if world.SnapshotID() != goldenLayer1SnapshotID {
		t.Fatalf("expected SnapshotID %s, got %s", goldenLayer1SnapshotID, world.SnapshotID())
	}

	components := []Component{
		mustStringComponent(t, marketComponentName, Market),
		mustStringComponent(t, marketDataVersionComponentName, MarketDataVersion),
		mustStringComponent(t, personSpecVersionComponentName, PersonSpecVersion),
		mustStringComponent(t, worldModelVersionComponentName, WorldModelVersion),
		mustUint64Component(t, worldSeedComponentName, layer1TestSeed),
	}
	wantWorldID, err := ComposeWorldID(world.namespace, components...)
	if err != nil {
		t.Fatalf("expected composed WorldID, got %v", err)
	}
	if world.WorldID() != wantWorldID {
		t.Fatalf("expected composed WorldID %s, got %s", wantWorldID, world.WorldID())
	}

	catalog := mustLayer1Catalog(t)
	catalogChecksum := catalog.Checksum()
	wantSnapshotID, err := ComposeSnapshotID(
		wantWorldID,
		mustStringComponent(t, referenceDateComponentName, layer1TestReferenceDate),
		mustStringComponent(t, faceCatalogVersionComponentName, catalog.Version()),
		mustBytesComponent(t, faceCatalogSHA256ComponentName, catalogChecksum[:]),
	)
	if err != nil {
		t.Fatalf("expected composed SnapshotID, got %v", err)
	}
	if world.SnapshotID() != wantSnapshotID {
		t.Fatalf("expected composed SnapshotID %s, got %s", wantSnapshotID, world.SnapshotID())
	}

	wantStreams, err := newStreamFactory(
		mustStringComponent(t, worldModelVersionComponentName, WorldModelVersion),
		mustUint64Component(t, worldSeedComponentName, layer1TestSeed),
	)
	if err != nil {
		t.Fatalf("expected Layer 1 stream root, got %v", err)
	}
	if world.streams.root != wantStreams.root {
		t.Fatalf("expected exact Layer 1 stream root %x, got %x", wantStreams.root, world.streams.root)
	}

	changedDate := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	if changedDate.WorldID() != world.WorldID() {
		t.Fatalf("expected ReferenceDate-independent WorldID %s, got %s", world.WorldID(), changedDate.WorldID())
	}
	if changedDate.SnapshotID() == world.SnapshotID() {
		t.Fatalf("expected ReferenceDate-dependent SnapshotID, got %s", changedDate.SnapshotID())
	}
	if changedDate.streams.root != world.streams.root {
		t.Fatalf("expected ReferenceDate-independent stream root %x, got %x", world.streams.root, changedDate.streams.root)
	}

}

func assertPersonAssetConsistency(t *testing.T, world *World, index PersonIndex, person Person) {
	t.Helper()
	birthDate, err := time.Parse(time.DateOnly, person.BirthDate)
	if err != nil {
		t.Fatalf("expected valid BirthDate %q, got %v", person.BirthDate, err)
	}
	if person.CurrentAge != ageAt(birthDate, world.referenceDate) {
		t.Fatalf("expected CurrentAge %d, got %d", ageAt(birthDate, world.referenceDate), person.CurrentAge)
	}
	_, _, selectedBirthDate, err := world.birth(index)
	if err != nil {
		t.Fatalf("expected selected birth date for %d, got %v", index, err)
	}
	cohort, err := cohortForYear(selectedBirthDate.Year())
	if err != nil {
		t.Fatalf("expected cohort for %d, got %v", index, err)
	}
	firstRows := world.data.firstNames[firstNameGroup{gender: person.Gender, cohort: cohort}]
	if !slices.ContainsFunc(firstRows, func(row firstNameRow) bool { return row.display == person.FirstName }) {
		t.Fatalf("expected first name %q in %s/%s data, got none", person.FirstName, person.Gender, cohort)
	}
	if !slices.ContainsFunc(world.data.lastNames, func(row nameRow) bool { return row.display == person.LastName }) {
		t.Fatalf("expected last name %q in frozen data, got none", person.LastName)
	}
	placePosition := slices.IndexFunc(world.data.places, func(place placeRow) bool {
		return place.municipality == person.Address.Municipality && place.stateProv == person.Address.StateProv
	})
	if placePosition < 0 {
		t.Fatalf("expected municipality %s/%s in frozen data, got none", person.Address.Municipality, person.Address.StateProv)
	}
	if !slices.Contains(world.data.postalCodes[world.data.places[placePosition].istatCode], person.Address.PostalCode) {
		t.Fatalf("expected CAP %s for %s, got none", person.Address.PostalCode, person.Address.Municipality)
	}
	wantPhotoID := "face-000013"
	if person.Gender == GenderMale {
		wantPhotoID = "face-000037"
	}
	if person.PhotoID != wantPhotoID {
		t.Fatalf("expected gender-compatible PhotoID %q, got %q", wantPhotoID, person.PhotoID)
	}
}

func containsFirstName(rows []firstNameRow, display, asciiAtom string) bool {
	return slices.ContainsFunc(rows, func(row firstNameRow) bool { return row.display == display && row.asciiAtom == asciiAtom })
}

func containsName(rows []nameRow, display, asciiAtom string) bool {
	return slices.ContainsFunc(rows, func(row nameRow) bool { return row.display == display && row.asciiAtom == asciiAtom })
}

func mustLayer1Catalog(t *testing.T) *FaceCatalog {
	t.Helper()
	catalog, err := loadLayer1TestCatalog()
	if err != nil {
		t.Fatalf("expected Layer 1 fixture catalog, got %v", err)
	}
	return catalog
}

func mustLayer1World(t *testing.T, referenceDate string, seed uint64) *World {
	t.Helper()
	namespace, err := NewIdentityNamespace("krenalis-demo", 1)
	if err != nil {
		t.Fatalf("expected identity namespace, got %v", err)
	}
	world, err := NewWorld(WorldConfig{
		IdentityNamespace: namespace,
		WorldSeed:         seed,
		ReferenceDate:     referenceDate,
		FaceCatalog:       mustLayer1Catalog(t),
	})
	if err != nil {
		t.Fatalf("expected Layer 1 World, got %v", err)
	}
	return world
}

func stablePerson(person Person) Person {
	person.CurrentAge = 0
	return person
}

func stablePersonWithoutID(person Person) Person {
	person.ID = ""
	person.CurrentAge = 0
	return person
}
