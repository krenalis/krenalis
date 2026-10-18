// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

const (
	usaSourceUSDSchema      = "usa-internal-source-usd-diagnostic-v1"
	usaSourceUSDProfile     = "us-2026a-socioeconomic-v1"
	usaSourceUSDVersion     = "generation-us-v1"
	usaSourceUSDRecordBytes = 16 << 10
)

// usaSourceUSDAdapter binds observations and typed facts to one real USA World.
// It owns no second model, roster, population cache or public snapshot.
type usaSourceUSDAdapter struct {
	world  *usaCountryWorld
	source *SourceInstance
}

func newUSASourceUSDAdapter(world *usaCountryWorld, config SourceInstanceConfig) (*usaSourceUSDAdapter, error) {

	if world == nil || world.data == nil || world.data.socioeconomic == nil || world.id == "" ||
		world.sampler.categorical == nil || config.Version != usaSourceUSDVersion {
		return nil, fmt.Errorf("prepared USA World and diagnostic source version are required")
	}
	d := &world.data.socioeconomic.document
	if d.Country != "US" || d.Profile != usaSourceUSDProfile || d.IncomeUnit.Currency != "USD" {
		return nil, fmt.Errorf("incompatible USA source country, profile or currency")
	}

	// This declared context replaces the Italian photo snapshot only in this
	// private diagnostic path. Source-v1's remaining framing is unchanged.
	identity, err := StringComponent("internal-world-id", string(world.id))
	if err != nil {
		return nil, err
	}
	if world.catalog != nil {
		digest, err := internationalDigest("fakedata/usa-demo-source/v1", struct {
			World  SocioeconomicWorldID
			Photos [32]byte
		}{world.id, world.catalog.checksum})
		if err != nil {
			return nil, err
		}
		identity, err = BytesComponent("internal-demo-id", digest[:])
		if err != nil {
			return nil, err
		}
	}
	namespace := world.sampler.categorical.base.namespace
	owner, err := newCountrySourceWorld(nil, namespace, identity, []CountryShare{{
		Code: "US", Version: usaSourceUSDVersion, Weight: 1, Generator: usaSourceUSDGenerator{world},
	}})
	if err != nil {
		return nil, err
	}
	source, err := newSourceInstance(owner, config)
	if err != nil {
		return nil, err
	}

	return &usaSourceUSDAdapter{world: world, source: source}, nil
}

