// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"slices"
	"testing"
)

// TestUSACompanyEmployerIsolation checks seed and identity separation and
// verifies each innovation against the unchanged frozen stream registry.
func TestUSACompanyEmployerIsolation(t *testing.T) {

	categorical := readUSAPersonalCategoricalHarness(t)
	base, model := categorical.base, categorical.companies.model
	w, err := newUSACompanyEmployerHarness(base, model)
	if err != nil {
		t.Fatalf("expected internal USA harness, got %v", err)
	}
	otherNamespace := *base
	otherNamespace.namespace = IdentityNamespace{Name: "usa-company-replay", Generation: 2}
	replay, err := newUSACompanyEmployerHarness(&otherNamespace, model)
	if err != nil {
		t.Fatalf("expected namespace replay, got %v", err)
	}
	otherBase, err := newUSALayer1Harness(base.namespace, 2027, base.base, base.phones)
	if err != nil {
		t.Fatalf("expected second seed base, got %v", err)
	}
	otherSeed, err := newUSACompanyEmployerHarness(otherBase, model)
	if err != nil {
		t.Fatalf("expected seed replay, got %v", err)
	}
	changedFacts := false
	for _, coordinate := range []CompanyIndex{1, 42, 1009, 8191, 32769, 65536} {

		company := usaCompanyEmployerAt(t, w, coordinate)
		namespaced := usaCompanyEmployerAt(t, replay, coordinate)
		seeded := usaCompanyEmployerAt(t, otherSeed, coordinate)
		if company.ID == namespaced.ID || company.EmployeeCount != namespaced.EmployeeCount ||
			company.Industry != namespaced.Industry || company.SizeBand != namespaced.SizeBand || company.ID != seeded.ID {
			t.Fatal("expected namespace-only IDs and seed-independent identity, got coupling")
		}
		changedFacts = changedFacts || seeded.EmployeeCount != company.EmployeeCount || seeded.Industry != company.Industry

		industry := socioeconomicReferenceSelection(t, w.sampler.base, "company", uint64(coordinate),
			usaSocioeconomicPaths[0], model.fingerprints[0], socioeconomicRawWeights(model.numeric.companyIndustry.cumulative))
		law := model.numeric.employeeCount[industry]
		point := socioeconomicReferenceSelection(t, w.sampler.base, "company", uint64(coordinate),
			usaSocioeconomicPaths[1], model.fingerprints[1], socioeconomicRawWeights(law.cumulative),
			mustUint64Component(t, "industry", uint64(industry)))
		if company.Industry != socioeconomicIndustries[industry] || company.EmployeeCount != law.values[point] {
			t.Fatal("expected frozen raw-coordinate Company draws, got mismatch")
		}

	}
	if !changedFacts {
		t.Fatal("expected changed stochastic facts for the actual second seed, got identical sparse records")
	}
	for condition := range 40 {
		family, level := socioeconomicJobFamilies[condition/4], socioeconomicJobLevels[condition%4]
		first, err := w.employee(42, family, level)
		if err != nil {
			t.Fatalf("expected employee, got %v", err)
		}
		second, err := replay.employee(42, family, level)
		if err != nil {
			t.Fatalf("expected namespace replay employee, got %v", err)
		}
		if first != second {
			t.Fatal("expected namespace-independent Employer selection, got different coordinate")
		}
	}

	// Private sampler views remain unusable as public Italian models/worlds.
	if w.sampler.id != "" || w.sampler.model.initialized || w.sampler.base.data != nil {
		t.Fatal("expected incomplete private sampler view, got public initialization")
	}
	_, err = NewSocioeconomicWorld(w.sampler.base, w.sampler.model)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "rejected public World admission")
	for _, node := range []int{socioeconomicCompanyIndustry, socioeconomicCompanyEmployeeCount, socioeconomicEmployer} {
		if usaSocioeconomicPaths[node] != socioeconomicPaths[node] {
			t.Fatal("expected unchanged frozen path for admitted node, got mismatch")
		}
	}

	// One fingerprint changes only its own innovation, even when a whole-model
	// diagnostic digest or a downstream Income fingerprint also changes.
	for _, changed := range []int{0, 1, 6, 9} {
		changedSampler := w.sampler
		changedModel := *w.sampler.model
		changedModel.fingerprints[changed][0] ^= 1
		changedModel.digest[0] ^= 1
		changedSampler.model = &changedModel
		for _, node := range []int{0, 1, 6} {
			parents := []socioeconomicParent{}
			if node == 1 {
				parents = append(parents, socioeconomicParent{"industry", 3})
			}
			if node == 6 {
				parents = append(parents, socioeconomicParent{"job-family", 3}, socioeconomicParent{"job-level", 2})
			}
			original, err := w.sampler.stream(42, node, parents...)
			if err != nil {
				t.Fatalf("expected original node stream, got %v", err)
			}
			modified, err := changedSampler.stream(42, node, parents...)
			if err != nil {
				t.Fatalf("expected modified node stream, got %v", err)
			}
			if (original.state == modified.state) != (node != changed) {
				t.Fatalf("expected local fingerprint isolation for %d/%d, got unexpected stream equality", node, changed)
			}
		}
	}

}

