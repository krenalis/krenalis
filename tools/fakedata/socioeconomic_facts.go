// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"slices"
)

// These arrays are the protocol's explicit ordinal orders, not lexical ranks.
var (
	socioeconomicEducationLevels = [...]EducationLevel{
		EducationLevelBelowPrimary, EducationLevelPrimary, EducationLevelLowerSecondary, EducationLevelUpperSecondary,
		EducationLevelPostSecondaryNonTertiary, EducationLevelTertiary, EducationLevelDoctoral,
	}
	socioeconomicActivities = [...]PrimaryActivity{
		PrimaryActivityEmployee, PrimaryActivitySelfEmployed, PrimaryActivityUnemployed,
		PrimaryActivityStudent, PrimaryActivityRetired, PrimaryActivityInactive,
	}
	socioeconomicIndustries = [...]Industry{
		IndustryAgriculture, IndustryProductionAndUtilities, IndustryConstruction, IndustryCommerce,
		IndustryTransportAndLogistics, IndustryHospitality, IndustryInformationAndCommunication,
		IndustryFinanceAndRealEstate, IndustryProfessionalAndBusinessServices, IndustryPublicAdministrationAndSafety,
		IndustryEducation, IndustryHealthAndSocialCare, IndustryArtsAndRecreation, IndustryOtherServices,
	}
	socioeconomicJobFamilies = [...]JobFamily{
		JobFamilyProductionAndFieldwork, JobFamilySkilledTrades, JobFamilyTransportAndLogistics,
		JobFamilyCustomerAndPersonalService, JobFamilyAdministration, JobFamilySalesAndCommercial,
		JobFamilyTechnicalAndScientific, JobFamilyProfessionalAndCreative, JobFamilyCareAndEducation,
		JobFamilyProtectiveServices,
	}
	socioeconomicJobLevels = [...]JobLevel{
		JobLevelIndividualContributor, JobLevelSupervisor, JobLevelManager, JobLevelExecutive,
	}
	socioeconomicSizeBands = [...]CompanySizeBand{
		CompanySizeBandZero, CompanySizeBandMicro, CompanySizeBandSmall, CompanySizeBandMedium, CompanySizeBandLarge,
	}
	socioeconomicRegions = [...]string{
		"Abruzzo", "Basilicata", "Calabria", "Campania", "Emilia-Romagna", "Friuli-Venezia Giulia", "Lazio", "Liguria",
		"Lombardia", "Marche", "Molise", "Piemonte", "Puglia", "Sardegna", "Sicilia", "Toscana",
		"Trentino-Alto Adige/Südtirol", "Umbria", "Valle d'Aosta/Vallée d'Aoste", "Veneto",
	}
)

