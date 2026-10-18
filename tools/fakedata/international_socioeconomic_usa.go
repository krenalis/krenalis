// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"context"
	"fmt"
	"io"
	"slices"
)

const (
	usaSocioeconomicAnchorDate = "2026-01-01"
	usaSocioeconomicProfile    = "us-2026a-socioeconomic-v1"
	usaSocioeconomicProtocol   = "socioeconomic-us-v1"
)

// usaSocioeconomicData admits only the US civilian noninstitutionalized 21–75
// profile; it has no public World, acquisition or application entry point.
type usaSocioeconomicData struct {
	document internationalSocioeconomic
	model    *usaSocioeconomicModel
}

func (d *usaSocioeconomicData) canonicalJSON() ([]byte, error) {
	if d == nil || d.model == nil || !d.model.initialized {
		return nil, fmt.Errorf("uninitialized USA socioeconomic document")
	}
	return internationalCanonical(d.document)
}

func (d *usaSocioeconomicData) logicalDigest() ([32]byte, error) {
	if d == nil || d.model == nil || !d.model.initialized {
		return [32]byte{}, fmt.Errorf("uninitialized USA socioeconomic document")
	}
	return internationalDigest("fakedata/socioeconomic-data/json-v1", d.document)
}

// decodeUSASocioeconomicData uses the closed v1 decoder without widening the
// Italian entry; callers must arrange interruption of blocking reads.
func decodeUSASocioeconomicData(ctx context.Context, reader io.Reader) (*usaSocioeconomicData, error) {

	var document internationalSocioeconomic
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	model, err := prepareUSASocioeconomic(&document)
	if err != nil {
		return nil, err
	}

	return &usaSocioeconomicData{document: document, model: model}, nil
}

