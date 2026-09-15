// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"net/http"
	"time"
)

// Stable Synthetic World v1 Layer 1 configuration.
const (
	Market               = "italy"
	MarketDataVersion    = "it-2026a"
	PersonSpecVersion    = "person-v1"
	WorldModelVersion    = "world-v1"
	PopulationAnchorDate = "2026-01-01"
	Layer1MaxPersonIndex = PersonIndex(60_466_176)
)

const (
	marketComponentName            = "market"
	marketDataVersionComponentName = "market-data-version"
	personSpecVersionComponentName = "person-spec-version"
	referenceDateComponentName     = "reference-date"
	worldModelVersionComponentName = "world-model-version"
	worldSeedComponentName         = "world-seed"

	streamIdentityGender                 = "identity/gender"
	streamIdentityBirthAge               = "identity/birth/age"
	streamIdentityBirthDate              = "identity/birth/date"
	streamIdentityNameFirst              = "identity/name/first"
	streamIdentityNameLast               = "identity/name/last"
	streamIdentityEmailProvider          = "identity/email/provider"
	streamIdentityEmailPattern           = "identity/email/pattern"
	streamIdentityEmailToken             = "identity/email/token"
	streamIdentitySiteDomain             = "identity/site/domain"
	streamIdentitySiteToken              = "identity/site/token"
	streamIdentityPhone                  = "identity/phone"
	streamGeographyPlaceMixture          = "geography/place/mixture"
	streamGeographyPlacePopulation       = "geography/place/population"
	streamGeographyPlaceUniform          = "geography/place/uniform"
	streamGeographyPlacePostal           = "geography/place/postal"
	streamGeographyAddressType           = "geography/address/type"
	streamGeographyAddressFragment       = "geography/address/fragment"
	streamGeographyAddressCivicA         = "geography/address/civic-a"
	streamGeographyAddressCivicB         = "geography/address/civic-b"
	streamGeographyAddressSuffixPresence = "geography/address/suffix-presence"
	streamGeographyAddressSuffixChoice   = "geography/address/suffix-choice"
	streamPhotoSelect                    = "photo/select"

	emailAndSiteTokenModulus = uint64(60_466_176)
	phoneAffineA             = uint64(685_006_667)
	phoneAffineB             = uint64(934_416_303)
	phoneModulus             = uint64(999_999_990)
	phoneTokenOffset         = uint64(10)
	phonePrefix              = "+39000"
)

var (
	emailPatterns = weightedStringTable{
		values:  []string{"first.last.token", "initial.last.token", "first.token", "first-last.token"},
		weights: []uint64{40, 25, 20, 15},
	}
	emailProviders = weightedStringTable{
		values: []string{
			"postauno.test", "nuvolamail.test", "mailia.test", "casellablu.test",
			"lettera.test", "reteposta.test", "postafacile.test", "casellamia.test",
		},
		weights: []uint64{24, 20, 16, 13, 10, 8, 5, 4},
	}
	siteDomains = weightedStringTable{
		values:  []string{"profilo.test", "pagina.test", "identita.test", "spaziopersonale.test"},
		weights: []uint64{45, 30, 15, 10},
	}
	streetTypes = weightedStringTable{
		values:  []string{"Via", "Viale", "Piazza", "Corso", "Largo", "Vicolo"},
		weights: []uint64{55, 15, 12, 8, 5, 5},
	}
	streetSuffixes = [...]string{"/A", "/B", "/C", "bis"}
)

// Address is the minimal Italian ground-truth address for a Person.
type Address struct {
	Street       string
	Municipality string
	StateProv    string
	PostalCode   string
	Country      string
}

// Gender is a Person's synthetic ground-truth gender partition.
type Gender string

// Supported Layer 1 genders.
const (
	GenderFemale Gender = "female"
	GenderMale   Gender = "male"
)

// Person contains complete stable Layer 1 ground truth and derived current age.
type Person struct {
	ID         string
	Gender     Gender
	BirthDate  string
	CurrentAge int
	FirstName  string
	LastName   string
	Email      string
	Phone      string
	SiteURL    string
	Address    Address
	PhotoID    string
}

// WorldConfig configures one immutable Synthetic World v1 snapshot.
type WorldConfig struct {
	IdentityNamespace IdentityNamespace
	WorldSeed         uint64
	ReferenceDate     string
	FaceCatalog       *FaceCatalog
}

// World provides deterministic random access to Layer 1 Persons.
type World struct {
	namespace     IdentityNamespace
	referenceDate time.Time
	referenceText string
	worldID       WorldID
	snapshotID    SnapshotID
	streams       streamFactory
	data          *italyData
	faceCatalog   *FaceCatalog
	emailToken    affinePermutation
	siteToken     affinePermutation
}

