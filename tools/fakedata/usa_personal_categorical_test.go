// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestUSAPersonalCategoricalAdmission verifies adopted bytes, independent
// fingerprints, all rows and ownership of the compiled tables.
func TestUSAPersonalCategoricalAdmission(t *testing.T) {

	assets := readUSAPersonalCategoricalAssets(t)
	m, err := prepareUSAPersonalCategoricalCandidate(assets)
	if err != nil {
		t.Fatalf("expected six admitted USA models, got %v", err)
	}
	input := readUSAPersonalCategoricalInput(t, assets)
	rows, weights := 0, 0
	var nodes []map[string]any
	for i, table := range m.tables {

		rows += len(table.cumulative) / table.width
		weights += len(table.cumulative)
		var zeros int
		for row := range len(table.cumulative) / table.width {
			var total uint64
			for outcome := range table.width {
				weight := input[i].Weights[row*table.width+outcome]
				total += uint64(weight)
				if weight == 0 {
					zeros++
				}
				if table.cumulative[row*table.width+outcome] != total {
					t.Fatalf("expected original U32 masses with row-local U64 sums, got node %d row %d", i, row)
				}
			}
		}
		// Independent JSON projection; no compiled subtree encoder is used here.
		asset, err := internationalCanonical(struct {
			Weights []uint32 `json:"weights"`
		}{input[i].Weights})
		if err != nil {
			t.Fatalf("expected canonical raw component, got %v", err)
		}
		components := []Component{mustBytesComponent(t, "asset", asset)}
		node := usaPersonalCategoricalNodes[i]
		if consumer := socioeconomicNodeAges[node]; consumer >= 0 {
			bounds := usaInternationalPartitionBounds()[consumer]
			var compiled []uint32
			for age := 21; age <= 75; age++ {
				bucket := 0
				for j, lower := range bounds {
					if age >= int(lower) {
						bucket = j
					}
				}
				compiled = append(compiled, uint32(bucket))
			}
			ageMap, err := internationalCanonical(struct {
				Compiled    []uint32 `json:"compiled"`
				LowerBounds []uint32 `json:"lowerBounds"`
			}{compiled, bounds})
			if err != nil {
				t.Fatalf("expected canonical independent age map, got %v", err)
			}
			components = append(components, mustBytesComponent(t, "age-map", ageMap))
		}
		fingerprint, err := canonicalDigest("fakedata/"+usaSocioeconomicPaths[node]+"/model", components)
		if err != nil {
			t.Fatalf("expected canonical local fingerprint, got %v", err)
		}
		if fingerprint != m.fingerprints[i] {
			t.Fatalf("expected independent fingerprint %x, got %x", fingerprint, m.fingerprints[i])
		}
		nodes = append(nodes, map[string]any{
			"node": node, "path": usaSocioeconomicPaths[node], "rows": len(table.cumulative) / table.width,
			"width": table.width, "weights": len(table.cumulative), "zeroWeights": zeros,
			"fingerprint": fmt.Sprintf("%x", fingerprint), "axes": input[i].Axes,
		})

	}
	if rows != 1114 || weights != 6482 {
		t.Fatalf("expected 1114 rows and 6482 weights, got %d/%d", rows, weights)
	}
	var before [6][]uint64
	for i, table := range m.tables {
		before[i] = slices.Clone(table.cumulative)
	}
	for i := range input {
		input[i].Weights[0]++
		clear(assets[i+4])
	}
	for i, table := range m.tables {
		if !slices.Equal(before[i], table.cumulative) {
			t.Fatalf("expected owned prepared masses at node %d, got mutable source alias", i)
		}
	}
	writeUSAPersonalCategoricalEvidence(t, "admission.json", map[string]any{
		"status": "PASS", "rows": rows, "weights": weights, "models": nodes,
		"admissionIdentity": fmt.Sprintf("%x", m.identity), "pins": usaPersonalCategoricalPinsForEvidence(),
		"wholeModelIdentityCreated": false, "publicWorld": false, "incomeLaws": 0,
	})

}

