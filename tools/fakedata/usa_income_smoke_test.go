// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

// TestUSAIncomeAdmission checks reviewed original masses, all maps, independent
// fingerprint projections, age boundaries and ownership of compiled arrays.
func TestUSAIncomeAdmission(t *testing.T) {

	assets := readUSAIncomeAssets(t)
	model, err := prepareUSAIncomeCandidate(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected reviewed Income admission, got %v", err)
	}
	doc := readUSAIncomeDocument(t, assets[1])
	unadmitted, err := compileUSAIncome(doc)
	if err != nil {
		t.Fatalf("expected numerical compilation, got %v", err)
	}
	if unadmitted.identity != ([32]byte{}) {
		t.Fatal("expected compilation without authority admission, got admitted identity")
	}
	var nodes []map[string]any
	structural, applicable, sentinel := 0, 0, 0
	for branch, input := range doc.ordered() {

		out := model.numeric.income[branch]
		law := input.Laws[0]
		for slot, id := range input.Map.LawIDs {
			structural++
			if id == nil {
				sentinel++
				if out.tupleMap[slot] != 65535 {
					t.Fatal("expected inapplicable sentinel65535, got productive law")
				}
				continue
			}
			applicable++
			if out.tupleMap[slot] != uint16(*id) {
				t.Fatal("expected original map reference, got changed law")
			}
		}
		var total uint64
		for i, weight := range law.Weights {
			total += uint64(weight)
			if out.laws[0].cumulative[i] != total || out.laws[0].values[i] != law.Supports[i] {
				t.Fatalf("expected original support/weight at %d/%d, got changed numerical asset", branch, i)
			}
		}
		// Independent raw JSON subtree; do not reuse the compiled encoder here.
		ids := make([]uint32, len(input.Map.LawIDs))
		for slot, id := range input.Map.LawIDs {
			ids[slot] = 65535
			if id != nil {
				ids[slot] = *id
			}
		}
		asset, err := internationalCanonical(struct {
			Laws     []internationalLaw `json:"laws"`
			TupleMap []uint32           `json:"tupleMap"`
		}{[]internationalLaw{{Values: law.Supports, Weights: law.Weights}}, ids})
		if err != nil {
			t.Fatalf("expected independently projected Income subtree, got %v", err)
		}
		components := []Component{mustBytesComponent(t, "asset", asset)}
		node := socioeconomicIncomeEmployee + branch
		if age := socioeconomicNodeAges[node]; age >= 0 {
			lower := usaInternationalPartitionBounds()[age]
			compiled := make([]uint32, 55)
			for offset := range 55 {
				for j, bound := range lower {
					if uint32(offset+21) >= bound {
						compiled[offset] = uint32(j)
					}
				}
				if model.numeric.ages[age][offset] != uint8(compiled[offset]) {
					t.Fatalf("expected exact age bucket at consumer %d age %d, got different map", age, offset+21)
				}
			}
			ageMap, err := internationalCanonical(struct {
				Compiled    []uint32 `json:"compiled"`
				LowerBounds []uint32 `json:"lowerBounds"`
			}{compiled, lower})
			if err != nil {
				t.Fatalf("expected independent age projection, got %v", err)
			}
			components = append(components, mustBytesComponent(t, "age-map", ageMap))
		}
		fingerprint, err := canonicalDigest("fakedata/"+usaSocioeconomicPaths[node]+"/model", components)
		if err != nil {
			t.Fatalf("expected local USA fingerprint, got %v", err)
		}
		if fingerprint != model.fingerprints[branch] {
			t.Fatalf("expected independently projected fingerprint %x, got %x", fingerprint, model.fingerprints[branch])
		}
		nodes = append(nodes, map[string]any{"branch": usaIncomeBranchNames[branch], "node": node,
			"path": usaSocioeconomicPaths[node], "fingerprint": fmt.Sprintf("%x", fingerprint),
			"supports": len(law.Supports), "denominator": total})
		input.Laws[0].Supports[1]++
		input.Laws[0].Weights[1]++
		if input.Laws[0].Supports[1] == out.laws[0].values[1] {
			t.Fatal("expected compiled ownership, got input support alias")
		}

	}
	if structural != 34272 || applicable != 29232 || sentinel != 5040 || model.identity == ([32]byte{}) {
		t.Fatalf("expected 34272/29232/5040 admitted slots, got %d/%d/%d", structural, applicable, sentinel)
	}
	clear(assets[1])
	if model.numeric.income[0].laws[0].values[1] != 780 {
		t.Fatal("expected owned prepared Income state, got byte alias")
	}
	var pins []map[string]any
	for _, pin := range usaIncomePins {
		pins = append(pins, map[string]any{"path": pin.path, "bytes": pin.bytes, "sha256": pin.sha256})
	}
	writeUSAIncomeEvidence(t, "admission.json", map[string]any{
		"status": "PASS", "state": "TEST-ONLY INTERNAL USA CANDIDATE", "nodes": nodes,
		"structuralSlots": structural, "applicableSlots": applicable, "sentinelSlots": sentinel,
		"admissionReceiptIdentity": fmt.Sprintf("%x", model.identity), "pins": pins,
		"wholeModelIdentityCreated": false, "publicWorld": false,
	})

}