// NewWorld validates config and returns an immutable Layer 1 World.
// It returns ErrInvalidIdentityNamespace, ErrInvalidReferenceDate,
// ErrCorruptFrozenDataset, ErrInvalidFaceCatalog, or ErrNoCompatiblePhoto as
// appropriate.
func NewWorld(config WorldConfig) (*World, error) {
	return newWorld(config)
}

func newWorld(config WorldConfig) (*World, error) {

	referenceDate, err := parseReferenceDate(config.ReferenceDate)
	if err != nil {
		return nil, err
	}
	data, err := loadLayer1ItalyData()
	if err != nil {
		return nil, err
	}
	err = config.FaceCatalog.validate()
	if err != nil {
		return nil, err
	}
	for _, gender := range [...]Gender{GenderFemale, GenderMale} {
		if !config.FaceCatalog.hasUsableGender(gender) {
			return nil, fmt.Errorf("%w: gender %s", ErrNoCompatiblePhoto, gender)
		}
	}

	market, err := StringComponent(marketComponentName, Market)
	if err != nil {
		return nil, err
	}
	marketDataVersion, err := StringComponent(marketDataVersionComponentName, MarketDataVersion)
	if err != nil {
		return nil, err
	}
	personSpecVersion, err := StringComponent(personSpecVersionComponentName, PersonSpecVersion)
	if err != nil {
		return nil, err
	}
	worldModelVersion, err := StringComponent(worldModelVersionComponentName, WorldModelVersion)
	if err != nil {
		return nil, err
	}
	worldSeed, err := Uint64Component(worldSeedComponentName, config.WorldSeed)
	if err != nil {
		return nil, err
	}
	worldID, err := ComposeWorldID(
		config.IdentityNamespace, market, marketDataVersion, personSpecVersion, worldModelVersion, worldSeed,
	)
	if err != nil {
		return nil, err
	}

	reference, err := StringComponent(referenceDateComponentName, config.ReferenceDate)
	if err != nil {
		return nil, err
	}
	faceVersion, err := StringComponent(faceCatalogVersionComponentName, config.FaceCatalog.version)
	if err != nil {
		return nil, err
	}
	faceChecksum, err := BytesComponent(faceCatalogSHA256ComponentName, config.FaceCatalog.checksum[:])
	if err != nil {
		return nil, err
	}
	snapshotID, err := ComposeSnapshotID(worldID, reference, faceVersion, faceChecksum)
	if err != nil {
		return nil, err
	}
	streams, err := newStreamFactory(worldModelVersion, worldSeed)
	if err != nil {
		return nil, err
	}
	emailToken, err := deriveAffinePermutation(streams, streamIdentityEmailToken)
	if err != nil {
		return nil, err
	}
	siteToken, err := deriveAffinePermutation(streams, streamIdentitySiteToken)
	if err != nil {
		return nil, err
	}

	return &World{
		namespace: config.IdentityNamespace, referenceDate: referenceDate, referenceText: config.ReferenceDate,
		worldID: worldID, snapshotID: snapshotID, streams: streams, data: data, faceCatalog: config.FaceCatalog,
		emailToken: emailToken, siteToken: siteToken,
	}, nil
}

// Person returns the deterministic Layer 1 Person at index.
// It returns ErrInvalidIndex for zero and ErrLayer1IndexOutOfRange above the
// Layer 1 operational maximum.
func (w *World) Person(index PersonIndex) (Person, error) {
	return w.person(index)
}

// PhotoAsset returns public metadata for a verified photo in this World's
// catalog.
// It returns ErrInvalidPhotoAsset when photoID or size is unavailable.
func (w *World) PhotoAsset(photoID string, size PhotoSize) (PhotoAsset, error) {
	return w.photoAsset(photoID, size)
}

// PhotoHandler returns an HTTP handler for this World's verified photo bytes.
func (w *World) PhotoHandler() http.Handler {
	return w.photoHandler()
}

// ReferenceDate returns the snapshot reference date in YYYY-MM-DD form.
func (w *World) ReferenceDate() string {
	return w.referenceText
}

// SnapshotID returns this World's snapshot fingerprint.
func (w *World) SnapshotID() SnapshotID {
	return w.snapshotID
}

// WorldID returns this World's non-temporal configuration fingerprint.
func (w *World) WorldID() WorldID {
	return w.worldID
}

