// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// TestUSASocioeconomicAdmission preserves every technical mass, age and tuple
// through the nine-Division typed entry and interprets supports only as USD.
func TestUSASocioeconomicAdmission(t *testing.T) {

	document := usaSocioeconomicFixture()
	data := mustUSASocioeconomicData(t, document)
	m := data.model
	if !m.initialized || m.protocol != "socioeconomic-us-v1" || m.numeric.territories != 9 ||
		m.numeric.companyCount != 65536 {
		t.Fatalf("expected initialized USA/9/65536 model, got %+v", m.interpretation)
	}
	if m.interpretation.Country != "US" || m.interpretation.Profile != "us-2026a-socioeconomic-v1" ||
		m.interpretation.AnchorDate != "2026-01-01" || m.interpretation.AgeDomain != (internationalAgeDomain{21, 75}) ||
		m.interpretation.IncomeUnit != (internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"}) {
		t.Fatalf("expected authorized USA national interpretation, got %+v", m.interpretation)
	}
	divisions := []string{
		"new-england", "middle-atlantic", "east-north-central", "west-north-central", "south-atlantic",
		"east-south-central", "west-south-central", "mountain", "pacific",
	}
	for i, category := range m.interpretation.Dimensions.Territory.Categories {
		if category.Code != divisions[i] {
			t.Fatalf("expected Division %d = %s, got %s", i, divisions[i], category.Code)
		}
	}

	tables := []socioeconomicWeights{
		m.numeric.companyIndustry, m.numeric.education, m.numeric.primaryActivity, m.numeric.employeeJobFamily,
		m.numeric.employeeJobLevel, m.numeric.selfEmployedJobFamily, m.numeric.selfEmployedIndustry,
	}
	var rows, weights int
	for i, source := range document.Models.categorical() {
		table := tables[i]
		rows += len(table.cumulative) / table.width
		weights += len(table.cumulative)
		var previous uint64
		for j, cumulative := range table.cumulative {
			if j%table.width == 0 {
				previous = 0
			}
			if cumulative-previous != uint64(source.Weights[j]) {
				t.Fatalf("expected unchanged categorical mass %d/%d, got %d", i, j, cumulative-previous)
			}
			previous = cumulative
		}
	}
	if rows != 1115 || weights != 6496 {
		t.Fatalf("expected 1115 rows and 6496 weights, got %d/%d", rows, weights)
	}
	for i, partition := range document.AgePartitions.ordered() {
		for offset, bucket := range m.numeric.ages[i] {
			expected := 0
			for j, bound := range partition.LowerBounds {
				if uint32(offset+21) >= bound {
					expected = j
				}
			}
			if int(bucket) != expected {
				t.Fatalf("expected age bucket %d/%d = %d, got %d", i, offset+21, expected, bucket)
			}
		}
	}

	var slots, nulls int
	for branch, source := range document.Models.Income.ordered() {
		compiled := m.numeric.income[branch]
		slots += len(compiled.tupleMap)
		for slot, id := range compiled.tupleMap {
			if source.TupleMap[slot] == nil {
				nulls++
				if id != 65535 {
					t.Fatalf("expected structural sentinel at %d/%d, got %d", branch, slot, id)
				}
				continue
			}
			if uint32(id) != *source.TupleMap[slot] {
				t.Fatalf("expected unchanged law ID at %d/%d, got %d", branch, slot, id)
			}
		}
		if len(compiled.laws) != 9 {
			t.Fatalf("expected nine technical laws in branch %d, got %d", branch, len(compiled.laws))
		}
		for law, expected := range source.Laws {
			for point, support := range expected.Values {
				value, err := m.incomeSupport(branch, law, point)
				if err != nil {
					t.Fatalf("expected admitted USD support, got %v", err)
				}
				if value != annualIncomeUSD(support) {
					t.Fatalf("expected USD %d, got %d", support, value)
				}
			}
		}
	}
	if slots != 34272 || nulls != 5040 {
		t.Fatalf("expected 34272 slots and 5040 nulls, got %d/%d", slots, nulls)
	}
	if len(m.numeric.employeeCount) != 14 || m.numeric.employer.cap != 1 {
		t.Fatalf("expected 14 count laws and independent USA cap 1, got %d/%d",
			len(m.numeric.employeeCount), m.numeric.employer.cap)
	}
	t.Logf("USA: %d rows, %d weights, %d slots, %d nulls, 440 age positions; USD 0..%d",
		rows, weights, slots, nulls, uint32(math.MaxUint32))

}