// TestUSAIncomeAdmissionRejects tests every pin and missing asset, including
// altered approval/authority/branch bytes, before any numerical admission.
func TestUSAIncomeAdmissionRejects(t *testing.T) {

	assets := readUSAIncomeAssets(t)
	for i := range usaIncomePins {
		t.Run(fmt.Sprintf("digest-%d", i), func(t *testing.T) {
			changed := assets
			changed[i] = bytes.Clone(assets[i])
			changed[i][0] ^= 1
			model, err := prepareUSAIncomeCandidate(t.Context(), changed)
			if model != nil {
				t.Fatal("expected no changed candidate, got admission")
			}
			usaCompanyEmployerRejected(t, err, nil, "changed Income pin rejection")
		})
		t.Run(fmt.Sprintf("missing-%d", i), func(t *testing.T) {
			changed := assets
			changed[i] = nil
			model, err := prepareUSAIncomeCandidate(t.Context(), changed)
			if model != nil {
				t.Fatal("expected no incomplete candidate, got admission")
			}
			usaCompanyEmployerRejected(t, err, nil, "missing Income pin rejection")
		})
	}
	changed := assets
	changed[1] = bytes.Replace(assets[1], []byte(`"ownerApprovedLaws":false`), []byte(`"ownerApprovedLaws":true`), 1)
	_, err := prepareUSAIncomeCandidate(t.Context(), changed)
	usaCompanyEmployerRejected(t, err, nil, "self-declared approval rejection")

}

