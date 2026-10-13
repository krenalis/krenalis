// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// TestSocioeconomicAcquisitionLifetime checks public acquisition, immutable
// reuse, origin/format independence and network-free construction and access.
func TestSocioeconomicAcquisitionLifetime(t *testing.T) {

	for _, source := range []*ItalianWorldSource{nil, {}} {
		model, err := source.Load(t.Context())
		if err != nil {
			if model != nil || errors.Is(err, ErrInvalidSocioeconomicWorld) {
				t.Fatalf("expected acquisition failure without a model or construction classification, got %p/%v",
					model, err)
			}
			continue
		}
		t.Fatalf("expected invalid acquisition owner failure, got success")
	}
	assets := internationalAssetFixture(t)
	input := assets["socioeconomic.json"]
	var formatted bytes.Buffer
	err := json.Indent(&formatted, input, "", "  ")
	if err != nil {
		t.Fatalf("expected formatting-only JSON variant, got %v", err)
	}
	var models [2]*SocioeconomicModel
	var prepared [2]*ItalianWorldData
	for origin, payload := range [][]byte{input, formatted.Bytes()} {
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			body, ok := assets[strings.TrimPrefix(r.URL.Path, "/")]
			if !ok {
				t.Errorf("expected declared JSON path, got %q", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
				return
			}
			if r.URL.Path == "/socioeconomic.json" {
				body = payload
			}
			_, _ = w.Write(body)
		}))
		origins := []string{server.URL}
		source, err := NewItalianWorldSource(origins)
		if err != nil {
			t.Fatalf("expected public source, got %v", err)
		}
		origins[0] = "http://invalid.test"
		data, err := source.Load(t.Context())
		server.Close()
		if err != nil {
			t.Fatalf("expected acquired immutable model, got %v", err)
		}
		again, err := source.Load(t.Context())
		if err != nil {
			t.Fatalf("expected retained model after origin closure, got %v", err)
		}
		if data != again || requests.Load() != 4 {
			t.Fatalf("expected four acquisitions and identical data handle, got %d requests and %p/%p",
				requests.Load(), data, again)
		}
		model, err := data.Model()
		if err != nil {
			t.Fatalf("expected prepared model, got %v", err)
		}
		prepared[origin] = data
		models[origin] = model
		// Only prepared data and models escape this acquisition owner's scope.
	}
	if models[0] == models[1] || models[0].digest != models[1].digest ||
		models[0].fingerprints != models[1].fingerprints {
		t.Fatalf("expected distinct handles with identical logical identities, got %p/%p", models[0], models[1])
	}

	bases := [2]*World{mustItalianJSONWorld(t, prepared[0]), mustItalianJSONWorld(t, prepared[1])}
	first := mustSocioeconomicWorld(t, bases[0], models[0])
	second := mustSocioeconomicWorld(t, bases[1], models[1])
	if first.ID() != second.ID() {
		t.Fatalf("expected origin-independent world identity, got %s/%s", first.ID(), second.ID())
	}
	for index := PersonIndex(1); index <= 64; index++ {
		a, b := socioeconomicFactsAt(t, first, index), socioeconomicFactsAt(t, second, index)
		if !reflect.DeepEqual(a, b) {
			t.Fatalf("expected origin-independent Facts, got %+v/%+v", a, b)
		}
	}
	for index := CompanyIndex(1); uint64(index) <= models[0].companyCount; index++ {
		if a, b := socioeconomicCompanyAt(t, first, index), socioeconomicCompanyAt(t, second, index); a != b {
			t.Fatalf("expected origin-independent Company, got %+v/%+v", a, b)
		}
	}

	var group sync.WaitGroup
	for worker := range 8 {
		group.Go(func() {
			world, err := NewSocioeconomicWorld(bases[worker%2], models[worker%2])
			if err != nil {
				t.Errorf("expected concurrent construction from shared model, got %v", err)
				return
			}
			facts, err := world.Facts(PersonIndex(worker + 1))
			if err != nil {
				t.Errorf("expected concurrent Facts, got %v", err)
				return
			}
			expected, err := first.Facts(PersonIndex(worker + 1))
			if err != nil {
				t.Errorf("expected reference Facts, got %v", err)
				return
			}
			if world.ID() != first.ID() || !reflect.DeepEqual(facts, expected) {
				t.Errorf("expected identical shared-model world and Facts, got %s/%+v", world.ID(), facts)
			}
		})
	}
	group.Wait()

}

