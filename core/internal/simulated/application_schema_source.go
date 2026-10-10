// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package simulated

import (
	"github.com/krenalis/krenalis/tools/fakedata"
	"github.com/krenalis/krenalis/tools/types"
)

var sourceSchema = types.Object([]types.Property{
	{Name: "customer_id", Type: types.String()},
	{Name: "first_name", Type: types.String()},
	{Name: "last_name", Type: types.String()},
	{Name: "email", Type: types.String().AsEmail(), ReadOptional: true},
	{Name: "gender", Type: types.String().WithValues("female", "male")},
	{Name: "birth_date", Type: types.Date()},
	{Name: "current_age", Type: types.Int(8)},
	{Name: "phone", Type: types.String().AsPhone()},
	{Name: "site_url", Type: types.String().AsURL()},
	{Name: "address", Type: types.Object([]types.Property{
		{Name: "street", Type: types.String()},
		{Name: "municipality", Type: types.String()},
		{Name: "state_prov", Type: types.String()},
		{Name: "postal_code", Type: types.String()},
		{Name: "country", Type: types.String().AsCountry(types.ISO3166Alpha2)},
	})},
	{Name: "photo", Type: types.String().AsURL()},
	{Name: "education_level", ReadOptional: true, Type: types.String().WithValues(
		string(fakedata.EducationLevelBelowPrimary), string(fakedata.EducationLevelPrimary),
		string(fakedata.EducationLevelLowerSecondary), string(fakedata.EducationLevelUpperSecondary),
		string(fakedata.EducationLevelPostSecondaryNonTertiary), string(fakedata.EducationLevelTertiary),
		string(fakedata.EducationLevelDoctoral),
	)},
	{Name: "primary_activity", ReadOptional: true, Type: types.String().WithValues(
		string(fakedata.PrimaryActivityEmployee), string(fakedata.PrimaryActivitySelfEmployed),
		string(fakedata.PrimaryActivityUnemployed), string(fakedata.PrimaryActivityStudent),
		string(fakedata.PrimaryActivityRetired), string(fakedata.PrimaryActivityInactive),
	)},
	{Name: "annual_income_eur", ReadOptional: true, Type: types.Int(32).Unsigned()},
	{Name: "job_family", ReadOptional: true, Type: types.String().WithValues(
		string(fakedata.JobFamilyProductionAndFieldwork), string(fakedata.JobFamilySkilledTrades),
		string(fakedata.JobFamilyTransportAndLogistics), string(fakedata.JobFamilyCustomerAndPersonalService),
		string(fakedata.JobFamilyAdministration), string(fakedata.JobFamilySalesAndCommercial),
		string(fakedata.JobFamilyTechnicalAndScientific), string(fakedata.JobFamilyProfessionalAndCreative),
		string(fakedata.JobFamilyCareAndEducation), string(fakedata.JobFamilyProtectiveServices),
	)},
	{Name: "industry", ReadOptional: true, Type: types.String().WithValues(
		string(fakedata.IndustryAgriculture), string(fakedata.IndustryProductionAndUtilities),
		string(fakedata.IndustryConstruction), string(fakedata.IndustryCommerce),
		string(fakedata.IndustryTransportAndLogistics), string(fakedata.IndustryHospitality),
		string(fakedata.IndustryInformationAndCommunication), string(fakedata.IndustryFinanceAndRealEstate),
		string(fakedata.IndustryProfessionalAndBusinessServices),
		string(fakedata.IndustryPublicAdministrationAndSafety),
		string(fakedata.IndustryEducation), string(fakedata.IndustryHealthAndSocialCare),
		string(fakedata.IndustryArtsAndRecreation), string(fakedata.IndustryOtherServices),
	)},
	{Name: "job_level", ReadOptional: true, Type: types.String().WithValues(
		string(fakedata.JobLevelIndividualContributor), string(fakedata.JobLevelSupervisor),
		string(fakedata.JobLevelManager), string(fakedata.JobLevelExecutive),
	)},
})
