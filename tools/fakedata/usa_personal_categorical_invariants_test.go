// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"reflect"
	"testing"
)

// TestUSAPersonalCategoricalBranchGates poisons inapplicable tables/indexes to
// prove they are never evaluated, rather than merely omitted from the result.
func TestUSAPersonalCategoricalBranchGates(t *testing.T) {

	w := readUSAPersonalCategoricalHarness(t)
	selected := map[PrimaryActivity]usaPersonalCategoricalRecord{}
	for index := PersonIndex(1); index <= 2000 && len(selected) < 6; index++ {
		r, err := w.person(index)
		if err != nil {
			t.Fatalf("expected real categorical branch, got %v", err)
		}
		selected[r.activity] = r
	}
	if len(selected) != 6 {
		t.Fatalf("expected six real branches, got %d", len(selected))
	}
	for _, activity := range socioeconomicActivities {
		t.Run(string(activity), func(t *testing.T) {

			r := selected[activity]
			// Fault injection after admission is confined to this test fixture;
			// no changed asset or changed compiled model is readmitted.
			trial, model, companies := *w, *w.model, *w.companies
			trial.model, trial.companies = &model, &companies
			for node := 2; node < 6; node++ {
				if activity == PrimaryActivityEmployee && node < 4 ||
					activity == PrimaryActivitySelfEmployed && node >= 4 {
					continue
				}
				model.tables[node] = socioeconomicWeights{}
			}
			if activity != PrimaryActivityEmployee {
				clear(companies.sampler.employers[:])
			}
			got, err := trial.person(r.coordinate)
			if err != nil {
				t.Fatalf("expected no access to poisoned inapplicable nodes, got %v", err)
			}
			if !reflect.DeepEqual(got, r) {
				t.Fatal("expected unchanged record with inapplicable nodes poisoned, got different record")
			}
			wrong := 2
			if activity == PrimaryActivityEmployee {
				wrong = 4
			}
			before := len(got.traces)
			_, err = trial.category(&got, wrong,
				socioeconomicParent{"education", 0}, socioeconomicParent{"region", 0})
			usaCompanyEmployerRejected(t, err, nil, "inapplicable occupational node query")
			if activity != PrimaryActivityEmployee {
				err = trial.employer(&got, 0, 0)
				usaCompanyEmployerRejected(t, err, nil, "inapplicable Employer query")
			}
			if len(got.traces) != before {
				t.Fatal("expected no draw for inapplicable query, got new trace")
			}

		})
	}
	for _, parents := range [][]socioeconomicParent{
		{{"age-bucket", -1}, {"region", 0}}, {{"age-bucket", 6}, {"region", 0}},
		{{"age-bucket", 0}, {"region", 9}}, {{"region", 0}, {"age-bucket", 0}}, {},
	} {
		r := selected[PrimaryActivityEmployee]
		_, err := w.category(&r, 0, parents...)
		usaCompanyEmployerRejected(t, err, nil, "invalid parent ordinal, order or cardinality")
	}
	for row := range 54 {
		cumulative := w.model.tables[0].cumulative[row*7 : (row+1)*7]
		if cumulative[3] != cumulative[4] || socioeconomicPosition(cumulative, cumulative[3]) != 5 {
			t.Fatalf("expected zero-width Education E=4 interval at row %d, got %v", row, cumulative)
		}
	}
	if reflect.TypeFor[EmployeeFacts]().NumField() != 3 || reflect.TypeFor[SelfEmployedFacts]().NumField() != 2 {
		t.Fatal("expected frozen branch payloads without relational duplicates, got changed shape")
	}

}

// TestUSAPersonalCategoricalGeographyDependency uses two admitted localities in
// explicit test fixtures, each selected once by the actual Layer 1 consumer.
func TestUSAPersonalCategoricalGeographyDependency(t *testing.T) {

	w := readUSAPersonalCategoricalHarness(t)
	var variants [2]usaPersonalCategoricalHarness
	for scenario, division := range []uint32{0, 8} {
		geography := *w.base.base.geography
		geography.weights = make([]uint64, len(geography.locations))
		for i, location := range geography.locations {
			if location.state.DivisionOrdinal == division {
				geography.weights[i] = 1
				break
			}
		}
		base, layer1 := *w.base.base, *w.base
		base.geography, layer1.base = &geography, &base
		variants[scenario] = *w
		variants[scenario].base = &layer1
	}
	var educationChanged, activityChanged int
	for index := PersonIndex(1); index <= 128; index++ {
		left, err := variants[0].person(index)
		if err != nil {
			t.Fatalf("expected first coherent locality fixture, got %v", err)
		}
		right, err := variants[1].person(index)
		if err != nil {
			t.Fatalf("expected second coherent locality fixture, got %v", err)
		}
		if left.layer1.divisionOrdinal != 0 || right.layer1.divisionOrdinal != 8 ||
			left.traces[0].parents[1].value != 0 || right.traces[0].parents[1].value != 8 ||
			left.traces[1].parents[2].value != 0 || right.traces[1].parents[2].value != 8 ||
			left.traces[0].before == right.traces[0].before || left.traces[1].before == right.traces[1].before {
			t.Fatal("expected actual locality Division in categorical parents and streams, got different binding")
		}
		if left.layer1.person.BirthDate != right.layer1.person.BirthDate ||
			left.layer1.person.FirstName != right.layer1.person.FirstName ||
			left.layer1.person.LastName != right.layer1.person.LastName {
			t.Fatal("expected geography-independent age and names, got changed Layer 1 values")
		}
		if left.education != right.education {
			educationChanged++
		}
		if left.activity != right.activity {
			activityChanged++
		}
	}
	if educationChanged == 0 || activityChanged == 0 {
		t.Fatalf("expected geography capable of changing both categorical nodes, got %d/%d",
			educationChanged, activityChanged)
	}

}