// TestUSASocioeconomicClosedDocuments applies the unchanged strict v1 decoder
// to missing, unknown, duplicated, incompatible and malformed USA fields.
func TestUSASocioeconomicClosedDocuments(t *testing.T) {

	encoded := internationalTestJSON(t, usaSocioeconomicFixture())
	for _, test := range []struct{ name, old, replacement string }{
		{"unknown root", `"schema":`, `"extra":0,"schema":`},
		{"embedded digest", `"schema":`, `"numericDigest":"` + strings.Repeat("0", 64) + `","schema":`},
		{"photo identity", `"schema":`, `"snapshotId":"invalid","schema":`},
		{"duplicate escaped", `"schema":`, `"\u0073chema":"sw-socioeconomic-data-v1","schema":`},
		{"unknown nested", `"minimum":21`, `"minimum":21,"other":0`},
		{"missing zero precision", `"decimalPlaces":0,`, ``},
		{"compiled forbidden", `"lowerBounds":`, `"compiled":[],"lowerBounds":`},
		{"extra partition", `"agePartitions":{`, `"agePartitions":{"employeeIncome":{"lowerBounds":[21]},`},
		{"outcome absent", `"outcome":"industry",`, ``},
		{"null source", `"schema":`, `"sourceModel":null,"schema":`},
		{"null definition", `"code":"below-primary"`, `"code":"below-primary","definition":null`},
		{"boolean", `"companyCount":65536`, `"companyCount":true`},
		{"quoted integer", `"companyCount":65536`, `"companyCount":"65536"`},
		{"exponent", `"companyCount":65536`, `"companyCount":65536e0`},
		{"decimal", `"companyCount":65536`, `"companyCount":65536.0`},
		{"negative zero", `"decimalPlaces":0`, `"decimalPlaces":-0`},
		{"uint64 overflow", `"companyCount":65536`, `"companyCount":18446744073709551616`},
		{"uint32 overflow", `"workforceCap":1`, `"workforceCap":4294967296`},
		{"unknown law property", `"values":[0,1]`, `"values":[0,1],"extra":0`},
		{"URL field", `"schema":`, `"url":"https://example.invalid/model","schema":`},
		{"leading BOM", `{`, "\xef\xbb\xbf{"},
		{"invalid UTF-8", `"US"`, "\"\xff\""},
		{"unpaired surrogate", `"US"`, `"\ud800"`},
		{"trailing comma", `"minimum":21`, `"minimum":21,`},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := bytes.Replace(encoded, []byte(test.old), []byte(test.replacement), 1)
			if bytes.Equal(input, encoded) {
				t.Fatal("expected fixture substitution, got unchanged bytes")
			}
			data, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(input))
			if err != nil {
				if data != nil {
					t.Fatal("expected no partially admitted state, got data")
				}
				return
			}
			t.Fatal("expected invalid USA document rejected, got success")
		})
	}
	var fields map[string]json.RawMessage
	err := json.Unmarshal(encoded, &fields)
	if err != nil {
		t.Fatalf("expected typed fixture fields, got %v", err)
	}
	for name := range fields {
		t.Run("missing "+name, func(t *testing.T) {
			copy := make(map[string]json.RawMessage, len(fields))
			for key, value := range fields {
				if key != name {
					copy[key] = value
				}
			}
			_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, copy)))
			if err != nil {
				return
			}
			t.Fatalf("expected missing %s rejected, got success", name)
		})
	}
	for _, input := range [][]byte{append(slices.Clone(encoded), encoded...), append(slices.Clone(encoded), 'x')} {
		_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(input))
		if err != nil {
			continue
		}
		t.Fatal("expected trailing document or token rejected, got success")
	}

}