// bind observes an already sampled Person; no Employer or Income draw occurs.
// Canonical identity is returned only through the separate hidden Oracle.
func (a *usaSourceUSDAdapter) bind(person usaCountryPerson) ([]usaSourceUSDRecord, []OracleRecord, error) {

	if a == nil || a.world == nil || a.source == nil {
		return nil, nil, fmt.Errorf("initialized USA source adapter is required")
	}
	index := person.Coordinate
	id, err := entityIDForIndex(a.source.world.namespace, personEntityKind, uint64(index))
	if err != nil {
		return nil, nil, err
	}
	if index > Layer1MaxPersonIndex || person.Person.ID != id || person.Country != "US" ||
		person.Currency != "USD" || person.Person.Address.Country != "US" ||
		person.AnchorAge < 21 || person.AnchorAge > 75 || person.Person.CurrentAge != person.AnchorAge {
		return nil, nil, fmt.Errorf("incompatible canonical USA source person")
	}
	if a.world.catalog == nil && person.Person.PhotoID != "" {
		return nil, nil, fmt.Errorf("unexpected USA source photo")
	}
	if a.world.catalog != nil {
		expected, err := a.world.catalog.selectPhoto(a.world.sampler.categorical.base.streams, index,
			person.Person.Gender, person.AnchorAge)
		if err != nil {
			return nil, nil, err
		}
		if person.Person.PhotoID != expected {
			return nil, nil, fmt.Errorf("incompatible canonical USA source photo")
		}
	}
	count, err := a.source.recordCount(index)
	if err != nil {
		return nil, nil, err
	}
	observations, err := a.source.observations(index, person.Person, count)
	if err != nil {
		return nil, nil, err
	}
	records := make([]usaSourceUSDRecord, count)
	oracle := make([]OracleRecord, count)
	for ordinal, observation := range observations {

		income := person.AnnualIncomeUSD
		record := usaSourceUSDRecord{
			Schema: usaSourceUSDSchema, Country: "US", Currency: "USD", Profile: usaSourceUSDProfile,
			WorldID: a.world.id, SourceInstanceID: a.source.config.ID, SourceVersion: a.source.config.Version,
			SourceProtocol: SourceSimulationVersion,
			Observation: usaSourceUSDObservation{
				ID: observation.ID, FirstName: observation.FirstName, LastName: observation.LastName,
				Email: observation.Email, Phone: observation.Phone, PhotoID: observation.PhotoID,
				Gender: person.Person.Gender, BirthDate: person.Person.BirthDate,
				CurrentAge: uint32(person.Person.CurrentAge), SiteURL: person.Person.SiteURL,
				Street: person.Person.Address.Street, Municipality: person.Person.Address.Municipality,
				State: person.Person.Address.StateProv, ZIP: person.Person.Address.PostalCode,
				LocalityID: person.LocalityID, Division: person.Division,
			},
			Education: person.Education, PrimaryActivity: person.PrimaryActivity, AnnualIncomeUSD: &income,
		}

		// Retain the catalog spelling in USA duplicate observations as well.
		if observation.FirstName != nil {
			record.Observation.FirstName = &person.Person.FirstName
		}

		if person.Employee != nil {
			if person.EmployerCompany == nil || person.SelfEmployed != nil {
				return nil, nil, fmt.Errorf("employee source projection requires its Company")
			}
			company := person.EmployerCompany
			record.Employee = &usaSourceUSDEmployee{
				EmployerCompanyID: company.ID, JobFamily: person.Employee.JobFamily, JobLevel: person.Employee.JobLevel,
				Industry: company.Industry, SizeBand: company.SizeBand,
			}
		}
		if person.SelfEmployed != nil {
			record.SelfEmployed = &usaSourceUSDSelfEmployed{
				JobFamily: person.SelfEmployed.JobFamily, Industry: person.SelfEmployed.Industry,
			}
		}
		err = a.validate(record)
		if err != nil {
			return nil, nil, err
		}
		if person.Employee != nil {
			employer := person.Employee.Employer
			sampler := &a.world.sampler.categorical.companies.sampler
			id, err := entityIDForIndex(a.source.world.namespace, "company", uint64(employer))
			if err != nil {
				return nil, nil, err
			}
			if uint64(employer) > uint64(len(sampler.companyCounts)) || person.EmployerCompany.ID != id ||
				person.EmployerCompany.EmployeeCount != sampler.companyCounts[employer-1] {
				return nil, nil, fmt.Errorf("USA source Employer differs from canonical Company")
			}
		}
		if person.Employee == nil && person.EmployerCompany != nil {
			return nil, nil, fmt.Errorf("unexpected USA source Company")
		}
		records[ordinal] = record
		oracle[ordinal] = OracleRecord{a.source.config.ID, observation.ID, person.Person.ID}

	}

	return records, oracle, nil
}

// decode verifies a bounded closed transport against its original logical
// record. It is not an application importer or an official checkpoint reader.
func (a *usaSourceUSDAdapter) decode(ctx context.Context, encoded []byte, expected usaSourceUSDRecord) (usaSourceUSDRecord, error) {

	if len(encoded) > usaSourceUSDRecordBytes {
		return usaSourceUSDRecord{}, fmt.Errorf("USA source record exceeds byte limit")
	}
	var record usaSourceUSDRecord
	err := decodeInternationalJSON(ctx, bytes.NewReader(encoded), &record)
	if err != nil {
		return usaSourceUSDRecord{}, err
	}
	err = a.validate(record)
	if err != nil {
		return usaSourceUSDRecord{}, err
	}
	if !reflect.DeepEqual(record, expected) {
		return usaSourceUSDRecord{}, fmt.Errorf("decoded USA observation differs from its logical source record")
	}

	return record, nil
}

