// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
)

// usaCountryPerson is an internal technical record, not Source CSV
// or SocioeconomicPerson. Its monetary field cannot contain an EUR value.
type usaCountryPerson struct {
	Schema          string             `json:"schema"`
	Coordinate      PersonIndex        `json:"coordinate"`
	Country         string             `json:"country"`
	Currency        string             `json:"currency"`
	Person          Person             `json:"person"`
	AnchorAge       int                `json:"anchorAge"`
	LocalityID      string             `json:"localityID"`
	Division        string             `json:"division"`
	Education       EducationLevel     `json:"education"`
	PrimaryActivity PrimaryActivity    `json:"primaryActivity"`
	Employee        *EmployeeFacts     `json:"employee"`
	SelfEmployed    *SelfEmployedFacts `json:"selfEmployed"`
	EmployerCompany *Company           `json:"employerCompany"`
	AnnualIncomeUSD annualIncomeUSD    `json:"annualIncomeUSD"`
}

// usaCountryWorld binds full compiled identity to shared internal samplers.
// The photo catalog is present only for an explicitly prepared internal demo.
type usaCountryWorld struct {
	data    *usaCountryData
	sampler usaIncomeHarness
	baseID  WorldID
	id      SocioeconomicWorldID
	catalog *FaceCatalog
}

// newUSACountryWorld prepares complete internal US/USD state and technical
// identities without opening the Italian public World or monetary boundary.
func newUSACountryWorld(data *usaCountryData, namespace IdentityNamespace, seed uint64) (*usaCountryWorld, error) {

	if data == nil || data.base == nil || data.phones == nil || data.socioeconomic == nil ||
		data.socioeconomic.model == nil || !data.socioeconomic.model.initialized {
		return nil, fmt.Errorf("complete admitted USA country data is required")
	}
	b, d, m := &data.baseDocument, &data.socioeconomic.document, data.socioeconomic.model
	if b.Country != "US" || d.Country != "US" || b.AnchorDate != d.AnchorDate ||
		b.Geography.TerritorySystem != d.Dimensions.Territory.SemanticID ||
		d.IncomeUnit.Currency != "USD" || m.numeric.territories != socioeconomicUSADivisions {
		return nil, fmt.Errorf("incompatible USA base, socioeconomic country, currency or geography")
	}
	projection, err := m.canonicalProjection()
	if err != nil {
		return nil, err
	}
	if sha256.Sum256(projection) != m.digest {
		return nil, fmt.Errorf("USA compiled model identity differs from its content")
	}
	base, err := newUSALayer1Harness(namespace, seed, data.base, data.phones)
	if err != nil {
		return nil, err
	}

	// Views retain only their own nodes. In particular, the frozen Company
	// sampler's uninitialized Italian facade never receives USD Income laws.
	n := &m.numeric
	companyModel := &usaCompanyEmployerCandidate{
		numeric: socioeconomicNumericModel{
			territories: n.territories, companyCount: n.companyCount,
			companyIndustry: n.companyIndustry, employeeCount: n.employeeCount, employer: n.employer,
		},
		interpretation: m.interpretation, fingerprints: m.fingerprints, identity: m.digest,
	}
	companies, err := newUSACompanyEmployerHarness(base, companyModel)
	if err != nil {
		return nil, err
	}
	personal := &usaPersonalCategoricalCandidate{identity: m.digest,
		tables: [6]socioeconomicWeights{n.education, n.primaryActivity, n.employeeJobFamily,
			n.employeeJobLevel, n.selfEmployedJobFamily, n.selfEmployedIndustry}}
	copy(personal.ages[:], n.ages[:3])
	for i, node := range usaPersonalCategoricalNodes {
		personal.fingerprints[i] = m.fingerprints[node]
	}
	categorical, err := newUSAPersonalCategoricalHarness(base, personal, companies)
	if err != nil {
		return nil, err
	}
	income := &usaIncomeCandidate{
		numeric:  socioeconomicNumericModel{territories: n.territories, ages: n.ages, income: n.income},
		identity: m.digest,
	}
	copy(income.fingerprints[:], m.fingerprints[socioeconomicIncomeEmployee:])

	// Compose technical identities with the existing KFD1 codecs. They never
	// become stream roots; the inherited factory remains world-v1 plus seed.
	var components []Component
	for _, field := range [][2]string{
		{marketComponentName, "usa"}, {marketDataVersionComponentName, b.Release},
		{personSpecVersionComponentName, "person-us-v1"}, {worldModelVersionComponentName, WorldModelVersion},
		{"generation-protocol", "generation-us-v1"}, {"contact-protocol", "us-2026a-contact-v1"},
		{"address-protocol", "us-2026a-address-v1"},
	} {
		component, err := StringComponent(field[0], field[1])
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	seedComponent, err := Uint64Component(worldSeedComponentName, seed)
	if err != nil {
		return nil, err
	}
	baseComponent, err := BytesComponent("base-data", data.base.digest[:])
	if err != nil {
		return nil, err
	}
	phoneComponent, err := BytesComponent("phone-rule", data.phones.identity[:])
	if err != nil {
		return nil, err
	}
	baseID, err := ComposeWorldID(namespace, append(components, seedComponent, baseComponent, phoneComponent)...)
	if err != nil {
		return nil, err
	}
	id, err := socioeconomicWorldIdentityForProtocol(baseID, m.digest, d.AnchorDate, usaSocioeconomicProtocol)
	if err != nil {
		return nil, err
	}

	return &usaCountryWorld{data: data, sampler: usaIncomeHarness{categorical, income}, baseID: baseID, id: id}, nil
}

func (w *usaCountryWorld) person(index PersonIndex) (usaCountryPerson, error) {

	if w == nil || w.id == "" || w.data == nil {
		return usaCountryPerson{}, fmt.Errorf("initialized internal USA country World is required")
	}
	record, err := w.sampler.person(index)
	if err != nil {
		return usaCountryPerson{}, err
	}
	c := &record.categorical
	person := c.layer1.person
	if w.catalog != nil {
		person.PhotoID, err = w.catalog.selectPhoto(w.sampler.categorical.base.streams, index,
			person.Gender, c.layer1.anchorAge)
		if err != nil {
			return usaCountryPerson{}, err
		}
	}

	return usaCountryPerson{
		Schema: "usa-internal-country-diagnostic-person-v1", Coordinate: index, Country: "US", Currency: "USD",
		Person: person, AnchorAge: c.layer1.anchorAge, LocalityID: c.layer1.localityID,
		Division: c.layer1.division, Education: c.education, PrimaryActivity: c.activity,
		Employee: c.employee, SelfEmployed: c.selfEmployed, EmployerCompany: c.company, AnnualIncomeUSD: record.income,
	}, nil
}