// prepareUSASocioeconomic binds the national interpretation before compiling
// nine-territory input and checking its complete numerical projection.
func prepareUSASocioeconomic(d *internationalSocioeconomic) (*usaSocioeconomicModel, error) {

	// Validate the national profile, annualized USD and optional provenance.
	if d == nil || d.Schema != "sw-socioeconomic-data-v1" || d.Profile != usaSocioeconomicProfile ||
		d.Country != "US" || d.AnchorDate != usaSocioeconomicAnchorDate || !internationalToken(d.Release) {
		return nil, fmt.Errorf("unsupported USA socioeconomic metadata")
	}
	if d.AgeDomain != (internationalAgeDomain{21, 75}) || d.CompanyCount != 65536 {
		return nil, fmt.Errorf("invalid USA age domain or CompanyCount")
	}
	if d.IncomeUnit != (internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"}) {
		return nil, fmt.Errorf("unsupported USA Income unit")
	}
	if source := d.SourceModel; source != nil {
		if !internationalToken(source.Schema) || !internationalSHA256(source.ByteSHA256) ||
			source.Schema == "SW-L2-D2B2-logical-v0" || source.ByteSHA256 == italianSourceModelSHA256 {
			return nil, fmt.Errorf("invalid USA sourceModel or Italian provenance claim")
		}
	}

	// Validate ordered taxonomies, branch meanings and exact size thresholds.
	expected := usaInternationalDimensions()
	for _, pair := range [][2]internationalTaxonomy{
		{d.Dimensions.Education, expected.Education}, {d.Dimensions.Industry, expected.Industry},
		{d.Dimensions.JobFamily, expected.JobFamily}, {d.Dimensions.JobLevel, expected.JobLevel},
		{d.Dimensions.Territory, expected.Territory},
	} {
		err := validateInternationalTaxonomy(pair[0], pair[1])
		if err != nil {
			return nil, err
		}
	}
	activity, size := d.Dimensions.Activity, d.Dimensions.SizeBand
	if activity.SemanticID != expected.Activity.SemanticID || len(activity.Categories) != 6 ||
		size.SemanticID != expected.SizeBand.SemanticID || len(size.Categories) != 5 {
		return nil, fmt.Errorf("unsupported activity or size taxonomy")
	}
	for i, category := range activity.Categories {
		if category.Code != expected.Activity.Categories[i].Code || category.Branch != internationalIncomeNames[i] ||
			category.Definition != nil && !internationalText(*category.Definition) {
			return nil, fmt.Errorf("invalid activity category %d", i)
		}
	}
	for i, category := range size.Categories {
		if category.Code != expected.SizeBand.Categories[i].Code ||
			category.MinimumEmployeeCount != expected.SizeBand.Categories[i].MinimumEmployeeCount ||
			category.Definition != nil && !internationalText(*category.Definition) {
			return nil, fmt.Errorf("invalid size category %d", i)
		}
	}

	// Reconstruct the eight independently owned age maps.
	input := &socioeconomicModelJSON{companyCount: d.CompanyCount}
	bounds := usaInternationalPartitionBounds()
	for i, partition := range d.AgePartitions.ordered() {
		if !slices.Equal(partition.LowerBounds, bounds[i]) {
			return nil, fmt.Errorf("invalid USA age partition %d", i)
		}
		input.ages[i].lowerBounds = slices.Clone(partition.LowerBounds)
		input.ages[i].compiled = make([]uint32, 55)
		bucket := 0
		for offset := range 55 {
			if bucket+1 < len(partition.LowerBounds) && uint32(offset+21) >= partition.LowerBounds[bucket+1] {
				bucket++
			}
			input.ages[i].compiled[offset] = uint32(bucket)
		}
	}

	// Validate the closed axis registry and nine-Division categorical grids.
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	lengths := []int{14, 378, 3024, 630, 1680, 630, 140}
	outputs := []*[]uint32{
		&input.companyIndustry, &input.education, &input.primaryActivity, &input.employeeJobFamily,
		&input.employeeJobLevel, &input.selfEmployedJobFamily, &input.selfEmployedIndustry,
	}
	for i, categorical := range d.Models.categorical() {
		if !slices.Equal(categorical.Axes, axes[i]) || categorical.Outcome != outcomes[i] ||
			len(categorical.Weights) != lengths[i] {
			return nil, fmt.Errorf("invalid categorical axes, outcome or length at model %d", i)
		}
		*outputs[i] = categorical.Weights
	}
	if !slices.Equal(d.Models.EmployeeCount.Axes, []string{"industry"}) || len(d.Models.EmployeeCount.Laws) != 14 {
		return nil, fmt.Errorf("invalid EmployeeCount axes or law count")
	}
	for _, law := range d.Models.EmployeeCount.Laws {
		if len(law.Values) > 128 || len(law.Weights) > 128 {
			return nil, fmt.Errorf("EmployeeCount law exceeds 128 points")
		}
		input.employeeCount = append(input.employeeCount, socioeconomicLawJSON{values: law.Values, weights: law.Weights})
	}

	// Validate Employer factors before narrowing; USA does not inherit Italy's cap.
	employer := d.Models.Employer
	if employer.WorkforceCap == 0 || employer.WorkforceCap > 65535 ||
		!slices.Equal(employer.IndustryFactors.Axes, []string{"jobFamily", "industry"}) ||
		!slices.Equal(employer.SizeFactors.Axes, []string{"jobLevel", "positiveSizeBand"}) {
		return nil, fmt.Errorf("invalid USA Employer cap or axes")
	}
	input.employer.cap = uint16(employer.WorkforceCap)
	for i, factors := range [][]uint32{employer.IndustryFactors.Values, employer.SizeFactors.Values} {
		out := input.employer.industryFactors[:]
		if i == 1 {
			out = input.employer.sizeFactors[:]
		}
		if len(factors) != len(out) {
			return nil, fmt.Errorf("invalid Employer factor count")
		}
		for j, factor := range factors {
			if factor == 0 || factor > 4095 {
				return nil, fmt.Errorf("Employer factor outside 1–4095")
			}
			out[j] = uint16(factor)
		}
	}

	// Translate only structural nulls; preserve every applicable branch-local ID.
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for branch, income := range d.Models.Income.ordered() {
		slots := [...]int{25200, 7560, 378, 378, 378, 378}
		maxLaws := [...]int{512, 256, 64, 64, 64, 64}
		if !slices.Equal(income.Axes, incomeAxes[branch]) || len(income.TupleMap) != slots[branch] ||
			len(income.Laws) == 0 || len(income.Laws) > maxLaws[branch] {
			return nil, fmt.Errorf("invalid Income branch %d axes or dimensions", branch)
		}
		out := &input.income[branch]
		out.tupleMap = make([]uint32, len(income.TupleMap))
		for slot, id := range income.TupleMap {
			inapplicable := branch == 0 && slot%5 == 0
			if (id == nil) != inapplicable {
				return nil, fmt.Errorf("invalid Income null at branch %d slot %d", branch, slot)
			}
			if inapplicable {
				out.tupleMap[slot] = 65535
				continue
			}
			if *id >= uint32(len(income.Laws)) {
				return nil, fmt.Errorf("invalid Income law reference at branch %d slot %d", branch, slot)
			}
			out.tupleMap[slot] = *id
		}
		for _, law := range income.Laws {
			out.laws = append(out.laws, socioeconomicLawJSON{values: law.Values, weights: law.Weights})
		}
	}

	numeric, err := compileSocioeconomicNumericModel(input, socioeconomicUSADivisions)
	if err != nil {
		return nil, err
	}
	interpretation := usaSocioeconomicInterpretation{
		Country: d.Country, Profile: d.Profile, AnchorDate: d.AnchorDate, AgeDomain: d.AgeDomain,
		Dimensions: expected, IncomeUnit: d.IncomeUnit,
	}
	projection, err := internationalCanonical(usaSocioeconomicProjection{
		usaSocioeconomicInterpretation: interpretation,
		AgePartitions:                  d.AgePartitions, CompanyCount: d.CompanyCount, Models: d.Models,
	})
	if err != nil {
		return nil, err
	}
	model := &usaSocioeconomicModel{numeric: *numeric, interpretation: interpretation, protocol: usaSocioeconomicProtocol}
	err = model.prepareIdentity(projection)
	if err != nil {
		return nil, err
	}
	model.initialized = true

	return model, nil
}