func (a *usaSourceUSDAdapter) validate(record usaSourceUSDRecord) error {

	if a == nil || a.world == nil || a.source == nil || record.Schema != usaSourceUSDSchema ||
		record.Country != "US" || record.Currency != "USD" || record.Profile != usaSourceUSDProfile ||
		record.WorldID != a.world.id || record.SourceInstanceID != a.source.config.ID ||
		record.SourceVersion != a.source.config.Version || record.SourceProtocol != SourceSimulationVersion ||
		record.AnnualIncomeUSD == nil {
		return fmt.Errorf("incompatible USA source identity, currency or mandatory Income")
	}
	o := &record.Observation
	token, err := hex.DecodeString(strings.TrimPrefix(o.ID, "sr_v2_"+a.source.config.ID+"_"))
	if err != nil {
		return fmt.Errorf("invalid opaque USA source record ID")
	}
	if len(token) != 16 {
		return fmt.Errorf("invalid opaque USA source record ID")
	}
	var coordinate [16]byte
	a.source.idBlock.Decrypt(coordinate[:], token)
	index, ordinal := binary.BigEndian.Uint64(coordinate[:8]), binary.BigEndian.Uint64(coordinate[8:])
	if index >= uint64(Layer1MaxPersonIndex) || ordinal > 1 ||
		o.ID != a.source.recordID(PersonIndex(index+1), int(ordinal)) {
		return fmt.Errorf("USA source record ID is outside its source scope")
	}
	if o.CurrentAge < 21 || o.CurrentAge > 75 ||
		(o.Gender != GenderFemale && o.Gender != GenderMale) {
		return fmt.Errorf("invalid non-photo USA source observation")
	}
	if a.world.catalog == nil && o.PhotoID != nil {
		return fmt.Errorf("unexpected USA source photo")
	}
	if a.world.catalog != nil {
		if o.PhotoID == nil {
			return fmt.Errorf("missing USA demo photo")
		}
		expected, err := a.world.catalog.selectPhoto(a.world.sampler.categorical.base.streams,
			PersonIndex(index+1), o.Gender, int(o.CurrentAge))
		if err != nil {
			return err
		}
		if *o.PhotoID != expected {
			return fmt.Errorf("incompatible USA demo photo")
		}
	}
	for _, text := range []*string{&o.ID, o.FirstName, o.LastName, o.Email, o.Phone, &o.BirthDate,
		&o.SiteURL, &o.Street, &o.Municipality, &o.State, &o.ZIP, &o.LocalityID, &o.Division} {
		if text != nil && (!internationalText(*text) || len(*text) > 1024) {
			return fmt.Errorf("invalid bounded USA source text")
		}
	}
	place, err := a.world.data.base.geography.resolve(o.LocalityID)
	if err != nil {
		return err
	}
	if o.State != place.state.USPS || o.Division != place.state.Division || o.Municipality != place.name ||
		!slices.Contains(place.zips, o.ZIP) {
		return fmt.Errorf("incompatible USA source locality, state, ZIP or Division")
	}
	if !slices.Contains(socioeconomicEducationLevels[:], record.Education) {
		return fmt.Errorf("invalid USA source Education")
	}

	// Validate branch presence and relational Company observations.
	switch record.PrimaryActivity {
	case PrimaryActivityEmployee:
		e := record.Employee
		if e == nil || record.SelfEmployed != nil || !slices.Contains(socioeconomicJobFamilies[:], e.JobFamily) ||
			!slices.Contains(socioeconomicJobLevels[:], e.JobLevel) {
			return fmt.Errorf("invalid employee USA source payload")
		}
		namespace := a.source.world.namespace
		index, err := entityIndexFromID(namespace, "company", e.EmployerCompanyID)
		if err != nil {
			return err
		}
		sampler := &a.world.sampler.categorical.companies.sampler
		if index == 0 || index > uint64(len(sampler.companyCounts)) ||
			index > uint64(len(sampler.companyIndustries)) {
			return fmt.Errorf("USA source Employer is outside its Company roster")
		}
		count, industry := sampler.companyCounts[index-1], int(sampler.companyIndustries[index-1])
		if industry >= len(socioeconomicIndustries) || count == 0 || e.Industry != socioeconomicIndustries[industry] ||
			e.SizeBand != socioeconomicSizeBands[socioeconomicSizeBand(count)] {
			return fmt.Errorf("USA source Industry or SizeBand differs from its Employer")
		}
	case PrimaryActivitySelfEmployed:
		s := record.SelfEmployed
		if s == nil || record.Employee != nil || !slices.Contains(socioeconomicJobFamilies[:], s.JobFamily) ||
			!slices.Contains(socioeconomicIndustries[:], s.Industry) {
			return fmt.Errorf("invalid self-employed USA source payload")
		}
	case PrimaryActivityUnemployed, PrimaryActivityStudent, PrimaryActivityRetired, PrimaryActivityInactive:
		if record.Employee != nil || record.SelfEmployed != nil {
			return fmt.Errorf("unexpected nonworking USA source payload")
		}
	default:
		return fmt.Errorf("invalid USA source PrimaryActivity")
	}

	return nil
}

