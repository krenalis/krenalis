// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

const (
	usaCompanyEmployerDirectory = "../../synthetic-world-usa-l2-company-employer-demo-v1-CANDIDATE/"
	usaCompanyRosterSHA256      = "b6715b2f3208208371d6d867c487f9133cf7a24095f5c8ef3e3e766b1753a28d"
)

// TestUSACompanyEmployerAdmission checks exact historical byte admission and
// the national interpretation without any complete Income-bearing document.
func TestUSACompanyEmployerAdmission(t *testing.T) {

	assets := readUSACompanyEmployerAssets(t)
	m, err := prepareUSACompanyEmployerCandidate(assets)
	if err != nil {
		t.Fatalf("expected pinned USA Company/Employer admission, got %v", err)
	}
	if m.identity == ([32]byte{}) || m.numeric.companyCount != 65536 || m.numeric.territories != 9 ||
		m.interpretation.Country != "US" || m.interpretation.Dimensions.Territory.SemanticID !=
		"sw-us-2026a-census-division-v1" || len(m.interpretation.Dimensions.Territory.Categories) != 9 ||
		m.interpretation.IncomeUnit != (internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"}) {
		t.Fatalf("expected complete USA admission identity, nine Divisions and USD, got %+v", m.interpretation)
	}
	if m.numeric.companyIndustry.width != 14 || m.numeric.companyIndustry.cumulative[13] != math.MaxUint32 ||
		len(m.numeric.employeeCount) != 14 || m.numeric.employer.cap != 65535 {
		t.Fatal("expected adopted dimensions, cumulative weights and cap, got mismatch")
	}
	for _, law := range m.numeric.employeeCount {
		if len(law.values) != 128 || law.values[0] != 1 || law.values[127] != 32768 ||
			law.cumulative[127] != math.MaxUint32 {
			t.Fatal("expected 128 adopted positive supports and exact weight total, got mismatch")
		}
	}
	for node, fingerprint := range m.fingerprints {
		expected := node == socioeconomicCompanyIndustry || node == socioeconomicCompanyEmployeeCount ||
			node == socioeconomicEmployer
		if (fingerprint != [32]byte{}) != expected {
			t.Fatalf("expected fingerprint presence %t at node %d, got %x", expected, node, fingerprint)
		}
	}
	for _, income := range m.numeric.income {
		if len(income.laws) != 0 || len(income.tupleMap) != 0 {
			t.Fatal("expected no Income dependency, got populated laws")
		}
	}

	// Every byte, including ignored projection metadata, is identity material.
	for i := range assets {
		changed := assets
		changed[i] = bytes.Clone(assets[i])
		changed[i][len(changed[i])-2] ^= 1
		_, err := prepareUSACompanyEmployerCandidate(changed)
		if err != nil {
			continue
		}
		t.Fatalf("expected rejected altered asset %d, got success", i)
	}
	for _, test := range []struct {
		name  string
		asset int
		old   string
		new   string
	}{
		{"approval flag", 2, `"CompanyEmployerOwnerApproved":false`, `"CompanyEmployerOwnerApproved":true`},
		{"NOEMP assumption", 1, `"NOEMPRead":false`, `"NOEMPRead":true`},
		{"source provenance", 1, `f07eef547763a7a21`, `0000000000000000`},
		{"missing Industry", 4, `"industry":"agriculture",`, ``},
		{"reordered Industry", 3, `"agriculture","production-and-utilities"`, `"production-and-utilities","agriculture"`},
		{"count beyond U32", 4, `"values":[1,2,3`, `"values":[4294967296,2,3`},
		{"zero count support", 4, `"values":[1,2,3`, `"values":[0,2,3`},
		{"duplicate count support", 4, `"values":[1,2,3`, `"values":[1,1,3`},
		{"factor beyond domain", 5, `"industryFactors":[[`, `"industryFactors":[[4096,`},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := assets
			changed[test.asset] = bytes.Replace(assets[test.asset], []byte(test.old), []byte(test.new), 1)
			if bytes.Equal(changed[test.asset], assets[test.asset]) {
				t.Fatal("expected effective tampering fixture, got unchanged bytes")
			}
			_, err := prepareUSACompanyEmployerCandidate(changed)
			if err != nil {
				return
			}
			t.Fatal("expected rejected incompatible model, got success")
		})
	}
	for i := range assets {
		partial := assets
		partial[i] = nil
		_, err := prepareUSACompanyEmployerCandidate(partial)
		if err != nil {
			continue
		}
		t.Fatalf("expected rejected partial asset set %d, got success", i)
	}

	pins := map[string]any{}
	for _, pin := range usaCompanyEmployerPins {
		pins[pin.name] = map[string]any{"bytes": pin.bytes, "sha256": pin.sha256}
	}
	writeUSACompanyEmployerEvidence(t, "go-admission.json", map[string]any{
		"schema": "sw-us-company-employer-go-admission-v1", "status": "PASS",
		"identity": fmt.Sprintf("%x", m.identity), "country": m.interpretation.Country,
		"territories": m.numeric.territories, "incomeUnitInterpretationOnly": m.interpretation.IncomeUnit,
		"companyCount": m.numeric.companyCount, "fingerprints": map[string]string{
			"companyIndustry": fmt.Sprintf("%x", m.fingerprints[0]),
			"employeeCount":   fmt.Sprintf("%x", m.fingerprints[1]),
			"employer":        fmt.Sprintf("%x", m.fingerprints[6]),
		}, "pins": pins, "IncomeLaws": 0, "publicWorld": false,
	})

}

