// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

const sourceSimulationVersion = "source-v1"

// CountryGenerator generates deterministic Layer 1 data at a stable index.
// Its Person ID must use the SourceWorld's identity namespace and index.
type CountryGenerator interface {
	Person(index PersonIndex) (Person, error)
}

// CountryShare configures one generation country and its relative weight.
// IT uses the supplied Layer 1 World and must have a nil Generator.
type CountryShare struct {
	Code      string
	Version   string
	Weight    uint64
	Generator CountryGenerator
}

// CountryPerson carries canonical data and its generation-country context.
// Country is a model/geography context, not citizenship.
type CountryPerson struct {
	Country string
	Person  Person
}

// SourceWorld chooses a country by Layer 1 coordinate and owns truth.
type SourceWorld struct {
	base      *World
	countries []CountryShare
	total     uint64
	streams   streamFactory
}

// NewSourceWorld validates all configured countries before any generation.
// The local limit of 16 country entries bounds configuration work and is not
// a limit on the population or on source-record cardinality.
func NewSourceWorld(base *World, countries []CountryShare) (*SourceWorld, error) {
	return newSourceWorld(base, countries)
}

func newSourceWorld(base *World, countries []CountryShare) (*SourceWorld, error) {

	if base == nil || len(countries) == 0 || len(countries) > 16 {
		return nil, fmt.Errorf("invalid source world configuration")
	}

	ordered := append([]CountryShare(nil), countries...)
	slices.SortFunc(ordered, func(a, b CountryShare) int { return strings.Compare(a.Code, b.Code) })
	components := make([]Component, 0, 1+3*len(ordered))
	snapshot, err := StringComponent("snapshot-id", string(base.snapshotID))
	if err != nil {
		return nil, err
	}
	components = append(components, snapshot)
	var total uint64
	for i, country := range ordered {

		if len(country.Code) != 2 || country.Code[0] < 'A' || country.Code[0] > 'Z' ||
			country.Code[1] < 'A' || country.Code[1] > 'Z' || !validLowerIdentifier(country.Version, 64) ||
			country.Weight == 0 || country.Weight > math.MaxUint64-total {
			return nil, fmt.Errorf("invalid country share")
		}
		if i > 0 && ordered[i-1].Code == country.Code {
			return nil, fmt.Errorf("duplicate country %s", country.Code)
		}
		if country.Code == "IT" {
			if country.Generator != nil || country.Version != MarketDataVersion {
				return nil, fmt.Errorf("invalid IT generator or version")
			}
		} else if country.Generator == nil {
			return nil, fmt.Errorf("unavailable country %s", country.Code)
		}
		total += country.Weight

		code, err := StringComponent("country-code-"+strconv.Itoa(i), country.Code)
		if err != nil {
			return nil, err
		}
		version, err := StringComponent("country-version-"+strconv.Itoa(i), country.Version)
		if err != nil {
			return nil, err
		}
		weight, err := Uint64Component("country-weight-"+strconv.Itoa(i), country.Weight)
		if err != nil {
			return nil, err
		}
		components = append(components, code, version, weight)

	}
	streams, err := newStreamFactory(components...)
	if err != nil {
		return nil, err
	}

	return &SourceWorld{base: base, countries: ordered, total: total, streams: streams}, nil
}

// Oracle returns hidden mappings for one source and person coordinate.
// It does not store or expose a population-wide map.
func (w *SourceWorld) Oracle(source *SourceInstance, index PersonIndex) ([]OracleRecord, error) {

	if w == nil || source == nil || source.world != w {
		return nil, fmt.Errorf("source does not belong to evaluation world")
	}

	count, err := source.recordCount(index)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}

	person, err := w.person(index)
	if err != nil {
		return nil, err
	}

	result := make([]OracleRecord, count)
	for ordinal := range result {
		result[ordinal] = OracleRecord{
			SourceInstanceID: source.config.ID,
			SourceRecordID:   source.recordID(index, ordinal),
			PersonID:         person.Person.ID,
		}
	}

	return result, nil
}

