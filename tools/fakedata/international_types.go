// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

// These transport types describe v1, independently of the positional numerical
// compiler input. Collection limits belong to the Italian consumer, not to the
// international format. Optional fields use pointers to preserve absence.
type internationalBase struct {
	Schema      string                    `json:"schema"`
	Release     string                    `json:"release"`
	Profile     string                    `json:"profile"`
	Country     string                    `json:"country"`
	AnchorDate  string                    `json:"anchorDate"`
	Genders     internationalTaxonomy     `json:"genders"`
	Ages        []internationalAgeWeight  `json:"ages" max:"55"`
	Cohorts     []internationalCohort     `json:"cohorts" max:"5"`
	FirstNames  []internationalFirstNames `json:"firstNames" max:"10"`
	LastNames   []internationalName       `json:"lastNames" max:"600"`
	Geography   internationalGeography    `json:"geography"`
	StreetNames internationalStreets      `json:"streetNames"`
}

type internationalAgeWeight struct {
	Age    uint32 `json:"age"`
	Weight uint64 `json:"weight"`
}

type internationalCohort struct {
	ID       string `json:"id"`
	FromYear uint32 `json:"fromYear"`
	ToYear   uint32 `json:"toYear"`
}

type internationalFirstNames struct {
	Gender  string              `json:"gender"`
	Cohort  string              `json:"cohort"`
	Entries []internationalName `json:"entries" max:"80"`
}

type internationalName struct {
	Display   string  `json:"display"`
	ASCIIAtom *string `json:"asciiAtom,omitempty"`
	Weight    uint64  `json:"weight"`
}

type internationalGeography struct {
	PlaceCodeSystem  string               `json:"placeCodeSystem"`
	TerritorySystem  string               `json:"territorySystem"`
	PostalCodeSystem string               `json:"postalCodeSystem"`
	Places           []internationalPlace `json:"places" max:"7896"`
	PostalCodes      map[string][]string  `json:"postalCodes" max:"7896"`
}

type internationalPlace struct {
	ID             string                        `json:"id"`
	Name           string                        `json:"name"`
	Population     uint64                        `json:"population"`
	Administrative []internationalAdministrative `json:"administrative" max:"1"`
	Territory      string                        `json:"territory"`
}

type internationalAdministrative struct {
	System string  `json:"system"`
	Code   *string `json:"code,omitempty"`
	Name   *string `json:"name,omitempty"`
}

type internationalStreets struct {
	Kind    string                `json:"kind"`
	Entries []internationalStreet `json:"entries" max:"400"`
}

type internationalStreet struct {
	Text   string `json:"text"`
	Weight uint64 `json:"weight"`
}

type internationalTaxonomy struct {
	SemanticID string                  `json:"semanticId"`
	Categories []internationalCategory `json:"categories" max:"20"`
}

type internationalCategory struct {
	Code       string  `json:"code"`
	Definition *string `json:"definition,omitempty"`
}

type internationalActivityTaxonomy struct {
	SemanticID string                          `json:"semanticId"`
	Categories []internationalActivityCategory `json:"categories" max:"6"`
}

type internationalActivityCategory struct {
	Code       string  `json:"code"`
	Definition *string `json:"definition,omitempty"`
	Branch     string  `json:"branch"`
}

type internationalSizeTaxonomy struct {
	SemanticID string                      `json:"semanticId"`
	Categories []internationalSizeCategory `json:"categories" max:"5"`
}

type internationalSizeCategory struct {
	Code                 string  `json:"code"`
	Definition           *string `json:"definition,omitempty"`
	MinimumEmployeeCount uint32  `json:"minimumEmployeeCount"`
}

type internationalSocioeconomic struct {
	Schema        string                     `json:"schema"`
	Release       string                     `json:"release"`
	Profile       string                     `json:"profile"`
	Country       string                     `json:"country"`
	AnchorDate    string                     `json:"anchorDate"`
	SourceModel   *internationalSourceModel  `json:"sourceModel,omitempty"`
	AgeDomain     internationalAgeDomain     `json:"ageDomain"`
	Dimensions    internationalDimensions    `json:"dimensions"`
	AgePartitions internationalAgePartitions `json:"agePartitions"`
	CompanyCount  uint64                     `json:"companyCount"`
	IncomeUnit    internationalIncomeUnit    `json:"incomeUnit"`
	Models        internationalModels        `json:"models"`
}

type internationalSourceModel struct {
	Schema     string `json:"schema"`
	ByteSHA256 string `json:"byteSha256"`
}

type internationalAgeDomain struct {
	Minimum uint32 `json:"minimum"`
	Maximum uint32 `json:"maximum"`
}

