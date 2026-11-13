// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"math"
	"reflect"
	"slices"
	"testing"
)

// TestUSACountryRuntimeReject rejects incompatible data and unsafe coordinates.
func TestUSACountryRuntimeReject(t *testing.T) {

	data := readUSACountryData(t)
	for _, test := range []struct {
		name   string
		change func(*usaCountryData)
	}{
		{"IT-base-US-model", func(d *usaCountryData) { d.baseDocument.Country = "IT" }},
		{"US-base-IT-model", func(d *usaCountryData) { d.socioeconomic.document.Country = "IT" }},
		{"unknown-base-country", func(d *usaCountryData) { d.baseDocument.Country = "XX" }},
		{"EUR-model", func(d *usaCountryData) { d.socioeconomic.document.IncomeUnit.Currency = "EUR" }},
		{"missing-model", func(d *usaCountryData) { d.socioeconomic.model = nil }},
		{"missing-phone-rule", func(d *usaCountryData) { d.phones = nil }},
		{"missing-geography-binding", func(d *usaCountryData) { d.baseDocument.Geography.TerritorySystem = "" }},
		{"corrupt-compiled-model", func(d *usaCountryData) {
			m := *d.socioeconomic.model
			m.numeric.ages[0][0]++
			d.socioeconomic.model = &m
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := *data
			model := *data.socioeconomic
			changed.socioeconomic = &model
			test.change(&changed)
			_, err := newUSACountryWorld(&changed, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
			if err != nil {
				return
			}
			t.Fatal("expected invalid complete World rejection, got World")
		})
	}
	w, err := newUSACountryWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected real USA World, got %v", err)
	}
	for _, coordinate := range []PersonIndex{0, Layer1MaxPersonIndex + 1, PersonIndex(math.MaxUint64)} {
		person, err := w.person(coordinate)
		if err != nil {
			if !reflect.DeepEqual(person, usaCountryPerson{}) {
				t.Fatal("expected empty rejected person, got partial output")
			}
			continue
		}
		t.Fatal("expected invalid Person coordinate rejection, got record")
	}
	for _, coordinate := range []PersonIndex{1, Layer1MaxPersonIndex} {
		_, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected valid boundary %d, got %v", coordinate, err)
		}
	}
	companies := w.sampler.categorical.companies
	index := &companies.sampler.employers[0]
	index.companies = slices.Clone(index.companies)
	for i := range index.companies {
		index.companies[i] = 65537
	}
	_, err = companies.employee(1, socioeconomicJobFamilies[0], socioeconomicJobLevels[0])
	if err != nil {
		return
	}

	t.Fatal("expected invalid Employer index rejection, got employee")

}
