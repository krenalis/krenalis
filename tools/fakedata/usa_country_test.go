// Copyright 2026 Open2b. All rights reserved.
// Use of this source code is governed by an Elastic License 2.0
// that can be found in the LICENSE file.

package fakedata

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/krenalis/krenalis/tools/fakedata/internal/usadiagnostic"
)

func TestUSACountryAssembly(t *testing.T) {

	assets := readUSACountryAssets(t)
	a, err := assembleUSACountryModel(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected complete typed USA model, got %v", err)
	}
	b, err := assembleUSACountryModel(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected independent complete USA model, got %v", err)
	}
	first, err := a.canonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical USA document, got %v", err)
	}
	second, err := b.canonicalJSON()
	if err != nil {
		t.Fatalf("expected independent canonical USA document, got %v", err)
	}
	if !bytes.Equal(first, second) || fmt.Sprintf("%x", a.model.digest) !=
		"f7c8d650c6f3963afbb34a95780c38c1293f29f72df505bbe07d5760e1d48a54" {
		t.Fatalf("expected byte-identical complete ModelID, got %x and %x", a.model.digest, b.model.digest)
	}
	if fmt.Sprintf("%x", sha256.Sum256(first)) != "d48443e062ba30f684c7bea088a8f184e8e5a971e417f08c596985358e71811f" {
		t.Fatalf("expected assembled asset identity, got %x", sha256.Sum256(first))
	}
	projection, err := a.model.canonicalProjection()
	if err != nil {
		t.Fatalf("expected compiled-state projection, got %v", err)
	}
	if sha256.Sum256(projection) != a.model.digest {
		t.Fatalf("expected actual numeric projection ModelID, got %x", sha256.Sum256(projection))
	}
	if a.document.SourceModel.Schema == "SW-L2-D2B2-logical-v0" || a.document.IncomeUnit.Currency != "USD" {
		t.Fatal("expected direct USA provenance and USD, got Italian binding")
	}

	// Compare all authored laws/maps directly, separately from runtime encoders.
	income := readUSAIncomeDocument(t, assets[usaIncomePins[1].path])
	slots, nulls := 0, 0
	for branch, original := range income.ordered() {
		assembled := a.document.Models.Income.ordered()[branch]
		if len(assembled.Laws) != 1 || !slices.Equal(assembled.Laws[0].Values, original.Laws[0].Supports) ||
			!slices.Equal(assembled.Laws[0].Weights, original.Laws[0].Weights) ||
			!reflect.DeepEqual(assembled.TupleMap, original.Map.LawIDs) {
			t.Fatalf("expected every original Income support, weight and slot at branch %d, got mismatch", branch)
		}
		for _, id := range assembled.TupleMap {
			slots++
			if id == nil {
				nulls++
			}
		}
	}
	if slots != 34272 || nulls != 5040 || len(a.document.Models.EmployeeCount.Laws) != 14 {
		t.Fatalf("expected 34272 slots, 5040 nulls and 14 count laws, got %d/%d/%d",
			slots, nulls, len(a.document.Models.EmployeeCount.Laws))
	}
	if &a.model.numeric.income[0].laws[0].values[0] == &b.model.numeric.income[0].laws[0].values[0] {
		t.Fatal("expected independent model ownership, got shared mutable support")
	}

}

func TestUSACountryAssetsReject(t *testing.T) {

	assets := readUSACountryAssets(t)
	for i, pin := range usaCountrySourcePins() {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			for _, missing := range []bool{false, true} {
				changed := usaCountryAssets{}
				for path, encoded := range assets {
					changed[path] = encoded
				}
				changed[pin.Path] = bytes.Clone(assets[pin.Path])
				changed[pin.Path][0] ^= 1
				if missing {
					delete(changed, pin.Path)
				}
				model, err := assembleUSACountryModel(t.Context(), changed)
				if err != nil {
					if model != nil {
						t.Fatal("expected no rejected model, got partial model")
					}
					continue
				}
				t.Fatalf("expected altered or missing asset %s rejection, got admission", pin.Path)
			}
		})
	}
	_, err := loadUSACountryAssets(t.Context(), t.TempDir())
	if err != nil {
		return
	}

	t.Fatal("expected missing on-disk source rejection, got admission")
}