// TestUSAIncomeCompilerRejects exercises semantic/numerical checks beneath
// the outer digest boundary, using only deliberate private negative fixtures.
func TestUSAIncomeCompilerRejects(t *testing.T) {

	assets := readUSAIncomeAssets(t)
	cases := []struct {
		name   string
		change func(*usaIncomeDocument)
	}{
		{"stage", func(d *usaIncomeDocument) { d.Stage = 1 }},
		{"public", func(d *usaIncomeDocument) { d.PubliclyAdmissibleSocioeconomicJSON = true }},
		{"approval", func(d *usaIncomeDocument) { d.OwnerApprovedLaws = true }},
		{"missing-law", func(d *usaIncomeDocument) { d.Branches.Employee.Laws = nil }},
		{"extra-law", func(d *usaIncomeDocument) {
			d.Branches.Employee.Laws = append(d.Branches.Employee.Laws, d.Branches.Employee.Laws[0])
		}},
		{"law-id", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].ID = 1 }},
		{"duplicate-support", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].Supports[2] = 780 }},
		{"unsorted-support", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].Supports[2] = 1 }},
		{"missing-support", func(d *usaIncomeDocument) {
			d.Branches.Employee.Laws[0].Supports = d.Branches.Employee.Laws[0].Supports[:384]
		}},
		{"zero-weight", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].Weights[1] = 0 }},
		{"denominator", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].Total++ }},
		{"weight-change", func(d *usaIncomeDocument) { d.Branches.Employee.Laws[0].Weights[1]++ }},
		{"gcd", func(d *usaIncomeDocument) {
			for i := range d.Branches.Employee.Laws[0].Weights {
				d.Branches.Employee.Laws[0].Weights[i] = 2
			}
		}},
		{"missing-map", func(d *usaIncomeDocument) { d.Branches.Employee.Map.LawIDs = nil }},
		{"zero-law-sentinel", func(d *usaIncomeDocument) { v := uint32(0); d.Branches.Employee.Map.LawIDs[0] = &v }},
		{"missing-valid-slot", func(d *usaIncomeDocument) { d.Branches.Employee.Map.LawIDs[1] = nil }},
		{"invalid-sentinel", func(d *usaIncomeDocument) { v := uint32(65535); d.Branches.Employee.Map.LawIDs[1] = &v }},
		{"wrong-law", func(d *usaIncomeDocument) { v := uint32(1); d.Branches.Student.Map.LawIDs[0] = &v }},
		{"wrong-branch", func(d *usaIncomeDocument) { d.Branches.Student = d.Branches.Retired }},
		{"wrong-axes", func(d *usaIncomeDocument) { d.Branches.SelfEmployed.Map.Axes[0] = "Education" }},
		{"wrong-shape", func(d *usaIncomeDocument) { d.Branches.SelfEmployed.Map.AxisSizes[0] = 5 }},
		{"canonical-identity", func(d *usaIncomeDocument) { d.Branches.Employee.CanonicalIdentitySHA256 = "bad" }},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			document := readUSAIncomeDocument(t, assets[1])
			test.change(document)
			model, err := compileUSAIncome(document)
			if model != nil {
				t.Fatal("expected invalid fixture rejected, got numerical candidate")
			}
			usaCompanyEmployerRejected(t, err, nil, "invalid numerical Income rejection")
		})
	}
	for _, replacement := range []string{"4294967296", "18446744073709551616", "-1", "0.5"} {
		t.Run("decode-width-"+replacement, func(t *testing.T) {
			encoded := bytes.Replace(assets[1], []byte(`"supports":[0,`), []byte(`"supports":[`+replacement+`,`), 1)
			var document usaIncomeDocument
			err := jsonv2.Unmarshal(encoded, &document, jsonv2.RejectUnknownMembers(true))
			usaCompanyEmployerRejected(t, err, nil, "invalid support representation rejection")
		})
	}
	for _, encoded := range [][]byte{
		append([]byte(`{"stage":2,`), assets[1][1:]...),
		append([]byte(`{"unknown":true,`), assets[1][1:]...),
		append(bytes.Clone(assets[1]), []byte(`{}`)...),
	} {
		var document usaIncomeDocument
		err := jsonv2.Unmarshal(encoded, &document, jsonv2.RejectUnknownMembers(true))
		usaCompanyEmployerRejected(t, err, nil, "closed bounded Income JSON")
	}

}

