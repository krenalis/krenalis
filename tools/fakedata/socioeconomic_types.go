// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

// AnnualIncomeEUR is gross personal monetary Income annualized at the anchor
// in whole EUR.
type AnnualIncomeEUR uint32

// CompanyIndex is a one-based coordinate for a synthetic Company.
type CompanyIndex uint64

// PrimaryActivity identifies the primary activity at the anchor.
type PrimaryActivity string

// Supported primary activities at PopulationAnchorDate.
const (
	PrimaryActivityEmployee     PrimaryActivity = "employee"
	PrimaryActivitySelfEmployed PrimaryActivity = "self-employed"
	PrimaryActivityUnemployed   PrimaryActivity = "unemployed"
	PrimaryActivityStudent      PrimaryActivity = "student"
	PrimaryActivityRetired      PrimaryActivity = "retired"
	PrimaryActivityInactive     PrimaryActivity = "inactive"
)

// EducationLevel identifies highest completed attainment at the anchor.
type EducationLevel string

// Supported completed educational attainment levels.
const (
	EducationLevelBelowPrimary             EducationLevel = "below-primary"
	EducationLevelPrimary                  EducationLevel = "primary"
	EducationLevelLowerSecondary           EducationLevel = "lower-secondary"
	EducationLevelUpperSecondary           EducationLevel = "upper-secondary"
	EducationLevelPostSecondaryNonTertiary EducationLevel = "post-secondary-non-tertiary"
	EducationLevelTertiary                 EducationLevel = "tertiary"
	EducationLevelDoctoral                 EducationLevel = "doctoral"
)

// Industry identifies primary economic activity at the anchor.
type Industry string

// Supported primary economic activity groups.
const (
	IndustryAgriculture                     Industry = "agriculture"
	IndustryProductionAndUtilities          Industry = "production-and-utilities"
	IndustryConstruction                    Industry = "construction"
	IndustryCommerce                        Industry = "commerce"
	IndustryTransportAndLogistics           Industry = "transport-and-logistics"
	IndustryHospitality                     Industry = "hospitality"
	IndustryInformationAndCommunication     Industry = "information-and-communication"
	IndustryFinanceAndRealEstate            Industry = "finance-and-real-estate"
	IndustryProfessionalAndBusinessServices Industry = "professional-and-business-services"
	IndustryPublicAdministrationAndSafety   Industry = "public-administration-and-safety"
	IndustryEducation                       Industry = "education"
	IndustryHealthAndSocialCare             Industry = "health-and-social-care"
	IndustryArtsAndRecreation               Industry = "arts-and-recreation"
	IndustryOtherServices                   Industry = "other-services"
)

// JobFamily identifies the primary occupational function at the anchor.
type JobFamily string

// Supported primary occupational functions.
const (
	JobFamilyProductionAndFieldwork     JobFamily = "production-and-fieldwork"
	JobFamilySkilledTrades              JobFamily = "skilled-trades"
	JobFamilyTransportAndLogistics      JobFamily = "transport-and-logistics"
	JobFamilyCustomerAndPersonalService JobFamily = "customer-and-personal-service"
	JobFamilyAdministration             JobFamily = "administration"
	JobFamilySalesAndCommercial         JobFamily = "sales-and-commercial"
	JobFamilyTechnicalAndScientific     JobFamily = "technical-and-scientific"
	JobFamilyProfessionalAndCreative    JobFamily = "professional-and-creative"
	JobFamilyCareAndEducation           JobFamily = "care-and-education"
	JobFamilyProtectiveServices         JobFamily = "protective-services"
)

// JobLevel identifies employee organizational responsibility at the anchor.
type JobLevel string

// Supported employee organizational responsibility tiers.
const (
	JobLevelIndividualContributor JobLevel = "individual-contributor"
	JobLevelSupervisor            JobLevel = "supervisor"
	JobLevelManager               JobLevel = "manager"
	JobLevelExecutive             JobLevel = "executive"
)

// CompanySizeBand identifies the modeled workforce interval at the anchor.
type CompanySizeBand string

// Supported workforce bands derived solely from EmployeeCount.
const (
	CompanySizeBandZero   CompanySizeBand = "zero"
	CompanySizeBandMicro  CompanySizeBand = "micro"
	CompanySizeBandSmall  CompanySizeBand = "small"
	CompanySizeBandMedium CompanySizeBand = "medium"
	CompanySizeBandLarge  CompanySizeBand = "large"
)

// Company contains the generated facts for an operational Company.
type Company struct {
	ID            string
	EmployeeCount uint32
	Industry      Industry
	SizeBand      CompanySizeBand
}

// EmployeeFacts contains current employee-specific facts at the anchor.
type EmployeeFacts struct {
	Employer  CompanyIndex
	JobFamily JobFamily
	JobLevel  JobLevel
}

// SelfEmployedFacts contains self-employment facts at the anchor.
type SelfEmployedFacts struct {
	Industry  Industry
	JobFamily JobFamily
}

// SocioeconomicFacts contains socioeconomic state at PopulationAnchorDate.
type SocioeconomicFacts struct {
	Education       EducationLevel
	PrimaryActivity PrimaryActivity
	Employee        *EmployeeFacts
	SelfEmployed    *SelfEmployedFacts
	Income          AnnualIncomeEUR
}

// SocioeconomicPerson combines a Layer 1 Person with anchor-date facts.
type SocioeconomicPerson struct {
	Person Person
	Facts  SocioeconomicFacts
}

// SocioeconomicWorldID identifies a stable socioeconomic configuration.
type SocioeconomicWorldID string