// TestUSASocioeconomicEditorialIdentity excludes editorial and provenance
// changes from numerical identity while retaining them in document identity.
func TestUSASocioeconomicEditorialIdentity(t *testing.T) {

	document := usaSocioeconomicFixture()
	baseline := mustUSASocioeconomicData(t, document)
	before, err := baseline.logicalDigest()
	if err != nil {
		t.Fatalf("expected document digest, got %v", err)
	}
	for _, mutate := range []struct {
		name string
		edit func(*internationalSocioeconomic)
	}{
		{"release", func(d *internationalSocioeconomic) { d.Release = "artificial-editorial-revision" }},
		{"definitions", func(d *internationalSocioeconomic) {
			text := "Editorial https://example.invalid/source and socioeconomic/test/path"
			for _, taxonomy := range []*internationalTaxonomy{
				&d.Dimensions.Education, &d.Dimensions.Industry, &d.Dimensions.JobFamily,
				&d.Dimensions.JobLevel, &d.Dimensions.Territory,
			} {
				for i := range taxonomy.Categories {
					taxonomy.Categories[i].Definition = &text
				}
			}
			for i := range d.Dimensions.Activity.Categories {
				d.Dimensions.Activity.Categories[i].Definition = &text
			}
			for i := range d.Dimensions.SizeBand.Categories {
				d.Dimensions.SizeBand.Categories[i].Definition = &text
			}
		}},
		{"provenance", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"artificial-source-v1", strings.Repeat("0", 64)}
		}},
		{"physical source digest", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"artificial-source-v1", strings.Repeat("1", 64)}
		}},
	} {
		t.Run(mutate.name, func(t *testing.T) {
			d := usaSocioeconomicFixture()
			mutate.edit(&d)
			got := mustUSASocioeconomicData(t, d)
			after, err := got.logicalDigest()
			if err != nil {
				t.Fatalf("expected edited document digest, got %v", err)
			}
			if before == after {
				t.Fatal("expected changed full document digest, got equality")
			}
			if !reflect.DeepEqual(baseline.model, got.model) {
				t.Fatal("expected identical numerical state, interpretation, digest and 15 fingerprints, got divergence")
			}
		})
	}

}

// TestUSASocioeconomicIdentity compares independent H.2 and compiled C1 bytes,
// reproducible digests and the exact local KFD1 asset and age projections.
func TestUSASocioeconomicIdentity(t *testing.T) {

	document := usaSocioeconomicFixture()
	data := mustUSASocioeconomicData(t, document)
	projection := usaSocioeconomicProjectionForTest(t, document)
	compiled, err := data.model.canonicalProjection()
	if err != nil {
		t.Fatalf("expected compiled numerical projection, got %v", err)
	}
	if !bytes.Equal(projection, compiled) || sha256.Sum256(projection) != data.model.digest {
		t.Fatal("expected equal document projection, prepared interpretation and compiled digest, got divergence")
	}
	want := "fe771522c11091d4619ba8ef67e5b34bfb78721b61e734b18b7bbc8db8a196e5"
	if got := fmt.Sprintf("%x", data.model.digest); got != want {
		t.Fatalf("expected artificial USA numerical digest %s, got %s", want, got)
	}
	var fields map[string]json.RawMessage
	err = json.Unmarshal(compiled, &fields)
	if err != nil {
		t.Fatalf("expected C1 projection, got %v", err)
	}
	if len(fields) != 9 {
		t.Fatalf("expected exactly nine H.2 projection fields, got %d", len(fields))
	}
	encoded, err := data.canonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical USA document, got %v", err)
	}
	reordered, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(reverseInternationalObjects(t, encoded)))
	if err != nil {
		t.Fatalf("expected nonsemantic transport order accepted, got %v", err)
	}
	if !reflect.DeepEqual(data.model, reordered.model) {
		t.Fatal("expected identical complete USA model after transport reorder, got divergence")
	}
	first, err := data.logicalDigest()
	if err != nil {
		t.Fatalf("expected first logical digest, got %v", err)
	}
	second, err := reordered.logicalDigest()
	if err != nil {
		t.Fatalf("expected reordered logical digest, got %v", err)
	}
	if first != second || first == data.model.digest || fmt.Sprintf("%x", data.model.digest) == italianSourceModelSHA256 {
		t.Fatal("expected reproducible separate USA document/numerical identities, got divergence or reuse")
	}

	unchanged := map[int]bool{0: true, 1: true, 5: true, 6: true, 8: true}
	for node, path := range usaSocioeconomicPaths {
		expected := strings.Replace(socioeconomicPaths[node], "socioeconomic/v1/", "socioeconomic/us-v1/", 1)
		if unchanged[node] {
			expected = socioeconomicPaths[node]
		}
		if path != expected {
			t.Fatalf("expected local path %s, got %s", expected, path)
		}
		asset, err := data.model.numeric.appendAsset(nil, node)
		if err != nil {
			t.Fatalf("expected local numerical asset, got %v", err)
		}
		component, err := BytesComponent("asset", append(asset, '\n'))
		if err != nil {
			t.Fatalf("expected asset component, got %v", err)
		}
		components := []Component{component}
		if age := socioeconomicNodeAges[node]; age >= 0 {
			component, err = BytesComponent("age-map", append(appendSocioeconomicAge(nil, data.model.numeric.ages[age]), '\n'))
			if err != nil {
				t.Fatalf("expected local age component, got %v", err)
			}
			components = append(components, component)
		}
		expectedDigest, err := canonicalDigest("fakedata/"+path+"/model", components)
		if err != nil {
			t.Fatalf("expected KFD1 local digest, got %v", err)
		}
		if data.model.fingerprints[node] != expectedDigest {
			t.Fatalf("expected local asset/age fingerprint at %s, got divergence", path)
		}
		frozen, err := canonicalDigest("fakedata/"+socioeconomicPaths[node]+"/model", components)
		if err != nil {
			t.Fatalf("expected existing compatible fingerprint formula, got %v", err)
		}
		if (frozen == expectedDigest) != unchanged[node] {
			t.Fatalf("expected exactly ten revised local domains, got fingerprint equality at %s", path)
		}
	}
	t.Logf("artificial USA numerical C1: %d bytes; SHA-256 %x; document KFD1 %x", len(compiled), data.model.digest, first)

}