// TestUSAIncomeIsolation verifies all six local streams and non-evaluation of
// inapplicable branches, plus actual consultation of the addressed tuple map.
func TestUSAIncomeIsolation(t *testing.T) {

	w := readUSAIncomeHarness(t)
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
		err = changed.prepareFingerprints()
		if err != nil {
			t.Fatalf("expected private perturbation fingerprint, got %v", err)
		}
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

// TestUSAIncomeReplay generates actual complete technical records and checks
// fresh-model reverse/sparse/boundary replay, without exporting Person rows.
func TestUSAIncomeReplay(t *testing.T) {

	a := readUSAIncomeHarness(t)
	doc := readUSAIncomeDocument(t, readUSAIncomeAssets(t)[1])
	oracle := newUSAPersonalCategoricalOracle(t, a.categorical)
	first := map[PersonIndex][sha256.Size]byte{}
	canonical, categorical := sha256.New(), sha256.New()
	var counts [6]int
	var histograms [6][]int
	for branch := range histograms {
		histograms[branch] = make([]int, 385)
	}
	companies := map[CompanyIndex]bool{}
	for coordinate := PersonIndex(1); coordinate <= 20000; coordinate++ {
		r, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected actual Income Person %d, got %v", coordinate, err)
		}
		oracle.check(t, a.categorical, r.categorical)
		checkUSAIncomeRecord(t, a, r, doc)
		encoded := usaIncomeTuple(t, r)
		first[coordinate] = sha256.Sum256(encoded)
		_, _ = canonical.Write(encoded)
		_, _ = categorical.Write(usaPersonalCategoricalTuple(t, r.categorical))
		counts[r.trace.branch]++
		histograms[r.trace.branch][r.trace.position]++
		if r.categorical.employee != nil {
			companies[r.categorical.employee.Employer] = true
		}
	}
	if fmt.Sprintf("%x", categorical.Sum(nil)) != usaPersonalCategoricalReplaySHA256 {
		t.Fatal("expected historical complete categorical/Layer1 replay digest, got perturbation")
	}
	if slices.Contains(counts[:], 0) {
		t.Fatalf("expected all six actual Income branches, got %v", counts)
	}
	b := readUSAIncomeHarness(t)
	if a.income == b.income || a.categorical.model == b.categorical.model ||
		a.categorical.companies == b.categorical.companies {
		t.Fatal("expected fresh admission and independent Company preparation, got reused model")
	}
	for coordinate := PersonIndex(20000); coordinate > 0; coordinate-- {
		r, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected descending fresh-model replay, got %v", err)
		}
		if sha256.Sum256(usaIncomeTuple(t, r)) != first[coordinate] {
			t.Fatalf("expected zero mismatch at Person %d, got changed complete tuple", coordinate)
		}
	}
	var sparse []PersonIndex
	sparseHash := sha256.New()
	for i := 1; i <= 1000; i++ {
		coordinate := PersonIndex(20001 + uint64(i)*(uint64(Layer1MaxPersonIndex)-20001)/1000)
		sparse = append(sparse, coordinate)
		r, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected sparse Income coordinate, got %v", err)
		}
		oracle.check(t, a.categorical, r.categorical)
		checkUSAIncomeRecord(t, a, r, doc)
		encoded := usaIncomeTuple(t, r)
		first[coordinate] = sha256.Sum256(encoded)
		_, _ = sparseHash.Write(encoded)
	}
	for i := range 1000 {
		coordinate := sparse[(i*337)%1000]
		r, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected permuted sparse Income coordinate, got %v", err)
		}
		if sha256.Sum256(usaIncomeTuple(t, r)) != first[coordinate] {
			t.Fatalf("expected sparse replay at %d, got mismatch", coordinate)
		}
	}
	boundaries := []PersonIndex{1, 20000, Layer1MaxPersonIndex - 1, Layer1MaxPersonIndex}
	for _, coordinate := range boundaries {
		left, err := a.person(coordinate)
		if err != nil {
			t.Fatalf("expected valid boundary Income coordinate, got %v", err)
		}
		right, err := b.person(coordinate)
		if err != nil {
			t.Fatalf("expected fresh boundary replay, got %v", err)
		}
		checkUSAIncomeRecord(t, a, left, doc)
		if !reflect.DeepEqual(left, right) {
			t.Fatalf("expected complete boundary equality at %d, got mismatch", coordinate)
		}
	}
	roster := sha256.New()
	for index := CompanyIndex(1); index <= 65536; index++ {
		company := usaCompanyEmployerAt(t, a.categorical.companies, index)
		otherIndex := 65537 - index
		other := usaCompanyEmployerAt(t, b.categorical.companies, otherIndex)
		if other.EmployeeCount != a.categorical.companies.sampler.companyCounts[otherIndex-1] ||
			other.Industry != socioeconomicIndustries[a.categorical.companies.sampler.companyIndustries[otherIndex-1]] {
			t.Fatal("expected independent historical Company roster, got mismatch")
		}
		_, _ = fmt.Fprintf(roster, "%d\t%s\t%s\t%d\t%s\n",
			index, company.ID, company.Industry, company.EmployeeCount, company.SizeBand)
	}
	if fmt.Sprintf("%x", roster.Sum(nil)) != usaCompanyRosterSHA256 {
		t.Fatal("expected intact historical roster digest, got mismatch")
	}
	digest := fmt.Sprintf("%x", canonical.Sum(nil))
	t.Logf("persons=20000 replay=42008 mismatch=0 income-sha256=%s branches=%v selected-companies=%d",
		digest, counts, len(companies))
	writeUSAIncomeEvidence(t, "replay.json", map[string]any{
		"status": "PASS", "seed": 2026, "namespace": a.categorical.base.namespace, "generation": 1,
		"companyCount": 65536, "companyRosterSHA256": usaCompanyRosterSHA256, "distinctSelectedCompanies": len(companies),
		"prefixCoordinates": 20000, "prefixSHA256": digest,
		"categoricalPrefixSHA256": fmt.Sprintf("%x", categorical.Sum(nil)),
		"prefixOrders":            []string{"ascending-A", "descending-fresh-B"}, "sparseCoordinates": 1000,
		"sparseSHA256": fmt.Sprintf("%x", sparseHash.Sum(nil)),
		"sparseOrders": []string{"ascending-A", "permutation337-fresh-B"},
		"boundaries":   boundaries, "personCallsInReplayTest": 42008, "distinctValidCoordinates": 21001, "mismatches": 0,
		"branchOrder": usaIncomeBranchNames, "branchCounts": counts, "supportHistograms": histograms,
		"empiricalStatistics":      "DIAGNOSTIC ONLY; histogram counts do not replace D0 or national evidence",
		"fullModelIdentityCreated": false, "publicWorld": false, "sourcePersonRowsExported": false,
	})

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