func TestUSACountryLegacyEquivalence(t *testing.T) {

	data := readUSACountryData(t)
	w, err := newUSACountryWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected internal World with full USA identity, got %v", err)
	}
	legacy := readUSAIncomeHarness(t)
	for i, node := range usaPersonalCategoricalNodes {
		if data.socioeconomic.model.fingerprints[node] != legacy.categorical.model.fingerprints[i] {
			t.Fatalf("expected historical local categorical fingerprint %d, got drift", node)
		}
	}
	for branch := range 6 {
		if data.socioeconomic.model.fingerprints[socioeconomicIncomeEmployee+branch] !=
			legacy.income.fingerprints[branch] {
			t.Fatalf("expected historical local Income fingerprint %d, got drift", branch)
		}
	}
	if data.socioeconomic.model.digest == legacy.income.identity || data.socioeconomic.model.digest ==
		legacy.categorical.model.identity {
		t.Fatal("expected complete numerical identity distinct from partial receipts, got receipt identity")
	}
	completeHash, categoricalHash := sha256.New(), sha256.New()
	var hashes [20000][32]byte
	for coordinate := PersonIndex(1); coordinate <= 20000; coordinate++ {
		actual, err := w.sampler.person(coordinate)
		if err != nil {
			t.Fatalf("expected complete native USA generation, got %v", err)
		}
		old, err := legacy.person(coordinate)
		if err != nil {
			t.Fatalf("expected historical USA generation, got %v", err)
		}
		// Normalize receipt metadata only for the historical evidence encoding;
		// neither record's parents, stream state, draw or sampled fact is changed.
		actual.categorical.replay = old.categorical.replay
		if !reflect.DeepEqual(actual, old) {
			t.Fatalf("expected identical local streams, traces and facts at %d, got mismatch", coordinate)
		}
		completeHash.Write(usaIncomeTuple(t, actual))
		categoricalHash.Write(usaPersonalCategoricalTuple(t, actual.categorical))
		person, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected typed USD person, got %v", err)
		}
		encoded, err := internationalCanonical(person)
		if err != nil {
			t.Fatalf("expected canonical full person, got %v", err)
		}
		hashes[coordinate-1] = sha256.Sum256(encoded)
	}
	if got := fmt.Sprintf("%x", completeHash.Sum(nil)); got !=
		"12203517c17d91ac476c7382044afebf6b10f322787720a873b45007314bed0a" {
		t.Fatalf("expected historical Income replay digest, got %s", got)
	}
	if got := fmt.Sprintf("%x", categoricalHash.Sum(nil)); got !=
		"170e2b5e514fccea030fb7a53f0ac03da63a7d646a0fd1c7d1553770368da04a" {
		t.Fatalf("expected historical categorical replay digest, got %s", got)
	}
	for coordinate := PersonIndex(20000); coordinate > 0; coordinate-- {
		person, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected reverse USD person, got %v", err)
		}
		encoded, err := internationalCanonical(person)
		if err != nil {
			t.Fatalf("expected reverse canonical person, got %v", err)
		}
		if sha256.Sum256(encoded) != hashes[coordinate-1] {
			t.Fatalf("expected zero reverse mismatch at %d, got mismatch", coordinate)
		}
	}
	t.Log("expected historical Income/categorical digests and 20000 exact stream comparisons, got PASS")

}

