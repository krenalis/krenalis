// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"slices"
)

// These are the exact historical bytes adopted by the coordinated owner entry;
// candidate metadata in those bytes deliberately remains unchanged.
var usaCompanyEmployerPins = [...]struct {
	name   string
	bytes  int
	sha256 string
}{
	{"manifest.json", 10077, "6fdd57dc3f180530f035eec9569d1fa346d057119b224f946f91d3261434b76a"},
	{"configuration.json", 1795, "5c7b69f3a7a18fcf0538957c14160e63da54db9df28b3980817797610b2a79ab"},
	{"provenance.json", 1343, "7e7d45d57999b764c1fdec2f8d28b600b2dafd2b46180257e76419de2041e400"},
	{"assets/company-industry-integer.json", 1323, "d55bfd64eb147f2df42b706af186066f6fff1d7da8078a92ec7a38364debb751"},
	{"components/company-count-laws.json", 26791, "e004b8af8bb713449245b9f7a816bac9c1b122fe718c9f7f5b7c34b023684d51"},
	{"assets/employer-factors.json", 60635, "103422f377a223acc48486d2e0b85c812d7744e8378fe215cfed8269c67503b4"},
	{"assets/employer-analytical-integer.json", 60575, "7b55f609e07b9cf565b5fdf7e1e8f28ebff1fb5ac5d7fd6e8fafcd02724e70b0"},
}

// usaCompanyEmployerAssets supplies only the seven pinned historical files, in
// registry order; it is not an international socioeconomic.json document.
type usaCompanyEmployerAssets [7][]byte

// usaCompanyEmployerCandidate holds three compiled nodes and an admission
// guard, either from pinned components or an admitted complete USA model.
type usaCompanyEmployerCandidate struct {
	numeric        socioeconomicNumericModel
	interpretation usaSocioeconomicInterpretation
	fingerprints   [15][sha256.Size]byte
	identity       [sha256.Size]byte
}

// compileUSACompanyEmployer compiles only the three admitted nodes; it neither
// invokes the full socioeconomic compiler nor supplies missing Income laws.
func compileUSACompanyEmployer(input *socioeconomicModelJSON) (*usaCompanyEmployerCandidate, error) {

	if input == nil || input.companyCount != 65536 || len(input.companyIndustry) != 14 ||
		len(input.employeeCount) != 14 || input.employer.cap != 65535 {
		return nil, fmt.Errorf("invalid USA Company/Employer dimensions or cap")
	}
	for _, weight := range input.companyIndustry {
		if weight == 0 {
			return nil, fmt.Errorf("USA Company Industry must have positive weight")
		}
	}
	model := &usaCompanyEmployerCandidate{
		numeric: socioeconomicNumericModel{
			territories: socioeconomicUSADivisions, companyCount: input.companyCount, employer: input.employer,
			companyIndustry: socioeconomicWeights{width: 14, cumulative: make([]uint64, 14)},
			employeeCount:   make([]socioeconomicLaw, 14),
		},
		interpretation: usaSocioeconomicInterpretation{
			Country: "US", Profile: usaSocioeconomicProfile, AnchorDate: usaSocioeconomicAnchorDate,
			AgeDomain: internationalAgeDomain{21, 75}, Dimensions: usaInternationalDimensions(),
			IncomeUnit: internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"},
		},
	}
	err := compileSocioeconomicWeights(input.companyIndustry, model.numeric.companyIndustry.cumulative)
	if err != nil {
		return nil, err
	}
	for i, law := range input.employeeCount {
		if len(law.values) != 128 || len(law.weights) != 128 || law.values[0] == 0 || law.values[127] != 32768 {
			return nil, fmt.Errorf("invalid USA productive EmployeeCount support")
		}
		compiled, err := compileSocioeconomicLaw(law)
		if err != nil {
			return nil, err
		}
		if i > 0 && (!slices.Equal(compiled.values, model.numeric.employeeCount[0].values) ||
			!slices.Equal(compiled.cumulative, model.numeric.employeeCount[0].cumulative)) {
			return nil, fmt.Errorf("USA EmployeeCount laws must share the adopted positive prior")
		}
		model.numeric.employeeCount[i] = compiled
	}
	for _, factor := range input.employer.industryFactors {
		if factor == 0 || factor > 4095 {
			return nil, fmt.Errorf("USA Employer Industry factor outside 1–4095")
		}
	}
	for _, factor := range input.employer.sizeFactors {
		if factor != 64 {
			return nil, fmt.Errorf("USA Employer SizeBand factor must be 64")
		}
	}

	// These three USA paths equal the frozen paths. Reuse their logical subtree
	// encoders and KFD1 fingerprint procedure without a whole-model stream salt.
	for _, node := range []int{socioeconomicCompanyIndustry, socioeconomicCompanyEmployeeCount, socioeconomicEmployer} {
		asset, err := model.numeric.appendAsset(nil, node)
		if err != nil {
			return nil, err
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			return nil, err
		}
		model.fingerprints[node], err = canonicalDigest("fakedata/"+usaSocioeconomicPaths[node]+"/model",
			[]Component{component})
		if err != nil {
			return nil, err
		}
	}

	return model, nil
}