const usaCompanyRosterSHA256 = "b6715b2f3208208371d6d867c487f9133cf7a24095f5c8ef3e3e766b1753a28d"

// TestUSACompanyEmployerRoster exercises the complete real roster, canonical
// IDs, ascending/descending generation and a second independent constructor.
func TestUSACompanyEmployerRoster(t *testing.T) {

	categorical := readUSAPersonalCategoricalHarness(t)
	base, model := categorical.base, categorical.companies.model
	w, err := newUSACompanyEmployerHarness(base, model)
	if err != nil {
		t.Fatalf("expected real USA roster, got %v", err)
	}
	replay, err := newUSACompanyEmployerHarness(base, model)
	if err != nil {
		t.Fatalf("expected independent replay constructor, got %v", err)
	}
	if len(w.sampler.companyCounts) != 65536 || len(w.sampler.companyIndustries) != 65536 {
		t.Fatal("expected exactly 65536 Companies, got different cardinality")
	}

	ids := make(map[string]bool, 65536)
	records := make([]Company, 65536)
	var industryCounts [14]uint64
	var bandCounts [5]uint64
	var totalWorkforce uint64
	var canonical []byte
	for coordinate := CompanyIndex(1); coordinate <= 65536; coordinate++ {

		company := usaCompanyEmployerAt(t, w, coordinate)
		industry := slices.Index(socioeconomicIndustries[:], company.Industry)
		band := slices.Index(socioeconomicSizeBands[:], company.SizeBand)
		if industry < 0 || band <= 0 || company.EmployeeCount == 0 ||
			!slices.Contains(model.numeric.employeeCount[industry].values, company.EmployeeCount) ||
			company.SizeBand != socioeconomicExpectedBand(company.EmployeeCount) || ids[company.ID] {
			t.Fatalf("expected unique valid productive Company at %d, got %+v", coordinate, company)
		}

		decoded, err := CompanyIndexFromID(base.namespace, company.ID)
		if err != nil {
			t.Fatalf("expected national Company ID inverse, got %v", err)
		}
		if decoded != coordinate {
			t.Fatalf("expected coordinate %d, got %d", coordinate, decoded)
		}
		if company.EmployeeCount != w.sampler.companyCounts[coordinate-1] ||
			industry != int(w.sampler.companyIndustries[coordinate-1]) {
			t.Fatal("expected materialized metadata equal to random access, got mismatch")
		}

		ids[company.ID] = true
		records[coordinate-1] = company
		industryCounts[industry]++
		bandCounts[band]++
		totalWorkforce += uint64(company.EmployeeCount)
		canonical = fmt.Appendf(canonical, "%d\t%s\t%s\t%d\t%s\n",
			coordinate, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)

	}
	for coordinate := CompanyIndex(65536); coordinate > 0; coordinate-- {
		if company := usaCompanyEmployerAt(t, replay, coordinate); company != records[coordinate-1] {
			t.Fatalf("expected reverse replay at %d, got %+v", coordinate, company)
		}
	}
	for _, coordinate := range []CompanyIndex{65536, 1, 8191, 42, 32769, 16384, 65535, 1009} {
		if company := usaCompanyEmployerAt(t, replay, coordinate); company != records[coordinate-1] {
			t.Fatalf("expected sparse replay at %d, got %+v", coordinate, company)
		}
	}
	for i, count := range industryCounts {
		if count == 0 {
			t.Fatalf("expected represented Industry %d for actual realization, got zero", i)
		}
	}
	for band := 1; band < 5; band++ {
		if bandCounts[band] == 0 {
			t.Fatalf("expected represented positive band %d, got zero", band)
		}
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(canonical))
	if digest != usaCompanyRosterSHA256 {
		t.Fatalf("expected adopted roster replay digest %s, got %s", usaCompanyRosterSHA256, digest)
	}
	t.Logf("roster: count=65536 sha256=%s industries=%v bands=%v workforce=%d",
		digest, industryCounts, bandCounts, totalWorkforce)

}