// TestUSAPersonalCategoricalAdmissionRejects exercises byte admission, rather
// than trusting historical approval flags or accepting altered JSON shapes.
func TestUSAPersonalCategoricalAdmissionRejects(t *testing.T) {

	assets := readUSAPersonalCategoricalAssets(t)
	for i, pin := range usaPersonalCategoricalPins {
		t.Run(fmt.Sprintf("pin-%d", i), func(t *testing.T) {
			changed := assets
			changed[i] = slices.Clone(assets[i])
			changed[i][0] ^= 1
			m, err := prepareUSAPersonalCategoricalCandidate(changed)
			if m != nil {
				t.Fatalf("expected no candidate for changed %s, got %v", pin.path, m)
			}
			usaCompanyEmployerRejected(t, err, nil, "rejection of changed pinned bytes")
		})
	}
	for _, test := range []struct {
		name   string
		asset  int
		change func(map[string]any)
	}{
		{"country", 5, func(d map[string]any) { d["country"] = "IT" }},
		{"approval-flag", 5, func(d map[string]any) { d["ownerApproved"] = true }},
		{"extra-income", 4, func(d map[string]any) { d["income"] = 0 }},
		{"missing-row", 5, func(d map[string]any) { d["primaryActivity"] = d["primaryActivity"].([]any)[1:] }},
		{"inverted-axes", 6, func(d map[string]any) { d["axes"] = []string{"territory", "education"} }},
		{"u32-overflow", 4, func(d map[string]any) { d["weights"].([]any)[0] = uint64(math.MaxUint32) + 1 }},
		{"negative-weight", 4, func(d map[string]any) { d["weights"].([]any)[0] = -1 }},
		{"fractional-weight", 4, func(d map[string]any) { d["weights"].([]any)[0] = 0.5 }},
		{"ordinal", 6, func(d map[string]any) { d["axisSizes"] = []int{7, 10} }},
		{"age-partition", 7, func(d map[string]any) { d["ageLowerBounds"] = []int{21, 30, 35, 45, 55, 65} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var document map[string]any
			err := json.Unmarshal(assets[test.asset], &document)
			if err != nil {
				t.Fatalf("expected source JSON, got %v", err)
			}
			test.change(document)
			encoded, err := json.Marshal(document)
			if err != nil {
				t.Fatalf("expected mutated JSON, got %v", err)
			}
			changed := assets
			changed[test.asset] = encoded
			m, err := prepareUSAPersonalCategoricalCandidate(changed)
			if m != nil {
				t.Fatal("expected no altered candidate, got admission")
			}
			usaCompanyEmployerRejected(t, err, nil, "closed pinned admission")
		})
	}

}

// TestUSAPersonalCategoricalAgePartitions checks every age, including all
// consumer boundaries, independently of the derived birth-date observation.
func TestUSAPersonalCategoricalAgePartitions(t *testing.T) {

	m := readUSAPersonalCategoricalModel(t)
	bounds := [3][]int{{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65}}
	for consumer, lower := range bounds {
		for age := 21; age <= 75; age++ {
			want := 0
			for i, boundary := range lower {
				if age >= boundary {
					want = i
				}
			}
			if got := int(m.ages[consumer][age-21]); got != want {
				t.Fatalf("expected consumer %d age %d bucket %d, got %d", consumer, age, want, got)
			}
		}
	}
	m.ages[0][0]++
	if m.ages[2][0] != 0 {
		t.Fatal("expected separately owned consumer maps, got mutable alias")
	}

}