type usaSourceUSDEmployee struct {
	EmployerCompanyID string          `json:"employerCompanyID"`
	JobFamily         JobFamily       `json:"jobFamily"`
	JobLevel          JobLevel        `json:"jobLevel"`
	Industry          Industry        `json:"industry"`
	SizeBand          CompanySizeBand `json:"sizeBand"`
}

// usaSourceUSDGenerator is the Layer 1 view of the same authoritative World.
// The adapter carries its facts directly instead of sampling this view twice.
type usaSourceUSDGenerator struct {
	world *usaCountryWorld
}

// Person supplies the existing CountryGenerator contract without EUR facts.
func (g usaSourceUSDGenerator) Person(index PersonIndex) (Person, error) {
	person, err := g.world.person(index)
	if err != nil {
		return Person{}, err
	}
	return person.Person, nil
}

type usaSourceUSDObservation struct {
	ID           string  `json:"id"`
	FirstName    *string `json:"firstName,omitempty"`
	LastName     *string `json:"lastName,omitempty"`
	Email        *string `json:"email,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	PhotoID      *string `json:"photoID,omitempty"`
	Gender       Gender  `json:"gender"`
	BirthDate    string  `json:"birthDate"`
	CurrentAge   uint32  `json:"currentAge"`
	SiteURL      string  `json:"siteURL"`
	Street       string  `json:"street"`
	Municipality string  `json:"municipality"`
	State        string  `json:"state"`
	ZIP          string  `json:"zip"`
	LocalityID   string  `json:"localityID"`
	Division     string  `json:"division"`
}

// usaSourceUSDRecord is an internal observation, never a canonical Person,
// application document, SourceRecord alias, release or resumable checkpoint.
// A pointer makes mandatory zero Income distinct from an absent value.
type usaSourceUSDRecord struct {
	Schema           string                    `json:"schema"`
	Country          string                    `json:"country"`
	Currency         string                    `json:"currency"`
	Profile          string                    `json:"profile"`
	WorldID          SocioeconomicWorldID      `json:"socioeconomicWorldID"`
	SourceInstanceID string                    `json:"sourceInstanceID"`
	SourceVersion    string                    `json:"sourceVersion"`
	SourceProtocol   string                    `json:"sourceProtocol"`
	Observation      usaSourceUSDObservation   `json:"observation"`
	Education        EducationLevel            `json:"education"`
	PrimaryActivity  PrimaryActivity           `json:"primaryActivity"`
	AnnualIncomeUSD  *annualIncomeUSD          `json:"annualIncomeUSD"`
	Employee         *usaSourceUSDEmployee     `json:"employee,omitempty"`
	SelfEmployed     *usaSourceUSDSelfEmployed `json:"selfEmployed,omitempty"`
}

type usaSourceUSDSelfEmployed struct {
	JobFamily JobFamily `json:"jobFamily"`
	Industry  Industry  `json:"industry"`
}