func (w *SocioeconomicWorld) facts(index PersonIndex) (SocioeconomicFacts, error) {

	if index == 0 {
		return SocioeconomicFacts{}, ErrInvalidIndex
	}
	if index > Layer1MaxPersonIndex {
		return SocioeconomicFacts{}, ErrLayer1IndexOutOfRange
	}
	anchorAge, _, _, err := w.base.birth(index)
	if err != nil {
		return SocioeconomicFacts{}, err
	}
	if anchorAge < 21 || anchorAge > 75 {
		return SocioeconomicFacts{}, fmt.Errorf("invalid socioeconomic AnchorAge")
	}
	place, err := w.base.addressMunicipality(index)
	if err != nil {
		return SocioeconomicFacts{}, err
	}
	region := slices.Index(socioeconomicRegions[:], place.region)
	if region < 0 {
		return SocioeconomicFacts{}, fmt.Errorf("invalid socioeconomic Region")
	}

	coordinate := uint64(index)
	m := w.model
	age := int(m.ages[0][anchorAge-21])
	row, err := m.territoryRow(age, region)
	if err != nil {
		return SocioeconomicFacts{}, err
	}
	territories := int(m.territories)
	education, err := w.category(coordinate, socioeconomicEducation, m.education, row,
		socioeconomicParent{"age-bucket", age}, socioeconomicParent{"region", region})
	if err != nil {
		return SocioeconomicFacts{}, err
	}

	age = int(m.ages[1][anchorAge-21])
	activity, err := w.category(coordinate, socioeconomicPrimaryActivity, m.primaryActivity,
		(age*7+education)*territories+region,
		socioeconomicParent{"age-bucket", age}, socioeconomicParent{"education", education},
		socioeconomicParent{"region", region})
	if err != nil {
		return SocioeconomicFacts{}, err
	}
	facts := SocioeconomicFacts{
		Education: socioeconomicEducationLevels[education], PrimaryActivity: socioeconomicActivities[activity],
	}

	switch facts.PrimaryActivity {

	case PrimaryActivityEmployee:

		family, err := w.category(coordinate, socioeconomicEmployeeJobFamily, m.employeeJobFamily,
			education*territories+region,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", region})
		if err != nil {
			return SocioeconomicFacts{}, err
		}

		age = int(m.ages[2][anchorAge-21])
		level, err := w.category(coordinate, socioeconomicEmployeeJobLevel, m.employeeJobLevel,
			(age*7+education)*10+family,
			socioeconomicParent{"age-bucket", age}, socioeconomicParent{"education", education},
			socioeconomicParent{"job-family", family})
		if err != nil {
			return SocioeconomicFacts{}, err
		}

		employers := &w.employers[4*family+level]
		position, err := w.sample(coordinate, socioeconomicEmployer, employers.cumulative,
			socioeconomicParent{"job-family", family}, socioeconomicParent{"job-level", level})
		if err != nil {
			return SocioeconomicFacts{}, err
		}
		if position >= len(employers.companies) {
			return SocioeconomicFacts{}, fmt.Errorf("invalid socioeconomic Employer interval")
		}
		employer := employers.companies[position]
		if employer == 0 || uint64(employer) > m.companyCount {
			return SocioeconomicFacts{}, fmt.Errorf("invalid socioeconomic Employer coordinate")
		}
		industry := int(w.companyIndustries[employer-1])
		band := socioeconomicSizeBand(w.companyCounts[employer-1])
		facts.Employee = &EmployeeFacts{
			Employer: employer, JobFamily: socioeconomicJobFamilies[family], JobLevel: socioeconomicJobLevels[level],
		}

		tuple := (((region*10+family)*4+level)*14+industry)*5 + band
		facts.Income, err = w.income(coordinate, 0, tuple,
			socioeconomicParent{"region", region}, socioeconomicParent{"job-family", family},
			socioeconomicParent{"job-level", level}, socioeconomicParent{"industry", industry},
			socioeconomicParent{"size-band", band})
		if err != nil {
			return SocioeconomicFacts{}, err
		}

	case PrimaryActivitySelfEmployed:

		family, err := w.category(coordinate, socioeconomicSelfEmployedJobFamily, m.selfEmployedJobFamily,
			education*territories+region,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", region})
		if err != nil {
			return SocioeconomicFacts{}, err
		}
		industry, err := w.category(coordinate, socioeconomicSelfEmployedIndustry, m.selfEmployedIndustry, family,
			socioeconomicParent{"job-family", family})
		if err != nil {
			return SocioeconomicFacts{}, err
		}
		facts.SelfEmployed = &SelfEmployedFacts{
			Industry: socioeconomicIndustries[industry], JobFamily: socioeconomicJobFamilies[family],
		}

		age = int(m.ages[3][anchorAge-21])
		tuple := ((age*territories+region)*10+family)*14 + industry
		facts.Income, err = w.income(coordinate, 1, tuple,
			socioeconomicParent{"age-bucket", age}, socioeconomicParent{"region", region},
			socioeconomicParent{"job-family", family}, socioeconomicParent{"industry", industry})
		if err != nil {
			return SocioeconomicFacts{}, err
		}

	case PrimaryActivityUnemployed, PrimaryActivityStudent, PrimaryActivityRetired, PrimaryActivityInactive:
		age = int(m.ages[activity+2][anchorAge-21])
		tuple := (age*7+education)*territories + region
		facts.Income, err = w.income(coordinate, activity, tuple,
			socioeconomicParent{"age-bucket", age}, socioeconomicParent{"education", education},
			socioeconomicParent{"region", region})
		if err != nil {
			return SocioeconomicFacts{}, err
		}
	default:
		return SocioeconomicFacts{}, fmt.Errorf("invalid socioeconomic PrimaryActivity")

	}

	return facts, nil
}
