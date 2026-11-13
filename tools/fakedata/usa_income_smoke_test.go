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

// TestUSAIncomeIsolation verifies all six local streams and non-evaluation of
// inapplicable branches, plus actual consultation of the addressed tuple map.
func TestUSAIncomeIsolation(t *testing.T) {

	data := publishedUSATestData(t)
	world, err := newUSACountryWorld(data.country, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected prepared USA runtime World, got %v", err)
	}
	w := &world.sampler
	var representatives [6]usaIncomeRecord
	for coordinate := PersonIndex(1); coordinate <= 2000; coordinate++ {
		r, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected representative Person, got %v", err)
		}
		representatives[r.trace.branch] = r
	}
	for branch, r := range representatives {

		if r.categorical.coordinate == 0 {
			t.Fatalf("expected branch %d representative, got absent branch", branch)
		}
		original := w.income
		poisoned := *original
		for other := range 6 {
			if other != branch {
				poisoned.numeric.income[other] = socioeconomicIncome{}
				poisoned.fingerprints[other] = [32]byte{}
			}
		}
		w.income = &poisoned
		value, trace, err := w.sampleIncome(r.categorical)
		if err != nil {
			t.Fatalf("expected selected branch only, got %v", err)
		}
		if value != r.income || !reflect.DeepEqual(trace, r.trace) {
			t.Fatal("expected unchanged selected stream with other pools poisoned, got mismatch")
		}
		w.income = original
		changed := *original
		changed.numeric.income[branch].laws = slices.Clone(original.numeric.income[branch].laws)
		law := &changed.numeric.income[branch].laws[0]
		law.values = slices.Clone(law.values)
		law.values[len(law.values)-1]++
		model := *data.country.socioeconomic.model
		model.numeric.income[branch] = changed.numeric.income[branch]
		projection, err := model.canonicalProjection()
		if err != nil {
			t.Fatalf("expected private perturbation projection, got %v", err)
		}
		err = model.prepareIdentity(projection)
		if err != nil {
			t.Fatalf("expected private perturbation fingerprint, got %v", err)
		}
		copy(changed.fingerprints[:], model.fingerprints[socioeconomicIncomeEmployee:])
		for other := range 6 {
			if (changed.fingerprints[other] != original.fingerprints[other]) != (other == branch) {
				t.Fatalf("expected branch-local fingerprint change at %d, got change at %d", branch, other)
			}
			if other != branch {
				w.income = &changed
				value, trace, err = w.sampleIncome(representatives[other].categorical)
				if err != nil {
					t.Fatalf("expected isolated other-branch stream, got %v", err)
				}
				if value != representatives[other].income || !reflect.DeepEqual(trace, representatives[other].trace) {
					t.Fatal("expected complete sibling Income stream isolation, got mismatch")
				}
			}
		}
		changed = *original
		changed.numeric.income[branch].tupleMap = slices.Clone(original.numeric.income[branch].tupleMap)
		changed.numeric.income[branch].tupleMap[r.trace.tuple] = 65535
		w.income = &changed
		_, _, err = w.sampleIncome(r.categorical)
		usaCompanyEmployerRejected(t, err, nil, "addressed map consultation")
		w.income = original
		for _, pool := range w.categorical.companies.model.numeric.income {
			if len(pool.laws) != 0 || len(pool.tupleMap) != 0 {
				t.Fatal("expected Company model without Income, got monetary dependency")
			}
		}

	}

}