// TestUSASocioeconomicInvalid rejects unsupported national meanings, grids,
// maps, factors and laws through the full typed USA admission boundary.
func TestUSASocioeconomicInvalid(t *testing.T) {

	for _, test := range []struct {
		name   string
		mutate func(*internationalSocioeconomic)
	}{
		{"schema", func(d *internationalSocioeconomic) { d.Schema = "SW-L2-D2B2-logical-v0" }},
		{"country", func(d *internationalSocioeconomic) { d.Country = "IT" }},
		{"profile", func(d *internationalSocioeconomic) { d.Profile = "us-2026b-socioeconomic-v1" }},
		{"anchor", func(d *internationalSocioeconomic) { d.AnchorDate = "2026-01-02" }},
		{"minimum age", func(d *internationalSocioeconomic) { d.AgeDomain.Minimum = 20 }},
		{"maximum age", func(d *internationalSocioeconomic) { d.AgeDomain.Maximum = 76 }},
		{"company count", func(d *internationalSocioeconomic) { d.CompanyCount = 65537 }},
		{"company zero", func(d *internationalSocioeconomic) { d.CompanyCount = 0 }},
		{"currency", func(d *internationalSocioeconomic) { d.IncomeUnit.Currency = "EUR" }},
		{"precision", func(d *internationalSocioeconomic) { d.IncomeUnit.DecimalPlaces = 2 }},
		{"period", func(d *internationalSocioeconomic) { d.IncomeUnit.Period = "month" }},
		{"basis", func(d *internationalSocioeconomic) { d.IncomeUnit.Basis = "net" }},
		{"release", func(d *internationalSocioeconomic) { d.Release = "invalid/release" }},
		{"Italian source schema", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"SW-L2-D2B2-logical-v0", strings.Repeat("0", 64)}
		}},
		{"Italian source digest", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"artificial-source-v1", italianSourceModelSHA256}
		}},
		{"source schema", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"invalid/schema", strings.Repeat("0", 64)}
		}},
		{"source digest", func(d *internationalSocioeconomic) {
			d.SourceModel = &internationalSourceModel{"artificial-source-v1", strings.Repeat("A", 64)}
		}},
		{"missing Division", func(d *internationalSocioeconomic) {
			d.Dimensions.Territory.Categories = d.Dimensions.Territory.Categories[:8]
		}},
		{"duplicate Division", func(d *internationalSocioeconomic) {
			d.Dimensions.Territory.Categories[0] = d.Dimensions.Territory.Categories[1]
		}},
		{"Division order", func(d *internationalSocioeconomic) {
			c := d.Dimensions.Territory.Categories
			c[0], c[1] = c[1], c[0]
		}},
		{"twenty territories", func(d *internationalSocioeconomic) {
			d.Dimensions.Territory = italianInternationalDimensions().Territory
		}},
		{"twenty-territory grid", func(d *internationalSocioeconomic) {
			d.Models.Education.Weights = make([]uint32, 840)
		}},
		{"twenty-territory map", func(d *internationalSocioeconomic) {
			d.Models.Income.Employee.TupleMap = make([]*uint32, 56000)
		}},
		{"activity taxonomy", func(d *internationalSocioeconomic) { d.Dimensions.Activity.SemanticID = "unknown" }},
		{"activity categories", func(d *internationalSocioeconomic) {
			d.Dimensions.Activity.Categories = d.Dimensions.Activity.Categories[:5]
		}},
		{"activity branch", func(d *internationalSocioeconomic) { d.Dimensions.Activity.Categories[0].Branch = "student" }},
		{"size taxonomy", func(d *internationalSocioeconomic) { d.Dimensions.SizeBand.SemanticID = "unknown" }},
		{"size categories", func(d *internationalSocioeconomic) {
			d.Dimensions.SizeBand.Categories = d.Dimensions.SizeBand.Categories[:4]
		}},
		{"size threshold", func(d *internationalSocioeconomic) {
			d.Dimensions.SizeBand.Categories[1].MinimumEmployeeCount = 2
		}},
		{"categorical zero", func(d *internationalSocioeconomic) { clear(d.Models.Education.Weights[:7]) }},
		{"categorical GCD", func(d *internationalSocioeconomic) {
			for i := range 7 {
				d.Models.Education.Weights[i] = 2
			}
		}},
		{"count axes", func(d *internationalSocioeconomic) { d.Models.EmployeeCount.Axes = []string{} }},
		{"count laws", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Laws = d.Models.EmployeeCount.Laws[:13]
		}},
		{"count points", func(d *internationalSocioeconomic) { d.Models.EmployeeCount.Laws[0].Values = make([]uint32, 129) }},
		{"count support", func(d *internationalSocioeconomic) { d.Models.EmployeeCount.Laws[0].Values[1] = 0 }},
		{"no workforce", func(d *internationalSocioeconomic) {
			for i := range d.Models.EmployeeCount.Laws {
				d.Models.EmployeeCount.Laws[i] = internationalLaw{[]uint32{0}, []uint32{1}}
			}
		}},
		{"cap zero", func(d *internationalSocioeconomic) { d.Models.Employer.WorkforceCap = 0 }},
		{"cap overflow", func(d *internationalSocioeconomic) { d.Models.Employer.WorkforceCap = 65536 }},
		{"factor zero", func(d *internationalSocioeconomic) { d.Models.Employer.SizeFactors.Values[0] = 0 }},
		{"factor overflow", func(d *internationalSocioeconomic) { d.Models.Employer.IndustryFactors.Values[0] = 65537 }},
		{"factor maximum", func(d *internationalSocioeconomic) { d.Models.Employer.IndustryFactors.Values[0] = 4096 }},
		{"factor count", func(d *internationalSocioeconomic) {
			d.Models.Employer.IndustryFactors.Values = d.Models.Employer.IndustryFactors.Values[:139]
		}},
		{"factor axes", func(d *internationalSocioeconomic) { d.Models.Employer.SizeFactors.Axes[1] = "sizeBand" }},
		{"null zero band", func(d *internationalSocioeconomic) {
			id := uint32(0)
			d.Models.Income.Employee.TupleMap[0] = &id
		}},
		{"null positive band", func(d *internationalSocioeconomic) { d.Models.Income.Employee.TupleMap[1] = nil }},
		{"law first occurrence", func(d *internationalSocioeconomic) {
			id := uint32(1)
			d.Models.Income.Student.TupleMap[0] = &id
		}},
		{"law duplicate", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws[1] = d.Models.Income.Student.Laws[0]
		}},
		{"law unreferenced", func(d *internationalSocioeconomic) {
			d.Models.Income.Student.Laws = append(d.Models.Income.Student.Laws, internationalLaw{[]uint32{7}, []uint32{1}})
		}},
		{"law empty", func(d *internationalSocioeconomic) { d.Models.Income.Student.Laws[0].Values = []uint32{} }},
		{"law order", func(d *internationalSocioeconomic) { d.Models.Income.Student.Laws[0].Values[1] = 0 }},
		{"law GCD", func(d *internationalSocioeconomic) { d.Models.Income.Student.Laws[0].Weights = []uint32{2, 2, 2} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			document := usaSocioeconomicFixture()
			test.mutate(&document)
			data, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, document)))
			if err != nil {
				if data != nil {
					t.Fatal("expected no partially admitted USA state, got data")
				}
				return
			}
			t.Fatal("expected invalid USA binding or numerical model rejected, got success")
		})
	}
	for dimension := range 5 {
		for _, property := range []string{"identity", "categories", "code"} {
			t.Run(fmt.Sprintf("taxonomy %d/%s", dimension, property), func(t *testing.T) {
				d := usaSocioeconomicFixture()
				taxonomy := []*internationalTaxonomy{
					&d.Dimensions.Education, &d.Dimensions.Industry, &d.Dimensions.JobFamily,
					&d.Dimensions.JobLevel, &d.Dimensions.Territory,
				}[dimension]
				switch property {
				case "identity":
					taxonomy.SemanticID = "unknown"
				case "categories":
					taxonomy.Categories = taxonomy.Categories[1:]
				case "code":
					taxonomy.Categories[0].Code = "unknown"
				}
				_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, d)))
				if err != nil {
					return
				}
				t.Fatal("expected closed USA taxonomy rejected, got success")
			})
		}
	}
	for partition := range 8 {
		t.Run(fmt.Sprintf("partition %d", partition), func(t *testing.T) {
			d := usaSocioeconomicFixture()
			d.AgePartitions.ordered()[partition].LowerBounds[1]++
			_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, d)))
			if err != nil {
				return
			}
			t.Fatal("expected unauthorized USA age partition rejected, got success")
		})
	}
	for categorical := range 7 {
		for _, property := range []string{"axes", "outcome", "length"} {
			t.Run(fmt.Sprintf("categorical %d/%s", categorical, property), func(t *testing.T) {
				d := usaSocioeconomicFixture()
				model := d.Models.categorical()[categorical]
				switch property {
				case "axes":
					model.Axes = []string{"unknown"}
				case "outcome":
					model.Outcome = "unknown"
				case "length":
					model.Weights = model.Weights[1:]
				}
				_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, d)))
				if err != nil {
					return
				}
				t.Fatal("expected invalid USA categorical model rejected, got success")
			})
		}
	}
	for branch := range 6 {
		for _, property := range []string{"axes", "map", "pool", "reference", "null", "points", "weight"} {
			t.Run(fmt.Sprintf("income %d/%s", branch, property), func(t *testing.T) {
				d := usaSocioeconomicFixture()
				income := d.Models.Income.ordered()[branch]
				switch property {
				case "axes":
					income.Axes = []string{}
				case "map":
					income.TupleMap = income.TupleMap[1:]
				case "pool":
					income.Laws = []internationalLaw{}
				case "reference":
					id := uint32(math.MaxUint32)
					income.TupleMap[1] = &id
				case "null":
					income.TupleMap[1] = nil
				case "points":
					income.Laws[0].Values = make([]uint32, 513)
				case "weight":
					income.Laws[0].Weights[0] = 0
				}
				_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, d)))
				if err != nil {
					return
				}
				t.Fatal("expected invalid USA Income branch rejected, got success")
			})
		}
	}

}

