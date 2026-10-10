// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"reflect"
	"testing"
)

// TestSocioeconomicCompanyCountIsolation verifies that N changes membership and
// Employer indexes without entering any local stochastic model identity.
func TestSocioeconomicCompanyCountIsolation(t *testing.T) {
	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	a := socioeconomicTestModel(t, nil)
	b := socioeconomicTestModel(t, func(document map[string]any) { document["companyCount"] = 128 })
	before, after := mustSocioeconomicWorld(t, base, a), mustSocioeconomicWorld(t, base, b)
	if a.fingerprints != b.fingerprints || a.digest == b.digest || before.ID() == after.ID() {
		t.Fatalf("expected N-only whole identity change with unchanged local fingerprints, got %s/%s",
			before.ID(), after.ID())
	}
	for index := CompanyIndex(1); uint64(index) <= a.companyCount; index++ {
		first, second := socioeconomicCompanyAt(t, before, index), socioeconomicCompanyAt(t, after, index)
		if first != second {
			t.Fatalf("expected unchanged common Company coordinate, got %+v/%+v", first, second)
		}
	}
	for index := PersonIndex(1); index <= 128; index++ {
		first, second := socioeconomicFactsAt(t, before, index), socioeconomicFactsAt(t, after, index)
		if first.Employee != nil && second.Employee != nil {
			first.Employee.Employer, second.Employee.Employer = 0, 0
			first.Income, second.Income = 0, 0
		}
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("expected N-independent non-Employer facts, got %+v/%+v", first, second)
		}
	}
}

// TestSocioeconomicEmployeeIncomeRelation changes only the selected employer
// key and within-band headcount while preserving Income's authorized parents.
func TestSocioeconomicEmployeeIncomeRelation(t *testing.T) {
	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	model := socioeconomicTestModel(t, func(document map[string]any) {
		setSocioeconomicTestCategory(document, "primaryActivity", 6, func(int) int { return 0 })
		setSocioeconomicTestCategory(document, "employeeJobFamily", 10, func(int) int { return 0 })
		setSocioeconomicTestCategory(document, "employeeJobLevel", 4, func(int) int { return 0 })
	})
	w := mustSocioeconomicWorld(t, base, model)
	var first, second CompanyIndex
	for i, industry := range w.companyIndustries {
		if w.companyCounts[i] == 0 {
			continue
		}
		for j := i + 1; j < len(w.companyCounts); j++ {
			if industry == w.companyIndustries[j] &&
				socioeconomicSizeBand(w.companyCounts[i]) == socioeconomicSizeBand(w.companyCounts[j]) {
				first, second = CompanyIndex(i+1), CompanyIndex(j+1)
				break
			}
		}
		if first != 0 {
			break
		}
	}
	if first == 0 {
		t.Fatalf("expected two Companies sharing Industry and SizeBand, got none")
	}
	w.employers[0] = socioeconomicEmployerIndex{companies: []CompanyIndex{first}, cumulative: []uint64{1}}
	before := socioeconomicFactsAt(t, w, 42)
	w.employers[0] = socioeconomicEmployerIndex{companies: []CompanyIndex{second}, cumulative: []uint64{1}}
	count := w.companyCounts[second-1]
	if count == 1 || count == 10 || count == 50 || count == 250 {
		w.companyCounts[second-1]++
	} else {
		w.companyCounts[second-1]--
	}
	after := socioeconomicFactsAt(t, w, 42)
	if before.Employee.Employer == after.Employee.Employer || before.Income != after.Income {
		t.Fatalf("expected different Employer with identical Income, got %+v/%+v", before, after)
	}
}

// TestSocioeconomicGenerationFailure rejects an impossible active-law lookup
// with no partial Person or Facts and no success value standing in for Income.
func TestSocioeconomicGenerationFailure(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	model := socioeconomicTestModel(t, func(document map[string]any) {
		setSocioeconomicTestCategory(document, "primaryActivity", 6, func(int) int { return 5 })
	})
	w := mustSocioeconomicWorld(t, base, model)
	model.income[5].laws = nil
	facts, err := w.Facts(42)
	if err != nil {
		if !reflect.DeepEqual(facts, SocioeconomicFacts{}) {
			t.Fatalf("expected zero Facts on failure, got %+v", facts)
		}
	}
	if err == nil {
		t.Fatalf("expected active-law failure, got success")
	}
	person, err := w.Person(42)
	if err != nil {
		if !reflect.DeepEqual(person, SocioeconomicPerson{}) {
			t.Fatalf("expected zero Person on failure, got %+v", person)
		}
		return
	}

	t.Fatalf("expected composite failure, got success")

}

