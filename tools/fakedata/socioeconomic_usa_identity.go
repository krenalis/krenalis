// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"fmt"
)

var usaSocioeconomicPaths = [...]string{
	"socioeconomic/v1/company/industry",
	"socioeconomic/v1/company/employee-count",
	"socioeconomic/us-v1/person/education",
	"socioeconomic/us-v1/person/primary-activity",
	"socioeconomic/us-v1/person/employee/job-family",
	"socioeconomic/v1/person/employee/job-level",
	"socioeconomic/v1/person/employee/employer",
	"socioeconomic/us-v1/person/self-employed/job-family",
	"socioeconomic/v1/person/self-employed/industry",
	"socioeconomic/us-v1/person/income/employee",
	"socioeconomic/us-v1/person/income/self-employed",
	"socioeconomic/us-v1/person/income/unemployed",
	"socioeconomic/us-v1/person/income/student",
	"socioeconomic/us-v1/person/income/retired",
	"socioeconomic/us-v1/person/income/inactive",
}

// annualIncomeUSD is nonnegative gross personal monetary Income annualized at
// the USA anchor, in whole nominal USD; it is not an application field.
type annualIncomeUSD uint32

// usaSocioeconomicInterpretation holds the closed national meanings, without
// editorial definitions, release, provenance, acquisition or photo identity.
type usaSocioeconomicInterpretation struct {
	Country    string                  `json:"country"`
	Profile    string                  `json:"profile"`
	AnchorDate string                  `json:"anchorDate"`
	AgeDomain  internationalAgeDomain  `json:"ageDomain"`
	Dimensions internationalDimensions `json:"dimensions"`
	IncomeUnit internationalIncomeUnit `json:"incomeUnit"`
}

// usaSocioeconomicModel owns compiled USA state and cannot be supplied to the
// Italian NewSocioeconomicWorld; its zero value is invalid.
type usaSocioeconomicModel struct {
	numeric        socioeconomicNumericModel
	interpretation usaSocioeconomicInterpretation
	protocol       string
	digest         [sha256.Size]byte
	fingerprints   [15][sha256.Size]byte
	initialized    bool
}

// canonicalProjection recovers the H.2 projection from compiled state and its
// prepared national interpretation, independently of the transport document.
func (m *usaSocioeconomicModel) canonicalProjection() ([]byte, error) {

	if m == nil || m.numeric.territories != socioeconomicUSADivisions || m.protocol != usaSocioeconomicProtocol {
		return nil, fmt.Errorf("invalid USA compiled interpretation")
	}
	p := usaSocioeconomicProjection{
		usaSocioeconomicInterpretation: m.interpretation, CompanyCount: m.numeric.companyCount,
	}
	for i, partition := range p.AgePartitions.ordered() {
		partition.LowerBounds = []uint32{21}
		for offset := 1; offset < 55; offset++ {
			if m.numeric.ages[i][offset] != m.numeric.ages[i][offset-1] {
				partition.LowerBounds = append(partition.LowerBounds, uint32(offset+21))
			}
		}
	}

	// Recover the exact categorical masses from the row-local cumulatives.
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	tables := []socioeconomicWeights{
		m.numeric.companyIndustry, m.numeric.education, m.numeric.primaryActivity, m.numeric.employeeJobFamily,
		m.numeric.employeeJobLevel, m.numeric.selfEmployedJobFamily, m.numeric.selfEmployedIndustry,
	}
	for i, categorical := range p.Models.categorical() {
		categorical.Axes = axes[i]
		categorical.Outcome = outcomes[i]
		categorical.Weights = make([]uint32, len(tables[i].cumulative))
		var previous uint64
		for j, value := range tables[i].cumulative {
			if j%tables[i].width == 0 {
				previous = 0
			}
			categorical.Weights[j] = uint32(value - previous)
			previous = value
		}
	}

	// Restore all seven law pools without changing supports or canonical IDs.
	p.Models.EmployeeCount.Axes = []string{"industry"}
	laws := [7]struct {
		input  []socioeconomicLaw
		output *[]internationalLaw
	}{{m.numeric.employeeCount, &p.Models.EmployeeCount.Laws}}
	for branch, income := range p.Models.Income.ordered() {
		laws[branch+1].input = m.numeric.income[branch].laws
		laws[branch+1].output = &income.Laws
	}
	for _, pool := range laws {
		*pool.output = make([]internationalLaw, len(pool.input))
		for i, law := range pool.input {
			out := &(*pool.output)[i]
			out.Values = law.values
			out.Weights = make([]uint32, len(law.cumulative))
			var previous uint64
			for j, value := range law.cumulative {
				out.Weights[j] = uint32(value - previous)
				previous = value
			}
		}
	}

	p.Models.Employer.WorkforceCap = uint32(m.numeric.employer.cap)
	p.Models.Employer.IndustryFactors.Axes = []string{"jobFamily", "industry"}
	p.Models.Employer.SizeFactors.Axes = []string{"jobLevel", "positiveSizeBand"}
	for i, factors := range [][]uint16{m.numeric.employer.industryFactors[:], m.numeric.employer.sizeFactors[:]} {
		out := &p.Models.Employer.IndustryFactors.Values
		if i == 1 {
			out = &p.Models.Employer.SizeFactors.Values
		}
		*out = make([]uint32, len(factors))
		for j, factor := range factors {
			(*out)[j] = uint32(factor)
		}
	}

	// Restore null only in the employee zero SizeBand, never as Income zero.
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for branch, income := range p.Models.Income.ordered() {
		income.Axes = incomeAxes[branch]
		input := m.numeric.income[branch].tupleMap
		ids := make([]uint32, len(input))
		income.TupleMap = make([]*uint32, len(input))
		for slot, id := range input {
			if branch == 0 && id == 65535 {
				continue
			}
			ids[slot] = uint32(id)
			income.TupleMap[slot] = &ids[slot]
		}
	}

	return internationalCanonical(p)
}