// TestUSASocioeconomicItalyBoundary proves national decoder, model, currency
// and public constructor separation without changing Italian fixtures.
func TestUSASocioeconomicItalyBoundary(t *testing.T) {

	t.Run("Italian entry", func(t *testing.T) {
		usa := usaSocioeconomicFixture()
		_, err := DecodeItalianSocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, usa)))
		if err != nil {
			if !strings.Contains(err.Error(), "Italian socioeconomic metadata") {
				t.Fatalf("expected Italian national admission rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected USA rejected by Italian entry, got success")
	})
	t.Run("USA entry", func(t *testing.T) {
		italian := internationalSocioeconomicFixture(t)
		_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, italian)))
		if err != nil {
			if !strings.Contains(err.Error(), "USA socioeconomic metadata") {
				t.Fatalf("expected USA national admission rejection, got %v", err)
			}
			return
		}
		t.Fatal("expected Italy rejected by USA entry, got success")
	})
	usaType := reflect.TypeFor[*usaSocioeconomicModel]()
	italianType := reflect.TypeOf(NewSocioeconomicWorld).In(1)
	if usaType.AssignableTo(italianType) || usaType.ConvertibleTo(italianType) {
		t.Fatal("expected distinct USA model excluded from public Italian constructor, got compatible type")
	}
	if reflect.TypeFor[annualIncomeUSD]() == reflect.TypeFor[AnnualIncomeEUR]() {
		t.Fatal("expected distinct monetary types, got equality")
	}
	if _, exists := usaType.MethodByName("Model"); exists || usaType.NumMethod() != 0 {
		t.Fatal("expected no public USA model methods, got public API")
	}

}