// TestSocioeconomicRandomAccess checks reordered and concurrent reads,
// composite agreement and caller-owned payloads.
func TestSocioeconomicRandomAccess(t *testing.T) {

	base := mustLayer1World(t, PopulationAnchorDate, layer1TestSeed)
	model := socioeconomicTestModel(t, nil)
	world := mustSocioeconomicWorld(t, base, model)
	coordinates := make([]PersonIndex, 130)
	for i := range 128 {
		coordinates[i] = PersonIndex(i + 1)
	}
	coordinates[128], coordinates[129] = 1234567, Layer1MaxPersonIndex
	people := map[PersonIndex]SocioeconomicPerson{}
	for _, index := range coordinates {
		person, err := world.Person(index)
		if err != nil {
			t.Fatalf("expected composite Person, got %v", err)
		}
		original, err := base.Person(index)
		if err != nil {
			t.Fatalf("expected original Layer 1 Person, got %v", err)
		}
		if person.Person != original || !reflect.DeepEqual(person.Facts, socioeconomicFactsAt(t, world, index)) {
			t.Fatalf("expected exact Layer 1 and facts-only agreement, got %+v", person)
		}
		people[index] = person
	}
	companies := make([]Company, model.companyCount)
	for i := range companies {
		companies[i] = socioeconomicCompanyAt(t, world, CompanyIndex(i+1))
	}
	orders := [][]PersonIndex{append([]PersonIndex(nil), coordinates...), append([]PersonIndex(nil), coordinates...)}
	for i := range len(orders[0]) / 2 {
		j := len(orders[0]) - i - 1
		orders[0][i], orders[0][j] = orders[0][j], orders[0][i]
	}
	rng := rand.New(rand.NewPCG(72, 6381))
	rng.Shuffle(len(orders[1]), func(i, j int) { orders[1][i], orders[1][j] = orders[1][j], orders[1][i] })
	for _, order := range orders {
		for _, coordinate := range order {
			person, err := world.Person(coordinate)
			if err != nil {
				t.Fatalf("expected reordered composite Person, got %v", err)
			}
			if !reflect.DeepEqual(person, people[coordinate]) {
				t.Fatalf("expected unchanged Person %d, got %+v", coordinate, person)
			}
			companyIndex := CompanyIndex(uint64(coordinate)%model.companyCount + 1)
			if company := socioeconomicCompanyAt(t, world, companyIndex); company != companies[companyIndex-1] {
				t.Fatalf("expected unchanged interleaved Company, got %+v", company)
			}
		}
	}
	for i := len(companies) - 1; i >= 0; i-- {
		if company := socioeconomicCompanyAt(t, world, CompanyIndex(i+1)); company != companies[i] {
			t.Fatalf("expected reverse-order Company, got %+v", company)
		}
	}

	var group sync.WaitGroup
	for worker := range 12 {
		group.Go(func() {
			for offset := range 64 {
				coordinate := coordinates[(worker*13+offset*7)%len(coordinates)]
				facts, err := world.Facts(coordinate)
				if err != nil {
					t.Errorf("expected concurrent Facts, got %v", err)
					return
				}
				person, err := world.Person(coordinate)
				if err != nil {
					t.Errorf("expected concurrent Person, got %v", err)
					return
				}
				companyIndex := CompanyIndex(uint64(coordinate)%model.companyCount + 1)
				company, err := world.Company(companyIndex)
				if err != nil {
					t.Errorf("expected concurrent Company, got %v", err)
					return
				}
				if !reflect.DeepEqual(facts, people[coordinate].Facts) ||
					!reflect.DeepEqual(person, people[coordinate]) ||
					company != companies[companyIndex-1] {
					t.Errorf("expected stable concurrent results at %d, got %+v/%+v/%+v",
						coordinate, facts, person, company)
					return
				}
			}
		})
	}
	group.Wait()

	for _, coordinate := range coordinates {
		facts := socioeconomicFactsAt(t, world, coordinate)
		if facts.Employee != nil {
			*facts.Employee = EmployeeFacts{}
		}
		if facts.SelfEmployed != nil {
			*facts.SelfEmployed = SelfEmployedFacts{}
		}
		facts.Education, facts.Income = "changed", 99
		if got := socioeconomicFactsAt(t, world, coordinate); !reflect.DeepEqual(got, people[coordinate].Facts) {
			t.Fatalf("expected caller-mutation isolation, got %+v", got)
		}
	}

}