// prepareUSACompanyEmployerCandidate binds every metadata and numerical byte
// before typed projection. Only this entry assigns an admission identity.
func prepareUSACompanyEmployerCandidate(assets usaCompanyEmployerAssets) (*usaCompanyEmployerCandidate, error) {

	components := make([]Component, len(assets))
	for i, pin := range usaCompanyEmployerPins {
		if len(assets[i]) != pin.bytes {
			return nil, fmt.Errorf("USA Company/Employer asset %d has incompatible byte count", i)
		}
		digest := sha256.Sum256(assets[i])
		if fmt.Sprintf("%x", digest) != pin.sha256 {
			return nil, fmt.Errorf("USA Company/Employer asset %d has incompatible digest", i)
		}
		component, err := BytesComponent(fmt.Sprintf("asset-%d", i), digest[:])
		if err != nil {
			return nil, err
		}
		components[i] = component
	}

	// Metadata omitted from these projections is still pinned in its entirety.
	var industry struct {
		Categories []Industry `json:"categories"`
		Weights    []uint32   `json:"weights"`
	}
	err := json.Unmarshal(assets[3], &industry)
	if err != nil {
		return nil, err
	}
	if !slices.Equal(industry.Categories, socioeconomicIndustries[:]) {
		return nil, fmt.Errorf("invalid USA Company Industry order")
	}
	var counts struct {
		CompanyCount       uint64             `json:"companyCount"`
		CompiledProjection []internationalLaw `json:"compiledProjection"`
		Laws               []struct {
			Industry Industry `json:"industry"`
		} `json:"laws"`
	}
	err = json.Unmarshal(assets[4], &counts)
	if err != nil {
		return nil, err
	}
	if len(counts.Laws) != 14 {
		return nil, fmt.Errorf("invalid USA count law coverage")
	}
	for i, law := range counts.Laws {
		if law.Industry != socioeconomicIndustries[i] {
			return nil, fmt.Errorf("invalid USA count law Industry order")
		}
	}
	var factors struct {
		Cap             uint32     `json:"cap"`
		IndustryFactors [][]uint32 `json:"industryFactors"`
		SizeFactors     [][]uint32 `json:"sizeFactors"`
	}
	err = json.Unmarshal(assets[5], &factors)
	if err != nil {
		return nil, err
	}
	if factors.Cap != 65535 || len(factors.IndustryFactors) != 10 || len(factors.SizeFactors) != 4 {
		return nil, fmt.Errorf("invalid USA Employer factor dimensions")
	}
	input := &socioeconomicModelJSON{companyCount: counts.CompanyCount, companyIndustry: industry.Weights}
	input.employer.cap = uint16(factors.Cap)
	for _, law := range counts.CompiledProjection {
		input.employeeCount = append(input.employeeCount, socioeconomicLawJSON{values: law.Values, weights: law.Weights})
	}
	for i, row := range factors.IndustryFactors {
		if len(row) != 14 {
			return nil, fmt.Errorf("invalid USA Employer Industry factor count")
		}
		for j, factor := range row {
			if factor == 0 || factor > 4095 {
				return nil, fmt.Errorf("USA Employer Industry factor outside 1–4095")
			}
			input.employer.industryFactors[i*14+j] = uint16(factor)
		}
	}
	for i, row := range factors.SizeFactors {
		if len(row) != 4 {
			return nil, fmt.Errorf("invalid USA Employer SizeBand factor count")
		}
		for j, factor := range row {
			if factor != 64 {
				return nil, fmt.Errorf("USA Employer SizeBand factor must be 64")
			}
			input.employer.sizeFactors[i*4+j] = uint16(factor)
		}
	}
	model, err := compileUSACompanyEmployer(input)
	if err != nil {
		return nil, err
	}
	model.identity, err = canonicalDigest("fakedata/socioeconomic/us-v1/company-employer/admission", components)
	if err != nil {
		return nil, err
	}

	return model, nil
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