func mustUSASocioeconomicData(t *testing.T, document internationalSocioeconomic) *usaSocioeconomicData {
	t.Helper()
	data, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(internationalTestJSON(t, document)))
	if err != nil {
		t.Fatalf("expected artificial USA document accepted, got %v", err)
	}
	return data
}

// usaSocioeconomicFixture uses ordinal technical masses and laws from the
// previous compiler suite, with independently specified national bindings.
func usaSocioeconomicFixture() internationalSocioeconomic {

	input := socioeconomicNumericFixture(9)
	d := internationalSocioeconomic{
		Schema: "sw-socioeconomic-data-v1", Release: "artificial-test", Profile: "us-2026a-socioeconomic-v1",
		Country: "US", AnchorDate: "2026-01-01", AgeDomain: internationalAgeDomain{21, 75}, CompanyCount: 65536,
		IncomeUnit: internationalIncomeUnit{"USD", 0, "year", "gross-personal-monetary-annualized"},
		Dimensions: internationalDimensions{
			Education: internationalTaxonomyFor("sw-l2-education-v1", socioeconomicEducationLevels[:]),
			Industry:  internationalTaxonomyFor("sw-l2-industry-v1", socioeconomicIndustries[:]),
			JobFamily: internationalTaxonomyFor("sw-l2-job-family-v1", socioeconomicJobFamilies[:]),
			JobLevel:  internationalTaxonomyFor("sw-l2-job-level-v1", socioeconomicJobLevels[:]),
			Territory: internationalTaxonomyFor("sw-us-2026a-census-division-v1", []string{
				"new-england", "middle-atlantic", "east-north-central", "west-north-central", "south-atlantic",
				"east-south-central", "west-south-central", "mountain", "pacific",
			}),
			Activity: internationalActivityTaxonomy{SemanticID: "sw-l2-primary-activity-v1"},
			SizeBand: internationalSizeTaxonomy{SemanticID: "sw-l2-company-size-band-v1"},
		},
	}
	for i, code := range socioeconomicActivities {
		d.Dimensions.Activity.Categories = append(d.Dimensions.Activity.Categories, internationalActivityCategory{
			Code: string(code), Branch: []string{"employee", "selfEmployed", "unemployed", "student", "retired", "inactive"}[i],
		})
	}
	for i, code := range socioeconomicSizeBands {
		d.Dimensions.SizeBand.Categories = append(d.Dimensions.SizeBand.Categories, internationalSizeCategory{
			Code: string(code), MinimumEmployeeCount: []uint32{0, 1, 10, 50, 250}[i],
		})
	}
	bounds := [8][]uint32{
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 60, 65, 70},
		{21, 25, 35, 45, 55, 65}, {21, 25, 35, 45, 55, 65},
		{21, 25, 35, 45, 55, 65}, {21, 25, 30, 40, 55, 65},
		{21, 45, 55, 60, 65, 70}, {21, 25, 35, 45, 55, 65},
	}
	for i, partition := range d.AgePartitions.ordered() {
		partition.LowerBounds = bounds[i]
	}
	axes := [][]string{
		{}, {"age.education", "territory"}, {"age.activity", "education", "territory"},
		{"education", "territory"}, {"age.employeeLevel", "education", "jobFamily"},
		{"education", "territory"}, {"jobFamily"},
	}
	outcomes := []string{"industry", "education", "activity", "jobFamily", "jobLevel", "jobFamily", "industry"}
	weights := [][]uint32{
		input.companyIndustry, input.education, input.primaryActivity, input.employeeJobFamily,
		input.employeeJobLevel, input.selfEmployedJobFamily, input.selfEmployedIndustry,
	}
	for i, categorical := range d.Models.categorical() {
		*categorical = internationalCategorical{Axes: axes[i], Outcome: outcomes[i], Weights: weights[i]}
	}
	d.Models.EmployeeCount.Axes = []string{"industry"}
	for _, law := range input.employeeCount {
		d.Models.EmployeeCount.Laws = append(d.Models.EmployeeCount.Laws, internationalLaw{law.values, law.weights})
	}
	d.Models.Employer.WorkforceCap = 1
	d.Models.Employer.IndustryFactors = internationalFactorMatrix{[]string{"jobFamily", "industry"}, make([]uint32, 140)}
	d.Models.Employer.SizeFactors = internationalFactorMatrix{[]string{"jobLevel", "positiveSizeBand"}, make([]uint32, 16)}
	for _, factors := range [][]uint32{d.Models.Employer.IndustryFactors.Values, d.Models.Employer.SizeFactors.Values} {
		for i := range factors {
			factors[i] = 1
		}
	}
	incomeAxes := [][]string{
		{"territory", "jobFamily", "jobLevel", "industry", "sizeBand"},
		{"age.selfEmployedIncome", "territory", "jobFamily", "industry"},
		{"age.unemployedIncome", "education", "territory"}, {"age.studentIncome", "education", "territory"},
		{"age.retiredIncome", "education", "territory"}, {"age.inactiveIncome", "education", "territory"},
	}
	for branch, income := range d.Models.Income.ordered() {
		income.Axes = incomeAxes[branch]
		income.TupleMap = make([]*uint32, len(input.income[branch].tupleMap))
		for slot, id := range input.income[branch].tupleMap {
			if id != 65535 {
				value := id
				income.TupleMap[slot] = &value
			}
		}
		for _, law := range input.income[branch].laws {
			income.Laws = append(income.Laws, internationalLaw{
				append(slices.Clone(law.values), math.MaxUint32), append(slices.Clone(law.weights), 1),
			})
		}
	}

	return d
}