// Person returns typed canonical data and its assigned generation country.
func (w *SourceWorld) Person(index PersonIndex) (CountryPerson, error) {
	if w == nil {
		return CountryPerson{}, fmt.Errorf("nil source world")
	}
	return w.person(index)
}

func (w *SourceWorld) person(index PersonIndex) (CountryPerson, error) {

	if index == 0 || index > Layer1MaxPersonIndex {
		return CountryPerson{}, fmt.Errorf("invalid source person index %d", index)
	}

	rng, err := w.streams.entityStream(personEntityKind, uint64(index), "source/country")
	if err != nil {
		return CountryPerson{}, err
	}
	draw, err := rng.Uint64n(w.total)
	if err != nil {
		return CountryPerson{}, err
	}

	for _, country := range w.countries {
		if draw < country.Weight {
			var person Person
			if country.Code == "IT" {
				person, err = w.base.person(index)
			} else {
				person, err = country.Generator.Person(index)
			}
			if err != nil {
				return CountryPerson{}, err
			}
			id, err := PersonIDForIndex(w.base.namespace, index)
			if err != nil {
				return CountryPerson{}, err
			}
			if person.ID != id {
				return CountryPerson{}, fmt.Errorf("country %s returned wrong person identity", country.Code)
			}
			return CountryPerson{Country: country.Code, Person: person}, nil
		}
		draw -= country.Weight
	}
	panic("country selection has no country")

}

// SourceInstanceConfig configures one independently simulated source.
// IDs must distinguish simultaneously used sources in one evaluation World.
type SourceInstanceConfig struct {
	ID                   string
	Version              string
	CoverageNumerator    uint64
	CoverageDenominator  uint64
	DuplicateNumerator   uint64
	DuplicateDenominator uint64
}

// SourceInstance generates bounded, independent observations of a SourceWorld.
type SourceInstance struct {
	world   *SourceWorld
	config  SourceInstanceConfig
	streams streamFactory
	idBlock cipher.Block
}

// NewSourceInstance validates a source and derives its decision streams.
func NewSourceInstance(world *SourceWorld, config SourceInstanceConfig) (*SourceInstance, error) {
	return newSourceInstance(world, config)
}

func newSourceInstance(world *SourceWorld, config SourceInstanceConfig) (*SourceInstance, error) {

	if world == nil || !validLowerIdentifier(config.ID, 32) || !validLowerIdentifier(config.Version, 64) ||
		config.CoverageDenominator == 0 || config.CoverageNumerator > config.CoverageDenominator ||
		config.DuplicateDenominator == 0 || config.DuplicateNumerator > config.DuplicateDenominator {
		return nil, fmt.Errorf("invalid source instance configuration")
	}

	worldRoot, err := BytesComponent("source-world", world.streams.root[:])
	if err != nil {
		return nil, err
	}
	id, err := StringComponent("source-id", config.ID)
	if err != nil {
		return nil, err
	}
	version, err := StringComponent("source-version", config.Version)
	if err != nil {
		return nil, err
	}
	protocol, err := StringComponent("simulation-version", sourceSimulationVersion)
	if err != nil {
		return nil, err
	}
	coverageNumerator, err := Uint64Component("coverage-numerator", config.CoverageNumerator)
	if err != nil {
		return nil, err
	}
	coverageDenominator, err := Uint64Component("coverage-denominator", config.CoverageDenominator)
	if err != nil {
		return nil, err
	}
	duplicateNumerator, err := Uint64Component("duplicate-numerator", config.DuplicateNumerator)
	if err != nil {
		return nil, err
	}
	duplicateDenominator, err := Uint64Component("duplicate-denominator", config.DuplicateDenominator)
	if err != nil {
		return nil, err
	}
	streams, err := newStreamFactory(
		worldRoot, id, version, protocol, coverageNumerator, coverageDenominator,
		duplicateNumerator, duplicateDenominator,
	)
	if err != nil {
		return nil, err
	}
	idBlock, err := aes.NewCipher(streams.root[:])
	if err != nil {
		return nil, err
	}

	return &SourceInstance{world: world, config: config, streams: streams, idBlock: idBlock}, nil
}

