// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"fmt"
	"reflect"
	"slices"
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

// TestUSAPersonalCategoricalIsolation checks permitted dependency boundaries
// with technical perturbations; it does not admit any replacement asset.
func TestUSAPersonalCategoricalIsolation(t *testing.T) {

	w := readUSAPersonalCategoricalHarness(t)
	input := readUSAPersonalCategoricalInput(t, readUSAPersonalCategoricalAssets(t))
	input[2].Weights[0]++
	changed, err := compileUSAPersonalCategorical(input)
	if err != nil {
		t.Fatalf("expected bounded professional test fixture, got %v", err)
	}
	if changed.identity != ([32]byte{}) || changed.fingerprints[2] == w.model.fingerprints[2] ||
		changed.fingerprints[0] != w.model.fingerprints[0] || changed.fingerprints[1] != w.model.fingerprints[1] {
		t.Fatal("expected only the changed professional fingerprint, without admission, got different dependencies")
	}
	professional := *w
	professional.model = changed
	_, err = professional.person(1)
	usaCompanyEmployerRejected(t, err, nil, "unadmitted professional fixture used as a candidate")

	companyInput := usaCompanyEmployerNumericalInput(w.companies.model)
	companyInput.companyIndustry[0]++
	companyModel, err := compileUSACompanyEmployer(companyInput)
	if err != nil {
		t.Fatalf("expected bounded Company test fixture, got %v", err)
	}
	if companyModel.fingerprints[0] == w.companies.model.fingerprints[0] || companyModel.identity != ([32]byte{}) {
		t.Fatal("expected Company Industry perturbation without admission, got unchanged or admitted model")
	}
	companyVariant := *w
	companyVariant.companies = &usaCompanyEmployerHarness{model: companyModel}

	// Only existing unused Income fingerprints are perturbed. No monetary law,
	// zero Income value, Income identity or full model identity is manufactured.
	downstream, companyCopy, samplerModel := *w, *w.companies, *w.companies.sampler.model
	companyCopy.sampler.model, downstream.companies = &samplerModel, &companyCopy
	for node := socioeconomicIncomeEmployee; node <= socioeconomicIncomeInactive; node++ {
		samplerModel.fingerprints[node][0] ^= 1
	}
	var changedFamilies int
	for index := PersonIndex(1); index <= 128; index++ {

		original, err := w.person(index)
		if err != nil {
			t.Fatalf("expected original internal record, got %v", err)
		}
		replayed, err := downstream.person(index)
		if err != nil {
			t.Fatalf("expected downstream-independent record, got %v", err)
		}
		if !reflect.DeepEqual(original, replayed) {
			t.Fatal("expected no Income fingerprint influence, got changed record")
		}
		root, err := professional.base.person(index)
		if err != nil {
			t.Fatalf("expected professional-independent actual Layer 1, got %v", err)
		}
		if !reflect.DeepEqual(root, original.layer1) {
			t.Fatal("expected unchanged age/names/geography/contacts, got professional influence")
		}
		for step := range 2 {

			trace := original.traces[step]
			for variant, harness := range []*usaPersonalCategoricalHarness{&professional, &companyVariant, w} {
				r := usaPersonalCategoricalRecord{coordinate: index, layer1: original.layer1, activity: original.activity}
				if variant == 2 {
					r.layer1.person.PhotoID = "test-photo-without-categorical-dependency"
				}
				got, err := harness.category(&r, step, trace.parents...)
				if err != nil {
					t.Fatalf("expected isolated common node, got %v", err)
				}
				if got != trace.position || !reflect.DeepEqual(r.traces[0], trace) {
					t.Fatalf("expected common node isolation for variant %d, got changed draw", variant)
				}
			}

		}
		education := slices.Index(socioeconomicEducationLevels[:], original.education)
		probe := usaPersonalCategoricalRecord{coordinate: index, activity: PrimaryActivityEmployee}
		before, err := w.category(&probe, 2,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", int(root.divisionOrdinal)})
		if err != nil {
			t.Fatalf("expected original professional conditional probe, got %v", err)
		}
		after, err := professional.category(&probe, 2,
			socioeconomicParent{"education", education}, socioeconomicParent{"region", int(root.divisionOrdinal)})
		if err != nil {
			t.Fatalf("expected changed professional conditional probe, got %v", err)
		}
		if before != after {
			changedFamilies++
		}

	}
	if changedFamilies == 0 {
		t.Fatal("expected professional perturbation capable of changing its own node, got no change")
	}
	writeUSAPersonalCategoricalEvidence(t, "invariants.json", map[string]any{
		"status": "PASS", "isolationCoordinates": 128, "professionalProbeChanges": changedFamilies,
		"privateHarnessOnly": true, "changedAssetsReadmitted": false,
		"professionalLayer1Isolation": true, "companyCommonNodeIsolation": true,
		"photoCommonNodeIsolation": true, "incomeFingerprintIsolation": true,
		"publicWorldConstructed": false, "incomeLawsProduced": 0,
		"rootDigest": fmt.Sprintf("%x", w.base.streams.root),
	})

}