// usaSocioeconomicProjectionForTest builds the H.2 field set from the incoming
// document rather than the prepared interpretation or compiled serializer.
func usaSocioeconomicProjectionForTest(t *testing.T, d internationalSocioeconomic) []byte {

	t.Helper()
	dimensions := d.Dimensions
	for _, taxonomy := range []*internationalTaxonomy{
		&dimensions.Education, &dimensions.Industry, &dimensions.JobFamily, &dimensions.JobLevel, &dimensions.Territory,
	} {
		taxonomy.Categories = slices.Clone(taxonomy.Categories)
		for i := range taxonomy.Categories {
			taxonomy.Categories[i].Definition = nil
		}
	}
	dimensions.Activity.Categories = slices.Clone(dimensions.Activity.Categories)
	for i := range dimensions.Activity.Categories {
		dimensions.Activity.Categories[i].Definition = nil
	}
	dimensions.SizeBand.Categories = slices.Clone(dimensions.SizeBand.Categories)
	for i := range dimensions.SizeBand.Categories {
		dimensions.SizeBand.Categories[i].Definition = nil
	}
	projection, err := internationalCanonical(map[string]any{
		"country": d.Country, "profile": d.Profile, "anchorDate": d.AnchorDate, "ageDomain": d.AgeDomain,
		"dimensions": dimensions, "agePartitions": d.AgePartitions, "companyCount": d.CompanyCount,
		"incomeUnit": d.IncomeUnit, "models": d.Models,
	})
	if err != nil {
		t.Fatalf("expected independently projected USA numerical bytes, got %v", err)
	}

	return projection
}
