// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"
)

// TestSocioeconomicCompanyIDs separates generic identity coordinates from
// membership of a constructed finite Company universe.
func TestSocioeconomicCompanyIDs(t *testing.T) {
	namespace := IdentityNamespace{Name: "krenalis-demo", Generation: 1}
	for _, index := range []CompanyIndex{1, 65536, 65537, math.MaxUint64} {
		id, err := CompanyIDForIndex(namespace, index)
		if err != nil {
			t.Fatalf("expected Company identity for %d, got %v", index, err)
		}
		if index == 1 && id != "company_v1_krenalis-demo_00000001_ef434ynxgmwyh" {
			t.Fatalf("expected frozen first Company identity, got %s", id)
		}
		decoded, err := CompanyIndexFromID(namespace, id)
		if err != nil {
			t.Fatalf("expected Company inverse, got %v", err)
		}
		if decoded != index {
			t.Fatalf("expected Company coordinate %d, got %d", index, decoded)
		}
	}
	id, err := CompanyIDForIndex(namespace, 0)
	assertSocioeconomicFailure(t, id, "", err, ErrInvalidIndex)
	id, err = CompanyIDForIndex(IdentityNamespace{}, 1)
	assertSocioeconomicFailure(t, id, "", err, ErrInvalidIdentityNamespace)
	person, err := PersonIDForIndex(namespace, 1)
	if err != nil {
		t.Fatalf("expected person identity, got %v", err)
	}
	index, err := CompanyIndexFromID(namespace, person)
	assertSocioeconomicFailure(t, index, CompanyIndex(0), err, ErrWrongEntityKind)
	index, err = CompanyIndexFromID(namespace, "invalid")
	assertSocioeconomicFailure(t, index, CompanyIndex(0), err, ErrMalformedSyntheticID)
}

// TestSocioeconomicInvalidState verifies all-or-nothing construction and the
// precedence of wrapper validation over coordinate validation.
func TestSocioeconomicInvalidState(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	model := socioeconomicTestModel(t, nil)
	for _, input := range []struct {
		base  *World
		model *SocioeconomicModel
	}{
		{nil, model}, {&World{}, model}, {base, nil}, {base, &SocioeconomicModel{}},
	} {
		world, err := NewSocioeconomicWorld(input.base, input.model)
		if err != nil {
			if !errors.Is(err, ErrInvalidSocioeconomicWorld) || world != nil {
				t.Fatalf("expected nil world and invalid-world error, got %v/%v", world, err)
			}
			continue
		}
		t.Fatalf("expected invalid construction failure, got success")
	}
	for _, world := range []*SocioeconomicWorld{nil, {}} {
		if world.ID() != "" {
			t.Fatalf("expected empty invalid identity, got %s", world.ID())
		}
		for _, index := range []uint64{0, 1, math.MaxUint64} {
			company, err := world.Company(CompanyIndex(index))
			assertSocioeconomicFailure(t, company, Company{}, err, ErrInvalidSocioeconomicWorld)
			facts, err := world.Facts(PersonIndex(index))
			assertSocioeconomicFailure(t, facts, SocioeconomicFacts{}, err, ErrInvalidSocioeconomicWorld)
			person, err := world.Person(PersonIndex(index))
			assertSocioeconomicFailure(t, person, SocioeconomicPerson{}, err, ErrInvalidSocioeconomicWorld)
		}
	}

	copyBase, copyModel := *base, *model
	w := mustSocioeconomicWorld(t, &copyBase, &copyModel)
	for _, test := range []struct {
		index PersonIndex
		want  error
	}{
		{0, ErrInvalidIndex}, {Layer1MaxPersonIndex + 1, ErrLayer1IndexOutOfRange},
		{PersonIndex(math.MaxUint64), ErrLayer1IndexOutOfRange},
	} {
		facts, err := w.Facts(test.index)
		assertSocioeconomicFailure(t, facts, SocioeconomicFacts{}, err, test.want)
		person, err := w.Person(test.index)
		assertSocioeconomicFailure(t, person, SocioeconomicPerson{}, err, test.want)
	}
	for _, test := range []struct {
		index CompanyIndex
		want  error
	}{
		{0, ErrInvalidIndex}, {CompanyIndex(model.companyCount + 1), ErrCompanyIndexOutOfRange},
		{CompanyIndex(math.MaxUint64), ErrCompanyIndexOutOfRange},
	} {
		company, err := w.Company(test.index)
		assertSocioeconomicFailure(t, company, Company{}, err, test.want)
	}
	_, err := w.Person(Layer1MaxPersonIndex)
	if err != nil {
		t.Fatalf("expected maximum operational Person, got %v", err)
	}

	// A valid law can realize an unusable one-Company universe; no fallback may
	// turn it into an artificial employer or expose lower-level weight errors.
	zero := socioeconomicTestModel(t, func(document map[string]any) {
		document["companyCount"] = 1
		laws := document["employeeCount"].(map[string]any)["laws"].([]any)
		for i := range laws {
			laws[i] = map[string]any{"values": []uint32{0, 1}, "weights": []uint32{math.MaxUint32, 1}}
		}
	})
	world, err := NewSocioeconomicWorld(base, zero)
	if err != nil {
		if world != nil || !errors.Is(err, ErrInvalidSocioeconomicWorld) ||
			errors.Is(err, ErrAllZeroWeights) || errors.Is(err, ErrTotalWeightOverflow) {
			t.Fatalf("expected only invalid-world classification and no world, got %v/%v", world, err)
		}
		return
	}

	t.Fatalf("expected zero-universe construction failure, got success")

}

