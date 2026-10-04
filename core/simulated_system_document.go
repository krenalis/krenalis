// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package core

import "github.com/krenalis/krenalis/tools/fakedata"

type simulatedSystemDocument struct {
	CustomerID string                  `json:"customer_id"`
	FirstName  *string                 `json:"first_name,omitempty"`
	LastName   *string                 `json:"last_name,omitempty"`
	Email      *string                 `json:"email,omitempty"`
	Gender     fakedata.Gender         `json:"gender"`
	BirthDate  string                  `json:"birth_date"`
	CurrentAge int                     `json:"current_age"`
	Phone      *string                 `json:"phone,omitempty"`
	SiteURL    string                  `json:"site_url"`
	Address    *simulatedSystemAddress `json:"address,omitempty"`
	Photo      *string                 `json:"photo,omitempty"`
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
