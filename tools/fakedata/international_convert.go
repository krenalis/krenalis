// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"io/fs"
	"reflect"
	"slices"
)

// ConvertItalianBaseData converts the six original it-2026a CSV files in source
// to international v1. It requires their frozen byte hashes and never reads
// embedded data. The returned release is it-2026a-json1.
// It returns file-system errors and context errors from reading source.
func ConvertItalianBaseData(ctx context.Context, source fs.FS) (*ItalianBaseData, error) {

	if ctx == nil || source == nil {
		return nil, fmt.Errorf("context and CSV source are required")
	}
	names := []string{
		"age_weights.csv", "first_names.csv", "last_names.csv", "places.csv", "postal_codes.csv", "street_names.csv",
	}
	hashes := []string{
		ageWeightsSHA256, firstNamesSHA256, lastNamesSHA256, placesSHA256, postalCodesSHA256, streetNamesSHA256,
	}
	headers := [][]string{
		{"age", "weight"}, {"gender", "cohort", "name", "ascii_atom", "weight"}, {"name", "ascii_atom", "weight"},
		{"istat_code", "city", "state_prov", "region", "population"},
		{"istat_code", "postal_codes"}, {"name", "weight"},
	}
	var records [6][][]string
	for i, name := range names {
		file, err := source.Open(name)
		if err != nil {
			return nil, err
		}
		reader := &internationalReader{ctx: ctx, reader: file}
		data, err := io.ReadAll(io.LimitReader(reader, maxInternationalDocumentBytes+1))
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > maxInternationalDocumentBytes || fmt.Sprintf("%x", sha256.Sum256(data)) != hashes[i] {
			return nil, fmt.Errorf("original %s checksum mismatch", name)
		}
		rows, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 || !slices.Equal(rows[0], headers[i]) {
			return nil, fmt.Errorf("invalid original CSV header")
		}
		records[i] = rows[1:]
	}

	ages, _, err := parseAgeRows(records[0])
	if err != nil {
		return nil, err
	}
	firstNames, _, err := parseFirstNameRows(records[1])
	if err != nil {
		return nil, err
	}
	lastNames, _, err := parseNameRows("last_names.csv", records[2], 600, true)
	if err != nil {
		return nil, err
	}
	places, err := parsePlaceRows(records[3])
	if err != nil {
		return nil, err
	}
	postal, err := parsePostalRows(records[4], places)
	if err != nil {
		return nil, err
	}
	streets, _, err := parseNameRows("street_names.csv", records[5], 400, false)
	if err != nil {
		return nil, err
	}

	d := internationalBase{
		Schema: "sw-base-data-v1", Release: "it-2026a-json1", Profile: "it-2026a-base-v1",
		Country: "IT", AnchorDate: PopulationAnchorDate,
		Genders: internationalTaxonomyFor("sw-l1-gender-v1", []string{"female", "male"}),
		Cohorts: italianInternationalCohorts(),
		Geography: internationalGeography{
			PlaceCodeSystem: "it-istat-municipality-it-2026a", TerritorySystem: "sw-it-2026a-region-v1",
			PostalCodeSystem: "it-postal-observations-it-2026a", PostalCodes: postal,
		},
		StreetNames: internationalStreets{Kind: "fragment"},
	}
	for _, row := range ages {
		d.Ages = append(d.Ages, internationalAgeWeight{Age: uint32(row.age), Weight: row.weight})
	}
	for _, gender := range []Gender{GenderFemale, GenderMale} {
		for _, cohort := range d.Cohorts {
			group := internationalFirstNames{Gender: string(gender), Cohort: cohort.ID}
			for _, row := range firstNames[firstNameGroup{gender: gender, cohort: cohort.ID}] {
				atom := row.asciiAtom
				group.Entries = append(group.Entries, internationalName{
					Display: row.display, ASCIIAtom: &atom, Weight: row.weight,
				})
			}
			d.FirstNames = append(d.FirstNames, group)
		}
	}
	for _, row := range lastNames {
		atom := row.asciiAtom
		d.LastNames = append(d.LastNames, internationalName{Display: row.display, ASCIIAtom: &atom, Weight: row.weight})
	}
	for _, row := range places {
		province := row.stateProv
		d.Geography.Places = append(d.Geography.Places, internationalPlace{
			ID: row.istatCode, Name: row.municipality, Population: row.population, Territory: row.region,
			Administrative: []internationalAdministrative{{System: "it-provincial-uts-it-2026a", Code: &province}},
		})
	}
	for _, row := range streets {
		entry := internationalStreet{Text: row.display, Weight: row.weight}
		d.StreetNames.Entries = append(d.StreetNames.Entries, entry)
	}
	err = validateItalianBase(&d)
	if err != nil {
		return nil, err
	}

	return &ItalianBaseData{document: d, initialized: true}, nil
}