// TestUSAPersonalCategoricalCompilerRejects checks the numeric boundary even
// for typed input that cannot obtain a pinned admission identity.
func TestUSAPersonalCategoricalCompilerRejects(t *testing.T) {

	assets := readUSAPersonalCategoricalAssets(t)
	for _, test := range []struct {
		name   string
		change func(*[6]internationalCategorical)
	}{
		{"missing-row", func(d *[6]internationalCategorical) { d[0].Weights = d[0].Weights[:371] }},
		{"inverted-axes", func(d *[6]internationalCategorical) { slices.Reverse(d[2].Axes) }},
		{"wrong-outcome", func(d *[6]internationalCategorical) { d[3].Outcome = "industry" }},
		{"empty-supported-outcome", func(d *[6]internationalCategorical) { d[5].Weights[0] = 0 }},
		{"positive-structural-zero", func(d *[6]internationalCategorical) { d[0].Weights[4] = 1 }},
		{"noncanonical-gcd", func(d *[6]internationalCategorical) {
			for i := range 14 {
				d[5].Weights[i] = 2
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := readUSAPersonalCategoricalInput(t, assets)
			test.change(&input)
			m, err := compileUSAPersonalCategorical(input)
			if m != nil {
				t.Fatal("expected no invalid numeric model, got model")
			}
			usaCompanyEmployerRejected(t, err, nil, "invalid numeric input")
		})
	}
	for _, bad := range []string{"4294967296", "18446744073709551616", "-1", "1.5", "null"} {
		t.Run("u32-"+bad, func(t *testing.T) {
			var value internationalCategorical
			encoded := strings.Replace(string(assets[4]), "46823542", bad, 1)
			err := decodeInternationalJSON(t.Context(), strings.NewReader(encoded), &value)
			usaCompanyEmployerRejected(t, err, nil, "invalid U32 decoding")
		})
	}
	input := readUSAPersonalCategoricalInput(t, assets)
	for i := range 14 {
		input[5].Weights[i] = math.MaxUint32
	}
	input[5].Weights[0]--
	m, err := compileUSAPersonalCategorical(input)
	if err != nil {
		t.Fatalf("expected bounded U64 total larger than U32, got %v", err)
	}
	if want, got := uint64(14)*math.MaxUint32-1, m.tables[5].cumulative[13]; got != want {
		t.Fatalf("expected exact maximum canonical row total %d, got %d", want, got)
	}
	if m.identity != ([32]byte{}) {
		t.Fatal("expected no admission identity for typed compiler input, got identity")
	}

}

// TestUSAPersonalCategoricalPublicBoundaries rejects partial Worlds and absent
// Income through the actual existing constructors and runtime methods.
func TestUSAPersonalCategoricalPublicBoundaries(t *testing.T) {

	w := readUSAPersonalCategoricalHarness(t)
	_, err := newUSAPersonalCategoricalHarness(w.base, w.model, nil)
	usaCompanyEmployerRejected(t, err, nil, "USA model without Employer")
	missing := *w.companies
	missing.sampler.employers[39] = socioeconomicEmployerIndex{}
	_, err = newUSAPersonalCategoricalHarness(w.base, w.model, &missing)
	usaCompanyEmployerRejected(t, err, nil, "missing Employer condition")
	_, err = NewSocioeconomicWorld(w.companies.sampler.base, w.companies.sampler.model)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial World")
	_, err = w.companies.sampler.Facts(1)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial Facts")
	_, err = w.companies.sampler.Person(1)
	usaCompanyEmployerRejected(t, err, ErrInvalidSocioeconomicWorld, "public partial Person")
	_, err = w.companies.sampler.income(1, 0, 0)
	usaCompanyEmployerRejected(t, err, nil, "absent Income used as a valid zero law")
	unknown, err := entityIDForIndex(w.base.namespace, "company", 65537)
	if err != nil {
		t.Fatalf("expected representable nonmember ID, got %v", err)
	}
	_, err = w.companyByID(unknown)
	usaCompanyEmployerRejected(t, err, ErrCompanyIndexOutOfRange, "unknown operational CompanyID")
	for _, index := range []PersonIndex{0, Layer1MaxPersonIndex + 1, PersonIndex(math.MaxUint64)} {
		record, err := w.person(index)
		if !reflect.DeepEqual(record, usaPersonalCategoricalRecord{}) {
			t.Fatal("expected no partial record on invalid coordinate, got record")
		}
		usaCompanyEmployerRejected(t, err, nil, "invalid Person coordinate")
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[usaPersonalCategoricalCandidate](), reflect.TypeFor[usaPersonalCategoricalRecord](),
	} {
		for i := range typ.NumField() {
			if strings.Contains(strings.ToLower(typ.Field(i).Name), "income") {
				t.Fatal("expected no Income placeholder field, got Income field")
			}
		}
	}

	// An otherwise complete national categorical projection still cannot pass
	// the existing full-model admission without all six monetary law families.
	d := internationalSocioeconomic{
		Schema: "sw-socioeconomic-data-v1", Release: "us-categorical-test", Profile: usaSocioeconomicProfile,
		Country: "US", AnchorDate: usaSocioeconomicAnchorDate, AgeDomain: internationalAgeDomain{21, 75},
		CompanyCount: 65536, Dimensions: usaInternationalDimensions(),
		IncomeUnit: internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"},
	}
	for i, partition := range d.AgePartitions.ordered() {
		partition.LowerBounds = usaInternationalPartitionBounds()[i]
	}
	input := readUSAPersonalCategoricalInput(t, readUSAPersonalCategoricalAssets(t))
	for i, categorical := range d.Models.categorical() {
		if i == 0 {
			categorical.Axes = []string{}
			categorical.Outcome = "industry"
			categorical.Weights = usaCompanyEmployerNumericalInput(w.companies.model).companyIndustry
			continue
		}
		*categorical = input[i-1]
	}
	d.Models.EmployeeCount.Axes = []string{"industry"}
	for _, law := range usaCompanyEmployerNumericalInput(w.companies.model).employeeCount {
		d.Models.EmployeeCount.Laws = append(d.Models.EmployeeCount.Laws, internationalLaw{
			Values: law.values, Weights: law.weights,
		})
	}
	d.Models.Employer.WorkforceCap = 65535
	d.Models.Employer.IndustryFactors.Axes = []string{"jobFamily", "industry"}
	d.Models.Employer.SizeFactors.Axes = []string{"jobLevel", "positiveSizeBand"}
	for _, factor := range w.companies.model.numeric.employer.industryFactors {
		d.Models.Employer.IndustryFactors.Values = append(d.Models.Employer.IndustryFactors.Values, uint32(factor))
	}
	for _, factor := range w.companies.model.numeric.employer.sizeFactors {
		d.Models.Employer.SizeFactors.Values = append(d.Models.Employer.SizeFactors.Values, uint32(factor))
	}
	_, err = prepareUSASocioeconomic(&d)
	if err != nil {
		if !strings.Contains(err.Error(), "Income branch 0") {
			t.Fatalf("expected specific missing Income rejection, got %v", err)
		}
		return
	}

	t.Fatal("expected complete-model rejection without Income, got admission")

}

// TestUSAPersonalCategoricalRegistrySnapshot checks historical fixture bytes
// independently of later appendices and preserves strict preparer admission.
func TestUSAPersonalCategoricalRegistrySnapshot(t *testing.T) {

	pin := usaPersonalCategoricalPins[0]
	current, err := os.ReadFile("../../" + pin.path)
	if err != nil {
		t.Fatalf("expected current registry, got %v", err)
	}
	if len(current) < pin.bytes {
		t.Fatalf("expected at least %d registry bytes, got %d", pin.bytes, len(current))
	}

	changed := slices.Clone(current)
	changed[pin.bytes/2] ^= 1

	for _, test := range []struct {
		name     string
		registry []byte
		isValid  bool
	}{
		{"current", current, true},
		{"appended", append(slices.Clone(current), []byte("\nFixture-only appendix.\n")...), true},
		{"historical", current[:pin.bytes], true},
		{"changed-prefix", changed, false},
		{"changed-snapshot", changed[:pin.bytes], false},
	} {
		t.Run(test.name, func(t *testing.T) {

			snapshot, err := usaPersonalCategoricalRegistrySnapshot(test.registry)
			if err != nil {
				if test.isValid || snapshot != nil {
					t.Fatalf("expected valid snapshot or empty rejected result, got %d bytes and %v", len(snapshot), err)
				}
				return
			}
			if !test.isValid {
				t.Fatal("expected corrupted snapshot rejection, got success")
			}
			if len(snapshot) != pin.bytes || fmt.Sprintf("%x", sha256.Sum256(snapshot)) != pin.sha256 {
				t.Fatalf("expected historical snapshot identity, got %d bytes and %x",
					len(snapshot), sha256.Sum256(snapshot))
			}
			if !slices.Equal(snapshot, current[:pin.bytes]) {
				t.Fatal("expected exact historical prefix, got different bytes")
			}

			firstByte := test.registry[0]
			snapshot[0] ^= 1
			if test.registry[0] != firstByte {
				t.Fatal("expected owned snapshot copy, got mutable registry alias")
			}

		})
	}

	t.Run("truncated", func(t *testing.T) {
		for length := range pin.bytes {
			snapshot, err := usaPersonalCategoricalRegistrySnapshot(current[:length])
			if err != nil {
				if snapshot != nil {
					t.Fatalf("expected no truncated snapshot, got %d bytes", len(snapshot))
				}
				continue
			}
			t.Fatalf("expected rejection of %d registry bytes, got success", length)
		}
	})

	t.Run("preparer-rejects-current", func(t *testing.T) {
		assets := readUSAPersonalCategoricalAssets(t)
		assets[0] = current
		m, err := prepareUSAPersonalCategoricalCandidate(assets)
		if m != nil {
			t.Fatal("expected no candidate for full current registry, got admission")
		}
		usaCompanyEmployerRejected(t, err, nil, "untrimmed registry")
	})

}

func readUSAPersonalCategoricalAssets(t *testing.T) usaPersonalCategoricalAssets {
	t.Helper()
	var assets usaPersonalCategoricalAssets
	for i, pin := range usaPersonalCategoricalPins {
		encoded, err := os.ReadFile("../../" + pin.path)
		if err != nil {
			t.Fatalf("expected adopted personal asset %s, got %v", pin.path, err)
		}
		if i == 0 {
			encoded, err = usaPersonalCategoricalRegistrySnapshot(encoded)
			if err != nil {
				t.Fatalf("expected adopted historical registry snapshot, got %v", err)
			}
		}
		assets[i] = encoded
	}
	return assets
}

func readUSAPersonalCategoricalHarness(t *testing.T) *usaPersonalCategoricalHarness {
	t.Helper()
	base := readUSACompletionHarness(t)
	m := readUSAPersonalCategoricalModel(t)
	cm, err := prepareUSACompanyEmployerCandidate(readUSACompanyEmployerAssets(t))
	if err != nil {
		t.Fatalf("expected adopted Company/Employer, got %v", err)
	}
	companies, err := newUSACompanyEmployerHarness(base, cm)
	if err != nil {
		t.Fatalf("expected regenerated Company roster, got %v", err)
	}
	w, err := newUSAPersonalCategoricalHarness(base, m, companies)
	if err != nil {
		t.Fatalf("expected USA categorical join, got %v", err)
	}
	return w
}

// readUSAPersonalCategoricalInput reads original component masses for the
// independent test oracle, rather than reversing the compiled cumulatives.
func readUSAPersonalCategoricalInput(t *testing.T, assets usaPersonalCategoricalAssets) [6]internationalCategorical {

	t.Helper()
	var result [6]internationalCategorical
	err := json.Unmarshal(assets[4], &result[0])
	if err != nil {
		t.Fatalf("expected Education component, got %v", err)
	}
	var activity struct {
		Axes    []string   `json:"axes"`
		Weights [][]uint32 `json:"primaryActivity"`
	}
	err = json.Unmarshal(assets[5], &activity)
	if err != nil {
		t.Fatalf("expected Activity component, got %v", err)
	}
	result[1].Axes, result[1].Outcome = activity.Axes, "activity"
	for _, row := range activity.Weights {
		result[1].Weights = append(result[1].Weights, row...)
	}
	for i, outcome := range []string{"jobFamily", "jobLevel", "jobFamily", "industry"} {
		err = json.Unmarshal(assets[i+6], &result[i+2])
		if err != nil {
			t.Fatalf("expected professional component %d, got %v", i, err)
		}
		result[i+2].Outcome = outcome
	}

	return result
}

func readUSAPersonalCategoricalModel(t *testing.T) *usaPersonalCategoricalCandidate {
	t.Helper()
	m, err := prepareUSAPersonalCategoricalCandidate(readUSAPersonalCategoricalAssets(t))
	if err != nil {
		t.Fatalf("expected admitted USA personal candidate, got %v", err)
	}
	return m
}

func usaPersonalCategoricalPinsForEvidence() []map[string]any {
	var pins []map[string]any
	for _, pin := range usaPersonalCategoricalPins {
		pins = append(pins, map[string]any{"path": pin.path, "bytes": pin.bytes, "sha256": pin.sha256})
	}
	return pins
}

// usaPersonalCategoricalRegistrySnapshot copies the verified authority prefix
// solely for constructing valid historical fixtures, never mutated inputs.
func usaPersonalCategoricalRegistrySnapshot(registry []byte) ([]byte, error) {
	pin := usaPersonalCategoricalPins[0]
	if len(registry) < pin.bytes {
		return nil, fmt.Errorf("USA historical registry snapshot is truncated")
	}
	snapshot := slices.Clone(registry[:pin.bytes])
	if fmt.Sprintf("%x", sha256.Sum256(snapshot)) != pin.sha256 {
		return nil, fmt.Errorf("USA historical registry snapshot has incompatible digest")
	}
	return snapshot, nil
}

// usaPersonalCategoricalTuple encodes a test tuple, never a public Person
// format. Nil branch pointers are JSON null, never ordinal-zero placeholders.
func usaPersonalCategoricalTuple(t *testing.T, record usaPersonalCategoricalRecord) []byte {

	t.Helper()
	var traces []any
	for _, trace := range record.traces {
		var parents []any
		for _, parent := range trace.parents {
			parents = append(parents, []any{parent.name, parent.value})
		}
		traces = append(traces, []any{
			trace.node, trace.row, parents, trace.position, trace.before, trace.after, trace.draw, trace.total,
		})
	}
	var replay []string
	for _, identity := range record.replay {
		replay = append(replay, fmt.Sprintf("%x", identity))
	}
	encoded, err := json.Marshal([]any{
		record.coordinate, record.layer1.person, record.layer1.anchorAge, record.layer1.cohort,
		record.layer1.localityID, record.layer1.division, record.layer1.divisionOrdinal, record.layer1.addressLine,
		record.education, record.activity, record.employee, record.selfEmployed, record.company, replay, traces,
	})
	if err != nil {
		t.Fatalf("expected deterministic technical tuple, got %v", err)
	}

	return append(encoded, '\n')
}

func writeUSAPersonalCategoricalEvidence(t *testing.T, name string, value any) {

	t.Helper()
	directory := os.Getenv("KRENALIS_USA_PERSONAL_CATEGORICAL_EVIDENCE")
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