func (w *World) address(index PersonIndex) (Address, error) {

	mixture, err := w.personStream(index, streamGeographyPlaceMixture)
	if err != nil {
		return Address{}, err
	}
	populationWeighted, err := mixture.BernoulliRatio(95, 100)
	if err != nil {
		return Address{}, err
	}
	place, err := w.municipality(index, populationWeighted)
	if err != nil {
		return Address{}, err
	}

	postalRNG, err := w.personStream(index, streamGeographyPlacePostal, w.data.postalDependency)
	if err != nil {
		return Address{}, err
	}
	postalCodes := w.data.postalCodes[place.istatCode]
	postalPosition, err := postalRNG.Uint64n(uint64(len(postalCodes)))
	if err != nil {
		return Address{}, err
	}

	streetType, err := w.weightedString(index, streamGeographyAddressType, streetTypes)
	if err != nil {
		return Address{}, err
	}
	fragmentRNG, err := w.personStream(index, streamGeographyAddressFragment, w.data.streetNameDependency)
	if err != nil {
		return Address{}, err
	}
	fragmentPosition, err := fragmentRNG.WeightedIndex(w.data.streetNameWeights)
	if err != nil {
		return Address{}, err
	}

	civicARNG, err := w.personStream(index, streamGeographyAddressCivicA)
	if err != nil {
		return Address{}, err
	}
	civicA, err := civicARNG.Uint64n(250)
	if err != nil {
		return Address{}, err
	}
	civicBRNG, err := w.personStream(index, streamGeographyAddressCivicB)
	if err != nil {
		return Address{}, err
	}
	civicB, err := civicBRNG.Uint64n(250)
	if err != nil {
		return Address{}, err
	}
	civic := 1 + min(civicA, civicB)

	suffix := ""
	suffixPresenceRNG, err := w.personStream(index, streamGeographyAddressSuffixPresence)
	if err != nil {
		return Address{}, err
	}
	hasSuffix, err := suffixPresenceRNG.BernoulliRatio(7, 100)
	if err != nil {
		return Address{}, err
	}
	if hasSuffix {
		suffixRNG, err := w.personStream(index, streamGeographyAddressSuffixChoice)
		if err != nil {
			return Address{}, err
		}
		position, err := suffixRNG.Uint64n(uint64(len(streetSuffixes)))
		if err != nil {
			return Address{}, err
		}
		suffix = streetSuffixes[position]
	}

	street := fmt.Sprintf("%s %s %d", streetType, w.data.streetNames[fragmentPosition].display, civic)
	if suffix == "bis" {
		street += " bis"
	} else {
		street += suffix
	}

	return Address{
		Street: street, Municipality: place.municipality, StateProv: place.stateProv,
		PostalCode: postalCodes[postalPosition], Country: "IT",
	}, nil
}

func (w *World) birth(index PersonIndex) (int, string, time.Time, error) {

	ageRNG, err := w.personStream(index, streamIdentityBirthAge, w.data.ageDependency)
	if err != nil {
		return 0, "", time.Time{}, err
	}
	agePosition, err := ageRNG.WeightedIndex(w.data.ageWeights)
	if err != nil {
		return 0, "", time.Time{}, err
	}
	anchorAge := w.data.ages[agePosition].age

	anchor := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	start := anchor.AddDate(-anchorAge-1, 0, 1)
	end := anchor.AddDate(-anchorAge, 0, 0)
	days := uint64(end.Sub(start)/(24*time.Hour)) + 1
	dateRNG, err := w.personStream(index, streamIdentityBirthDate)
	if err != nil {
		return 0, "", time.Time{}, err
	}
	offset, err := dateRNG.Uint64n(days)
	if err != nil {
		return 0, "", time.Time{}, err
	}
	birthDate := start.AddDate(0, 0, int(offset))

	return anchorAge, birthDate.Format(time.DateOnly), birthDate, nil
}

func (w *World) email(index PersonIndex, firstName, lastName string) (string, error) {

	provider, err := w.weightedString(index, streamIdentityEmailProvider, emailProviders)
	if err != nil {
		return "", err
	}
	pattern, err := w.weightedString(index, streamIdentityEmailPattern, emailPatterns)
	if err != nil {
		return "", err
	}
	token := encodeBase36Fixed5(w.emailToken.value(index))

	var local string
	switch pattern {
	case "first.last.token":
		local = firstName + "." + lastName + "." + token
	case "initial.last.token":
		local = firstName[:1] + "." + lastName + "." + token
	case "first.token":
		local = firstName + "." + token
	case "first-last.token":
		local = firstName + "-" + lastName + "." + token
	default:
		return "", fmt.Errorf("%w: unsupported email pattern", ErrCorruptFrozenDataset)
	}

	return local + "@" + provider, nil
}