// TestUSAIncomeRuntimeRejects prevents skipped/partial successful generation,
// invalid coordinates and treating a missing selected map as productive zero.
func TestUSAIncomeRuntimeRejects(t *testing.T) {

	w := readUSAIncomeHarness(t)
	for _, index := range []PersonIndex{0, Layer1MaxPersonIndex + 1, math.MaxUint64} {
		result, err := w.person(index)
		if !reflect.DeepEqual(result, usaIncomeRecord{}) {
			t.Fatal("expected zero result on invalid coordinate, got partial success")
		}
		usaCompanyEmployerRejected(t, err, nil, "invalid Person coordinate")
	}
	original := w.income
	w.income = nil
	_, err := w.person(1)
	usaCompanyEmployerRejected(t, err, nil, "missing Income admission")
	w.income = original
	r, err := w.person(1)
	if err != nil {
		t.Fatalf("expected actual branch, got %v", err)
	}
	w.income = &usaIncomeCandidate{}
	_, _, err = w.sampleIncome(r.categorical)
	usaCompanyEmployerRejected(t, err, nil, "unadmitted zero candidate")
	w.income = original
	for _, mutate := range []func(*usaPersonalCategoricalRecord){
		func(r *usaPersonalCategoricalRecord) { r.activity = PrimaryActivity("unknown") },
		func(r *usaPersonalCategoricalRecord) { r.layer1.anchorAge = 20 },
		func(r *usaPersonalCategoricalRecord) { r.layer1.divisionOrdinal = 9 },
	} {
		changed := r.categorical
		mutate(&changed)
		_, _, err = w.sampleIncome(changed)
		usaCompanyEmployerRejected(t, err, nil, "invalid Income parent")
	}
	counterfactual := r.categorical
	counterfactual.activity = PrimaryActivityInactive
	counterfactual.education = socioeconomicEducationLevels[4]
	counterfactual.employee, counterfactual.selfEmployed, counterfactual.company = nil, nil, nil
	_, trace, err := w.sampleIncome(counterfactual)
	if err != nil {
		t.Fatalf("expected preserved E4 structural map, got %v", err)
	}
	if trace.branch != 5 || trace.law != 0 || trace.parents[1].value != 4 {
		t.Fatal("expected counterfactual Education address, got skipped map axis")
	}

}

// TestUSAIncomeSelectedCompany checks that Employer is only a relational key;
// equal selected Industry/SizeBand payloads give identical Income innovations.
func TestUSAIncomeSelectedCompany(t *testing.T) {

	w := readUSAIncomeHarness(t)
	var employee usaIncomeRecord
	for coordinate := PersonIndex(1); coordinate <= 100; coordinate++ {
		r, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected actual Person, got %v", err)
		}
		if r.categorical.activity == PrimaryActivityEmployee {
			employee = r
			break
		}
	}
	if employee.categorical.employee == nil {
		t.Fatal("expected Employee representative, got no selected Company")
	}
	var replacement CompanyIndex
	for index := CompanyIndex(1); index <= 65536; index++ {
		company := usaCompanyEmployerAt(t, w.categorical.companies, index)
		if index != employee.categorical.employee.Employer &&
			company.Industry == employee.categorical.company.Industry &&
			company.SizeBand == employee.categorical.company.SizeBand {
			replacement = index
			break
		}
	}
	if replacement == 0 {
		t.Fatal("expected second real Company with equal permitted payload, got no fixture")
	}
	r := employee.categorical
	facts := *r.employee
	facts.Employer = replacement
	r.employee = &facts
	company := usaCompanyEmployerAt(t, w.categorical.companies, replacement)
	r.company = &company
	value, trace, err := w.sampleIncome(r)
	if err != nil {
		t.Fatalf("expected equivalent selected payload, got %v", err)
	}
	if value != employee.income || !reflect.DeepEqual(trace, employee.trace) {
		t.Fatal("expected Employer-key equivalence, got employer-specific Income effect")
	}
	r.layer1.anchorAge = 75
	r.education = socioeconomicEducationLevels[6]
	value, trace, err = w.sampleIncome(r)
	if err != nil {
		t.Fatalf("expected employee excluded-parent invariance, got %v", err)
	}
	if value != employee.income || !reflect.DeepEqual(trace, employee.trace) {
		t.Fatal("expected employee age/Education exclusion, got changed stream")
	}

}

// readUSAIncomeHarness selects the sampler of the remotely loaded USA World.
func readUSAIncomeHarness(t *testing.T) *usaIncomeHarness {

	t.Helper()
	world, err := newUSACountryWorld(publishedUSATestData(t).country,
		IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected prepared USA runtime World, got %v", err)
	}

	return &world.sampler
}