func TestUSACountryModelReject(t *testing.T) {

	assets := readUSACountryAssets(t)
	model, err := assembleUSACountryModel(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected assembled source, got %v", err)
	}
	for _, test := range []struct {
		name   string
		change func(*internationalSocioeconomic)
	}{
		{"Italian-country", func(d *internationalSocioeconomic) { d.Country = "IT" }},
		{"unknown-country", func(d *internationalSocioeconomic) { d.Country = "XX" }},
		{"unknown-currency", func(d *internationalSocioeconomic) { d.IncomeUnit.Currency = "XXX" }},
		{"EUR", func(d *internationalSocioeconomic) { d.IncomeUnit.Currency = "EUR" }},
		{"missing-income-map", func(d *internationalSocioeconomic) { d.Models.Income.Employee.TupleMap = nil }},
		{"productive-zero-band", func(d *internationalSocioeconomic) {
			zero := uint32(0)
			d.Models.Income.Employee.TupleMap[0] = &zero
		}},
		{"altered-fast-law", func(d *internationalSocioeconomic) { d.Models.Income.Employee.Laws[0].Values[384]++ }},
		{"missing-personal-model", func(d *internationalSocioeconomic) { d.Models.Education.Weights = nil }},
		{"missing-company-law", func(d *internationalSocioeconomic) {
			d.Models.EmployeeCount.Laws = d.Models.EmployeeCount.Laws[:13]
		}},
		{"missing-employer-factor", func(d *internationalSocioeconomic) { d.Models.Employer.SizeFactors.Values = nil }},
		{"factor-overflow", func(d *internationalSocioeconomic) {
			d.Models.Employer.IndustryFactors.Values[0] = 65536
		}},
		{"support-not-increasing", func(d *internationalSocioeconomic) {
			d.Models.Income.Retired.Laws[0].Values[1] = 0
		}},
		{"zero-weight", func(d *internationalSocioeconomic) { d.Models.Income.Student.Laws[0].Weights[1] = 0 }},
		{"law-index-outside-domain", func(d *internationalSocioeconomic) {
			bad := uint32(65535)
			d.Models.Income.Employee.TupleMap[1] = &bad
		}},
		{"noncanonical-GCD", func(d *internationalSocioeconomic) {
			for i := range d.Models.Income.Inactive.Laws[0].Weights {
				d.Models.Income.Inactive.Laws[0].Weights[i] = 2
			}
		}},
		{"age-map", func(d *internationalSocioeconomic) { d.AgePartitions.RetiredIncome.LowerBounds[1]++ }},
		{"taxonomy-order", func(d *internationalSocioeconomic) { slices.Reverse(d.Dimensions.Territory.Categories) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := model.canonicalJSON()
			if err != nil {
				t.Fatalf("expected original document, got %v", err)
			}
			var changed internationalSocioeconomic
			err = json.Unmarshal(encoded, &changed)
			if err != nil {
				t.Fatalf("expected typed original document, got %v", err)
			}
			test.change(&changed)
			encoded, err = json.Marshal(changed)
			if err != nil {
				t.Fatalf("expected mutated fixture, got %v", err)
			}
			prepared, err := prepareUSACountryData(t.Context(), assets, encoded)
			if err != nil {
				if prepared != nil {
					t.Fatal("expected no invalid country, got partial data")
				}
				return
			}
			t.Fatal("expected altered or incomplete full model rejection, got admission")
		})
	}
	encoded, err := model.canonicalJSON()
	if err != nil {
		t.Fatalf("expected source document, got %v", err)
	}
	for _, invalid := range []string{"4294967296", "-1", "0.5"} {
		t.Run("U32-"+invalid, func(t *testing.T) {
			changed := bytes.Replace(encoded, []byte(`"workforceCap":65535`), []byte(`"workforceCap":`+invalid), 1)
			_, err := decodeUSASocioeconomicData(t.Context(), bytes.NewReader(changed))
			if err != nil {
				return
			}
			t.Fatal("expected out-of-domain U32 rejection, got admission")
		})
	}
	for _, field := range []string{"values", "weights"} {
		t.Run(field+"-overflow", func(t *testing.T) {
			var changed map[string]any
			err := json.Unmarshal(encoded, &changed)
			if err != nil {
				t.Fatalf("expected original JSON object, got %v", err)
			}
			income := changed["models"].(map[string]any)["income"].(map[string]any)
			law := income["employee"].(map[string]any)["laws"].([]any)[0].(map[string]any)
			law[field].([]any)[0] = uint64(math.MaxUint32) + 1
			mutated, err := json.Marshal(changed)
			if err != nil {
				t.Fatalf("expected out-of-domain fixture, got %v", err)
			}
			_, err = decodeUSASocioeconomicData(t.Context(), bytes.NewReader(mutated))
			if err != nil {
				return
			}
			t.Fatalf("expected out-of-domain %s rejection, got admission", field)
		})
	}

}