// TestUSACompanyEmployerCompilerRejects bounds internal numerical projections
// independently of the outer byte-identity rejection.
func TestUSACompanyEmployerCompilerRejects(t *testing.T) {

	model, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected admitted model, got %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func(*socioeconomicModelJSON)
	}{
		{"zero cardinality", func(d *socioeconomicModelJSON) { d.companyCount = 0 }},
		{"cardinality overflow", func(d *socioeconomicModelJSON) { d.companyCount = math.MaxUint64 }},
		{"missing Industry", func(d *socioeconomicModelJSON) { d.companyIndustry = d.companyIndustry[:13] }},
		{"zero Industry", func(d *socioeconomicModelJSON) { d.companyIndustry[0] = 0 }},
		{"missing law", func(d *socioeconomicModelJSON) { d.employeeCount = d.employeeCount[:13] }},
		{"oversized law", func(d *socioeconomicModelJSON) { d.employeeCount[0].values = make([]uint32, 129) }},
		{"unaligned law", func(d *socioeconomicModelJSON) { d.employeeCount[0].weights = d.employeeCount[0].weights[:127] }},
		{"zero count", func(d *socioeconomicModelJSON) { d.employeeCount[0].values[0] = 0 }},
		{"duplicate support", func(d *socioeconomicModelJSON) { d.employeeCount[0].values[1] = 1 }},
		{"unsorted support", func(d *socioeconomicModelJSON) { d.employeeCount[0].values[2] = 1 }},
		{"excess tail", func(d *socioeconomicModelJSON) { d.employeeCount[0].values[127] = math.MaxUint32 }},
		{"zero weight", func(d *socioeconomicModelJSON) { d.employeeCount[0].weights[0] = 0 }},
		{"noncanonical weight", func(d *socioeconomicModelJSON) {
			for i := range d.employeeCount[0].weights {
				d.employeeCount[0].weights[i] = 2
			}
		}},
		{"different sector law", func(d *socioeconomicModelJSON) { d.employeeCount[1].weights[0]++ }},
		{"zero cap", func(d *socioeconomicModelJSON) { d.employer.cap = 0 }},
		{"wrong cap", func(d *socioeconomicModelJSON) { d.employer.cap = 4096 }},
		{"zero factor", func(d *socioeconomicModelJSON) { d.employer.industryFactors[0] = 0 }},
		{"excess factor", func(d *socioeconomicModelJSON) { d.employer.industryFactors[0] = 4096 }},
		{"wrong size factor", func(d *socioeconomicModelJSON) { d.employer.sizeFactors[0] = 65 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := usaCompanyEmployerNumericalInput(model)
			test.mutate(input)
			_, err := compileUSACompanyEmployer(input)
			if err != nil {
				return
			}
			t.Fatal("expected rejected invalid numerical projection, got success")
		})
	}

	_, err = compileUSACompanyEmployer(nil)
	if err != nil {
		return
	}
	t.Fatal("expected rejected nil numerical projection, got success")

}

