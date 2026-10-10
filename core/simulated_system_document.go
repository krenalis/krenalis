// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import (
	"fmt"

	"github.com/krenalis/krenalis/tools/fakedata"
)

type simulatedSystemDocument struct {
	CustomerID      string                    `json:"customer_id"`
	FirstName       *string                   `json:"first_name,omitempty"`
	LastName        *string                   `json:"last_name,omitempty"`
	Email           *string                   `json:"email,omitempty"`
	Gender          fakedata.Gender           `json:"gender"`
	BirthDate       string                    `json:"birth_date"`
	CurrentAge      int                       `json:"current_age"`
	Phone           *string                   `json:"phone,omitempty"`
	SiteURL         string                    `json:"site_url"`
	Address         *simulatedSystemAddress   `json:"address,omitempty"`
	Photo           *string                   `json:"photo,omitempty"`
	EducationLevel  fakedata.EducationLevel   `json:"education_level,omitempty"`
	PrimaryActivity fakedata.PrimaryActivity  `json:"primary_activity,omitempty"`
	AnnualIncomeEUR *fakedata.AnnualIncomeEUR `json:"annual_income_eur,omitempty"`
	JobFamily       fakedata.JobFamily        `json:"job_family,omitempty"`
	Industry        fakedata.Industry         `json:"industry,omitempty"`
	JobLevel        fakedata.JobLevel         `json:"job_level,omitempty"`
}

type simulatedSystemAddress struct {
	Street       string  `json:"street,omitempty"`
	Municipality string  `json:"municipality,omitempty"`
	StateProv    string  `json:"state_prov,omitempty"`
	PostalCode   string  `json:"postal_code,omitempty"`
	Country      *string `json:"country,omitempty"`
}

func newSimulatedSystemDocument(person fakedata.CountryPerson, record fakedata.SourceRecord, photo *string) simulatedSystemDocument {
	address := simulatedSystemAddress{
		Street:       person.Person.Address.Street,
		Municipality: person.Person.Address.Municipality,
		StateProv:    person.Person.Address.StateProv,
		PostalCode:   person.Person.Address.PostalCode,
		Country:      record.Country,
	}
	document := simulatedSystemDocument{
		CustomerID: record.ID,
		FirstName:  record.FirstName,
		LastName:   record.LastName,
		Email:      record.Email,
		Gender:     person.Person.Gender,
		BirthDate:  person.Person.BirthDate,
		CurrentAge: person.Person.CurrentAge,
		Phone:      record.Phone,
		SiteURL:    person.Person.SiteURL,
		Photo:      photo,
	}
	if address.Street != "" || address.Municipality != "" || address.StateProv != "" ||
		address.PostalCode != "" || address.Country != nil {
		document.Address = &address
	}
	return document
}

// setSimulatedSystemSocioeconomicFacts projects anchor-date personal facts,
// resolving employee Industry through the selected Employer.
func setSimulatedSystemSocioeconomicFacts(document *simulatedSystemDocument, world *fakedata.SocioeconomicWorld, facts fakedata.SocioeconomicFacts) error {

	if facts.Education == "" {
		return fmt.Errorf("socioeconomic education is missing")
	}

	switch facts.PrimaryActivity {

	case fakedata.PrimaryActivityEmployee:
		if facts.Employee == nil || facts.SelfEmployed != nil ||
			facts.Employee.JobFamily == "" || facts.Employee.JobLevel == "" {
			return fmt.Errorf("invalid employee socioeconomic payload")
		}
		company, err := world.Company(facts.Employee.Employer)
		if err != nil {
			return err
		}
		document.JobFamily = facts.Employee.JobFamily
		document.Industry = company.Industry
		document.JobLevel = facts.Employee.JobLevel

	case fakedata.PrimaryActivitySelfEmployed:
		if facts.SelfEmployed == nil || facts.Employee != nil ||
			facts.SelfEmployed.JobFamily == "" || facts.SelfEmployed.Industry == "" {
			return fmt.Errorf("invalid self-employed socioeconomic payload")
		}
		document.JobFamily = facts.SelfEmployed.JobFamily
		document.Industry = facts.SelfEmployed.Industry

	case fakedata.PrimaryActivityUnemployed, fakedata.PrimaryActivityStudent,
		fakedata.PrimaryActivityRetired, fakedata.PrimaryActivityInactive:
		if facts.Employee != nil || facts.SelfEmployed != nil {
			return fmt.Errorf("unexpected socioeconomic occupational payload")
		}

	default:
		return fmt.Errorf("invalid socioeconomic primary activity")

	}

	document.EducationLevel = facts.Education
	document.PrimaryActivity = facts.PrimaryActivity
	document.AnnualIncomeEUR = &facts.Income

	return nil
}