// checkUSAIncomeRecord reconstructs parent addresses and a linear cumulative
// selection from the original masses, independently of the harness sampler.
func checkUSAIncomeRecord(t *testing.T, w *usaIncomeHarness, record usaIncomeRecord, document *usaIncomeDocument) {

	t.Helper()
	r := record.categorical
	branch := slices.Index(socioeconomicActivities[:], r.activity)
	division := int(r.layer1.divisionOrdinal)
	education := slices.Index(socioeconomicEducationLevels[:], r.education)
	var parents []socioeconomicParent
	var tuple int
	switch branch {
	case 0:
		family := slices.Index(socioeconomicJobFamilies[:], r.employee.JobFamily)
		level := slices.Index(socioeconomicJobLevels[:], r.employee.JobLevel)
		industry := slices.Index(socioeconomicIndustries[:], r.company.Industry)
		band := slices.Index(socioeconomicSizeBands[:], r.company.SizeBand)
		tuple = ((((division*10)+family)*4+level)*14+industry)*5 + band
		parents = []socioeconomicParent{
			{"region", division}, {"job-family", family}, {"job-level", level}, {"industry", industry}, {"size-band", band},
		}
	case 1:
		family := slices.Index(socioeconomicJobFamilies[:], r.selfEmployed.JobFamily)
		industry := slices.Index(socioeconomicIndustries[:], r.selfEmployed.Industry)
		bucket := 0
		for i, bound := range []int{21, 25, 35, 45, 55, 65} {
			if r.layer1.anchorAge >= bound {
				bucket = i
			}
		}
		tuple = ((bucket*9+division)*10+family)*14 + industry
		parents = []socioeconomicParent{
			{"age-bucket", bucket}, {"region", division}, {"job-family", family}, {"industry", industry},
		}
	default:
		bounds := [4][]int{
			{21, 25, 35, 45, 55, 65}, {21, 25, 30, 40, 55, 65}, {21, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65},
		}
		bucket := 0
		for i, bound := range bounds[branch-2] {
			if r.layer1.anchorAge >= bound {
				bucket = i
			}
		}
		tuple = (bucket*7+education)*9 + division
		parents = []socioeconomicParent{{"age-bucket", bucket}, {"education", education}, {"region", division}}
	}
	input := document.ordered()[branch]
	id := input.Map.LawIDs[tuple]
	if id == nil || *id != 0 || record.trace.branch != branch || record.trace.tuple != tuple || record.trace.law != 0 ||
		!reflect.DeepEqual(record.trace.parents, parents) {
		t.Fatal("expected original branch/tuple map and closed ordered parents, got wrong Income address")
	}
	components := []Component{mustBytesComponent(t, "model", w.income.fingerprints[branch][:])}
	for _, parent := range parents {
		components = append(components, mustUint64Component(t, parent.name, uint64(parent.value)))
	}
	rng, err := w.categorical.base.streams.entityStream("person", uint64(r.coordinate),
		usaSocioeconomicPaths[socioeconomicIncomeEmployee+branch], components...)
	if err != nil {
		t.Fatalf("expected frozen Income stream, got %v", err)
	}
	before := rng.state
	law := input.Laws[0]
	draw, err := rng.Uint64n(law.Total)
	if err != nil {
		t.Fatalf("expected one bounded Income draw, got %v", err)
	}
	var cumulative uint64
	position := -1
	for i, mass := range law.Weights {
		cumulative += uint64(mass)
		if draw < cumulative {
			position = i
			break
		}
	}
	if position < 0 || annualIncomeUSD(law.Supports[position]) != record.income || record.trace.position != position ||
		record.trace.before != before || record.trace.after != rng.state || record.trace.draw != draw ||
		record.trace.total != law.Total {
		t.Fatal("expected original-law linear selection and one frozen stream draw, got Income mismatch")
	}

}