// TestUSACompanyEmployerTechnicalFixtures preserves zero eligibility, cap
// saturation and strict corruption failure outside productive byte admission.
func TestUSACompanyEmployerTechnicalFixtures(t *testing.T) {

	categorical := readUSAPersonalCategoricalHarness(t)
	base, model := categorical.base, categorical.companies.model

	fixture := *model
	fixture.numeric.employeeCount = make([]socioeconomicLaw, 14)
	for i := range fixture.numeric.employeeCount {
		fixture.numeric.employeeCount[i] = socioeconomicLaw{
			values: []uint32{0, 1, 65535, math.MaxUint32}, cumulative: []uint64{1, 2, 3, 4},
		}
	}
	w, err := newUSACompanyEmployerHarness(base, &fixture)
	if err != nil {
		t.Fatalf("expected zero-valid technical fixture, got %v", err)
	}
	zeros := 0
	for _, count := range w.sampler.companyCounts {
		if count == 0 {
			zeros++
		}
	}
	if zeros == 0 || zeros == 65536 {
		t.Fatalf("expected both zero and positive Companies in fixture, got %d zero", zeros)
	}
	for condition, index := range w.sampler.employers {
		if len(index.companies) != 65536-zeros {
			t.Fatal("expected every positive fixture Company eligible, got different coverage")
		}
		var previous uint64
		for position, coordinate := range index.companies {
			count := w.sampler.companyCounts[coordinate-1]
			industry := w.sampler.companyIndustries[coordinate-1]
			expected := min(uint64(count), 65535) *
				uint64(model.numeric.employer.industryFactors[(condition/4)*14+int(industry)]) * 64
			if count == 0 || index.cumulative[position]-previous != expected {
				t.Fatal("expected zero excluded and cap applied before multiplication, got mismatch")
			}
			previous = index.cumulative[position]
		}
	}
	for i := range fixture.numeric.employeeCount {
		fixture.numeric.employeeCount[i] = socioeconomicLaw{values: []uint32{0}, cumulative: []uint64{1}}
	}
	_, err = newUSACompanyEmployerHarness(base, &fixture)
	usaCompanyEmployerRejected(t, err, nil, "rejected all-zero realized roster")
	_, err = w.company(0)
	usaCompanyEmployerRejected(t, err, ErrInvalidIndex, "rejected zero Company coordinate")
	_, err = w.company(65537)
	usaCompanyEmployerRejected(t, err, ErrCompanyIndexOutOfRange, "rejected Company above 65536")
	for _, coordinate := range []PersonIndex{0, Layer1MaxPersonIndex + 1} {
		_, err := w.employee(coordinate, socioeconomicJobFamilies[0], socioeconomicJobLevels[0])
		if err != nil {
			continue
		}
		t.Fatal("expected rejected invalid Person coordinate, got success")
	}
	_, err = w.employee(42, "", socioeconomicJobLevels[0])
	usaCompanyEmployerRejected(t, err, nil, "rejected empty JobFamily")
	rng := splitMix64{}
	_, err = rng.WeightedIndex([]uint64{math.MaxUint64, 1})
	usaCompanyEmployerRejected(t, err, ErrTotalWeightOverflow, "rejected U64 overflow")
	w.sampler.employers[0] = socioeconomicEmployerIndex{}
	_, err = w.employee(42, socioeconomicJobFamilies[0], socioeconomicJobLevels[0])
	if err != nil {
		return
	}
	t.Fatal("expected rejected corrupt Employer index without fallback, got success")

}

func usaCompanyEmployerAt(t *testing.T, w *usaCompanyEmployerHarness, index CompanyIndex) Company {
	t.Helper()
	company, err := w.company(index)
	if err != nil {
		t.Fatalf("expected USA Company %d, got %v", index, err)
	}
	return company
}

func usaCompanyEmployerRejected(t *testing.T, err, expected error, label string) {

	t.Helper()
	if err != nil {
		if expected != nil && !errors.Is(err, expected) {
			t.Fatalf("expected %s error %v, got %v", label, expected, err)
		}
		return
	}

	t.Fatalf("expected %s, got success", label)

}