// TestSocioeconomicModelIsolation mutates all fifteen owned assets and checks
// their local fingerprints and non-descendant facts independently.
func TestSocioeconomicModelIsolation(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	original := socioeconomicTestModel(t, nil)
	before := mustSocioeconomicWorld(t, base, original)
	for node, path := range socioeconomicPaths {
		t.Run(path, func(t *testing.T) {
			changed := socioeconomicTestModel(t, func(document map[string]any) {
				switch node {
				case 1:
					laws := document["employeeCount"].(map[string]any)["laws"].([]any)
					weights := laws[0].(map[string]any)["weights"].([]any)
					weights[1] = weights[1].(float64) + 1
				case 6:
					document["employer"].(map[string]any)["cap"] = 18
				case 9, 10, 11, 12, 13, 14:
					income := document["income"].(map[string]any)[socioeconomicBranchNames[node-9]].(map[string]any)
					values := income["laws"].([]any)[0].(map[string]any)["values"].([]any)
					values[1] = values[1].(float64) + 1
				default:
					names := map[int]string{
						0: "companyIndustry", 2: "education", 3: "primaryActivity", 4: "employeeJobFamily",
						5: "employeeJobLevel", 7: "selfEmployedJobFamily", 8: "selfEmployedIndustry",
					}
					weights := document[names[node]].(map[string]any)["weights"].([]any)
					weights[1] = weights[1].(float64) + 1
				}
			})
			for other := range original.fingerprints {
				if (original.fingerprints[other] != changed.fingerprints[other]) != (node == other) {
					t.Fatalf("expected only node %d fingerprint to change, got change at node %d", node, other)
				}
			}
			after := mustSocioeconomicWorld(t, base, changed)
			if before.ID() == after.ID() || original.digest == changed.digest {
				t.Fatalf("expected changed whole-model/world identity, got unchanged %s", after.ID())
			}
			for index := CompanyIndex(1); uint64(index) <= original.companyCount; index++ {
				a, b := socioeconomicCompanyAt(t, before, index), socioeconomicCompanyAt(t, after, index)
				if node >= 2 && a != b {
					t.Fatalf("expected unchanged Company for person-model mutation, got %+v/%+v", a, b)
				}
				if node == 1 && a.Industry != b.Industry {
					t.Fatalf("expected count-model-independent Industry, got %s/%s", a.Industry, b.Industry)
				}
			}
			for coordinate := PersonIndex(1); coordinate <= 128; coordinate++ {
				a, b := socioeconomicFactsAt(t, before, coordinate), socioeconomicFactsAt(t, after, coordinate)
				if node != 2 && a.Education != b.Education {
					t.Fatalf("expected unchanged Education, got %s/%s", a.Education, b.Education)
				}
				if node == 2 || node == 3 {
					continue
				}
				if a.PrimaryActivity != b.PrimaryActivity {
					t.Fatalf("expected unchanged PrimaryActivity, got %s/%s", a.PrimaryActivity, b.PrimaryActivity)
				}
				switch {
				case node >= 9:
					if a.PrimaryActivity == socioeconomicActivities[node-9] {
						a.Income, b.Income = 0, 0
					}
				case node == 0 || node == 1 || node == 6:
					if a.Employee != nil {
						a.Employee.Employer, b.Employee.Employer = 0, 0
						a.Income, b.Income = 0, 0
					}
				case node == 4 || node == 5:
					if a.Employee != nil {
						if node == 5 && a.Employee.JobFamily != b.Employee.JobFamily {
							t.Fatalf("expected unchanged employee JobFamily, got %+v/%+v", a.Employee, b.Employee)
						}
						continue
					}
				case node == 7 || node == 8:
					if a.SelfEmployed != nil {
						if node == 8 && a.SelfEmployed.JobFamily != b.SelfEmployed.JobFamily {
							t.Fatalf("expected unchanged self-employed JobFamily, got %+v/%+v",
								a.SelfEmployed, b.SelfEmployed)
						}
						continue
					}
				}
				if !reflect.DeepEqual(a, b) {
					t.Fatalf("expected non-descendant facts unchanged for node %d, got %+v/%+v", node, a, b)
				}
			}
		})
	}

	for age, name := range socioeconomicAgeNames {
		t.Run(name+" age map", func(t *testing.T) {
			changed := socioeconomicTestModel(t, func(document map[string]any) {
				mapping := document["ages"].(map[string]any)[name].(map[string]any)
				lower := mapping["lowerBounds"].([]any)
				boundary := int(lower[1].(float64))
				lower[1] = boundary + 1
				mapping["compiled"].([]any)[boundary-21] = 0
			})
			for node, ownAge := range socioeconomicNodeAges {
				if (original.fingerprints[node] != changed.fingerprints[node]) != (ownAge == age) {
					t.Fatalf("expected age map %d to affect only its consumer, got node %d", age, node)
				}
			}
		})
	}

}