func TestUSACountryRegisterRevision(t *testing.T) {

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("expected repository root, got %v", err)
	}
	pin := usaIncomePins[0]
	current, err := os.ReadFile(filepath.Join(root, pin.path))
	if err != nil {
		t.Fatalf("expected approved register revision, got %v", err)
	}
	assets := readUSACountryAssets(t)
	if len(assets[pin.path]) != pin.bytes || fmt.Sprintf("%x", sha256.Sum256(assets[pin.path])) != pin.sha256 {
		t.Fatal("expected exact historical source evidence, got changed lock input")
	}
	for _, name := range []string{"historical-change", "approval-change", "arbitrary-append", "truncation"} {

		t.Run(name, func(t *testing.T) {

			changed := bytes.Clone(current)
			switch name {
			case "historical-change":
				changed[0] ^= 1
			case "approval-change":
				changed[292333] ^= 1
			case "arbitrary-append":
				changed = append(changed, '\n')
			case "truncation":
				changed = changed[:len(changed)-1]
			}
			isolated := t.TempDir()
			for _, source := range usaCountrySourcePins() {

				target := filepath.Join(isolated, source.Path)
				err := os.MkdirAll(filepath.Dir(target), 0700)
				if err != nil {
					t.Fatalf("expected isolated directory, got %v", err)
				}
				if source.Path == pin.path {
					err = os.WriteFile(target, changed, 0600)
				} else {
					err = os.Symlink(filepath.Join(root, source.Path), target)
				}
				if err != nil {
					t.Fatalf("expected isolated source, got %v", err)
				}

			}
			loaded, err := loadUSACountryAssets(t.Context(), isolated)
			if err != nil {
				if loaded != nil {
					t.Fatal("expected no partial assets, got loaded data")
				}
				return
			}

			t.Fatal("expected unapproved register rejection, got admission")

		})

	}

}

func TestUSACountryRuntimeReject(t *testing.T) {

	data := readUSACountryData(t)
	for _, test := range []struct {
		name   string
		change func(*usaCountryData)
	}{
		{"IT-base-US-model", func(d *usaCountryData) { d.baseDocument.Country = "IT" }},
		{"US-base-IT-model", func(d *usaCountryData) { d.socioeconomic.document.Country = "IT" }},
		{"unknown-base-country", func(d *usaCountryData) { d.baseDocument.Country = "XX" }},
		{"EUR-model", func(d *usaCountryData) { d.socioeconomic.document.IncomeUnit.Currency = "EUR" }},
		{"missing-model", func(d *usaCountryData) { d.socioeconomic.model = nil }},
		{"missing-phone-rule", func(d *usaCountryData) { d.phones = nil }},
		{"missing-geography-binding", func(d *usaCountryData) { d.baseDocument.Geography.TerritorySystem = "" }},
		{"corrupt-compiled-model", func(d *usaCountryData) {
			m := *d.socioeconomic.model
			m.numeric.ages[0][0]++
			d.socioeconomic.model = &m
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := *data
			model := *data.socioeconomic
			changed.socioeconomic = &model
			test.change(&changed)
			_, err := newUSACountryWorld(&changed, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
			if err != nil {
				return
			}
			t.Fatal("expected invalid complete World rejection, got World")
		})
	}
	w, err := newUSACountryWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected real USA World, got %v", err)
	}
	for _, coordinate := range []PersonIndex{0, Layer1MaxPersonIndex + 1, PersonIndex(math.MaxUint64)} {
		person, err := w.person(coordinate)
		if err != nil {
			if !reflect.DeepEqual(person, usaCountryPerson{}) {
				t.Fatal("expected empty rejected person, got partial output")
			}
			continue
		}
		t.Fatal("expected invalid Person coordinate rejection, got record")
	}
	for _, coordinate := range []PersonIndex{1, Layer1MaxPersonIndex} {
		_, err := w.person(coordinate)
		if err != nil {
			t.Fatalf("expected valid boundary %d, got %v", coordinate, err)
		}
	}
	file, err := os.CreateTemp(t.TempDir(), "roster")
	if err != nil {
		t.Fatalf("expected diagnostic destination, got %v", err)
	}
	defer file.Close()
	company, err := w.sampler.categorical.companies.company(1)
	if err != nil {
		t.Fatalf("expected first Company, got %v", err)
	}
	output := usaCountryDiagnosticOutput{world: w, companies: file, next: 2}
	t.Run("duplicate-CompanyID", func(t *testing.T) {
		err := output.company(2, company)
		if err != nil {
			info, err := file.Stat()
			if err != nil {
				t.Fatalf("expected negative fixture destination, got %v", err)
			}
			if info.Size() != 0 {
				t.Fatalf("expected no rejected roster bytes, got %d", info.Size())
			}
			return
		}
		t.Fatal("expected duplicate CompanyID rejection, got emission")
	})
	companies := w.sampler.categorical.companies
	index := &companies.sampler.employers[0]
	index.companies = slices.Clone(index.companies)
	for i := range index.companies {
		index.companies[i] = 65537
	}
	_, err = companies.employee(1, socioeconomicJobFamilies[0], socioeconomicJobLevels[0])
	if err != nil {
		return
	}

	t.Fatal("expected invalid Employer index rejection, got employee")
}