// ConvertItalianSocioeconomicData converts the original canonical logical-v0
// asset, requiring its frozen SHA-256, to international v1, release
// it-2026a-json1. It verifies the typed projection and frozen canonical
// bytes with the existing compiler. It returns context errors from reading.
func ConvertItalianSocioeconomicData(ctx context.Context, reader io.Reader) (*ItalianSocioeconomicData, error) {

	if ctx == nil || reader == nil {
		return nil, fmt.Errorf("context and canonical source are required")
	}
	source := &internationalReader{ctx: ctx, reader: reader}
	data, err := io.ReadAll(io.LimitReader(source, maxSocioeconomicModelBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxSocioeconomicModelBytes || fmt.Sprintf("%x", sha256.Sum256(data)) != italianSourceModelSHA256 {
		return nil, fmt.Errorf("original socioeconomic model checksum mismatch")
	}
	original, err := decodeSocioeconomicModelJSON(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	_, err = compileSocioeconomicModel(original)
	if err != nil {
		return nil, err
	}

	d := internationalSocioeconomic{
		Schema: "sw-socioeconomic-data-v1", Release: "it-2026a-json1", Profile: "it-2026a-socioeconomic-v1",
		Country: "IT", AnchorDate: PopulationAnchorDate,
		SourceModel: &internationalSourceModel{"SW-L2-D2B2-logical-v0", italianSourceModelSHA256},
		AgeDomain:   internationalAgeDomain{21, 75}, Dimensions: italianInternationalDimensions(),
		CompanyCount: original.companyCount,
		IncomeUnit:   internationalIncomeUnit{"EUR", 0, "year", "gross-personal-monetary-annualized"},
	}
	for i, partition := range d.AgePartitions.ordered() {
		partition.LowerBounds = original.ages[i].lowerBounds
	}
	weights := [][]uint32{
		original.companyIndustry, original.education, original.primaryActivity, original.employeeJobFamily,
		original.employeeJobLevel, original.selfEmployedJobFamily, original.selfEmployedIndustry,
	}
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	for i, categorical := range d.Models.categorical() {
		*categorical = internationalCategorical{Axes: axes[i], Outcome: outcomes[i], Weights: weights[i]}
	}
	d.Models.EmployeeCount.Axes = []string{"industry"}
	for _, law := range original.employeeCount {
		entry := internationalLaw{Values: law.values, Weights: law.weights}
		d.Models.EmployeeCount.Laws = append(d.Models.EmployeeCount.Laws, entry)
	}
	d.Models.Employer.WorkforceCap = uint32(original.employer.cap)
	d.Models.Employer.IndustryFactors.Axes = []string{"jobFamily", "industry"}
	for _, factor := range original.employer.industryFactors {
		d.Models.Employer.IndustryFactors.Values = append(d.Models.Employer.IndustryFactors.Values, uint32(factor))
	}
	d.Models.Employer.SizeFactors.Axes = []string{"jobLevel", "positiveSizeBand"}
	for _, factor := range original.employer.sizeFactors {
		d.Models.Employer.SizeFactors.Values = append(d.Models.Employer.SizeFactors.Values, uint32(factor))
	}
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for branch, income := range d.Models.Income.ordered() {
		income.Axes = incomeAxes[branch]
		income.TupleMap = make([]*uint32, len(original.income[branch].tupleMap))
		for slot, id := range original.income[branch].tupleMap {
			if branch == 0 && slot%5 == 0 {
				continue
			}
			income.TupleMap[slot] = &id
		}
		for _, law := range original.income[branch].laws {
			income.Laws = append(income.Laws, internationalLaw{Values: law.values, Weights: law.weights})
		}
	}

	projected, err := projectItalianSocioeconomic(&d)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(projected, original) {
		return nil, fmt.Errorf("international projection differs from original numerical input")
	}
	model, err := compileSocioeconomicModel(projected)
	if err != nil {
		return nil, err
	}
	canonical, err := model.canonicalModel()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, canonical) {
		return nil, fmt.Errorf("reconstructed canonical model differs from original")
	}

	return &ItalianSocioeconomicData{document: d, model: model}, nil
}
