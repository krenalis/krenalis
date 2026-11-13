// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"fmt"
	"math"
	"slices"
)

// usaCompanyEmployerCandidate holds three compiled nodes and an admission
// guard, either from pinned components or an admitted complete USA model.
type usaCompanyEmployerCandidate struct {
	numeric        socioeconomicNumericModel
	interpretation usaSocioeconomicInterpretation
	fingerprints   [15][sha256.Size]byte
	identity       [sha256.Size]byte
}

// usaCompanyEmployerHarness borrows the existing CAT, POINT and EMP methods.
// Its sampler is deliberately not initialized as a public SocioeconomicWorld
// or SocioeconomicModel, and cannot pass either Italian public constructor.
type usaCompanyEmployerHarness struct {
	sampler SocioeconomicWorld
	model   *usaCompanyEmployerCandidate
}

// newUSACompanyEmployerHarness inherits the actual USA base's namespace and
// stream factory; all work and allocations are bounded by the adopted model.
func newUSACompanyEmployerHarness(base *usaLayer1Harness, model *usaCompanyEmployerCandidate) (*usaCompanyEmployerHarness, error) {

	if base == nil || base.base == nil || base.base.digest == ([32]byte{}) || base.streams == (streamFactory{}) ||
		model == nil || model.identity == ([32]byte{}) || model.numeric.companyCount != 65536 ||
		model.numeric.territories != socioeconomicUSADivisions {
		return nil, fmt.Errorf("prepared USA base and Company/Employer candidate are required")
	}
	err := base.namespace.validate()
	if err != nil {
		return nil, err
	}

	// Reserve complete Contract-2 volumes before any roster/index allocation.
	// Dense ascending input needs no sorting copy; both slices have exact size.
	// Extra descriptors and Industry bytes are charged separately. The 8 MiB
	// reservation covers the bounded model and its acquisition representations.
	n := model.numeric.companyCount
	indexBytes := uint64(64 + 32*40 + 16*40*n)
	additionalBytes := uint64(64 + n + 16*40)
	retainedBytes := 64 + 4*n + indexBytes + additionalBytes
	peakBytes := retainedBytes + 8*n + 16*n + (8 << 20) + 4096
	if indexBytes > 64<<20 || retainedBytes > 72<<20 || peakBytes > 128<<20 {
		return nil, fmt.Errorf("USA Company/Employer resource bounds exceeded")
	}

	w := &usaCompanyEmployerHarness{
		model: model,
		sampler: SocioeconomicWorld{
			base: &World{namespace: base.namespace, streams: base.streams},
			model: &SocioeconomicModel{
				socioeconomicNumericModel: model.numeric, fingerprints: model.fingerprints,
			},
			companyCounts: make([]uint32, n), companyIndustries: make([]uint8, n),
		},
	}
	for offset := range w.sampler.companyCounts {
		company, err := w.company(CompanyIndex(offset + 1))
		if err != nil {
			return nil, err
		}
		w.sampler.companyCounts[offset] = company.EmployeeCount
		w.sampler.companyIndustries[offset] = uint8(slices.Index(socioeconomicIndustries[:], company.Industry))
	}

	// Assemble the frozen raw integer intervals, without GCD or normalization.
	// Zero remains a valid technical count and is omitted from every index.
	for condition := range w.sampler.employers {

		index := &w.sampler.employers[condition]
		positive := 0
		for _, count := range w.sampler.companyCounts {
			if count > 0 {
				positive++
			}
		}

		index.companies = make([]CompanyIndex, 0, positive)
		index.cumulative = make([]uint64, 0, positive)
		var total uint64
		for offset, count := range w.sampler.companyCounts {

			if count == 0 {
				continue
			}

			industry := int(w.sampler.companyIndustries[offset])
			band := socioeconomicSizeBand(count)
			weight := min(uint64(count), uint64(model.numeric.employer.cap))
			for _, factor := range [...]uint16{
				model.numeric.employer.industryFactors[(condition/4)*14+industry],
				model.numeric.employer.sizeFactors[(condition%4)*4+band-1],
			} {
				if factor == 0 || uint64(factor) > math.MaxUint64/weight {
					return nil, fmt.Errorf("invalid USA Employer factor or weight overflow")
				}
				weight *= uint64(factor)
			}
			if weight > math.MaxUint64-total {
				return nil, fmt.Errorf("USA Employer total overflow")
			}
			total += weight
			index.companies = append(index.companies, CompanyIndex(offset+1))
			index.cumulative = append(index.cumulative, total)

		}
		if total == 0 {
			return nil, fmt.Errorf("USA Employer condition %d has zero total", condition)
		}

	}

	return w, nil
}

// company regenerates a Company at its raw coordinate through the existing
// sampler; no predecessor, Person, Income or mutable roster state is read.
func (w *usaCompanyEmployerHarness) company(index CompanyIndex) (Company, error) {

	if w == nil || w.model == nil || w.model.identity == ([32]byte{}) {
		return Company{}, fmt.Errorf("uninitialized USA Company/Employer harness")
	}
	if index == 0 {
		return Company{}, ErrInvalidIndex
	}
	if uint64(index) > w.model.numeric.companyCount {
		return Company{}, ErrCompanyIndexOutOfRange
	}

	industry, err := w.sampler.category(uint64(index), socioeconomicCompanyIndustry, w.model.numeric.companyIndustry, 0)
	if err != nil {
		return Company{}, err
	}
	count, err := w.sampler.point(uint64(index), socioeconomicCompanyEmployeeCount,
		w.model.numeric.employeeCount[industry],
		socioeconomicParent{"industry", industry})
	if err != nil {
		return Company{}, err
	}
	id, err := entityIDForIndex(w.sampler.base.namespace, "company", uint64(index))
	if err != nil {
		return Company{}, err
	}

	return Company{
		ID: id, EmployeeCount: count, Industry: socioeconomicIndustries[industry],
		SizeBand: socioeconomicSizeBands[socioeconomicSizeBand(count)],
	}, nil
}

// employee exercises the Employer node only; it returns the frozen relational
// payload, never a complete Person or an independently sampled Industry.
func (w *usaCompanyEmployerHarness) employee(index PersonIndex, family JobFamily, level JobLevel) (EmployeeFacts, error) {

	if w == nil || w.model == nil || w.model.identity == ([32]byte{}) {
		return EmployeeFacts{}, fmt.Errorf("uninitialized USA Company/Employer harness")
	}
	if index == 0 {
		return EmployeeFacts{}, ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return EmployeeFacts{}, ErrLayer1IndexOutOfRange
	}
	f := slices.Index(socioeconomicJobFamilies[:], family)
	l := slices.Index(socioeconomicJobLevels[:], level)
	if f < 0 || l < 0 {
		return EmployeeFacts{}, fmt.Errorf("invalid USA Employer condition")
	}

	employers := &w.sampler.employers[4*f+l]
	position, err := w.sampler.sample(uint64(index), socioeconomicEmployer, employers.cumulative,
		socioeconomicParent{"job-family", f}, socioeconomicParent{"job-level", l})
	if err != nil {
		return EmployeeFacts{}, err
	}
	if position >= len(employers.companies) {
		return EmployeeFacts{}, fmt.Errorf("invalid USA Employer interval")
	}
	employer := employers.companies[position]
	if employer == 0 || uint64(employer) > w.model.numeric.companyCount || w.sampler.companyCounts[employer-1] == 0 {
		return EmployeeFacts{}, fmt.Errorf("invalid USA Employer coordinate or eligibility")
	}

	return EmployeeFacts{Employer: employer, JobFamily: family, JobLevel: level}, nil
}