func assertSocioeconomicFailure[T any](t *testing.T, got, zero T, err, expected error) {
	t.Helper()
	if err != nil {
		if !errors.Is(err, expected) || !reflect.DeepEqual(got, zero) {
			t.Fatalf("expected zero result and %v, got %#v/%v", expected, got, err)
		}
		return
	}
	t.Fatalf("expected %v, got success", expected)
}

func mustSocioeconomicWorld(t *testing.T, base *World, model *SocioeconomicModel) *SocioeconomicWorld {
	t.Helper()
	w, err := NewSocioeconomicWorld(base, model)
	if err != nil {
		t.Fatalf("expected complete socioeconomic world, got %v", err)
	}
	return w
}

func setSocioeconomicTestCategory(document map[string]any, name string, width int, choose func(int) int) {
	weights := document[name].(map[string]any)["weights"].([]any)
	for i := range weights {
		weights[i] = 0
		if i%width == choose(i/width) {
			weights[i] = 1
		}
	}
}

func socioeconomicCompanyAt(t *testing.T, world *SocioeconomicWorld, index CompanyIndex) Company {
	t.Helper()
	company, err := world.Company(index)
	if err != nil {
		t.Fatalf("expected Company %d, got %v", index, err)
	}
	return company
}

func socioeconomicFactsAt(t *testing.T, world *SocioeconomicWorld, index PersonIndex) SocioeconomicFacts {
	t.Helper()
	facts, err := world.Facts(index)
	if err != nil {
		t.Fatalf("expected Facts %d, got %v", index, err)
	}
	return facts
}

// socioeconomicTestInput supplies varied, complete canonical grids. It is a
// hermetic runtime fixture, not the prescribed production numerical model.
func socioeconomicTestInput(t *testing.T, change func(map[string]any)) string {

	t.Helper()
	var document map[string]any
	err := json.Unmarshal([]byte(socioeconomicFixture(t)), &document)
	if err != nil {
		t.Fatalf("expected foundation fixture, got %v", err)
	}
	document["companyCount"] = 96
	ages := document["ages"].(map[string]any)
	for i, lower := range []int{35, 40, 45, 30, 50, 32, 60, 55} {
		values := make([]uint32, 55)
		for age := lower; age <= 75; age++ {
			values[age-21] = 1
		}
		ages[socioeconomicAgeNames[i]] = map[string]any{"lowerBounds": []int{21, lower}, "compiled": values}
	}
	for _, table := range []struct {
		name  string
		rows  int
		width int
	}{
		{"companyIndustry", 1, 14}, {"education", 40, 7}, {"primaryActivity", 280, 6},
		{"employeeJobFamily", 140, 10}, {"employeeJobLevel", 140, 4},
		{"selfEmployedJobFamily", 140, 10}, {"selfEmployedIndustry", 10, 14},
	} {
		weights := make([]uint32, table.rows*table.width)
		for i := range weights {
			weights[i] = 1 + uint32((i/table.width*5+i%table.width*3)%11)
			if i%table.width == 0 {
				weights[i] = 1
			}
		}
		document[table.name] = map[string]any{"weights": weights}
	}
	laws := make([]any, 14)
	for industry := range laws {
		laws[industry] = map[string]any{
			"values":  []uint32{0, 1, 9, 10, 49, 50, 249, 250, 1000, math.MaxUint32},
			"weights": []uint32{1, uint32(industry + 1), 2, 3, 1, 2, 3, 1, 2, 1},
		}
	}
	document["employeeCount"] = map[string]any{"laws": laws}
	industryFactors, sizeFactors := make([]uint16, 140), make([]uint16, 16)
	for i := range industryFactors {
		industryFactors[i] = uint16(2 * (i/14 + 1) * (i%14 + 1))
	}
	for i := range sizeFactors {
		sizeFactors[i] = uint16(3 * (i/4 + 1) * (i%4 + 1))
	}
	document["employer"] = map[string]any{"cap": 17, "industryFactors": industryFactors, "sizeFactors": sizeFactors}
	income := document["income"].(map[string]any)
	for branch, name := range socioeconomicBranchNames {
		count := 280
		if branch == 0 {
			count = 56000
		}
		if branch == 1 {
			count = 5600
		}
		tupleMap := make([]uint16, count)
		for i := range tupleMap {
			tupleMap[i] = uint16((i/3 + i/17 + i/127) % 5)
			if branch == 0 && i%5 == 0 {
				tupleMap[i] = math.MaxUint16
			}
		}
		laws := make([]any, 5)
		for law := range laws {
			laws[law] = map[string]any{
				"values":  []uint32{0, uint32(branch*10000 + law*10 + 100), uint32(branch*10000 + law*100 + 1000)},
				"weights": []uint32{1, uint32(law + 1), 2},
			}
		}
		income[name] = map[string]any{"tupleMap": tupleMap, "laws": laws}
	}

	// Normalize fixture representation so mutation callbacks always see JSON
	// arrays as []any and numeric values as exactly representable float64s.
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("expected encoded fixture, got %v", err)
	}
	err = json.Unmarshal(encoded, &document)
	if err != nil {
		t.Fatalf("expected normalized fixture, got %v", err)
	}
	if change != nil {
		change(document)
	}
	encoded, err = json.Marshal(document)
	if err != nil {
		t.Fatalf("expected mutated fixture JSON, got %v", err)
	}

	return string(encoded)
}

func socioeconomicTestModel(t *testing.T, change func(map[string]any)) *SocioeconomicModel {
	t.Helper()
	model, err := decodeSocioeconomicModel(strings.NewReader(socioeconomicTestInput(t, change)))
	if err != nil {
		t.Fatalf("expected valid runtime model, got %v", err)
	}
	return model
}