type internationalDimensions struct {
	Education internationalTaxonomy         `json:"education"`
	Activity  internationalActivityTaxonomy `json:"activity"`
	Industry  internationalTaxonomy         `json:"industry"`
	JobFamily internationalTaxonomy         `json:"jobFamily"`
	JobLevel  internationalTaxonomy         `json:"jobLevel"`
	Territory internationalTaxonomy         `json:"territory"`
	SizeBand  internationalSizeTaxonomy     `json:"sizeBand"`
}

type internationalAgePartitions struct {
	Education          internationalPartition `json:"education"`
	Activity           internationalPartition `json:"activity"`
	EmployeeLevel      internationalPartition `json:"employeeLevel"`
	SelfEmployedIncome internationalPartition `json:"selfEmployedIncome"`
	UnemployedIncome   internationalPartition `json:"unemployedIncome"`
	StudentIncome      internationalPartition `json:"studentIncome"`
	RetiredIncome      internationalPartition `json:"retiredIncome"`
	InactiveIncome     internationalPartition `json:"inactiveIncome"`
}

func (p *internationalAgePartitions) ordered() [8]*internationalPartition {
	return [8]*internationalPartition{
		&p.Education, &p.Activity, &p.EmployeeLevel, &p.SelfEmployedIncome,
		&p.UnemployedIncome, &p.StudentIncome, &p.RetiredIncome, &p.InactiveIncome,
	}
}

type internationalPartition struct {
	LowerBounds []uint32 `json:"lowerBounds" max:"8"`
}

type internationalIncomeUnit struct {
	Currency      string `json:"currency"`
	DecimalPlaces uint32 `json:"decimalPlaces"`
	Period        string `json:"period"`
	Basis         string `json:"basis"`
}

type internationalModels struct {
	CompanyIndustry       internationalCategorical `json:"companyIndustry"`
	EmployeeCount         internationalCountLaws   `json:"employeeCount"`
	Education             internationalCategorical `json:"education"`
	PrimaryActivity       internationalCategorical `json:"primaryActivity"`
	EmployeeJobFamily     internationalCategorical `json:"employeeJobFamily"`
	EmployeeJobLevel      internationalCategorical `json:"employeeJobLevel"`
	SelfEmployedJobFamily internationalCategorical `json:"selfEmployedJobFamily"`
	SelfEmployedIndustry  internationalCategorical `json:"selfEmployedIndustry"`
	Employer              internationalEmployer    `json:"employer"`
	Income                internationalIncome      `json:"income"`
}

func (m *internationalModels) categorical() [7]*internationalCategorical {
	return [7]*internationalCategorical{
		&m.CompanyIndustry, &m.Education, &m.PrimaryActivity, &m.EmployeeJobFamily,
		&m.EmployeeJobLevel, &m.SelfEmployedJobFamily, &m.SelfEmployedIndustry,
	}
}

type internationalCategorical struct {
	Axes    []string `json:"axes" max:"3"`
	Outcome string   `json:"outcome"`
	Weights []uint32 `json:"weights" max:"6720"`
}

type internationalCountLaws struct {
	Axes []string           `json:"axes" max:"1"`
	Laws []internationalLaw `json:"laws" max:"14"`
}

type internationalLaw struct {
	Values  []uint32 `json:"values" max:"512"`
	Weights []uint32 `json:"weights" max:"512"`
}

type internationalFactorMatrix struct {
	Axes   []string `json:"axes" max:"2"`
	Values []uint32 `json:"values" max:"140"`
}

type internationalEmployer struct {
	WorkforceCap    uint32                    `json:"workforceCap"`
	IndustryFactors internationalFactorMatrix `json:"industryFactors"`
	SizeFactors     internationalFactorMatrix `json:"sizeFactors"`
}

type internationalIncome struct {
	Employee     internationalIncomeBranch `json:"employee"`
	SelfEmployed internationalIncomeBranch `json:"selfEmployed"`
	Unemployed   internationalIncomeBranch `json:"unemployed"`
	Student      internationalIncomeBranch `json:"student"`
	Retired      internationalIncomeBranch `json:"retired"`
	Inactive     internationalIncomeBranch `json:"inactive"`
}

func (m *internationalIncome) ordered() [6]*internationalIncomeBranch {
	return [6]*internationalIncomeBranch{
		&m.Employee, &m.SelfEmployed, &m.Unemployed, &m.Student, &m.Retired, &m.Inactive,
	}
}

type internationalIncomeBranch struct {
	Axes     []string           `json:"axes" max:"5"`
	TupleMap []*uint32          `json:"tupleMap" max:"56000" nullable:"true"`
	Laws     []internationalLaw `json:"laws" max:"512"`
}