func readUSAIncomeAssets(t *testing.T) usaIncomeAssets {
	t.Helper()
	var result usaIncomeAssets
	for i, pin := range usaIncomePins {
		data, err := os.ReadFile("../../" + pin.path)
		if err != nil {
			t.Fatalf("expected reviewed Income asset %s, got %v", pin.path, err)
		}
		result[i] = data
	}
	return result
}

func readUSAIncomeDocument(t *testing.T, data []byte) *usaIncomeDocument {
	t.Helper()
	var document usaIncomeDocument
	err := jsonv2.Unmarshal(data, &document, jsonv2.RejectUnknownMembers(true))
	if err != nil {
		t.Fatalf("expected original closed stage2 Income JSON, got %v", err)
	}
	return &document
}

func readUSAIncomeHarness(t *testing.T) *usaIncomeHarness {

	t.Helper()
	assets := readUSAIncomeAssets(t)
	income, err := prepareUSAIncomeCandidate(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected reviewed internal Income admission, got %v", err)
	}
	// The categorical admission consumes its exact historical authority snapshot.
	// The entire current register and its authorized append are already pinned
	// above; its old prefix is byte-identical and rechecked by the old admission.
	personal := readUSAPersonalCategoricalAssets(t)
	personal[0] = bytes.Clone(assets[0][:usaPersonalCategoricalPins[0].bytes])
	model, err := prepareUSAPersonalCategoricalCandidate(personal)
	if err != nil {
		t.Fatalf("expected unchanged historical categorical admission, got %v", err)
	}
	base := readUSACompletionHarness(t)
	companyModel, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected unchanged Company/Employer admission, got %v", err)
	}
	companies, err := newUSACompanyEmployerHarness(base, companyModel)
	if err != nil {
		t.Fatalf("expected historical Company roster, got %v", err)
	}
	categorical, err := newUSAPersonalCategoricalHarness(base, model, companies)
	if err != nil {
		t.Fatalf("expected existing categorical DAG join, got %v", err)
	}

	return &usaIncomeHarness{categorical: categorical, income: income}
}

func usaIncomeTuple(t *testing.T, record usaIncomeRecord) []byte {

	t.Helper()

	var parents []any
	for _, parent := range record.trace.parents {
		parents = append(parents, []any{parent.name, parent.value})
	}
	trace := record.trace
	categorical := bytes.TrimSuffix(usaPersonalCategoricalTuple(t, record.categorical), []byte{'\n'})
	encoded, err := json.Marshal([]any{json.RawMessage(categorical), record.income, []any{
		socioeconomicIncomeEmployee + trace.branch, trace.tuple, trace.law, parents, trace.position,
		trace.before, trace.after, trace.draw, trace.total,
	}})
	if err != nil {
		t.Fatalf("expected canonical technical Income tuple, got %v", err)
	}

	return append(encoded, '\n')
}

func writeUSAIncomeEvidence(t *testing.T, name string, value any) {

	t.Helper()

	directory := os.Getenv("KRENALIS_USA_INCOME_EVIDENCE")
	if directory == "" {
		return
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("expected aggregate Income evidence, got %v", err)
	}
	err = os.WriteFile(filepath.Join(directory, name), append(encoded, '\n'), 0o644)
	if err != nil {
		t.Fatalf("expected saved aggregate Income evidence, got %v", err)
	}

}