func TestUSACountryScopeAndIsolation(t *testing.T) {

	for _, options := range []usadiagnostic.Options{
		{Publish: true}, {Country: "XX", Currency: "USD"}, {Country: "US", Currency: "XXX"},
	} {
		err := executeUSACountryDiagnostic(t.Context(), options)
		if err != nil {
			continue
		}
		t.Fatal("expected unauthorized public/country/currency rejection, got success")
	}
	t.Run("Source-photo-boundary", func(t *testing.T) {
		_, err := NewSourceCSVWriter(io.Discard, nil)
		if err != nil {
			return
		}
		t.Fatal("expected public Source writer rejection without photo catalog, got writer")
	})
	if reflect.TypeFor[annualIncomeUSD]() == reflect.TypeFor[AnnualIncomeEUR]() {
		t.Fatal("expected distinct USD and EUR types, got interchangeable types")
	}
	data := readUSACountryData(t)
	italian := mustSocioeconomicWorld(t,
		mustLayer1World(t, layer1TestReferenceDate, layer1TestSeed), socioeconomicTestModel(t, nil))
	before := socioeconomicFactsAt(t, italian, 42)
	italianID := italian.ID()
	usa, err := newUSACountryWorld(data, IdentityNamespace{Name: "usa-layer1", Generation: 1}, 2026)
	if err != nil {
		t.Fatalf("expected independent USA World, got %v", err)
	}
	person, err := usa.person(42)
	if err != nil {
		t.Fatalf("expected USA record with USD Income, got %v", err)
	}
	if !reflect.DeepEqual(before, socioeconomicFactsAt(t, italian, 42)) || italian.ID() != italianID ||
		person.Country != "US" || person.Currency != "USD" || usa.id == italianID {
		t.Fatal("expected unchanged Italian EUR facts/identity and separate US/USD World, got contamination")
	}
	usaBytes, err := data.socioeconomic.canonicalJSON()
	if err != nil {
		t.Fatalf("expected USA transport, got %v", err)
	}
	_, err = decodeItalianSocioeconomicData(t.Context(), bytes.NewReader(usaBytes))
	if err != nil {
		italian, err := os.ReadFile("testdata/international-socioeconomic-v1.json")
		if err != nil {
			t.Fatalf("expected Italian fixture, got %v", err)
		}
		_, err = decodeUSASocioeconomicData(t.Context(), bytes.NewReader(italian))
		if err != nil {
			return
		}
		t.Fatal("expected Italian model rejection by USA decoder, got admission")
	}

	t.Fatal("expected USA model rejection by Italian decoder, got admission")
}

func readUSACountryAssets(t *testing.T) usaCountryAssets {
	t.Helper()
	assets, err := loadUSACountryAssets(t.Context(), "../..")
	if err != nil {
		t.Fatalf("expected byte-locked USA sources, got %v", err)
	}
	return assets
}

func readUSACountryData(t *testing.T) *usaCountryData {

	t.Helper()
	assets := readUSACountryAssets(t)
	model, err := assembleUSACountryModel(t.Context(), assets)
	if err != nil {
		t.Fatalf("expected assembled complete USA model, got %v", err)
	}
	encoded, err := model.canonicalJSON()
	if err != nil {
		t.Fatalf("expected canonical USA model, got %v", err)
	}
	data, err := prepareUSACountryData(t.Context(), assets, encoded)
	if err != nil {
		t.Fatalf("expected admitted complete USA country, got %v", err)
	}

	return data
}