// TestSocioeconomicPhotoFailure preserves Layer 1 errors and complete results.
func TestSocioeconomicPhotoFailure(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	w := mustSocioeconomicWorld(t, base, socioeconomicTestModel(t, nil))
	expected := socioeconomicFactsAt(t, w, 42)

	// Simulate an unexpected Layer 1 failure after valid construction.
	catalog := *base.faceCatalog
	catalog.faces = nil
	base.faceCatalog = &catalog
	person, err := w.Person(42)
	assertSocioeconomicFailure(t, person, SocioeconomicPerson{}, err, ErrNoCompatiblePhoto)
	if got := socioeconomicFactsAt(t, w, 42); !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected Facts independent of photo failure, got %+v", got)
	}

}

// TestSocioeconomicSnapshotIsolation verifies anchor semantics, namespace/root
// separation, and use of the base's stored WorldID instead of SnapshotID.
func TestSocioeconomicSnapshotIsolation(t *testing.T) {

	model := socioeconomicTestModel(t, nil)
	anchor := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	future := mustLayer1World(t, "2030-12-31", layer1TestSeed)
	other, err := NewWorld(WorldConfig{
		IdentityNamespace: IdentityNamespace{Name: "other", Generation: 2}, WorldSeed: layer1TestSeed,
		ReferenceDate: PopulationAnchorDate, FaceCatalog: anchor.faceCatalog,
	})
	if err != nil {
		t.Fatalf("expected second namespace, got %v", err)
	}
	a := mustSocioeconomicWorld(t, anchor, model)
	b := mustSocioeconomicWorld(t, future, model)
	c := mustSocioeconomicWorld(t, other, model)
	if a.ID() != b.ID() || a.ID() == c.ID() || anchor.snapshotID == future.snapshotID {
		t.Fatalf("expected snapshot-independent, namespace-bound identity, got %s/%s/%s", a.ID(), b.ID(), c.ID())
	}
	for index := PersonIndex(1); index <= 128; index++ {
		first, err := a.Person(index)
		if err != nil {
			t.Fatalf("expected anchor Person, got %v", err)
		}
		later, err := b.Person(index)
		if err != nil {
			t.Fatalf("expected later Person, got %v", err)
		}
		if first.Person.CurrentAge >= later.Person.CurrentAge || !reflect.DeepEqual(first.Facts, later.Facts) ||
			!reflect.DeepEqual(first.Facts, socioeconomicFactsAt(t, c, index)) {
			t.Fatalf("expected changing current age with unchanged anchor facts, got %+v/%+v", first, later)
		}
	}

	// Hold the base realization fixed while changing only its already-stored
	// identity. This detects reconstructing a replacement ID from configuration.
	stored := *anchor
	stored.worldID = other.worldID
	w := mustSocioeconomicWorld(t, &stored, model)
	if w.ID() != c.ID() {
		t.Fatalf("expected binding to actual stored WorldID, got %s/%s", w.ID(), c.ID())
	}

}