// TestUSACompanyEmployerFortyConditions checks intervals and their endpoints
// against independent widened products, then exercises the real EMP sampler.
func TestUSACompanyEmployerFortyConditions(t *testing.T) {

	base := readUSACompletionHarness(t)
	assets := readUSACompanyEmployerAssets(t)
	model, err := prepareUSACompanyEmployerCandidate(assets)
	if err != nil {
		t.Fatalf("expected admitted model, got %v", err)
	}
	w, err := newUSACompanyEmployerHarness(base, model)
	if err != nil {
		t.Fatalf("expected real USA roster, got %v", err)
	}
	var factors struct {
		IndustryFactors [][]uint64 `json:"industryFactors"`
		SizeFactors     [][]uint64 `json:"sizeFactors"`
	}
	err = json.Unmarshal(assets[5], &factors)
	if err != nil {
		t.Fatalf("expected raw adopted factors, got %v", err)
	}

	var conditions []map[string]any
	var samples []byte
	coordinates := []PersonIndex{1, 42, 1009, 65536, Layer1MaxPersonIndex}
	for condition, index := range w.sampler.employers {

		family, level := condition/4, condition%4
		if len(index.companies) != 65536 || len(index.cumulative) != 65536 {
			t.Fatalf("expected all positive Companies eligible at condition %d, got %d", condition, len(index.companies))
		}

		var byIndustry [14]uint64
		var byBand [5]uint64
		weights := make([]uint64, 65536)
		var total uint64
		for offset, count := range w.sampler.companyCounts {

			industry := int(w.sampler.companyIndustries[offset])
			band := slices.Index(socioeconomicSizeBands[:], socioeconomicExpectedBand(count))
			high, weight := bits.Mul64(min(uint64(count), 65535), factors.IndustryFactors[family][industry])
			if high != 0 {
				t.Fatal("expected U64 Industry product, got overflow")
			}
			high, weight = bits.Mul64(weight, factors.SizeFactors[level][band-1])
			if high != 0 || weight == 0 {
				t.Fatal("expected positive U64 Employer weight, got zero or overflow")
			}

			previous := total
			var carry uint64
			total, carry = bits.Add64(total, weight, 0)
			if carry != 0 || index.companies[offset] != CompanyIndex(offset+1) || index.cumulative[offset] != total {
				t.Fatalf("expected exact checked interval ending %d at %d/%d, got mismatch", total, condition, offset)
			}
			for _, draw := range []uint64{previous, total - 1} {
				if got := socioeconomicPosition(index.cumulative, draw); got != offset {
					t.Fatalf("expected strict interval %d for draw %d, got %d", offset, draw, got)
				}
			}
			weights[offset] = weight
			byIndustry[industry] += weight
			byBand[band] += weight

		}
		if total == 0 {
			t.Fatalf("expected positive condition %d total, got zero", condition)
		}
		selections := make([]EmployeeFacts, len(coordinates))
		for sample, coordinate := range coordinates {

			facts, err := w.employee(coordinate, socioeconomicJobFamilies[family], socioeconomicJobLevels[level])
			if err != nil {
				t.Fatalf("expected real Employer selection, got %v", err)
			}
			position := socioeconomicReferenceSelection(t, w.sampler.base, "person", uint64(coordinate),
				usaSocioeconomicPaths[socioeconomicEmployer], model.fingerprints[socioeconomicEmployer], weights,
				mustUint64Component(t, "job-family", uint64(family)), mustUint64Component(t, "job-level", uint64(level)))
			if facts.Employer != CompanyIndex(position+1) || facts.JobFamily != socioeconomicJobFamilies[family] ||
				facts.JobLevel != socioeconomicJobLevels[level] {
				t.Fatalf("expected reference Employer %d, got %+v", position+1, facts)
			}

			selections[sample] = facts
			company := usaCompanyEmployerAt(t, w, facts.Employer)
			if company.Industry != socioeconomicIndustries[w.sampler.companyIndustries[facts.Employer-1]] ||
				company.SizeBand != socioeconomicExpectedBand(w.sampler.companyCounts[facts.Employer-1]) {
				t.Fatal("expected selected Company Industry/SizeBand, got mismatch")
			}
			samples = fmt.Appendf(samples, "%d\t%d\t%d\t%s\t%s\n",
				condition, coordinate, facts.Employer, company.Industry, company.SizeBand)
			// Interleave unrelated Company requests before replaying the employee.
			_ = usaCompanyEmployerAt(t, w, CompanyIndex(65537-uint64(facts.Employer)))
			replayed, err := w.employee(coordinate, facts.JobFamily, facts.JobLevel)
			if err != nil {
				t.Fatalf("expected replayed employee, got %v", err)
			}
			if replayed != facts {
				t.Fatalf("expected replay %+v, got %+v", facts, replayed)
			}

		}
		for sample := len(coordinates) - 1; sample >= 0; sample-- {
			facts, err := w.employee(coordinates[sample], socioeconomicJobFamilies[family], socioeconomicJobLevels[level])
			if err != nil {
				t.Fatalf("expected descending sparse Employer replay, got %v", err)
			}
			if facts != selections[sample] {
				t.Fatal("expected descending Employer replay equality, got mismatch")
			}
		}
		conditions = append(conditions, map[string]any{
			"condition": condition, "familyOrdinal": family, "levelOrdinal": level,
			"status": "PASS", "eligibleCompanies": len(index.companies), "total": total,
			"industryWeights": byIndustry, "sizeBandWeights": byBand,
		})

	}
	if reflect.TypeFor[EmployeeFacts]().NumField() != 3 {
		t.Fatal("expected frozen Employer/JobFamily/JobLevel payload only, got an independent field")
	}
	writeUSACompanyEmployerEvidence(t, "employer-forty-conditions.json", map[string]any{
		"schema": "sw-us-real-roster-employer-conditions-v1", "status": "PASS", "conditions": conditions,
		"intervalsChecked": 40 * 65536, "intervalEndpointsChecked": 2 * 40 * 65536,
		"personCoordinatesPerCondition": coordinates, "personSelections": 40 * len(coordinates),
		"canonicalSampleRecord": "condition TAB PersonIndex TAB EmployerIndex TAB selected Industry TAB selected SizeBand LF",
		"sampleSHA256":          fmt.Sprintf("%x", sha256.Sum256(samples)), "sampleBytes": len(samples),
		"independentEmployeeIndustryDraw": false, "IncomeDependency": false,
	})

}