func (w *World) firstName(index PersonIndex, gender Gender, birthYear int) (string, string, error) {
	cohort, err := cohortForYear(birthYear)
	if err != nil {
		return "", "", err
	}
	group := firstNameGroup{gender: gender, cohort: cohort}
	rows, exists := w.data.firstNames[group]
	if !exists {
		return "", "", fmt.Errorf("%w: missing first-name group", ErrCorruptFrozenDataset)
	}
	rng, err := w.personStream(index, streamIdentityNameFirst, w.data.firstNameDependency)
	if err != nil {
		return "", "", err
	}
	position, err := rng.WeightedIndex(w.data.firstNameWeights[group])
	if err != nil {
		return "", "", err
	}
	return rows[position].display, rows[position].asciiAtom, nil
}

func (w *World) gender(index PersonIndex) (Gender, error) {
	rng, err := w.personStream(index, streamIdentityGender)
	if err != nil {
		return "", err
	}
	value, err := rng.Uint64n(2)
	if err != nil {
		return "", err
	}
	if value == 0 {
		return GenderFemale, nil
	}
	return GenderMale, nil
}

func (w *World) lastName(index PersonIndex) (string, string, error) {

	rng, err := w.personStream(index, streamIdentityNameLast, w.data.lastNameDependency)
	if err != nil {
		return "", "", err
	}
	position, err := rng.WeightedIndex(w.data.lastNameWeights)
	if err != nil {
		return "", "", err
	}

	row := w.data.lastNames[position]

	return row.display, row.asciiAtom, nil
}

func (w *World) municipality(index PersonIndex, populationWeighted bool) (placeRow, error) {

	if !populationWeighted {
		rng, err := w.personStream(index, streamGeographyPlaceUniform, w.data.placesDependency)
		if err != nil {
			return placeRow{}, err
		}
		position, err := rng.Uint64n(uint64(len(w.data.places)))
		if err != nil {
			return placeRow{}, err
		}
		return w.data.places[position], nil
	}

	rng, err := w.personStream(index, streamGeographyPlacePopulation, w.data.placesDependency)
	if err != nil {
		return placeRow{}, err
	}
	bucketPosition, err := rng.WeightedIndex(w.data.placeWeightBucketTotals)
	if err != nil {
		return placeRow{}, err
	}
	bucket := w.data.placeWeightBuckets[bucketPosition]
	withinBucket, err := rng.WeightedIndex(bucket.weights)
	if err != nil {
		return placeRow{}, err
	}

	return w.data.places[bucket.start+withinBucket], nil
}

func (w *World) person(index PersonIndex) (Person, error) {

	if index == 0 {
		return Person{}, ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return Person{}, ErrLayer1IndexOutOfRange
	}

	id, err := PersonIDForIndex(w.namespace, index)
	if err != nil {
		return Person{}, err
	}
	gender, err := w.gender(index)
	if err != nil {
		return Person{}, err
	}
	anchorAge, birthDate, birthTime, err := w.birth(index)
	if err != nil {
		return Person{}, err
	}
	firstName, firstASCII, err := w.firstName(index, gender, birthTime.Year())
	if err != nil {
		return Person{}, err
	}
	lastName, lastASCII, err := w.lastName(index)
	if err != nil {
		return Person{}, err
	}
	email, err := w.email(index, firstASCII, lastASCII)
	if err != nil {
		return Person{}, err
	}
	siteURL, err := w.siteURL(index)
	if err != nil {
		return Person{}, err
	}
	address, err := w.address(index)
	if err != nil {
		return Person{}, err
	}
	photoID, err := w.faceCatalog.selectPhoto(w.streams, index, gender, anchorAge)
	if err != nil {
		return Person{}, err
	}

	return Person{
		ID: id, Gender: gender, BirthDate: birthDate, CurrentAge: ageAt(birthTime, w.referenceDate),
		FirstName: firstName, LastName: lastName, Email: email, Phone: phoneForIndex(index), SiteURL: siteURL,
		Address: address, PhotoID: photoID,
	}, nil
}

func (w *World) personStream(index PersonIndex, path string, dependencies ...Component) (splitMix64, error) {
	return w.streams.entityStream(personEntityKind, uint64(index), path, dependencies...)
}

func (w *World) photoAsset(photoID string, size PhotoSize) (PhotoAsset, error) {
	return w.faceCatalog.asset(photoID, size)
}

func (w *World) photoHandler() http.Handler {
	return w.faceCatalog.handler()
}

func (w *World) siteURL(index PersonIndex) (string, error) {
	domain, err := w.weightedString(index, streamIdentitySiteDomain, siteDomains)
	if err != nil {
		return "", err
	}
	token := encodeBase36Fixed5(w.siteToken.value(index))
	return "https://" + domain + "/" + token, nil
}

func (w *World) weightedString(index PersonIndex, path string, table weightedStringTable) (string, error) {
	rng, err := w.personStream(index, path)
	if err != nil {
		return "", err
	}
	position, err := rng.WeightedIndex(table.weights)
	if err != nil {
		return "", err
	}
	return table.values[position], nil
}