// incomeSupport interprets an admitted support as USD, without sampling or
// traversing the Italian monetary API.
func (m *usaSocioeconomicModel) incomeSupport(branch, law, point int) (annualIncomeUSD, error) {

	if m == nil || !m.initialized {
		return 0, fmt.Errorf("uninitialized USA socioeconomic model")
	}
	if branch < 0 || branch >= len(m.numeric.income) {
		return 0, fmt.Errorf("invalid USA Income branch")
	}
	pool := m.numeric.income[branch].laws
	if law < 0 || law >= len(pool) {
		return 0, fmt.Errorf("invalid USA Income law")
	}
	if point < 0 || point >= len(pool[law].values) {
		return 0, fmt.Errorf("invalid USA Income support position")
	}

	return annualIncomeUSD(pool[law].values[point]), nil
}

func (m *usaSocioeconomicModel) prepareIdentity(projection []byte) error {

	canonical, err := m.canonicalProjection()
	if err != nil {
		return err
	}
	if !bytes.Equal(canonical, projection) {
		return fmt.Errorf("inconsistent USA numerical projection and compiled interpretation")
	}
	digest := sha256.Sum256(canonical)
	var fingerprints [15][sha256.Size]byte
	for node, path := range usaSocioeconomicPaths {

		asset, err := m.numeric.appendAsset(nil, node)
		if err != nil {
			return err
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			return err
		}
		components := []Component{component}
		if age := socioeconomicNodeAges[node]; age >= 0 {
			component, err = BytesComponent("age-map", append(appendSocioeconomicAge(nil, m.numeric.ages[age]), '\n'))
			if err != nil {
				return err
			}
			components = append(components, component)
		}
		fingerprints[node], err = canonicalDigest("fakedata/"+path+"/model", components)
		if err != nil {
			return err
		}

	}
	m.digest = digest
	m.fingerprints = fingerprints

	return nil
}

// usaSocioeconomicProjection is exactly Spec-USA H.2, with the ordered semantic
// bindings and all eight partitions; it is not another transport schema.
type usaSocioeconomicProjection struct {
	usaSocioeconomicInterpretation
	AgePartitions internationalAgePartitions `json:"agePartitions"`
	CompanyCount  uint64                     `json:"companyCount"`
	Models        internationalModels        `json:"models"`
}