func usaInternationalDimensions() internationalDimensions {

	d := internationalDimensions{
		Education: internationalTaxonomyFor("sw-l2-education-v1", socioeconomicEducationLevels[:]),
		Industry:  internationalTaxonomyFor("sw-l2-industry-v1", socioeconomicIndustries[:]),
		JobFamily: internationalTaxonomyFor("sw-l2-job-family-v1", socioeconomicJobFamilies[:]),
		JobLevel:  internationalTaxonomyFor("sw-l2-job-level-v1", socioeconomicJobLevels[:]),
		Territory: internationalTaxonomyFor("sw-us-2026a-census-division-v1", []string{
			"new-england", "middle-atlantic", "east-north-central", "west-north-central", "south-atlantic",
			"east-south-central", "west-south-central", "mountain", "pacific",
		}),
		Activity: internationalActivityTaxonomy{SemanticID: "sw-l2-primary-activity-v1"},
		SizeBand: internationalSizeTaxonomy{SemanticID: "sw-l2-company-size-band-v1"},
	}
	for i, code := range socioeconomicActivities {
		d.Activity.Categories = append(d.Activity.Categories, internationalActivityCategory{
			Code: string(code), Branch: internationalIncomeNames[i],
		})
	}
	for i, code := range socioeconomicSizeBands {
		minimum := [...]uint32{0, 1, 10, 50, 250}
		d.SizeBand.Categories = append(d.SizeBand.Categories, internationalSizeCategory{
			Code: string(code), MinimumEmployeeCount: minimum[i],
		})
	}

	return d
}

func usaInternationalPartitionBounds() [8][]uint32 {
	return [8][]uint32{
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70},
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 65},
		{21, 25, 35, 45, 55, 65}, {21, 25, 30, 40, 55, 65},
		{21, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65},
	}
}