// TestUSACompanyEmployerIsolation checks seed and identity separation and
// verifies each innovation against the unchanged frozen stream registry.
func TestUSACompanyEmployerIsolation(t *testing.T) {

	base := readUSACompletionHarness(t)
	model, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected admitted model, got %v", err)
	}
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

// TestUSACompanyEmployerRoster exercises the complete real roster, canonical
// IDs, ascending/descending generation and a second independent constructor.
func TestUSACompanyEmployerRoster(t *testing.T) {

	base := readUSACompletionHarness(t)
	model, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected admitted model, got %v", err)
	}
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
	var industryWorkforce [14]uint64
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
		industryWorkforce[industry] += uint64(company.EmployeeCount)
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
	writeUSACompanyEmployerEvidence(t, "roster.json", map[string]any{
		"schema": "sw-us-company-roster-evidence-v1", "status": "PASS", "companyCount": len(ids),
		"seed": 2026, "namespace": base.namespace, "canonicalRecordBytes": len(canonical), "sha256": digest,
		"canonicalRecord": "ascending coordinate, TAB, ID, TAB, Industry machine ID, TAB, " +
			"decimal EmployeeCount, TAB, SizeBand machine ID, LF; UTF-8",
		"industryCounts": industryCounts, "sizeBandCounts": bandCounts, "industryWorkforce": industryWorkforce,
		"totalWorkforce": totalWorkforce, "ascendingDescendingSparseAndIndependentConstruction": true,
		"IncomeDependency": false, "rosterUsedToReplaceD0": false,
	})

}

// TestUSACompanyEmployerTechnicalFixtures preserves zero eligibility, cap
// saturation and strict corruption failure outside productive byte admission.
func TestUSACompanyEmployerTechnicalFixtures(t *testing.T) {

	base := readUSACompletionHarness(t)
	model, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected admitted model, got %v", err)
	}

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

func readUSACompanyEmployerAssets(t *testing.T) usaCompanyEmployerAssets {
	t.Helper()
	var assets usaCompanyEmployerAssets
	for i, pin := range usaCompanyEmployerPins {
		encoded, err := os.ReadFile(usaCompanyEmployerDirectory + pin.name)
		if err != nil {
			t.Fatalf("expected adopted asset %s, got %v", pin.name, err)
		}
		assets[i] = encoded
	}
	return assets
}

func usaCompanyEmployerAt(t *testing.T, w *usaCompanyEmployerHarness, index CompanyIndex) Company {
	t.Helper()
	company, err := w.company(index)
	if err != nil {
		t.Fatalf("expected USA Company %d, got %v", index, err)
	}
	return company
}

func usaCompanyEmployerNumericalInput(m *usaCompanyEmployerCandidate) *socioeconomicModelJSON {

	input := &socioeconomicModelJSON{
		companyCount: m.numeric.companyCount, employer: m.numeric.employer,
		companyIndustry: make([]uint32, 14), employeeCount: make([]socioeconomicLawJSON, 14),
	}
	var previous uint64
	for i, cumulative := range m.numeric.companyIndustry.cumulative {
		input.companyIndustry[i] = uint32(cumulative - previous)
		previous = cumulative
	}

	for i, law := range m.numeric.employeeCount {
		input.employeeCount[i].values = slices.Clone(law.values)
		input.employeeCount[i].weights = make([]uint32, len(law.values))
		previous = 0
		for j, cumulative := range law.cumulative {
			input.employeeCount[i].weights[j] = uint32(cumulative - previous)
			previous = cumulative
		}
	}

	return input
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

func writeUSACompanyEmployerEvidence(t *testing.T, name string, value any) {

	t.Helper()
	directory := os.Getenv("KRENALIS_USA_COMPANY_EMPLOYER_EVIDENCE")
	if directory == "" {
		return
	}

	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("expected encoded evidence, got %v", err)
	}
	err = os.WriteFile(filepath.Join(directory, name), append(encoded, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected saved evidence, got %v", err)
	}

}
