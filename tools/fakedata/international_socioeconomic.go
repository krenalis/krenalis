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

const italianSourceModelSHA256 = "f049103241e6e43ffb5dd4a1247b64b2d0ca1bff787950f25284b5b435fd21a8"

var internationalIncomeNames = [...]string{"employee", "selfEmployed", "unemployed", "student", "retired", "inactive"}

// ItalianSocioeconomicData is a validated immutable international v1 document
// and its directly projected, compiled Italian numerical model. Its zero value
// is invalid. It performs no acquisition or World construction.
type ItalianSocioeconomicData struct {
	document internationalSocioeconomic
	model    *SocioeconomicModel
}

// DecodeItalianSocioeconomicData decodes the Italian v1 profile from reader.
// It returns a context error when cancellation is observed between reads;
// callers must arrange interruption of any blocking read themselves.
// Malformed JSON tokens return a tools/json SyntaxError.
func DecodeItalianSocioeconomicData(ctx context.Context, reader io.Reader) (*ItalianSocioeconomicData, error) {
	return decodeItalianSocioeconomicData(ctx, reader)
}

// CanonicalJSON returns the document's C1 serialization with its final LF.
func (d *ItalianSocioeconomicData) CanonicalJSON() ([]byte, error) {
	if d == nil || d.model == nil {
		return nil, fmt.Errorf("uninitialized Italian socioeconomic document")
	}
	return internationalCanonical(d.document)
}

// LogicalDigest returns the international document's KFD1/C1 digest.
func (d *ItalianSocioeconomicData) LogicalDigest() ([32]byte, error) {
	if d == nil || d.model == nil {
		return [32]byte{}, fmt.Errorf("uninitialized Italian socioeconomic document")
	}
	return internationalDigest("fakedata/socioeconomic-data/json-v1", d.document)
}

// Model returns the immutable numerical model compiled by the frozen compiler.
func (d *ItalianSocioeconomicData) Model() (*SocioeconomicModel, error) {
	if d == nil || d.model == nil {
		return nil, fmt.Errorf("uninitialized Italian socioeconomic document")
	}
	return d.model, nil
}

func decodeItalianSocioeconomicData(ctx context.Context, reader io.Reader) (*ItalianSocioeconomicData, error) {
	var document internationalSocioeconomic
	err := decodeInternationalJSON(ctx, reader, &document)
	if err != nil {
		return nil, err
	}
	input, err := projectItalianSocioeconomic(&document)
	if err != nil {
		return nil, err
	}
	model, err := compileSocioeconomicModel(input)
	if err != nil {
		return nil, err
	}
	return &ItalianSocioeconomicData{document: document, model: model}, nil
}

func internationalTaxonomyFor[T ~string](semanticID string, codes []T) internationalTaxonomy {
	taxonomy := internationalTaxonomy{SemanticID: semanticID, Categories: make([]internationalCategory, len(codes))}
	for i, code := range codes {
		taxonomy.Categories[i].Code = string(code)
	}
	return taxonomy
}

func italianInternationalDimensions() internationalDimensions {
	d := internationalDimensions{
		Education: internationalTaxonomyFor("sw-l2-education-v1", socioeconomicEducationLevels[:]),
		Industry:  internationalTaxonomyFor("sw-l2-industry-v1", socioeconomicIndustries[:]),
		JobFamily: internationalTaxonomyFor("sw-l2-job-family-v1", socioeconomicJobFamilies[:]),
		JobLevel:  internationalTaxonomyFor("sw-l2-job-level-v1", socioeconomicJobLevels[:]),
		Territory: internationalTaxonomyFor("sw-it-2026a-region-v1", socioeconomicRegions[:]),
		Activity:  internationalActivityTaxonomy{SemanticID: "sw-l2-primary-activity-v1"},
		SizeBand:  internationalSizeTaxonomy{SemanticID: "sw-l2-company-size-band-v1"},
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

func italianInternationalPartitionBounds() [8][]uint32 {
	return [8][]uint32{
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70},
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 65},
		{21, 25, 35, 45, 55, 65}, {21, 25, 30, 40, 55, 65},
		{21, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65},
	}
}

// projectItalianSocioeconomic checks the binding and all decoder preconditions
// before narrowing or invoking the existing compiler. No old JSON is emitted.
func projectItalianSocioeconomic(d *internationalSocioeconomic) (*socioeconomicModelJSON, error) {

	if d.Schema != "sw-socioeconomic-data-v1" || d.Profile != "it-2026a-socioeconomic-v1" ||
		d.Country != "IT" || d.AnchorDate != PopulationAnchorDate || !internationalToken(d.Release) {
		return nil, fmt.Errorf("unsupported Italian socioeconomic metadata")
	}
	if d.SourceModel == nil || d.SourceModel.Schema != "SW-L2-D2B2-logical-v0" ||
		d.SourceModel.ByteSHA256 != italianSourceModelSHA256 {
		return nil, fmt.Errorf("invalid Italian sourceModel")
	}
	if d.AgeDomain.Minimum != 21 || d.AgeDomain.Maximum != 75 || d.CompanyCount != 65536 {
		return nil, fmt.Errorf("invalid Italian age domain or CompanyCount")
	}
	if d.IncomeUnit != (internationalIncomeUnit{"EUR", 0, "year", "gross-personal-monetary-annualized"}) {
		return nil, fmt.Errorf("unsupported Italian Income unit")
	}

	// Validate taxonomy identities, codes, ordinal order, branches and thresholds.
	expected := italianInternationalDimensions()
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

	// Reconstruct every compiled age position from its independently owned bounds.
	input := &socioeconomicModelJSON{companyCount: d.CompanyCount}
	bounds := italianInternationalPartitionBounds()
	for i, partition := range d.AgePartitions.ordered() {
		if !slices.Equal(partition.LowerBounds, bounds[i]) {
			return nil, fmt.Errorf("invalid Italian age partition %d", i)
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

	// Validate the closed axis registry and dense categorical dimensions.
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	lengths := []int{14, 840, 6720, 1400, 1680, 1400, 140}
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
		entry := socioeconomicLawJSON{values: law.Values, weights: law.Weights}
		input.employeeCount = append(input.employeeCount, entry)
	}

	// Check factors in their wide representation before uint16 conversion.
	employer := d.Models.Employer
	if employer.WorkforceCap != 4096 ||
		!slices.Equal(employer.IndustryFactors.Axes, []string{"jobFamily", "industry"}) ||
		!slices.Equal(employer.SizeFactors.Axes, []string{"jobLevel", "positiveSizeBand"}) {
		return nil, fmt.Errorf("invalid Italian Employer cap or axes")
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

	// Preserve IDs and map only structurally inapplicable Employee slots to 65535.
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for branch, income := range d.Models.Income.ordered() {
		slots := [...]int{56000, 16800, 840, 840, 840, 840}
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
			if len(law.Values) > 512 || len(law.Weights) > 512 {
				return nil, fmt.Errorf("Income law exceeds 512 points")
			}
			out.laws = append(out.laws, socioeconomicLawJSON{values: law.Values, weights: law.Weights})
		}
	}

	return input, nil
}

func validateInternationalTaxonomy(actual, expected internationalTaxonomy) error {
	if actual.SemanticID != expected.SemanticID || len(actual.Categories) != len(expected.Categories) {
		return fmt.Errorf("unsupported taxonomy binding")
	}
	for i, category := range actual.Categories {
		if category.Code != expected.Categories[i].Code ||
			category.Definition != nil && !internationalText(*category.Definition) {
			return fmt.Errorf("invalid taxonomy category %d", i)
		}
	}
	return nil
}