// ID returns the source-local namespace used by Oracle mappings.
func (s *SourceInstance) ID() string {
	return s.config.ID
}

// Records generates zero, one, or two logical records for one person.
// The two-record maximum is this first policy's local operational bound; the
// SourceRecord and Oracle models do not impose a semantic cardinality limit.
func (s *SourceInstance) Records(index PersonIndex) ([]SourceRecord, error) {

	if s == nil {
		return nil, fmt.Errorf("nil source instance")
	}
	count, err := s.recordCount(index)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}

	canonical, err := s.world.person(index)
	if err != nil {
		return nil, err
	}

	records := make([]SourceRecord, count)
	for ordinal := range records {
		person := canonical.Person
		record := SourceRecord{ID: s.recordID(index, ordinal)}
		if person.PhotoID != "" {
			record.PhotoID = &person.PhotoID
		}
		if person.Address.Country != "" {
			record.Country = &person.Address.Country
		}
		if ordinal == 0 {
			record.FirstName = &person.FirstName
			record.LastName = &person.LastName
			record.Phone = &person.Phone
			rng, err := s.streams.entityStream(personEntityKind, uint64(index), "source/email/omit")
			if err != nil {
				return nil, err
			}
			omit, err := rng.BernoulliRatio(1, 8)
			if err != nil {
				return nil, err
			}
			if !omit {
				record.Email = &person.Email
			}
		} else {
			firstName := strings.ToUpper(person.FirstName)
			record.FirstName = &firstName
			record.LastName = &person.LastName
			phone := person.Phone
			if len(phone) > 0 && phone[len(phone)-1] >= '0' && phone[len(phone)-1] <= '9' {
				phone = phone[:len(phone)-1] + string('0'+(phone[len(phone)-1]-'0'+1)%10)
			}
			record.Phone = &phone
		}
		records[ordinal] = record
	}

	return records, nil
}

func (s *SourceInstance) recordCount(index PersonIndex) (int, error) {

	if index == 0 || index > Layer1MaxPersonIndex {
		return 0, fmt.Errorf("invalid source person index %d", index)
	}

	rng, err := s.streams.entityStream(personEntityKind, uint64(index), "source/membership")
	if err != nil {
		return 0, err
	}
	member, err := rng.BernoulliRatio(s.config.CoverageNumerator, s.config.CoverageDenominator)
	if err != nil {
		return 0, err
	}
	if !member {
		return 0, nil
	}

	rng, err = s.streams.entityStream(personEntityKind, uint64(index), "source/duplicate")
	if err != nil {
		return 0, err
	}
	duplicate, err := rng.BernoulliRatio(s.config.DuplicateNumerator, s.config.DuplicateDenominator)
	if err != nil {
		return 0, err
	}
	if duplicate {
		return 2, nil
	}

	return 1, nil
}

func (s *SourceInstance) recordID(index PersonIndex, ordinal int) string {
	// Distinct (index, ordinal) pairs encode distinct blocks. AES is a
	// permutation of blocks under this source's fixed key, so IDs are unique
	// within the source without a collision table.
	var coordinate [aes.BlockSize]byte
	binary.BigEndian.PutUint64(coordinate[:8], uint64(index)-1)
	binary.BigEndian.PutUint64(coordinate[8:], uint64(ordinal))
	var token [aes.BlockSize]byte
	s.idBlock.Encrypt(token[:], coordinate[:])
	return fmt.Sprintf("sr_v2_%s_%s", s.config.ID, hex.EncodeToString(token[:]))
}

// SourceRecord is an ordinary local observation with no canonical identity.
// A nil field means absent; an empty present string remains distinct.
// PhotoID refers to a catalog asset, not to a person. Country is the observed
// address country.
type SourceRecord struct {
	ID        string
	FirstName *string
	LastName  *string
	Email     *string
	Phone     *string
	PhotoID   *string
	Country   *string
}

// OracleRecord is hidden evaluation truth, never part of ordinary CSV output.
type OracleRecord struct {
	SourceInstanceID string
	SourceRecordID   string
	PersonID         string
}
